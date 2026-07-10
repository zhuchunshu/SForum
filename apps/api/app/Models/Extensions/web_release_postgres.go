package extensions

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const webReleaseColumns = `
	id, desired_generation, trigger_kind, trigger_extension_id,
	composition_hash, composition_snapshot, active_theme_id, theme_version, theme_layer_path,
	theme_package_digest, status, activation_checkpoint, reload_mode,
	artifact_path, artifact_digest, server_entry, build_log,
	public_reason, public_message, previous_release_id,
	requested_by_user_id, activated_by_user_id, created_at, updated_at,
	ready_at, activation_started_at, activated_at, completed_at
`

type webReleaseRow interface {
	Scan(...any) error
}

type webReleaseRows interface {
	Close()
	Err() error
	Next() bool
	Scan(...any) error
}

type webReleaseSQL interface {
	Exec(context.Context, string, ...any) (int64, error)
	Query(context.Context, string, ...any) (webReleaseRows, error)
	QueryRow(context.Context, string, ...any) webReleaseRow
}

type webReleaseTransaction interface {
	webReleaseSQL
	Commit(context.Context) error
	Rollback(context.Context) error
}

type webReleaseDatabase interface {
	webReleaseSQL
	Begin(context.Context) (webReleaseTransaction, error)
}

type pgxWebReleaseSQL struct {
	queryer interface {
		Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
		Query(context.Context, string, ...any) (pgx.Rows, error)
		QueryRow(context.Context, string, ...any) pgx.Row
	}
}

func (p pgxWebReleaseSQL) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	tag, err := p.queryer.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (p pgxWebReleaseSQL) Query(ctx context.Context, query string, args ...any) (webReleaseRows, error) {
	return p.queryer.Query(ctx, query, args...)
}

func (p pgxWebReleaseSQL) QueryRow(ctx context.Context, query string, args ...any) webReleaseRow {
	return p.queryer.QueryRow(ctx, query, args...)
}

type pgxWebReleaseTx struct {
	pgxWebReleaseSQL
	tx pgx.Tx
}

func (p pgxWebReleaseTx) Commit(ctx context.Context) error {
	return p.tx.Commit(ctx)
}

func (p pgxWebReleaseTx) Rollback(ctx context.Context) error {
	return p.tx.Rollback(ctx)
}

type pgxWebReleaseDB struct {
	pgxWebReleaseSQL
	pool *pgxpool.Pool
}

func newPGXWebReleaseDB(pool *pgxpool.Pool) pgxWebReleaseDB {
	return pgxWebReleaseDB{
		pgxWebReleaseSQL: pgxWebReleaseSQL{queryer: pool},
		pool:             pool,
	}
}

func (p pgxWebReleaseDB) Begin(ctx context.Context) (webReleaseTransaction, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return pgxWebReleaseTx{pgxWebReleaseSQL: pgxWebReleaseSQL{queryer: tx}, tx: tx}, nil
}

type PostgresWebReleaseStore struct {
	db webReleaseDatabase
}

func NewPostgresWebReleaseStore(pool *pgxpool.Pool) *PostgresWebReleaseStore {
	return &PostgresWebReleaseStore{db: newPGXWebReleaseDB(pool)}
}

func newPostgresWebReleaseStore(db webReleaseDatabase) *PostgresWebReleaseStore {
	return &PostgresWebReleaseStore{db: db}
}

func (s *PostgresWebReleaseStore) CreateWebRelease(ctx context.Context, input WebReleaseCreateInput) (WebRelease, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return WebRelease{}, fmt.Errorf("begin web release creation: %w", err)
	}
	defer tx.Rollback(ctx)

	release, err := createWebRelease(ctx, tx, input)
	if err != nil {
		return WebRelease{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return WebRelease{}, fmt.Errorf("commit web release creation: %w", err)
	}
	return release, nil
}

func (s *PostgresWebReleaseStore) CreateWebReleaseTx(ctx context.Context, tx pgx.Tx, input WebReleaseCreateInput) (WebRelease, error) {
	runner := pgxWebReleaseSQL{queryer: tx}
	return createWebRelease(ctx, runner, input)
}

func createWebRelease(ctx context.Context, tx webReleaseSQL, input WebReleaseCreateInput) (WebRelease, error) {
	reloadMode := strings.TrimSpace(input.ReloadMode)
	if reloadMode == "" {
		reloadMode = WebReleaseReloadPrompt
	}
	compositionSnapshot, err := canonicalJSONObject(input.CompositionSnapshot)
	if err != nil {
		return WebRelease{}, fmt.Errorf("marshal web release composition snapshot: %w", err)
	}
	compositionDigest := fmt.Sprintf("%x", sha256.Sum256(compositionSnapshot))
	if compositionDigest != input.CompositionHash {
		return WebRelease{}, fmt.Errorf("%w: expected %s, got %s", ErrWebReleaseCompositionMismatch, compositionDigest, input.CompositionHash)
	}
	release, err := scanWebRelease(tx.QueryRow(ctx, `
		INSERT INTO web_releases (
			trigger_kind, trigger_extension_id,
			composition_hash, composition_snapshot, active_theme_id, theme_version, theme_layer_path,
			theme_package_digest, status, activation_checkpoint, reload_mode,
			previous_release_id, requested_by_user_id
		)
		VALUES (
			$1, $2, $3, $4::jsonb, $5, $6, $7, $8, 'queued', 'pending', $9, $10, $11
		)
		RETURNING `+webReleaseColumns,
		input.TriggerKind,
		input.TriggerExtensionID,
		input.CompositionHash,
		compositionSnapshot,
		input.ActiveThemeID,
		input.ThemeVersion,
		input.ThemeLayerPath,
		input.ThemePackageDigest,
		reloadMode,
		input.PreviousReleaseID,
		input.RequestedByUserID,
	))
	if err != nil {
		return WebRelease{}, fmt.Errorf("insert web release: %w", err)
	}

	for _, extension := range input.Extensions {
		componentMap, err := marshalJSONObject(extension.ComponentMap)
		if err != nil {
			return WebRelease{}, fmt.Errorf("marshal component map for %s: %w", extension.ExtensionID, err)
		}
		trustedComponents, err := marshalJSONArray(extension.TrustedComponents)
		if err != nil {
			return WebRelease{}, fmt.Errorf("marshal trusted components for %s: %w", extension.ExtensionID, err)
		}
		localeMap, err := marshalJSONObject(extension.LocaleMap)
		if err != nil {
			return WebRelease{}, fmt.Errorf("marshal locale map for %s: %w", extension.ExtensionID, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO web_release_extensions (
				web_release_id, extension_id, extension_version, package_digest,
				frontend_root, component_map, api_version, trusted_components,
				locale_map, locale_map_digest, lockfile_digest,
				resolved_dependencies, resolved_dependency_snapshot_digest, sort_order
			)
			VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8::jsonb,
			        $9::jsonb, $10, $11, '[]'::jsonb, '', $12)
		`,
			release.ID,
			extension.ExtensionID,
			extension.ExtensionVersion,
			extension.PackageDigest,
			extension.FrontendRoot,
			componentMap,
			extension.APIVersion,
			trustedComponents,
			localeMap,
			extension.LocaleMapDigest,
			extension.LockfileDigest,
			extension.SortOrder,
		); err != nil {
			return WebRelease{}, fmt.Errorf("insert web release extension %s: %w", extension.ExtensionID, err)
		}
	}

	for _, effect := range input.Effects {
		if _, err := tx.Exec(ctx, `
			INSERT INTO web_release_extension_effects (
				web_release_id, extension_id, previous_status, target_status,
				activation_checkpoint
			)
			VALUES ($1, $2, $3, $4, 'pending')
		`, release.ID, effect.ExtensionID, effect.PreviousStatus, effect.TargetStatus); err != nil {
			return WebRelease{}, fmt.Errorf("insert web release effect %s: %w", effect.ExtensionID, err)
		}
	}

	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = "web_release.queued"
	}
	if err := insertWebReleaseEvent(ctx, tx, release.ID, input.RequestedByUserID, nil, WebReleaseQueued, reason, input.Message); err != nil {
		return WebRelease{}, err
	}
	return release, nil
}

func (s *PostgresWebReleaseStore) TransitionWebRelease(ctx context.Context, input WebReleaseTransitionInput) (WebRelease, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return WebRelease{}, fmt.Errorf("begin web release transition: %w", err)
	}
	defer tx.Rollback(ctx)

	release, err := transitionWebRelease(ctx, tx, input)
	if err != nil {
		return WebRelease{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return WebRelease{}, fmt.Errorf("commit web release transition: %w", err)
	}
	return release, nil
}

func transitionWebRelease(ctx context.Context, tx webReleaseSQL, input WebReleaseTransitionInput) (WebRelease, error) {
	var currentText string
	var previousReleaseID sql.NullInt64
	if err := tx.QueryRow(ctx, `
		SELECT status, previous_release_id
		FROM web_releases
		WHERE id = $1
		FOR UPDATE
	`, input.ID).Scan(&currentText, &previousReleaseID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WebRelease{}, ErrWebReleaseNotFound
		}
		return WebRelease{}, fmt.Errorf("lock web release: %w", err)
	}
	current := WebReleaseStatus(currentText)
	if current != input.ExpectedStatus {
		return WebRelease{}, fmt.Errorf("%w: expected %s, found %s", ErrWebReleaseStale, input.ExpectedStatus, current)
	}
	if err := ValidateWebReleaseTransitionWithOptions(current, input.NextStatus, TransitionOptions{Compensation: input.Compensation}); err != nil {
		return WebRelease{}, err
	}
	if current == input.NextStatus {
		release, err := scanWebRelease(tx.QueryRow(ctx, `
			SELECT `+webReleaseColumns+`
			FROM web_releases
			WHERE id = $1
		`, input.ID))
		if err != nil {
			return WebRelease{}, fmt.Errorf("reload idempotent web release transition: %w", err)
		}
		return release, nil
	}

	if input.NextStatus == WebReleaseActive {
		if err := replaceActiveWebRelease(ctx, tx, input, nullInt64Pointer(previousReleaseID)); err != nil {
			return WebRelease{}, err
		}
	}

	release, err := scanWebRelease(tx.QueryRow(ctx, `
		UPDATE web_releases
		SET status = $3,
		    activation_checkpoint = CASE WHEN $4 = '' THEN activation_checkpoint ELSE $4 END,
		    artifact_path = CASE WHEN $5 = '' THEN artifact_path ELSE $5 END,
		    artifact_digest = CASE WHEN $6 = '' THEN artifact_digest ELSE $6 END,
		    server_entry = CASE WHEN $7 = '' THEN server_entry ELSE $7 END,
		    build_log = CASE WHEN $8 = '' THEN build_log ELSE $8 END,
		    public_reason = CASE WHEN $9 = '' THEN public_reason ELSE $9 END,
		    public_message = CASE WHEN $10 = '' THEN public_message ELSE $10 END,
		    activated_by_user_id = CASE
		      WHEN $3 = 'active' THEN COALESCE($11, activated_by_user_id)
		      ELSE activated_by_user_id
		    END,
		    ready_at = CASE WHEN $3 = 'ready' THEN COALESCE(ready_at, now()) ELSE ready_at END,
		    activation_started_at = CASE
		      WHEN $3 = 'activating' THEN COALESCE(activation_started_at, now())
		      ELSE activation_started_at
		    END,
		    activated_at = CASE WHEN $3 = 'active' THEN COALESCE(activated_at, now()) ELSE activated_at END,
		    completed_at = CASE
		      WHEN $3 IN ('active', 'inactive', 'failed', 'superseded', 'rolled_back')
		        THEN COALESCE(completed_at, now())
		      ELSE completed_at
		    END,
		    updated_at = now()
		WHERE id = $1 AND status = $2
		RETURNING `+webReleaseColumns,
		input.ID,
		string(input.ExpectedStatus),
		string(input.NextStatus),
		input.ActivationCheckpoint,
		input.ArtifactPath,
		input.ArtifactDigest,
		input.ServerEntry,
		input.BuildLog,
		input.PublicReason,
		input.PublicMessage,
		input.ActivatedByUserID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return WebRelease{}, ErrWebReleaseStale
	}
	if err != nil {
		return WebRelease{}, fmt.Errorf("update web release: %w", err)
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = "web_release." + string(input.NextStatus)
	}
	if err := insertWebReleaseEvent(ctx, tx, release.ID, input.ActorUserID, &current, input.NextStatus, reason, input.Message); err != nil {
		return WebRelease{}, err
	}
	return release, nil
}

func replaceActiveWebRelease(
	ctx context.Context,
	tx webReleaseSQL,
	input WebReleaseTransitionInput,
	expectedActiveReleaseID *int64,
) error {
	var activeID int64
	err := tx.QueryRow(ctx, `
		SELECT id
		FROM web_releases
		WHERE status = 'active' AND id <> $1
		ORDER BY desired_generation DESC, id DESC
		LIMIT 1
		FOR UPDATE
	`, input.ID).Scan(&activeID)
	if errors.Is(err, pgx.ErrNoRows) {
		if expectedActiveReleaseID != nil {
			return fmt.Errorf("%w: expected active web release %d, found none", ErrWebReleaseStale, *expectedActiveReleaseID)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("lock active web release: %w", err)
	}
	if expectedActiveReleaseID == nil {
		return fmt.Errorf("%w: expected no active web release, found %d", ErrWebReleaseStale, activeID)
	}
	if activeID != *expectedActiveReleaseID {
		return fmt.Errorf("%w: expected active web release %d, found %d", ErrWebReleaseStale, *expectedActiveReleaseID, activeID)
	}

	next := WebReleaseInactive
	reason := "web_release.replaced"
	message := fmt.Sprintf("Replaced by web release %d", input.ID)
	options := TransitionOptions{}
	if input.Compensation {
		next = WebReleaseRolledBack
		reason = "web_release.compensated"
		message = fmt.Sprintf("Compensated by web release %d", input.ID)
		options.Compensation = true
	}
	if err := ValidateWebReleaseTransitionWithOptions(WebReleaseActive, next, options); err != nil {
		return err
	}
	affected, err := tx.Exec(ctx, `
		UPDATE web_releases
		SET status = $2,
		    public_reason = $3,
		    public_message = $4,
		    completed_at = COALESCE(completed_at, now()),
		    updated_at = now()
		WHERE id = $1 AND status = 'active'
	`, activeID, string(next), reason, message)
	if err != nil {
		return fmt.Errorf("replace active web release: %w", err)
	}
	if affected != 1 {
		return ErrWebReleaseStale
	}
	previous := WebReleaseActive
	return insertWebReleaseEvent(ctx, tx, activeID, input.ActorUserID, &previous, next, reason, message)
}

func insertWebReleaseEvent(
	ctx context.Context,
	tx webReleaseSQL,
	releaseID int64,
	actorUserID *int64,
	previous *WebReleaseStatus,
	next WebReleaseStatus,
	reason string,
	message string,
) error {
	var previousValue any
	if previous != nil {
		previousValue = string(*previous)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO web_release_events (
			web_release_id, actor_user_id, previous_status, next_status, reason, message
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, releaseID, actorUserID, previousValue, string(next), reason, message); err != nil {
		return fmt.Errorf("insert web release event: %w", err)
	}
	return nil
}

func (s *PostgresWebReleaseStore) WebRelease(ctx context.Context, id int64) (WebReleaseDetail, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return WebReleaseDetail{}, fmt.Errorf("begin web release detail read: %w", err)
	}
	defer tx.Rollback(ctx)

	release, err := scanWebRelease(tx.QueryRow(ctx, `
		SELECT `+webReleaseColumns+`
		FROM web_releases
		WHERE id = $1
		FOR SHARE
	`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return WebReleaseDetail{}, ErrWebReleaseNotFound
	}
	if err != nil {
		return WebReleaseDetail{}, fmt.Errorf("load web release: %w", err)
	}
	detail := WebReleaseDetail{WebRelease: release}
	if detail.Extensions, err = s.listWebReleaseExtensions(ctx, tx, id); err != nil {
		return WebReleaseDetail{}, err
	}
	if detail.Effects, err = s.listWebReleaseEffects(ctx, tx, id); err != nil {
		return WebReleaseDetail{}, err
	}
	if detail.Events, err = s.listWebReleaseEvents(ctx, tx, id); err != nil {
		return WebReleaseDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return WebReleaseDetail{}, fmt.Errorf("commit web release detail read: %w", err)
	}
	return detail, nil
}

func (s *PostgresWebReleaseStore) listWebReleaseExtensions(ctx context.Context, db webReleaseSQL, releaseID int64) ([]WebReleaseExtension, error) {
	rows, err := db.Query(ctx, `
		SELECT web_release_id, extension_id, extension_version, package_digest,
		       frontend_root, component_map, api_version, trusted_components,
		       locale_map, locale_map_digest, lockfile_digest,
		       resolved_dependencies, resolved_dependency_snapshot_digest,
		       sort_order, created_at
		FROM web_release_extensions
		WHERE web_release_id = $1
		ORDER BY sort_order, extension_id
	`, releaseID)
	if err != nil {
		return nil, fmt.Errorf("list web release extensions: %w", err)
	}
	defer rows.Close()
	items := []WebReleaseExtension{}
	for rows.Next() {
		item, err := scanWebReleaseExtension(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate web release extensions: %w", err)
	}
	return items, nil
}

func (s *PostgresWebReleaseStore) listWebReleaseEffects(ctx context.Context, db webReleaseSQL, releaseID int64) ([]WebReleaseExtensionEffect, error) {
	rows, err := db.Query(ctx, `
		SELECT web_release_id, extension_id, previous_status, target_status,
		       activation_checkpoint, public_reason, public_message,
		       created_at, updated_at, compensated_at
		FROM web_release_extension_effects
		WHERE web_release_id = $1
		ORDER BY extension_id
	`, releaseID)
	if err != nil {
		return nil, fmt.Errorf("list web release effects: %w", err)
	}
	defer rows.Close()
	items := []WebReleaseExtensionEffect{}
	for rows.Next() {
		item, err := scanWebReleaseEffect(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate web release effects: %w", err)
	}
	return items, nil
}

func (s *PostgresWebReleaseStore) listWebReleaseEvents(ctx context.Context, db webReleaseSQL, releaseID int64) ([]WebReleaseEvent, error) {
	rows, err := db.Query(ctx, `
		SELECT id, web_release_id, actor_user_id, previous_status,
		       next_status, reason, message, created_at
		FROM web_release_events
		WHERE web_release_id = $1
		ORDER BY created_at, id
	`, releaseID)
	if err != nil {
		return nil, fmt.Errorf("list web release events: %w", err)
	}
	defer rows.Close()
	items := []WebReleaseEvent{}
	for rows.Next() {
		item, err := scanWebReleaseEvent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate web release events: %w", err)
	}
	return items, nil
}

func (s *PostgresWebReleaseStore) ListWebReleases(ctx context.Context, input WebReleaseListInput) (WebReleasePage, error) {
	page, perPage := normalizeWebReleasePage(input.Page, input.PerPage)
	status := string(input.Status)
	var total int64
	if err := s.db.QueryRow(ctx, `
		SELECT count(*)
		FROM web_releases
		WHERE ($1 = '' OR status = $1)
	`, status).Scan(&total); err != nil {
		return WebReleasePage{}, fmt.Errorf("count web releases: %w", err)
	}
	rows, err := s.db.Query(ctx, `
		SELECT `+webReleaseColumns+`
		FROM web_releases
		WHERE ($1 = '' OR status = $1)
		ORDER BY desired_generation DESC, id DESC
		LIMIT $2 OFFSET $3
	`, status, perPage, (page-1)*perPage)
	if err != nil {
		return WebReleasePage{}, fmt.Errorf("list web releases: %w", err)
	}
	defer rows.Close()
	items := []WebRelease{}
	for rows.Next() {
		item, err := scanWebRelease(rows)
		if err != nil {
			return WebReleasePage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return WebReleasePage{}, fmt.Errorf("iterate web releases: %w", err)
	}
	return WebReleasePage{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func normalizeWebReleasePage(page int, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

func (s *PostgresWebReleaseStore) RecordWebReleaseDependencySnapshot(
	ctx context.Context,
	input WebReleaseDependencySnapshotInput,
) error {
	if strings.TrimSpace(input.Digest) == "" {
		return fmt.Errorf("%w: dependency snapshot digest is empty", ErrWebReleaseDependencySnapshotConflict)
	}
	dependencies := append([]Dependency(nil), input.ResolvedDependencies...)
	sort.Slice(dependencies, func(i, j int) bool {
		if dependencies[i].Name != dependencies[j].Name {
			return dependencies[i].Name < dependencies[j].Name
		}
		if dependencies[i].Version != dependencies[j].Version {
			return dependencies[i].Version < dependencies[j].Version
		}
		return dependencies[i].Integrity < dependencies[j].Integrity
	})
	resolved, err := marshalJSONArray(dependencies)
	if err != nil {
		return fmt.Errorf("marshal resolved dependency snapshot: %w", err)
	}
	affected, err := s.db.Exec(ctx, `
		UPDATE web_release_extensions AS extension_snapshot
		SET resolved_dependencies = $3::jsonb,
		    resolved_dependency_snapshot_digest = $4
		FROM web_releases AS release
		WHERE extension_snapshot.web_release_id = $1
		  AND extension_snapshot.extension_id = $2
		  AND release.id = extension_snapshot.web_release_id
		  AND (
		    (
		      extension_snapshot.resolved_dependency_snapshot_digest = ''
		      AND release.status IN ('resolving', 'installing', 'building')
		    )
		    OR (
		      extension_snapshot.resolved_dependency_snapshot_digest = $4
		      AND extension_snapshot.resolved_dependencies = $3::jsonb
		    )
		  )
	`, input.WebReleaseID, input.ExtensionID, resolved, input.Digest)
	if err != nil {
		return fmt.Errorf("record resolved dependency snapshot: %w", err)
	}
	if affected == 1 {
		return nil
	}
	var currentDigest string
	var currentStatus string
	if err := s.db.QueryRow(ctx, `
		SELECT extension_snapshot.resolved_dependency_snapshot_digest, release.status
		FROM web_release_extensions AS extension_snapshot
		JOIN web_releases AS release ON release.id = extension_snapshot.web_release_id
		WHERE extension_snapshot.web_release_id = $1 AND extension_snapshot.extension_id = $2
	`, input.WebReleaseID, input.ExtensionID).Scan(&currentDigest, &currentStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrWebReleaseNotFound
		}
		return fmt.Errorf("load resolved dependency snapshot: %w", err)
	}
	return fmt.Errorf(
		"%w: web release %d extension %s in status %s already records digest %s",
		ErrWebReleaseDependencySnapshotConflict,
		input.WebReleaseID,
		input.ExtensionID,
		currentStatus,
		currentDigest,
	)
}

func scanWebRelease(row webReleaseRow) (WebRelease, error) {
	var release WebRelease
	var status string
	var previousReleaseID sql.NullInt64
	var requestedByUserID sql.NullInt64
	var activatedByUserID sql.NullInt64
	var readyAt sql.NullTime
	var activationStartedAt sql.NullTime
	var activatedAt sql.NullTime
	var completedAt sql.NullTime
	if err := row.Scan(
		&release.ID,
		&release.DesiredGeneration,
		&release.TriggerKind,
		&release.TriggerExtensionID,
		&release.CompositionHash,
		&release.CompositionSnapshot,
		&release.ActiveThemeID,
		&release.ThemeVersion,
		&release.ThemeLayerPath,
		&release.ThemePackageDigest,
		&status,
		&release.ActivationCheckpoint,
		&release.ReloadMode,
		&release.ArtifactPath,
		&release.ArtifactDigest,
		&release.ServerEntry,
		&release.BuildLog,
		&release.PublicReason,
		&release.PublicMessage,
		&previousReleaseID,
		&requestedByUserID,
		&activatedByUserID,
		&release.CreatedAt,
		&release.UpdatedAt,
		&readyAt,
		&activationStartedAt,
		&activatedAt,
		&completedAt,
	); err != nil {
		return WebRelease{}, err
	}
	release.Status = WebReleaseStatus(status)
	release.PreviousReleaseID = nullInt64Pointer(previousReleaseID)
	release.RequestedByUserID = nullInt64Pointer(requestedByUserID)
	release.ActivatedByUserID = nullInt64Pointer(activatedByUserID)
	release.ReadyAt = nullTimePointer(readyAt)
	release.ActivationStartedAt = nullTimePointer(activationStartedAt)
	release.ActivatedAt = nullTimePointer(activatedAt)
	release.CompletedAt = nullTimePointer(completedAt)
	return release, nil
}

func scanWebReleaseExtension(row webReleaseRow) (WebReleaseExtension, error) {
	var item WebReleaseExtension
	var componentMap []byte
	var trustedComponents []byte
	var localeMap []byte
	var resolvedDependencies []byte
	if err := row.Scan(
		&item.WebReleaseID,
		&item.ExtensionID,
		&item.ExtensionVersion,
		&item.PackageDigest,
		&item.FrontendRoot,
		&componentMap,
		&item.APIVersion,
		&trustedComponents,
		&localeMap,
		&item.LocaleMapDigest,
		&item.LockfileDigest,
		&resolvedDependencies,
		&item.ResolvedDependencySnapshotDigest,
		&item.SortOrder,
		&item.CreatedAt,
	); err != nil {
		return WebReleaseExtension{}, err
	}
	if err := json.Unmarshal(componentMap, &item.ComponentMap); err != nil {
		return WebReleaseExtension{}, fmt.Errorf("decode web release component map: %w", err)
	}
	if err := json.Unmarshal(trustedComponents, &item.TrustedComponents); err != nil {
		return WebReleaseExtension{}, fmt.Errorf("decode web release trusted components: %w", err)
	}
	if err := json.Unmarshal(localeMap, &item.LocaleMap); err != nil {
		return WebReleaseExtension{}, fmt.Errorf("decode web release locale map: %w", err)
	}
	if err := json.Unmarshal(resolvedDependencies, &item.ResolvedDependencies); err != nil {
		return WebReleaseExtension{}, fmt.Errorf("decode web release dependencies: %w", err)
	}
	return item, nil
}

func scanWebReleaseEffect(row webReleaseRow) (WebReleaseExtensionEffect, error) {
	var item WebReleaseExtensionEffect
	var compensatedAt sql.NullTime
	if err := row.Scan(
		&item.WebReleaseID,
		&item.ExtensionID,
		&item.PreviousStatus,
		&item.TargetStatus,
		&item.ActivationCheckpoint,
		&item.PublicReason,
		&item.PublicMessage,
		&item.CreatedAt,
		&item.UpdatedAt,
		&compensatedAt,
	); err != nil {
		return WebReleaseExtensionEffect{}, err
	}
	item.CompensatedAt = nullTimePointer(compensatedAt)
	return item, nil
}

func scanWebReleaseEvent(row webReleaseRow) (WebReleaseEvent, error) {
	var item WebReleaseEvent
	var actorUserID sql.NullInt64
	var previousStatus sql.NullString
	var nextStatus string
	if err := row.Scan(
		&item.ID,
		&item.WebReleaseID,
		&actorUserID,
		&previousStatus,
		&nextStatus,
		&item.Reason,
		&item.Message,
		&item.CreatedAt,
	); err != nil {
		return WebReleaseEvent{}, err
	}
	item.ActorUserID = nullInt64Pointer(actorUserID)
	if previousStatus.Valid {
		status := WebReleaseStatus(previousStatus.String)
		item.PreviousStatus = &status
	}
	item.NextStatus = WebReleaseStatus(nextStatus)
	return item, nil
}

func marshalJSONObject[T any](value map[string]T) ([]byte, error) {
	if value == nil {
		value = map[string]T{}
	}
	return json.Marshal(value)
}

func marshalJSONArray[T any](value []T) ([]byte, error) {
	if value == nil {
		value = []T{}
	}
	return json.Marshal(value)
}

func canonicalJSONObject(value json.RawMessage) ([]byte, error) {
	if len(value) == 0 {
		return []byte(`{}`), nil
	}
	var object map[string]any
	if err := json.Unmarshal(value, &object); err != nil {
		return nil, err
	}
	if object == nil {
		return nil, errors.New("JSON value must be an object")
	}
	return json.Marshal(object)
}

func nullInt64Pointer(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	result := value.Int64
	return &result
}

func nullTimePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
