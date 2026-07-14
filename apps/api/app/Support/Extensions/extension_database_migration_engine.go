package extensionsruntime

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	extensionDatabaseMigrationFailureExecution  = "migration.execution_failed"
	extensionDatabaseMigrationFailureUnknown    = "migration.outcome_unknown"
	extensionDatabaseMigrationFailureStateDrift = "migration.state_drift"
	extensionDatabaseMigrationFailureRecovery   = "migration.recovery_required"
)

var ErrExtensionDatabaseMigrationRecoveryRequired = errors.New("extension database migration requires explicit recovery")

type PostgresLifecycleMigrationEngine struct {
	pool   *pgxpool.Pool
	random io.Reader
}

func NewPostgresLifecycleMigrationEngine(
	pool *pgxpool.Pool,
	random io.Reader,
) *PostgresLifecycleMigrationEngine {
	if random == nil {
		random = rand.Reader
	}
	return &PostgresLifecycleMigrationEngine{pool: pool, random: random}
}

func (e *PostgresLifecycleMigrationEngine) ReconcileLifecycleMigration(
	ctx context.Context,
	plan LifecycleMigrationEnginePlan,
) error {
	if e == nil || e.pool == nil || e.random == nil || ctx == nil {
		return ErrExtensionDatabaseMigrationInvalid
	}
	if err := validateExtensionDatabaseMigrationEnginePlan(plan); err != nil {
		return err
	}
	identifiers, err := ExtensionDatabaseIdentifiersFor(plan.Target.ExtensionID)
	if err != nil {
		return ErrExtensionDatabaseMigrationInvalid
	}
	connection, err := acquireExtensionDatabaseMigrationLock(ctx, e.pool, identifiers.LockKey)
	if err != nil {
		return newExtensionDatabaseMigrationFailure(extensionDatabaseMigrationFailureRecovery, err)
	}
	defer releaseExtensionDatabaseMigrationLock(connection, identifiers.LockKey)

	if proof, proofErr := loadExtensionDatabaseMigrationProof(ctx, connection, plan.PlanDigest); proofErr == nil {
		record, recordErr := loadExtensionDatabaseMigrationPlan(ctx, connection, plan.PlanDigest, false)
		if recordErr != nil || !record.matchesEnginePlan(plan) || proof.PlanDigest != plan.PlanDigest {
			return ErrExtensionDatabaseResourceConflict
		}
		if proof.TargetReady {
			return nil
		}
		if !proof.SourceResumeSafe {
			return newExtensionDatabaseMigrationFailure(
				extensionDatabaseMigrationFailureRecovery, ErrExtensionDatabaseMigrationRecoveryRequired,
			)
		}
	} else if !errors.Is(proofErr, ErrExtensionDatabaseGrantNotFound) {
		return proofErr
	}

	dryRun, err := discoverExtensionDatabaseMigrationPlan(ctx, connection, plan)
	if err != nil {
		if persistErr := e.persistDiscoveryFailure(ctx, connection, plan, identifiers, err); persistErr != nil {
			return errors.Join(err, persistErr)
		}
		return err
	}
	return e.reconcileDiscoveredMigration(ctx, connection, dryRun)
}

func (e *PostgresLifecycleMigrationEngine) InspectLifecycleMigration(
	ctx context.Context,
	plan LifecycleMigrationEnginePlan,
) (LifecycleMigrationEngineProof, error) {
	if e == nil || e.pool == nil || ctx == nil {
		return LifecycleMigrationEngineProof{}, ErrExtensionDatabaseMigrationInvalid
	}
	if err := validateExtensionDatabaseMigrationEnginePlan(plan); err != nil {
		return LifecycleMigrationEngineProof{}, err
	}
	record, err := loadExtensionDatabaseMigrationPlan(ctx, e.pool, plan.PlanDigest, false)
	if err != nil {
		return LifecycleMigrationEngineProof{}, err
	}
	if !record.matchesEnginePlan(plan) {
		return LifecycleMigrationEngineProof{}, ErrExtensionDatabaseResourceConflict
	}
	proof, err := loadExtensionDatabaseMigrationProof(ctx, e.pool, plan.PlanDigest)
	if err != nil {
		return LifecycleMigrationEngineProof{}, err
	}
	if proof.PlanDigest != plan.PlanDigest || !validLifecycleMigrationProofID(proof.ProofID) ||
		!validLifecycleCleanupDigest(proof.ProofDigest) {
		return LifecycleMigrationEngineProof{}, ErrLifecycleMigrationEngineProofBad
	}
	return proof, nil
}

func (e *PostgresLifecycleMigrationEngine) persistDiscoveryFailure(
	ctx context.Context,
	connection *pgxpool.Conn,
	plan LifecycleMigrationEnginePlan,
	identifiers ExtensionDatabaseIdentifiers,
	discoveryErr error,
) error {
	failureCode := extensionDatabaseMigrationFailureCode(discoveryErr)
	digestMaterial := sha256.Sum256([]byte(plan.PlanDigest + "\x00" + failureCode))
	dryRun := extensionDatabaseMigrationDryRun{
		EnginePlan: plan, Identifiers: identifiers,
		Target: extensionDatabaseExactMigrationArtifact{LifecycleMigrationArtifact: plan.Target},
		Digest: hex.EncodeToString(digestMaterial[:]),
	}
	if plan.Source != nil {
		dryRun.Source = &extensionDatabaseExactMigrationArtifact{LifecycleMigrationArtifact: *plan.Source}
	}
	tx, err := connection.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := ensureExtensionDatabaseMigrationFailureResource(
		ctx, tx, plan.Target.ExtensionID, identifiers, failureCode,
	); err != nil {
		return err
	}
	record, _, err := ensureExtensionDatabaseMigrationPlan(ctx, tx, dryRun)
	if err != nil {
		return err
	}
	if record.Status == "failed" || record.Status == "indeterminate" || record.Status == "succeeded" {
		return tx.Commit(ctx)
	}
	if _, err := finalizeExtensionDatabaseMigrationPlan(
		ctx, tx, record, "failed", failureCode, true,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (e *PostgresLifecycleMigrationEngine) reconcileDiscoveredMigration(
	ctx context.Context,
	connection *pgxpool.Conn,
	dryRun extensionDatabaseMigrationDryRun,
) error {
	tx, err := connection.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	databaseName, err := ensureExtensionDatabaseResources(
		ctx, tx, dryRun.Target.ExtensionID, dryRun.Identifiers,
	)
	if err != nil {
		return err
	}
	if err := markExtensionDatabaseResourceProvisioned(ctx, tx, dryRun.Target.ExtensionID); err != nil {
		return err
	}
	record, stepRecords, err := ensureExtensionDatabaseMigrationPlan(ctx, tx, dryRun)
	if err != nil {
		return err
	}
	switch record.Status {
	case "succeeded":
		return tx.Commit(ctx)
	case "indeterminate":
		return newExtensionDatabaseMigrationFailure(
			extensionDatabaseMigrationFailureRecovery, ErrExtensionDatabaseMigrationRecoveryRequired,
		)
	case "running":
		if _, err := finalizeExtensionDatabaseMigrationPlan(
			ctx, tx, record, "indeterminate", extensionDatabaseMigrationFailureUnknown, false,
		); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return newExtensionDatabaseMigrationFailure(
			extensionDatabaseMigrationFailureUnknown, ErrExtensionDatabaseMigrationRecoveryRequired,
		)
	case "failed":
		if !record.SourceResumeSafe {
			return newExtensionDatabaseMigrationFailure(
				extensionDatabaseMigrationFailureRecovery, ErrExtensionDatabaseMigrationRecoveryRequired,
			)
		}
		record, err = resetExtensionDatabaseMigrationPlanForRetry(ctx, tx, record)
		if err != nil {
			return err
		}
		stepRecords, err = loadExtensionDatabaseMigrationSteps(ctx, tx, record.ID, true)
		if err != nil {
			return err
		}
	case "planned":
	default:
		return ErrExtensionDatabaseResourceConflict
	}

	stepPlans := make(map[int]extensionDatabaseMigrationStepPlan, len(dryRun.Steps))
	for _, step := range dryRun.Steps {
		stepPlans[step.Position] = step
	}
	executable := make([]extensionDatabaseExecutableStep, 0, len(stepRecords))
	for _, stepRecord := range stepRecords {
		stepPlan := stepPlans[stepRecord.Position]
		state, exists, stateErr := loadExtensionDatabaseMigrationState(
			ctx, tx, dryRun.Target.ExtensionID, stepPlan.Declaration.ID,
		)
		if stateErr != nil {
			return stateErr
		}
		shouldSkip := false
		stateDrift := false
		if stepPlan.Direction == "up" {
			shouldSkip = exists && state.matches(stepPlan)
			stateDrift = exists && !state.matches(stepPlan)
		} else {
			shouldSkip = !exists
			stateDrift = exists && !state.matches(stepPlan)
		}
		if stateDrift {
			if err := markExtensionDatabaseMigrationExecutionFailure(
				ctx, tx, record.ID, stepRecord.ID,
				extensionDatabaseMigrationFailureStateDrift, false, false,
			); err != nil {
				return err
			}
			if _, err := finalizeExtensionDatabaseMigrationPlan(
				ctx, tx, record, "failed", extensionDatabaseMigrationFailureStateDrift, true,
			); err != nil {
				return err
			}
			if err := tx.Commit(ctx); err != nil {
				return err
			}
			return newExtensionDatabaseMigrationFailure(
				extensionDatabaseMigrationFailureStateDrift, ErrExtensionDatabaseMigrationChecksumDrift,
			)
		}
		if shouldSkip {
			if stepRecord.Status == "pending" {
				if err := markExtensionDatabaseMigrationStepSkipped(ctx, tx, stepRecord); err != nil {
					return err
				}
			}
			continue
		}
		if stepRecord.Status != "pending" {
			return ErrExtensionDatabaseResourceConflict
		}
		executable = append(executable, extensionDatabaseExecutableStep{Record: stepRecord, Plan: stepPlan})
	}
	record, err = markExtensionDatabaseMigrationPlanRunning(ctx, tx, record)
	if err != nil {
		return err
	}
	allTransactional := extensionDatabaseExecutableStepsTransactional(executable)
	if allTransactional && len(executable) > 0 {
		records := make([]extensionDatabaseMigrationStepRecord, 0, len(executable))
		for _, step := range executable {
			records = append(records, step.Record)
		}
		if err := markExtensionDatabaseMigrationStepsRunning(ctx, tx, records); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if len(executable) == 0 {
		return e.finalizeSuccessfulMigration(ctx, connection, record, dryRun.Target, true)
	}

	roleName, err := ExtensionDatabaseMigrationRoleFor(dryRun.Target.ExtensionID, dryRun.EnginePlan.PlanDigest)
	if err != nil {
		return e.finalizeMigrationFailure(
			ctx, connection, record, 0, extensionDatabaseMigrationFailureCredential,
			true, false, false, err,
		)
	}
	password, err := e.newMigrationCredentialSecret()
	if err != nil {
		return e.finalizeMigrationFailure(
			ctx, connection, record, 0, extensionDatabaseMigrationFailureCredential,
			true, false, false, err,
		)
	}
	if err := prepareExtensionDatabaseMigrationCredential(
		ctx, connection, roleName, dryRun.Identifiers, databaseName, password,
	); err != nil {
		return e.finalizeMigrationFailure(
			ctx, connection, record, 0, extensionDatabaseMigrationFailureCredential,
			true, false, false, err,
		)
	}

	scoped, err := connectExtensionDatabaseMigrationRole(ctx, e.pool, roleName, password, databaseName)
	if err != nil {
		retireErr := retireExtensionDatabaseMigrationCredential(
			ctx, connection, roleName, dryRun.Identifiers, databaseName,
		)
		return e.finalizeMigrationFailure(
			ctx, connection, record, 0, extensionDatabaseMigrationFailureCredential,
			true, false, false, errors.Join(err, retireErr),
		)
	}

	effectsCommitted := false
	if allTransactional {
		outcome := executeExtensionDatabaseMigrationTransaction(ctx, scoped, dryRun.Identifiers, executable)
		_ = scoped.Close(context.Background())
		retireErr := retireExtensionDatabaseMigrationCredential(
			ctx, connection, roleName, dryRun.Identifiers, databaseName,
		)
		if outcome.Err != nil {
			return e.finalizeMigrationFailure(
				ctx, connection, record, outcome.FailingStepID, outcome.FailureCode,
				outcome.SourceResumeSafe, outcome.Indeterminate, true,
				errors.Join(outcome.Err, retireErr),
			)
		}
		if err := e.persistAppliedMigrationSteps(ctx, connection, record, executable); err != nil {
			return e.finalizeMigrationFailure(
				ctx, connection, record, 0, extensionDatabaseMigrationFailureUnknown,
				false, true, false, errors.Join(err, retireErr),
			)
		}
		effectsCommitted = true
		if retireErr != nil {
			return e.finalizeMigrationFailure(
				ctx, connection, record, 0, extensionDatabaseMigrationFailureCredential,
				false, true, false, retireErr,
			)
		}
	} else {
		for _, step := range executable {
			if err := e.markMigrationStepRunning(ctx, connection, step.Record); err != nil {
				_ = scoped.Close(context.Background())
				_ = retireExtensionDatabaseMigrationCredential(
					ctx, connection, roleName, dryRun.Identifiers, databaseName,
				)
				return err
			}
			outcome := executeExtensionDatabaseMigrationStep(ctx, scoped, dryRun.Identifiers, step)
			if outcome.Err != nil {
				_ = scoped.Close(context.Background())
				retireErr := retireExtensionDatabaseMigrationCredential(
					ctx, connection, roleName, dryRun.Identifiers, databaseName,
				)
				sourceSafe := !effectsCommitted && outcome.SourceResumeSafe
				return e.finalizeMigrationFailure(
					ctx, connection, record, step.Record.ID, outcome.FailureCode,
					sourceSafe, outcome.Indeterminate, false,
					errors.Join(outcome.Err, retireErr),
				)
			}
			if err := e.persistAppliedMigrationSteps(
				ctx, connection, record, []extensionDatabaseExecutableStep{step},
			); err != nil {
				_ = scoped.Close(context.Background())
				retireErr := retireExtensionDatabaseMigrationCredential(
					ctx, connection, roleName, dryRun.Identifiers, databaseName,
				)
				return e.finalizeMigrationFailure(
					ctx, connection, record, step.Record.ID, extensionDatabaseMigrationFailureUnknown,
					false, true, false, errors.Join(err, retireErr),
				)
			}
			effectsCommitted = true
		}
		_ = scoped.Close(context.Background())
		if err := retireExtensionDatabaseMigrationCredential(
			ctx, connection, roleName, dryRun.Identifiers, databaseName,
		); err != nil {
			return e.finalizeMigrationFailure(
				ctx, connection, record, 0, extensionDatabaseMigrationFailureCredential,
				false, true, false, err,
			)
		}
	}
	return e.finalizeSuccessfulMigration(ctx, connection, record, dryRun.Target, !effectsCommitted)
}

func (e *PostgresLifecycleMigrationEngine) persistAppliedMigrationSteps(
	ctx context.Context,
	connection *pgxpool.Conn,
	record extensionDatabaseMigrationPlanRecord,
	steps []extensionDatabaseExecutableStep,
) error {
	tx, err := connection.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	records := make([]extensionDatabaseMigrationStepRecord, 0, len(steps))
	plans := make(map[int]extensionDatabaseMigrationStepPlan, len(steps))
	for _, step := range steps {
		records = append(records, step.Record)
		plans[step.Record.Position] = step.Plan
	}
	if err := persistAppliedExtensionDatabaseMigrationSteps(ctx, tx, record, records, plans); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (e *PostgresLifecycleMigrationEngine) markMigrationStepRunning(
	ctx context.Context,
	connection *pgxpool.Conn,
	step extensionDatabaseMigrationStepRecord,
) error {
	tx, err := connection.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := markExtensionDatabaseMigrationStepsRunning(ctx, tx, []extensionDatabaseMigrationStepRecord{step}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (e *PostgresLifecycleMigrationEngine) finalizeSuccessfulMigration(
	ctx context.Context,
	connection *pgxpool.Conn,
	record extensionDatabaseMigrationPlanRecord,
	target extensionDatabaseExactMigrationArtifact,
	sourceResumeSafe bool,
) error {
	tx, err := connection.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := assertExtensionDatabaseMigrationTargetState(ctx, tx, target); err != nil {
		if _, proofErr := finalizeExtensionDatabaseMigrationPlan(
			ctx, tx, record, "failed", extensionDatabaseMigrationFailureStateDrift, false,
		); proofErr != nil {
			return errors.Join(err, proofErr)
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return errors.Join(err, commitErr)
		}
		return newExtensionDatabaseMigrationFailure(extensionDatabaseMigrationFailureStateDrift, err)
	}
	if _, err := finalizeExtensionDatabaseMigrationPlan(
		ctx, tx, record, "succeeded", "", sourceResumeSafe,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (e *PostgresLifecycleMigrationEngine) finalizeMigrationFailure(
	ctx context.Context,
	connection *pgxpool.Conn,
	record extensionDatabaseMigrationPlanRecord,
	failingStepID int64,
	failureCode string,
	sourceResumeSafe bool,
	indeterminate bool,
	resetOtherRunning bool,
	cause error,
) error {
	compensationCtx, cancel := lifecycleBoundaryCompensationContext(ctx)
	defer cancel()
	tx, err := connection.Begin(compensationCtx)
	if err != nil {
		return errors.Join(cause, err)
	}
	defer tx.Rollback(compensationCtx)
	if failingStepID > 0 {
		if err := markExtensionDatabaseMigrationExecutionFailure(
			compensationCtx, tx, record.ID, failingStepID, failureCode,
			indeterminate, resetOtherRunning,
		); err != nil {
			return errors.Join(cause, err)
		}
	}
	status := "failed"
	if indeterminate {
		status = "indeterminate"
	}
	if _, err := finalizeExtensionDatabaseMigrationPlan(
		compensationCtx, tx, record, status, failureCode, sourceResumeSafe,
	); err != nil {
		return errors.Join(cause, err)
	}
	if err := tx.Commit(compensationCtx); err != nil {
		return errors.Join(cause, err)
	}
	if cause == nil {
		cause = ErrExtensionDatabaseMigrationRecoveryRequired
	}
	return newExtensionDatabaseMigrationFailure(failureCode, cause)
}

func (e *PostgresLifecycleMigrationEngine) newMigrationCredentialSecret() (string, error) {
	registry := NewPostgresExtensionDatabaseRegistry(e.pool, e.random)
	password, _, err := registry.newCredentialSecret()
	return password, err
}

var _ LifecycleMigrationEngine = (*PostgresLifecycleMigrationEngine)(nil)
