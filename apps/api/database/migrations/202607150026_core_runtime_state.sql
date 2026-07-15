-- +goose Up
-- 高风险数据库租约只能依据已完整完成 Core 与 River 迁移的语义版本签发。
-- 首次创建保持 migrating；migrator 在全部迁移成功后原子发布 ready。
CREATE TABLE sforum_core_runtime_state (
  singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
  current_version TEXT NOT NULL DEFAULT '',
  target_version TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'migrating'
    CHECK (status IN ('migrating', 'ready')),
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  migration_started_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  migrated_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  CHECK (updated_at >= created_at),
  CHECK (
    (status = 'migrating' AND migrated_at IS NULL)
    OR (
      status = 'ready'
      AND current_version <> ''
      AND target_version = current_version
      AND migrated_at IS NOT NULL
    )
  )
);

INSERT INTO sforum_core_runtime_state (singleton)
VALUES (TRUE);

-- +goose Down
-- 已发布的语义版本是滚动节点和高风险租约的安全边界，不能静默删除。
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM sforum_core_runtime_state
    WHERE status = 'ready' OR current_version <> '' OR revision > 1
  ) THEN
    RAISE EXCEPTION 'cannot remove published Core runtime version state';
  END IF;
END $$;
-- +goose StatementEnd

DROP TABLE sforum_core_runtime_state;
