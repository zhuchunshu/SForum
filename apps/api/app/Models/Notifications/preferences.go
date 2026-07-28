package notifications

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var (
	ErrPreferenceConflict = errors.New("notifications: preference revision conflict")
	ErrPreferenceInvalid  = errors.New("notifications: preference is not configurable")
)

type PreferenceInput struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
	State   string `json:"state"`
}

type PreferenceItem struct {
	Type             string `json:"type"`
	Category         string `json:"category"`
	Channel          string `json:"channel"`
	Active           bool   `json:"active"`
	Enabled          bool   `json:"enabled"`
	Recommended      bool   `json:"recommendedEnabled"`
	UserConfigurable bool   `json:"userConfigurable"`
	Required         bool   `json:"required"`
	State            string `json:"state"`
	Effective        bool   `json:"effective"`
}

type PreferenceCatalog struct {
	Revision int64            `json:"revision"`
	Items    []PreferenceItem `json:"items"`
}

type PreferenceStore interface {
	ListPreferences(context.Context, int64) (PreferenceCatalog, error)
	ReplacePreferences(context.Context, int64, int64, []PreferenceInput) (PreferenceCatalog, error)
	RestorePreferences(context.Context, int64, int64) (PreferenceCatalog, error)
}

func (s *PostgresStore) ListPreferences(ctx context.Context, userID int64) (PreferenceCatalog, error) {
	var revision int64
	if err := s.runner.QueryRow(ctx, `SELECT revision FROM notification_preference_revisions WHERE user_id=$1`, userID).Scan(&revision); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return PreferenceCatalog{}, fmt.Errorf("read notification preference revision: %w", err)
	}
	rows, err := s.runner.Query(ctx, `
		SELECT descriptor.type, descriptor.category, policy.channel, descriptor.active,
		  policy.enabled, policy.recommended_enabled, policy.user_configurable,
		  policy.required, COALESCE(preference.state, 'inherit')
		FROM notification_type_descriptors descriptor
		JOIN notification_type_policies policy ON policy.type=descriptor.type
		LEFT JOIN notification_preferences preference
		  ON preference.user_id=$1 AND preference.type=descriptor.type AND preference.channel=policy.channel
		ORDER BY descriptor.category, descriptor.type, policy.channel`, userID)
	if err != nil {
		return PreferenceCatalog{}, fmt.Errorf("list notification preferences: %w", err)
	}
	defer rows.Close()
	catalog := PreferenceCatalog{Revision: revision, Items: []PreferenceItem{}}
	for rows.Next() {
		var item PreferenceItem
		if err := rows.Scan(&item.Type, &item.Category, &item.Channel, &item.Active, &item.Enabled, &item.Recommended, &item.UserConfigurable, &item.Required, &item.State); err != nil {
			return PreferenceCatalog{}, err
		}
		item.Effective = item.Active && item.Enabled && (item.Required || (!item.UserConfigurable && item.Recommended) || (item.UserConfigurable && map[string]bool{"enabled": true, "disabled": false, "inherit": item.Recommended}[item.State]))
		catalog.Items = append(catalog.Items, item)
	}
	return catalog, rows.Err()
}

func (s *PostgresStore) ReplacePreferences(ctx context.Context, userID, expectedRevision int64, inputs []PreferenceInput) (PreferenceCatalog, error) {
	return s.mutatePreferences(ctx, userID, expectedRevision, inputs, false)
}

func (s *PostgresStore) RestorePreferences(ctx context.Context, userID, expectedRevision int64) (PreferenceCatalog, error) {
	return s.mutatePreferences(ctx, userID, expectedRevision, nil, true)
}

func (s *PostgresStore) mutatePreferences(ctx context.Context, userID, expectedRevision int64, inputs []PreferenceInput, restore bool) (PreferenceCatalog, error) {
	if s.pool == nil || expectedRevision < 0 {
		return PreferenceCatalog{}, ErrPreferenceInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PreferenceCatalog{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `INSERT INTO notification_preference_revisions (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID); err != nil {
		return PreferenceCatalog{}, err
	}
	var revision int64
	if err := tx.QueryRow(ctx, `SELECT revision FROM notification_preference_revisions WHERE user_id=$1 FOR UPDATE`, userID).Scan(&revision); err != nil {
		return PreferenceCatalog{}, err
	}
	if revision != expectedRevision {
		return PreferenceCatalog{}, ErrPreferenceConflict
	}
	if _, err := tx.Exec(ctx, `DELETE FROM notification_preferences WHERE user_id=$1`, userID); err != nil {
		return PreferenceCatalog{}, err
	}
	if !restore {
		for _, input := range inputs {
			if input.State != "inherit" && input.State != "enabled" && input.State != "disabled" {
				return PreferenceCatalog{}, ErrPreferenceInvalid
			}
			var active, enabled, configurable, required bool
			if err := tx.QueryRow(ctx, `
				SELECT descriptor.active, policy.enabled, policy.user_configurable, policy.required
				FROM notification_type_descriptors descriptor JOIN notification_type_policies policy ON policy.type=descriptor.type
				WHERE descriptor.type=$1 AND policy.channel=$2`, input.Type, input.Channel).Scan(&active, &enabled, &configurable, &required); err != nil || !active || !enabled || required || !configurable {
				return PreferenceCatalog{}, ErrPreferenceInvalid
			}
			if input.State == "inherit" {
				continue
			}
			if _, err := tx.Exec(ctx, `INSERT INTO notification_preferences (user_id, type, channel, state) VALUES ($1,$2,$3,$4)`, userID, input.Type, input.Channel, input.State); err != nil {
				return PreferenceCatalog{}, err
			}
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE notification_preference_revisions SET revision=revision+1, updated_at=now() WHERE user_id=$1`, userID); err != nil {
		return PreferenceCatalog{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PreferenceCatalog{}, err
	}
	return s.ListPreferences(ctx, userID)
}
