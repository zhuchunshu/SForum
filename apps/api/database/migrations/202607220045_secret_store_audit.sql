-- +goose Up
-- Durable Secret Store audit (no plaintext values). Complements secret_store.
CREATE TABLE IF NOT EXISTS secret_store_audit (
    id         BIGSERIAL PRIMARY KEY,
    audit_id   TEXT        NOT NULL,
    action     TEXT        NOT NULL,
    namespace  TEXT        NOT NULL DEFAULT '',
    secret_id  TEXT        NOT NULL DEFAULT '',
    version    BIGINT      NOT NULL DEFAULT 0,
    actor      TEXT        NOT NULL DEFAULT '',
    purpose    TEXT        NOT NULL DEFAULT '',
    ok         BOOLEAN     NOT NULL DEFAULT FALSE,
    at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS secret_store_audit_at_idx
    ON secret_store_audit (at DESC, id DESC);

CREATE INDEX IF NOT EXISTS secret_store_audit_namespace_idx
    ON secret_store_audit (namespace, secret_id, at DESC);

COMMENT ON TABLE secret_store_audit IS 'Host Secret Store audit: lifecycle events without secret values.';

-- +goose Down
DROP TABLE IF EXISTS secret_store_audit;
