package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const identityStaleRoleSuggestionRejectionMigration = "202607290077_identity_stale_role_suggestion_rejection.sql"

func TestIdentityStaleRoleSuggestionRejectionMigrationKeepsApprovalFailClosed(t *testing.T) {
	body, err := fs.ReadFile(Files(), identityStaleRoleSuggestionRejectionMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("stale role suggestion rejection migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE OR REPLACE FUNCTION validate_extension_permission_role_suggestion_decision() RETURNS trigger",
		"IF NEW.approval_state = 'approved' THEN",
		"permission role suggestion decision exact artifact is inactive",
		"permission role suggestion decision is stale",
		"permission role suggestion decision actor lacks role.manage",
		"permission role suggestion rejection audit claims a grant",
		"permission role suggestion rejection cannot carry grant evidence",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("stale role suggestion rejection migration missing %q", clause)
		}
	}

	approvalGate := strings.Index(up, "IF NEW.approval_state = 'approved' THEN")
	artifactGate := strings.Index(up, "FROM extension_versions AS version")
	actorGate := strings.Index(up, "SELECT status INTO actor_status")
	if approvalGate < 0 || artifactGate < approvalGate || actorGate < artifactGate {
		t.Fatal("exact artifact validation must stay inside the approval branch before actor authorization")
	}
	for _, forbidden := range []string{
		"INSERT INTO role_permissions",
		"DELETE FROM role_permissions",
		"DELETE FROM extension_permission_role_suggestions",
	} {
		if strings.Contains(strings.ToUpper(up), forbidden) {
			t.Fatalf("stale role suggestion rejection migration must not contain %q", forbidden)
		}
	}
}

func TestIdentityStaleRoleSuggestionRejectionMigrationIsForwardOnly(t *testing.T) {
	body, err := fs.ReadFile(Files(), identityStaleRoleSuggestionRejectionMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("stale role suggestion rejection migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	if !strings.Contains(down, "cannot reverse stale role suggestion rejection repair 202607290077") {
		t.Fatal("stale role suggestion rejection migration must remain forward-only")
	}
}
