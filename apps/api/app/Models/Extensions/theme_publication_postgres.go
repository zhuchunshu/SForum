package extensions

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type themePublicationTuple struct {
	ID        string
	VersionID int64
	Version   string
	Digest    string
}

const themeRuntimeActivationLockKey = "sforum.theme.activation.v1"

func (t themePublicationTuple) matches(id, version, digest string) bool {
	return t.ID == id && t.Version == version && strings.EqualFold(t.Digest, digest)
}

func (s *PostgresStore) activateThemePublished(
	ctx context.Context,
	id string,
	expected *ThemeActivationInput,
	reason ThemeRuntimePublicationReason,
	actorUserID int64,
	approveCoreReplacements bool,
	hostApproveProtectedDefault bool,
) (ThemeActivationResult, error) {
	if s == nil || s.pool == nil || ctx == nil || !validThemePublicationReason(reason) ||
		(approveCoreReplacements && actorUserID <= 0) {
		return ThemeActivationResult{}, ErrThemePublicationConflict
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ThemeActivationResult{}, fmt.Errorf("begin theme activation publication: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, themeRuntimeActivationLockKey); err != nil {
		return ThemeActivationResult{}, fmt.Errorf("lock theme activation publication: %w", err)
	}

	current, found, err := lockCurrentThemeTuple(ctx, tx)
	if err != nil {
		return ThemeActivationResult{}, err
	}
	var target themePublicationTuple
	var promoteStaged bool
	if expected == nil {
		target, err = lockThemeTuple(ctx, tx, id)
	} else {
		target, promoteStaged, err = lockExactThemeActivationTarget(ctx, tx, id, expected.Version, expected.PackageDigest)
	}
	if err != nil {
		return ThemeActivationResult{}, err
	}
	if expected != nil {
		if !current.matches(expected.CurrentThemeID, expected.CurrentThemeVersion, expected.CurrentThemeDigest) ||
			(!found && (expected.CurrentThemeID != "" || expected.CurrentThemeVersion != "" || expected.CurrentThemeDigest != "")) ||
			!target.matches(id, expected.Version, expected.PackageDigest) {
			return ThemeActivationResult{}, ErrThemePreviewStale
		}
	}
	sourceApproved, sourceActor, sourceApprovalSource, err := lockThemeApproval(ctx, tx, current.ID)
	if err != nil {
		return ThemeActivationResult{}, err
	}
	approvalSource := ThemeApprovalSourceAdmin
	if !approveCoreReplacements && actorUserID == 0 && hostApproveProtectedDefault {
		// 只有 Host 自带的受保护默认主题可以在没有人类 actor 的启动修复/
		// 首次导入中由 Host 自己批准核心页替换。精确制品已有人工审批时
		// 保留人工来源，不降级成 host。
		protectedDefault, protectedErr := lockProtectedBuiltinDefault(ctx, tx, target.ID)
		if protectedErr != nil {
			return ThemeActivationResult{}, protectedErr
		}
		if protectedDefault {
			approved, approvedActor, approvedSource, approvalErr := lockThemeExactArtifactApproval(
				ctx, tx, target.ID, target.Version, target.Digest,
			)
			if approvalErr != nil {
				return ThemeActivationResult{}, approvalErr
			}
			approveCoreReplacements = true
			if approved && approvedSource == ThemeApprovalSourceAdmin && approvedActor > 0 {
				actorUserID, approvalSource = approvedActor, ThemeApprovalSourceAdmin
			} else {
				approvalSource = ThemeApprovalSourceHost
			}
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE extensions SET status = 'disabled', updated_at = now()
		WHERE type = 'theme' AND id <> $1 AND status = 'enabled'
	`, id); err != nil {
		return ThemeActivationResult{}, fmt.Errorf("disable previous active theme: %w", err)
	}
	command, err := tx.Exec(ctx, `
		UPDATE extensions
		SET status = 'enabled',
		    active_version_id = CASE WHEN $2 THEN $3 ELSE active_version_id END,
		    staged_version_id = CASE WHEN $2 THEN NULL ELSE staged_version_id END,
		    updated_at = now()
		WHERE id = $1 AND type = 'theme'
	`, id, promoteStaged, target.VersionID)
	if err != nil {
		return ThemeActivationResult{}, fmt.Errorf("activate theme: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ThemeActivationResult{}, ErrExtensionNotFound
	}
	publication, err := insertThemeRuntimePublication(ctx, tx, ThemeRuntimePublication{
		DesiredState: ThemeRuntimePublicationActive,
		ThemeID:      target.ID, ThemeVersion: target.Version, PackageDigest: target.Digest,
		SourceThemeID: current.ID, SourceThemeVersion: current.Version, SourcePackageDigest: current.Digest,
		SourceCoreReplacementsApproved: sourceApproved, SourceActorUserID: sourceActor,
		SourceApprovalSource:     sourceApprovalSource,
		CoreReplacementsApproved: approveCoreReplacements,
		ApprovalSource:           approvalSource,
		ActorUserID:              actorUserID, Reason: reason,
	})
	if err != nil {
		return ThemeActivationResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ThemeActivationResult{}, fmt.Errorf("commit theme activation publication: %w", err)
	}
	extension, err := s.Get(ctx, id)
	if err != nil {
		return ThemeActivationResult{}, err
	}
	return ThemeActivationResult{Extension: extension, Publication: publication}, nil
}

// lockProtectedBuiltinDefault 判定目标是否是 Host 自带的受保护默认主题。
// 调用方已持有该 extensions 行的锁。
func lockProtectedBuiltinDefault(ctx context.Context, tx pgx.Tx, id string) (bool, error) {
	if id == "" {
		return false, nil
	}
	var source string
	var isSystem, isDeletable bool
	err := tx.QueryRow(ctx, `
		SELECT source, is_system, is_deletable FROM extensions WHERE id = $1
	`, id).Scan(&source, &isSystem, &isDeletable)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrExtensionNotFound
	}
	if err != nil {
		return false, err
	}
	return id == DefaultThemeID && source == SourceBuiltin && isSystem && !isDeletable, nil
}

func (s *PostgresStore) CompensateThemeActivation(
	ctx context.Context,
	failed ThemeRuntimePublication,
	previous *Extension,
) (ThemeActivationResult, error) {
	if s == nil || s.pool == nil || ctx == nil || failed.Revision <= 0 ||
		failed.DesiredState != ThemeRuntimePublicationActive {
		return ThemeActivationResult{}, ErrThemePublicationConflict
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ThemeActivationResult{}, fmt.Errorf("begin theme activation compensation: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, themeRuntimeActivationLockKey); err != nil {
		return ThemeActivationResult{}, fmt.Errorf("lock theme activation compensation: %w", err)
	}
	latest, err := loadLatestThemeRuntimePublication(ctx, tx, true)
	if err != nil || !sameThemeRuntimePublication(latest, failed) {
		return ThemeActivationResult{}, ErrThemePublicationConflict
	}
	current, found, err := lockCurrentThemeTuple(ctx, tx)
	if err != nil || !found || !current.matches(failed.ThemeID, failed.ThemeVersion, failed.PackageDigest) {
		return ThemeActivationResult{}, ErrThemePublicationConflict
	}

	publication := ThemeRuntimePublication{
		SourceThemeID: current.ID, SourceThemeVersion: current.Version, SourcePackageDigest: current.Digest,
		SourceCoreReplacementsApproved: failed.CoreReplacementsApproved,
		SourceActorUserID:              failed.ActorUserID,
		SourceApprovalSource:           failed.ApprovalSource,
		ActorUserID:                    failed.ActorUserID, Reason: ThemeRuntimePublicationCompensation,
	}
	if previous == nil {
		if _, err := tx.Exec(ctx, `
			UPDATE extensions SET status = 'disabled', updated_at = now()
			WHERE type = 'theme' AND status = 'enabled'
		`); err != nil {
			return ThemeActivationResult{}, fmt.Errorf("disable failed first theme activation: %w", err)
		}
		publication.DesiredState = ThemeRuntimePublicationNone
	} else {
		previousTuple, previousActiveVersionID, err := lockExactThemeVersionTuple(
			ctx, tx, previous.ID, previous.ActiveVersionID,
		)
		if err != nil || !previousTuple.matches(previous.ID, previous.Version, previous.PackageDigest) {
			return ThemeActivationResult{}, ErrThemePublicationConflict
		}
		if _, err := tx.Exec(ctx, `
			UPDATE extensions SET status = 'disabled', updated_at = now()
			WHERE type = 'theme' AND id <> $1 AND status = 'enabled'
		`, previous.ID); err != nil {
			return ThemeActivationResult{}, err
		}
		var command pgconn.CommandTag
		if previous.ID == current.ID {
			command, err = tx.Exec(ctx, `
				UPDATE extensions
				SET status = 'enabled',
				    active_version_id = $2,
				    staged_version_id = CASE WHEN $2 <> $3 THEN $3 ELSE NULL END,
				    updated_at = now()
				WHERE id = $1 AND type = 'theme' AND active_version_id = $3
			`, previous.ID, previousTuple.VersionID, current.VersionID)
		} else if previousActiveVersionID != previousTuple.VersionID {
			return ThemeActivationResult{}, ErrThemePublicationConflict
		} else {
			command, err = tx.Exec(ctx, `
				UPDATE extensions SET status = 'enabled', updated_at = now()
				WHERE id = $1 AND type = 'theme' AND active_version_id = $2
			`, previous.ID, previousTuple.VersionID)
		}
		if err != nil || command.RowsAffected() != 1 {
			return ThemeActivationResult{}, ErrThemePublicationConflict
		}
		publication.DesiredState = ThemeRuntimePublicationActive
		publication.ThemeID = previousTuple.ID
		publication.ThemeVersion = previousTuple.Version
		publication.PackageDigest = previousTuple.Digest
		publication.CoreReplacementsApproved = failed.SourceCoreReplacementsApproved
		if publication.CoreReplacementsApproved {
			publication.ActorUserID = failed.SourceActorUserID
			publication.ApprovalSource = failed.SourceApprovalSource
		}
	}
	publication, err = insertThemeRuntimePublication(ctx, tx, publication)
	if err != nil {
		return ThemeActivationResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ThemeActivationResult{}, fmt.Errorf("commit theme activation compensation: %w", err)
	}
	result := ThemeActivationResult{Publication: publication}
	if previous != nil {
		result.Extension, err = s.Get(ctx, previous.ID)
	}
	return result, err
}

func (s *PostgresStore) LatestThemeRuntimePublication(ctx context.Context) (ThemeRuntimePublication, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return ThemeRuntimePublication{}, ErrThemePublicationNotFound
	}
	return loadLatestThemeRuntimePublication(ctx, s.pool, false)
}

func (s *PostgresStore) ThemeRuntimePublicationByRevision(ctx context.Context, revision int64) (ThemeRuntimePublication, error) {
	if s == nil || s.pool == nil || ctx == nil || revision <= 0 {
		return ThemeRuntimePublication{}, ErrThemePublicationNotFound
	}
	return scanThemeRuntimePublication(s.pool.QueryRow(ctx, themeRuntimePublicationSelect+` WHERE revision = $1`, revision))
}

const themeRuntimePublicationSelect = `
	SELECT revision, desired_state, theme_id, theme_version, package_digest,
	       source_theme_id, source_theme_version, source_package_digest,
	       source_core_replacements_approved, source_actor_user_id, source_approval_source,
	       core_replacements_approved, actor_user_id, approval_source, reason, created_at
	FROM theme_runtime_publications`

type themePublicationQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadLatestThemeRuntimePublication(ctx context.Context, query themePublicationQuerier, lock bool) (ThemeRuntimePublication, error) {
	suffix := ` ORDER BY revision DESC LIMIT 1`
	if lock {
		suffix += ` FOR UPDATE`
	}
	return scanThemeRuntimePublication(query.QueryRow(ctx, themeRuntimePublicationSelect+suffix))
}

func scanThemeRuntimePublication(row pgx.Row) (ThemeRuntimePublication, error) {
	var result ThemeRuntimePublication
	var themeID, themeVersion, digest sql.NullString
	var sourceID, sourceVersion, sourceDigest sql.NullString
	var sourceActor, actor sql.NullInt64
	if err := row.Scan(
		&result.Revision, &result.DesiredState, &themeID, &themeVersion, &digest,
		&sourceID, &sourceVersion, &sourceDigest,
		&result.SourceCoreReplacementsApproved, &sourceActor, &result.SourceApprovalSource,
		&result.CoreReplacementsApproved, &actor, &result.ApprovalSource, &result.Reason, &result.CreatedAt,
	); errors.Is(err, pgx.ErrNoRows) {
		return ThemeRuntimePublication{}, ErrThemePublicationNotFound
	} else if err != nil {
		return ThemeRuntimePublication{}, err
	}
	result.ThemeID, result.ThemeVersion, result.PackageDigest = themeID.String, themeVersion.String, digest.String
	result.SourceThemeID, result.SourceThemeVersion, result.SourcePackageDigest = sourceID.String, sourceVersion.String, sourceDigest.String
	result.SourceActorUserID = sourceActor.Int64
	result.ActorUserID = actor.Int64
	return result, nil
}

func insertThemeRuntimePublication(
	ctx context.Context,
	tx pgx.Tx,
	publication ThemeRuntimePublication,
) (ThemeRuntimePublication, error) {
	if !validThemeRuntimePublication(publication) {
		return ThemeRuntimePublication{}, ErrThemePublicationConflict
	}
	err := tx.QueryRow(ctx, `
		INSERT INTO theme_runtime_publications (
			desired_state, theme_id, theme_version, package_digest,
			source_theme_id, source_theme_version, source_package_digest,
			source_core_replacements_approved, source_actor_user_id, source_approval_source,
			core_replacements_approved, actor_user_id, approval_source, reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING revision, created_at
	`, publication.DesiredState,
		nullableThemePublicationValue(publication.ThemeID), nullableThemePublicationValue(publication.ThemeVersion), nullableThemePublicationValue(publication.PackageDigest),
		nullableThemePublicationValue(publication.SourceThemeID), nullableThemePublicationValue(publication.SourceThemeVersion), nullableThemePublicationValue(publication.SourcePackageDigest),
		publication.SourceCoreReplacementsApproved, nullableThemePublicationActor(publication.SourceActorUserID),
		themeApprovalSourceOrAdmin(publication.SourceApprovalSource),
		publication.CoreReplacementsApproved, nullableThemePublicationActor(publication.ActorUserID),
		themeApprovalSourceOrAdmin(publication.ApprovalSource), publication.Reason,
	).Scan(&publication.Revision, &publication.CreatedAt)
	if err != nil {
		return ThemeRuntimePublication{}, fmt.Errorf("insert theme runtime publication: %w", err)
	}
	return publication, nil
}

// themeApprovalSourceOrAdmin 归一化落库值：未批准的发布不带来源语义，
// 统一写 'admin'（列默认值），只有 host 批准会写 'host'。
func themeApprovalSourceOrAdmin(source string) string {
	if source == ThemeApprovalSourceHost {
		return ThemeApprovalSourceHost
	}
	return ThemeApprovalSourceAdmin
}

func lockCurrentThemeTuple(ctx context.Context, tx pgx.Tx) (themePublicationTuple, bool, error) {
	var result themePublicationTuple
	err := tx.QueryRow(ctx, `
		SELECT e.id, v.id, v.version, v.package_digest
		FROM extensions e
		JOIN extension_versions v ON v.id = e.active_version_id
		WHERE e.type = 'theme' AND e.status = 'enabled'
		FOR UPDATE OF e
	`).Scan(&result.ID, &result.VersionID, &result.Version, &result.Digest)
	if errors.Is(err, pgx.ErrNoRows) {
		return themePublicationTuple{}, false, nil
	}
	return result, err == nil, err
}

func lockThemeTuple(ctx context.Context, tx pgx.Tx, id string) (themePublicationTuple, error) {
	var result themePublicationTuple
	err := tx.QueryRow(ctx, `
		SELECT e.id, v.id, v.version, v.package_digest
		FROM extensions e
		JOIN extension_versions v ON v.id = e.active_version_id
		WHERE e.id = $1 AND e.type = 'theme'
		FOR UPDATE OF e
	`, id).Scan(&result.ID, &result.VersionID, &result.Version, &result.Digest)
	if errors.Is(err, pgx.ErrNoRows) {
		return themePublicationTuple{}, ErrExtensionNotFound
	}
	return result, err
}

func lockExactThemeVersionTuple(
	ctx context.Context,
	tx pgx.Tx,
	id string,
	versionID int64,
) (themePublicationTuple, int64, error) {
	if versionID <= 0 {
		return themePublicationTuple{}, 0, ErrThemePublicationConflict
	}
	var result themePublicationTuple
	var activeVersionID int64
	err := tx.QueryRow(ctx, `
		SELECT e.id, COALESCE(e.active_version_id, 0), v.id, v.version, v.package_digest
		FROM extensions e
		JOIN extension_versions v ON v.id = $2 AND v.extension_id = e.id
		WHERE e.id = $1 AND e.type = 'theme'
		FOR UPDATE OF e
	`, id, versionID).Scan(
		&result.ID, &activeVersionID, &result.VersionID, &result.Version, &result.Digest,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return themePublicationTuple{}, 0, ErrThemePublicationConflict
	}
	return result, activeVersionID, err
}

func lockExactThemeActivationTarget(
	ctx context.Context,
	tx pgx.Tx,
	id string,
	version string,
	digest string,
) (themePublicationTuple, bool, error) {
	var result themePublicationTuple
	var activeVersionID, stagedVersionID int64
	err := tx.QueryRow(ctx, `
		SELECT e.id, COALESCE(e.active_version_id, 0), COALESCE(e.staged_version_id, 0),
		       v.id, v.version, v.package_digest
		FROM extensions e
		JOIN extension_versions v
		  ON v.extension_id = e.id
		 AND v.version = $2
		 AND v.package_digest = $3
		WHERE e.id = $1 AND e.type = 'theme'
		  AND v.id IN (e.active_version_id, e.staged_version_id)
		FOR UPDATE OF e
	`, id, version, digest).Scan(
		&result.ID, &activeVersionID, &stagedVersionID,
		&result.VersionID, &result.Version, &result.Digest,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return themePublicationTuple{}, false, ErrThemePreviewStale
	}
	if err != nil {
		return themePublicationTuple{}, false, err
	}
	return result, stagedVersionID != 0 && result.VersionID == stagedVersionID && result.VersionID != activeVersionID, nil
}

// lockThemeApproval 读取某扩展是否存在已批准的核心页替换绑定（任意制品）。
// 人工批准优先于 host 批准：只要有一条 admin 绑定，来源就记 admin。
func lockThemeApproval(ctx context.Context, tx pgx.Tx, extensionID string) (bool, int64, string, error) {
	if extensionID == "" {
		return false, 0, "", nil
	}
	return scanThemeApprovalRows(ctx, tx, `
		SELECT COALESCE(approved_by, 0), approval_source
		FROM page_provider_bindings
		WHERE extension_id = $1 AND (approved_by IS NOT NULL OR approval_source = 'host')
		FOR UPDATE
	`, extensionID)
}

// lockThemeExactArtifactApproval 只认匹配 version + package digest 的绑定：
// 新 digest 不会被旧制品的审批顺带放行。
func lockThemeExactArtifactApproval(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
	version string,
	digest string,
) (bool, int64, string, error) {
	if extensionID == "" {
		return false, 0, "", nil
	}
	return scanThemeApprovalRows(ctx, tx, `
		SELECT COALESCE(approved_by, 0), approval_source
		FROM page_provider_bindings
		WHERE extension_id = $1 AND version = $2 AND package_digest = $3
		  AND (approved_by IS NOT NULL OR approval_source = 'host')
		FOR UPDATE
	`, extensionID, version, digest)
}

func scanThemeApprovalRows(
	ctx context.Context,
	tx pgx.Tx,
	query string,
	args ...any,
) (bool, int64, string, error) {
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return false, 0, "", err
	}
	defer rows.Close()
	var actor int64
	hostSeen := false
	for rows.Next() {
		var current int64
		var source string
		if err := rows.Scan(&current, &source); err != nil {
			return false, 0, "", err
		}
		if current > 0 {
			if actor != 0 && actor != current {
				return false, 0, "", ErrThemePublicationConflict
			}
			actor = current
			continue
		}
		if source == ThemeApprovalSourceHost {
			hostSeen = true
		}
	}
	if err := rows.Err(); err != nil {
		return false, 0, "", err
	}
	if actor > 0 {
		return true, actor, ThemeApprovalSourceAdmin, nil
	}
	if hostSeen {
		return true, 0, ThemeApprovalSourceHost, nil
	}
	return false, 0, "", nil
}

func validThemeRuntimePublication(value ThemeRuntimePublication) bool {
	if !validThemePublicationReason(value.Reason) ||
		value.ActorUserID < 0 || value.SourceActorUserID < 0 {
		return false
	}
	sourceEmpty := value.SourceThemeID == "" && value.SourceThemeVersion == "" && value.SourcePackageDigest == ""
	sourceExact := value.SourceThemeID != "" && value.SourceThemeVersion != "" && validPackageDigest(value.SourcePackageDigest)
	if !sourceEmpty && !sourceExact {
		return false
	}
	if value.SourceCoreReplacementsApproved {
		if !sourceExact || !validThemeApprovalProvenance(value.SourceApprovalSource, value.SourceActorUserID) {
			return false
		}
	} else if value.SourceActorUserID != 0 {
		return false
	}
	if value.CoreReplacementsApproved {
		if !validThemeApprovalProvenance(value.ApprovalSource, value.ActorUserID) {
			return false
		}
		// host 批准只属于 Host 自带的受保护默认主题。
		if value.ApprovalSource == ThemeApprovalSourceHost && value.ThemeID != DefaultThemeID {
			return false
		}
	}
	switch value.DesiredState {
	case ThemeRuntimePublicationActive:
		return value.ThemeID != "" && value.ThemeVersion != "" && validPackageDigest(value.PackageDigest)
	case ThemeRuntimePublicationNone:
		return value.ThemeID == "" && value.ThemeVersion == "" && value.PackageDigest == "" && !value.CoreReplacementsApproved
	default:
		return false
	}
}

// validThemeApprovalProvenance 约束「已批准」与「谁批准」的对应关系：
// admin 必须有真实 actor，host 必须没有 actor。空来源等价 admin（与列默认值
// 一致），因此「忘记写 host」只会 fail-closed，不会悄悄变成 host 批准。
func validThemeApprovalProvenance(source string, actor int64) bool {
	if strings.TrimSpace(source) == "" {
		source = ThemeApprovalSourceAdmin
	}
	switch source {
	case ThemeApprovalSourceAdmin:
		return actor > 0
	case ThemeApprovalSourceHost:
		return actor == 0
	default:
		return false
	}
}

func validThemePublicationReason(value ThemeRuntimePublicationReason) bool {
	return value == ThemeRuntimePublicationActivation || value == ThemeRuntimePublicationCompensation ||
		value == ThemeRuntimePublicationStartupRepair
}

func validPackageDigest(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func sameThemeRuntimePublication(left, right ThemeRuntimePublication) bool {
	return left.Revision == right.Revision && left.DesiredState == right.DesiredState &&
		left.ThemeID == right.ThemeID && left.ThemeVersion == right.ThemeVersion && strings.EqualFold(left.PackageDigest, right.PackageDigest) &&
		left.SourceThemeID == right.SourceThemeID && left.SourceThemeVersion == right.SourceThemeVersion && strings.EqualFold(left.SourcePackageDigest, right.SourcePackageDigest) &&
		left.SourceCoreReplacementsApproved == right.SourceCoreReplacementsApproved && left.SourceActorUserID == right.SourceActorUserID &&
		themeApprovalSourceOrAdmin(left.SourceApprovalSource) == themeApprovalSourceOrAdmin(right.SourceApprovalSource) &&
		left.CoreReplacementsApproved == right.CoreReplacementsApproved && left.ActorUserID == right.ActorUserID &&
		themeApprovalSourceOrAdmin(left.ApprovalSource) == themeApprovalSourceOrAdmin(right.ApprovalSource) &&
		left.Reason == right.Reason
}

func nullableThemePublicationValue(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableThemePublicationActor(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}

var _ ThemeRuntimePublicationRepository = (*PostgresStore)(nil)

var (
	ErrThemePublicationNotFound = errors.New("extensions: theme runtime publication not found")
	ErrThemePublicationConflict = errors.New("extensions: theme runtime publication conflict")
)

type ThemeRuntimePublicationRepository interface {
	LatestThemeRuntimePublication(context.Context) (ThemeRuntimePublication, error)
	ThemeRuntimePublicationByRevision(context.Context, int64) (ThemeRuntimePublication, error)
}

// InitialThemeRuntimePublicationEnsurer imports the pre-publication active
// theme exactly once. Watchers use it before registering a boot lease so an
// empty desired-state ledger can never be mistaken for successful convergence.
type InitialThemeRuntimePublicationEnsurer interface {
	EnsureInitialThemeRuntimePublication(context.Context) (ThemeRuntimePublication, error)
}
