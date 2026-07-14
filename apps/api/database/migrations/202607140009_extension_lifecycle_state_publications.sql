-- +goose Up
-- V3 P4：extension row 的精确发布/补偿事实独立持久化。source/target 快照不引用
-- 可删除的 extensions/version 行；operation 与 publication marker 是保留根。
CREATE TABLE extension_lifecycle_state_publications (
  id BIGSERIAL PRIMARY KEY,
  operation_id BIGINT NOT NULL,
  operation TEXT NOT NULL
    CHECK (operation IN ('install', 'enable', 'disable', 'upgrade', 'rollback', 'uninstall')),
  step_id TEXT NOT NULL CHECK (octet_length(step_id) BETWEEN 1 AND 512),
  position INTEGER NOT NULL CHECK (position >= 0),
  publication_mode TEXT NOT NULL
    CHECK (publication_mode IN ('activate', 'deactivate')),
  extension_id TEXT NOT NULL CHECK (extension_id <> ''),

  source_status TEXT NOT NULL CHECK (source_status IN ('installed', 'enabled', 'disabled')),
  source_active_version_id BIGINT NOT NULL CHECK (source_active_version_id > 0),
  source_active_version TEXT NOT NULL CHECK (source_active_version <> ''),
  source_active_package_digest TEXT NOT NULL
    CHECK (source_active_package_digest ~ '^[0-9a-f]{64}$'),
  source_staged_version_id BIGINT,
  source_staged_version TEXT,
  source_staged_package_digest TEXT,

  target_status TEXT NOT NULL CHECK (target_status IN ('installed', 'enabled', 'disabled')),
  target_active_version_id BIGINT NOT NULL CHECK (target_active_version_id > 0),
  target_active_version TEXT NOT NULL CHECK (target_active_version <> ''),
  target_active_package_digest TEXT NOT NULL
    CHECK (target_active_package_digest ~ '^[0-9a-f]{64}$'),
  target_staged_version_id BIGINT,
  target_staged_version TEXT,
  target_staged_package_digest TEXT,

  transaction_state TEXT NOT NULL DEFAULT 'source'
    CHECK (transaction_state IN ('source', 'target')),
  first_attempt INTEGER NOT NULL CHECK (first_attempt > 0),
  last_attempt INTEGER NOT NULL CHECK (last_attempt >= first_attempt),
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  prepared_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  published_at TIMESTAMPTZ,
  restored_at TIMESTAMPTZ,

  UNIQUE (operation_id, step_id, publication_mode),
  FOREIGN KEY (operation_id, step_id, publication_mode)
    REFERENCES extension_lifecycle_publications(operation_id, step_id, publication_mode)
    ON DELETE CASCADE,
  CHECK (
    (source_staged_version_id IS NULL
      AND source_staged_version IS NULL
      AND source_staged_package_digest IS NULL)
    OR
    (source_staged_version_id > 0
      AND source_staged_version IS NOT NULL AND source_staged_version <> ''
      AND source_staged_package_digest ~ '^[0-9a-f]{64}$')
  ),
  CHECK (
    (target_staged_version_id IS NULL
      AND target_staged_version IS NULL
      AND target_staged_package_digest IS NULL)
    OR
    (target_staged_version_id > 0
      AND target_staged_version IS NOT NULL AND target_staged_version <> ''
      AND target_staged_package_digest ~ '^[0-9a-f]{64}$')
  ),
  CHECK (source_staged_version_id IS NULL OR source_staged_version_id <> source_active_version_id),
  CHECK (target_staged_version_id IS NULL OR target_staged_version_id <> target_active_version_id),
  CHECK ((publication_mode = 'activate' AND target_status = 'enabled')
      OR (publication_mode = 'deactivate' AND target_status = 'disabled')),
  CHECK (updated_at >= prepared_at),
  CHECK (published_at IS NULL OR published_at >= prepared_at),
  CHECK (restored_at IS NULL OR restored_at >= prepared_at)
);

CREATE INDEX extension_lifecycle_state_publications_state_idx
  ON extension_lifecycle_state_publications (transaction_state, updated_at, operation_id, id);

-- +goose Down
-- source snapshot 是补偿依据；存在任何记录时禁止迁移回滚丢失恢复事实。
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_lifecycle_state_publications) THEN
    RAISE EXCEPTION 'cannot remove lifecycle state publication history';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS extension_lifecycle_state_publications_state_idx;
DROP TABLE IF EXISTS extension_lifecycle_state_publications;
