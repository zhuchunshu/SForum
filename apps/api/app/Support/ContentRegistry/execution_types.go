package contentregistry

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

const (
	ExecutionSchemaVersion        = "sforum.content-execution@1"
	EditorDocumentSchemaVersion   = "sforum.editor-document@1"
	SerializedSchemaVersion       = "sforum.serialized-content@1"
	RenderSegmentsSchemaVersion   = "sforum.render-segments@1"
	RenderTextEncodingHTMLEscaped = "html-escaped"
)

const (
	ActionAdd     = "add"
	ActionBefore  = "before"
	ActionAfter   = "after"
	ActionWrap    = "wrap"
	ActionReplace = "replace"
	ActionHide    = "hide"
	ActionFilter  = "filter"
)

const (
	FallbackClosed         = "closed"
	FallbackOmit           = "omit"
	FallbackBase           = "base"
	FallbackPreserveSource = "preserve_source"
)

const (
	SegmentHTML        = "html"
	SegmentText        = "text"
	SegmentUnsupported = "unsupported"
)

const (
	OperationEditor     = "editor"
	OperationValidator  = "validator"
	OperationSerializer = "serializer"
	OperationRenderer   = "renderer"
	OperationFilter     = "filter"
	OperationHide       = "hide"
	OperationSource     = "source"
	OperationRelease    = "release"
)

var (
	ErrExecutionInvalid     = errors.New("content execution request is invalid")
	ErrExecutionDenied      = errors.New("content execution permission recheck denied")
	ErrRuntimeUnavailable   = errors.New("content execution exact runtime is unavailable")
	ErrRuntimeQuarantined   = errors.New("content execution exact runtime is quarantined")
	ErrContractStale        = errors.New("content execution contract is stale")
	ErrSchemaRejected       = errors.New("content execution schema rejected the document")
	ErrExecutionTimeout     = errors.New("content execution exceeded its deadline")
	ErrProviderPanic        = errors.New("content execution provider panicked")
	ErrExecutionLimit       = errors.New("content execution exceeded a bounded limit")
	ErrProviderFailed       = errors.New("content execution provider failed")
	ErrFallbackUnavailable  = errors.New("content execution fallback is unavailable")
	ErrCompositionInvalid   = errors.New("content execution composition is invalid")
	ErrContractInsufficient = errors.New("content manifest contract is insufficient for the requested execution semantics")
)

// ExecutionLimits are Host configuration, never extension input. CallTimeout
// bounds the complete Execute call as well as each individual provider call.
// MaxConcurrentCalls independently bounds runtime-owned work and Host policy/
// schema callbacks so validation nested under an exact runtime lease cannot
// deadlock a one-slot deployment. Values above the hard ceilings are rejected
// so tuning cannot accidentally remove a process safety boundary.
type ExecutionLimits struct {
	MaxInputBytes      int
	MaxOutputBytes     int
	MaxJSONDepth       int
	MaxJSONNodes       int
	MaxSegments        int
	MaxBindings        int
	MaxCacheTags       int
	MaxConcurrentCalls int
	CallTimeout        time.Duration
}

// EditorDocument is the stable storage-facing envelope. Value remains typed by
// the declaration schema; the Host canonicalizes and bounds it before calls.
type EditorDocument struct {
	SchemaVersion   string          `json:"schemaVersion"`
	ContentID       string          `json:"contentId"`
	ContractVersion string          `json:"contractVersion"`
	Schema          string          `json:"schema"`
	StorageVersion  string          `json:"storageVersion"`
	Value           json.RawMessage `json:"value"`
}

type SerializedContent struct {
	SchemaVersion   string `json:"schemaVersion"`
	ContentID       string `json:"contentId"`
	ContractVersion string `json:"contractVersion"`
	StorageVersion  string `json:"storageVersion"`
	MediaType       string `json:"mediaType"`
	Data            []byte `json:"data"`
}

// RenderSegment is deliberately small. In provider requests and responses,
// Text contains plain text. ExecutionResult.Render escapes Text exactly once
// and sets RenderSegments.TextEncoding, so SSR consumers must append Text as
// already escaped HTML text rather than escaping it again. HTML always passes
// through the Host sanitizer and is unaffected by the text encoding contract.
type RenderSegment struct {
	Kind string `json:"kind"`
	HTML string `json:"html,omitempty"`
	Text string `json:"text,omitempty"`
}

type RenderSegments struct {
	SchemaVersion   string          `json:"schemaVersion"`
	ContentID       string          `json:"contentId"`
	ContractVersion string          `json:"contractVersion"`
	TextEncoding    string          `json:"textEncoding,omitempty"`
	Segments        []RenderSegment `json:"segments"`
	// PlainText is unescaped extraction data for excerpts/search, never HTML.
	PlainText string `json:"plainText"`
}

type SchemaValidationPhase string

const (
	SchemaPhaseInput  SchemaValidationPhase = "input"
	SchemaPhaseOutput SchemaValidationPhase = "output"
)

type SchemaValidationRequest struct {
	SchemaRef       string
	ContentID       string
	ContractVersion string
	Phase           SchemaValidationPhase
	Value           any
}

// SchemaValidator is a Host-owned catalog of already compiled schemas. It must
// be safe for concurrent use, honor context cancellation, keep validation work
// proportional to the already bounded value, and never mutate or retain Value.
// The execution layer never loads package paths on the request path.
// ManifestContent currently declares only the target editor/storage value
// schema; RenderSegments use Host structural validation instead of pretending
// that the same schema describes a different envelope.
type SchemaValidator interface {
	ValidateContentSchema(context.Context, SchemaValidationRequest) error
}

type SchemaValidatorFunc func(context.Context, SchemaValidationRequest) error

func (f SchemaValidatorFunc) ValidateContentSchema(ctx context.Context, request SchemaValidationRequest) error {
	if f == nil {
		return ErrSchemaRejected
	}
	return f(ctx, request)
}

type PermissionClaim struct {
	TargetID              string
	TargetContractVersion string
	TargetSchema          string
	TargetArtifact        Artifact
	ContentID             string
	ContractVersion       string
	Schema                string
	Action                string
	Operation             string
	Artifact              Artifact
	ResourceID            string
	Locale                string
	Scope                 string
}

// PermissionRecheck is mandatory even for anonymous/public rendering. Only
// the Host callback decides whether a resource is visible to the current actor;
// implementations must honor context cancellation and expose no raw actor data.
type PermissionRecheck interface {
	AuthorizeContent(context.Context, PermissionClaim) error
}

type PermissionRecheckFunc func(context.Context, PermissionClaim) error

func (f PermissionRecheckFunc) AuthorizeContent(ctx context.Context, claim PermissionClaim) error {
	if f == nil {
		return ErrExecutionDenied
	}
	return f(ctx, claim)
}

type PermissionInput struct {
	ActorFingerprint  string
	PolicyFingerprint string
	Recheck           PermissionRecheck
}

type AdmissionRequest struct {
	TargetID              string
	TargetContractVersion string
	TargetSchema          string
	TargetArtifact        Artifact
	ContentID             string
	ContractVersion       string
	HandlerReference      string
	RendererReference     string
	Action                string
	Operation             string
	Artifact              Artifact
}

type AdmissionLease interface {
	CallContext() context.Context
	Release()
}

// RuntimeAdmission must verify the exact package digest, version row, runtime
// instance, target tuple, and declared handler reference. Implementations must
// honor context cancellation. Static renderer-only declarations use a separate
// Host renderer and cannot enter ProviderSet as executable callbacks.
type RuntimeAdmission interface {
	AcquireContentExecution(context.Context, AdmissionRequest) (AdmissionLease, error)
}

type EditorProviderRequest struct {
	Target     Contribution
	Provider   Contribution
	Action     string
	Document   EditorDocument
	ResourceID string
	Locale     string
	Scope      string
}

type EditorProvider interface {
	PrepareEditorDocument(context.Context, EditorProviderRequest) (EditorDocument, error)
}

type EditorProviderFunc func(context.Context, EditorProviderRequest) (EditorDocument, error)

func (f EditorProviderFunc) PrepareEditorDocument(ctx context.Context, request EditorProviderRequest) (EditorDocument, error) {
	return f(ctx, request)
}

type ValidatorProviderRequest struct {
	Target     Contribution
	Provider   Contribution
	Action     string
	Document   EditorDocument
	ResourceID string
	Locale     string
	Scope      string
}

type ValidatorProvider interface {
	ValidateEditorDocument(context.Context, ValidatorProviderRequest) error
}

type ValidatorProviderFunc func(context.Context, ValidatorProviderRequest) error

func (f ValidatorProviderFunc) ValidateEditorDocument(ctx context.Context, request ValidatorProviderRequest) error {
	return f(ctx, request)
}

type SerializerProviderRequest struct {
	Target     Contribution
	Provider   Contribution
	Action     string
	Document   EditorDocument
	ResourceID string
	Locale     string
	Scope      string
}

type SerializerProvider interface {
	SerializeEditorDocument(context.Context, SerializerProviderRequest) (SerializedContent, error)
}

type SerializerProviderFunc func(context.Context, SerializerProviderRequest) (SerializedContent, error)

func (f SerializerProviderFunc) SerializeEditorDocument(ctx context.Context, request SerializerProviderRequest) (SerializedContent, error) {
	return f(ctx, request)
}

type RendererProviderRequest struct {
	Target     Contribution
	Provider   Contribution
	Action     string
	Document   EditorDocument
	Serialized SerializedContent
	Inner      RenderSegments
	ResourceID string
	Locale     string
	Scope      string
}

type RendererProvider interface {
	RenderContent(context.Context, RendererProviderRequest) (RenderSegments, error)
}

type RendererProviderFunc func(context.Context, RendererProviderRequest) (RenderSegments, error)

func (f RendererProviderFunc) RenderContent(ctx context.Context, request RendererProviderRequest) (RenderSegments, error) {
	return f(ctx, request)
}

type FilterProviderRequest struct {
	Target     Contribution
	Provider   Contribution
	Action     string
	Document   EditorDocument
	Serialized SerializedContent
	Render     RenderSegments
	ResourceID string
	Locale     string
	Scope      string
}

type FilterProvider interface {
	FilterRenderedContent(context.Context, FilterProviderRequest) (RenderSegments, error)
}

type FilterProviderFunc func(context.Context, FilterProviderRequest) (RenderSegments, error)

func (f FilterProviderFunc) FilterRenderedContent(ctx context.Context, request FilterProviderRequest) (RenderSegments, error) {
	return f(ctx, request)
}

// ProviderSet contains Host-owned adapters, not extension authority. An
// in-process function value cannot prove which subprocess handler produced it.
// Production wiring must dispatch HandlerReference through the exact runtime
// lease returned for AdmissionRequest; unit fixtures may bind functions here.
// Adapters must be concurrent-safe and must not retain or mutate request values
// after returning.
type ProviderSet struct {
	Editor     EditorProvider
	Validator  ValidatorProvider
	Serializer SerializerProvider
	Renderer   RendererProvider
	Filter     FilterProvider
}

// ExecutionBinding is Host-authored execution wiring over a frozen declaration.
// Action, target, priority, fallback, and cache tags are intentionally not
// ManifestContent fields. Lifecycle code must not populate them from invented
// extension metadata until a future manifest contract explicitly adds them.
type ExecutionBinding struct {
	TargetID              string
	TargetContractVersion string
	DeclarationID         string
	ContractVersion       string
	Artifact              Artifact
	Action                string
	Priority              int
	Fallback              string
	CacheTags             []string
	Providers             ProviderSet
}

type ExecutionRequest struct {
	TargetID        string
	ContractVersion string
	Document        EditorDocument
	Permission      PermissionInput
	ResourceID      string
	Locale          string
	Scope           string
	CacheTags       []string
}

type Attribution struct {
	ContentID       string   `json:"contentId"`
	ContractVersion string   `json:"contractVersion"`
	Action          string   `json:"action"`
	Priority        int      `json:"priority"`
	Artifact        Artifact `json:"artifact"`
}

// ExecutionResult is the public release envelope. EditorDocument and
// SerializedContent are deliberately absent: storage/source data stays inside
// Execute and only Host-sanitized SSR segments cross this boundary.
type ExecutionResult struct {
	SchemaVersion   string         `json:"schemaVersion"`
	Revision        uint64         `json:"revision"`
	Digest          string         `json:"digest"`
	Render          RenderSegments `json:"render"`
	CacheKey        string         `json:"cacheKey"`
	CacheTags       []string       `json:"cacheTags"`
	Attribution     []Attribution  `json:"attribution"`
	FallbackUsed    bool           `json:"fallbackUsed,omitempty"`
	SourcePreserved bool           `json:"sourcePreserved,omitempty"`
	Hidden          bool           `json:"hidden,omitempty"`
}
