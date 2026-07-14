package extensionsruntime

import (
	"context"
	"errors"
	"testing"
)

func TestParseExtensionDatabaseMigrationTransactionPolicies(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		policy        string
		transactional bool
		warning       string
	}{
		{name: "required plain SQL", body: `CREATE TABLE items (id BIGINT);`, policy: "required", transactional: true},
		{name: "forbidden plain SQL", body: `CREATE INDEX CONCURRENTLY items_id_idx ON items (id);`, policy: "forbidden", warning: extensionDatabaseMigrationWarningNonTransactional},
		{name: "auto defaults transactional", body: `ALTER TABLE items ADD COLUMN label TEXT;`, policy: "auto", transactional: true},
		{name: "auto honors goose no transaction", body: `-- +goose NO TRANSACTION
-- +goose Up
VACUUM items;`, policy: "auto", warning: extensionDatabaseMigrationWarningNonTransactional},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := parseExtensionDatabaseMigration(context.Background(), []byte(test.body), "up", test.policy)
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Transactional != test.transactional || parsed.WarningCode != test.warning || len(parsed.Statements) != 1 {
				t.Fatalf("unexpected parsed plan: %#v", parsed)
			}
		})
	}
}

func TestParseExtensionDatabaseMigrationUsesGooseSectionsAndStatementBlocks(t *testing.T) {
	body := []byte(`-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION answer() RETURNS BIGINT AS $$
BEGIN
  RETURN 42;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
-- +goose Down
DROP FUNCTION answer();`)
	up, err := parseExtensionDatabaseMigration(context.Background(), body, "up", "required")
	if err != nil {
		t.Fatal(err)
	}
	if len(up.Statements) != 1 || up.Statements[0] == "" {
		t.Fatalf("function body was not kept as one Goose statement: %#v", up.Statements)
	}
	down, err := parseExtensionDatabaseMigration(context.Background(), body, "down", "required")
	if err != nil {
		t.Fatal(err)
	}
	if len(down.Statements) != 1 || down.Statements[0] == up.Statements[0] {
		t.Fatalf("Goose direction was not respected: up=%#v down=%#v", up.Statements, down.Statements)
	}
}

func TestParseExtensionDatabaseMigrationRejectsPolicyMismatchAndMissingDown(t *testing.T) {
	body := []byte(`-- +goose NO TRANSACTION
-- +goose Up
VACUUM items;`)
	if _, err := parseExtensionDatabaseMigration(context.Background(), body, "up", "required"); !errors.Is(err, ErrExtensionDatabaseMigrationPolicy) {
		t.Fatalf("required policy accepted NO TRANSACTION: %v", err)
	}
	if _, err := parseExtensionDatabaseMigration(context.Background(), []byte(`CREATE TABLE items (id BIGINT);`), "down", "required"); !errors.Is(err, ErrExtensionDatabaseMigrationParse) {
		t.Fatalf("missing Down section was accepted: %v", err)
	}
}
