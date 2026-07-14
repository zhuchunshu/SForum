-- +goose Up
-- V3 P4：恢复决定必须记录“本次是谁、关联哪条审计、是否接受跳过/强制风险”。
-- 原 requested_by_user_id/authority_snapshot 继续表示最初 exact-artifact 授权，不能被恢复操作者覆盖。
ALTER TABLE extension_lifecycle_operations
  ADD COLUMN recovery_actor_user_id BIGINT,
  ADD COLUMN recovery_audit_event_id BIGINT,
  ADD CONSTRAINT extension_lifecycle_operations_recovery_actor_check
    CHECK (recovery_actor_user_id IS NULL OR recovery_actor_user_id > 0),
  ADD CONSTRAINT extension_lifecycle_operations_recovery_audit_check
    CHECK (recovery_audit_event_id IS NULL OR recovery_audit_event_id > 0),
  ADD CONSTRAINT extension_lifecycle_operations_recovery_pair_check
    CHECK ((recovery_actor_user_id IS NULL) = (recovery_audit_event_id IS NULL));

CREATE TABLE extension_lifecycle_recovery_decisions (
  id BIGSERIAL PRIMARY KEY,
  operation_id BIGINT NOT NULL
    REFERENCES extension_lifecycle_operations(id) ON DELETE CASCADE,
  operation_attempt INTEGER NOT NULL CHECK (operation_attempt > 1),
  decision TEXT NOT NULL CHECK (decision IN ('retry', 'skip_step')),
  escalate_forced BOOLEAN NOT NULL DEFAULT FALSE,
  reason TEXT NOT NULL DEFAULT '' CHECK (octet_length(reason) <= 4096),
  actor_user_id BIGINT NOT NULL CHECK (actor_user_id > 0),
  -- audit_events 可按保留期清理；保留不可复用的数值关联而不建立 FK。
  audit_event_id BIGINT NOT NULL CHECK (audit_event_id > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (operation_id, operation_attempt),
  CHECK (decision = 'retry' OR octet_length(reason) > 0),
  CHECK (escalate_forced = FALSE OR octet_length(reason) > 0)
);

CREATE INDEX extension_lifecycle_recovery_decisions_operation_idx
  ON extension_lifecycle_recovery_decisions (operation_id, operation_attempt DESC, id DESC);

-- +goose Down
-- 一旦有恢复/跳过/强制决定，降级会丢失操作者和残余风险证据，因此必须显式阻止。
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_lifecycle_recovery_decisions)
     OR EXISTS (
       SELECT 1 FROM extension_lifecycle_operations
       WHERE recovery_actor_user_id IS NOT NULL OR recovery_audit_event_id IS NOT NULL
     ) THEN
    RAISE EXCEPTION 'cannot remove lifecycle recovery decision history';
  END IF;
END $$;

DROP INDEX IF EXISTS extension_lifecycle_recovery_decisions_operation_idx;
DROP TABLE IF EXISTS extension_lifecycle_recovery_decisions;
ALTER TABLE extension_lifecycle_operations
  DROP CONSTRAINT IF EXISTS extension_lifecycle_operations_recovery_pair_check,
  DROP CONSTRAINT IF EXISTS extension_lifecycle_operations_recovery_audit_check,
  DROP CONSTRAINT IF EXISTS extension_lifecycle_operations_recovery_actor_check,
  DROP COLUMN IF EXISTS recovery_audit_event_id,
  DROP COLUMN IF EXISTS recovery_actor_user_id;
