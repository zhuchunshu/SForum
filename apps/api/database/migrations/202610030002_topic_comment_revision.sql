-- +goose Up
-- 评论区实时信号：topics.comment_revision 是「公开评论可见状态」的修订号。
-- 它只作为 SSE 的唤醒/对账信号，正文、权限与可见性始终由 ListComments 裁决，
-- 因此这里不做内容投影，也不参与任何授权判断。
--
-- 为什么用触发器而不是应用层递增：评论写入并不只发生在 forum 包里
-- （moderation 审核批准/删除走 workbench_store 直接 UPDATE，seed 走 COPY，
-- admin 路径直接 UPDATE status），任何一处漏调用都会让信号永久失真。
--
-- 采用语句级 + transition table：一次 COPY/批量审核只产生一次 topics 更新与
-- 一次 NOTIFY，避免逐行触发器在 5 万条 seed 或批量审核时的写放大。
ALTER TABLE topics ADD COLUMN IF NOT EXISTS comment_revision BIGINT NOT NULL DEFAULT 0;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION sforum_bump_topic_comment_revision() RETURNS trigger AS $$
BEGIN
  IF TG_OP = 'INSERT' THEN
    -- 评论落库即公开（pending/rejected 不算公开变化）。
    UPDATE topics SET comment_revision = comment_revision + 1
    WHERE id IN (SELECT DISTINCT topic_id FROM revision_inserted WHERE status = 'active');
    PERFORM pg_notify('sforum_forum_comment_revision', topics.id::text)
    FROM topics
    WHERE topics.id IN (SELECT DISTINCT topic_id FROM revision_inserted WHERE status = 'active');
  ELSIF TG_OP = 'UPDATE' THEN
    -- 公开可见变化：状态迁移、回复计数、正文编辑时间。
    -- 插入后写入 path_key/root_comment_id/depth 的定位更新会同时刷新 updated_at，
    -- 但它不是读者可见的变化，用 path 守卫排除，避免每条新评论触发两次信号。
    UPDATE topics SET comment_revision = comment_revision + 1
    WHERE id IN (
      SELECT DISTINCT revision_new.topic_id
      FROM revision_new
      JOIN revision_old ON revision_old.id = revision_new.id
      WHERE (revision_new.status = 'active' OR revision_old.status = 'active')
        AND (
          revision_old.status IS DISTINCT FROM revision_new.status
          OR revision_old.reply_count IS DISTINCT FROM revision_new.reply_count
          OR (
            revision_old.updated_at IS DISTINCT FROM revision_new.updated_at
            AND revision_old.path_key IS NOT DISTINCT FROM revision_new.path_key
            AND revision_old.root_comment_id IS NOT DISTINCT FROM revision_new.root_comment_id
            AND revision_old.depth IS NOT DISTINCT FROM revision_new.depth
          )
        )
    );
    PERFORM pg_notify('sforum_forum_comment_revision', topics.id::text)
    FROM topics
    WHERE topics.id IN (
      SELECT DISTINCT revision_new.topic_id
      FROM revision_new
      JOIN revision_old ON revision_old.id = revision_new.id
      WHERE (revision_new.status = 'active' OR revision_old.status = 'active')
        AND (
          revision_old.status IS DISTINCT FROM revision_new.status
          OR revision_old.reply_count IS DISTINCT FROM revision_new.reply_count
          OR (
            revision_old.updated_at IS DISTINCT FROM revision_new.updated_at
            AND revision_old.path_key IS NOT DISTINCT FROM revision_new.path_key
            AND revision_old.root_comment_id IS NOT DISTINCT FROM revision_new.root_comment_id
            AND revision_old.depth IS NOT DISTINCT FROM revision_new.depth
          )
        )
    );
  ELSE
    -- 软删/物理删：公开列表少一行。
    UPDATE topics SET comment_revision = comment_revision + 1
    WHERE id IN (SELECT DISTINCT topic_id FROM revision_deleted WHERE status = 'active');
    PERFORM pg_notify('sforum_forum_comment_revision', topics.id::text)
    FROM topics
    WHERE topics.id IN (SELECT DISTINCT topic_id FROM revision_deleted WHERE status = 'active');
  END IF;
  RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- 三个语句级触发器各自声明 transition table；函数按 TG_OP 只规划命中的分支。
DROP TRIGGER IF EXISTS comments_revision_insert ON comments;
CREATE TRIGGER comments_revision_insert
  AFTER INSERT ON comments
  REFERENCING NEW TABLE AS revision_inserted
  FOR EACH STATEMENT EXECUTE FUNCTION sforum_bump_topic_comment_revision();

DROP TRIGGER IF EXISTS comments_revision_update ON comments;
CREATE TRIGGER comments_revision_update
  AFTER UPDATE ON comments
  REFERENCING OLD TABLE AS revision_old NEW TABLE AS revision_new
  FOR EACH STATEMENT EXECUTE FUNCTION sforum_bump_topic_comment_revision();

DROP TRIGGER IF EXISTS comments_revision_delete ON comments;
CREATE TRIGGER comments_revision_delete
  AFTER DELETE ON comments
  REFERENCING OLD TABLE AS revision_deleted
  FOR EACH STATEMENT EXECUTE FUNCTION sforum_bump_topic_comment_revision();

-- +goose Down
DROP TRIGGER IF EXISTS comments_revision_insert ON comments;
DROP TRIGGER IF EXISTS comments_revision_update ON comments;
DROP TRIGGER IF EXISTS comments_revision_delete ON comments;
DROP FUNCTION IF EXISTS sforum_bump_topic_comment_revision();
ALTER TABLE topics DROP COLUMN IF EXISTS comment_revision;
