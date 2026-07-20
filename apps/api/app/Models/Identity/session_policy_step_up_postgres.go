package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresSessionPolicyStepUpStore persists Host-owned one-use step_up evidence.
type PostgresSessionPolicyStepUpStore struct {
	pool *pgxpool.Pool
}

func NewPostgresSessionPolicyStepUpStore(pool *pgxpool.Pool) (*PostgresSessionPolicyStepUpStore, error) {
	if pool == nil {
		return nil, ErrSessionPolicyStepUpStore
	}
	return &PostgresSessionPolicyStepUpStore{pool: pool}, nil
}

func (s *PostgresSessionPolicyStepUpStore) Issue(
	ctx context.Context,
	claim SessionPolicyStepUpClaim,
	expiresAt time.Time,
	tokenHash string,
) error {
	if s == nil || s.pool == nil {
		return ErrSessionPolicyStepUpStore
	}
	if ctx == nil {
		return ErrSessionPolicyStepUpInvalid
	}
	claim, err := normalizeSessionPolicyStepUpClaim(claim)
	if err != nil {
		return err
	}
	tokenHash = strings.ToLower(strings.TrimSpace(tokenHash))
	if !isHex64(tokenHash) {
		return ErrSessionPolicyStepUpInvalid
	}
	if !expiresAt.After(time.Now().UTC()) {
		return ErrSessionPolicyStepUpInvalid
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO identity_session_policy_step_up_evidence (
		  token_hash, user_id, token_version, purpose, policy_id,
		  selection_revision, registry_revision, registry_digest,
		  package_digest, owner_extension_id, correlation_id, device_fingerprint,
		  expires_at
		) VALUES (
		  $1, $2, $3, $4, $5,
		  $6, $7, $8,
		  $9, $10, $11, $12,
		  $13
		)
	`, tokenHash, claim.UserID, claim.TokenVersion, claim.Purpose, claim.PolicyID,
		claim.SelectionRevision, int64(claim.RegistryRevision), claim.RegistryDigest,
		claim.PackageDigest, claim.OwnerExtensionID, claim.CorrelationID, claim.DeviceFingerprint,
		expiresAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("%w: insert step-up evidence: %v", ErrSessionPolicyStepUpStore, err)
	}
	return nil
}

func (s *PostgresSessionPolicyStepUpStore) ConsumeForEffect(
	ctx context.Context,
	tokenHash string,
	expected SessionPolicyStepUpClaim,
) error {
	if s == nil || s.pool == nil {
		return ErrSessionPolicyStepUpStore
	}
	if ctx == nil {
		return ErrSessionPolicyStepUpInvalid
	}
	expected, err := normalizeSessionPolicyStepUpClaim(expected)
	if err != nil {
		return err
	}
	tokenHash = strings.ToLower(strings.TrimSpace(tokenHash))
	if !isHex64(tokenHash) {
		return ErrSessionPolicyStepUpInvalid
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: begin step-up consume: %v", ErrSessionPolicyStepUpStore, err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	var (
		userID            int64
		tokenVersion      int64
		purpose           string
		policyID          string
		selectionRevision int64
		registryRevision  int64
		registryDigest    string
		packageDigest     string
		ownerExtensionID  string
		correlationID     string
		deviceFingerprint string
		expiresAt         time.Time
		consumedAt        *time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT user_id, token_version, purpose, policy_id, selection_revision,
		       registry_revision, registry_digest, package_digest, owner_extension_id,
		       correlation_id, device_fingerprint, expires_at, consumed_at
		FROM identity_session_policy_step_up_evidence
		WHERE token_hash = $1
		FOR UPDATE
	`, tokenHash).Scan(
		&userID, &tokenVersion, &purpose, &policyID, &selectionRevision,
		&registryRevision, &registryDigest, &packageDigest, &ownerExtensionID,
		&correlationID, &deviceFingerprint, &expiresAt, &consumedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrSessionPolicyStepUpInvalid
	}
	if err != nil {
		return fmt.Errorf("%w: lock step-up evidence: %v", ErrSessionPolicyStepUpStore, err)
	}
	if consumedAt != nil {
		return ErrSessionPolicyStepUpReplayed
	}
	if !time.Now().UTC().Before(expiresAt.UTC()) {
		return ErrSessionPolicyStepUpExpired
	}
	stored := SessionPolicyStepUpClaim{
		UserID: userID, TokenVersion: tokenVersion, Purpose: purpose, PolicyID: policyID,
		SelectionRevision: selectionRevision, RegistryRevision: uint64(registryRevision),
		RegistryDigest: registryDigest, PackageDigest: packageDigest,
		OwnerExtensionID: ownerExtensionID, CorrelationID: correlationID,
		DeviceFingerprint: deviceFingerprint,
	}
	if !sameSessionPolicyStepUpClaim(stored, expected) {
		return ErrSessionPolicyStepUpStale
	}
	tag, err := tx.Exec(ctx, `
		UPDATE identity_session_policy_step_up_evidence
		SET consumed_at = statement_timestamp()
		WHERE token_hash = $1 AND consumed_at IS NULL
	`, tokenHash)
	if err != nil {
		return fmt.Errorf("%w: consume step-up evidence: %v", ErrSessionPolicyStepUpStore, err)
	}
	if tag.RowsAffected() != 1 {
		return ErrSessionPolicyStepUpReplayed
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: commit step-up consume: %v", ErrSessionPolicyStepUpStore, err)
	}
	return nil
}
