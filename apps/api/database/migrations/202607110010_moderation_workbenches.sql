-- +goose Up
INSERT INTO permissions (key, module, description)
VALUES
  ('moderation.manage', 'moderation', 'Manage moderation settings and audit history.'),
  ('moderation.review', 'moderation', 'Review pending content and moderation reports.')
ON CONFLICT (key) DO UPDATE
SET module = EXCLUDED.module, description = EXCLUDED.description;

INSERT INTO role_permissions (role_id, permission_key)
SELECT role_id, 'moderation.review'
FROM role_permissions
WHERE permission_key = 'moderation.report_review'
ON CONFLICT DO NOTHING;

INSERT INTO user_permission_overrides (user_id, permission_key, effect, updated_by_user_id, created_at, updated_at)
SELECT user_id, 'moderation.review', effect, updated_by_user_id, created_at, updated_at
FROM user_permission_overrides
WHERE permission_key = 'moderation.report_review'
ON CONFLICT (user_id, permission_key) DO UPDATE
SET effect = EXCLUDED.effect,
    updated_by_user_id = EXCLUDED.updated_by_user_id,
    updated_at = EXCLUDED.updated_at;

INSERT INTO role_permissions (role_id, permission_key)
SELECT id, permission.key
FROM roles
CROSS JOIN (VALUES ('moderation.manage'), ('moderation.review')) AS permission(key)
WHERE roles.key = 'super_admin'
ON CONFLICT DO NOTHING;

DELETE FROM permissions WHERE key = 'moderation.report_review';

ALTER TABLE topics DROP CONSTRAINT topics_status_check;
ALTER TABLE topics ADD CONSTRAINT topics_status_check
  CHECK (status IN ('active', 'locked', 'hidden', 'deleted', 'pending', 'rejected'));

ALTER TABLE comments DROP CONSTRAINT comments_status_check;
ALTER TABLE comments ADD CONSTRAINT comments_status_check
  CHECK (status IN ('active', 'hidden', 'deleted', 'pending', 'rejected'));

CREATE TABLE moderation_settings (
  singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
  mode TEXT NOT NULL CHECK (mode IN ('off', 'rules', 'all')),
  review_new_users BOOLEAN NOT NULL,
  new_user_max_age_days INTEGER NOT NULL CHECK (new_user_max_age_days BETWEEN 0 AND 3650),
  review_external_links BOOLEAN NOT NULL,
  updated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO moderation_settings (
  singleton, mode, review_new_users, new_user_max_age_days, review_external_links
) VALUES (TRUE, 'off', TRUE, 7, TRUE);

CREATE TABLE moderation_decisions (
  id BIGSERIAL PRIMARY KEY,
  source TEXT NOT NULL CHECK (source IN ('pre_publish', 'report')),
  target_type TEXT NOT NULL CHECK (target_type IN ('topic', 'comment')),
  target_id BIGINT NOT NULL,
  report_id BIGINT REFERENCES moderation_reports(id) ON DELETE SET NULL,
  action TEXT NOT NULL CHECK (action IN ('approve', 'reject', 'keep_and_close', 'hide_and_close', 'delete_and_close')),
  reviewer_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  review_note TEXT NOT NULL DEFAULT '',
  trigger_snapshot JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX moderation_pending_topics_idx
  ON topics (created_at DESC, id DESC) WHERE status = 'pending';
CREATE INDEX moderation_pending_comments_idx
  ON comments (created_at DESC, id DESC) WHERE status = 'pending';
CREATE INDEX moderation_decisions_created_idx
  ON moderation_decisions (created_at DESC, id DESC);
CREATE INDEX moderation_decisions_target_idx
  ON moderation_decisions (target_type, target_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS moderation_decisions_target_idx;
DROP INDEX IF EXISTS moderation_decisions_created_idx;
DROP INDEX IF EXISTS moderation_pending_comments_idx;
DROP INDEX IF EXISTS moderation_pending_topics_idx;
DROP TABLE IF EXISTS moderation_decisions;
DROP TABLE IF EXISTS moderation_settings;

ALTER TABLE comments DROP CONSTRAINT comments_status_check;
ALTER TABLE comments ADD CONSTRAINT comments_status_check
  CHECK (status IN ('active', 'hidden', 'deleted'));

ALTER TABLE topics DROP CONSTRAINT topics_status_check;
ALTER TABLE topics ADD CONSTRAINT topics_status_check
  CHECK (status IN ('active', 'locked', 'hidden', 'deleted'));

INSERT INTO permissions (key, module, description)
VALUES ('moderation.report_review', 'moderation', 'Review moderation reports.')
ON CONFLICT (key) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key)
SELECT role_id, 'moderation.report_review'
FROM role_permissions
WHERE permission_key = 'moderation.review'
ON CONFLICT DO NOTHING;

INSERT INTO user_permission_overrides (user_id, permission_key, effect, updated_by_user_id, created_at, updated_at)
SELECT user_id, 'moderation.report_review', effect, updated_by_user_id, created_at, updated_at
FROM user_permission_overrides
WHERE permission_key = 'moderation.review'
ON CONFLICT (user_id, permission_key) DO UPDATE
SET effect = EXCLUDED.effect,
    updated_by_user_id = EXCLUDED.updated_by_user_id,
    updated_at = EXCLUDED.updated_at;

DELETE FROM permissions WHERE key IN ('moderation.manage', 'moderation.review');
