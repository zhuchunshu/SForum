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
