-- +goose Up
-- V3 P4：跨 family Registry 发布以 lifecycle publication marker 为唯一决策点。
-- 本表只保存 exact artifact、不可变计划摘要和稳定 source/target 相位；运行时
-- provider 闭包不会持久化，重启后由 immutable artifact + exact process 重建。
CREATE TABLE extension_lifecycle_registry_publications (
  id BIGSERIAL PRIMARY KEY,
  publication_id BIGINT NOT NULL UNIQUE
    REFERENCES extension_lifecycle_publications(id) ON DELETE CASCADE,
  operation_id BIGINT NOT NULL
    REFERENCES extension_lifecycle_operations(id) ON DELETE CASCADE,
  operation TEXT NOT NULL
    CHECK (operation IN ('install', 'enable', 'disable', 'upgrade', 'rollback', 'uninstall')),
  step_id TEXT NOT NULL CHECK (octet_length(step_id) BETWEEN 1 AND 512),
  position INTEGER NOT NULL CHECK (position >= 0),
  publication_mode TEXT NOT NULL
    CHECK (publication_mode IN ('activate', 'deactivate')),

  source_extension_id TEXT,
  source_extension_version TEXT,
  source_package_digest TEXT,
  source_version_id BIGINT,
  source_runtime_instance_id TEXT,
  source_plan_digest TEXT NOT NULL CHECK (source_plan_digest ~ '^[0-9a-f]{64}$'),

  target_extension_id TEXT NOT NULL CHECK (target_extension_id <> ''),
  target_extension_version TEXT NOT NULL CHECK (target_extension_version <> ''),
  target_package_digest TEXT NOT NULL CHECK (target_package_digest ~ '^[0-9a-f]{64}$'),
  target_version_id BIGINT NOT NULL CHECK (target_version_id > 0),
  target_runtime_instance_id TEXT NOT NULL DEFAULT ''
    CHECK (target_runtime_instance_id = btrim(target_runtime_instance_id)
      AND octet_length(target_runtime_instance_id) <= 512),
  target_plan_digest TEXT NOT NULL CHECK (target_plan_digest ~ '^[0-9a-f]{64}$'),

  first_attempt INTEGER NOT NULL CHECK (first_attempt > 0),
  last_attempt INTEGER NOT NULL CHECK (last_attempt >= first_attempt),
  transaction_state TEXT NOT NULL DEFAULT 'source'
    CHECK (transaction_state IN ('source', 'target')),
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  prepared_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  published_at TIMESTAMPTZ,
  restored_at TIMESTAMPTZ,

  UNIQUE (operation_id, step_id, publication_mode),
  CHECK (
    (source_extension_id IS NULL
      AND source_extension_version IS NULL
      AND source_package_digest IS NULL
      AND source_version_id IS NULL
      AND source_runtime_instance_id IS NULL)
    OR
    (source_extension_id IS NOT NULL AND source_extension_id <> ''
      AND source_extension_version IS NOT NULL AND source_extension_version <> ''
      AND source_package_digest ~ '^[0-9a-f]{64}$'
      AND source_version_id > 0
      AND source_runtime_instance_id IS NOT NULL
      AND source_runtime_instance_id = btrim(source_runtime_instance_id)
      AND octet_length(source_runtime_instance_id) <= 512)
  ),
  CHECK (source_extension_id IS NULL OR source_extension_id = target_extension_id),
  CHECK ((publication_mode = 'activate' AND target_runtime_instance_id <> '')
      OR publication_mode = 'deactivate'),
  CHECK (updated_at >= prepared_at),
  CHECK (transaction_state <> 'target' OR published_at IS NOT NULL),
  CHECK (transaction_state <> 'source' OR published_at IS NULL OR restored_at IS NOT NULL),
  CHECK (published_at IS NULL OR published_at >= prepared_at),
  CHECK (restored_at IS NULL OR restored_at >= prepared_at)
);

CREATE INDEX extension_lifecycle_registry_publications_operation_idx
  ON extension_lifecycle_registry_publications (operation_id, position, id);

-- +goose Down
-- Registry intent is recovery evidence. Once written, an automatic downgrade
-- may not erase which exact artifact should be reconstructed.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_lifecycle_registry_publications) THEN
    RAISE EXCEPTION 'cannot remove lifecycle registry publication history';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS extension_lifecycle_registry_publications_operation_idx;
DROP TABLE IF EXISTS extension_lifecycle_registry_publications;
