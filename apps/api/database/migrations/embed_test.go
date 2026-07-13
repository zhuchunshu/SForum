package migrations

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"io/fs"
	"strings"
	"sync"
	"testing"

	"github.com/pressly/goose/v3"
)

func TestFilesIncludesSQLMigrations(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".sql") {
			return
		}
	}
	t.Fatal("expected embedded SQL migrations")
}

func TestFilesIncludesForumTaxonomyMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	const expected = "202607070003_forum_taxonomy.sql"
	for _, entry := range entries {
		if entry.Name() == expected {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", expected)
}

func TestFilesIncludesUserSessionsMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	const expected = "202607100002_user_sessions.sql"
	for _, entry := range entries {
		if entry.Name() == expected {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", expected)
}

func TestFilesIncludesClientIPAddressMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	const expected = "202607120010_client_ip_address.sql"
	for _, entry := range entries {
		if entry.Name() == expected {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", expected)
}

func TestFilesIncludesClientIPFollowupsMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	const expected = "202607120011_client_ip_followups.sql"
	for _, entry := range entries {
		if entry.Name() == expected {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", expected)
}

func TestFilesIncludesImmutableExtensionVersionsMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	const expected = "202607100004_immutable_extension_versions.sql"
	for _, entry := range entries {
		if entry.Name() == expected {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", expected)
}

func TestFilesIncludesAdminFrontendTrustGrantsMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	const expected = "202607100005_admin_frontend_trust_grants.sql"
	for _, entry := range entries {
		if entry.Name() == expected {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", expected)
}

func TestAdminFrontendTrustMigrationBindsImmutableDigest(t *testing.T) {
	body, err := fs.ReadFile(Files(), "202607100005_admin_frontend_trust_grants.sql")
	if err != nil {
		t.Fatalf("read admin frontend trust migration: %v", err)
	}
	sql := strings.Join(strings.Fields(string(body)), " ")
	for _, clause := range []string{
		"admin_frontend_digest TEXT NOT NULL",
		"api_version INTEGER NOT NULL",
		"component_ids JSONB NOT NULL",
		"granted_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL",
		"extension_id, extension_version, api_version, admin_frontend_digest",
	} {
		if !strings.Contains(sql, clause) {
			t.Fatalf("admin frontend trust migration missing %q", clause)
		}
	}
	if strings.Contains(sql, "web_releases") {
		t.Fatal("admin frontend trust migration must not recreate Web Release tables")
	}
}

func TestEmbeddedSQLMigrationsParseWithGoose(t *testing.T) {
	db := openNoopMigrationDB(t)
	defer db.Close()

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		Files(),
		goose.WithDisableGlobalRegistry(true),
		goose.WithDisableVersioning(true),
	)
	if err != nil {
		t.Fatalf("create goose provider: %v", err)
	}

	if _, err := provider.Up(context.Background()); err != nil {
		t.Fatalf("parse embedded migrations with goose: %v", err)
	}
}

const noopMigrationDriverName = "sforum_migration_parse_noop"

var registerNoopMigrationDriver sync.Once

func openNoopMigrationDB(t *testing.T) *sql.DB {
	t.Helper()

	registerNoopMigrationDriver.Do(func() {
		sql.Register(noopMigrationDriverName, noopMigrationDriver{})
	})

	db, err := sql.Open(noopMigrationDriverName, "")
	if err != nil {
		t.Fatalf("open noop migration db: %v", err)
	}
	return db
}

type noopMigrationDriver struct{}

func (noopMigrationDriver) Open(string) (driver.Conn, error) {
	return noopMigrationConn{}, nil
}

type noopMigrationConn struct{}

func (noopMigrationConn) Prepare(string) (driver.Stmt, error) {
	return noopMigrationStmt{}, nil
}

func (noopMigrationConn) Close() error {
	return nil
}

func (noopMigrationConn) Begin() (driver.Tx, error) {
	return noopMigrationTx{}, nil
}

func (noopMigrationConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return noopMigrationTx{}, nil
}

func (noopMigrationConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return noopMigrationResult(0), nil
}

type noopMigrationStmt struct{}

func (noopMigrationStmt) Close() error {
	return nil
}

func (noopMigrationStmt) NumInput() int {
	return -1
}

func (noopMigrationStmt) Exec([]driver.Value) (driver.Result, error) {
	return noopMigrationResult(0), nil
}

func (noopMigrationStmt) Query([]driver.Value) (driver.Rows, error) {
	return noopMigrationRows{}, nil
}

type noopMigrationTx struct{}

func (noopMigrationTx) Commit() error {
	return nil
}

func (noopMigrationTx) Rollback() error {
	return nil
}

type noopMigrationResult int64

func (r noopMigrationResult) LastInsertId() (int64, error) {
	return int64(r), nil
}

func (r noopMigrationResult) RowsAffected() (int64, error) {
	return int64(r), nil
}

type noopMigrationRows struct{}

func (noopMigrationRows) Columns() []string {
	return nil
}

func (noopMigrationRows) Close() error {
	return nil
}

func (noopMigrationRows) Next([]driver.Value) error {
	return io.EOF
}
