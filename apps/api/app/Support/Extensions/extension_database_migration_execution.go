package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type extensionDatabaseExecutableStep struct {
	Record extensionDatabaseMigrationStepRecord
	Plan   extensionDatabaseMigrationStepPlan
}

type extensionDatabaseMigrationExecutionOutcome struct {
	FailingStepID    int64
	FailureCode      string
	SourceResumeSafe bool
	Indeterminate    bool
	Err              error
}

func acquireExtensionDatabaseSessionLock(
	ctx context.Context,
	pool *pgxpool.Pool,
	lockKey int64,
) (*pgxpool.Conn, error) {
	for {
		connection, err := pool.Acquire(ctx)
		if err != nil {
			return nil, err
		}
		var acquired bool
		err = connection.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, lockKey).Scan(&acquired)
		if err != nil {
			connection.Release()
			return nil, err
		}
		if acquired {
			return connection, nil
		}
		connection.Release()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func releaseExtensionDatabaseSessionLock(connection *pgxpool.Conn, lockKey int64) {
	if connection == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	var unlocked bool
	err := connection.QueryRow(ctx, `SELECT pg_advisory_unlock($1)`, lockKey).Scan(&unlocked)
	cancel()
	if err == nil && unlocked {
		connection.Release()
		return
	}
	// A session lock must never return to the pool unless PostgreSQL confirmed
	// its release. Hijacking also keeps pgxpool from recycling an uncertain or
	// broken connection after timeout/cancellation.
	raw := connection.Hijack()
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer closeCancel()
	_ = raw.Close(closeCtx)
}

func prepareExtensionDatabaseMigrationCredential(
	ctx context.Context,
	connection *pgxpool.Conn,
	roleName string,
	identifiers ExtensionDatabaseIdentifiers,
	databaseName string,
	password string,
) error {
	tx, err := connection.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := prepareExtensionDatabaseMigrationRole(
		ctx, tx, roleName, identifiers.OwnerRole, identifiers.Schema, databaseName, password,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func retireExtensionDatabaseMigrationCredential(
	ctx context.Context,
	connection *pgxpool.Conn,
	roleName string,
	identifiers ExtensionDatabaseIdentifiers,
	databaseName string,
) error {
	compensationCtx, cancel := lifecycleBoundaryCompensationContext(ctx)
	defer cancel()
	tx, err := connection.Begin(compensationCtx)
	if err != nil {
		return err
	}
	defer tx.Rollback(compensationCtx)
	if err := revokeExtensionDatabaseMigrationRole(
		compensationCtx, tx, roleName, identifiers.OwnerRole, identifiers.Schema, databaseName,
	); err != nil {
		return err
	}
	return tx.Commit(compensationCtx)
}

func connectExtensionDatabaseMigrationRole(
	ctx context.Context,
	pool *pgxpool.Pool,
	roleName string,
	password string,
	databaseName string,
) (*pgx.Conn, error) {
	config := pool.Config().ConnConfig.Copy()
	config.User = roleName
	config.Password = password
	config.Database = databaseName
	delete(config.RuntimeParams, "search_path")
	delete(config.RuntimeParams, "role")
	return pgx.ConnectConfig(ctx, config)
}

func extensionDatabaseExecutableStepsTransactional(steps []extensionDatabaseExecutableStep) bool {
	for _, step := range steps {
		if !step.Plan.Transactional {
			return false
		}
	}
	return true
}

func executeExtensionDatabaseMigrationTransaction(
	ctx context.Context,
	connection *pgx.Conn,
	identifiers ExtensionDatabaseIdentifiers,
	steps []extensionDatabaseExecutableStep,
) extensionDatabaseMigrationExecutionOutcome {
	outcome := extensionDatabaseMigrationExecutionOutcome{
		FailureCode: extensionDatabaseMigrationFailureExecution, SourceResumeSafe: true,
	}
	if len(steps) == 0 || connection == nil {
		outcome.Err = ErrExtensionDatabaseMigrationInvalid
		return outcome
	}
	if err := scopeExtensionDatabaseMigrationConnection(ctx, connection, identifiers); err != nil {
		outcome.FailingStepID = steps[0].Record.ID
		outcome.Err = err
		return outcome
	}
	tx, err := connection.Begin(ctx)
	if err != nil {
		outcome.FailingStepID = steps[0].Record.ID
		outcome.Err = err
		return outcome
	}
	for _, step := range steps {
		for _, statement := range step.Plan.Statements {
			if _, err := tx.Exec(ctx, statement, pgx.QueryExecModeSimpleProtocol); err != nil {
				outcome.FailingStepID = step.Record.ID
				rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				rollbackErr := tx.Rollback(rollbackCtx)
				cancel()
				outcome.Err = errors.Join(err, rollbackErr)
				outcome.SourceResumeSafe = rollbackErr == nil
				outcome.Indeterminate = rollbackErr != nil
				if outcome.Indeterminate {
					outcome.FailureCode = extensionDatabaseMigrationFailureUnknown
				}
				return outcome
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		outcome.FailingStepID = steps[len(steps)-1].Record.ID
		outcome.FailureCode = extensionDatabaseMigrationFailureUnknown
		outcome.SourceResumeSafe = false
		outcome.Indeterminate = true
		outcome.Err = err
		return outcome
	}
	return extensionDatabaseMigrationExecutionOutcome{}
}

func executeExtensionDatabaseMigrationStep(
	ctx context.Context,
	connection *pgx.Conn,
	identifiers ExtensionDatabaseIdentifiers,
	step extensionDatabaseExecutableStep,
) extensionDatabaseMigrationExecutionOutcome {
	if step.Plan.Transactional {
		return executeExtensionDatabaseMigrationTransaction(
			ctx, connection, identifiers, []extensionDatabaseExecutableStep{step},
		)
	}
	outcome := extensionDatabaseMigrationExecutionOutcome{
		FailingStepID: step.Record.ID, FailureCode: extensionDatabaseMigrationFailureExecution,
		SourceResumeSafe: false,
	}
	if connection == nil {
		outcome.Err = ErrExtensionDatabaseMigrationInvalid
		return outcome
	}
	if err := scopeExtensionDatabaseMigrationConnection(ctx, connection, identifiers); err != nil {
		outcome.SourceResumeSafe = true
		outcome.Err = err
		return outcome
	}
	for _, statement := range step.Plan.Statements {
		if _, err := connection.Exec(ctx, statement, pgx.QueryExecModeSimpleProtocol); err != nil {
			outcome.Err = err
			outcome.Indeterminate = extensionDatabaseMigrationOutcomeUnknown(connection, err)
			if outcome.Indeterminate {
				outcome.FailureCode = extensionDatabaseMigrationFailureUnknown
			}
			return outcome
		}
	}
	return extensionDatabaseMigrationExecutionOutcome{}
}

func scopeExtensionDatabaseMigrationConnection(
	ctx context.Context,
	connection *pgx.Conn,
	identifiers ExtensionDatabaseIdentifiers,
) error {
	if connection == nil || !identifiers.Valid() {
		return ErrExtensionDatabaseMigrationInvalid
	}
	owner := pgx.Identifier{identifiers.OwnerRole}.Sanitize()
	schema := pgx.Identifier{identifiers.Schema}.Sanitize()
	for _, query := range []string{
		`RESET ROLE`,
		`SET ROLE ` + owner,
		`SET search_path TO ` + schema + `, pg_catalog`,
	} {
		if _, err := connection.Exec(ctx, query); err != nil {
			return fmt.Errorf("scope extension database migration connection: %w", err)
		}
	}
	return nil
}

func extensionDatabaseMigrationOutcomeUnknown(connection *pgx.Conn, err error) bool {
	if err == nil || pgconn.SafeToRetry(err) {
		return false
	}
	return connection == nil || connection.IsClosed() ||
		errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
