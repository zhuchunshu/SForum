-- +goose Up
-- Host-owned namespaced Secret Store (V3 P11).
-- Ciphertext only (OptionCipher enc:: format). No plaintext secrets.
CREATE TABLE IF NOT EXISTS secret_store (
    namespace   TEXT        NOT NULL,
    secret_id  TEXT        NOT NULL,
    version    BIGINT      NOT NULL CHECK (version > 0),
    value      TEXT        NOT NULL DEFAULT '',
    media_type TEXT        NOT NULL DEFAULT 'text/plain',
    purposes   TEXT[]      NOT NULL DEFAULT '{}',
    revoked    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT        NOT NULL DEFAULT '',
    PRIMARY KEY (namespace, secret_id, version)
);

CREATE INDEX IF NOT EXISTS secret_store_namespace_idx
    ON secret_store (namespace);

CREATE INDEX IF NOT EXISTS secret_store_latest_idx
    ON secret_store (namespace, secret_id, version DESC)
    WHERE revoked = FALSE;

COMMENT ON TABLE secret_store IS 'Host Secret Store: versioned ciphertext secrets by namespace; plugins Resolve only.';

-- +goose Down
DROP TABLE IF EXISTS secret_store;
