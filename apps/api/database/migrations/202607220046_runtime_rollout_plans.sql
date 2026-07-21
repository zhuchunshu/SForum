-- +goose Up
-- V3 P12: durable multi-node runtime rollout plans. Process-local maps cannot
-- coordinate API nodes; PostgreSQL is authoritative. Plans recover after
-- API/worker restart. Concurrent Create uses advisory lock + unique active plan.

CREATE TABLE IF NOT EXISTS runtime_rollout_plans (
    plan_id           TEXT PRIMARY KEY,
    schema_version    TEXT NOT NULL DEFAULT 'sforum.runtime-rollout@1',
    extension_id      TEXT NOT NULL CHECK (extension_id <> ''),
    source_digest     TEXT NOT NULL CHECK (source_digest ~ '^[0-9a-f]{64}$'),
    target_digest     TEXT NOT NULL CHECK (target_digest ~ '^[0-9a-f]{64}$'),
    migration_ready   BOOLEAN NOT NULL DEFAULT FALSE,
    canary_percent    INTEGER NOT NULL DEFAULT 10
        CHECK (canary_percent >= 1 AND canary_percent <= 100),
    phase             TEXT NOT NULL
        CHECK (phase IN (
            'pending', 'migrating', 'staged', 'canary', 'draining',
            'promoting', 'active', 'rolling_back', 'failed', 'rolled_back'
        )),
    snapshot_id       TEXT NOT NULL DEFAULT '',
    retain_versions   INTEGER NOT NULL DEFAULT 3 CHECK (retain_versions >= 1),
    actor             TEXT NOT NULL DEFAULT '',
    reason            TEXT NOT NULL DEFAULT '',
    last_error        TEXT NOT NULL DEFAULT '',
    retained_digests  JSONB NOT NULL DEFAULT '[]'::jsonb,
    node_acks         JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    CONSTRAINT runtime_rollout_source_ne_target CHECK (source_digest <> target_digest)
);

-- At most one non-terminal plan per extension (winner of multi-API race).
CREATE UNIQUE INDEX IF NOT EXISTS runtime_rollout_plans_active_extension_uidx
    ON runtime_rollout_plans (extension_id)
    WHERE phase NOT IN ('active', 'failed', 'rolled_back');

CREATE INDEX IF NOT EXISTS runtime_rollout_plans_extension_updated_idx
    ON runtime_rollout_plans (extension_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS runtime_rollout_plans_phase_idx
    ON runtime_rollout_plans (phase, updated_at DESC);

-- System tier membership (operator-managed early infra providers).
CREATE TABLE IF NOT EXISTS system_tier_members (
    extension_id   TEXT PRIMARY KEY CHECK (extension_id <> ''),
    role           TEXT NOT NULL
        CHECK (role IN ('auth', 'cache', 'storage', 'infra')),
    priority       INTEGER NOT NULL DEFAULT 100,
    enabled        BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    updated_by     TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS system_tier_members_enabled_priority_idx
    ON system_tier_members (enabled, priority, extension_id);

-- Privacy export/erase audit (no PII payload bodies).
CREATE TABLE IF NOT EXISTS privacy_operation_audit (
    id             BIGSERIAL PRIMARY KEY,
    audit_id       TEXT NOT NULL UNIQUE,
    operation      TEXT NOT NULL CHECK (operation IN ('export', 'erase')),
    actor          TEXT NOT NULL,
    user_id        TEXT NOT NULL,
    status         TEXT NOT NULL CHECK (status IN ('ok', 'partial', 'failed')),
    detail         JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp()
);

CREATE INDEX IF NOT EXISTS privacy_operation_audit_user_idx
    ON privacy_operation_audit (user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS privacy_operation_audit;
DROP TABLE IF EXISTS system_tier_members;
DROP INDEX IF EXISTS runtime_rollout_plans_phase_idx;
DROP INDEX IF EXISTS runtime_rollout_plans_extension_updated_idx;
DROP INDEX IF EXISTS runtime_rollout_plans_active_extension_uidx;
DROP TABLE IF EXISTS runtime_rollout_plans;
