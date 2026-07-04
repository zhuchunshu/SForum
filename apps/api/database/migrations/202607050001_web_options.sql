-- +goose Up
CREATE TABLE web_options (
  name TEXT PRIMARY KEY,
  value TEXT NOT NULL DEFAULT ''
);

INSERT INTO web_options (name, value)
VALUES
  ('site.name', 'SForum');

-- +goose Down
DROP TABLE IF EXISTS web_options;
