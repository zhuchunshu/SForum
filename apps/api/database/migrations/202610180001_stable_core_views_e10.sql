-- +goose Up
-- E10 stable core views (knowledge/reports/2026-10-18-host-query-inventory.md,
-- policy decisions/2026-10-18-extension-data-surface-policy.md). Same security
-- posture as 202607140016: security_invoker=false (definer semantics) so a
-- future SELECT grant on the view never implies a grant on its base tables,
-- security_barrier keeps caller predicates outside the Host-owned filtering
-- boundary, and OFFSET 0 keeps every view structurally non-updatable.

-- Single-row public totals aligned with the stable read views: topics use the
-- forum_topics visibility predicate, comments the forum_comments one, users the
-- safe_users one, attachments the public_attachment_metadata one.
CREATE VIEW sforum_core_v1.site_stats
WITH (security_barrier = true, security_invoker = false) AS
SELECT
  (SELECT COUNT(*)
     FROM public.topics
     JOIN public.categories ON categories.id = topics.category_id
    WHERE topics.status IN ('active', 'locked')
      AND topics.deleted_at IS NULL
      AND categories.visibility = 'public') AS topic_count,
  (SELECT COUNT(*)
     FROM public.comments
     JOIN public.topics ON topics.id = comments.topic_id
     JOIN public.categories ON categories.id = topics.category_id
    WHERE comments.status = 'active'
      AND comments.deleted_at IS NULL
      AND topics.status IN ('active', 'locked')
      AND topics.deleted_at IS NULL
      AND categories.visibility = 'public') AS comment_count,
  (SELECT COUNT(*) FROM public.users WHERE users.status = 'active') AS user_count,
  (SELECT COUNT(*)
     FROM public.attachments
    WHERE attachments.status = 'active'
      AND attachments.visibility = 'public'
      AND attachments.deleted_at IS NULL) AS public_attachment_count
OFFSET 0;

COMMENT ON VIEW sforum_core_v1.site_stats IS
  'SForum core view v1 (E10): public site totals aligned with the stable read views; no per-user, private, or moderation data.';

-- Public categories mirror the shortcode public-category projection predicates
-- (visibility on the category and its group) without moderation or operator
-- fields.
CREATE VIEW sforum_core_v1.public_categories
WITH (security_barrier = true, security_invoker = false) AS
SELECT
  categories.id,
  categories.slug,
  categories.name,
  categories.description,
  categories.icon,
  categories.icon_color
FROM public.categories
JOIN public.category_groups ON category_groups.id = categories.group_id
WHERE categories.visibility = 'public'
  AND category_groups.visibility = 'public'
OFFSET 0;

COMMENT ON VIEW sforum_core_v1.public_categories IS
  'SForum core view v1 (E10): public categories with their group visibility enforced; no operator-only columns.';

-- Public identity + avatar projection: every active user, with the avatar
-- attachment metadata only when the linked attachment is live. No email,
-- credentials, locale, or administrative flags.
CREATE VIEW sforum_core_v1.safe_user_avatars
WITH (security_barrier = true, security_invoker = false) AS
SELECT
  users.id AS user_id,
  users.username,
  users.display_name,
  attachments.public_id AS avatar_public_id,
  attachments.content_type AS avatar_content_type,
  attachments.image_width AS avatar_image_width,
  attachments.image_height AS avatar_image_height
FROM public.users
LEFT JOIN public.user_profiles ON user_profiles.user_id = users.id
LEFT JOIN public.attachments
  ON attachments.id = user_profiles.avatar_attachment_id
 AND attachments.status = 'active'
 AND attachments.deleted_at IS NULL
WHERE users.status = 'active'
OFFSET 0;

COMMENT ON VIEW sforum_core_v1.safe_user_avatars IS
  'SForum core view v1 (E10): active safe identity with live avatar metadata; users without a live avatar keep NULL avatar columns.';

REVOKE ALL ON ALL TABLES IN SCHEMA sforum_core_v1 FROM PUBLIC;

-- +goose Down
DROP VIEW IF EXISTS sforum_core_v1.safe_user_avatars;
DROP VIEW IF EXISTS sforum_core_v1.public_categories;
DROP VIEW IF EXISTS sforum_core_v1.site_stats;
