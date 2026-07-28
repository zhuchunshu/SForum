-- +goose Up
-- One site-wide CAS token protects the complete notification policy document.
CREATE TABLE notification_policy_revisions (
  singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO notification_policy_revisions (singleton, revision)
VALUES (TRUE, 1)
ON CONFLICT (singleton) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS notification_policy_revisions;
