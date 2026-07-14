package extensionsruntime

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type extensionDatabaseMigrationPlanRecord struct {
	ID                     int64
	OperationID            int64
	Operation              string
	Mode                   string
	StepID                 string
	Attempt                int
	PlanDigest             string
	ExtensionID            string
	SourceVersionID        sql.NullInt64
	SourceVersion          sql.NullString
	SourcePackageDigest    sql.NullString
	SourceMigrationsDigest sql.NullString
	TargetVersionID        int64
	TargetVersion          string
	TargetPackageDigest    string
	TargetMigrationsDigest string
	SchemaName             string
	OwnerRoleName          string
	DryRunDigest           string
	Status                 string
	CurrentStep            int
	TotalSteps             int
	FailureCode            string
	WarningCode            string
	HasNonTransactional    bool
	TargetReady            bool
	SourceResumeSafe       bool
	Revision               int64
}

type extensionDatabaseMigrationStepRecord struct {
	ID                int64
	PlanID            int64
	Position          int
	Direction         string
	MigrationID       string
	ContractVersion   string
	PackagePath       string
	Checksum          string
	TransactionPolicy string
	ExecutionMode     string
	Status            string
	FailureCode       string
	WarningCode       string
	ResultDigest      *string
}

type extensionDatabaseMigrationStateRecord struct {
	MigrationID       string
	ContractVersion   string
	PackagePath       string
	Checksum          string
	TransactionPolicy string
}

func ensureExtensionDatabaseMigrationPlan(
	ctx context.Context,
	tx pgx.Tx,
	dryRun extensionDatabaseMigrationDryRun,
) (extensionDatabaseMigrationPlanRecord, []extensionDatabaseMigrationStepRecord, error) {
	var sourceVersionID, sourceVersion, sourcePackageDigest, sourceMigrationsDigest any
	if dryRun.Source != nil {
		sourceVersionID = dryRun.Source.VersionID
		sourceVersion = dryRun.Source.Version
		sourcePackageDigest = dryRun.Source.PackageDigest
		sourceMigrationsDigest = dryRun.Source.MigrationsDigest
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO extension_database_migration_plans (
			operation_id, operation, migration_mode, step_id, attempt, plan_digest,
			extension_id, source_extension_version_id, source_extension_version,
			source_package_digest, source_migrations_digest,
			target_extension_version_id, target_extension_version,
			target_package_digest, target_migrations_digest,
			schema_name, owner_role_name, dry_run_digest,
			total_steps, warning_code, has_non_transactional
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21
		)
		ON CONFLICT (plan_digest) DO NOTHING
	`, dryRun.EnginePlan.OperationID, dryRun.EnginePlan.Operation, dryRun.EnginePlan.Mode,
		dryRun.EnginePlan.StepID, dryRun.EnginePlan.Attempt, dryRun.EnginePlan.PlanDigest,
		dryRun.Target.ExtensionID, sourceVersionID, sourceVersion, sourcePackageDigest,
		sourceMigrationsDigest, dryRun.Target.VersionID, dryRun.Target.Version,
		dryRun.Target.PackageDigest, dryRun.Target.MigrationsDigest,
		dryRun.Identifiers.Schema, dryRun.Identifiers.OwnerRole, dryRun.Digest,
		len(dryRun.Steps), dryRun.WarningCode, dryRun.HasNonTransactional)
	if err != nil {
		return extensionDatabaseMigrationPlanRecord{}, nil, fmt.Errorf("insert extension database migration plan: %w", err)
	}
	record, err := loadExtensionDatabaseMigrationPlan(ctx, tx, dryRun.EnginePlan.PlanDigest, true)
	if err != nil {
		return extensionDatabaseMigrationPlanRecord{}, nil, err
	}
	if !record.matchesDryRun(dryRun) || dryRun.EnginePlan.Attempt < record.Attempt {
		return extensionDatabaseMigrationPlanRecord{}, nil, ErrExtensionDatabaseResourceConflict
	}
	if dryRun.EnginePlan.Attempt > record.Attempt {
		tag, updateErr := tx.Exec(ctx, `
			UPDATE extension_database_migration_plans
			SET attempt = $2, revision = revision + 1, updated_at = statement_timestamp()
			WHERE id = $1 AND revision = $3
		`, record.ID, dryRun.EnginePlan.Attempt, record.Revision)
		if updateErr != nil {
			return extensionDatabaseMigrationPlanRecord{}, nil, fmt.Errorf("advance extension migration attempt: %w", updateErr)
		}
		if tag.RowsAffected() != 1 {
			return extensionDatabaseMigrationPlanRecord{}, nil, ErrExtensionDatabaseResourceConflict
		}
		record.Attempt = dryRun.EnginePlan.Attempt
		record.Revision++
	}
	for _, step := range dryRun.Steps {
		executionMode := "transactional"
		if !step.Transactional {
			executionMode = "non_transactional"
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO extension_database_migration_steps (
				plan_id, position, direction, migration_id, contract_version,
				package_path, checksum, transaction_policy, execution_mode,
				warning_code
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (plan_id, position) DO NOTHING
		`, record.ID, step.Position, step.Direction, step.Declaration.ID,
			step.Declaration.ContractVersion, step.Declaration.Path, step.Declaration.Digest,
			step.Declaration.Transaction, executionMode, step.WarningCode)
		if err != nil {
			return extensionDatabaseMigrationPlanRecord{}, nil, fmt.Errorf("insert extension database migration step: %w", err)
		}
	}
	steps, err := loadExtensionDatabaseMigrationSteps(ctx, tx, record.ID, true)
	if err != nil {
		return extensionDatabaseMigrationPlanRecord{}, nil, err
	}
	if !extensionDatabaseMigrationStepsMatchDryRun(steps, dryRun.Steps) {
		return extensionDatabaseMigrationPlanRecord{}, nil, ErrExtensionDatabaseResourceConflict
	}
	return record, steps, nil
}

func (r extensionDatabaseMigrationPlanRecord) matchesDryRun(dryRun extensionDatabaseMigrationDryRun) bool {
	if r.OperationID != dryRun.EnginePlan.OperationID || r.Operation != string(dryRun.EnginePlan.Operation) ||
		r.Mode != string(dryRun.EnginePlan.Mode) || r.StepID != dryRun.EnginePlan.StepID ||
		r.PlanDigest != dryRun.EnginePlan.PlanDigest || r.ExtensionID != dryRun.Target.ExtensionID ||
		r.TargetVersionID != dryRun.Target.VersionID || r.TargetVersion != dryRun.Target.Version ||
		r.TargetPackageDigest != dryRun.Target.PackageDigest ||
		r.TargetMigrationsDigest != dryRun.Target.MigrationsDigest ||
		r.SchemaName != dryRun.Identifiers.Schema || r.OwnerRoleName != dryRun.Identifiers.OwnerRole ||
		r.DryRunDigest != dryRun.Digest || r.TotalSteps != len(dryRun.Steps) ||
		r.WarningCode != dryRun.WarningCode || r.HasNonTransactional != dryRun.HasNonTransactional {
		return false
	}
	if dryRun.Source == nil {
		return !r.SourceVersionID.Valid && !r.SourceVersion.Valid &&
			!r.SourcePackageDigest.Valid && !r.SourceMigrationsDigest.Valid
	}
	return r.SourceVersionID.Valid && r.SourceVersionID.Int64 == dryRun.Source.VersionID &&
		r.SourceVersion.Valid && r.SourceVersion.String == dryRun.Source.Version &&
		r.SourcePackageDigest.Valid && r.SourcePackageDigest.String == dryRun.Source.PackageDigest &&
		r.SourceMigrationsDigest.Valid && r.SourceMigrationsDigest.String == dryRun.Source.MigrationsDigest
}

func loadExtensionDatabaseMigrationPlan(
	ctx context.Context,
	querier extensionDatabaseQuerier,
	planDigest string,
	forUpdate bool,
) (extensionDatabaseMigrationPlanRecord, error) {
	query := `
		SELECT id, operation_id, operation, migration_mode, step_id, attempt,
		       plan_digest, extension_id,
		       source_extension_version_id, source_extension_version,
		       source_package_digest, source_migrations_digest,
		       target_extension_version_id, target_extension_version,
		       target_package_digest, target_migrations_digest,
		       schema_name, owner_role_name, dry_run_digest, status,
		       current_step, total_steps, failure_code, warning_code,
		       has_non_transactional, target_ready, source_resume_safe, revision
		FROM extension_database_migration_plans WHERE plan_digest = $1
	`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	var record extensionDatabaseMigrationPlanRecord
	err := querier.QueryRow(ctx, query, planDigest).Scan(
		&record.ID, &record.OperationID, &record.Operation, &record.Mode,
		&record.StepID, &record.Attempt, &record.PlanDigest, &record.ExtensionID,
		&record.SourceVersionID, &record.SourceVersion, &record.SourcePackageDigest,
		&record.SourceMigrationsDigest, &record.TargetVersionID, &record.TargetVersion,
		&record.TargetPackageDigest, &record.TargetMigrationsDigest,
		&record.SchemaName, &record.OwnerRoleName,
		&record.DryRunDigest, &record.Status, &record.CurrentStep, &record.TotalSteps,
		&record.FailureCode, &record.WarningCode, &record.HasNonTransactional,
		&record.TargetReady, &record.SourceResumeSafe, &record.Revision,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return extensionDatabaseMigrationPlanRecord{}, ErrExtensionDatabaseGrantNotFound
	}
	if err != nil {
		return extensionDatabaseMigrationPlanRecord{}, fmt.Errorf("load extension database migration plan: %w", err)
	}
	return record, nil
}

func loadExtensionDatabaseMigrationSteps(
	ctx context.Context,
	querier interface {
		Query(context.Context, string, ...any) (pgx.Rows, error)
	},
	planID int64,
	forUpdate bool,
) ([]extensionDatabaseMigrationStepRecord, error) {
	query := `
		SELECT id, plan_id, position, direction, migration_id, contract_version,
		       package_path, checksum, transaction_policy, execution_mode,
		       status, failure_code, warning_code, result_digest
		FROM extension_database_migration_steps
		WHERE plan_id = $1 ORDER BY position
	`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	rows, err := querier.Query(ctx, query, planID)
	if err != nil {
		return nil, fmt.Errorf("load extension database migration steps: %w", err)
	}
	defer rows.Close()
	steps := make([]extensionDatabaseMigrationStepRecord, 0)
	for rows.Next() {
		var step extensionDatabaseMigrationStepRecord
		if err := rows.Scan(
			&step.ID, &step.PlanID, &step.Position, &step.Direction,
			&step.MigrationID, &step.ContractVersion, &step.PackagePath,
			&step.Checksum, &step.TransactionPolicy, &step.ExecutionMode,
			&step.Status, &step.FailureCode, &step.WarningCode, &step.ResultDigest,
		); err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return steps, nil
}

func extensionDatabaseMigrationStepsMatchDryRun(
	records []extensionDatabaseMigrationStepRecord,
	plans []extensionDatabaseMigrationStepPlan,
) bool {
	if len(records) != len(plans) {
		return false
	}
	for index := range records {
		executionMode := "transactional"
		if !plans[index].Transactional {
			executionMode = "non_transactional"
		}
		record, plan := records[index], plans[index]
		if record.Position != plan.Position || record.Direction != plan.Direction ||
			record.MigrationID != plan.Declaration.ID ||
			record.ContractVersion != plan.Declaration.ContractVersion ||
			record.PackagePath != plan.Declaration.Path || record.Checksum != plan.Declaration.Digest ||
			record.TransactionPolicy != plan.Declaration.Transaction || record.ExecutionMode != executionMode ||
			record.WarningCode != plan.WarningCode {
			return false
		}
	}
	return true
}

func resetExtensionDatabaseMigrationPlanForRetry(
	ctx context.Context,
	tx pgx.Tx,
	record extensionDatabaseMigrationPlanRecord,
) (extensionDatabaseMigrationPlanRecord, error) {
	if record.Status != "failed" || !record.SourceResumeSafe {
		return record, nil
	}
	if _, err := tx.Exec(ctx, `DELETE FROM extension_database_migration_proofs WHERE plan_id = $1`, record.ID); err != nil {
		return record, fmt.Errorf("clear retryable extension migration proof: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE extension_database_migration_steps
		SET status = 'pending', failure_code = '', result_digest = NULL,
		    execution_started_at = NULL, completed_at = NULL
		WHERE plan_id = $1 AND status IN ('running', 'failed', 'indeterminate')
	`, record.ID); err != nil {
		return record, fmt.Errorf("reset retryable extension migration steps: %w", err)
	}
	tag, err := tx.Exec(ctx, `
		UPDATE extension_database_migration_plans
		SET status = 'planned', current_step = 0, failure_code = '',
		    target_ready = FALSE, execution_started_at = NULL, completed_at = NULL,
		    revision = revision + 1, updated_at = statement_timestamp()
		WHERE id = $1 AND revision = $2 AND status = 'failed' AND source_resume_safe = TRUE
	`, record.ID, record.Revision)
	if err != nil {
		return record, fmt.Errorf("reset retryable extension migration plan: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return record, ErrExtensionDatabaseResourceConflict
	}
	record.Status = "planned"
	record.CurrentStep = 0
	record.FailureCode = ""
	record.Revision++
	return record, nil
}

func markExtensionDatabaseMigrationPlanRunning(
	ctx context.Context,
	tx pgx.Tx,
	record extensionDatabaseMigrationPlanRecord,
) (extensionDatabaseMigrationPlanRecord, error) {
	tag, err := tx.Exec(ctx, `
		UPDATE extension_database_migration_plans
		SET status = 'running', execution_started_at = statement_timestamp(),
		    completed_at = NULL, failure_code = '', target_ready = FALSE,
		    revision = revision + 1, updated_at = statement_timestamp()
		WHERE id = $1 AND revision = $2 AND status = 'planned'
	`, record.ID, record.Revision)
	if err != nil {
		return record, fmt.Errorf("start extension database migration plan: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return record, ErrExtensionDatabaseResourceConflict
	}
	record.Status = "running"
	record.Revision++
	return record, nil
}

func markExtensionDatabaseMigrationStepSkipped(
	ctx context.Context,
	tx pgx.Tx,
	step extensionDatabaseMigrationStepRecord,
) error {
	tag, err := tx.Exec(ctx, `
		UPDATE extension_database_migration_steps
		SET status = 'skipped', failure_code = '', completed_at = statement_timestamp()
		WHERE id = $1 AND status = 'pending'
	`, step.ID)
	if err != nil {
		return fmt.Errorf("skip extension database migration step: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrExtensionDatabaseResourceConflict
	}
	return nil
}

func markExtensionDatabaseMigrationStepsRunning(
	ctx context.Context,
	tx pgx.Tx,
	steps []extensionDatabaseMigrationStepRecord,
) error {
	for _, step := range steps {
		tag, err := tx.Exec(ctx, `
			UPDATE extension_database_migration_steps
			SET status = 'running', failure_code = '',
			    execution_started_at = statement_timestamp(), completed_at = NULL
			WHERE id = $1 AND status = 'pending'
		`, step.ID)
		if err != nil {
			return fmt.Errorf("start extension database migration step: %w", err)
		}
		if tag.RowsAffected() != 1 {
			return ErrExtensionDatabaseResourceConflict
		}
	}
	return nil
}

func loadExtensionDatabaseMigrationState(
	ctx context.Context,
	querier extensionDatabaseQuerier,
	extensionID string,
	migrationID string,
) (extensionDatabaseMigrationStateRecord, bool, error) {
	var record extensionDatabaseMigrationStateRecord
	err := querier.QueryRow(ctx, `
		SELECT migration_id, contract_version, package_path, checksum, transaction_policy
		FROM extension_database_migration_state
		WHERE extension_id = $1 AND migration_id = $2
	`, extensionID, migrationID).Scan(
		&record.MigrationID, &record.ContractVersion, &record.PackagePath,
		&record.Checksum, &record.TransactionPolicy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return extensionDatabaseMigrationStateRecord{}, false, nil
	}
	if err != nil {
		return extensionDatabaseMigrationStateRecord{}, false, fmt.Errorf("load extension database migration state: %w", err)
	}
	return record, true, nil
}

func (r extensionDatabaseMigrationStateRecord) matches(step extensionDatabaseMigrationStepPlan) bool {
	return r.MigrationID == step.Declaration.ID && r.ContractVersion == step.Declaration.ContractVersion &&
		r.PackagePath == step.Declaration.Path && r.Checksum == step.Declaration.Digest &&
		r.TransactionPolicy == step.Declaration.Transaction
}

func persistAppliedExtensionDatabaseMigrationSteps(
	ctx context.Context,
	tx pgx.Tx,
	record extensionDatabaseMigrationPlanRecord,
	steps []extensionDatabaseMigrationStepRecord,
	plans map[int]extensionDatabaseMigrationStepPlan,
) error {
	for _, step := range steps {
		plan, ok := plans[step.Position]
		if !ok {
			return ErrExtensionDatabaseResourceConflict
		}
		resultDigest := extensionDatabaseMigrationStepResultDigest(record.PlanDigest, plan)
		tag, err := tx.Exec(ctx, `
			UPDATE extension_database_migration_steps
			SET status = 'applied', failure_code = '', result_digest = $2,
			    completed_at = statement_timestamp()
			WHERE id = $1 AND status = 'running'
		`, step.ID, resultDigest)
		if err != nil {
			return fmt.Errorf("complete extension database migration step: %w", err)
		}
		if tag.RowsAffected() != 1 {
			return ErrExtensionDatabaseResourceConflict
		}
		if plan.Direction == "down" {
			if _, err := tx.Exec(ctx, `
				DELETE FROM extension_database_migration_state
				WHERE extension_id = $1 AND migration_id = $2 AND checksum = $3
			`, record.ExtensionID, plan.Declaration.ID, plan.Declaration.Digest); err != nil {
				return fmt.Errorf("remove reverted extension database migration state: %w", err)
			}
		} else {
			if _, err := tx.Exec(ctx, `
				INSERT INTO extension_database_migration_state (
					extension_id, migration_id, contract_version, package_path,
					checksum, transaction_policy, applied_extension_version_id,
					applied_package_digest, applied_plan_id, applied_step_id
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
				ON CONFLICT (extension_id, migration_id) DO UPDATE SET
					contract_version = EXCLUDED.contract_version,
					package_path = EXCLUDED.package_path,
					checksum = EXCLUDED.checksum,
					transaction_policy = EXCLUDED.transaction_policy,
					applied_extension_version_id = EXCLUDED.applied_extension_version_id,
					applied_package_digest = EXCLUDED.applied_package_digest,
					applied_plan_id = EXCLUDED.applied_plan_id,
					applied_step_id = EXCLUDED.applied_step_id,
					applied_at = statement_timestamp()
			`, record.ExtensionID, plan.Declaration.ID, plan.Declaration.ContractVersion,
				plan.Declaration.Path, plan.Declaration.Digest, plan.Declaration.Transaction,
				plan.Artifact.VersionID, plan.Artifact.PackageDigest, record.ID, step.ID); err != nil {
				return fmt.Errorf("upsert applied extension database migration state: %w", err)
			}
		}
		if _, err := tx.Exec(ctx, `
			UPDATE extension_database_migration_plans
			SET current_step = GREATEST(current_step, $2),
			    revision = revision + 1, updated_at = statement_timestamp()
			WHERE id = $1
		`, record.ID, step.Position); err != nil {
			return fmt.Errorf("advance extension database migration progress: %w", err)
		}
	}
	return nil
}

func extensionDatabaseMigrationStepResultDigest(
	planDigest string,
	step extensionDatabaseMigrationStepPlan,
) string {
	material := fmt.Sprintf("%s\x00%d\x00%s\x00%s\x00%s", planDigest, step.Position,
		step.Direction, step.Declaration.ID, step.Declaration.Digest)
	digest := sha256.Sum256([]byte(material))
	return hex.EncodeToString(digest[:])
}

func assertExtensionDatabaseMigrationTargetState(
	ctx context.Context,
	tx pgx.Tx,
	target extensionDatabaseExactMigrationArtifact,
) error {
	rows, err := tx.Query(ctx, `
		SELECT migration_id, contract_version, package_path, checksum, transaction_policy
		FROM extension_database_migration_state
		WHERE extension_id = $1 ORDER BY migration_id
	`, target.ExtensionID)
	if err != nil {
		return err
	}
	defer rows.Close()
	actual := make(map[string]extensionDatabaseMigrationStateRecord)
	for rows.Next() {
		var state extensionDatabaseMigrationStateRecord
		if err := rows.Scan(&state.MigrationID, &state.ContractVersion, &state.PackagePath, &state.Checksum, &state.TransactionPolicy); err != nil {
			return err
		}
		actual[state.MigrationID] = state
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(actual) != len(target.Migrations) {
		return ErrExtensionDatabaseMigrationChecksumDrift
	}
	for _, declaration := range target.Migrations {
		state, ok := actual[declaration.ID]
		plan := extensionDatabaseMigrationStepPlan{Declaration: declaration}
		if !ok || !state.matches(plan) {
			return ErrExtensionDatabaseMigrationChecksumDrift
		}
	}
	return nil
}

func finalizeExtensionDatabaseMigrationPlan(
	ctx context.Context,
	tx pgx.Tx,
	record extensionDatabaseMigrationPlanRecord,
	status string,
	failureCode string,
	sourceResumeSafe bool,
) (LifecycleMigrationEngineProof, error) {
	targetReady := status == "succeeded"
	if status != "succeeded" && status != "failed" && status != "indeterminate" {
		return LifecycleMigrationEngineProof{}, ErrExtensionDatabaseMigrationInvalid
	}
	if status == "succeeded" {
		failureCode = ""
	}
	proofID := fmt.Sprintf("db-plan:%d", record.ID)
	steps, err := loadExtensionDatabaseMigrationSteps(ctx, tx, record.ID, true)
	if err != nil {
		return LifecycleMigrationEngineProof{}, err
	}
	proofDigest, err := extensionDatabaseMigrationEngineProofDigest(
		record.PlanDigest, status, targetReady, sourceResumeSafe,
		failureCode, record.WarningCode, steps,
	)
	if err != nil {
		return LifecycleMigrationEngineProof{}, err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE extension_database_migration_plans
		SET status = $2, current_step = CASE WHEN $3 THEN total_steps ELSE current_step END,
		    failure_code = $4, target_ready = $3, source_resume_safe = $5,
		    completed_at = statement_timestamp(), revision = revision + 1,
		    updated_at = statement_timestamp()
		WHERE id = $1 AND status IN ('planned', 'running', 'failed', 'indeterminate')
	`, record.ID, status, targetReady, failureCode, sourceResumeSafe)
	if err != nil {
		return LifecycleMigrationEngineProof{}, fmt.Errorf("finalize extension database migration plan: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return LifecycleMigrationEngineProof{}, ErrExtensionDatabaseResourceConflict
	}
	warnings := []string{}
	if record.WarningCode != "" {
		warnings = append(warnings, record.WarningCode)
	}
	warningsJSON, err := json.Marshal(warnings)
	if err != nil {
		return LifecycleMigrationEngineProof{}, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO extension_database_migration_proofs (
			plan_id, plan_digest, proof_id, proof_digest,
			target_ready, source_resume_safe, warning_codes, failure_code
		) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)
		ON CONFLICT (plan_id) DO UPDATE SET
			proof_digest = EXCLUDED.proof_digest,
			target_ready = EXCLUDED.target_ready,
			source_resume_safe = EXCLUDED.source_resume_safe,
			warning_codes = EXCLUDED.warning_codes,
			failure_code = EXCLUDED.failure_code,
			created_at = statement_timestamp()
	`, record.ID, record.PlanDigest, proofID, proofDigest,
		targetReady, sourceResumeSafe, warningsJSON, failureCode)
	if err != nil {
		return LifecycleMigrationEngineProof{}, fmt.Errorf("persist extension database migration proof: %w", err)
	}
	return LifecycleMigrationEngineProof{
		ProofID: proofID, ProofDigest: proofDigest, PlanDigest: record.PlanDigest,
		TargetReady: targetReady, SourceResumeSafe: sourceResumeSafe,
	}, nil
}

func extensionDatabaseMigrationEngineProofDigest(
	planDigest string,
	status string,
	targetReady bool,
	sourceResumeSafe bool,
	failureCode string,
	warningCode string,
	steps []extensionDatabaseMigrationStepRecord,
) (string, error) {
	type proofStep struct {
		Position     int     `json:"position"`
		MigrationID  string  `json:"migrationId"`
		Direction    string  `json:"direction"`
		Status       string  `json:"status"`
		ResultDigest *string `json:"resultDigest,omitempty"`
	}
	document := struct {
		PlanDigest       string      `json:"planDigest"`
		Status           string      `json:"status"`
		TargetReady      bool        `json:"targetReady"`
		SourceResumeSafe bool        `json:"sourceResumeSafe"`
		FailureCode      string      `json:"failureCode,omitempty"`
		WarningCode      string      `json:"warningCode,omitempty"`
		Steps            []proofStep `json:"steps"`
	}{planDigest, status, targetReady, sourceResumeSafe, failureCode, warningCode, nil}
	for _, step := range steps {
		document.Steps = append(document.Steps, proofStep{
			Position: step.Position, MigrationID: step.MigrationID,
			Direction: step.Direction, Status: step.Status, ResultDigest: step.ResultDigest,
		})
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func loadExtensionDatabaseMigrationProof(
	ctx context.Context,
	querier extensionDatabaseQuerier,
	planDigest string,
) (LifecycleMigrationEngineProof, error) {
	var proof LifecycleMigrationEngineProof
	err := querier.QueryRow(ctx, `
		SELECT proof_id, proof_digest, plan_digest, target_ready, source_resume_safe
		FROM extension_database_migration_proofs WHERE plan_digest = $1
	`, planDigest).Scan(
		&proof.ProofID, &proof.ProofDigest, &proof.PlanDigest,
		&proof.TargetReady, &proof.SourceResumeSafe,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return LifecycleMigrationEngineProof{}, ErrExtensionDatabaseGrantNotFound
	}
	if err != nil {
		return LifecycleMigrationEngineProof{}, fmt.Errorf("load extension database migration proof: %w", err)
	}
	return proof, nil
}

func ensureExtensionDatabaseMigrationFailureResource(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
	identifiers ExtensionDatabaseIdentifiers,
	failureCode string,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO extension_database_resources (
			extension_id, schema_name, owner_role_name, runtime_role_name,
			status, failure_code
		) VALUES ($1, $2, $3, $4, 'failed', $5)
		ON CONFLICT (extension_id) DO NOTHING
	`, extensionID, identifiers.Schema, identifiers.OwnerRole, identifiers.RuntimeRole, failureCode)
	if err != nil {
		return fmt.Errorf("record extension database migration failure resource: %w", err)
	}
	resource, err := loadExtensionDatabaseResource(ctx, tx, extensionID, true)
	if err != nil {
		return err
	}
	if !resource.matches(identifiers) {
		return ErrExtensionDatabaseResourceConflict
	}
	return nil
}

func markExtensionDatabaseMigrationExecutionFailure(
	ctx context.Context,
	tx pgx.Tx,
	planID int64,
	failingStepID int64,
	failureCode string,
	indeterminate bool,
	resetOtherRunning bool,
) error {
	if resetOtherRunning {
		if _, err := tx.Exec(ctx, `
			UPDATE extension_database_migration_steps
			SET status = 'pending', failure_code = '', result_digest = NULL,
			    execution_started_at = NULL, completed_at = NULL
			WHERE plan_id = $1 AND status = 'running'
		`, planID); err != nil {
			return fmt.Errorf("reset rolled-back extension migration steps: %w", err)
		}
	}
	status := "failed"
	if indeterminate {
		status = "indeterminate"
	}
	tag, err := tx.Exec(ctx, `
		UPDATE extension_database_migration_steps
		SET status = $2, failure_code = $3,
		    execution_started_at = COALESCE(execution_started_at, statement_timestamp()),
		    completed_at = statement_timestamp()
		WHERE id = $1 AND status IN ('pending', 'running')
	`, failingStepID, status, failureCode)
	if err != nil {
		return fmt.Errorf("record extension migration step failure: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrExtensionDatabaseResourceConflict
	}
	return nil
}

func (r extensionDatabaseMigrationPlanRecord) matchesEnginePlan(plan LifecycleMigrationEnginePlan) bool {
	if r.OperationID != plan.OperationID || r.Operation != string(plan.Operation) ||
		r.Mode != string(plan.Mode) || r.StepID != plan.StepID || r.PlanDigest != plan.PlanDigest ||
		r.ExtensionID != plan.Target.ExtensionID || r.TargetVersionID != plan.Target.VersionID ||
		r.TargetVersion != plan.Target.Version || r.TargetPackageDigest != plan.Target.PackageDigest ||
		r.TargetMigrationsDigest != plan.Target.MigrationsDigest {
		return false
	}
	if plan.Source == nil {
		return !r.SourceVersionID.Valid && !r.SourceVersion.Valid &&
			!r.SourcePackageDigest.Valid && !r.SourceMigrationsDigest.Valid
	}
	return r.SourceVersionID.Valid && r.SourceVersionID.Int64 == plan.Source.VersionID &&
		r.SourceVersion.Valid && r.SourceVersion.String == plan.Source.Version &&
		r.SourcePackageDigest.Valid && r.SourcePackageDigest.String == plan.Source.PackageDigest &&
		r.SourceMigrationsDigest.Valid && r.SourceMigrationsDigest.String == plan.Source.MigrationsDigest
}
