package sitechrome

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
)

func (s *PostgresStore) ReadNavigationDocument(ctx context.Context) (NavigationDocument, error) {
	if s == nil || s.pool == nil {
		return NavigationDocument{}, ErrInvalid
	}
	return readNavigationDocument(ctx, s.pool)
}

type navigationDocumentQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func readNavigationDocument(ctx context.Context, query navigationDocumentQuerier) (NavigationDocument, error) {
	document := NavigationDocument{Revision: 1, Definitions: []NavigationDefinition{}, Placements: []NavigationPlacement{}}
	if err := query.QueryRow(ctx, `SELECT revision FROM site_navigation_state WHERE id = 1`).Scan(&document.Revision); err != nil {
		return NavigationDocument{}, fmt.Errorf("read navigation revision: %w", err)
	}

	definitions, err := query.Query(ctx, `
		SELECT source_key, source_kind, link_kind, label_zh_cn, label_en_us, href, icon,
		       open_in_new_tab, extension_id, contribution_id
		FROM site_navigation_definitions
		ORDER BY source_key ASC
	`)
	if err != nil {
		return NavigationDocument{}, fmt.Errorf("list navigation definitions: %w", err)
	}
	defer definitions.Close()
	for definitions.Next() {
		var definition NavigationDefinition
		if err := definitions.Scan(&definition.SourceKey, &definition.SourceKind, &definition.LinkKind,
			&definition.LabelZhCN, &definition.LabelEnUS, &definition.Href, &definition.Icon,
			&definition.OpenInNewTab, &definition.ExtensionID, &definition.ContributionID); err != nil {
			return NavigationDocument{}, fmt.Errorf("scan navigation definition: %w", err)
		}
		document.Definitions = append(document.Definitions, definition)
	}
	if err := definitions.Err(); err != nil {
		return NavigationDocument{}, err
	}

	placements, err := query.Query(ctx, `
		SELECT source_key, location, position, enabled, visibility, permission, label_zh_cn, label_en_us, icon
		FROM site_navigation_placements
		ORDER BY location ASC, position ASC, source_key ASC
	`)
	if err != nil {
		return NavigationDocument{}, fmt.Errorf("list navigation placements: %w", err)
	}
	defer placements.Close()
	for placements.Next() {
		var placement NavigationPlacement
		if err := placements.Scan(&placement.SourceKey, &placement.Location, &placement.Order, &placement.Enabled,
			&placement.Visibility, &placement.Permission, &placement.LabelZhCN, &placement.LabelEnUS, &placement.Icon); err != nil {
			return NavigationDocument{}, fmt.Errorf("scan navigation placement: %w", err)
		}
		document.Placements = append(document.Placements, placement)
	}
	if err := placements.Err(); err != nil {
		return NavigationDocument{}, err
	}
	return document, nil
}

func (s *PostgresStore) ExecuteNavigationTransaction(
	ctx context.Context,
	expectedRevision uint64,
	mutation func(context.Context, pgx.Tx, NavigationDocument) (NavigationTransactionResult, error),
) (NavigationDocument, error) {
	if s == nil || s.pool == nil || expectedRevision < 1 || mutation == nil {
		return NavigationDocument{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return NavigationDocument{}, fmt.Errorf("begin navigation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentRevision uint64
	if err := tx.QueryRow(ctx, `SELECT revision FROM site_navigation_state WHERE id = 1 FOR UPDATE`).Scan(&currentRevision); err != nil {
		return NavigationDocument{}, fmt.Errorf("lock navigation revision: %w", err)
	}
	if currentRevision != expectedRevision {
		return NavigationDocument{}, ErrConflict
	}
	current, err := readNavigationDocument(ctx, tx)
	if err != nil {
		return NavigationDocument{}, err
	}
	result, err := mutation(ctx, tx, current)
	if err != nil {
		return NavigationDocument{}, err
	}
	if err := replaceNavigationDocument(ctx, tx, result.Document); err != nil {
		return NavigationDocument{}, err
	}
	if err := insertNavigationSnapshot(ctx, tx, current, result); err != nil {
		return NavigationDocument{}, err
	}
	nextRevision := currentRevision + 1
	if _, err := tx.Exec(ctx, `UPDATE site_navigation_state SET revision = $1, updated_at = now() WHERE id = 1`, nextRevision); err != nil {
		return NavigationDocument{}, fmt.Errorf("advance navigation revision: %w", err)
	}
	if err := retainNavigationSnapshots(ctx, tx); err != nil {
		return NavigationDocument{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return NavigationDocument{}, fmt.Errorf("commit navigation transaction: %w", err)
	}
	result.Document.Revision = nextRevision
	return result.Document, nil
}

func replaceNavigationDocument(ctx context.Context, tx pgx.Tx, document NavigationDocument) error {
	if _, err := tx.Exec(ctx, `DELETE FROM site_navigation_placements`); err != nil {
		return fmt.Errorf("clear navigation placements: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM site_navigation_definitions`); err != nil {
		return fmt.Errorf("clear navigation definitions: %w", err)
	}
	definitions := append([]NavigationDefinition(nil), document.Definitions...)
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].SourceKey < definitions[j].SourceKey })
	for _, definition := range definitions {
		if _, err := tx.Exec(ctx, `
			INSERT INTO site_navigation_definitions
			(source_key, source_kind, link_kind, label_zh_cn, label_en_us, href, icon, open_in_new_tab, extension_id, contribution_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		`, definition.SourceKey, definition.SourceKind, definition.LinkKind, definition.LabelZhCN, definition.LabelEnUS,
			definition.Href, definition.Icon, definition.OpenInNewTab, definition.ExtensionID, definition.ContributionID); err != nil {
			return fmt.Errorf("insert navigation definition: %w", err)
		}
	}
	placements := append([]NavigationPlacement(nil), document.Placements...)
	sort.Slice(placements, func(i, j int) bool {
		if placements[i].Location != placements[j].Location {
			return placements[i].Location < placements[j].Location
		}
		if placements[i].Order != placements[j].Order {
			return placements[i].Order < placements[j].Order
		}
		return placements[i].SourceKey < placements[j].SourceKey
	})
	for _, placement := range placements {
		if _, err := tx.Exec(ctx, `
			INSERT INTO site_navigation_placements
			(source_key, location, position, enabled, visibility, permission, label_zh_cn, label_en_us, icon)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		`, placement.SourceKey, placement.Location, placement.Order, placement.Enabled, placement.Visibility,
			placement.Permission, placement.LabelZhCN, placement.LabelEnUS, placement.Icon); err != nil {
			return fmt.Errorf("insert navigation placement: %w", err)
		}
	}
	return nil
}

func insertNavigationSnapshot(ctx context.Context, tx pgx.Tx, current NavigationDocument, result NavigationTransactionResult) error {
	document, err := json.Marshal(current)
	if err != nil {
		return fmt.Errorf("encode navigation snapshot: %w", err)
	}
	locations, err := json.Marshal(result.AffectedLocations)
	if err != nil {
		return fmt.Errorf("encode navigation snapshot locations: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO site_navigation_snapshots (revision, operation, reason, actor_user_id, affected_locations, document)
		VALUES ($1, $2, $3, NULLIF($4::bigint, 0), $5::jsonb, $6::jsonb)
	`, current.Revision, result.Operation, result.Reason, result.ActorUserID, string(locations), string(document)); err != nil {
		return fmt.Errorf("insert navigation snapshot: %w", err)
	}
	return nil
}

func retainNavigationSnapshots(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM site_navigation_snapshots
		WHERE id NOT IN (
			SELECT id FROM site_navigation_snapshots ORDER BY revision DESC, id DESC LIMIT $1
		)
	`, NavigationMaxSnapshots); err != nil {
		return fmt.Errorf("retain navigation snapshots: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListNavigationSnapshots(ctx context.Context) ([]NavigationSnapshot, error) {
	if s == nil || s.pool == nil {
		return nil, ErrInvalid
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, revision, actor_user_id, operation, reason, affected_locations, document, created_at
		FROM site_navigation_snapshots ORDER BY revision DESC, id DESC LIMIT $1
	`, NavigationMaxSnapshots)
	if err != nil {
		return nil, fmt.Errorf("list navigation snapshots: %w", err)
	}
	defer rows.Close()
	result := make([]NavigationSnapshot, 0)
	for rows.Next() {
		snapshot, err := scanNavigationSnapshot(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, snapshot)
	}
	return result, rows.Err()
}

func (s *PostgresStore) GetNavigationSnapshot(ctx context.Context, id int64) (NavigationSnapshot, error) {
	if s == nil || s.pool == nil || id <= 0 {
		return NavigationSnapshot{}, ErrInvalid
	}
	snapshot, err := scanNavigationSnapshot(s.pool.QueryRow(ctx, `
		SELECT id, revision, actor_user_id, operation, reason, affected_locations, document, created_at
		FROM site_navigation_snapshots WHERE id = $1
	`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return NavigationSnapshot{}, ErrNotFound
	}
	return snapshot, err
}

type navigationSnapshotRow interface{ Scan(...any) error }

func scanNavigationSnapshot(row navigationSnapshotRow) (NavigationSnapshot, error) {
	var snapshot NavigationSnapshot
	var actorUserID sql.NullInt64
	var locations, document []byte
	if err := row.Scan(&snapshot.ID, &snapshot.Revision, &actorUserID, &snapshot.Operation, &snapshot.Reason, &locations, &document, &snapshot.CreatedAt); err != nil {
		return NavigationSnapshot{}, err
	}
	if actorUserID.Valid {
		snapshot.ActorUserID = actorUserID.Int64
	}
	if err := json.Unmarshal(locations, &snapshot.AffectedLocations); err != nil {
		return NavigationSnapshot{}, fmt.Errorf("decode snapshot locations: %w", err)
	}
	if err := json.Unmarshal(document, &snapshot.Document); err != nil {
		return NavigationSnapshot{}, fmt.Errorf("decode snapshot document: %w", err)
	}
	return snapshot, nil
}
