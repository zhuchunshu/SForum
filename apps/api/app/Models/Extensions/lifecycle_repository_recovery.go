package extensions

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresLifecycleRepository) RecoveryDecision(
	ctx context.Context,
	operationID int64,
	operationAttempt int,
) (LifecycleRecoveryDecision, error) {
	if r == nil || r.pool == nil || ctx == nil || operationID <= 0 || operationAttempt <= 1 {
		return LifecycleRecoveryDecision{}, ErrLifecycleInvalidInput
	}
	decision, err := scanLifecycleRecoveryDecision(r.pool.QueryRow(ctx, lifecycleRecoveryDecisionSelectSQL()+`
		WHERE operation_id = $1 AND operation_attempt = $2
	`, operationID, operationAttempt))
	if errors.Is(err, pgx.ErrNoRows) {
		return LifecycleRecoveryDecision{}, ErrLifecycleRecoveryNotFound
	}
	if err != nil {
		return LifecycleRecoveryDecision{}, fmt.Errorf("load lifecycle recovery decision: %w", err)
	}
	return decision, nil
}

func (r *PostgresLifecycleRepository) ListRecoveryDecisions(
	ctx context.Context,
	operationID int64,
) ([]LifecycleRecoveryDecision, error) {
	if r == nil || r.pool == nil || ctx == nil || operationID <= 0 {
		return nil, ErrLifecycleInvalidInput
	}
	rows, err := r.pool.Query(ctx, lifecycleRecoveryDecisionSelectSQL()+`
		WHERE operation_id = $1
		ORDER BY operation_attempt ASC, id ASC
	`, operationID)
	if err != nil {
		return nil, fmt.Errorf("list lifecycle recovery decisions: %w", err)
	}
	defer rows.Close()
	decisions := make([]LifecycleRecoveryDecision, 0)
	for rows.Next() {
		decision, scanErr := scanLifecycleRecoveryDecision(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan lifecycle recovery decision: %w", scanErr)
		}
		decisions = append(decisions, decision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate lifecycle recovery decisions: %w", err)
	}
	return decisions, nil
}

func scanLifecycleRecoveryDecision(scanner lifecycleScanner) (LifecycleRecoveryDecision, error) {
	var decision LifecycleRecoveryDecision
	err := scanner.Scan(
		&decision.ID, &decision.OperationID, &decision.OperationAttempt,
		&decision.Decision, &decision.EscalateForced, &decision.Reason,
		&decision.ActorUserID, &decision.AuditEventID, &decision.CreatedAt,
	)
	return decision, err
}

func lifecycleRecoveryDecisionSelectSQL() string {
	return `
		SELECT id, operation_id, operation_attempt, decision, escalate_forced,
		       reason, actor_user_id, audit_event_id, created_at
		FROM extension_lifecycle_recovery_decisions
	`
}

// LastSuccessfulLifecycleAuthority 与下面的 recovery 读取同属「生命周期决策读面」，
// 因此放在同一文件：两者都只对 PostgresLifecycleRepository 做只读查询，且共用
// 同一套「没有行就是 Err*NotFound」的语义。拆成两个文件只是历史上先后落地的顺序，
// 不是职责边界。
//
// LastSuccessfulLifecycleAuthority returns an immutable exact-artifact grant
// snapshot. It intentionally ignores current trust-row revocation state so an
// operator can still disable or rollback code that was previously approved.
func (r *PostgresLifecycleRepository) LastSuccessfulLifecycleAuthority(
	ctx context.Context,
	input ExactExtensionVersionInput,
) (LifecycleAuthoritySnapshot, error) {
	if r == nil || r.pool == nil {
		return LifecycleAuthoritySnapshot{}, ErrLifecycleCoordinatorUnavailable
	}
	if err := validateExactExtensionVersionInput(input); err != nil {
		return LifecycleAuthoritySnapshot{}, err
	}
	var document []byte
	err := r.pool.QueryRow(ctx, `
		SELECT authority_snapshot
		FROM extension_lifecycle_operations
		WHERE extension_id = $1
		  AND extension_version = $2
		  AND package_digest = $3
		  AND terminal_result = $4
		  AND completed_at IS NOT NULL
		ORDER BY completed_at DESC, id DESC
		LIMIT 1
	`, input.ExtensionID, input.Version, input.PackageDigest, LifecycleTerminalSucceeded).Scan(&document)
	if errors.Is(err, pgx.ErrNoRows) {
		return LifecycleAuthoritySnapshot{}, ErrLifecycleAuthorityNotFound
	}
	if err != nil {
		return LifecycleAuthoritySnapshot{}, fmt.Errorf("load successful lifecycle authority: %w", err)
	}
	return decodeLifecycleAuthoritySnapshot(document)
}

var _ LifecycleAuthorityRepository = (*PostgresLifecycleRepository)(nil)
