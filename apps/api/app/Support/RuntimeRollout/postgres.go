package runtimerollout

import (
	"context"
	"encoding/json"
	"errors"
	"hash/fnv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore is the production multi-node rollout authority.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore builds a durable plan store.
func NewPostgresStore(pool *pgxpool.Pool) (*PostgresStore, error) {
	if pool == nil {
		return nil, ErrInvalid
	}
	return &PostgresStore{pool: pool}, nil
}

// Create inserts a plan under advisory lock; concurrent creates for the same
// extension yield exactly one winner (unique partial index).
func (s *PostgresStore) Create(ctx context.Context, plan Plan) (Plan, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return Plan{}, ErrInvalid
	}
	plan = clonePlan(plan)
	if plan.PlanID == "" || plan.ExtensionID == "" {
		return Plan{}, ErrInvalid
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return Plan{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	lockKey := advisoryKey("rollout", plan.ExtensionID)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, lockKey); err != nil {
		return Plan{}, err
	}

	var existing string
	err = tx.QueryRow(ctx, `
		SELECT plan_id FROM runtime_rollout_plans
		WHERE extension_id = $1
		  AND phase NOT IN ('active', 'failed', 'rolled_back')
		LIMIT 1
	`, plan.ExtensionID).Scan(&existing)
	if err == nil {
		return Plan{}, ErrConflict
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, err
	}

	if err := insertPlan(ctx, tx, plan); err != nil {
		if isUniqueViolation(err) {
			return Plan{}, ErrConflict
		}
		return Plan{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Plan{}, err
	}
	return clonePlan(plan), nil
}

// Save updates a plan row.
func (s *PostgresStore) Save(ctx context.Context, plan Plan) (Plan, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return Plan{}, ErrInvalid
	}
	plan = clonePlan(plan)
	acks, err := json.Marshal(plan.NodeAcks)
	if err != nil {
		return Plan{}, ErrInvalid
	}
	retained, err := json.Marshal(plan.RetainedDigests)
	if err != nil {
		return Plan{}, ErrInvalid
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE runtime_rollout_plans SET
			migration_ready = $2,
			canary_percent = $3,
			phase = $4,
			snapshot_id = $5,
			retain_versions = $6,
			actor = $7,
			reason = $8,
			last_error = $9,
			retained_digests = $10::jsonb,
			node_acks = $11::jsonb,
			updated_at = $12
		WHERE plan_id = $1
	`, plan.PlanID, plan.MigrationReady, plan.CanaryPercent, plan.Phase,
		plan.SnapshotID, plan.RetainVersions, plan.Actor, plan.Reason, plan.LastError,
		string(retained), string(acks), plan.UpdatedAt)
	if err != nil {
		return Plan{}, err
	}
	if tag.RowsAffected() == 0 {
		return Plan{}, ErrNotFound
	}
	return clonePlan(plan), nil
}

// Get loads one plan.
func (s *PostgresStore) Get(ctx context.Context, planID string) (Plan, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return Plan{}, ErrInvalid
	}
	plan, err := scanPlan(ctx, s.pool, `
		SELECT plan_id, schema_version, extension_id, source_digest, target_digest,
			migration_ready, canary_percent, phase, snapshot_id, retain_versions,
			actor, reason, last_error, retained_digests, node_acks, updated_at
		FROM runtime_rollout_plans WHERE plan_id = $1
	`, strings.TrimSpace(planID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, ErrNotFound
	}
	return plan, err
}

// List returns plans optionally filtered by extension.
func (s *PostgresStore) List(ctx context.Context, extensionID string) ([]Plan, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return nil, ErrInvalid
	}
	extensionID = strings.ToLower(strings.TrimSpace(extensionID))
	var rows pgx.Rows
	var err error
	if extensionID == "" {
		rows, err = s.pool.Query(ctx, `
			SELECT plan_id, schema_version, extension_id, source_digest, target_digest,
				migration_ready, canary_percent, phase, snapshot_id, retain_versions,
				actor, reason, last_error, retained_digests, node_acks, updated_at
			FROM runtime_rollout_plans ORDER BY plan_id
		`)
	} else {
		rows, err = s.pool.Query(ctx, `
			SELECT plan_id, schema_version, extension_id, source_digest, target_digest,
				migration_ready, canary_percent, phase, snapshot_id, retain_versions,
				actor, reason, last_error, retained_digests, node_acks, updated_at
			FROM runtime_rollout_plans WHERE extension_id = $1 ORDER BY plan_id
		`, extensionID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Plan, 0)
	for rows.Next() {
		plan, err := scanPlanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, plan)
	}
	return out, rows.Err()
}

// ActiveForExtension returns the non-terminal plan if any.
func (s *PostgresStore) ActiveForExtension(ctx context.Context, extensionID string) (Plan, bool, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return Plan{}, false, ErrInvalid
	}
	plan, err := scanPlan(ctx, s.pool, `
		SELECT plan_id, schema_version, extension_id, source_digest, target_digest,
			migration_ready, canary_percent, phase, snapshot_id, retain_versions,
			actor, reason, last_error, retained_digests, node_acks, updated_at
		FROM runtime_rollout_plans
		WHERE extension_id = $1
		  AND phase NOT IN ('active', 'failed', 'rolled_back')
		ORDER BY updated_at DESC
		LIMIT 1
	`, strings.ToLower(strings.TrimSpace(extensionID)))
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, false, nil
	}
	if err != nil {
		return Plan{}, false, err
	}
	return plan, true, nil
}

type querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func scanPlan(ctx context.Context, q querier, sql string, args ...any) (Plan, error) {
	row := q.QueryRow(ctx, sql, args...)
	return scanPlanFromRow(row)
}

func scanPlanRow(rows pgx.Rows) (Plan, error) {
	return scanPlanFromRow(rows)
}

type scannable interface {
	Scan(dest ...any) error
}

func scanPlanFromRow(row scannable) (Plan, error) {
	var plan Plan
	var retainedRaw, acksRaw []byte
	var updatedAt time.Time
	err := row.Scan(
		&plan.PlanID, &plan.SchemaVersion, &plan.ExtensionID, &plan.SourceDigest, &plan.TargetDigest,
		&plan.MigrationReady, &plan.CanaryPercent, &plan.Phase, &plan.SnapshotID, &plan.RetainVersions,
		&plan.Actor, &plan.Reason, &plan.LastError, &retainedRaw, &acksRaw, &updatedAt,
	)
	if err != nil {
		return Plan{}, err
	}
	plan.UpdatedAt = updatedAt
	if len(retainedRaw) > 0 {
		_ = json.Unmarshal(retainedRaw, &plan.RetainedDigests)
	}
	if len(acksRaw) > 0 {
		_ = json.Unmarshal(acksRaw, &plan.NodeAcks)
	}
	if plan.NodeAcks == nil {
		plan.NodeAcks = map[string]NodeAck{}
	}
	return clonePlan(plan), nil
}

func insertPlan(ctx context.Context, tx pgx.Tx, plan Plan) error {
	acks, err := json.Marshal(plan.NodeAcks)
	if err != nil {
		return ErrInvalid
	}
	if plan.NodeAcks == nil {
		acks = []byte("{}")
	}
	retained, err := json.Marshal(plan.RetainedDigests)
	if err != nil {
		return ErrInvalid
	}
	if plan.RetainedDigests == nil {
		retained = []byte("[]")
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO runtime_rollout_plans (
			plan_id, schema_version, extension_id, source_digest, target_digest,
			migration_ready, canary_percent, phase, snapshot_id, retain_versions,
			actor, reason, last_error, retained_digests, node_acks, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::jsonb,$15::jsonb,$16
		)
	`, plan.PlanID, plan.SchemaVersion, plan.ExtensionID, plan.SourceDigest, plan.TargetDigest,
		plan.MigrationReady, plan.CanaryPercent, plan.Phase, plan.SnapshotID, plan.RetainVersions,
		plan.Actor, plan.Reason, plan.LastError, string(retained), string(acks), plan.UpdatedAt)
	return err
}

func advisoryKey(parts ...string) int64 {
	h := fnv.New64a()
	for _, p := range parts {
		_, _ = h.Write([]byte(p))
		_, _ = h.Write([]byte{0})
	}
	// pg advisory locks use int64; keep positive.
	return int64(h.Sum64() & 0x7fffffffffffffff)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Compile-time check.
var (
	_ PlanStore = (*PostgresStore)(nil)
	_ PlanStore = (*MemoryStore)(nil)
)
