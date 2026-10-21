-- +goose Up
-- +sforum OnlineSafe
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '1min';

-- 昵称（users.display_name）是展示标签，不是身份：唯一性只属于 username。
-- 这里只记录「本人上次改名时间」用于冷却判定。管理员改名不写该列——
-- 一次版务修正不应该把用户锁在冷却里。
ALTER TABLE users ADD COLUMN IF NOT EXISTS nickname_changed_at TIMESTAMPTZ;

-- 改名历史。产品语义（谁在什么时候把昵称从 A 改成 B）与 audit_events 分开：
-- 审计事件只记新值，事后还原不出旧名。
CREATE TABLE IF NOT EXISTS user_nickname_changes (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  old_name TEXT NOT NULL,
  new_name TEXT NOT NULL,
  source TEXT NOT NULL CHECK (source IN ('self', 'admin')),
  changed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS user_nickname_changes_user_created_idx
  ON user_nickname_changes (user_id, created_at DESC);

-- 改名后要按作者回写搜索文档的作者昵称；缺这个索引就是全表扫描。
CREATE INDEX IF NOT EXISTS search_documents_author_user_id_idx
  ON search_documents (author_user_id);

-- +goose Down
DROP INDEX IF EXISTS search_documents_author_user_id_idx;
DROP TABLE IF EXISTS user_nickname_changes;
ALTER TABLE users DROP COLUMN IF EXISTS nickname_changed_at;
