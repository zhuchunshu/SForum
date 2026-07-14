package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const lifecycleRecoveryDecisionsMigration = "202607140006_extension_lifecycle_recovery_decisions.sql"

func TestLifecycleRecoveryDecisionsMigrationIsEmbedded(t *testing.T) {
	if _, err := fs.Stat(Files(), lifecycleRecoveryDecisionsMigration); err != nil {
		t.Fatalf("embedded recovery decisions migration: %v", err)
	}
}

func TestLifecycleRecoveryDecisionsMigrationPreservesAuthorityAndAuditHistory(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecycleRecoveryDecisionsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("recovery decisions migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"ADD COLUMN recovery_actor_user_id BIGINT",
		"ADD COLUMN recovery_audit_event_id BIGINT",
		"(recovery_actor_user_id IS NULL) = (recovery_audit_event_id IS NULL)",
		"CREATE TABLE extension_lifecycle_recovery_decisions",
		"operation_attempt INTEGER NOT NULL CHECK (operation_attempt > 1)",
		"decision TEXT NOT NULL CHECK (decision IN ('retry', 'skip_step'))",
		"escalate_forced BOOLEAN NOT NULL DEFAULT FALSE",
		"actor_user_id BIGINT NOT NULL CHECK (actor_user_id > 0)",
		"audit_event_id BIGINT NOT NULL CHECK (audit_event_id > 0)",
		"UNIQUE (operation_id, operation_attempt)",
		"decision = 'retry' OR octet_length(reason) > 0",
		"escalate_forced = FALSE OR octet_length(reason) > 0",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("recovery decisions Up missing %q", clause)
		}
	}
	if strings.Contains(up, "REFERENCES users") || strings.Contains(up, "REFERENCES audit_events") {
		t.Fatal("recovery actor/audit evidence must survive user and audit retention")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	guard := "IF EXISTS (SELECT 1 FROM extension_lifecycle_recovery_decisions)"
	if !strings.Contains(down, guard) || !strings.Contains(down, "RAISE EXCEPTION 'cannot remove lifecycle recovery decision history'") {
		t.Fatal("recovery Down must refuse to delete retained decisions")
	}
	if strings.Index(down, guard) > strings.Index(down, "DROP TABLE IF EXISTS extension_lifecycle_recovery_decisions") {
		t.Fatal("recovery Down drops history before its retention guard")
	}
}
