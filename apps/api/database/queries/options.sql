-- name: ListWebOptions :many
SELECT name, value
FROM web_options
ORDER BY name;

-- name: GetWebOption :one
SELECT name, value
FROM web_options
WHERE name = $1;

-- name: InsertMissingWebOption :exec
INSERT INTO web_options (name, value)
VALUES ($1, $2)
ON CONFLICT (name) DO NOTHING;

-- name: UpsertWebOption :one
INSERT INTO web_options (name, value)
VALUES ($1, $2)
ON CONFLICT (name) DO UPDATE
SET value = EXCLUDED.value
RETURNING name, value;
