-- +goose Up
CREATE TABLE user_permission_overrides (
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  permission_key TEXT NOT NULL REFERENCES permissions(key) ON DELETE CASCADE,
  effect TEXT NOT NULL CHECK (effect IN ('allow', 'deny')),
  updated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, permission_key)
);

CREATE INDEX user_permission_overrides_permission_key_idx
  ON user_permission_overrides (permission_key);

-- +goose Down
DROP TABLE IF EXISTS user_permission_overrides;
