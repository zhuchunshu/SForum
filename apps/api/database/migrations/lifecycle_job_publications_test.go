package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const lifecycleJobPublicationsMigration = "202607140010_extension_lifecycle_job_publications.sql"

func TestLifecycleJobPublicationsMigrationDefinesDeferredExactReconciliation(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecycleJobPublicationsMigration)
	if err != nil {
		t.Fatal(err)
	}
	document := strings.Join(strings.Fields(string(body)), " ")
	for _, required := range []string{
		"CREATE TABLE extension_lifecycle_job_publications",
		"FOREIGN KEY (operation_id, step_id, publication_mode)",
		"REFERENCES extension_lifecycle_publications(operation_id, step_id, publication_mode)",
		"source_snapshot JSONB NOT NULL",
		"target_snapshot JSONB NOT NULL",
		"reconciliation_plan JSONB NOT NULL",
		"publication_state TEXT NOT NULL DEFAULT 'source'",
		"reconciliation_state TEXT NOT NULL DEFAULT 'pending'",
		"reconciliation_result JSONB",
		"octet_length(reconciliation_error) <= 128",
		"reconciliation_error ~ '^[a-z0-9]+([._-][a-z0-9]+)*$'",
		"reconciliation_state = 'failed' AND reconciliation_error <> ''",
		"reconciliation_state <> 'failed' AND reconciliation_error = ''",
		"(reconciliation_state = 'succeeded') = (reconciliation_result IS NOT NULL)",
		"reconciliation_state <> 'succeeded' OR publication_state = 'target'",
		"reconciliation_state = 'pending' AND reconciled_by_user_id IS NULL",
		"reconciliation_state <> 'pending' AND reconciled_by_user_id > 0",
		"reconciled_by_user_id BIGINT",
		"reconciliation_audit_event_id BIGINT",
		"UNIQUE (operation_id, step_id, publication_mode)",
		"WHERE publication_state = 'target' AND reconciliation_state <> 'succeeded'",
	} {
		if !strings.Contains(document, required) {
			t.Fatalf("migration is missing %q", required)
		}
	}
	if strings.Contains(document, "UPDATE river_job") || strings.Contains(document, "DELETE FROM river_job") {
		t.Fatal("lifecycle migration must not mutate River private storage")
	}
}

func TestLifecycleJobPublicationsMigrationRetainsCommittedEvidenceOnDown(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecycleJobPublicationsMigration)
	if err != nil {
		t.Fatal(err)
	}
	document := string(body)
	for _, required := range []string{
		"IF EXISTS (SELECT 1 FROM extension_lifecycle_job_publications)",
		"cannot remove lifecycle job publication history",
		"DROP TABLE IF EXISTS extension_lifecycle_job_publications",
	} {
		if !strings.Contains(document, required) {
			t.Fatalf("migration Down is missing %q", required)
		}
	}
}
