-- +goose Up
-- Stable core views deliberately use security-definer semantics
-- (security_invoker=false): a future SELECT-only grant on a view must not imply
-- a grant on its base tables. security_barrier keeps caller predicates outside
-- the Host-owned filtering boundary. OFFSET 0 makes every view structurally
-- non-updatable even for its owner.
CREATE SCHEMA sforum_core_v1 AUTHORIZATION CURRENT_USER;
REVOKE ALL ON SCHEMA sforum_core_v1 FROM PUBLIC;

CREATE VIEW sforum_core_v1.safe_users
WITH (security_barrier = true, security_invoker = false) AS
SELECT
  users.id,
  users.username,
  users.display_name,
  users.created_at,
  users.updated_at
FROM public.users
WHERE users.status = 'active'
OFFSET 0;

COMMENT ON VIEW sforum_core_v1.safe_users IS
  'SForum core view v1: active safe identity without email, credentials, token state, locale, or administrative flags.';

CREATE VIEW sforum_core_v1.forum_topics
WITH (security_barrier = true, security_invoker = false) AS
SELECT
  topics.id,
  topics.category_id,
  categories.slug AS category_slug,
  topics.author_user_id,
  topics.title,
  topics.slug,
  topics.status,
  topics.is_pinned,
  topics.comment_count,
  topics.view_count,
  topics.last_activity_at,
  topics.created_at,
  topics.updated_at,
  posts.html_content,
  posts.plain_text,
  posts.source_format,
  posts.render_version,
  posts.content_hash
FROM public.topics
JOIN public.categories ON categories.id = topics.category_id
JOIN public.posts ON posts.id = topics.content_id
WHERE topics.status IN ('active', 'locked')
  AND topics.deleted_at IS NULL
  AND categories.visibility = 'public'
OFFSET 0;

COMMENT ON VIEW sforum_core_v1.forum_topics IS
  'SForum core view v1: public rendered topic content without source documents, IP addresses, or moderation internals.';

CREATE VIEW sforum_core_v1.forum_comments
WITH (security_barrier = true, security_invoker = false) AS
SELECT
  comments.id,
  comments.topic_id,
  comments.author_user_id,
  comments.parent_comment_id,
  comments.root_comment_id,
  comments.path_key,
  comments.depth,
  comments.reply_count,
  comments.created_at,
  comments.updated_at,
  posts.html_content,
  posts.plain_text,
  posts.source_format,
  posts.render_version,
  posts.content_hash
FROM public.comments
JOIN public.topics ON topics.id = comments.topic_id
JOIN public.categories ON categories.id = topics.category_id
JOIN public.posts ON posts.id = comments.content_id
WHERE comments.status = 'active'
  AND comments.deleted_at IS NULL
  AND topics.status IN ('active', 'locked')
  AND topics.deleted_at IS NULL
  AND categories.visibility = 'public'
OFFSET 0;

COMMENT ON VIEW sforum_core_v1.forum_comments IS
  'SForum core view v1: public rendered comment content without source documents, IP addresses, or moderation internals.';

CREATE VIEW sforum_core_v1.public_entity_meta
WITH (security_barrier = true, security_invoker = false) AS
SELECT
  values.entity_type,
  values.entity_id,
  values.field_key,
  definitions.value_type,
  values.value_text,
  definitions.owner_extension_id,
  values.updated_at
FROM public.entity_meta_values AS values
JOIN public.entity_field_definitions AS definitions
  ON definitions.field_key = values.field_key
WHERE definitions.enabled
  AND definitions.visibility = 'public'
OFFSET 0;

COMMENT ON VIEW sforum_core_v1.public_entity_meta IS
  'SForum core view v1: enabled public entity metadata without owner/admin fields or editor identity.';

CREATE VIEW sforum_core_v1.public_attachment_metadata
WITH (security_barrier = true, security_invoker = false) AS
SELECT
  attachments.id,
  attachments.public_id,
  attachments.owner_user_id,
  attachments.original_name,
  attachments.content_type,
  attachments.extension,
  attachments.size_bytes,
  attachments.sha256,
  attachments.image_width,
  attachments.image_height,
  attachments.reference_count,
  attachments.created_at,
  attachments.updated_at
FROM public.attachments
WHERE attachments.status = 'active'
  AND attachments.visibility = 'public'
  AND attachments.deleted_at IS NULL
OFFSET 0;

COMMENT ON VIEW sforum_core_v1.public_attachment_metadata IS
  'SForum core view v1: public attachment metadata without provider names, storage object keys, or deleted/private objects.';

REVOKE ALL ON ALL TABLES IN SCHEMA sforum_core_v1 FROM PUBLIC;

-- +goose Down
DROP VIEW IF EXISTS sforum_core_v1.public_attachment_metadata;
DROP VIEW IF EXISTS sforum_core_v1.public_entity_meta;
DROP VIEW IF EXISTS sforum_core_v1.forum_comments;
DROP VIEW IF EXISTS sforum_core_v1.forum_topics;
DROP VIEW IF EXISTS sforum_core_v1.safe_users;
DROP SCHEMA IF EXISTS sforum_core_v1;
