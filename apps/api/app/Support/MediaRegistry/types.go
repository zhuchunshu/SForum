package mediaregistry

import (
	"context"
	"errors"
	"time"
)

const SchemaVersion = "sforum.media-pipeline-registry@1"

const (
	PlanUpload    = "upload"
	PlanProcess   = "process"
	PlanDelivery  = "delivery"
	PlanRetention = "retention"
	PlanDelete    = "delete"
)

const (
	StageValidate     = "validate"
	StageScan         = "scan"
	StageMetadata     = "metadata"
	StageTransform    = "transform"
	StageCDN          = "cdn"
	StageRetention    = "retention"
	StageBeforeDelete = "before_delete"
	StageAfterDelete  = "after_delete"
)

const (
	ProcessorCompose   = "compose"
	ProcessorExclusive = "exclusive"

	ExecutionSync       = "sync"
	ExecutionBackground = "background"

	FailureFailClosed       = "fail_closed"
	FailureSkip             = "skip"
	FailureFallbackOriginal = "fallback_original"
)

const (
	RetryNone        = "none"
	RetryTransient   = "transient"
	RetryRateLimited = "rate_limited"
	RetryCrash       = "provider_crash"
	RetryPermanent   = "permanent"

	DecisionAllow  = "allow"
	DecisionReject = "reject"
)

const (
	SourceOriginal      = "original"
	SourceSourceOfTruth = "source_of_truth"
	SourceVariant       = "variant"

	ConflictMIMEPolicy = "mime_policy"
	ConflictProcessor  = "processor"
	ConflictVariant    = "variant"

	VariantBindingActive  = "active"
	VariantBindingPending = "pending"

	VariantPendingProcessorMissing          = "processor_missing"
	VariantPendingProcessorIdentityMismatch = "processor_identity_mismatch"
)

var (
	ErrInvalid             = errors.New("media registry input is invalid")
	ErrConflict            = errors.New("media registry graph conflicts")
	ErrArtifactConflict    = errors.New("media registry exact artifact does not own the publication")
	ErrRevisionConflict    = errors.New("media registry revision changed")
	ErrSafeMode            = errors.New("media registry rejects third-party publication in safe mode")
	ErrNotFound            = errors.New("media registry contribution is not found")
	ErrPolicyUnavailable   = errors.New("media registry MIME policy is unavailable")
	ErrPermissionDenied    = errors.New("media registry permission was denied")
	ErrRuntimeUnavailable  = errors.New("media registry exact runtime is unavailable")
	ErrRuntimeLeaseRelease = errors.New("media registry exact runtime lease release failed")
	ErrPlanStale           = errors.New("media registry plan is stale or forged")
	ErrBudgetExceeded      = errors.New("media registry budget was exceeded")
	ErrMIMEConfusion       = errors.New("media registry detected MIME confusion")
	ErrMediaRejected       = errors.New("media registry processor rejected the media")
	ErrOutputRejected      = errors.New("media registry rejected provider output")
	ErrExecutionTimeout    = errors.New("media registry execution exceeded its Host deadline")
	ErrRuntimeQuarantined  = errors.New("media registry exact runtime is quarantined")
	ErrDeletionFence       = errors.New("media registry after_delete requires a durable Host deletion fence")
	ErrReceiptAuthority    = errors.New("media registry durable receipt authority is unavailable")
	ErrOperationBusy       = errors.New("media registry operation is already claimed")
	ErrPrivatePlan         = errors.New("media registry private plan cannot be serialized")
)

// Artifact binds every declaration to one exact package and runtime. Core
// artifacts carry a package-private seal issued only by NewCoreArtifact.
type Artifact struct {
	ExtensionID       string `json:"extensionId"`
	ExtensionVersion  string `json:"extensionVersion"`
	PackageDigest     string `json:"packageDigest"`
	ImpactDigest      string `json:"impactDigest"`
	VersionID         int64  `json:"versionId,omitempty"`
	RuntimeInstanceID string `json:"runtimeInstanceId,omitempty"`
	Core              bool   `json:"core,omitempty"`
	coreSeal          [32]byte
}

// Budget is an exact policy ceiling. Host-wide hard limits remain authoritative
// even when a trusted contribution declares a larger value.
type Budget struct {
	MaxFileBytes          int64 `json:"maxFileBytes"`
	MaxFiles              int   `json:"maxFiles"`
	MaxDecompressedBytes  int64 `json:"maxDecompressedBytes"`
	MaxDecompressionRatio int64 `json:"maxDecompressionRatio"`
	MaxFilenameBytes      int   `json:"maxFilenameBytes"`
	MaxMIMECandidates     int   `json:"maxMimeCandidates"`
	MaxMetadataBytes      int   `json:"maxMetadataBytes"`
	MaxVariants           int   `json:"maxVariants"`
}

type MIMEAlias struct {
	Declared string `json:"declared"`
	Detected string `json:"detected"`
}

type MIMEPolicyDeclaration struct {
	ID                  string      `json:"id"`
	ContractVersion     string      `json:"contractVersion"`
	Purpose             string      `json:"purpose"`
	Priority            int         `json:"priority,omitempty"`
	RequiredPermission  string      `json:"requiredPermission"`
	AllowedMIMEs        []string    `json:"allowedMimes"`
	DeniedMIMEs         []string    `json:"deniedMimes,omitempty"`
	AllowedExtensions   []string    `json:"allowedExtensions,omitempty"`
	MIMEAliases         []MIMEAlias `json:"mimeAliases,omitempty"`
	StrictDeclaredMIME  bool        `json:"strictDeclaredMime"`
	RequireExpandedSize bool        `json:"requireExpandedSize,omitempty"`
	Budget              Budget      `json:"budget"`
}

type RetryPolicy struct {
	MaxAttempts      int `json:"maxAttempts"`
	BaseDelaySeconds int `json:"baseDelaySeconds"`
	MaxDelaySeconds  int `json:"maxDelaySeconds"`
}

type ProcessorDeclaration struct {
	ID                 string      `json:"id"`
	ContractVersion    string      `json:"contractVersion"`
	Stage              string      `json:"stage"`
	Purpose            string      `json:"purpose"`
	MIMEs              []string    `json:"mimes"`
	Handler            string      `json:"handler"`
	Priority           int         `json:"priority,omitempty"`
	Mode               string      `json:"mode"`
	Slot               string      `json:"slot,omitempty"`
	Execution          string      `json:"execution"`
	FailureMode        string      `json:"failureMode"`
	RequiredPermission string      `json:"requiredPermission"`
	Retry              RetryPolicy `json:"retry"`
}

// VariantDeclaration describes regenerable output only. Source is always the
// immutable original/source-of-truth; a declaration cannot replace it.
type VariantDeclaration struct {
	ID                        string `json:"id"`
	ContractVersion           string `json:"contractVersion"`
	Purpose                   string `json:"purpose"`
	Name                      string `json:"name"`
	ProcessorID               string `json:"processorId"`
	ProcessorContractVersion  string `json:"processorContractVersion"`
	ProcessorOwnerExtensionID string `json:"processorOwnerExtensionId"`
	ProcessorPackageDigest    string `json:"processorPackageDigest"`
	OutputMIME                string `json:"outputMime"`
	Priority                  int    `json:"priority,omitempty"`
}

type Publication struct {
	Artifact   Artifact                `json:"artifact"`
	Policies   []MIMEPolicyDeclaration `json:"policies,omitempty"`
	Processors []ProcessorDeclaration  `json:"processors,omitempty"`
	Variants   []VariantDeclaration    `json:"variants,omitempty"`
}

type MIMEPolicyContribution struct {
	MIMEPolicyDeclaration
	Artifact Artifact `json:"artifact"`
}

type ProcessorContribution struct {
	ProcessorDeclaration
	Artifact Artifact `json:"artifact"`
}

type VariantContribution struct {
	VariantDeclaration
	Artifact Artifact `json:"artifact"`
}

type ProviderRef struct {
	ContributionID string   `json:"contributionId"`
	Artifact       Artifact `json:"artifact"`
}

// VariantBinding exposes whether an inert variant declaration currently has
// its exact declarative processor dependency. Processor is present only for an
// active binding and pins the exact artifact selected by the built state.
type VariantBinding struct {
	Variant   VariantContribution `json:"variant"`
	Status    string              `json:"status"`
	Reason    string              `json:"reason,omitempty"`
	Processor *ProviderRef        `json:"processor,omitempty"`
}

type ProviderSelection struct {
	Family   string      `json:"family"`
	Key      string      `json:"key"`
	Provider ProviderRef `json:"provider"`
}

type ProviderConflict struct {
	Family              string        `json:"family"`
	Key                 string        `json:"key"`
	Candidates          []ProviderRef `json:"candidates"`
	Winner              ProviderRef   `json:"winner"`
	SelectionConfigured bool          `json:"selectionConfigured,omitempty"`
}

type Snapshot struct {
	SchemaVersion   string                   `json:"schemaVersion"`
	Revision        uint64                   `json:"revision"`
	Digest          string                   `json:"digest"`
	SafeMode        bool                     `json:"safeMode,omitempty"`
	Publications    []Publication            `json:"publications"`
	Policies        []MIMEPolicyContribution `json:"policies"`
	Processors      []ProcessorContribution  `json:"processors"`
	Variants        []VariantContribution    `json:"variants"`
	VariantBindings []VariantBinding         `json:"variantBindings"`
	Selections      []ProviderSelection      `json:"selections,omitempty"`
	Conflicts       []ProviderConflict       `json:"conflicts,omitempty"`
}

// SourceAsset is metadata for a Host-owned immutable object. The Registry
// never receives bytes or storage credentials and never mutates this value.
type SourceAsset struct {
	ID        string `json:"id"`
	Digest    string `json:"digest"`
	Kind      string `json:"kind"`
	MIME      string `json:"mime"`
	Filename  string `json:"-"`
	SizeBytes int64  `json:"sizeBytes"`
	Immutable bool   `json:"immutable"`
}

type UploadFacts struct {
	BatchFileCount    int      `json:"batchFileCount"`
	DeclaredMIME      string   `json:"declaredMime,omitempty"`
	DetectedMIMEs     []string `json:"detectedMimes"`
	Archive           bool     `json:"archive,omitempty"`
	DecompressedBytes int64    `json:"decompressedBytes,omitempty"`
}

type Actor struct {
	ID                    string `json:"id"`
	PermissionFingerprint string `json:"permissionFingerprint"`
}

type AuthorizationRequest struct {
	Actor      Actor
	Permission string
	PlanKind   string
	Purpose    string
	SourceID   string
}

// Authorizer is the mandatory Host authority boundary. Implementations must
// recheck current actor/resource policy; copied permission lists are not used.
type Authorizer interface {
	Authorize(context.Context, AuthorizationRequest) bool
}

type PlanRequest struct {
	Kind       string
	Purpose    string
	Permission string
	Actor      Actor
	Source     SourceAsset
	Upload     UploadFacts
}

type PlanStep struct {
	ID        string                `json:"id"`
	Processor ProcessorContribution `json:"processor"`
	Variants  []VariantContribution `json:"variants,omitempty"`
}

type Plan struct {
	SchemaVersion    string                 `json:"schemaVersion"`
	Revision         uint64                 `json:"revision"`
	RegistryDigest   string                 `json:"registryDigest"`
	Digest           string                 `json:"digest"`
	SafeMode         bool                   `json:"safeMode,omitempty"`
	Kind             string                 `json:"kind"`
	Purpose          string                 `json:"purpose"`
	Permission       string                 `json:"permission"`
	Actor            Actor                  `json:"-"`
	Source           SourceAsset            `json:"source"`
	Upload           UploadFacts            `json:"upload"`
	Policy           MIMEPolicyContribution `json:"policy"`
	Steps            []PlanStep             `json:"steps"`
	Conflicts        []ProviderConflict     `json:"conflicts,omitempty"`
	OriginalFallback SourceAsset            `json:"originalFallback"`
}

type RuntimeAdmission interface {
	Available(Artifact) bool
	Acquire(context.Context, Artifact) (RuntimeLease, error)
}

// RuntimeLease must prove the exact artifact, not only the extension id. Its
// Context is an independent runtime-drain signal and must remain usable after
// Acquire returns until Release; it must not merely alias the Acquire context.
type RuntimeLease interface {
	Context() context.Context
	Artifact() Artifact
	Release()
}

// ExecutionLimits are Host configuration, never extension declarations.
// OperationTimeout bounds the complete ExecuteOperation call while CallTimeout
// bounds admission plus the provider callback. MaxConcurrentCalls remains held
// until a non-cooperative callback actually exits.
type ExecutionLimits struct {
	OperationTimeout   time.Duration
	CallTimeout        time.Duration
	MaxConcurrentCalls int
}

type BackgroundOperation struct {
	SchemaVersion string
	Key           string
	StepID        string
	Attempt       int
	Plan          Plan
	Prerequisites OperationPrerequisites
}

// ProviderSource is the minimum immutable source description exposed to a
// processor. Original filenames and actor identity remain Host-private.
type ProviderSource struct {
	ID        string `json:"id"`
	Digest    string `json:"digest"`
	Kind      string `json:"kind"`
	MIME      string `json:"mime"`
	SizeBytes int64  `json:"sizeBytes"`
	Immutable bool   `json:"immutable"`
}

type Invocation struct {
	OperationKey string         `json:"operationKey"`
	Attempt      int            `json:"attempt"`
	PlanKind     string         `json:"planKind"`
	Purpose      string         `json:"purpose"`
	Source       ProviderSource `json:"source"`
	Budget       Budget         `json:"budget"`
	Step         PlanStep       `json:"step"`
}

type VariantOutput struct {
	Name         string `json:"name"`
	Handle       string `json:"handle"`
	Digest       string `json:"digest"`
	SourceDigest string `json:"sourceDigest"`
	MIME         string `json:"mime"`
	SizeBytes    int64  `json:"sizeBytes"`
}

type ProviderOutput struct {
	Decision    string            `json:"decision,omitempty"`
	ReasonCode  string            `json:"reasonCode,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Variants    []VariantOutput   `json:"variants,omitempty"`
	CDNURL      string            `json:"cdnUrl,omitempty"`
	RetainUntil time.Time         `json:"retainUntil,omitempty"`
}

// ProviderInvoker must treat OperationKey as the stable idempotency identity
// across attempts and uncertain Host completion. The Host lease prevents live
// duplicates, but local CAS cannot make a remote side effect exactly-once if
// the process dies after that effect and before terminal receipt commit.
type ProviderInvoker interface {
	Invoke(context.Context, Invocation) (ProviderOutput, error)
}

type RetryDecision struct {
	Class       string        `json:"class"`
	Retry       bool          `json:"retry"`
	Delay       time.Duration `json:"delay,omitempty"`
	NextAttempt int           `json:"nextAttempt,omitempty"`
}

type ExecutionResult struct {
	OperationKey     string           `json:"operationKey"`
	StepID           string           `json:"stepId"`
	Output           ProviderOutput   `json:"output"`
	Receipt          OperationReceipt `json:"receipt"`
	FallbackOriginal bool             `json:"fallbackOriginal,omitempty"`
	Skipped          bool             `json:"skipped,omitempty"`
	Replayed         bool             `json:"replayed,omitempty"`
	Retry            RetryDecision    `json:"retry"`
}
