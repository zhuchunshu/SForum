package navigationregistry

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestComposerEnforcesActorLocaleAndAllCompositionActions(t *testing.T) {
	registry := New()
	plugin := publication("plugin.navigation", false, 'a')
	main := navigation("plugin.navigation.item.main", NavigationKindItem, ActionAdd, CorePrimaryMenuID, 20)
	main.Label = "Plugin"
	main.Labels = map[string]string{"zh-CN": "插件", "en-US": "Plugin"}
	main.Permission = "forum.read"
	main.Visibility = VisibilityAuthenticated
	replace := navigation("plugin.navigation.item.replace", NavigationKindItem, ActionReplace, main.ID, 0)
	replace.Handler = "plugin.navigation.render.replace"
	filter := navigation("plugin.navigation.item.filter", NavigationKindItem, ActionFilter, main.ID, 0)
	filter.Handler = "plugin.navigation.render.filter"
	plugin.Navigation = []NavigationDeclaration{
		main,
		navigation("plugin.navigation.item.before", NavigationKindItem, ActionBefore, main.ID, 0),
		navigation("plugin.navigation.item.after", NavigationKindItem, ActionAfter, main.ID, 0),
		navigation("plugin.navigation.item.wrap", NavigationKindItem, ActionWrap, main.ID, 0),
		replace,
		filter,
	}
	hideSidebar := navigation("plugin.navigation.sidebar.hide", NavigationKindSidebar, ActionHide, CoreSidebarNavigationID, 0)
	hideSidebar.Permission = "forum.hide_sidebar"
	plugin.Navigation = append(plugin.Navigation, hideSidebar)
	if _, err := registry.ReplaceAll([]Publication{plugin, CorePublication()}); err != nil {
		t.Fatal(err)
	}

	runtime := newCompositionRuntime()
	runtime.outputs[replace.Handler] = RuntimeOutput{Label: "Replacement"}
	runtime.render = func(invocation RuntimeInvocation, output RuntimeOutput) (RuntimeOutput, error) {
		if invocation.Handler == filter.Handler {
			if invocation.Locale == "en-US" {
				return RuntimeOutput{Label: "Filtered EN"}, nil
			}
			return RuntimeOutput{Label: "过滤后"}, nil
		}
		return output, nil
	}
	traces := NewTraceRing(64)
	composer := NewComposer(registry, runtime, runtime, traces)

	denied, err := composer.Compose(t.Context(), CompositionRequest{Locale: "en-US"})
	if err != nil {
		t.Fatal(err)
	}
	if menu := composedByID(denied.Navigation, CorePrimaryMenuID); menu == nil || hasComposedID(menu.Children, main.ID) {
		t.Fatalf("guest saw authenticated permission item: %#v", menu)
	}

	allowed, err := composer.Compose(t.Context(), CompositionRequest{
		Locale: "en_us", Visibility: VisibilityInput{Authenticated: true, Permissions: []string{"forum.read"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	menu := composedByID(allowed.Navigation, CorePrimaryMenuID)
	if menu == nil || len(menu.Children) != 3 ||
		menu.Children[0].ID != "plugin.navigation.item.before" || menu.Children[1].ID != main.ID ||
		menu.Children[2].ID != "plugin.navigation.item.after" || menu.Children[1].Label != "Filtered EN" ||
		len(menu.Children[1].Wrappers) != 1 {
		t.Fatalf("composed menu=%#v", menu)
	}
	if allowed.Locale != "en-US" || allowed.CacheKey == denied.CacheKey {
		t.Fatalf("locale/visibility cache isolation denied=%s allowed=%#v", denied.CacheKey, allowed)
	}

	chinese, err := composer.Compose(t.Context(), CompositionRequest{
		Locale: "zh-CN", Visibility: VisibilityInput{Authenticated: true, Permissions: []string{"forum.read"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if item := composedByID(composedByID(chinese.Navigation, CorePrimaryMenuID).Children, main.ID); item == nil || item.Label != "过滤后" || chinese.CacheKey == allowed.CacheKey {
		t.Fatalf("locale projection leaked cached output: %#v", chinese)
	}

	hidden, err := composer.Compose(t.Context(), CompositionRequest{
		Locale: "zh-CN", Visibility: VisibilityInput{
			Authenticated: true, Permissions: []string{"forum.read", "forum.hide_sidebar"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if composedByID(hidden.Navigation, CoreSidebarNavigationID) != nil {
		t.Fatalf("permission-scoped hide was not enforced: %#v", hidden.Navigation)
	}
	runtime.setUnavailable(plugin.Artifact, true)
	quarantined, err := composer.Compose(t.Context(), CompositionRequest{
		Locale: "zh-CN", Visibility: VisibilityInput{
			Authenticated: true, Permissions: []string{"forum.hide_sidebar", "forum.read"},
		},
	})
	if err != nil || composedByID(quarantined.Navigation, CoreSidebarNavigationID) == nil {
		t.Fatalf("unavailable declarative hide removed Core fallback: %#v err=%v", quarantined.Navigation, err)
	}
	runtime.setUnavailable(plugin.Artifact, false)
	if len(hidden.Regions) != 5 || composedByID(hidden.Regions, CoreWidgetRegionID) == nil {
		t.Fatalf("region families missing: %#v", hidden.Regions)
	}
	inspection, err := NewInspector(registry, traces).Inspect(main.ID, 64)
	if err != nil || len(inspection.Traces) == 0 {
		t.Fatalf("inspector traces=%#v err=%v", inspection, err)
	}
	for _, trace := range inspection.Traces {
		if trace.TargetID != main.ID || trace.Artifact.ExtensionID == "" || trace.ContractVersion == "" {
			t.Fatalf("unattributed trace=%#v", trace)
		}
	}
}

func TestComposerProviderTieSelectionResetFallbackAndStaleRuntime(t *testing.T) {
	registry := New()
	alpha := publication("alpha.navigation", false, 'a')
	alphaReplace := navigation("alpha.navigation.header.replace", NavigationKindHeader, ActionReplace, CoreHeaderNavigationID, 0)
	alphaReplace.Handler = "alpha.navigation.render.header"
	alpha.Navigation = []NavigationDeclaration{alphaReplace}
	beta := publication("beta.navigation", false, 'b')
	betaReplace := navigation("beta.navigation.header.replace", NavigationKindHeader, ActionReplace, CoreHeaderNavigationID, 0)
	betaReplace.Handler = "beta.navigation.render.header"
	beta.Navigation = []NavigationDeclaration{betaReplace}
	if _, err := registry.ReplaceAll([]Publication{beta, CorePublication(), alpha}); err != nil {
		t.Fatal(err)
	}

	runtime := newCompositionRuntime()
	runtime.outputs[alphaReplace.Handler] = RuntimeOutput{Label: "Alpha"}
	runtime.outputs[betaReplace.Handler] = RuntimeOutput{Label: "Beta"}
	traces := NewTraceRing(32)
	composer := NewComposer(registry, runtime, runtime, traces)

	initial := mustCompose(t, composer)
	if header := composedByID(initial.Navigation, CoreHeaderNavigationID); header == nil || header.Label != "Alpha" {
		t.Fatalf("deterministic tie winner=%#v", header)
	}
	selectedRevision, err := registry.SelectProvider(SelectProviderRequest{
		ExpectedRevision: registry.Revision(), Family: ProviderFamilyNavigation, TargetID: CoreHeaderNavigationID,
		Provider: ProviderRef{ContributionID: betaReplace.ID, Artifact: beta.Artifact},
	})
	if err != nil {
		t.Fatal(err)
	}
	selected := mustCompose(t, composer)
	if header := composedByID(selected.Navigation, CoreHeaderNavigationID); header == nil || header.Label != "Beta" || selected.CacheKey == initial.CacheKey {
		t.Fatalf("selected provider/cache=%#v", selected)
	}
	runtime.setUnavailable(beta.Artifact, true)
	if _, err := composer.Compose(t.Context(), CompositionRequest{}); !errors.Is(err, ErrTrustedReplace) {
		t.Fatalf("selected stale runtime must fail closed: %v", err)
	}
	runtime.setUnavailable(beta.Artifact, false)
	if revision, reset, err := registry.ResetProvider(ResetProviderRequest{
		ExpectedRevision: selectedRevision, Family: ProviderFamilyNavigation, TargetID: CoreHeaderNavigationID,
	}); err != nil || !reset || revision != selectedRevision+1 {
		t.Fatalf("reset revision=%d reset=%t err=%v", revision, reset, err)
	}

	runtime.setFailure(alphaReplace.Handler, errors.New("optional crash"))
	fallback := mustCompose(t, composer)
	if header := composedByID(fallback.Navigation, CoreHeaderNavigationID); header == nil || header.Label != "Beta" {
		t.Fatalf("optional replace did not fall through: %#v", fallback.Navigation)
	}
	runtime.setFailure(betaReplace.Handler, errors.New("optional crash"))
	coreFallback := mustCompose(t, composer)
	if header := composedByID(coreFallback.Navigation, CoreHeaderNavigationID); header == nil || header.Label != "页头" {
		t.Fatalf("core fallback missing: %#v", coreFallback.Navigation)
	}

	runtime.setFailure(alphaReplace.Handler, nil)
	runtime.setFailure(betaReplace.Handler, nil)
	if _, err := registry.SelectProvider(SelectProviderRequest{
		ExpectedRevision: registry.Revision(), Family: ProviderFamilyNavigation, TargetID: CoreHeaderNavigationID,
		Provider: ProviderRef{ContributionID: alphaReplace.ID, Artifact: alpha.Artifact},
	}); err != nil {
		t.Fatal(err)
	}
	runtime.setLeaseRuntime(alpha.Artifact, "stale-runtime")
	if _, err := composer.Compose(t.Context(), CompositionRequest{}); !errors.Is(err, ErrTrustedReplace) {
		t.Fatalf("mismatched exact lease must fail closed: %v", err)
	}
	runtime.setLeaseRuntime(alpha.Artifact, alpha.Artifact.RuntimeInstanceID)
	runtime.setCancelOnRender(true)
	if _, err := composer.Compose(t.Context(), CompositionRequest{}); !errors.Is(err, ErrTrustedReplace) || !errors.Is(err, ErrRuntimeUnavailable) {
		t.Fatalf("revoked lease result must fail closed: %v", err)
	}
	runtime.setCancelOnRender(false)
	inspection, err := NewInspector(registry, traces).Inspect(CoreHeaderNavigationID, 32)
	if err != nil || len(inspection.Snapshot.ProviderSelections) != 1 || len(inspection.Traces) == 0 {
		t.Fatalf("selection/trace inspection=%#v err=%v", inspection, err)
	}
}

func TestRuntimeVisibilityRedactsHostControlsAndUnrelatedPermissions(t *testing.T) {
	visibility := VisibilityInput{
		Authenticated: true,
		Permissions:   []string{"plugin.navigation.manage", "other.secret", "forum.read"},
		HiddenIDs:     []string{"core.navigation.header"},
		DisabledProviders: []ProviderRef{{
			ContributionID: "other.provider",
			Artifact:       publication("other.provider", false, 'f').Artifact,
		}},
	}
	got := runtimeVisibilityFor("forum.read", "plugin.navigation.manage", visibility)
	if !got.Authenticated ||
		!reflect.DeepEqual(got.Permissions, []string{"forum.read", "plugin.navigation.manage"}) ||
		len(got.HiddenIDs) != 0 || len(got.DisabledProviders) != 0 {
		t.Fatalf("runtime visibility leaked Host controls or unrelated authority: %#v", got)
	}
}

func TestCompositionRejectsUnsafeDOMAttributes(t *testing.T) {
	item := ComposedItem{
		ID: "core.navigation.item.fixture", ContractVersion: "core.navigation.item.fixture@1",
		ProviderID: "core.navigation.item.fixture", ProviderContractVersion: "core.navigation.item.fixture@1",
		Kind: NavigationKindItem, Label: "Fixture", Href: "/fixture", Artifact: CorePublication().Artifact,
	}
	for _, key := range []string{"onclick", "onerror", "style", "srcdoc", "formaction", "data-foreign"} {
		if _, _, err := applyRuntimeOutput(item, RuntimeOutput{Attributes: map[string]string{key: "unsafe"}}, true); !errors.Is(err, ErrInvalid) {
			t.Fatalf("unsafe runtime attribute %q error=%v", key, err)
		}
	}
	safe, _, err := applyRuntimeOutput(item, RuntimeOutput{Attributes: map[string]string{
		"aria-label": "Fixture", "data-sforum-slot": "primary", "open-in-new-tab": "true",
	}}, true)
	if err != nil || len(safe.Attributes) != 3 {
		t.Fatalf("safe semantic attributes=%#v error=%v", safe.Attributes, err)
	}
	if _, _, err := applyRuntimeOutput(item, RuntimeOutput{Attributes: map[string]string{
		"ARIA-label": "first", "aria-label": "second",
	}}, true); !errors.Is(err, ErrInvalid) {
		t.Fatalf("normalized runtime attribute collision error=%v", err)
	}
	item.Attributes = map[string]string{"onclick": "unsafe"}
	if _, _, err := normalizeBaseChildren(map[string][]ComposedItem{CorePrimaryMenuID: {item}}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unsafe Host base attribute error=%v", err)
	}
}

func TestRegistrySafeModeCoreIdentityDepthAndOutputBounds(t *testing.T) {
	registry := New()
	broken := publication("broken.navigation", false, 'c')
	broken.Navigation = []NavigationDeclaration{{ID: "invalid"}}
	if _, err := registry.ReplaceAllWithSafeMode([]Publication{broken, CorePublication()}, true); err != nil {
		t.Fatalf("safe mode parsed corrupt plugin: %v", err)
	}
	snapshot := registry.Snapshot()
	if !snapshot.SafeMode || len(snapshot.Publications) != 1 || !snapshot.Publications[0].Artifact.Core {
		t.Fatalf("safe snapshot=%#v", snapshot)
	}
	if _, err := registry.Publish(publication("blocked.navigation", false, 'd')); !errors.Is(err, ErrSafeMode) {
		t.Fatalf("safe mode publication=%v", err)
	}
	if _, err := New().Publish(Publication{Artifact: Artifact{
		ExtensionID: CoreNavigationExtensionID, ExtensionVersion: "1.0.0",
		PackageDigest: strings.Repeat("a", 64), ImpactDigest: strings.Repeat("a", 64), Core: true,
	}}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unsealed core authority=%v", err)
	}
	composition, err := NewComposer(registry, nil, nil, nil).Compose(t.Context(), CompositionRequest{})
	if err != nil || !composition.SafeMode || len(composition.Navigation) != 5 || len(composition.Regions) != 5 {
		t.Fatalf("safe mode core composition=%#v err=%v", composition, err)
	}

	deep := publication("deep.navigation", false, 'e')
	parent := ""
	for index := 0; index <= maxTargetDepth; index++ {
		id := "deep.navigation.region." + string(rune('a'+index))
		deep.Regions = append(deep.Regions, region(id, RegionKindWidget, ActionAdd, parent))
		parent = id
	}
	if _, err := New().ReplaceAll([]Publication{deep}); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("depth overflow=%v", err)
	}

	base := make([]ComposedItem, maxComposedItems+1)
	core := CorePublication().Artifact
	for index := range base {
		id := "core.navigation.base.item." + leftPad(index)
		base[index] = ComposedItem{
			ID: id, ContractVersion: id + "@1", ProviderID: id, ProviderContractVersion: id + "@1",
			Kind: NavigationKindItem, Label: id, Href: "/", Artifact: core,
		}
	}
	active := New()
	if _, err := active.Publish(CorePublication()); err != nil {
		t.Fatal(err)
	}
	if _, err := NewComposer(active, nil, nil, nil).Compose(t.Context(), CompositionRequest{
		BaseNavigationChildren: map[string][]ComposedItem{CorePrimaryMenuID: base},
	}); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("item overflow=%v", err)
	}

	bounded := publication("bounded.output", false, 'f')
	replace := region("bounded.output.widget.replace", RegionKindWidget, ActionReplace, CoreWidgetRegionID)
	replace.Handler = "bounded.output.render.widget"
	bounded.Regions = []RegionDeclaration{replace}
	boundedRegistry := New()
	if _, err := boundedRegistry.ReplaceAll([]Publication{CorePublication(), bounded}); err != nil {
		t.Fatal(err)
	}
	runtime := newCompositionRuntime()
	runtime.outputs[replace.Handler] = RuntimeOutput{Content: strings.Repeat("x", maxRuntimeContentRunes+1)}
	boundedComposer := NewComposer(boundedRegistry, runtime, runtime, nil)
	optional, err := boundedComposer.Compose(t.Context(), CompositionRequest{})
	if err != nil || composedByID(optional.Regions, CoreWidgetRegionID) == nil {
		t.Fatalf("bounded optional output did not retain Core fallback: %#v err=%v", optional, err)
	}
	if _, err := boundedRegistry.SelectProvider(SelectProviderRequest{
		ExpectedRevision: boundedRegistry.Revision(), Family: ProviderFamilyRegion, TargetID: CoreWidgetRegionID,
		Provider: ProviderRef{ContributionID: replace.ID, Artifact: bounded.Artifact},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := boundedComposer.Compose(t.Context(), CompositionRequest{}); !errors.Is(err, ErrTrustedReplace) || !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("selected oversized output must fail closed with limit evidence: %v", err)
	}

	oversizedHref := publication("bounded.href", false, 'b')
	link := navigation("bounded.href.item.link", NavigationKindItem, ActionAdd, "", 0)
	link.Href = "/" + strings.Repeat("x", maxNavigationHrefRunes)
	oversizedHref.Navigation = []NavigationDeclaration{link}
	if _, err := New().Publish(oversizedHref); !errors.Is(err, ErrInvalid) {
		t.Fatalf("oversized navigation href error=%v", err)
	}
}

func TestInspectorFiltersBeforeApplyingTargetLimit(t *testing.T) {
	registry := New()
	if _, err := registry.Publish(CorePublication()); err != nil {
		t.Fatal(err)
	}
	ring := NewTraceRing(8)
	artifact := CorePublication().Artifact
	target := CorePrimaryMenuID
	ring.AppendNavigationTrace(TraceEvent{
		Revision: registry.Revision(), Family: ProviderFamilyNavigation, TargetID: target,
		ContributionID: target, ContractVersion: target + "@1", Action: ActionAdd,
		Locale: "zh-CN", Outcome: TraceSucceeded, Artifact: artifact,
	})
	for _, unrelated := range []string{CoreHeaderNavigationID, CoreFooterNavigationID, CoreSidebarNavigationID} {
		ring.AppendNavigationTrace(TraceEvent{
			Revision: registry.Revision(), Family: ProviderFamilyNavigation, TargetID: unrelated,
			ContributionID: unrelated, ContractVersion: unrelated + "@1", Action: ActionAdd,
			Locale: "zh-CN", Outcome: TraceSucceeded, Artifact: artifact,
		})
	}
	inspection, err := NewInspector(registry, ring).Inspect(target, 1)
	if err != nil || len(inspection.Traces) != 1 || inspection.Traces[0].TargetID != target {
		t.Fatalf("target trace limit applied before filter: %#v err=%v", inspection.Traces, err)
	}
}

type compositionRuntime struct {
	mu           sync.RWMutex
	unavailable  map[string]bool
	leaseRuntime map[string]string
	outputs      map[string]RuntimeOutput
	failures     map[string]error
	render       func(RuntimeInvocation, RuntimeOutput) (RuntimeOutput, error)
	cancelRender bool
	leaseCancel  context.CancelFunc
}

func newCompositionRuntime() *compositionRuntime {
	return &compositionRuntime{
		unavailable: map[string]bool{}, leaseRuntime: map[string]string{},
		outputs: map[string]RuntimeOutput{}, failures: map[string]error{},
	}
}

func (r *compositionRuntime) Available(artifact Artifact) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return !r.unavailable[artifactKey(artifact)]
}

func (r *compositionRuntime) Acquire(ctx context.Context, artifact Artifact) (RuntimeLease, error) {
	if !r.Available(artifact) {
		return nil, ErrRuntimeUnavailable
	}
	r.mu.Lock()
	runtimeID := r.leaseRuntime[artifactKey(artifact)]
	leaseCtx, cancel := context.WithCancel(ctx)
	r.leaseCancel = cancel
	r.mu.Unlock()
	if runtimeID == "" {
		runtimeID = artifact.RuntimeInstanceID
	}
	return &compositionLease{ctx: leaseCtx, runtimeID: runtimeID, cancel: cancel}, nil
}

func (r *compositionRuntime) RenderNavigation(_ context.Context, invocation RuntimeInvocation) (RuntimeOutput, error) {
	return r.renderInvocation(invocation)
}

func (r *compositionRuntime) RenderRegion(_ context.Context, invocation RuntimeInvocation) (RuntimeOutput, error) {
	return r.renderInvocation(invocation)
}

func (r *compositionRuntime) renderInvocation(invocation RuntimeInvocation) (RuntimeOutput, error) {
	r.mu.RLock()
	output, failure, render := r.outputs[invocation.Handler], r.failures[invocation.Handler], r.render
	cancelRender, cancel := r.cancelRender, r.leaseCancel
	r.mu.RUnlock()
	if cancelRender && cancel != nil {
		cancel()
	}
	if failure != nil {
		return RuntimeOutput{}, failure
	}
	if render != nil {
		return render(invocation, output)
	}
	return output, nil
}

func (r *compositionRuntime) setUnavailable(artifact Artifact, unavailable bool) {
	r.mu.Lock()
	r.unavailable[artifactKey(artifact)] = unavailable
	r.mu.Unlock()
}

func (r *compositionRuntime) setFailure(handler string, err error) {
	r.mu.Lock()
	r.failures[handler] = err
	r.mu.Unlock()
}

func (r *compositionRuntime) setLeaseRuntime(artifact Artifact, runtimeID string) {
	r.mu.Lock()
	r.leaseRuntime[artifactKey(artifact)] = runtimeID
	r.mu.Unlock()
}

func (r *compositionRuntime) setCancelOnRender(enabled bool) {
	r.mu.Lock()
	r.cancelRender = enabled
	r.mu.Unlock()
}

type compositionLease struct {
	ctx       context.Context
	runtimeID string
	cancel    context.CancelFunc
}

func (l *compositionLease) Context() context.Context  { return l.ctx }
func (l *compositionLease) RuntimeInstanceID() string { return l.runtimeID }
func (l *compositionLease) Release() {
	if l.cancel != nil {
		l.cancel()
	}
}

func artifactKey(artifact Artifact) string {
	return artifact.ExtensionID + "\x00" + artifact.ExtensionVersion + "\x00" + artifact.PackageDigest + "\x00" +
		artifact.ImpactDigest + "\x00" + artifact.RuntimeInstanceID
}

func mustCompose(t *testing.T, composer *Composer) Composition {
	t.Helper()
	result, err := composer.Compose(t.Context(), CompositionRequest{})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func composedByID(items []ComposedItem, id string) *ComposedItem {
	for index := range items {
		if items[index].ID == id {
			return &items[index]
		}
	}
	return nil
}

func hasComposedID(items []ComposedItem, id string) bool {
	return composedByID(items, id) != nil
}

func leftPad(value int) string {
	return fmt.Sprintf("%04d", value)
}
