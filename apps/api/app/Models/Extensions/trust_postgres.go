package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresExecutableTrustStore struct {
	pool            *pgxpool.Pool
	commitRevokeAll func(context.Context, pgx.Tx) error
}

const (
	executableTrustExtensionLockNamespace = "sforum:executable-trust:"
	executableTrustCommitReadbackTimeout  = 5 * time.Second
)

var errTrustRevocationCommitNotVerified = errors.New("extensions: executable trust revocation commit was not verified")

func NewPostgresExecutableTrustStore(pool *pgxpool.Pool) *PostgresExecutableTrustStore {
	return &PostgresExecutableTrustStore{pool: pool}
}

func (s *PostgresExecutableTrustStore) CreateChallenge(ctx context.Context, input TrustChallengeRecord) error {
	artifacts, err := json.Marshal(input.ArtifactDigests)
	if err != nil {
		return fmt.Errorf("marshal trust artifact digests: %w", err)
	}
	impact, err := json.Marshal(input.Impact)
	if err != nil {
		return fmt.Errorf("marshal trust impact: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin trust challenge: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := LockExecutableTrustExtensionTx(ctx, tx, input.Identity.ExtensionID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE extension_trust_challenges
		SET invalidated_at = now(), invalidation_reason = 'superseded'
		WHERE actor_user_id = $1 AND extension_id = $2 AND action = $3
		  AND consumed_at IS NULL AND invalidated_at IS NULL
	`, input.ActorUserID, input.Identity.ExtensionID, input.Identity.Action); err != nil {
		return fmt.Errorf("invalidate prior trust challenges: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO extension_trust_challenges (
			token_hash, actor_user_id, extension_id, extension_version,
			package_digest, action, artifact_digests, impact_document,
			impact_digest, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9, $10)
	`, input.TokenHash, input.ActorUserID, input.Identity.ExtensionID,
		input.Identity.ExtensionVersion, input.Identity.PackageDigest, input.Identity.Action,
		string(artifacts), string(impact), input.Identity.ImpactDigest, input.ExpiresAt); err != nil {
		return fmt.Errorf("insert trust challenge: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit trust challenge: %w", err)
	}
	return nil
}

func (s *PostgresExecutableTrustStore) HasLiveGrant(ctx context.Context, identity TrustIdentity) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM extension_trust_grants
			WHERE extension_id = $1 AND extension_version = $2
			  AND package_digest = $3 AND action = $4 AND impact_digest = $5
			  AND revoked_at IS NULL
		)
	`, identity.ExtensionID, identity.ExtensionVersion, identity.PackageDigest,
		identity.Action, identity.ImpactDigest).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check executable trust grant: %w", err)
	}
	return exists, nil
}

func (s *PostgresExecutableTrustStore) LiveGrant(ctx context.Context, identity TrustIdentity) (TrustGrant, error) {
	var grant TrustGrant
	err := s.pool.QueryRow(ctx, `
		SELECT id, extension_id, extension_version, package_digest, action,
		       impact_digest, granted_by_user_id, granted_at, revoked_at,
		       COALESCE(revoked_by_user_id, 0), revocation_reason
		FROM extension_trust_grants
		WHERE extension_id = $1 AND extension_version = $2
		  AND package_digest = $3 AND action = $4 AND impact_digest = $5
		  AND revoked_at IS NULL
		ORDER BY granted_at DESC, id DESC
		LIMIT 1
	`, identity.ExtensionID, identity.ExtensionVersion, identity.PackageDigest,
		identity.Action, identity.ImpactDigest).Scan(
		&grant.ID, &grant.ExtensionID, &grant.ExtensionVersion, &grant.PackageDigest,
		&grant.Action, &grant.ImpactDigest, &grant.GrantedByUserID, &grant.GrantedAt,
		&grant.RevokedAt, &grant.RevokedByUserID, &grant.RevocationReason,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return TrustGrant{}, ErrTrustGrantNotFound
	}
	if err != nil {
		return TrustGrant{}, fmt.Errorf("load executable trust grant: %w", err)
	}
	return grant, nil
}

func (s *PostgresExecutableTrustStore) ConsumeChallenge(ctx context.Context, input TrustConsumeInput) (TrustGrant, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return TrustGrant{}, fmt.Errorf("begin consume trust challenge: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := LockExecutableTrustExtensionTx(ctx, tx, input.Identity.ExtensionID); err != nil {
		return TrustGrant{}, err
	}

	var (
		challengeID                         int64
		actorUserID                         int64
		extensionID, version, packageDigest string
		action, impactDigest                string
		artifactJSON, impactJSON            []byte
		expiresAt                           time.Time
		consumedAt, invalidatedAt           *time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT id, actor_user_id, extension_id, extension_version, package_digest,
		       action, artifact_digests, impact_document, impact_digest, expires_at,
		       consumed_at, invalidated_at
		FROM extension_trust_challenges
		WHERE token_hash = $1
		FOR UPDATE
	`, input.TokenHash).Scan(
		&challengeID, &actorUserID, &extensionID, &version, &packageDigest,
		&action, &artifactJSON, &impactJSON, &impactDigest, &expiresAt,
		&consumedAt, &invalidatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return TrustGrant{}, ErrTrustChallengeInvalid
	}
	if err != nil {
		return TrustGrant{}, fmt.Errorf("load trust challenge: %w", err)
	}
	if actorUserID != input.ActorUserID {
		return TrustGrant{}, ErrTrustChallengeInvalid
	}
	if consumedAt != nil {
		return TrustGrant{}, ErrTrustChallengeReplayed
	}
	if invalidatedAt != nil {
		return TrustGrant{}, ErrTrustChallengeStale
	}
	if !time.Now().UTC().Before(expiresAt) {
		return TrustGrant{}, invalidateTrustChallenge(ctx, tx, challengeID, "expired", ErrTrustChallengeExpired)
	}
	if extensionID != input.Identity.ExtensionID || version != input.Identity.ExtensionVersion ||
		packageDigest != input.Identity.PackageDigest || action != input.Identity.Action ||
		impactDigest != input.Identity.ImpactDigest {
		return TrustGrant{}, invalidateTrustChallenge(ctx, tx, challengeID, "artifact_changed", ErrTrustChallengeStale)
	}
	// 同一 exact identity 的挑战消费必须串行，否则两个节点都可能把同一
	// live grant 误判为“本次创建”，失败补偿就会撤销另一个成功请求的授权。
	lockIdentity := fmt.Sprintf(
		"%d:%s%d:%s%d:%s%d:%s%d:%s",
		len(extensionID), extensionID, len(version), version, len(packageDigest), packageDigest,
		len(action), action, len(impactDigest), impactDigest,
	)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockIdentity); err != nil {
		return TrustGrant{}, fmt.Errorf("lock executable trust identity: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE extension_trust_challenges SET consumed_at = now() WHERE id = $1
	`, challengeID); err != nil {
		return TrustGrant{}, fmt.Errorf("consume trust challenge: %w", err)
	}

	var grant TrustGrant
	err = tx.QueryRow(ctx, `
		SELECT id, extension_id, extension_version, package_digest, action,
		       impact_digest, COALESCE(granted_by_user_id, 0), granted_at, revoked_at,
		       COALESCE(revoked_by_user_id, 0), revocation_reason
		FROM extension_trust_grants
		WHERE extension_id = $1 AND extension_version = $2 AND package_digest = $3
		  AND action = $4 AND impact_digest = $5 AND revoked_at IS NULL
		FOR UPDATE
	`, extensionID, version, packageDigest, action, impactDigest).Scan(
		&grant.ID, &grant.ExtensionID, &grant.ExtensionVersion, &grant.PackageDigest,
		&grant.Action, &grant.ImpactDigest, &grant.GrantedByUserID, &grant.GrantedAt,
		&grant.RevokedAt, &grant.RevokedByUserID, &grant.RevocationReason,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			INSERT INTO extension_trust_grants (
				extension_id, extension_version, package_digest, action,
				artifact_digests, impact_document, impact_digest, granted_by_user_id
			) VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, $8)
			RETURNING id, extension_id, extension_version, package_digest, action,
			          impact_digest, COALESCE(granted_by_user_id, 0), granted_at, revoked_at,
			          COALESCE(revoked_by_user_id, 0), revocation_reason
		`, extensionID, version, packageDigest, action, artifactJSON, impactJSON,
			impactDigest, actorUserID).Scan(
			&grant.ID, &grant.ExtensionID, &grant.ExtensionVersion, &grant.PackageDigest,
			&grant.Action, &grant.ImpactDigest, &grant.GrantedByUserID, &grant.GrantedAt,
			&grant.RevokedAt, &grant.RevokedByUserID, &grant.RevocationReason,
		)
		grant.created = err == nil
	}
	if err != nil {
		return TrustGrant{}, fmt.Errorf("create executable trust grant: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return TrustGrant{}, fmt.Errorf("commit executable trust grant: %w", err)
	}
	return grant, nil
}

func (s *PostgresExecutableTrustStore) revokeExactGrant(
	ctx context.Context,
	grant TrustGrant,
	actorUserID int64,
	reason string,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin exact executable trust revocation: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := LockExecutableTrustExtensionTx(ctx, tx, grant.ExtensionID); err != nil {
		return err
	}
	command, err := tx.Exec(ctx, `
		UPDATE extension_trust_grants
		SET revoked_at = now(), revoked_by_user_id = $7, revocation_reason = $8
		WHERE id = $1 AND extension_id = $2 AND extension_version = $3
		  AND package_digest = $4 AND action = $5 AND impact_digest = $6
		  AND revoked_at IS NULL
	`, grant.ID, grant.ExtensionID, grant.ExtensionVersion, grant.PackageDigest,
		grant.Action, grant.ImpactDigest, nullableTrustActor(actorUserID), reason)
	if err != nil {
		return fmt.Errorf("revoke exact executable trust grant: %w", err)
	}
	if command.RowsAffected() == 1 {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit exact executable trust revocation: %w", err)
		}
		return nil
	}
	var revoked bool
	err = tx.QueryRow(ctx, `
		SELECT revoked_at IS NOT NULL
		FROM extension_trust_grants
		WHERE id = $1 AND extension_id = $2 AND extension_version = $3
		  AND package_digest = $4 AND action = $5 AND impact_digest = $6
	`, grant.ID, grant.ExtensionID, grant.ExtensionVersion, grant.PackageDigest,
		grant.Action, grant.ImpactDigest).Scan(&revoked)
	if err == nil && revoked {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) || err == nil {
		return ErrTrustGrantNotFound
	}
	return fmt.Errorf("verify exact executable trust revocation: %w", err)
}

func (s *PostgresExecutableTrustStore) RevokeAll(ctx context.Context, extensionID string, actorUserID int64, reason string) error {
	if s == nil || s.pool == nil || ctx == nil {
		return ErrTrustChallengeInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin revoke executable trust: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := LockExecutableTrustExtensionTx(ctx, tx, extensionID); err != nil {
		return err
	}
	var hasGrantHistory bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM extension_trust_grants WHERE extension_id = $1
		)
	`, extensionID).Scan(&hasGrantHistory); err != nil {
		return fmt.Errorf("inspect executable trust grant history: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE extension_trust_grants
		SET revoked_at = now(), revoked_by_user_id = $2, revocation_reason = $3
		WHERE extension_id = $1 AND revoked_at IS NULL
	`, extensionID, nullableTrustActor(actorUserID), reason); err != nil {
		return fmt.Errorf("revoke executable trust grants: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE extension_trust_challenges
		SET invalidated_at = now(), invalidation_reason = $2
		WHERE extension_id = $1 AND consumed_at IS NULL AND invalidated_at IS NULL
	`, extensionID, reason); err != nil {
		return fmt.Errorf("invalidate executable trust challenges: %w", err)
	}
	// 有任一 grant 历史后，即使 live 行已在上次未知结果中撤销，也必须重放
	// desired full-set removal；从未进入可执行信任域的 builtin 不能被误删。
	expectedRemovalRevision := int64(0)
	if hasGrantHistory {
		publication, published, err := PublishPluginRuntimeTrustRevocationTx(ctx, tx, extensionID, actorUserID)
		if err != nil {
			return fmt.Errorf("publish executable trust runtime revocation: %w", err)
		}
		if published {
			expectedRemovalRevision = publication.Revision
		}
	}
	commit := tx.Commit
	if s.commitRevokeAll != nil {
		commit = func(commitCtx context.Context) error {
			return s.commitRevokeAll(commitCtx, tx)
		}
	}
	if err := commit(ctx); err != nil {
		return s.resolveExecutableTrustRevocationCommit(
			ctx,
			executableTrustRevocationExpectation{
				extensionID: extensionID, hasGrantHistory: hasGrantHistory,
				expectedRemovalRevision: expectedRemovalRevision,
			},
			err,
		)
	}
	return nil
}

type executableTrustRevocationExpectation struct {
	extensionID             string
	hasGrantHistory         bool
	expectedRemovalRevision int64
}

func (s *PostgresExecutableTrustStore) resolveExecutableTrustRevocationCommit(
	ctx context.Context,
	expectation executableTrustRevocationExpectation,
	commitErr error,
) error {
	wrapped := fmt.Errorf("commit revoke executable trust: %w", commitErr)
	if executableTrustRevocationCommitDefinitelyFailed(commitErr) {
		return wrapped
	}
	verified, verificationErr := s.verifyExecutableTrustRevocationCommit(ctx, expectation)
	if verificationErr == nil && verified {
		return nil
	}
	if verificationErr != nil {
		return &TrustRevocationCommitUnknownError{
			commitErr: wrapped, verificationErr: verificationErr,
		}
	}
	verificationErr = errTrustRevocationCommitNotVerified
	return &TrustRevocationCommitUnknownError{
		commitErr: wrapped, verificationErr: verificationErr,
	}
}

func executableTrustRevocationCommitDefinitelyFailed(commitErr error) bool {
	if errors.Is(commitErr, pgx.ErrTxCommitRollback) || pgconn.SafeToRetry(commitErr) {
		return true
	}
	var postgresErr *pgconn.PgError
	if !errors.As(commitErr, &postgresErr) {
		return false
	}
	// Class 40 means PostgreSQL aborted the transaction, except 40003 whose
	// completion is explicitly unknown. Connection exception 08007 is likewise
	// ambiguous, and unrelated server errors do not prove that COMMIT rolled back.
	return strings.HasPrefix(postgresErr.Code, "40") && postgresErr.Code != "40003"
}

func (s *PostgresExecutableTrustStore) verifyExecutableTrustRevocationCommit(
	ctx context.Context,
	expectation executableTrustRevocationExpectation,
) (bool, error) {
	if s == nil || s.pool == nil || ctx == nil || expectation.extensionID == "" ||
		expectation.extensionID != strings.TrimSpace(expectation.extensionID) ||
		expectation.expectedRemovalRevision < 0 {
		return false, ErrTrustChallengeInvalid
	}
	readbackCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx), executableTrustCommitReadbackTimeout,
	)
	defer cancel()
	tx, err := s.pool.BeginTx(readbackCtx, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return false, fmt.Errorf("begin executable trust revocation commit readback: %w", err)
	}
	defer func() { _ = tx.Rollback(readbackCtx) }()

	var liveGrant, openChallenge bool
	if err := tx.QueryRow(readbackCtx, `
		SELECT
			EXISTS (
				SELECT 1 FROM extension_trust_grants
				WHERE extension_id = $1 AND revoked_at IS NULL
			),
			EXISTS (
				SELECT 1 FROM extension_trust_challenges
				WHERE extension_id = $1 AND consumed_at IS NULL AND invalidated_at IS NULL
			)
	`, expectation.extensionID).Scan(&liveGrant, &openChallenge); err != nil {
		return false, fmt.Errorf("read executable trust revocation commit state: %w", err)
	}
	if liveGrant || openChallenge {
		return false, nil
	}
	if !expectation.hasGrantHistory {
		return true, nil
	}
	latest, err := loadPluginRuntimePublication(
		readbackCtx,
		tx,
		pluginRuntimePublicationSelect+` ORDER BY revision DESC LIMIT 1`,
	)
	if errors.Is(err, ErrPluginRuntimePublicationNotFound) && expectation.expectedRemovalRevision == 0 {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("read executable trust runtime removal publication: %w", err)
	}
	if latest.Revision < expectation.expectedRemovalRevision {
		return false, nil
	}
	if _, found := pluginRuntimeMemberForExtension(latest.Members, expectation.extensionID); found {
		return false, nil
	}
	return true, nil
}

// LockExecutableTrustExtensionTx serializes grant/challenge changes with the
// final runtime-publication trust check. Callers must acquire this fence before
// the plugin desired-set lock.
func LockExecutableTrustExtensionTx(ctx context.Context, tx pgx.Tx, extensionID string) error {
	if ctx == nil || tx == nil || extensionID == "" || extensionID != strings.TrimSpace(extensionID) {
		return ErrTrustChallengeInvalid
	}
	key := executableTrustExtensionLockNamespace + fmt.Sprintf("%d:%s", len([]byte(extensionID)), extensionID)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, key); err != nil {
		return fmt.Errorf("lock executable trust extension: %w", err)
	}
	return nil
}

// RequireLiveExecutableTrustGrantTx proves that an already staged exact
// artifact still owns a live enable grant at the publication linearization
// point. The same transaction must publish the desired full-set afterwards.
func RequireLiveExecutableTrustGrantTx(
	ctx context.Context,
	tx pgx.Tx,
	grantID int64,
	extensionID string,
	extensionVersion string,
	packageDigest string,
) error {
	if grantID <= 0 || extensionVersion == "" || extensionVersion != strings.TrimSpace(extensionVersion) ||
		!validPackageDigest(packageDigest) {
		return ErrTrustGrantNotFound
	}
	if err := LockExecutableTrustExtensionTx(ctx, tx, extensionID); err != nil {
		return err
	}
	var live bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM extension_trust_grants
			WHERE id = $1 AND extension_id = $2 AND extension_version = $3
			  AND package_digest = $4 AND action = $5 AND revoked_at IS NULL
		)
	`, grantID, extensionID, extensionVersion, packageDigest, TrustActionEnable).Scan(&live); err != nil {
		return fmt.Errorf("validate live executable trust grant: %w", err)
	}
	if !live {
		return ErrTrustGrantNotFound
	}
	return nil
}

func invalidateTrustChallenge(ctx context.Context, tx pgx.Tx, id int64, reason string, result error) error {
	if _, err := tx.Exec(ctx, `
		UPDATE extension_trust_challenges
		SET invalidated_at = now(), invalidation_reason = $2
		WHERE id = $1
	`, id, reason); err != nil {
		return fmt.Errorf("invalidate trust challenge: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit invalid trust challenge: %w", err)
	}
	return result
}

func nullableTrustActor(actorUserID int64) any {
	if actorUserID <= 0 {
		return nil
	}
	return actorUserID
}
