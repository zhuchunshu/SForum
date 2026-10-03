package contentregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
)

const (
	HostPostBodyTargetID              = "sforum.core.content.post-body"
	HostPostBodyContractVersion       = "sforum.core.content.post-body@1"
	HostPostBodySchema                = "sforum.core.content.post-body.schema@1"
	hostPostBodyCoreExtensionID       = "core.content"
	hostPostBodyCoreExtensionVersion  = "1.0.0"
	hostPostBodyCoreHandler           = "host.content.post-body"
	forumPostBodyPolicyFingerprint    = "forum.post-body.public@1"
	defaultForumPostBodyTraceCapacity = 512
)

// ForumPostBodyCorePublication is retained through restart and Safe Mode. Its
// digest is derived only from fixed Host contract material.
func ForumPostBodyCorePublication() (Publication, error) {
	material := strings.Join([]string{
		HostPostBodyTargetID, HostPostBodyContractVersion, HostPostBodySchema,
		hostPostBodyCoreHandler,
	}, "\x00")
	digest := sha256.Sum256([]byte(material))
	artifact, err := NewCoreArtifact(
		hostPostBodyCoreExtensionID,
		hostPostBodyCoreExtensionVersion,
		hex.EncodeToString(digest[:]),
	)
	if err != nil {
		return Publication{}, err
	}
	return Publication{Artifact: artifact, Content: []Declaration{{
		ID: HostPostBodyTargetID, ContractVersion: HostPostBodyContractVersion,
		Kind: KindBlock, Handler: hostPostBodyCoreHandler, Schema: HostPostBodySchema,
	}}}, nil
}

// ContentProviderResolver creates Host-owned adapters for one exact active
// declaration. Implementations dispatch through Executor's acquired lease.
type ContentProviderResolver interface {
	ResolveContentProviders(Contribution) (ProviderSet, error)
}

type ContentProviderResolverFunc func(Contribution) (ProviderSet, error)

func (f ContentProviderResolverFunc) ResolveContentProviders(contribution Contribution) (ProviderSet, error) {
	if f == nil {
		return ProviderSet{}, ErrRuntimeUnavailable
	}
	return f(contribution)
}

type ForumPostFilterConfig struct {
	Registry  *Registry
	Admission RuntimeAdmission
	Providers ContentProviderResolver
	Limits    ExecutionLimits
	Trace     *ContentTraceRing
}

type ForumPostFilter struct {
	registry  *Registry
	admission RuntimeAdmission
	providers ContentProviderResolver
	limits    ExecutionLimits
	trace     *ContentTraceRing

	mu       sync.Mutex
	revision uint64
	executor *Executor
}

// NewForumPostFilter is identity-only for compatibility and focused tests.
// Production uses NewProductionForumPostFilter so declarations without an
// exact Protocol invoker cannot be reported as executed.
func NewForumPostFilter(registry *Registry) *ForumPostFilter {
	return &ForumPostFilter{registry: registry}
}

func NewProductionForumPostFilter(config ForumPostFilterConfig) (*ForumPostFilter, error) {
	if config.Registry == nil || config.Admission == nil || config.Providers == nil {
		return nil, ErrExecutionInvalid
	}
	target, err := config.Registry.Resolve(HostPostBodyTargetID)
	if err != nil || !IsHostCoreArtifact(target.Artifact) || target.ContractVersion != HostPostBodyContractVersion ||
		target.Schema != HostPostBodySchema || target.Handler != hostPostBodyCoreHandler {
		return nil, fmt.Errorf("%w: Forum post-body Core target", ErrCompositionInvalid)
	}
	if _, err := normalizeExecutionLimits(config.Limits); err != nil {
		return nil, err
	}
	trace := config.Trace
	if trace == nil {
		trace = NewContentTraceRing(defaultForumPostBodyTraceCapacity)
	}
	filter := &ForumPostFilter{
		registry: config.Registry, admission: config.Admission, providers: config.Providers,
		limits: config.Limits, trace: trace,
	}
	if filter.HasFilterContributions() {
		if _, err := filter.executorForCurrentRevision(); err != nil {
			return nil, err
		}
	}
	return filter, nil
}

func (f *ForumPostFilter) AfterHostRender(
	ctx context.Context,
	html, plain, resource, resourceID, scope string,
) (nextHTML, nextPlain string, err error) {
	if f == nil || f.registry == nil || ctx == nil {
		return html, plain, nil
	}
	if err := ctx.Err(); err != nil {
		return html, plain, err
	}
	if !f.HasFilterContributions() {
		return html, plain, nil
	}
	if f.admission == nil || f.providers == nil {
		return html, plain, ErrRuntimeUnavailable
	}
	executor, err := f.executorForCurrentRevision()
	if err != nil {
		f.recordConstructionFallback(err)
		return html, plain, nil
	}
	value, err := json.Marshal(map[string]any{
		"html": html, "plain": plain, "resource": resource,
	})
	if err != nil {
		return html, plain, nil
	}
	request := ExecutionRequest{
		TargetID: HostPostBodyTargetID, ContractVersion: HostPostBodyContractVersion,
		Document: EditorDocument{
			SchemaVersion: EditorDocumentSchemaVersion, ContentID: HostPostBodyTargetID,
			ContractVersion: HostPostBodyContractVersion, Schema: HostPostBodySchema,
			StorageVersion: "1", Value: value,
		},
		Permission: PermissionInput{
			ActorFingerprint: "public", PolicyFingerprint: forumPostBodyPolicyFingerprint,
			Recheck: forumPostBodyPermissionRecheck{},
		},
		ResourceID: resourceID, Locale: "und", Scope: scope,
	}
	result, err := executor.Execute(ctx, request)
	if err != nil {
		if ctx.Err() != nil {
			return html, plain, ctx.Err()
		}
		return html, plain, nil
	}
	output := renderSegmentsHTML(result.Render)
	if strings.TrimSpace(output) == "" && strings.TrimSpace(html) != "" {
		return html, plain, nil
	}
	return output, plain, nil
}

func (f *ForumPostFilter) executorForCurrentRevision() (*Executor, error) {
	snapshot := f.registry.Snapshot()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.executor != nil && f.revision == snapshot.Revision {
		return f.executor, nil
	}
	target, err := exactForumPostBodyTarget(snapshot)
	if err != nil {
		return nil, err
	}
	bindings := []ExecutionBinding{{
		TargetID: target.ID, TargetContractVersion: target.ContractVersion,
		DeclarationID: target.ID, ContractVersion: target.ContractVersion,
		Artifact: target.Artifact, Action: ActionAdd, Fallback: FallbackClosed,
		Providers: ProviderSet{Renderer: RendererProviderFunc(renderForumPostBody)},
	}}
	for _, contribution := range snapshot.Content {
		if contribution.Kind != KindRenderFilter && contribution.Kind != KindSanitizer {
			continue
		}
		if strings.TrimSpace(contribution.Handler) == "" {
			return nil, fmt.Errorf("%w: active filter %s has no Protocol handler", ErrContractInsufficient, contribution.ID)
		}
		providers, resolveErr := f.providers.ResolveContentProviders(contribution)
		if resolveErr != nil || providers.Filter == nil {
			return nil, errors.Join(ErrRuntimeUnavailable, resolveErr)
		}
		bindings = append(bindings, ExecutionBinding{
			TargetID: target.ID, TargetContractVersion: target.ContractVersion,
			DeclarationID: contribution.ID, ContractVersion: contribution.ContractVersion,
			Artifact: contribution.Artifact, Action: ActionFilter, Priority: 0,
			Fallback: FallbackBase, Providers: ProviderSet{Filter: providers.Filter},
		})
	}
	next, err := NewExecutor(
		f.registry, bindings, f.admission, forumPostBodySchemaValidator{}, f.limits,
		WithContentTraceSink(f.trace),
	)
	if err != nil {
		return nil, err
	}
	previous := f.executor
	f.executor = next
	f.revision = snapshot.Revision
	if previous != nil {
		previous.Close()
	}
	return next, nil
}

func exactForumPostBodyTarget(snapshot Snapshot) (Contribution, error) {
	for _, target := range snapshot.Content {
		if target.ID == HostPostBodyTargetID && target.ContractVersion == HostPostBodyContractVersion &&
			target.Schema == HostPostBodySchema && target.Handler == hostPostBodyCoreHandler &&
			IsHostCoreArtifact(target.Artifact) {
			return target, nil
		}
	}
	return Contribution{}, ErrContractStale
}

func renderForumPostBody(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
	var input struct {
		HTML     string `json:"html"`
		Plain    string `json:"plain"`
		Resource string `json:"resource"`
	}
	if err := json.Unmarshal(request.Document.Value, &input); err != nil {
		return RenderSegments{}, ErrExecutionInvalid
	}
	return RenderSegments{
		SchemaVersion: RenderSegmentsSchemaVersion, ContentID: request.Target.ID,
		ContractVersion: request.Target.ContractVersion,
		Segments:        []RenderSegment{{Kind: SegmentHTML, HTML: input.HTML}},
	}, nil
}

type forumPostBodySchemaValidator struct{}

func (forumPostBodySchemaValidator) ValidateContentSchema(_ context.Context, request SchemaValidationRequest) error {
	if request.SchemaRef != HostPostBodySchema || request.ContentID != HostPostBodyTargetID ||
		request.ContractVersion != HostPostBodyContractVersion {
		return ErrSchemaRejected
	}
	value, ok := request.Value.(map[string]any)
	if !ok || len(value) != 3 {
		return ErrSchemaRejected
	}
	html, htmlOK := value["html"].(string)
	plain, plainOK := value["plain"].(string)
	resource, resourceOK := value["resource"].(string)
	if !htmlOK || !plainOK || !resourceOK || (resource != "topic" && resource != "comment") ||
		len(html) > hardMaxInputBytes || len(plain) > hardMaxInputBytes {
		return ErrSchemaRejected
	}
	return nil
}

type forumPostBodyPermissionRecheck struct{}

func (forumPostBodyPermissionRecheck) AuthorizeContent(_ context.Context, claim PermissionClaim) error {
	if !IsExactPermissionClaim(claim) || claim.TargetID != HostPostBodyTargetID ||
		claim.TargetContractVersion != HostPostBodyContractVersion || claim.TargetSchema != HostPostBodySchema ||
		!IsHostCoreArtifact(claim.TargetArtifact) || (claim.Scope != "public" && claim.Scope != "author") ||
		(claim.Operation != OperationSource && claim.Operation != OperationRenderer &&
			claim.Operation != OperationFilter && claim.Operation != OperationRelease) || claim.ResourceID == "" {
		return ErrExecutionDenied
	}
	return nil
}

func renderSegmentsHTML(render RenderSegments) string {
	var output strings.Builder
	for _, segment := range render.Segments {
		if segment.Kind == SegmentHTML {
			output.WriteString(segment.HTML)
		} else {
			output.WriteString(segment.Text)
		}
	}
	return output.String()
}

func (f *ForumPostFilter) recordConstructionFallback(err error) {
	if f == nil || f.trace == nil {
		return
	}
	outcome := TraceFailed
	if errors.Is(err, ErrContractStale) {
		outcome = TraceStale
	}
	event := ContentTraceEvent{
		Revision: f.registry.Revision(), TargetID: HostPostBodyTargetID,
		ContentID: HostPostBodyTargetID, ContractVersion: HostPostBodyContractVersion,
		Action: ActionFilter, Operation: OperationFilter, Outcome: outcome,
		Fallback: FallbackBase,
	}
	for _, contribution := range f.registry.Snapshot().Content {
		if contribution.Kind == KindRenderFilter || contribution.Kind == KindSanitizer {
			event.ContentID = contribution.ID
			event.ContractVersion = contribution.ContractVersion
			event.Artifact = contribution.Artifact
			break
		}
	}
	appendContentTraceSafely(f.trace, event)
}

func (f *ForumPostFilter) HasFilterContributions() bool {
	if f == nil || f.registry == nil {
		return false
	}
	for _, contribution := range f.registry.Snapshot().Content {
		if contribution.Kind == KindRenderFilter || contribution.Kind == KindSanitizer {
			return true
		}
	}
	return false
}

func (f *ForumPostFilter) ContentTraces(limit int) []ContentTraceRecord {
	if f == nil || f.trace == nil {
		return nil
	}
	return f.trace.ContentTracesForTarget(HostPostBodyTargetID, limit)
}

var _ SchemaValidator = forumPostBodySchemaValidator{}
var _ PermissionRecheck = forumPostBodyPermissionRecheck{}
