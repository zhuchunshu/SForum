package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestAttachmentUploadPoliciesMigrationKeepsIdentityReferencesAndBounds(t *testing.T) {
	body, err := fs.ReadFile(Files(), "202607310002_attachment_upload_policies.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("attachment upload policies migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"role_id BIGINT PRIMARY KEY REFERENCES roles(id) ON DELETE CASCADE",
		"user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE",
		"max_file_size_bytes BIGINT NOT NULL CHECK (max_file_size_bytes > 0)",
		"updated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL",
		"'attachment.upload_policy.manage'",
		"roles.key IN ('super_admin', 'operator')",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("attachment upload policies migration missing %q", clause)
		}
	}
}
