package notifications

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var ErrPolicyConflict = errors.New("notifications: policy revision conflict")

type AdminPolicyUpdate struct {
	Type               string `json:"type"`
	Channel            string `json:"channel"`
	Enabled            bool   `json:"enabled"`
	RecommendedEnabled bool   `json:"recommendedEnabled"`
	UserConfigurable   bool   `json:"userConfigurable"`
}

type AdminPolicyItem struct {
	AdminPolicyUpdate
	Category         string `json:"category"`
	OwnerExtensionID string `json:"ownerExtensionId,omitempty"`
	Active           bool   `json:"active"`
	Required         bool   `json:"required"`
}

type AdminPolicyCatalog struct {
	Revision int64             `json:"revision"`
	Items    []AdminPolicyItem `json:"items"`
}

type AdminPolicyStore interface {
	ListAdminPolicy(context.Context) (AdminPolicyCatalog, error)
	ReplaceAdminPolicy(context.Context, int64, []AdminPolicyUpdate) (AdminPolicyCatalog, error)
	RestoreAdminPolicy(context.Context, int64) (AdminPolicyCatalog, error)
}

func (s *PostgresStore) ListAdminPolicy(ctx context.Context) (AdminPolicyCatalog, error) {
	var catalog AdminPolicyCatalog
	if err := s.runner.QueryRow(ctx, `SELECT revision FROM notification_policy_revisions WHERE singleton=TRUE`).Scan(&catalog.Revision); err != nil {
		return AdminPolicyCatalog{}, fmt.Errorf("read notification policy revision: %w", err)
	}
	rows, err := s.runner.Query(ctx, `
		SELECT descriptor.type, descriptor.category, descriptor.owner_extension_id,
		  descriptor.active, policy.channel, policy.enabled,
		  policy.recommended_enabled, policy.user_configurable, policy.required
		FROM notification_type_descriptors descriptor
		JOIN notification_type_policies policy ON policy.type=descriptor.type
		ORDER BY descriptor.category, descriptor.type, policy.channel`)
	if err != nil {
		return AdminPolicyCatalog{}, err
	}
	defer rows.Close()
	catalog.Items = []AdminPolicyItem{}
	for rows.Next() {
		var item AdminPolicyItem
		if err := rows.Scan(&item.Type, &item.Category, &item.OwnerExtensionID, &item.Active, &item.Channel, &item.Enabled, &item.RecommendedEnabled, &item.UserConfigurable, &item.Required); err != nil {
			return AdminPolicyCatalog{}, err
		}
		catalog.Items = append(catalog.Items, item)
	}
	return catalog, rows.Err()
}

func (s *PostgresStore) ReplaceAdminPolicy(ctx context.Context, expectedRevision int64, updates []AdminPolicyUpdate) (AdminPolicyCatalog, error) {
	return s.mutateAdminPolicy(ctx, expectedRevision, updates, false)
}

func (s *PostgresStore) RestoreAdminPolicy(ctx context.Context, expectedRevision int64) (AdminPolicyCatalog, error) {
	return s.mutateAdminPolicy(ctx, expectedRevision, nil, true)
}

func (s *PostgresStore) mutateAdminPolicy(ctx context.Context, expectedRevision int64, updates []AdminPolicyUpdate, restore bool) (AdminPolicyCatalog, error) {
	if s.pool == nil || expectedRevision <= 0 || len(updates) > 500 {
		return AdminPolicyCatalog{}, ErrPreferenceInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AdminPolicyCatalog{}, err
	}
	defer tx.Rollback(ctx)
	var revision int64
	if err := tx.QueryRow(ctx, `SELECT revision FROM notification_policy_revisions WHERE singleton=TRUE FOR UPDATE`).Scan(&revision); err != nil {
		return AdminPolicyCatalog{}, err
	}
	if revision != expectedRevision {
		return AdminPolicyCatalog{}, ErrPolicyConflict
	}
	if restore {
		_, err = tx.Exec(ctx, `
			UPDATE notification_type_policies policy
			SET enabled = CASE WHEN descriptor.owner_extension_id='' AND policy.channel IN ('in_app','email') THEN TRUE ELSE FALSE END,
			    recommended_enabled = CASE WHEN descriptor.owner_extension_id='' AND policy.channel IN ('in_app','email') THEN TRUE ELSE FALSE END,
			    user_configurable = NOT policy.required,
			    revision=policy.revision+1, updated_at=now()
			FROM notification_type_descriptors descriptor
			WHERE descriptor.type=policy.type`)
		if err != nil {
			return AdminPolicyCatalog{}, err
		}
	} else {
		for _, update := range updates {
			var required bool
			if err := tx.QueryRow(ctx, `SELECT required FROM notification_type_policies WHERE type=$1 AND channel=$2 FOR UPDATE`, update.Type, update.Channel).Scan(&required); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return AdminPolicyCatalog{}, ErrPreferenceInvalid
				}
				return AdminPolicyCatalog{}, err
			}
			if required && (!update.Enabled || update.UserConfigurable) {
				return AdminPolicyCatalog{}, ErrPreferenceInvalid
			}
			if _, err := tx.Exec(ctx, `UPDATE notification_type_policies SET enabled=$3, recommended_enabled=$4, user_configurable=$5, revision=revision+1, updated_at=now() WHERE type=$1 AND channel=$2`, update.Type, update.Channel, update.Enabled, update.RecommendedEnabled, update.UserConfigurable); err != nil {
				return AdminPolicyCatalog{}, err
			}
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE notification_policy_revisions SET revision=revision+1, updated_at=now() WHERE singleton=TRUE`); err != nil {
		return AdminPolicyCatalog{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AdminPolicyCatalog{}, err
	}
	return s.ListAdminPolicy(ctx)
}
