-- +goose Up
-- V3 P12：030 只建立兼容历史行的 nullable 外键。本迁移收紧新写入：最终
-- marker 必须在同一 INSERT/UPDATE 绑定 desired revision；历史 committed/NULL
-- 不能事后补绑，避免把新证据伪装成旧生命周期决策的一部分。
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION enforce_lifecycle_plugin_runtime_binding() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF TG_OP = 'UPDATE'
    AND OLD.commit_marker IS TRUE
    AND NEW.commit_marker IS NOT TRUE THEN
    RAISE EXCEPTION 'committed lifecycle marker is immutable';
  END IF;

  IF NEW.plugin_runtime_publication_revision IS NOT NULL
    AND NEW.commit_marker IS NOT TRUE THEN
    RAISE EXCEPTION 'lifecycle plugin runtime binding requires committed marker';
  END IF;

  IF TG_OP = 'INSERT' THEN
    IF NEW.commit_marker IS TRUE
      AND NEW.plugin_runtime_publication_revision IS NULL THEN
      RAISE EXCEPTION 'new committed lifecycle marker requires plugin runtime binding';
    END IF;
    RETURN NEW;
  END IF;

  IF OLD.commit_marker IS FALSE
    AND NEW.commit_marker IS TRUE
    AND NEW.plugin_runtime_publication_revision IS NULL THEN
    RAISE EXCEPTION 'new committed lifecycle marker requires plugin runtime binding';
  END IF;
  IF OLD.commit_marker IS TRUE
    AND OLD.plugin_runtime_publication_revision IS NULL
    AND NEW.plugin_runtime_publication_revision IS NOT NULL THEN
    RAISE EXCEPTION 'historical lifecycle marker binding cannot be backfilled';
  END IF;
  IF OLD.plugin_runtime_publication_revision IS NOT NULL
    AND NEW.plugin_runtime_publication_revision IS DISTINCT FROM OLD.plugin_runtime_publication_revision THEN
    RAISE EXCEPTION 'lifecycle plugin runtime binding is immutable';
  END IF;

  RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- 任何 031 之后写入的绑定都依赖同一 CAS 约束，存在时禁止降级到可补绑语义。
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM extension_lifecycle_publications
    WHERE plugin_runtime_publication_revision IS NOT NULL
  ) THEN
    RAISE EXCEPTION 'cannot weaken lifecycle plugin runtime binding enforcement';
  END IF;
END $$;
-- +goose StatementEnd

-- 恢复 030 的兼容函数；仅在完全没有新 binding evidence 时允许。
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION enforce_lifecycle_plugin_runtime_binding() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF NEW.plugin_runtime_publication_revision IS NOT NULL
    AND NEW.commit_marker IS NOT TRUE THEN
    RAISE EXCEPTION 'lifecycle plugin runtime binding requires committed marker';
  END IF;

  IF TG_OP = 'UPDATE'
    AND OLD.plugin_runtime_publication_revision IS NOT NULL
    AND NEW.plugin_runtime_publication_revision IS DISTINCT FROM OLD.plugin_runtime_publication_revision THEN
    RAISE EXCEPTION 'lifecycle plugin runtime binding is immutable';
  END IF;

  RETURN NEW;
END;
$$;
-- +goose StatementEnd
