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

const (
	externalIdentityLinkTransactionAttempts = 3
	externalIdentityLinkReadbackTimeout     = 2 * time.Second
	externalIdentityLinkLockNamespace       = "sforum:identity-external-link:idempotency:"
)

var errExternalIdentityLinkRetry = errors.New("identity: external link transaction should retry")
var errExternalIdentityLinkCommitNotVerified = errors.New("identity: external link commit was not verified")

type PostgresExternalIdentityLinkStore struct {
	pool *pgxpool.Pool
}

func NewPostgresExternalIdentityLinkStore(pool *pgxpool.Pool) *PostgresExternalIdentityLinkStore {
	return &PostgresExternalIdentityLinkStore{pool: pool}
}

func (s *PostgresExternalIdentityLinkStore) Link(
	ctx context.Context,
	input LinkExternalIdentityInput,
	fence ExternalIdentityLinkCommitFence,
) (ExternalIdentityLinkMutation, error) {
	if s == nil || s.pool == nil || ctx == nil || fence == nil {
		return ExternalIdentityLinkMutation{}, ErrExternalIdentityLinkInvalid
	}
	prepared, err := prepareExternalIdentityLink(input)
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	// 注册必须与用户创建共用 caller-owned 事务，不能从独立入口提交半套状态。
	// 已认证账号可由显式 link.complete 或未绑定 login.complete continuation 独立提交。
	if prepared.providerOperation != "link.complete" && prepared.providerOperation != "login.complete" {
		return ExternalIdentityLinkMutation{}, ErrExternalIdentityLinkInvalid
	}
	return s.mutate(
		ctx, prepared.idempotencyKey, ExternalIdentityLinkActionLink, prepared.fingerprint, fence,
		func(tx pgx.Tx) (ExternalIdentityLinkMutation, error) {
			return linkExternalIdentityTx(ctx, tx, prepared)
		},
	)
}

// LinkTx composes registration user creation and its external link in one
// caller-owned transaction. Existing-account link flows may also compose other
// Host effects here. The caller must run its exact runtime fence immediately
// before committing that transaction.
func (s *PostgresExternalIdentityLinkStore) LinkTx(
	ctx context.Context,
	tx pgx.Tx,
	input LinkExternalIdentityInput,
) (ExternalIdentityLinkMutation, error) {
	if s == nil || s.pool == nil || ctx == nil || tx == nil {
		return ExternalIdentityLinkMutation{}, ErrExternalIdentityLinkInvalid
	}
	prepared, err := prepareExternalIdentityLink(input)
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	return linkExternalIdentityTx(ctx, tx, prepared)
}

func (s *PostgresExternalIdentityLinkStore) Unlink(
	ctx context.Context,
	input TransitionExternalIdentityLinkInput,
) (ExternalIdentityLinkMutation, error) {
	return s.transition(ctx, ExternalIdentityLinkActionUnlink, input)
}

func (s *PostgresExternalIdentityLinkStore) Erase(
	ctx context.Context,
	input TransitionExternalIdentityLinkInput,
) (ExternalIdentityLinkMutation, error) {
	return s.transition(ctx, ExternalIdentityLinkActionErase, input)
}

func (s *PostgresExternalIdentityLinkStore) transition(
	ctx context.Context,
	action string,
	input TransitionExternalIdentityLinkInput,
) (ExternalIdentityLinkMutation, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return ExternalIdentityLinkMutation{}, ErrExternalIdentityLinkInvalid
	}
	prepared, err := prepareExternalIdentityTransition(action, input)
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	return s.mutate(
		ctx, prepared.idempotencyKey, prepared.action, prepared.fingerprint, nil,
		func(tx pgx.Tx) (ExternalIdentityLinkMutation, error) {
			return transitionExternalIdentityTx(ctx, tx, prepared)
		},
	)
}

// TransitionTx lets privacy/account workflows compose unlink or erase with
// other Host-owned effects. The caller owns transaction retry and commit.
func (s *PostgresExternalIdentityLinkStore) TransitionTx(
	ctx context.Context,
	tx pgx.Tx,
	action string,
	input TransitionExternalIdentityLinkInput,
) (ExternalIdentityLinkMutation, error) {
	if s == nil || s.pool == nil || ctx == nil || tx == nil {
		return ExternalIdentityLinkMutation{}, ErrExternalIdentityLinkInvalid
	}
	prepared, err := prepareExternalIdentityTransition(action, input)
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	return transitionExternalIdentityTx(ctx, tx, prepared)
}

func (s *PostgresExternalIdentityLinkStore) Get(ctx context.Context, id int64) (ExternalIdentityLink, error) {
	if s == nil || s.pool == nil || ctx == nil || id <= 0 {
		return ExternalIdentityLink{}, ErrExternalIdentityLinkInvalid
	}
	link, err := scanExternalIdentityLink(s.pool.QueryRow(ctx, externalIdentityLinkSelect+` WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return ExternalIdentityLink{}, ErrExternalIdentityLinkNotFound
	}
	if err != nil {
		return ExternalIdentityLink{}, mapExternalIdentityLinkStoreError(err)
	}
	return link, nil
}

func (s *PostgresExternalIdentityLinkStore) FindActive(
	ctx context.Context,
	providerID string,
	providerSubjectDigest string,
) (ExternalIdentityLink, error) {
	providerID = strings.ToLower(strings.TrimSpace(providerID))
	providerSubjectDigest = strings.ToLower(strings.TrimSpace(providerSubjectDigest))
	if s == nil || s.pool == nil || ctx == nil || providerID == "" ||
		!validExternalIdentityDigest(providerSubjectDigest) {
		return ExternalIdentityLink{}, ErrExternalIdentityLinkInvalid
	}
	link, err := scanExternalIdentityLink(s.pool.QueryRow(ctx, externalIdentityLinkSelect+`
		WHERE provider_id = $1 AND provider_subject_digest = $2 AND status = 'active'
	`, providerID, providerSubjectDigest))
	if errors.Is(err, pgx.ErrNoRows) {
		return ExternalIdentityLink{}, ErrExternalIdentityLinkNotFound
	}
	if err != nil {
		return ExternalIdentityLink{}, mapExternalIdentityLinkStoreError(err)
	}
	return link, nil
}

func (s *PostgresExternalIdentityLinkStore) ListUser(
	ctx context.Context,
	userID int64,
) ([]ExternalIdentityLink, error) {
	if s == nil || s.pool == nil || ctx == nil || userID <= 0 {
		return nil, ErrExternalIdentityLinkInvalid
	}
	rows, err := s.pool.Query(ctx, externalIdentityLinkSelect+`
		WHERE user_id = $1
		ORDER BY status ASC, provider_id ASC, id ASC
	`, userID)
	if err != nil {
		return nil, mapExternalIdentityLinkStoreError(err)
	}
	defer rows.Close()
	result := make([]ExternalIdentityLink, 0)
	for rows.Next() {
		link, scanErr := scanExternalIdentityLink(rows)
		if scanErr != nil {
			return nil, mapExternalIdentityLinkStoreError(scanErr)
		}
		result = append(result, link)
	}
	if err := rows.Err(); err != nil {
		return nil, mapExternalIdentityLinkStoreError(err)
	}
	return result, nil
}

func (s *PostgresExternalIdentityLinkStore) mutate(
	ctx context.Context,
	idempotencyKey string,
	action string,
	fingerprint string,
	fence ExternalIdentityLinkCommitFence,
	operation func(pgx.Tx) (ExternalIdentityLinkMutation, error),
) (ExternalIdentityLinkMutation, error) {
	for attempt := 0; attempt < externalIdentityLinkTransactionAttempts; attempt++ {
		tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if err != nil {
			return ExternalIdentityLinkMutation{}, mapExternalIdentityLinkStoreError(err)
		}
		result, operationErr := operation(tx)
		if operationErr != nil {
			_ = tx.Rollback(context.Background())
			mapped := mapExternalIdentityLinkStoreError(operationErr)
			if errors.Is(mapped, errExternalIdentityLinkRetry) && attempt+1 < externalIdentityLinkTransactionAttempts && ctx.Err() == nil {
				continue
			}
			return ExternalIdentityLinkMutation{}, publicExternalIdentityLinkStoreError(mapped)
		}

		fenceConsumed := false
		if fence != nil {
			fenceConsumed = true
			if err := fence(); err != nil {
				_ = tx.Rollback(context.Background())
				return ExternalIdentityLinkMutation{}, fmt.Errorf("finalize external identity link admission: %w", err)
			}
		}
		if err := tx.Commit(ctx); err == nil {
			return result, nil
		} else if externalIdentityLinkCommitDefinitelyFailed(err) {
			mapped := mapExternalIdentityLinkStoreError(err)
			if !fenceConsumed && errors.Is(mapped, errExternalIdentityLinkRetry) &&
				attempt+1 < externalIdentityLinkTransactionAttempts && ctx.Err() == nil {
				continue
			}
			return ExternalIdentityLinkMutation{}, publicExternalIdentityLinkStoreError(mapped)
		} else {
			verified, found, verificationErr := s.readbackMutation(
				ctx, idempotencyKey, action, fingerprint,
			)
			if verificationErr == nil && found {
				verified.Replayed = true
				return verified, nil
			}
			if verificationErr == nil {
				verificationErr = errExternalIdentityLinkCommitNotVerified
			}
			return ExternalIdentityLinkMutation{}, &ExternalIdentityLinkCommitUnknownError{
				CommitError:       fmt.Errorf("commit external identity link: %w", err),
				VerificationError: verificationErr,
			}
		}
	}
	return ExternalIdentityLinkMutation{}, ErrExternalIdentityLinkStateConflict
}

func linkExternalIdentityTx(
	ctx context.Context,
	tx pgx.Tx,
	input preparedExternalIdentityLink,
) (ExternalIdentityLinkMutation, error) {
	if err := lockExternalIdentityIdempotencyKey(ctx, tx, input.idempotencyKey); err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	if replay, found, err := replayExternalIdentityMutation(
		ctx, tx, input.idempotencyKey, ExternalIdentityLinkActionLink, input.fingerprint, true,
	); err != nil || found {
		return replay, err
	}
	// 已提交效果的幂等回放不依赖 provider 仍处于 active；新效果才需要
	// 锁住精确声明。这样 disable/uninstall 后保留的 inert link 仍可确定读回。
	tip, err := lockExactExternalIdentityProvider(ctx, tx, input.provider)
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	if err := lockActiveExternalIdentityUser(ctx, tx, input.userID); err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	var existingID int64
	err = tx.QueryRow(ctx, `
		SELECT id
		FROM identity_external_links
		WHERE provider_id = $1 AND provider_subject_digest = $2 AND status = 'active'
		FOR UPDATE
	`, input.provider.ID, input.providerSubjectDigest).Scan(&existingID)
	if err == nil {
		return ExternalIdentityLinkMutation{}, ErrExternalIdentitySubjectConflict
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ExternalIdentityLinkMutation{}, err
	}

	var linkID int64
	if err := tx.QueryRow(ctx, `
		SELECT nextval(pg_get_serial_sequence('identity_external_links', 'id'))
	`).Scan(&linkID); err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	auditID, err := insertExternalIdentityLinkAudit(ctx, tx, externalIdentityLinkAuditInput{
		Action: ExternalIdentityLinkActionLink, LinkID: linkID, UserID: input.userID,
		ProviderID: input.provider.ID, ProviderContractVersion: input.provider.ContractVersion,
		ProviderOperation:       input.providerOperation,
		OwnerExtensionID:        input.provider.Artifact.ExtensionID,
		OwnerExtensionVersionID: input.provider.Artifact.VersionID,
		OwnerExtensionVersion:   input.provider.Artifact.ExtensionVersion,
		OwnerPackageDigest:      input.provider.Artifact.PackageDigest,
		DeclarationRevision:     tip.revision, DeclarationDigest: tip.declarationDigest,
		PreviousRevision: 0, NextRevision: 1,
		PreviousStatus: "", NextStatus: ExternalIdentityLinkStatusActive,
		ActorUserID: input.actorUserID,
	})
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	link, err := scanExternalIdentityLink(tx.QueryRow(ctx, `
		INSERT INTO identity_external_links (
			id, user_id, provider_id, provider_contract_version,
			owner_extension_id, owner_extension_version_id, owner_extension_version,
			owner_package_digest, declaration_revision, provider_subject_digest,
			status, revision, actor_user_id, audit_event_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			'active', 1, $11, $12
		)
			RETURNING `+externalIdentityLinkColumns, linkID, input.userID, input.provider.ID, input.provider.ContractVersion,
		input.provider.Artifact.ExtensionID, input.provider.Artifact.VersionID,
		input.provider.Artifact.ExtensionVersion, input.provider.Artifact.PackageDigest,
		tip.revision, input.providerSubjectDigest, nullableExternalIdentityActor(input.actorUserID), auditID))
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	event, err := insertExternalIdentityLinkEvent(
		ctx, tx, link, ExternalIdentityLinkActionLink, input.idempotencyKey, input.fingerprint,
		0, "", input.actorUserID, auditID,
	)
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	return ExternalIdentityLinkMutation{Link: link, Event: event}, nil
}

func transitionExternalIdentityTx(
	ctx context.Context,
	tx pgx.Tx,
	input preparedExternalIdentityTransition,
) (ExternalIdentityLinkMutation, error) {
	if err := lockExternalIdentityIdempotencyKey(ctx, tx, input.idempotencyKey); err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	if replay, found, err := replayExternalIdentityMutation(
		ctx, tx, input.idempotencyKey, input.action, input.fingerprint, true,
	); err != nil || found {
		return replay, err
	}
	current, err := scanExternalIdentityLink(tx.QueryRow(ctx, externalIdentityLinkSelect+`
		WHERE id = $1 FOR UPDATE
	`, input.linkID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ExternalIdentityLinkMutation{}, ErrExternalIdentityLinkNotFound
	}
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	if current.Revision != input.expectedRevision || current.Status == ExternalIdentityLinkStatusErased ||
		(input.action == ExternalIdentityLinkActionUnlink && current.Status != ExternalIdentityLinkStatusActive) {
		return ExternalIdentityLinkMutation{}, ErrExternalIdentityLinkStateConflict
	}
	nextStatus := ExternalIdentityLinkStatusUnlinked
	if input.action == ExternalIdentityLinkActionErase {
		nextStatus = ExternalIdentityLinkStatusErased
	}
	auditID, err := insertExternalIdentityLinkAudit(ctx, tx, externalIdentityLinkAuditInput{
		Action: input.action, LinkID: current.ID, UserID: current.UserID,
		ProviderID: current.ProviderID, ProviderContractVersion: current.ProviderContractVersion,
		OwnerExtensionID:        current.OwnerExtensionID,
		OwnerExtensionVersionID: current.OwnerExtensionVersionID,
		OwnerExtensionVersion:   current.OwnerExtensionVersion,
		OwnerPackageDigest:      current.OwnerPackageDigest,
		DeclarationRevision:     current.DeclarationRevision,
		PreviousRevision:        current.Revision, NextRevision: current.Revision + 1,
		PreviousStatus: current.Status, NextStatus: nextStatus, ActorUserID: input.actorUserID,
	})
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	query := `
		UPDATE identity_external_links
		SET status = $2,
		    revision = revision + 1,
		    unlinked_at = CASE WHEN $2 = 'unlinked' THEN transaction_timestamp() ELSE unlinked_at END,
		    erased_at = CASE WHEN $2 = 'erased' THEN transaction_timestamp() ELSE erased_at END,
		    provider_subject_digest = CASE WHEN $2 = 'erased' THEN NULL ELSE provider_subject_digest END,
		    actor_user_id = $3,
		    audit_event_id = $4,
		    updated_at = transaction_timestamp()
		WHERE id = $1 AND revision = $5`
	if input.action == ExternalIdentityLinkActionUnlink {
		query += ` AND status = 'active'`
	} else {
		query += ` AND status IN ('active', 'unlinked')`
	}
	query += ` RETURNING ` + externalIdentityLinkColumns
	link, err := scanExternalIdentityLink(tx.QueryRow(
		ctx, query, current.ID, nextStatus, nullableExternalIdentityActor(input.actorUserID),
		auditID, input.expectedRevision,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return ExternalIdentityLinkMutation{}, ErrExternalIdentityLinkStateConflict
	}
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	event, err := insertExternalIdentityLinkEvent(
		ctx, tx, link, input.action, input.idempotencyKey, input.fingerprint,
		current.Revision, current.Status, input.actorUserID, auditID,
	)
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	return ExternalIdentityLinkMutation{Link: link, Event: event}, nil
}
