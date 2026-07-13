package extensions

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	LifecycleOperationInstall   = "install"
	LifecycleOperationEnable    = "enable"
	LifecycleOperationDisable   = "disable"
	LifecycleOperationUpgrade   = "upgrade"
	LifecycleOperationRollback  = "rollback"
	LifecycleOperationUninstall = "uninstall"

	LifecycleStatePlanned      = "planned"
	LifecycleStateMigrating    = "migrating"
	LifecycleStateStarting     = "starting"
	LifecycleStateHealthy      = "healthy"
	LifecycleStateRegistering  = "registering"
	LifecycleStateEnabled      = "enabled"
	LifecycleStateDraining     = "draining"
	LifecycleStateUninstalling = "uninstalling"
	LifecycleStateFailed       = "failed"
	LifecycleStateRecovery     = "recovery"

	LifecycleTerminalSucceeded = "succeeded"
	LifecycleTerminalFailed    = "failed"
	LifecycleTerminalCancelled = "cancelled"
	LifecycleTerminalSkipped   = "skipped"

	LifecycleAuthorityBuiltin    = "builtin"
	LifecycleAuthorityTrustGrant = "trust_grant"

	LifecycleRemovalPreserve         = "preserve"
	LifecycleRemovalExportThenRemove = "export_then_remove"
	LifecycleRemovalComplete         = "complete_removal"

	LifecycleStepPlanned   = "planned"
	LifecycleStepRunning   = "running"
	LifecycleStepWaiting   = "waiting"
	LifecycleStepSucceeded = "succeeded"
	LifecycleStepFailed    = "failed"
	LifecycleStepCancelled = "cancelled"
	LifecycleStepSkipped   = "skipped"
)

var (
	ErrLifecycleOperationNotFound   = errors.New("extensions: lifecycle operation not found")
	ErrLifecycleFingerprintConflict = errors.New("extensions: lifecycle idempotency fingerprint conflict")
	ErrLifecycleOperationInProgress = errors.New("extensions: lifecycle operation already in progress")
	ErrLifecycleRevisionConflict    = errors.New("extensions: lifecycle operation revision conflict")
	ErrLifecycleOperationClosed     = errors.New("extensions: lifecycle operation is closed")
	ErrLifecycleNotRecoverable      = errors.New("extensions: lifecycle operation is not recoverable")
	ErrLifecycleStepNotFound        = errors.New("extensions: lifecycle step attempt not found")
	ErrLifecycleStepConflict        = errors.New("extensions: lifecycle step contract conflict")
	ErrLifecycleStepClosed          = errors.New("extensions: lifecycle step attempt is closed")
	ErrLifecycleProgressRegression  = errors.New("extensions: lifecycle progress regressed")
	ErrLifecycleStepLeaseConflict   = errors.New("extensions: lifecycle step lease conflict")
	ErrLifecycleStepLeaseExpired    = errors.New("extensions: lifecycle step lease expired")
	ErrLifecycleInvalidInput        = errors.New("extensions: invalid lifecycle repository input")
)

type LifecycleExecutionError struct {
	Code       string          `json:"code,omitempty"`
	Reason     string          `json:"reason,omitempty"`
	Message    string          `json:"message,omitempty"`
	Retryable  bool            `json:"retryable,omitempty"`
	RetryAfter *time.Time      `json:"retryAfter,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

type LifecycleOperation struct {
	ID                 int64                   `json:"id"`
	ExtensionID        string                  `json:"extensionId"`
	ExtensionVersion   string                  `json:"extensionVersion"`
	PackageDigest      string                  `json:"packageDigest"`
	ArtifactDigests    json.RawMessage         `json:"artifactDigests"`
	Operation          string                  `json:"operation"`
	State              string                  `json:"state"`
	PlanVersion        string                  `json:"planVersion"`
	IdempotencyKey     string                  `json:"idempotencyKey"`
	RequestFingerprint string                  `json:"requestFingerprint"`
	AuthorityType      string                  `json:"authorityType"`
	TrustGrantID       int64                   `json:"trustGrantId,omitempty"`
	AuthoritySnapshot  json.RawMessage         `json:"authoritySnapshot"`
	RequestedByUserID  int64                   `json:"requestedByUserId,omitempty"`
	AuditEventID       int64                   `json:"auditEventId,omitempty"`
	RemovalMode        string                  `json:"removalMode,omitempty"`
	Forced             bool                    `json:"forced"`
	AttemptCount       int                     `json:"attemptCount"`
	Revision           int64                   `json:"revision"`
	CurrentStepID      string                  `json:"currentStepId,omitempty"`
	Checkpoint         json.RawMessage         `json:"checkpoint"`
	Progress           json.RawMessage         `json:"progress"`
	TerminalResult     string                  `json:"terminalResult,omitempty"`
	ResultDocument     json.RawMessage         `json:"resultDocument,omitempty"`
	Error              LifecycleExecutionError `json:"error,omitempty"`
	CreatedAt          time.Time               `json:"createdAt"`
	UpdatedAt          time.Time               `json:"updatedAt"`
	StartedAt          *time.Time              `json:"startedAt,omitempty"`
	CompletedAt        *time.Time              `json:"completedAt,omitempty"`
}

type AcquireLifecycleOperationInput struct {
	ExtensionID        string
	ExtensionVersion   string
	PackageDigest      string
	ArtifactDigests    json.RawMessage
	Operation          string
	PlanVersion        string
	IdempotencyKey     string
	RequestFingerprint string
	AuthorityType      string
	TrustGrantID       int64
	AuthoritySnapshot  json.RawMessage
	RequestedByUserID  int64
	AuditEventID       int64
	RemovalMode        string
	Forced             bool
}

type AcquireLifecycleOperationResult struct {
	Operation LifecycleOperation
	Created   bool
}

type TransitionLifecycleOperationInput struct {
	OperationID      int64
	ExpectedRevision int64
	ExpectedState    string
	State            string
	CurrentStepID    string
	Checkpoint       json.RawMessage
	Progress         json.RawMessage
}

type CompleteLifecycleOperationInput struct {
	OperationID      int64
	ExpectedRevision int64
	ExpectedState    string
	State            string
	TerminalResult   string
	ResultDocument   json.RawMessage
	Error            LifecycleExecutionError
	AuditEventID     int64
}

type ResumeLifecycleOperationInput struct {
	OperationID      int64
	ExpectedRevision int64
	ExpectedState    string
}

type LifecycleStepAttempt struct {
	ID               int64                   `json:"id"`
	OperationID      int64                   `json:"operationId"`
	StepID           string                  `json:"stepId"`
	LifecycleAction  string                  `json:"lifecycleAction"`
	PlanVersion      string                  `json:"planVersion"`
	Attempt          int                     `json:"attempt"`
	Status           string                  `json:"status"`
	Checkpoint       string                  `json:"checkpoint,omitempty"`
	CompletedUnits   int64                   `json:"completedUnits"`
	TotalUnits       int64                   `json:"totalUnits"`
	ProgressMessage  string                  `json:"progressMessage,omitempty"`
	InputDocument    json.RawMessage         `json:"inputDocument,omitempty"`
	ResultDocument   json.RawMessage         `json:"resultDocument,omitempty"`
	Error            LifecycleExecutionError `json:"error,omitempty"`
	SkipReason       string                  `json:"skipReason,omitempty"`
	Forced           bool                    `json:"forced"`
	ActorUserID      int64                   `json:"actorUserId,omitempty"`
	AuditEventID     int64                   `json:"auditEventId,omitempty"`
	LeaseOwnerToken  string                  `json:"leaseOwnerToken,omitempty"`
	LeaseExpiresAt   *time.Time              `json:"leaseExpiresAt,omitempty"`
	LeaseRevision    int64                   `json:"leaseRevision"`
	LeaseHeartbeatAt *time.Time              `json:"leaseHeartbeatAt,omitempty"`
	CreatedAt        time.Time               `json:"createdAt"`
	UpdatedAt        time.Time               `json:"updatedAt"`
	StartedAt        *time.Time              `json:"startedAt,omitempty"`
	CompletedAt      *time.Time              `json:"completedAt,omitempty"`
}

type BeginLifecycleStepAttemptInput struct {
	OperationID     int64
	StepID          string
	LifecycleAction string
	PlanVersion     string
	InputDocument   json.RawMessage
	Checkpoint      string
	ActorUserID     int64
	AuditEventID    int64
}

type BeginLifecycleStepAttemptResult struct {
	Attempt LifecycleStepAttempt
	Created bool
}

type UpdateLifecycleStepProgressInput struct {
	AttemptID       int64
	LeaseOwnerToken string
	LeaseRevision   int64
	Status          string
	Checkpoint      string
	CompletedUnits  int64
	TotalUnits      int64
	Message         string
}

type CompleteLifecycleStepAttemptInput struct {
	AttemptID       int64
	LeaseOwnerToken string
	LeaseRevision   int64
	Status          string
	Checkpoint      string
	CompletedUnits  int64
	TotalUnits      int64
	Message         string
	ResultDocument  json.RawMessage
	Error           LifecycleExecutionError
	SkipReason      string
	Forced          bool
	ActorUserID     int64
	AuditEventID    int64
}

type ClaimLifecycleStepLeaseInput struct {
	AttemptID        int64
	ExpectedRevision int64
	OwnerToken       string
	DurationMS       int64
}

type HeartbeatLifecycleStepLeaseInput struct {
	AttemptID  int64
	OwnerToken string
	Revision   int64
	DurationMS int64
}

type ReleaseLifecycleStepLeaseInput struct {
	AttemptID  int64
	OwnerToken string
	Revision   int64
}

type PostgresLifecycleRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresLifecycleRepository(pool *pgxpool.Pool) *PostgresLifecycleRepository {
	return &PostgresLifecycleRepository{pool: pool}
}

type lifecycleScanner interface {
	Scan(...any) error
}

func scanLifecycleOperation(scanner lifecycleScanner) (LifecycleOperation, error) {
	var item LifecycleOperation
	var artifacts, authority, checkpoint, progress, result, errorMetadata []byte
	if err := scanner.Scan(
		&item.ID, &item.ExtensionID, &item.ExtensionVersion, &item.PackageDigest,
		&artifacts, &item.Operation, &item.State, &item.PlanVersion,
		&item.IdempotencyKey, &item.RequestFingerprint, &item.AuthorityType,
		&item.TrustGrantID, &authority, &item.RequestedByUserID, &item.AuditEventID,
		&item.RemovalMode, &item.Forced, &item.AttemptCount, &item.Revision,
		&item.CurrentStepID, &checkpoint, &progress, &item.TerminalResult, &result,
		&item.Error.Code, &item.Error.Reason, &item.Error.Message, &item.Error.Retryable,
		&item.Error.RetryAfter, &errorMetadata, &item.CreatedAt, &item.UpdatedAt,
		&item.StartedAt, &item.CompletedAt,
	); err != nil {
		return LifecycleOperation{}, err
	}
	item.ArtifactDigests = cloneLifecycleJSON(artifacts)
	item.AuthoritySnapshot = cloneLifecycleJSON(authority)
	item.Checkpoint = cloneLifecycleJSON(checkpoint)
	item.Progress = cloneLifecycleJSON(progress)
	item.ResultDocument = cloneLifecycleJSON(result)
	item.Error.Metadata = cloneLifecycleJSON(errorMetadata)
	return item, nil
}

func scanLifecycleStepAttempt(scanner lifecycleScanner) (LifecycleStepAttempt, error) {
	var item LifecycleStepAttempt
	var input, result, errorMetadata []byte
	if err := scanner.Scan(
		&item.ID, &item.OperationID, &item.StepID, &item.LifecycleAction,
		&item.PlanVersion, &item.Attempt, &item.Status, &item.Checkpoint,
		&item.CompletedUnits, &item.TotalUnits, &item.ProgressMessage,
		&input, &result, &item.Error.Code, &item.Error.Reason, &item.Error.Message,
		&item.Error.Retryable, &item.Error.RetryAfter, &errorMetadata,
		&item.SkipReason, &item.Forced, &item.ActorUserID, &item.AuditEventID,
		&item.LeaseOwnerToken, &item.LeaseExpiresAt, &item.LeaseRevision, &item.LeaseHeartbeatAt,
		&item.CreatedAt, &item.UpdatedAt, &item.StartedAt, &item.CompletedAt,
	); err != nil {
		return LifecycleStepAttempt{}, err
	}
	item.InputDocument = cloneLifecycleJSON(input)
	item.ResultDocument = cloneLifecycleJSON(result)
	item.Error.Metadata = cloneLifecycleJSON(errorMetadata)
	return item, nil
}

func lifecycleOperationSelectSQL() string {
	return `
		SELECT id, extension_id, extension_version, package_digest, artifact_digests,
		       operation, state, plan_version, idempotency_key, request_fingerprint,
		       authority_type, COALESCE(trust_grant_id, 0), authority_snapshot,
		       COALESCE(requested_by_user_id, 0), COALESCE(audit_event_id, 0),
		       COALESCE(removal_mode, ''), forced, attempt_count, revision,
		       current_step_id, checkpoint, progress, COALESCE(terminal_result, ''),
		       result_document, error_code, error_reason, error_message,
		       error_retryable, error_retry_after, error_metadata,
		       created_at, updated_at, started_at, completed_at
		FROM extension_lifecycle_operations
	`
}

func lifecycleStepAttemptSelectSQL() string {
	return `
		SELECT id, operation_id, step_id, lifecycle_action, plan_version, attempt,
		       status, checkpoint, completed_units, total_units, progress_message,
		       input_document, result_document, error_code, error_reason,
		       error_message, error_retryable, error_retry_after, error_metadata,
		       skip_reason, forced, COALESCE(actor_user_id, 0), COALESCE(audit_event_id, 0),
		       lease_owner_token, lease_expires_at, lease_revision, lease_heartbeat_at,
		       created_at, updated_at, started_at, completed_at
		FROM extension_lifecycle_steps
	`
}

func lifecycleJSONObject(value json.RawMessage) string {
	if len(bytes.TrimSpace(value)) == 0 {
		return "{}"
	}
	return string(value)
}

func lifecycleNullableJSON(value json.RawMessage) any {
	if len(bytes.TrimSpace(value)) == 0 {
		return nil
	}
	return string(value)
}

func lifecycleJSONEqual(left, right json.RawMessage) bool {
	if len(bytes.TrimSpace(left)) == 0 || len(bytes.TrimSpace(right)) == 0 {
		return len(bytes.TrimSpace(left)) == 0 && len(bytes.TrimSpace(right)) == 0
	}
	var leftValue, rightValue any
	if json.Unmarshal(left, &leftValue) != nil || json.Unmarshal(right, &rightValue) != nil {
		return bytes.Equal(bytes.TrimSpace(left), bytes.TrimSpace(right))
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

func cloneLifecycleJSON(value []byte) json.RawMessage {
	if len(value) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), value...)
}

func nullableLifecycleID(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}
