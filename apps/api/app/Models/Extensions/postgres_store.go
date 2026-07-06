package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) List(ctx context.Context) ([]Extension, error) {
	rows, err := s.pool.Query(ctx, extensionSelectSQL()+`
		ORDER BY extensions.type, extensions.name, extensions.id
	`)
	if err != nil {
		return nil, fmt.Errorf("list extensions: %w", err)
	}
	defer rows.Close()

	items := []Extension{}
	for rows.Next() {
		item, err := scanExtension(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate extensions: %w", err)
	}
	return items, nil
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Extension, error) {
	row := s.pool.QueryRow(ctx, extensionSelectSQL()+`
		WHERE extensions.id = $1
	`, id)
	item, err := scanExtension(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Extension{}, ErrExtensionNotFound
	}
	if err != nil {
		return Extension{}, err
	}
	return item, nil
}

func (s *PostgresStore) SaveInstalled(ctx context.Context, input SaveInstalledInput) (Extension, error) {
	manifestJSON, err := json.Marshal(input.Manifest)
	if err != nil {
		return Extension{}, fmt.Errorf("marshal extension manifest: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Extension{}, fmt.Errorf("begin extension install: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO extensions (id, type, name, status)
		VALUES ($1, $2, $3, 'installed')
		ON CONFLICT (id) DO UPDATE
		SET type = EXCLUDED.type,
		    name = EXCLUDED.name,
		    status = 'installed',
		    source = 'uploaded',
		    is_system = false,
		    is_deletable = true,
		    updated_at = now()
	`, input.Manifest.ID, input.Manifest.Type, input.Manifest.Name); err != nil {
		return Extension{}, fmt.Errorf("upsert extension: %w", err)
	}

	var versionID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO extension_versions (extension_id, version, manifest, package_path)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (extension_id, version) DO UPDATE
		SET manifest = EXCLUDED.manifest,
		    package_path = EXCLUDED.package_path
		RETURNING id
	`, input.Manifest.ID, input.Manifest.Version, manifestJSON, input.PackagePath).Scan(&versionID); err != nil {
		return Extension{}, fmt.Errorf("upsert extension version: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE extensions
		SET active_version_id = $2,
		    updated_at = now()
		WHERE id = $1
	`, input.Manifest.ID, versionID); err != nil {
		return Extension{}, fmt.Errorf("activate installed extension version: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Extension{}, fmt.Errorf("commit extension install: %w", err)
	}
	return s.Get(ctx, input.Manifest.ID)
}

func (s *PostgresStore) SaveBuiltin(ctx context.Context, input SaveBuiltinInput) (Extension, error) {
	manifestJSON, err := json.Marshal(input.Manifest)
	if err != nil {
		return Extension{}, fmt.Errorf("marshal builtin extension manifest: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Extension{}, fmt.Errorf("begin builtin extension sync: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO extensions (id, type, name, status, source, is_system, is_deletable)
		VALUES ($1, $2, $3, 'enabled', 'builtin', true, false)
		ON CONFLICT (id) DO UPDATE
		SET type = EXCLUDED.type,
		    name = EXCLUDED.name,
		    source = 'builtin',
		    is_system = true,
		    is_deletable = false,
		    updated_at = now()
	`, input.Manifest.ID, input.Manifest.Type, input.Manifest.Name); err != nil {
		return Extension{}, fmt.Errorf("upsert builtin extension: %w", err)
	}

	var versionID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO extension_versions (extension_id, version, manifest, package_path)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (extension_id, version) DO UPDATE
		SET manifest = EXCLUDED.manifest,
		    package_path = EXCLUDED.package_path
		RETURNING id
	`, input.Manifest.ID, input.Manifest.Version, manifestJSON, input.PackagePath).Scan(&versionID); err != nil {
		return Extension{}, fmt.Errorf("upsert builtin extension version: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE extensions
		SET active_version_id = $2,
		    updated_at = now()
		WHERE id = $1
	`, input.Manifest.ID, versionID); err != nil {
		return Extension{}, fmt.Errorf("activate builtin extension version: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Extension{}, fmt.Errorf("commit builtin extension sync: %w", err)
	}
	return s.Get(ctx, input.Manifest.ID)
}

func (s *PostgresStore) PruneMissingBuiltins(ctx context.Context, activeIDs []string) error {
	if len(activeIDs) == 0 {
		return nil
	}
	if _, err := s.pool.Exec(ctx, `
		DELETE FROM extensions
		WHERE source = 'builtin'
		  AND NOT (id = ANY($1::text[]))
	`, activeIDs); err != nil {
		return fmt.Errorf("prune missing builtin extensions: %w", err)
	}
	return nil
}

func (s *PostgresStore) Enable(ctx context.Context, id string, extensionType string) (Extension, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Extension{}, fmt.Errorf("begin extension enable: %w", err)
	}
	defer tx.Rollback(ctx)

	if extensionType == TypeTheme {
		if _, err := tx.Exec(ctx, `
			UPDATE extensions
			SET status = 'disabled', updated_at = now()
			WHERE type = 'theme' AND id <> $1 AND status = 'enabled'
		`, id); err != nil {
			return Extension{}, fmt.Errorf("disable previous active theme: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE extensions
		SET status = 'enabled', updated_at = now()
		WHERE id = $1
	`, id); err != nil {
		return Extension{}, fmt.Errorf("enable extension: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Extension{}, fmt.Errorf("commit extension enable: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *PostgresStore) ActivateTheme(ctx context.Context, id string) (Extension, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Extension{}, fmt.Errorf("begin theme activation: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE extensions
		SET status = 'disabled', updated_at = now()
		WHERE type = 'theme' AND id <> $1 AND status = 'enabled'
	`, id); err != nil {
		return Extension{}, fmt.Errorf("disable previous active theme: %w", err)
	}
	command, err := tx.Exec(ctx, `
		UPDATE extensions
		SET status = 'enabled', updated_at = now()
		WHERE id = $1 AND type = 'theme'
	`, id)
	if err != nil {
		return Extension{}, fmt.Errorf("activate theme: %w", err)
	}
	if command.RowsAffected() == 0 {
		return Extension{}, ErrExtensionNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return Extension{}, fmt.Errorf("commit theme activation: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *PostgresStore) ActiveTheme(ctx context.Context) (Extension, error) {
	row := s.pool.QueryRow(ctx, extensionSelectSQL()+`
		WHERE extensions.type = 'theme' AND extensions.status = 'enabled'
		ORDER BY extensions.source = 'uploaded' DESC, extensions.updated_at DESC
		LIMIT 1
	`)
	item, err := scanExtension(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Extension{}, ErrExtensionNotFound
	}
	if err != nil {
		return Extension{}, err
	}
	return item, nil
}

func (s *PostgresStore) Disable(ctx context.Context, id string) (Extension, error) {
	command, err := s.pool.Exec(ctx, `
		UPDATE extensions
		SET status = 'disabled', updated_at = now()
		WHERE id = $1
	`, id)
	if err != nil {
		return Extension{}, fmt.Errorf("disable extension: %w", err)
	}
	if command.RowsAffected() == 0 {
		return Extension{}, ErrExtensionNotFound
	}
	return s.Get(ctx, id)
}

func (s *PostgresStore) CreateEvent(ctx context.Context, input EventInput) (ExtensionEvent, error) {
	var event ExtensionEvent
	err := s.pool.QueryRow(ctx, `
		INSERT INTO extension_events (extension_id, actor_user_id, action, message)
		VALUES ($1, $2, $3, $4)
		RETURNING id, extension_id, COALESCE(actor_user_id, 0), action, message, created_at
	`, input.ExtensionID, nullableActorID(input.ActorUserID), input.Action, input.Message).Scan(
		&event.ID,
		&event.ExtensionID,
		&event.ActorUserID,
		&event.Action,
		&event.Message,
		&event.CreatedAt,
	)
	if err != nil {
		return ExtensionEvent{}, fmt.Errorf("create extension event: %w", err)
	}
	return event, nil
}

func (s *PostgresStore) ListEvents(ctx context.Context, extensionID string, limit int) ([]ExtensionEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, extension_id, COALESCE(actor_user_id, 0), action, message, created_at
		FROM extension_events
		WHERE extension_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`, extensionID, limit)
	if err != nil {
		return nil, fmt.Errorf("list extension events: %w", err)
	}
	defer rows.Close()

	events := []ExtensionEvent{}
	for rows.Next() {
		var event ExtensionEvent
		if err := rows.Scan(&event.ID, &event.ExtensionID, &event.ActorUserID, &event.Action, &event.Message, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan extension event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate extension events: %w", err)
	}
	return events, nil
}

func (s *PostgresStore) CreateEventDelivery(ctx context.Context, input EventDeliveryInput) (ExtensionEventDelivery, error) {
	status := input.Status
	if status == "" {
		status = DeliveryQueued
	}
	var delivery ExtensionEventDelivery
	err := s.pool.QueryRow(ctx, `
		INSERT INTO extension_event_deliveries
		  (extension_id, event_name, event_kind, status, reason, message, correlation_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, extension_id, event_name, event_kind, status, reason, message,
		  correlation_id, attempt_count, created_at, updated_at, completed_at
	`, input.ExtensionID, input.EventName, input.EventKind, status, input.Reason, input.Message, input.CorrelationID).Scan(
		&delivery.ID,
		&delivery.ExtensionID,
		&delivery.EventName,
		&delivery.EventKind,
		&delivery.Status,
		&delivery.Reason,
		&delivery.Message,
		&delivery.CorrelationID,
		&delivery.AttemptCount,
		&delivery.CreatedAt,
		&delivery.UpdatedAt,
		&delivery.CompletedAt,
	)
	if err != nil {
		return ExtensionEventDelivery{}, fmt.Errorf("create extension event delivery: %w", err)
	}
	return delivery, nil
}

func (s *PostgresStore) UpdateEventDelivery(ctx context.Context, input EventDeliveryUpdateInput) error {
	completedSQL := "completed_at"
	if input.Completed {
		completedSQL = "now()"
	}
	command, err := s.pool.Exec(ctx, `
		UPDATE extension_event_deliveries
		SET status = $2,
		    reason = $3,
		    message = $4,
		    attempt_count = $5,
		    updated_at = now(),
		    completed_at = `+completedSQL+`
		WHERE id = $1
	`, input.ID, input.Status, input.Reason, input.Message, input.AttemptCount)
	if err != nil {
		return fmt.Errorf("update extension event delivery: %w", err)
	}
	if command.RowsAffected() == 0 {
		return ErrExtensionNotFound
	}
	return nil
}

func (s *PostgresStore) ListEventDeliveries(ctx context.Context, input EventDeliveryListInput) ([]ExtensionEventDelivery, error) {
	limit := input.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, extension_id, event_name, event_kind, status, reason, message,
		  correlation_id, attempt_count, created_at, updated_at, completed_at
		FROM extension_event_deliveries
		WHERE ($1 = '' OR extension_id = $1)
		  AND ($2 = '' OR event_name = $2)
		  AND ($3 = '' OR status = $3)
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`, input.ExtensionID, input.EventName, input.Status, limit)
	if err != nil {
		return nil, fmt.Errorf("list extension event deliveries: %w", err)
	}
	defer rows.Close()

	deliveries := []ExtensionEventDelivery{}
	for rows.Next() {
		var delivery ExtensionEventDelivery
		if err := rows.Scan(
			&delivery.ID,
			&delivery.ExtensionID,
			&delivery.EventName,
			&delivery.EventKind,
			&delivery.Status,
			&delivery.Reason,
			&delivery.Message,
			&delivery.CorrelationID,
			&delivery.AttemptCount,
			&delivery.CreatedAt,
			&delivery.UpdatedAt,
			&delivery.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan extension event delivery: %w", err)
		}
		deliveries = append(deliveries, delivery)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate extension event deliveries: %w", err)
	}
	return deliveries, nil
}

func extensionSelectSQL() string {
	return `
		SELECT extensions.id, extensions.name, extensions.type, extensions.status,
		  extensions.source, extensions.is_system, extensions.is_deletable,
		  extension_versions.version, extension_versions.manifest, extension_versions.package_path,
		  extension_versions.installed_at, extensions.updated_at
		FROM extensions
		JOIN extension_versions ON extension_versions.id = extensions.active_version_id
	`
}

type extensionRow interface {
	Scan(dest ...any) error
}

func scanExtension(row extensionRow) (Extension, error) {
	var item Extension
	var manifestJSON []byte
	if err := row.Scan(
		&item.ID,
		&item.Name,
		&item.Type,
		&item.Status,
		&item.Source,
		&item.IsSystem,
		&item.IsDeletable,
		&item.Version,
		&manifestJSON,
		&item.PackagePath,
		&item.InstalledAt,
		&item.UpdatedAt,
	); err != nil {
		return Extension{}, err
	}
	if err := json.Unmarshal(manifestJSON, &item.Manifest); err != nil {
		return Extension{}, fmt.Errorf("decode extension manifest: %w", err)
	}
	return item, nil
}

func nullableActorID(userID int64) any {
	if userID == 0 {
		return nil
	}
	return userID
}
