-- +goose Up
-- V3 P12：生命周期最终 marker 与 executable plugin desired full-set 必须指向
-- 同一事务内确定的不可变 revision。历史 marker 没有该证据，保持 NULL；新代码
-- 只允许在提交 marker 的同一 CAS UPDATE 中写入绑定。
ALTER TABLE extension_lifecycle_publications
  ADD COLUMN plugin_runtime_publication_revision BIGINT;

ALTER TABLE extension_lifecycle_publications
  ADD CONSTRAINT extension_lifecycle_publications_plugin_runtime_fk
  FOREIGN KEY (plugin_runtime_publication_revision)
  REFERENCES plugin_runtime_publications(revision) ON DELETE RESTRICT;

ALTER TABLE extension_lifecycle_publications
  ADD CONSTRAINT extension_lifecycle_publications_plugin_runtime_state_chk
  CHECK (commit_marker = TRUE OR plugin_runtime_publication_revision IS NULL);

CREATE INDEX extension_lifecycle_publications_plugin_runtime_idx
  ON extension_lifecycle_publications (plugin_runtime_publication_revision)
  WHERE plugin_runtime_publication_revision IS NOT NULL;

-- +goose StatementBegin
CREATE FUNCTION enforce_lifecycle_plugin_runtime_binding() RETURNS trigger
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

CREATE TRIGGER extension_lifecycle_publications_plugin_runtime_valid
BEFORE INSERT OR UPDATE ON extension_lifecycle_publications
FOR EACH ROW EXECUTE FUNCTION enforce_lifecycle_plugin_runtime_binding();

-- +goose Down
-- 已绑定 revision 是最终生命周期决策的一部分；存在任何新证据时禁止降级丢失。
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM extension_lifecycle_publications
    WHERE plugin_runtime_publication_revision IS NOT NULL
  ) THEN
    RAISE EXCEPTION 'cannot remove lifecycle plugin runtime binding history';
  END IF;
END $$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS extension_lifecycle_publications_plugin_runtime_valid
  ON extension_lifecycle_publications;
DROP FUNCTION IF EXISTS enforce_lifecycle_plugin_runtime_binding();
DROP INDEX IF EXISTS extension_lifecycle_publications_plugin_runtime_idx;
ALTER TABLE extension_lifecycle_publications
  DROP CONSTRAINT IF EXISTS extension_lifecycle_publications_plugin_runtime_state_chk;
ALTER TABLE extension_lifecycle_publications
  DROP CONSTRAINT IF EXISTS extension_lifecycle_publications_plugin_runtime_fk;
ALTER TABLE extension_lifecycle_publications
  DROP COLUMN IF EXISTS plugin_runtime_publication_revision;
