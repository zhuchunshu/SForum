-- +goose Up
-- V3 P4：跨进程发布 marker 必须独立于可删除的 extensions 行。生命周期 operation
-- 是保留根；同一 stable Host step/mode 只允许一个 exact-artifact 发布事实。
CREATE TABLE extension_lifecycle_publications (
  id BIGSERIAL PRIMARY KEY,
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

  target_extension_id TEXT NOT NULL CHECK (target_extension_id <> ''),
  target_extension_version TEXT NOT NULL CHECK (target_extension_version <> ''),
  target_package_digest TEXT NOT NULL
    CHECK (target_package_digest ~ '^[0-9a-f]{64}$'),
  target_version_id BIGINT NOT NULL CHECK (target_version_id > 0),
  target_runtime_instance_id TEXT NOT NULL DEFAULT ''
    CHECK (target_runtime_instance_id = btrim(target_runtime_instance_id)
      AND octet_length(target_runtime_instance_id) <= 512),

  first_attempt INTEGER NOT NULL CHECK (first_attempt > 0),
  last_attempt INTEGER NOT NULL CHECK (last_attempt >= first_attempt),
  -- instance id 是进程级执行元数据；重启后同一 artifact/step attempt 可绑定新 id。
  runtime_attempts JSONB NOT NULL DEFAULT '[]'::jsonb
    CHECK (jsonb_typeof(runtime_attempts) = 'array'),
  committed_attempt INTEGER,
  commit_marker BOOLEAN NOT NULL DEFAULT FALSE,
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  prepared_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  committed_at TIMESTAMPTZ,

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
      OR (publication_mode = 'deactivate'
        AND source_runtime_instance_id IS NOT NULL
        AND source_runtime_instance_id <> '')),
  CHECK ((commit_marker = FALSE AND committed_attempt IS NULL AND committed_at IS NULL)
      OR (commit_marker = TRUE
        AND committed_attempt BETWEEN first_attempt AND last_attempt
        AND committed_at IS NOT NULL)),
  CHECK (updated_at >= prepared_at),
  CHECK (committed_at IS NULL OR committed_at >= prepared_at)
);

CREATE INDEX extension_lifecycle_publications_operation_idx
  ON extension_lifecycle_publications (operation_id, position, id);
CREATE INDEX extension_lifecycle_publications_uncommitted_idx
  ON extension_lifecycle_publications (updated_at, operation_id, id)
  WHERE commit_marker = FALSE;

-- +goose Down
-- publication marker 决定崩溃恢复向前还是向后收敛；有任何证据时禁止降级丢失它。
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_lifecycle_publications) THEN
    RAISE EXCEPTION 'cannot remove lifecycle publication history';
  END IF;
END $$;

DROP INDEX IF EXISTS extension_lifecycle_publications_uncommitted_idx;
DROP INDEX IF EXISTS extension_lifecycle_publications_operation_idx;
DROP TABLE IF EXISTS extension_lifecycle_publications;
