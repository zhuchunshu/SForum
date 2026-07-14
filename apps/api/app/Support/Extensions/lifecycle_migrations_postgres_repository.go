package extensionsruntime

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
)

type lifecycleMigrationProofRecord struct {
	ID                     int64
	OperationID            int64
	Operation              string
	Mode                   LifecycleBoundaryMigrationMode
	StepID                 string
	Position               int
	SourceExtensionID      sql.NullString
	SourceExtensionVersion sql.NullString
	SourcePackageDigest    sql.NullString
	SourceVersionID        sql.NullInt64
	SourceMigrationsDigest sql.NullString
	TargetExtensionID      string
	TargetExtensionVersion string
	TargetPackageDigest    string
	TargetVersionID        int64
	TargetMigrationsDigest string
	PlanDigest             string
	FirstAttempt           int
	LastAttempt            int
	LastObservedStepID     string
	LastObservedAttempt    int
	Status                 string
	TargetReady            bool
	SourceResumeSafe       bool
	ProofKind              sql.NullString
	ProofID                sql.NullString
	ProofDigest            sql.NullString
	Revision               int64
}

func (r lifecycleMigrationProofRecord) matches(plan lifecycleMigrationPlan) bool {
	if r.OperationID != plan.OperationID || r.Operation != string(plan.Operation) ||
		r.Mode != plan.Mode || r.StepID != plan.StepID || r.Position != plan.Position ||
		r.TargetExtensionID != plan.Target.ExtensionID ||
		r.TargetExtensionVersion != plan.Target.Version ||
		r.TargetPackageDigest != plan.Target.PackageDigest ||
		r.TargetVersionID != plan.Target.VersionID ||
		r.TargetMigrationsDigest != plan.Target.MigrationsDigest || r.PlanDigest != plan.PlanDigest {
		return false
	}
	if plan.Source == nil {
		return !r.SourceExtensionID.Valid && !r.SourceExtensionVersion.Valid &&
			!r.SourcePackageDigest.Valid && !r.SourceVersionID.Valid && !r.SourceMigrationsDigest.Valid
	}
	return r.SourceExtensionID.Valid && r.SourceExtensionID.String == plan.Source.ExtensionID &&
		r.SourceExtensionVersion.Valid && r.SourceExtensionVersion.String == plan.Source.Version &&
		r.SourcePackageDigest.Valid && r.SourcePackageDigest.String == plan.Source.PackageDigest &&
		r.SourceVersionID.Valid && r.SourceVersionID.Int64 == plan.Source.VersionID &&
		r.SourceMigrationsDigest.Valid && r.SourceMigrationsDigest.String == plan.Source.MigrationsDigest
}

func ensureLifecycleMigrationProof(
	ctx context.Context,
	tx pgx.Tx,
	plan lifecycleMigrationPlan,
) (lifecycleMigrationProofRecord, error) {
	var sourceID, sourceVersion, sourceDigest, sourceMigrations any
	var sourceVersionID any
	if plan.Source != nil {
		sourceID, sourceVersion = plan.Source.ExtensionID, plan.Source.Version
		sourceDigest, sourceVersionID = plan.Source.PackageDigest, plan.Source.VersionID
		sourceMigrations = plan.Source.MigrationsDigest
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO extension_lifecycle_migration_proofs (
			operation_id, operation, migration_mode, step_id, position,
			source_extension_id, source_extension_version, source_package_digest,
			source_version_id, source_migrations_digest,
			target_extension_id, target_extension_version, target_package_digest,
			target_version_id, target_migrations_digest, plan_digest
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16
		)
		ON CONFLICT (operation_id, migration_mode) DO NOTHING
	`, plan.OperationID, plan.Operation, plan.Mode, plan.StepID, plan.Position,
		sourceID, sourceVersion, sourceDigest, sourceVersionID, sourceMigrations,
		plan.Target.ExtensionID, plan.Target.Version, plan.Target.PackageDigest,
		plan.Target.VersionID, plan.Target.MigrationsDigest, plan.PlanDigest)
	if err != nil {
		return lifecycleMigrationProofRecord{}, fmt.Errorf("insert lifecycle migration proof fence: %w", err)
	}
	record, err := loadLifecycleMigrationProof(ctx, tx, plan.OperationID, plan.Mode, true)
	if err != nil {
		return lifecycleMigrationProofRecord{}, mapLifecycleMigrationProofError("load lifecycle migration proof fence", err)
	}
	if !record.matches(plan) {
		return lifecycleMigrationProofRecord{}, ErrLifecycleMigrationsConflict
	}
	return record, nil
}

func loadLifecycleMigrationProof(
	ctx context.Context,
	querier interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	operationID int64,
	mode LifecycleBoundaryMigrationMode,
	forUpdate bool,
) (lifecycleMigrationProofRecord, error) {
	query := `
		SELECT id, operation_id, operation, migration_mode, step_id, position,
		       source_extension_id, source_extension_version, source_package_digest,
		       source_version_id, source_migrations_digest,
		       target_extension_id, target_extension_version, target_package_digest,
		       target_version_id, target_migrations_digest, plan_digest,
		       first_attempt, last_attempt, last_observed_step_id, last_observed_attempt,
		       status, target_ready, source_resume_safe,
		       proof_kind, proof_id, proof_digest, revision
		FROM extension_lifecycle_migration_proofs
		WHERE operation_id = $1 AND migration_mode = $2
	`
	if forUpdate {
		query += " FOR UPDATE"
	}
	var record lifecycleMigrationProofRecord
	err := querier.QueryRow(ctx, query, operationID, mode).Scan(
		&record.ID, &record.OperationID, &record.Operation, &record.Mode,
		&record.StepID, &record.Position,
		&record.SourceExtensionID, &record.SourceExtensionVersion,
		&record.SourcePackageDigest, &record.SourceVersionID, &record.SourceMigrationsDigest,
		&record.TargetExtensionID, &record.TargetExtensionVersion,
		&record.TargetPackageDigest, &record.TargetVersionID,
		&record.TargetMigrationsDigest, &record.PlanDigest,
		&record.FirstAttempt, &record.LastAttempt,
		&record.LastObservedStepID, &record.LastObservedAttempt,
		&record.Status, &record.TargetReady, &record.SourceResumeSafe,
		&record.ProofKind, &record.ProofID, &record.ProofDigest, &record.Revision,
	)
	return record, err
}

type lifecycleMigrationArtifactProof struct {
	RecordID int64
	ProofID  string
	Digest   string
}

func findLifecycleMigrationArtifactProof(
	ctx context.Context,
	tx pgx.Tx,
	artifact LifecycleMigrationArtifact,
	excludeRecordID int64,
) (lifecycleMigrationArtifactProof, bool, error) {
	var proof lifecycleMigrationArtifactProof
	err := tx.QueryRow(ctx, `
		SELECT id, proof_id, proof_digest
		FROM extension_lifecycle_migration_proofs
		WHERE target_extension_id = $1
		  AND target_extension_version = $2
		  AND target_package_digest = $3
		  AND target_version_id = $4
		  AND target_migrations_digest = $5
		  AND target_ready = TRUE
		  AND status = 'target_ready'
		  AND proof_kind IS NOT NULL
		  AND proof_id IS NOT NULL
		  AND proof_digest ~ '^[0-9a-f]{64}$'
		  AND id <> $6
		ORDER BY proven_at DESC, id DESC
		LIMIT 1
	`, artifact.ExtensionID, artifact.Version, artifact.PackageDigest,
		artifact.VersionID, artifact.MigrationsDigest, excludeRecordID).Scan(
		&proof.RecordID, &proof.ProofID, &proof.Digest,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return lifecycleMigrationArtifactProof{}, false, nil
	}
	if err != nil {
		return lifecycleMigrationArtifactProof{}, false, fmt.Errorf("find exact artifact migration proof: %w", err)
	}
	return proof, true, nil
}

type lifecycleMigrationDurableProof struct {
	Kind             string
	ID               string
	Digest           string
	TargetReady      bool
	SourceResumeSafe bool
}

func hostNoopLifecycleMigrationProof(plan lifecycleMigrationPlan) (lifecycleMigrationDurableProof, error) {
	id := "host-noop:" + plan.PlanDigest
	digest, err := lifecycleMigrationProofDigest(struct {
		Kind       string `json:"kind"`
		PlanDigest string `json:"planDigest"`
	}{lifecycleMigrationProofHostNoop, plan.PlanDigest})
	return lifecycleMigrationDurableProof{
		Kind: lifecycleMigrationProofHostNoop, ID: id, Digest: digest,
		TargetReady: true, SourceResumeSafe: true,
	}, err
}

func reusedLifecycleMigrationProof(
	plan lifecycleMigrationPlan,
	source lifecycleMigrationArtifactProof,
) lifecycleMigrationDurableProof {
	digest, _ := lifecycleMigrationProofDigest(struct {
		Kind         string `json:"kind"`
		PlanDigest   string `json:"planDigest"`
		SourceID     int64  `json:"sourceId"`
		SourceProof  string `json:"sourceProof"`
		SourceDigest string `json:"sourceDigest"`
	}{lifecycleMigrationProofReused, plan.PlanDigest, source.RecordID, source.ProofID, source.Digest})
	return lifecycleMigrationDurableProof{
		Kind: lifecycleMigrationProofReused,
		ID:   "reused:" + strconv.FormatInt(source.RecordID, 10), Digest: digest,
		TargetReady: true, SourceResumeSafe: true,
	}
}

func lifecycleMigrationProofDigest(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode lifecycle migration proof: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func writeLifecycleMigrationProof(
	ctx context.Context,
	tx pgx.Tx,
	record lifecycleMigrationProofRecord,
	attempt int,
	proof lifecycleMigrationDurableProof,
) error {
	if attempt <= 0 || !validLifecycleMigrationProofID(proof.ID) ||
		!validLifecycleCleanupDigest(proof.Digest) {
		return ErrLifecycleMigrationEngineProofBad
	}
	status := lifecycleMigrationStatusBlocked
	if proof.TargetReady {
		status = lifecycleMigrationStatusTargetReady
	}
	tag, err := tx.Exec(ctx, `
		UPDATE extension_lifecycle_migration_proofs
		SET first_attempt = CASE WHEN first_attempt = 0 THEN $2 ELSE first_attempt END,
		    last_attempt = GREATEST(last_attempt, $2),
		    status = $3,
		    target_ready = $4,
		    source_resume_safe = $5,
		    proof_kind = $6,
		    proof_id = $7,
		    proof_digest = $8,
		    proven_at = statement_timestamp(),
		    revision = revision + 1,
		    updated_at = statement_timestamp()
		WHERE id = $1 AND revision = $9 AND last_attempt <= $2
	`, record.ID, attempt, status, proof.TargetReady, proof.SourceResumeSafe,
		proof.Kind, proof.ID, proof.Digest, record.Revision)
	if err != nil {
		return fmt.Errorf("write lifecycle migration durable proof: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrLifecycleMigrationsConflict
	}
	return nil
}

func blockLifecycleMigrationProof(
	ctx context.Context,
	tx pgx.Tx,
	record lifecycleMigrationProofRecord,
	attempt int,
	sourceResumeSafe bool,
) error {
	if attempt <= 0 || attempt < record.LastAttempt {
		return ErrLifecycleMigrationsConflict
	}
	tag, err := tx.Exec(ctx, `
		UPDATE extension_lifecycle_migration_proofs
		SET first_attempt = CASE WHEN first_attempt = 0 THEN $2 ELSE first_attempt END,
		    last_attempt = GREATEST(last_attempt, $2),
		    status = 'blocked', target_ready = FALSE,
		    source_resume_safe = $3,
		    revision = revision + 1,
		    updated_at = statement_timestamp()
		WHERE id = $1 AND revision = $4 AND last_attempt <= $2
	`, record.ID, attempt, sourceResumeSafe, record.Revision)
	if err != nil {
		return fmt.Errorf("block lifecycle migration plan: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrLifecycleMigrationsConflict
	}
	return nil
}

func startLifecycleMigrationExecution(
	ctx context.Context,
	tx pgx.Tx,
	record lifecycleMigrationProofRecord,
	attempt int,
) error {
	if attempt <= 0 || attempt < record.LastAttempt {
		return ErrLifecycleMigrationsConflict
	}
	tag, err := tx.Exec(ctx, `
		UPDATE extension_lifecycle_migration_proofs
		SET first_attempt = CASE WHEN first_attempt = 0 THEN $2 ELSE first_attempt END,
		    last_attempt = GREATEST(last_attempt, $2),
		    status = 'executing', target_ready = FALSE,
		    source_resume_safe = FALSE,
		    proof_kind = NULL, proof_id = NULL, proof_digest = NULL, proven_at = NULL,
		    execution_started_at = COALESCE(execution_started_at, statement_timestamp()),
		    revision = revision + 1,
		    updated_at = statement_timestamp()
		WHERE id = $1 AND revision = $3 AND last_attempt <= $2
	`, record.ID, attempt, record.Revision)
	if err != nil {
		return fmt.Errorf("start lifecycle migration execution: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrLifecycleMigrationsConflict
	}
	return nil
}

func validateLifecycleMigrationEngineProof(
	plan lifecycleMigrationPlan,
	proof LifecycleMigrationEngineProof,
) error {
	if proof.PlanDigest != plan.PlanDigest || !validLifecycleMigrationProofID(proof.ProofID) ||
		!validLifecycleCleanupDigest(proof.ProofDigest) {
		return ErrLifecycleMigrationEngineProofBad
	}
	return nil
}

func validLifecycleMigrationProofID(value string) bool {
	if value == "" || len(value) > 200 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '.' || char == '_' ||
			char == '-' || char == ':' {
			continue
		}
		return false
	}
	return true
}

func mapLifecycleMigrationProofError(action string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %s", ErrLifecycleMigrationsConflict, action)
	}
	return fmt.Errorf("%s: %w", action, err)
}
