package extensionsruntime

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"testing/fstest"

	"github.com/pressly/goose/v3"
)

const (
	extensionDatabaseMigrationWarningNonTransactional = "migration.non_transactional"
	extensionDatabaseMigrationFailureParse            = "migration.parse_failed"
	extensionDatabaseMigrationFailurePolicy           = "migration.transaction_policy"
)

var (
	ErrExtensionDatabaseMigrationParse  = errors.New("extension database migration parse failed")
	ErrExtensionDatabaseMigrationPolicy = errors.New("extension database migration transaction policy mismatch")
)

type extensionDatabaseParsedMigration struct {
	Statements    []string
	Transactional bool
	WarningCode   string
}

func parseExtensionDatabaseMigration(
	ctx context.Context,
	body []byte,
	direction string,
	policy string,
) (extensionDatabaseParsedMigration, error) {
	if ctx == nil || len(body) == 0 || (direction != "up" && direction != "down") {
		return extensionDatabaseParsedMigration{}, ErrExtensionDatabaseMigrationParse
	}
	if policy != "required" && policy != "forbidden" && policy != "auto" {
		return extensionDatabaseParsedMigration{}, ErrExtensionDatabaseMigrationPolicy
	}
	prepared, hasNoTransaction := prepareExtensionDatabaseGooseSQL(body, policy)
	if policy == "required" && hasNoTransaction {
		return extensionDatabaseParsedMigration{}, fmt.Errorf(
			"%w: required migration declares NO TRANSACTION", ErrExtensionDatabaseMigrationPolicy,
		)
	}

	recorder := &extensionDatabaseGooseRecorder{}
	database := sql.OpenDB(extensionDatabaseGooseConnector{recorder: recorder})
	defer database.Close()
	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		database,
		fstest.MapFS{"00001_extension.sql": &fstest.MapFile{Data: prepared, Mode: fs.FileMode(0o600)}},
		goose.WithDisableGlobalRegistry(true),
		goose.WithDisableVersioning(true),
	)
	if err != nil {
		return extensionDatabaseParsedMigration{}, fmt.Errorf("%w: create parser: %v", ErrExtensionDatabaseMigrationParse, err)
	}
	up := direction == "up"
	if _, err := provider.ApplyVersion(ctx, 1, up); err != nil {
		return extensionDatabaseParsedMigration{}, fmt.Errorf("%w: %v", ErrExtensionDatabaseMigrationParse, err)
	}
	statements, transactional := recorder.snapshot()
	if len(statements) == 0 {
		return extensionDatabaseParsedMigration{}, fmt.Errorf(
			"%w: %s section has no statements", ErrExtensionDatabaseMigrationParse, direction,
		)
	}
	if policy == "required" && !transactional {
		return extensionDatabaseParsedMigration{}, ErrExtensionDatabaseMigrationPolicy
	}
	if policy == "forbidden" && transactional {
		return extensionDatabaseParsedMigration{}, ErrExtensionDatabaseMigrationPolicy
	}
	warning := ""
	if !transactional {
		warning = extensionDatabaseMigrationWarningNonTransactional
	}
	return extensionDatabaseParsedMigration{
		Statements: statements, Transactional: transactional, WarningCode: warning,
	}, nil
}

func prepareExtensionDatabaseGooseSQL(body []byte, policy string) ([]byte, bool) {
	text := string(body)
	hasUp := hasExtensionDatabaseGooseDirective(text, "Up")
	hasNoTransaction := hasExtensionDatabaseGooseDirective(text, "NO TRANSACTION")
	prefix := make([]string, 0, 2)
	if policy == "forbidden" && !hasNoTransaction {
		prefix = append(prefix, "-- +goose NO TRANSACTION")
		hasNoTransaction = true
	}
	if !hasUp {
		prefix = append(prefix, "-- +goose Up")
	}
	if len(prefix) == 0 {
		return append([]byte(nil), body...), hasNoTransaction
	}
	return []byte(strings.Join(prefix, "\n") + "\n" + text), hasNoTransaction
}

func hasExtensionDatabaseGooseDirective(body string, directive string) bool {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if strings.EqualFold(line, "-- +goose "+directive) {
			return true
		}
	}
	return false
}

type extensionDatabaseGooseConnector struct {
	recorder *extensionDatabaseGooseRecorder
}

func (c extensionDatabaseGooseConnector) Connect(context.Context) (driver.Conn, error) {
	return &extensionDatabaseGooseConnection{recorder: c.recorder}, nil
}

func (c extensionDatabaseGooseConnector) Driver() driver.Driver {
	return extensionDatabaseGooseDriver{recorder: c.recorder}
}

type extensionDatabaseGooseDriver struct {
	recorder *extensionDatabaseGooseRecorder
}

func (d extensionDatabaseGooseDriver) Open(string) (driver.Conn, error) {
	return &extensionDatabaseGooseConnection{recorder: d.recorder}, nil
}

type extensionDatabaseGooseConnection struct {
	recorder *extensionDatabaseGooseRecorder
}

func (c *extensionDatabaseGooseConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("extension migration dry-run does not prepare statements")
}

func (c *extensionDatabaseGooseConnection) Close() error { return nil }

func (c *extensionDatabaseGooseConnection) Begin() (driver.Tx, error) {
	c.recorder.begin()
	return extensionDatabaseGooseTransaction{}, nil
}

func (c *extensionDatabaseGooseConnection) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	c.recorder.begin()
	return extensionDatabaseGooseTransaction{}, nil
}

func (c *extensionDatabaseGooseConnection) ExecContext(
	_ context.Context,
	query string,
	_ []driver.NamedValue,
) (driver.Result, error) {
	c.recorder.record(query)
	return driver.RowsAffected(0), nil
}

type extensionDatabaseGooseTransaction struct{}

func (extensionDatabaseGooseTransaction) Commit() error   { return nil }
func (extensionDatabaseGooseTransaction) Rollback() error { return nil }

type extensionDatabaseGooseRecorder struct {
	mu            sync.Mutex
	transactional bool
	statements    []string
}

func (r *extensionDatabaseGooseRecorder) begin() {
	r.mu.Lock()
	r.transactional = true
	r.mu.Unlock()
}

func (r *extensionDatabaseGooseRecorder) record(query string) {
	r.mu.Lock()
	r.statements = append(r.statements, query)
	r.mu.Unlock()
}

func (r *extensionDatabaseGooseRecorder) snapshot() ([]string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.statements...), r.transactional
}
