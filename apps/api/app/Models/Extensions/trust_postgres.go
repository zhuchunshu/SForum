package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresExecutableTrustStore struct {
	pool *pgxpool.Pool
}

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

func (s *PostgresExecutableTrustStore) ConsumeChallenge(ctx context.Context, input TrustConsumeInput) (TrustGrant, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return TrustGrant{}, fmt.Errorf("begin consume trust challenge: %w", err)
	}
	defer tx.Rollback(ctx)

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
	if _, err := tx.Exec(ctx, `
		UPDATE extension_trust_challenges SET consumed_at = now() WHERE id = $1
	`, challengeID); err != nil {
		return TrustGrant{}, fmt.Errorf("consume trust challenge: %w", err)
	}

	var grant TrustGrant
	err = tx.QueryRow(ctx, `
		INSERT INTO extension_trust_grants (
			extension_id, extension_version, package_digest, action,
			artifact_digests, impact_document, impact_digest, granted_by_user_id
		) VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, $8)
		ON CONFLICT (extension_id, extension_version, package_digest, action, impact_digest)
		  WHERE revoked_at IS NULL
		DO UPDATE SET granted_by_user_id = EXCLUDED.granted_by_user_id, granted_at = now()
		RETURNING id, extension_id, extension_version, package_digest, action,
		          impact_digest, granted_by_user_id, granted_at, revoked_at,
		          COALESCE(revoked_by_user_id, 0), revocation_reason
	`, extensionID, version, packageDigest, action, artifactJSON, impactJSON,
		impactDigest, actorUserID).Scan(
		&grant.ID, &grant.ExtensionID, &grant.ExtensionVersion, &grant.PackageDigest,
		&grant.Action, &grant.ImpactDigest, &grant.GrantedByUserID, &grant.GrantedAt,
		&grant.RevokedAt, &grant.RevokedByUserID, &grant.RevocationReason,
	)
	if err != nil {
		return TrustGrant{}, fmt.Errorf("create executable trust grant: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return TrustGrant{}, fmt.Errorf("commit executable trust grant: %w", err)
	}
	return grant, nil
}

func (s *PostgresExecutableTrustStore) RevokeAll(ctx context.Context, extensionID string, actorUserID int64, reason string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin revoke executable trust: %w", err)
	}
	defer tx.Rollback(ctx)
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
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit revoke executable trust: %w", err)
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
