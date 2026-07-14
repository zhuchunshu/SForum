package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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

func (s *PostgresStore) GetMailProviderExtension(ctx context.Context, id string) (Extension, error) {
	return s.Get(ctx, id)
}

func (s *PostgresStore) SelectedMailProvider(ctx context.Context) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT extension_id FROM mail_provider_selection WHERE slot = 'mail.provider'`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func (s *PostgresStore) SelectMailProvider(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO mail_provider_selection (slot, extension_id) VALUES ('mail.provider', $1)
		ON CONFLICT (slot) DO UPDATE SET extension_id=EXCLUDED.extension_id, updated_at=NOW()`, id)
	return err
}

func (s *PostgresStore) RestoreMailProvider(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM mail_provider_selection WHERE slot = 'mail.provider'`)
	return err
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

	inserted, err := tx.Exec(ctx, `
		INSERT INTO extensions (id, type, name, status, source, is_system, is_deletable)
		VALUES ($1, $2, $3, 'installed', 'uploaded', false, true)
		ON CONFLICT (id) DO NOTHING
	`, input.Manifest.ID, input.Manifest.Type, input.Manifest.Name)
	if err != nil {
		return Extension{}, fmt.Errorf("upsert extension: %w", err)
	}
	created := inserted.RowsAffected() == 1
	var activeVersionID int64
	var existingType, source string
	var isSystem bool
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(active_version_id, 0), type, source, is_system
		FROM extensions
		WHERE id = $1
		FOR UPDATE
	`, input.Manifest.ID).Scan(&activeVersionID, &existingType, &source, &isSystem); err != nil {
		return Extension{}, fmt.Errorf("lock extension install: %w", err)
	}
	if existingType != input.Manifest.Type {
		return Extension{}, ErrInvalidManifest
	}
	if !created && (source == SourceBuiltin || isSystem) {
		return Extension{}, ErrNotDeletable
	}
	if !created {
		if _, err := tx.Exec(ctx, `
			UPDATE extensions
			SET name = $2, updated_at = now()
			WHERE id = $1
		`, input.Manifest.ID, input.Manifest.Name); err != nil {
			return Extension{}, fmt.Errorf("update extension metadata: %w", err)
		}
	}

	versionID, err := ensureExtensionVersion(ctx, tx, extensionVersionInput{
		ExtensionID:         input.Manifest.ID,
		Version:             input.Manifest.Version,
		ManifestJSON:        manifestJSON,
		PackagePath:         input.PackagePath,
		PackageDigest:       input.PackageDigest,
		AdminFrontendDigest: input.AdminFrontendDigest,
	})
	if err != nil {
		return Extension{}, fmt.Errorf("upsert extension version: %w", err)
	}

	if created || activeVersionID == 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE extensions
			SET active_version_id = $2,
			    staged_version_id = NULL,
			    updated_at = now()
			WHERE id = $1
		`, input.Manifest.ID, versionID); err != nil {
			return Extension{}, fmt.Errorf("activate initial extension version: %w", err)
		}
	} else if activeVersionID != versionID {
		if _, err := tx.Exec(ctx, `
			UPDATE extensions
			SET staged_version_id = $2,
			    updated_at = now()
			WHERE id = $1
		`, input.Manifest.ID, versionID); err != nil {
			return Extension{}, fmt.Errorf("stage extension version: %w", err)
		}
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

	versionID, err := ensureExtensionVersion(ctx, tx, extensionVersionInput{
		ExtensionID:         input.Manifest.ID,
		Version:             input.Manifest.Version,
		ManifestJSON:        manifestJSON,
		PackagePath:         input.PackagePath,
		PackageDigest:       input.PackageDigest,
		AdminFrontendDigest: input.AdminFrontendDigest,
	})
	if err != nil {
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

// Delete 卸载扩展：CASCADE 清理 versions/settings/events/ledger 等。
func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin extension delete: %w", err)
	}
	defer tx.Rollback(ctx)
	// 先清 mail provider 选择，避免悬挂引用。
	if _, err := tx.Exec(ctx, `DELETE FROM mail_provider_selection WHERE extension_id = $1`, id); err != nil {
		return fmt.Errorf("clear mail provider on delete: %w", err)
	}
	command, err := tx.Exec(ctx, `DELETE FROM extensions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete extension: %w", err)
	}
	if command.RowsAffected() == 0 {
		return ErrExtensionNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit extension delete: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListMigrationLedger(ctx context.Context, extensionID string) ([]MigrationRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT path, checksum, status, applied_at, message
		FROM extension_migration_ledger
		WHERE extension_id = $1
		ORDER BY applied_at ASC, path ASC
	`, extensionID)
	if err != nil {
		return nil, fmt.Errorf("list migration ledger: %w", err)
	}
	defer rows.Close()
	items := []MigrationRecord{}
	for rows.Next() {
		var item MigrationRecord
		if err := rows.Scan(&item.Path, &item.Checksum, &item.Status, &item.AppliedAt, &item.Message); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) RecordMigration(ctx context.Context, extensionID string, record MigrationRecord) error {
	status := record.Status
	if status == "" {
		status = "recorded"
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO extension_migration_ledger (extension_id, path, checksum, status, message)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (extension_id, path) DO UPDATE
		SET checksum = EXCLUDED.checksum,
		    status = EXCLUDED.status,
		    message = EXCLUDED.message,
		    applied_at = now()
	`, extensionID, record.Path, record.Checksum, status, record.Message)
	if err != nil {
		return fmt.Errorf("record migration: %w", err)
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
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('sforum.theme.activation.v1'))`); err != nil {
			return Extension{}, fmt.Errorf("lock theme activation: %w", err)
		}
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
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('sforum.theme.activation.v1'))`); err != nil {
		return Extension{}, fmt.Errorf("lock theme activation: %w", err)
	}

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

func (s *PostgresStore) ActivateThemeExact(ctx context.Context, id string, expected ThemeActivationInput) (Extension, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Extension{}, fmt.Errorf("begin exact theme activation: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('sforum.theme.activation.v1'))`); err != nil {
		return Extension{}, fmt.Errorf("lock theme activation: %w", err)
	}
	var currentID, currentVersion, currentDigest string
	err = tx.QueryRow(ctx, `
		SELECT e.id, v.version, v.package_digest
		FROM extensions e
		JOIN extension_versions v ON v.id = e.active_version_id
		WHERE e.type = 'theme' AND e.status = 'enabled'
		FOR UPDATE OF e`).Scan(&currentID, &currentVersion, &currentDigest)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	if err != nil {
		return Extension{}, err
	}
	if currentID != expected.CurrentThemeID || currentVersion != expected.CurrentThemeVersion ||
		!strings.EqualFold(currentDigest, expected.CurrentThemeDigest) {
		return Extension{}, ErrThemePreviewStale
	}
	var targetVersion, targetDigest string
	if err := tx.QueryRow(ctx, `
		SELECT v.version, v.package_digest
		FROM extensions e
		JOIN extension_versions v ON v.id = e.active_version_id
		WHERE e.id = $1 AND e.type = 'theme'
		FOR UPDATE OF e`, id).Scan(&targetVersion, &targetDigest); errors.Is(err, pgx.ErrNoRows) {
		return Extension{}, ErrExtensionNotFound
	} else if err != nil {
		return Extension{}, err
	}
	if targetVersion != expected.Version || !strings.EqualFold(targetDigest, expected.PackageDigest) {
		return Extension{}, ErrThemePreviewStale
	}
	if _, err := tx.Exec(ctx, `UPDATE extensions SET status = 'disabled', updated_at = now() WHERE type = 'theme' AND id <> $1 AND status = 'enabled'`, id); err != nil {
		return Extension{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE extensions SET status = 'enabled', updated_at = now() WHERE id = $1 AND type = 'theme'`, id); err != nil {
		return Extension{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Extension{}, err
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
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Extension{}, fmt.Errorf("begin disable extension: %w", err)
	}
	defer tx.Rollback(ctx)
	command, err := tx.Exec(ctx, `
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
	if _, err := tx.Exec(ctx, `DELETE FROM mail_provider_selection WHERE extension_id=$1`, id); err != nil {
		return Extension{}, fmt.Errorf("clear disabled mail provider: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Extension{}, fmt.Errorf("commit disable extension: %w", err)
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

func (s *PostgresStore) ListSettings(ctx context.Context, extensionID string) (map[string]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT name, value
		FROM extension_settings
		WHERE extension_id = $1
		ORDER BY name
	`, extensionID)
	if err != nil {
		return nil, fmt.Errorf("list extension settings: %w", err)
	}
	defer rows.Close()

	values := map[string]string{}
	for rows.Next() {
		var name string
		var value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("scan extension setting: %w", err)
		}
		values[name] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate extension settings: %w", err)
	}
	return values, nil
}

func (s *PostgresStore) ReplaceSettings(ctx context.Context, extensionID string, values map[string]string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin extension settings update: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "DELETE FROM extension_settings WHERE extension_id = $1", extensionID); err != nil {
		return fmt.Errorf("clear extension settings: %w", err)
	}
	for name, value := range values {
		if _, err := tx.Exec(ctx, `
			INSERT INTO extension_settings (extension_id, name, value)
			VALUES ($1, $2, $3)
		`, extensionID, name, value); err != nil {
			return fmt.Errorf("save extension setting %s: %w", name, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit extension settings update: %w", err)
	}
	return nil
}

func (s *PostgresStore) CompareAndSwapSetting(ctx context.Context, extensionID, name, oldValue, newValue string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE extension_settings
		SET value = $4, updated_at = now()
		WHERE extension_id = $1 AND name = $2 AND value = $3
	`, extensionID, name, oldValue, newValue)
	if err != nil {
		return false, fmt.Errorf("compare and swap extension setting %s: %w", name, err)
	}
	return tag.RowsAffected() == 1, nil
}

func (s *PostgresStore) ResetSettings(ctx context.Context, extensionID string) error {
	if _, err := s.pool.Exec(ctx, "DELETE FROM extension_settings WHERE extension_id = $1", extensionID); err != nil {
		return fmt.Errorf("reset extension settings: %w", err)
	}
	return nil
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
		  extensions.active_version_id,
		  extension_versions.version, extension_versions.manifest, extension_versions.package_digest,
		  extension_versions.admin_frontend_digest,
		  extension_versions.package_path,
		  extension_versions.installed_at, extensions.updated_at,
		  COALESCE(staged_versions.id, 0),
		  CASE WHEN staged_versions.id IS NULL THEN NULL ELSE jsonb_build_object(
		    'version', staged_versions.version,
		    'manifest', staged_versions.manifest,
		    'packageDigest', staged_versions.package_digest,
		    'adminFrontendDigest', staged_versions.admin_frontend_digest,
		    'packagePath', staged_versions.package_path,
		    'installedAt', staged_versions.installed_at
		  ) END
		FROM extensions
		JOIN extension_versions ON extension_versions.id = extensions.active_version_id
		LEFT JOIN extension_versions AS staged_versions ON staged_versions.id = extensions.staged_version_id
	`
}

type extensionRow interface {
	Scan(dest ...any) error
}

func scanExtension(row extensionRow) (Extension, error) {
	var item Extension
	var manifestJSON []byte
	var stagedVersionJSON []byte
	var stagedVersionID int64
	if err := row.Scan(
		&item.ID,
		&item.Name,
		&item.Type,
		&item.Status,
		&item.Source,
		&item.IsSystem,
		&item.IsDeletable,
		&item.ActiveVersionID,
		&item.Version,
		&manifestJSON,
		&item.PackageDigest,
		&item.AdminFrontendDigest,
		&item.PackagePath,
		&item.InstalledAt,
		&item.UpdatedAt,
		&stagedVersionID,
		&stagedVersionJSON,
	); err != nil {
		return Extension{}, err
	}
	if err := json.Unmarshal(manifestJSON, &item.Manifest); err != nil {
		return Extension{}, fmt.Errorf("decode extension manifest: %w", err)
	}
	if len(stagedVersionJSON) > 0 {
		var staged ExtensionVersion
		if err := json.Unmarshal(stagedVersionJSON, &staged); err != nil {
			return Extension{}, fmt.Errorf("decode staged extension version: %w", err)
		}
		staged.ID = stagedVersionID
		item.StagedVersion = &staged
	}
	return item, nil
}

type extensionVersionInput struct {
	ExtensionID         string
	Version             string
	ManifestJSON        []byte
	PackagePath         string
	PackageDigest       string
	AdminFrontendDigest string
}

// ensureExtensionVersion 以完整包摘要区分同版本的不可变安装内容。
// 完全相同的三元组只复用原记录，不允许后续安装覆盖已批准的清单或包路径。
func ensureExtensionVersion(ctx context.Context, tx pgx.Tx, input extensionVersionInput) (int64, error) {
	var versionID int64
	err := tx.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO extension_versions (
				 extension_id, version, manifest, package_path, package_digest, admin_frontend_digest
			)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (extension_id, version, package_digest) DO NOTHING
			RETURNING id
		)
		SELECT id FROM inserted
		UNION ALL
		SELECT id
		FROM extension_versions
		WHERE extension_id = $1
		  AND version = $2
		  AND package_digest = $5
		LIMIT 1
	`, input.ExtensionID, input.Version, input.ManifestJSON, input.PackagePath, input.PackageDigest, input.AdminFrontendDigest).Scan(&versionID)
	return versionID, err
}

func nullableActorID(userID int64) any {
	if userID == 0 {
		return nil
	}
	return userID
}
