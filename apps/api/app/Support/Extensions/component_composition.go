package extensionsruntime

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultComponentCompositionTimeout     = 2 * time.Second
	DefaultComponentCompositionMaxTimeout  = 5 * time.Second
	DefaultComponentCompositionMaxDepth    = 16
	DefaultComponentCompositionMaxSegments = 256
	DefaultComponentCompositionMaxBytes    = 1 << 20
	DefaultComponentCompositionMaxCalls    = 32
	DefaultComponentCompositionTraceLimit  = 256
)

const (
	ComponentRenderEncodingEscapedText   = "html-escaped-text"
	ComponentRenderEncodingSanitizedHTML = "sanitized-html"
)

var (
	ErrComponentCompositionInvalid      = errors.New("component composition request is invalid")
	ErrComponentCompositionStale        = errors.New("component composition snapshot is stale")
	ErrComponentCompositionUnauthorized = errors.New("component composition artifact is not authorized")
	ErrComponentCompositionTimeout      = errors.New("component composition call timed out")
	ErrComponentCompositionCrash        = errors.New("component composition renderer crashed")
	ErrComponentCompositionCycle        = errors.New("component composition cycle detected")
	ErrComponentCompositionDepth        = errors.New("component composition depth exceeded")
	ErrComponentCompositionOutput       = errors.New("component composition output exceeds Host bounds")
	ErrComponentCompositionMutation     = errors.New("component composition changed a protected field")
	ErrComponentCompositionSEO          = errors.New("component composition removed primary SSR content")
	ErrComponentCompositionBusy         = errors.New("component composition renderer capacity is exhausted")
)

// ComponentDocumentValidator is supplied by the owner of the target contract.
// Core and plugin business schemas therefore remain authoritative; the
// composition executor does not infer a wider contract from a contribution.
type ComponentDocumentValidator func(context.Context, map[string]any) error

type ComponentCompositionContract struct {
	ValidateProps        ComponentDocumentValidator
	ValidateResult       ComponentDocumentValidator
	MutablePropsFields   []string
	MutableResultFields  []string
	RetainPrimaryContent bool
}

type ComponentRenderFragment struct {
	// Text is ordinary text and is escaped by html/template before release.
	Text string `json:"text,omitempty"`
	// ReviewedHTML is still untrusted transport input. The Host sanitizes it
	// with its component HTML policy before it can become a render segment.
	ReviewedHTML   string `json:"reviewedHtml,omitempty"`
	PrimaryContent bool   `json:"primaryContent,omitempty"`

	safeHTML string
	encoding string
}

// ComponentRenderResponse contains only data a renderer may produce. Owner,
// contract, action, order, and fallback attribution are assigned by the Host.
type ComponentRenderResponse struct {
	Artifact  HookArtifact              `json:"artifact"`
	Document  map[string]any            `json:"document,omitempty"`
	Fragments []ComponentRenderFragment `json:"fragments,omitempty"`
}

type ComponentRenderCall struct {
	TargetID              string                   `json:"targetId"`
	TargetContractVersion string                   `json:"targetContractVersion"`
	Contribution          ComponentContribution    `json:"contribution"`
	Artifact              HookArtifact             `json:"artifact"`
	Props                 map[string]any           `json:"props"`
	Result                map[string]any           `json:"result"`
	Children              []ComponentRenderSegment `json:"children,omitempty"`
	Depth                 int                      `json:"depth"`
}

type ComponentSSRRenderer interface {
	RenderComponent(context.Context, ComponentRenderCall) (ComponentRenderResponse, error)
}

type ComponentSSRRendererFunc func(context.Context, ComponentRenderCall) (ComponentRenderResponse, error)

func (f ComponentSSRRendererFunc) RenderComponent(
	ctx context.Context,
	call ComponentRenderCall,
) (ComponentRenderResponse, error) {
	return f(ctx, call)
}

type ComponentFallbackCall struct {
	TargetID              string         `json:"targetId"`
	TargetContractVersion string         `json:"targetContractVersion"`
	Props                 map[string]any `json:"props"`
	Depth                 int            `json:"depth"`
}

type ComponentFallbackRenderer func(context.Context, ComponentFallbackCall) (ComponentRenderResponse, error)

type ComponentTargetBinding struct {
	Contract ComponentCompositionContract
	Fallback ComponentFallbackRenderer
}

type ComponentTargetBindingResolver func(
	context.Context,
	ComponentTarget,
) (ComponentTargetBinding, error)

type ComponentRuntimeAdmissionRequest struct {
	Revision              uint64       `json:"revision"`
	TargetID              string       `json:"targetId"`
	TargetContractVersion string       `json:"targetContractVersion"`
	ContributionID        string       `json:"contributionId"`
	Action                string       `json:"action"`
	Artifact              HookArtifact `json:"artifact"`
}

// ComponentRuntimeAdmissionLease is exact-artifact authority for one
// composition. Release must be idempotent. Validate is the final Host-owned
// trust fence and must recheck the same artifact/revision represented by the
// acquisition request.
type ComponentRuntimeAdmissionLease interface {
	Context() context.Context
	Validate(context.Context) error
	Release()
}

type ComponentRuntimeAdmission interface {
	AcquireComponentRuntime(context.Context, ComponentRuntimeAdmissionRequest) (ComponentRuntimeAdmissionLease, error)
}

// ComponentRendererTermination is a Host-owned, non-blocking signal to the
// process adapter. It does not grant permission to release a runtime lease:
// the lease remains held until RenderComponent really returns.
type ComponentRendererTermination struct {
	Request ComponentRuntimeAdmissionRequest
	Cause   error
}

type ComponentRendererTerminator interface {
	TerminateComponentCall(ComponentRendererTermination)
}

type ComponentCallPolicy struct {
	FailurePolicy string
	Timeout       time.Duration
}

type ComponentCallPolicyResolver func(ComponentContribution) ComponentCallPolicy

type ComponentCompositionExecutorConfig struct {
	Registry           *ComponentRegistry
	Renderer           ComponentSSRRenderer
	ResolveTarget      ComponentTargetBindingResolver
	Admission          ComponentRuntimeAdmission
	Terminator         ComponentRendererTerminator
	ResolvePolicy      ComponentCallPolicyResolver
	DefaultTimeout     time.Duration
	MaxTimeout         time.Duration
	MaxDepth           int
	MaxSegments        int
	MaxOutputBytes     int
	MaxConcurrentCalls int
	TraceLimit         int
}

type ComponentCompositionRequest struct {
	TargetID              string
	TargetContractVersion string
	ExpectedRevision      uint64
	Props                 map[string]any
	Binding               ComponentTargetBinding
}

type ComponentFallbackEvidence struct {
	ContributionID string `json:"contributionId,omitempty"`
	Action         string `json:"action,omitempty"`
	Reason         string `json:"reason"`
	FailurePolicy  string `json:"failurePolicy,omitempty"`
}

// ComponentRenderSegment is an SSR-first render tree. Children on wrap
// segments retain nested composition evidence; browser L2 identifiers are
// descriptive only and are never invoked by this executor.
type ComponentRenderSegment struct {
	OwnerID         string `json:"ownerId"`
	ComponentID     string `json:"componentId"`
	ContractVersion string `json:"contractVersion"`
	Action          string `json:"action"`
	Order           int    `json:"order"`
	Depth           int    `json:"depth"`
	// HTML is safe for an HTML text node/fragment according to Encoding. It is
	// never copied directly from a renderer response.
	HTML           string                      `json:"html"`
	Encoding       string                      `json:"encoding"`
	PrimaryContent bool                        `json:"primaryContent,omitempty"`
	L2Component    string                      `json:"l2Component,omitempty"`
	Artifact       *HookArtifact               `json:"artifact,omitempty"`
	Fallback       []ComponentFallbackEvidence `json:"fallback,omitempty"`
	Children       []ComponentRenderSegment    `json:"children,omitempty"`
}

type ComponentCompositionResult struct {
	Revision uint64                   `json:"revision"`
	Target   ComponentTarget          `json:"target"`
	Props    map[string]any           `json:"props"`
	Result   map[string]any           `json:"result"`
	Segments []ComponentRenderSegment `json:"segments"`
	Hidden   bool                     `json:"hidden"`
	TraceID  string                   `json:"traceId"`
}

type ComponentCompositionTraceStep struct {
	Sequence       int          `json:"sequence"`
	TargetID       string       `json:"targetId"`
	ContributionID string       `json:"contributionId,omitempty"`
	Action         string       `json:"action"`
	Artifact       HookArtifact `json:"artifact"`
	Status         string       `json:"status"`
	FailurePolicy  string       `json:"failurePolicy,omitempty"`
	TimeoutMS      int64        `json:"timeoutMs,omitempty"`
	FallbackReason string       `json:"fallbackReason,omitempty"`
	DurationMicros int64        `json:"durationMicros"`
}

type ComponentCompositionTrace struct {
	ID                    string                          `json:"id"`
	Revision              uint64                          `json:"revision"`
	TargetID              string                          `json:"targetId"`
	TargetContractVersion string                          `json:"targetContractVersion"`
	StartedAt             time.Time                       `json:"startedAt"`
	DurationMicros        int64                           `json:"durationMicros"`
	Status                string                          `json:"status"`
	Error                 string                          `json:"error,omitempty"`
	Steps                 []ComponentCompositionTraceStep `json:"steps"`
}

type ComponentCompositionExecutor struct {
	registry       *ComponentRegistry
	renderer       ComponentSSRRenderer
	resolveTarget  ComponentTargetBindingResolver
	admission      ComponentRuntimeAdmission
	terminator     ComponentRendererTerminator
	resolvePolicy  ComponentCallPolicyResolver
	defaultTimeout time.Duration
	maxTimeout     time.Duration
	maxDepth       int
	maxSegments    int
	maxOutputBytes int
	callSlots      chan struct{}
	fallbackSlots  chan struct{}
	traceLimit     int

	traceSequence atomic.Uint64
	traceMu       sync.Mutex
	traces        []ComponentCompositionTrace
}

func NewComponentCompositionExecutor(config ComponentCompositionExecutorConfig) (*ComponentCompositionExecutor, error) {
	if config.Registry == nil || config.Renderer == nil || config.Admission == nil {
		return nil, ErrComponentCompositionInvalid
	}
	config.DefaultTimeout = componentDurationOrDefault(config.DefaultTimeout, DefaultComponentCompositionTimeout)
	config.MaxTimeout = componentDurationOrDefault(config.MaxTimeout, DefaultComponentCompositionMaxTimeout)
	config.MaxDepth = componentIntOrDefault(config.MaxDepth, DefaultComponentCompositionMaxDepth)
	config.MaxSegments = componentIntOrDefault(config.MaxSegments, DefaultComponentCompositionMaxSegments)
	config.MaxOutputBytes = componentIntOrDefault(config.MaxOutputBytes, DefaultComponentCompositionMaxBytes)
	config.MaxConcurrentCalls = componentIntOrDefault(config.MaxConcurrentCalls, DefaultComponentCompositionMaxCalls)
	config.TraceLimit = componentIntOrDefault(config.TraceLimit, DefaultComponentCompositionTraceLimit)
	if config.DefaultTimeout > config.MaxTimeout || config.MaxDepth < 1 || config.MaxSegments < 1 ||
		config.MaxOutputBytes < 1 || config.MaxConcurrentCalls < 1 || config.TraceLimit < 1 {
		return nil, ErrComponentCompositionInvalid
	}
	return &ComponentCompositionExecutor{
		registry: config.Registry, renderer: config.Renderer, resolveTarget: config.ResolveTarget,
		admission: config.Admission, terminator: config.Terminator, resolvePolicy: config.ResolvePolicy,
		defaultTimeout: config.DefaultTimeout, maxTimeout: config.MaxTimeout,
		maxDepth: config.MaxDepth, maxSegments: config.MaxSegments,
		maxOutputBytes: config.MaxOutputBytes, callSlots: make(chan struct{}, config.MaxConcurrentCalls),
		fallbackSlots: make(chan struct{}, config.MaxConcurrentCalls),
		traceLimit:    config.TraceLimit,
	}, nil
}

func componentDurationOrDefault(value, fallback time.Duration) time.Duration {
	if value <= 0 {
		return fallback
	}
	return value
}

func componentIntOrDefault(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}
