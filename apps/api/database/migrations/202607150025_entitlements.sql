-- +goose Up
-- Entitlement 是 provider-neutral 的授权事实；支付、货币和网关语义不属于该表。
CREATE TABLE entitlements (
  id BIGSERIAL PRIMARY KEY,
  subject_type TEXT NOT NULL
    CHECK (subject_type = btrim(subject_type) AND octet_length(subject_type) BETWEEN 1 AND 100),
  subject_id TEXT NOT NULL
    CHECK (subject_id = btrim(subject_id) AND octet_length(subject_id) BETWEEN 1 AND 512),

  scope_kind TEXT NOT NULL CHECK (scope_kind IN ('resource', 'capability')),
  resource_type TEXT
    CHECK (resource_type IS NULL OR (
      resource_type = btrim(resource_type) AND octet_length(resource_type) BETWEEN 1 AND 100
    )),
  resource_id TEXT
    CHECK (resource_id IS NULL OR (
      resource_id = btrim(resource_id) AND octet_length(resource_id) BETWEEN 1 AND 512
    )),
  capability TEXT
    CHECK (capability IS NULL OR (
      capability = btrim(capability) AND octet_length(capability) BETWEEN 1 AND 200
    )),

  status TEXT NOT NULL CHECK (status IN ('active', 'revoked', 'expired')),
  source_type TEXT NOT NULL
    CHECK (source_type = btrim(source_type) AND octet_length(source_type) BETWEEN 1 AND 100),
  source_id TEXT NOT NULL
    CHECK (source_id = btrim(source_id) AND octet_length(source_id) BETWEEN 1 AND 512),
  valid_from TIMESTAMPTZ NOT NULL,
  valid_until TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  expired_at TIMESTAMPTZ,
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),

  CHECK (valid_until IS NULL OR valid_until > valid_from),
  CHECK (
    (scope_kind = 'resource' AND resource_type IS NOT NULL AND resource_id IS NOT NULL AND capability IS NULL)
    OR
    (scope_kind = 'capability' AND resource_type IS NULL AND resource_id IS NULL AND capability IS NOT NULL)
  ),
  CHECK (
    (status = 'active' AND revoked_at IS NULL AND expired_at IS NULL)
    OR
    (status = 'revoked' AND revoked_at IS NOT NULL AND expired_at IS NULL)
    OR
    (status = 'expired' AND revoked_at IS NULL AND expired_at IS NOT NULL)
  )
);

-- 事件本身是不可变审计证据；audit_event_id 不设外键，避免审计保留期清理
-- 阻塞 entitlement 历史，同时仍保留当时写入 audit_events 的关联编号。
CREATE TABLE entitlement_events (
  id BIGSERIAL PRIMARY KEY,
  entitlement_id BIGINT NOT NULL REFERENCES entitlements(id) ON DELETE RESTRICT,
  action TEXT NOT NULL CHECK (action IN ('grant', 'revoke', 'expire')),
  idempotency_key TEXT NOT NULL UNIQUE
    CHECK (
      octet_length(idempotency_key) BETWEEN 1 AND 128
      AND idempotency_key !~ '[^!-~]'
    ),
  request_fingerprint TEXT NOT NULL
    CHECK (request_fingerprint ~ '^[0-9a-f]{64}$'),
  previous_status TEXT CHECK (previous_status IS NULL OR previous_status IN ('active', 'revoked', 'expired')),
  next_status TEXT NOT NULL CHECK (next_status IN ('active', 'revoked', 'expired')),
  actor_user_id BIGINT CHECK (actor_user_id IS NULL OR actor_user_id > 0),
  audit_event_id BIGINT NOT NULL CHECK (audit_event_id > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),

  CHECK (
    (action = 'grant' AND previous_status IS NULL AND next_status = 'active')
    OR
    (action = 'revoke' AND previous_status = 'active' AND next_status = 'revoked')
    OR
    (action = 'expire' AND previous_status = 'active' AND next_status = 'expired')
  )
);

CREATE INDEX entitlements_resource_effective_idx
  ON entitlements (
    subject_type, subject_id, resource_type, resource_id, valid_from, valid_until
  )
  WHERE status = 'active' AND scope_kind = 'resource';

CREATE INDEX entitlements_capability_effective_idx
  ON entitlements (
    subject_type, subject_id, capability, valid_from, valid_until
  )
  WHERE status = 'active' AND scope_kind = 'capability';

CREATE INDEX entitlements_source_idx
  ON entitlements (source_type, source_id, created_at DESC, id DESC);

CREATE INDEX entitlement_events_entitlement_idx
  ON entitlement_events (entitlement_id, created_at DESC, id DESC);

-- +goose Down
-- Entitlement 与事件是授权和撤销证据；一旦存在数据，回滚必须显式处理，不能静默删除。
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM entitlement_events) OR EXISTS (SELECT 1 FROM entitlements) THEN
    RAISE EXCEPTION 'cannot remove entitlement evidence';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS entitlement_events_entitlement_idx;
DROP INDEX IF EXISTS entitlements_source_idx;
DROP INDEX IF EXISTS entitlements_capability_effective_idx;
DROP INDEX IF EXISTS entitlements_resource_effective_idx;
DROP TABLE IF EXISTS entitlement_events;
DROP TABLE IF EXISTS entitlements;
