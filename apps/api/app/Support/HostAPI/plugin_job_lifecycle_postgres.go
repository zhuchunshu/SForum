package hostapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// PluginJobLifecycleRiverClient 是 lifecycle adapter 使用的 River 公共事务面。
// 保持这个接口很小，避免 Host 依赖 River driver 的私有 SQL 或 schema 细节。
type PluginJobLifecycleRiverClient interface {
	InsertTx(context.Context, pgx.Tx, river.JobArgs, *river.InsertOpts) (*rivertype.JobInsertResult, error)
	JobCancelTx(context.Context, pgx.Tx, int64) (*rivertype.JobRow, error)
}

type PostgresPluginJobLifecycleStore struct {
	pool  *pgxpool.Pool
	river PluginJobLifecycleRiverClient
}

func NewPostgresPluginJobLifecycleStore(
	pool *pgxpool.Pool,
	riverClient PluginJobLifecycleRiverClient,
) *PostgresPluginJobLifecycleStore {
	return &PostgresPluginJobLifecycleStore{pool: pool, river: riverClient}
}

func (s *PostgresPluginJobLifecycleStore) WithPluginJobLifecycleTx(
	ctx context.Context,
	fn func(PluginJobLifecycleTx) error,
) error {
	if s == nil || s.pool == nil || s.river == nil {
		return ErrPluginJobLifecycleUnavailable
	}
	if fn == nil {
		return fmt.Errorf("%w: lifecycle transaction callback is required", ErrInvalidRequest)
	}
	err := pgx.BeginTxFunc(ctx, s.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return fn(&postgresPluginJobLifecycleTx{tx: tx, river: s.river})
	})
	if err != nil {
		return fmt.Errorf("plugin job lifecycle transaction: %w", err)
	}
	return nil
}

type postgresPluginJobLifecycleTx struct {
	tx    pgx.Tx
	river PluginJobLifecycleRiverClient
}

var (
	_ PluginJobLifecycleStore = (*PostgresPluginJobLifecycleStore)(nil)
	_ PluginJobLifecycleTx    = (*postgresPluginJobLifecycleTx)(nil)
)

func (t *postgresPluginJobLifecycleTx) LockPluginJobs(
	ctx context.Context,
	extensionID string,
) ([]PluginJobLifecycleRow, error) {
	// 不分页也不排除终态：同一事务必须看到这个 extension 的完整 River 快照，
	// 由纯 planner 统一判断 execute/drain/migrate/cancel/ignore。
	rows, err := t.tx.Query(ctx, `
		SELECT id, kind, state::text, args, attempt, max_attempts,
		       queue, priority, scheduled_at, COALESCE(tags, '{}'::varchar[])
		FROM river_job
		WHERE kind = 'extension.plugin_job'
		  AND args ->> 'extensionId' = $1
		ORDER BY id
		FOR UPDATE
	`, extensionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]PluginJobLifecycleRow, 0)
	for rows.Next() {
		var row PluginJobLifecycleRow
		var state string
		if err := rows.Scan(
			&row.JobID, &row.Kind, &state, &row.EncodedArgs,
			&row.Attempt, &row.MaxAttempts, &row.Queue, &row.Priority,
			&row.ScheduledAt, &row.Tags,
		); err != nil {
			return nil, err
		}
		row.State = rivertype.JobState(state)
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (t *postgresPluginJobLifecycleTx) ClaimPluginJobMigration(
	ctx context.Context,
	entry PluginJobMigrationLedgerEntry,
) (PluginJobMigrationClaim, error) {
	sourceContract, targetContract, err := marshalPluginJobLedgerContracts(entry)
	if err != nil {
		return PluginJobMigrationClaim{}, err
	}
	tag, err := t.tx.Exec(ctx, `
		INSERT INTO extension_plugin_job_migrations (
			old_job_id, extension_id, migration_id, source_contract,
			source_trust_grant_id, target_contract, target_trust_grant_id
		) VALUES ($1, $2, $3, $4::jsonb, $5, $6::jsonb, $7)
		ON CONFLICT (old_job_id) DO NOTHING
	`, entry.OldJobID, entry.ExtensionID, entry.MigrationID, sourceContract,
		entry.SourceTrustGrantID, targetContract, entry.TargetTrustGrantID)
	if err != nil {
		return PluginJobMigrationClaim{}, err
	}
	if tag.RowsAffected() > 1 {
		return PluginJobMigrationClaim{}, fmt.Errorf("%w: claim affected more than one row", ErrPluginJobMigrationConflict)
	}

	claim := PluginJobMigrationClaim{Claimed: tag.RowsAffected() == 1}
	var persistedSource, persistedTarget []byte
	var newJobID *int64
	err = t.tx.QueryRow(ctx, `
		SELECT old_job_id, extension_id, migration_id, source_contract,
		       source_trust_grant_id, target_contract, target_trust_grant_id,
		       new_job_id
		FROM extension_plugin_job_migrations
		WHERE old_job_id = $1
		FOR UPDATE
	`, entry.OldJobID).Scan(
		&claim.Ledger.OldJobID, &claim.Ledger.ExtensionID, &claim.Ledger.MigrationID,
		&persistedSource, &claim.Ledger.SourceTrustGrantID,
		&persistedTarget, &claim.Ledger.TargetTrustGrantID, &newJobID,
	)
	if err != nil {
		return PluginJobMigrationClaim{}, err
	}
	if err := json.Unmarshal(persistedSource, &claim.Ledger.SourceContract); err != nil {
		return PluginJobMigrationClaim{}, fmt.Errorf("decode source plugin job contract: %w", err)
	}
	if err := json.Unmarshal(persistedTarget, &claim.Ledger.TargetContract); err != nil {
		return PluginJobMigrationClaim{}, fmt.Errorf("decode target plugin job contract: %w", err)
	}
	if newJobID != nil {
		claim.NewJobID = *newJobID
	}
	return claim, nil
}

func (t *postgresPluginJobLifecycleTx) InsertPluginJob(
	ctx context.Context,
	args PluginJobArgs,
	opts *river.InsertOpts,
) (int64, error) {
	result, err := t.river.InsertTx(ctx, t.tx, args, opts)
	if err != nil {
		return 0, err
	}
	if result == nil || result.Job == nil || result.Job.ID <= 0 {
		return 0, fmt.Errorf("%w: River returned no inserted plugin job", ErrInvalidRequest)
	}
	return result.Job.ID, nil
}

func (t *postgresPluginJobLifecycleTx) CompletePluginJobMigration(
	ctx context.Context,
	entry PluginJobMigrationLedgerEntry,
	newJobID int64,
) error {
	sourceContract, targetContract, err := marshalPluginJobLedgerContracts(entry)
	if err != nil {
		return err
	}
	tag, err := t.tx.Exec(ctx, `
		UPDATE extension_plugin_job_migrations
		SET new_job_id = $8, completed_at = transaction_timestamp()
		WHERE old_job_id = $1
		  AND extension_id = $2
		  AND migration_id = $3
		  AND source_contract = $4::jsonb
		  AND source_trust_grant_id = $5
		  AND target_contract = $6::jsonb
		  AND target_trust_grant_id = $7
		  AND new_job_id IS NULL
		  AND completed_at IS NULL
	`, entry.OldJobID, entry.ExtensionID, entry.MigrationID, sourceContract,
		entry.SourceTrustGrantID, targetContract, entry.TargetTrustGrantID, newJobID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%w: old job %d ledger identity changed before link", ErrPluginJobMigrationConflict, entry.OldJobID)
	}
	return nil
}

func (t *postgresPluginJobLifecycleTx) CancelPluginJob(ctx context.Context, jobID int64) error {
	_, err := t.river.JobCancelTx(ctx, t.tx, jobID)
	return err
}

func marshalPluginJobLedgerContracts(entry PluginJobMigrationLedgerEntry) (string, string, error) {
	source, err := json.Marshal(entry.SourceContract)
	if err != nil {
		return "", "", fmt.Errorf("encode source plugin job contract: %w", err)
	}
	target, err := json.Marshal(entry.TargetContract)
	if err != nil {
		return "", "", fmt.Errorf("encode target plugin job contract: %w", err)
	}
	return string(source), string(target), nil
}
