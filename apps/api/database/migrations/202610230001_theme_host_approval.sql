-- +goose Up
-- +sforum OnlineSafe
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '1min';

-- 审批来源。'admin' = super_admin 显式批准（approved_by 必须是真实用户）；
-- 'host' = Host 对自带受保护默认主题的首次/修复性批准（approved_by 为 NULL）。
-- 历史行统一取默认 'admin'：本表是 append-only（202607150020 的触发器禁止
-- UPDATE/DELETE），无法回填；未批准行的该值无语义，由 CHECK 与 Go 校验忽略。
ALTER TABLE page_provider_bindings
  ADD COLUMN approval_source TEXT NOT NULL DEFAULT 'admin'
    CHECK (approval_source IN ('admin', 'host'));

ALTER TABLE theme_runtime_publications
  ADD COLUMN approval_source TEXT NOT NULL DEFAULT 'admin'
    CHECK (approval_source IN ('admin', 'host')),
  ADD COLUMN source_approval_source TEXT NOT NULL DEFAULT 'admin'
    CHECK (source_approval_source IN ('admin', 'host'));

-- 原「已批准 ⇒ 必须有 actor」的 CHECK 是匿名声明的，名字由 PostgreSQL 生成。
-- 先按名字删掉 source 侧约束（它的定义同样含 core_replacements_approved /
-- actor_user_id IS NOT NULL，会干扰按定义匹配），再按定义定位并替换目标约束。
ALTER TABLE theme_runtime_publications
  DROP CONSTRAINT IF EXISTS theme_runtime_publications_source_approval_check;

-- +goose StatementBegin
DO $$
DECLARE
  target_constraint TEXT;
BEGIN
  SELECT conname INTO target_constraint
  FROM pg_constraint
  WHERE conrelid = 'theme_runtime_publications'::regclass
    AND contype = 'c'
    AND conname NOT LIKE '%source%'
    AND pg_get_constraintdef(oid) LIKE '%core_replacements_approved%'
    AND pg_get_constraintdef(oid) LIKE '%actor_user_id IS NOT NULL%';
  IF target_constraint IS NULL THEN
    RAISE EXCEPTION 'theme_runtime_publications approved=>actor constraint not found';
  END IF;
  EXECUTE format('ALTER TABLE theme_runtime_publications DROP CONSTRAINT %I', target_constraint);
END $$;
-- +goose StatementEnd

ALTER TABLE theme_runtime_publications
  ADD CONSTRAINT theme_runtime_publications_approval_check
    CHECK (
      core_replacements_approved = FALSE
      OR (approval_source = 'admin' AND actor_user_id IS NOT NULL)
      OR (approval_source = 'host' AND actor_user_id IS NULL)
    );

ALTER TABLE theme_runtime_publications
  ADD CONSTRAINT theme_runtime_publications_source_approval_check
    CHECK (
      (source_core_replacements_approved = FALSE AND source_actor_user_id IS NULL)
      OR
      (source_core_replacements_approved = TRUE
        AND source_approval_source = 'admin'
        AND source_actor_user_id IS NOT NULL
        AND source_theme_id IS NOT NULL)
      OR
      (source_core_replacements_approved = TRUE
        AND source_approval_source = 'host'
        AND source_actor_user_id IS NULL
        AND source_theme_id IS NOT NULL)
    );

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM theme_runtime_publications WHERE approval_source = 'host') THEN
    RAISE EXCEPTION 'cannot remove host-approved theme runtime history';
  END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE theme_runtime_publications
  DROP CONSTRAINT IF EXISTS theme_runtime_publications_source_approval_check,
  DROP CONSTRAINT IF EXISTS theme_runtime_publications_approval_check,
  ADD CONSTRAINT theme_runtime_publications_source_approval_check
    CHECK (
      (source_core_replacements_approved = FALSE AND source_actor_user_id IS NULL)
      OR
      (source_core_replacements_approved = TRUE
        AND source_actor_user_id IS NOT NULL
        AND source_theme_id IS NOT NULL)
    ),
  ADD CONSTRAINT theme_runtime_publications_check2
    CHECK (core_replacements_approved = FALSE OR actor_user_id IS NOT NULL);

ALTER TABLE theme_runtime_publications
  DROP COLUMN IF EXISTS source_approval_source,
  DROP COLUMN IF EXISTS approval_source;

ALTER TABLE page_provider_bindings
  DROP COLUMN IF EXISTS approval_source;
