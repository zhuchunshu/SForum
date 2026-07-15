package routes

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

// P6 路由动作矩阵（只读回归）：仅钉住生产已实现的 Registry/Plan/Dispatcher
// 行为。不冻结可变字段、wrap 同优先级、after fail-closed、redirect SEO/canonical、
// raw session/header authority 或组合 stream middleware 等仍开放的语义。

func TestP6RouteActionMatrixTerminals(t *testing.T) {
	registry := NewRegistry()
	artifact := routeArtifact("matrix.actions", "1.0.0", 'a')
	home := coreRoute("core.route.matrix.home", "GET", "/matrix/home/:id")
	add := pluginRoute("matrix.actions.add", "/matrix/custom/:id", 0, "GET")
	alias := pluginRoute("matrix.actions.alias", "/matrix/legacy/:id", 0, "GET")
	alias.Action, alias.TargetID, alias.Handler, alias.ResponseSchema = extensionmanifest.RouteActionAlias, home.ID, "", ""
	alias.Guard = extensionmanifest.GuardCoreInherit
	redirect := pluginRoute("matrix.actions.redirect", "/matrix/old", 0, "GET")
	redirect.Action, redirect.Handler, redirect.ResponseSchema, redirect.Destination =
		extensionmanifest.RouteActionRedirect, "", "", "/matrix/new"
	rewrite := pluginRoute("matrix.actions.rewrite", "/matrix/internal/:id", 0, "GET")
	rewrite.Action, rewrite.TargetID, rewrite.Handler, rewrite.ResponseSchema =
		extensionmanifest.RouteActionRewrite, home.ID, "", ""
	rewrite.Guard = extensionmanifest.GuardCoreInherit

	if _, err := registry.Publish(Publication{
		Core:    []CoreRoute{home},
		Plugins: []PluginRouteSet{{Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{add, alias, redirect, rewrite}}},
	}); err != nil {
		t.Fatal(err)
	}

	// 地址类动作：Registry 解析 + ExecutionPlan 终端身份。
	terminals := []struct {
		path        string
		action      string
		targetID    string
		destination string
		targetPath  string
	}{
		{"/matrix/custom/7", extensionmanifest.RouteActionAdd, "", "", ""},
		{"/matrix/legacy/7", extensionmanifest.RouteActionAlias, home.ID, "", "/matrix/home/7"},
		{"/matrix/old", extensionmanifest.RouteActionRedirect, "", "/matrix/new", ""},
		{"/matrix/internal/7", extensionmanifest.RouteActionRewrite, home.ID, "", "/matrix/home/7"},
	}
	for _, test := range terminals {
		match, err := registry.Resolve("GET", test.path)
		if err != nil || match.Route.Action != test.action || match.Route.Provider.Artifact != artifact {
			t.Fatalf("resolve %s = %#v, %v", test.path, match, err)
		}
		plan, err := registry.BuildExecutionPlan("GET", test.path)
		if err != nil {
			t.Fatalf("plan %s: %v", test.path, err)
		}
		terminal := plan.Terminal()
		if len(plan.Chain()) != 1 || terminal.Action != test.action || terminal.TargetID != test.targetID ||
			terminal.Destination != test.destination || terminal.TargetPath != test.targetPath {
			t.Fatalf("plan %s terminal=%#v chain=%#v", test.path, terminal, plan.Chain())
		}
	}

	// redirect：生产固定为 308 Permanent Redirect + Location，不声明 SEO/canonical 策略。
	dispatcher := NewDispatcher(DispatcherConfig{
		Plans: matrixPlanResolver{registry: registry}, Guard: &dispatchGuard{}, Schemas: &dispatchSchemas{},
	})
	result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{Method: "GET", Path: "/matrix/old"}, nil)
	if err != nil || !result.Handled || result.Response.Status != http.StatusPermanentRedirect ||
		result.Response.Headers.Get("Location") != "/matrix/new" {
		t.Fatalf("redirect dispatch = %#v err=%v", result, err)
	}

	// alias/rewrite：Dispatcher 走 CoreInvoker，不二次进入插件 runtime。
	coreCalls := 0
	core := &dispatchCoreInvoker{invoke: func(_ context.Context, step RouteExecutionStep, request DispatchRequest) (DispatchResponse, error) {
		coreCalls++
		if step.Action != extensionmanifest.RouteActionAlias && step.Action != extensionmanifest.RouteActionRewrite {
			t.Fatalf("unexpected core step action=%q", step.Action)
		}
		if request.Params["id"] != "7" {
			t.Fatalf("core params=%#v", request.Params)
		}
		return DispatchResponse{Status: http.StatusOK, Body: []byte(step.Action + ":" + step.TargetPath)}, nil
	}}
	for _, path := range []string{"/matrix/legacy/7", "/matrix/internal/7"} {
		result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{Method: "GET", Path: path}, core)
		if err != nil || !result.Handled || result.Response.Status != http.StatusOK {
			t.Fatalf("mapping %s = %#v err=%v", path, result, err)
		}
	}
	if coreCalls != 2 {
		t.Fatalf("core calls=%d", coreCalls)
	}

	// add：插件 handler 通过 StepInvoker 执行；无 Core 第二写者。
	addCore := &dispatchCoreInvoker{invoke: func(context.Context, RouteExecutionStep, DispatchRequest) (DispatchResponse, error) {
		t.Fatal("add must not invoke core writer")
		return DispatchResponse{}, nil
	}}
	addDispatcher := NewDispatcher(DispatcherConfig{
		Plans: matrixPlanResolver{registry: registry}, Steps: &dispatchStepInvoker{
			invoke: func(_ context.Context, input RouteInvocation) (RouteInvocationResult, error) {
				if input.Step.Action != extensionmanifest.RouteActionAdd || input.Step.RouteID != add.ID {
					t.Fatalf("add step=%#v", input.Step)
				}
				return RouteInvocationResult{Response: &DispatchResponse{Status: http.StatusCreated, Body: []byte("added")}}, nil
			},
		},
		Guard: &dispatchGuard{}, Schemas: &dispatchSchemas{},
	})
	result, err = addDispatcher.Dispatch(context.Background(), DispatchRequest{Method: "GET", Path: "/matrix/custom/7"}, addCore)
	if err != nil || !result.Handled || result.Response.Status != http.StatusCreated || string(result.Response.Body) != "added" {
		t.Fatalf("add dispatch = %#v err=%v", result, err)
	}
}

func TestP6RouteModifierAndGlobalChainSelection(t *testing.T) {
	// before/after/filter/wrap 与 global middleware 均按声明进入链，并按不同 priority 降序排列。
	// 同 priority 的 wrap 次序不在本矩阵内声明（避免冻结仍开放的 tie 语义）。
	registry := NewRegistry()
	artifact := routeArtifact("matrix.chain", "1.0.0", 'b')
	targetID := "core.route.matrix.topic"
	targetPath := "/matrix/topics/:slug"
	globalHigh := executionGlobalRoute("matrix.chain.global_high", 40)
	globalLow := executionGlobalRoute("matrix.chain.global_low", 10)
	beforeHigh := modifierRoute("matrix.chain.before_high", targetID, targetPath, extensionmanifest.RouteActionBefore, "GET", 90)
	beforeLow := modifierRoute("matrix.chain.before_low", targetID, targetPath, extensionmanifest.RouteActionBefore, "GET", 20)
	filterHigh := modifierRoute("matrix.chain.filter_high", targetID, targetPath, extensionmanifest.RouteActionFilter, "GET", 70)
	filterLow := modifierRoute("matrix.chain.filter_low", targetID, targetPath, extensionmanifest.RouteActionFilter, "GET", 30)
	wrapHigh := modifierRoute("matrix.chain.wrap_high", targetID, targetPath, extensionmanifest.RouteActionWrap, "GET", 60)
	wrapLow := modifierRoute("matrix.chain.wrap_low", targetID, targetPath, extensionmanifest.RouteActionWrap, "GET", 15)
	afterHigh := modifierRoute("matrix.chain.after_high", targetID, targetPath, extensionmanifest.RouteActionAfter, "GET", 50)
	afterLow := modifierRoute("matrix.chain.after_low", targetID, targetPath, extensionmanifest.RouteActionAfter, "GET", 5)
	// 另一目标的 filter/wrap 不得进入本请求链。
	otherFilter := modifierRoute("matrix.chain.other_filter", "core.route.matrix.other", "/matrix/other", extensionmanifest.RouteActionFilter, "GET", 100)
	otherWrap := modifierRoute("matrix.chain.other_wrap", "core.route.matrix.other", "/matrix/other", extensionmanifest.RouteActionWrap, "GET", 100)

	if _, err := registry.Publish(Publication{
		Core: []CoreRoute{
			coreRoute(targetID, "GET", targetPath),
			coreRoute("core.route.matrix.other", "GET", "/matrix/other"),
		},
		Plugins: []PluginRouteSet{{Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{
			globalLow, afterLow, wrapLow, filterLow, beforeLow, otherFilter, otherWrap,
			afterHigh, wrapHigh, filterHigh, beforeHigh, globalHigh,
		}}},
	}); err != nil {
		t.Fatal(err)
	}

	plan, err := registry.BuildExecutionPlan("GET", "/matrix/topics/welcome")
	if err != nil {
		t.Fatal(err)
	}
	wantIDs := []string{
		"matrix.chain.global_high", "matrix.chain.global_low",
		"matrix.chain.before_high", "matrix.chain.before_low",
		"matrix.chain.filter_high", "matrix.chain.filter_low",
		"matrix.chain.wrap_high", "matrix.chain.wrap_low",
		targetID,
		"matrix.chain.after_high", "matrix.chain.after_low",
	}
	wantPhases := []RouteExecutionPhase{
		RoutePhaseGlobal, RoutePhaseGlobal,
		RoutePhaseBefore, RoutePhaseBefore,
		RoutePhaseFilter, RoutePhaseFilter,
		RoutePhaseWrap, RoutePhaseWrap,
		RoutePhaseHandler,
		RoutePhaseAfter, RoutePhaseAfter,
	}
	chain := plan.Chain()
	if len(chain) != len(wantIDs) {
		t.Fatalf("chain=%#v", chain)
	}
	gotIDs := make([]string, 0, len(chain))
	for index, step := range chain {
		gotIDs = append(gotIDs, step.RouteID)
		if step.RouteID != wantIDs[index] || step.Phase != wantPhases[index] {
			t.Fatalf("step %d = %#v want id=%s phase=%s", index, step, wantIDs[index], wantPhases[index])
		}
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("ids=%#v", gotIDs)
	}
	if plan.Terminal().RouteID != targetID || plan.Terminal().Provider.Kind != ProviderCore {
		t.Fatalf("terminal=%#v", plan.Terminal())
	}

	// Dispatcher 按计划相位顺序调用插件步骤，after 可见 handler 响应。
	order := make([]string, 0, len(chain))
	invoker := &dispatchStepInvoker{invoke: func(_ context.Context, input RouteInvocation) (RouteInvocationResult, error) {
		order = append(order, string(input.Step.Phase)+":"+input.Step.RouteID)
		if input.Step.Phase == RoutePhaseAfter {
			if input.Response == nil || string(input.Response.Body) != "core-body" {
				t.Fatalf("after response = %#v", input.Response)
			}
			value := cloneDispatchResponse(*input.Response)
			value.Headers.Set("X-Matrix-After", "1")
			return RouteInvocationResult{Response: &value}, nil
		}
		return RouteInvocationResult{}, nil
	}}
	core := &dispatchCoreInvoker{invoke: func(_ context.Context, step RouteExecutionStep, _ DispatchRequest) (DispatchResponse, error) {
		order = append(order, string(step.Phase)+":"+step.RouteID)
		return DispatchResponse{Status: http.StatusOK, Body: []byte("core-body")}, nil
	}}
	dispatcher := NewDispatcher(DispatcherConfig{
		Plans: matrixPlanResolver{registry: registry}, Steps: invoker, Guard: &dispatchGuard{}, Schemas: &dispatchSchemas{},
	})
	result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{Method: "GET", Path: "/matrix/topics/welcome"}, core)
	if err != nil || !result.Handled || result.Response.Headers.Get("X-Matrix-After") != "1" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	wantOrder := []string{
		"global:matrix.chain.global_high", "global:matrix.chain.global_low",
		"before:matrix.chain.before_high", "before:matrix.chain.before_low",
		"filter:matrix.chain.filter_high", "filter:matrix.chain.filter_low",
		"wrap:matrix.chain.wrap_high", "wrap:matrix.chain.wrap_low",
		"handler:" + targetID,
		"after:matrix.chain.after_high", "after:matrix.chain.after_low",
	}
	if !reflect.DeepEqual(order, wantOrder) {
		t.Fatalf("order=%#v want=%#v", order, wantOrder)
	}
}

func TestP6RoutePriorityAndConflictSelection(t *testing.T) {
	t.Run("path method conflict fails closed", func(t *testing.T) {
		registry := NewRegistry()
		first := routeArtifact("matrix.conflict.a", "1.0.0", 'a')
		second := routeArtifact("matrix.conflict.b", "1.0.0", 'b')
		snapshot, err := registry.Publish(Publication{Plugins: []PluginRouteSet{
			{Artifact: first, Routes: []extensionmanifest.ManifestRoute{pluginRoute("matrix.conflict.a.route", "/matrix/shared/:id", 10, "GET")}},
			{Artifact: second, Routes: []extensionmanifest.ManifestRoute{pluginRoute("matrix.conflict.b.route", "/matrix/shared/:slug", 20, "GET")}},
		}})
		if err != nil {
			t.Fatal(err)
		}
		if len(snapshot.Conflicts) != 1 || snapshot.Conflicts[0].Kind != ConflictPathMethod {
			t.Fatalf("conflicts=%#v", snapshot.Conflicts)
		}
		match, err := registry.Resolve("GET", "/matrix/shared/value")
		if !errors.Is(err, ErrAmbiguousRoute) || len(match.Candidates) != 2 {
			t.Fatalf("resolve=%#v err=%v", match, err)
		}
		if _, err := registry.BuildExecutionPlan("GET", "/matrix/shared/value"); !errors.Is(err, ErrAmbiguousRoute) {
			t.Fatalf("plan err=%v", err)
		}
	})

	t.Run("higher path specificity wins without conflict", func(t *testing.T) {
		registry := NewRegistry()
		artifact := routeArtifact("matrix.specificity", "1.0.0", 'c')
		// 更具体的静态段应压过参数段；两者 path signature 不同故不构成 path_method 冲突。
		snapshot, err := registry.Publish(Publication{Plugins: []PluginRouteSet{{
			Artifact: artifact,
			Routes: []extensionmanifest.ManifestRoute{
				pluginRoute("matrix.specificity.param", "/matrix/items/:id", 0, "GET"),
				pluginRoute("matrix.specificity.static", "/matrix/items/featured", 0, "GET"),
			},
		}}})
		if err != nil {
			t.Fatal(err)
		}
		if len(snapshot.Conflicts) != 0 {
			t.Fatalf("specific routes must not conflict: %#v", snapshot.Conflicts)
		}
		match, err := registry.Resolve("GET", "/matrix/items/featured")
		if err != nil || match.Route.ID != "matrix.specificity.static" {
			t.Fatalf("resolve=%#v err=%v", match, err)
		}
		plan, err := registry.BuildExecutionPlan("GET", "/matrix/items/featured")
		if err != nil || plan.Terminal().RouteID != "matrix.specificity.static" {
			t.Fatalf("plan=%#v err=%v", plan.Terminal(), err)
		}
	})

	t.Run("replace requires explicit unique provider", func(t *testing.T) {
		registry := NewRegistry()
		artifact := routeArtifact("matrix.replace", "1.0.0", 'd')
		target := coreRoute("core.route.matrix.create", "POST", "/matrix/create")
		replacement := modifierRoute(
			"matrix.replace.writer", target.ID, target.Path,
			extensionmanifest.RouteActionReplace, "POST", 100,
		)
		snapshot, err := registry.Publish(Publication{
			Core:    []CoreRoute{target},
			Plugins: []PluginRouteSet{{Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{replacement}}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(snapshot.Conflicts) != 1 || snapshot.Conflicts[0].Kind != ConflictProviderSelection {
			t.Fatalf("conflicts=%#v", snapshot.Conflicts)
		}
		if _, err := registry.BuildExecutionPlan("POST", target.Path); !errors.Is(err, ErrAmbiguousRoute) {
			t.Fatalf("unselected replace plan err=%v", err)
		}

		providers := NewProviderSelectionAPI(registry, newMemoryProviderSelectionStore())
		conflicts, err := providers.Conflicts(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(conflicts) != 1 || conflicts[0].SelectionStatus != "unselected" {
			t.Fatalf("provider conflicts=%#v", conflicts)
		}
		key := conflicts[0].Key
		if _, err := providers.Select(t.Context(), SelectProviderRequest{
			Key: key, ProviderRouteID: replacement.ID, ProviderContractVersion: replacement.ContractVersion,
			ProviderArtifact: artifact, ActorUserID: 7, AuditEventID: 17,
		}); err != nil {
			t.Fatal(err)
		}
		// 显式选择后必须经过生产 ProviderSelectionAPI 解析，不允许测试手工制造 winner。
		plan, err := providers.BuildExecutionPlan(t.Context(), "POST", target.Path)
		if err != nil || plan.Terminal().Action != extensionmanifest.RouteActionReplace ||
			plan.Terminal().Provider.Artifact != artifact || !plan.UnsafeMethod() {
			t.Fatalf("selected replace plan=%#v err=%v", plan.Terminal(), err)
		}

		core := &dispatchCoreInvoker{invoke: func(context.Context, RouteExecutionStep, DispatchRequest) (DispatchResponse, error) {
			t.Fatal("selected replacement must be the only writer")
			return DispatchResponse{}, nil
		}}
		pluginCalls := 0
		dispatcher := NewDispatcher(DispatcherConfig{
			Plans: providers,
			Steps: &dispatchStepInvoker{invoke: func(_ context.Context, input RouteInvocation) (RouteInvocationResult, error) {
				pluginCalls++
				if input.Step.Action != extensionmanifest.RouteActionReplace || input.Step.Provider.Artifact != artifact {
					t.Fatalf("replace step=%#v", input.Step)
				}
				return RouteInvocationResult{Response: &DispatchResponse{Status: http.StatusCreated, Body: []byte("replaced")}}, nil
			}},
			Guard: &dispatchGuard{}, Schemas: &dispatchSchemas{},
		})
		result, err := dispatcher.Dispatch(t.Context(), DispatchRequest{Method: "POST", Path: target.Path}, core)
		if err != nil || !result.Handled || result.Response.Status != http.StatusCreated ||
			string(result.Response.Body) != "replaced" || pluginCalls != 1 || core.calls != 0 {
			t.Fatalf("replace dispatch=%#v plugin=%d core=%d err=%v", result, pluginCalls, core.calls, err)
		}
	})
}

func TestP6RouteSafeModeBypassesPluginContributions(t *testing.T) {
	registry := NewRegistry()
	core := coreRoute("core.route.matrix.health", "GET", "/matrix/health")
	plugin := pluginRoute("matrix.safe.plugin", "/matrix/plugin", 0, "GET")
	before := modifierRoute(
		"matrix.safe.before", core.ID, core.Path, extensionmanifest.RouteActionBefore, "GET", 50,
	)
	snapshot, err := registry.Publish(Publication{
		SafeMode: true,
		Core:     []CoreRoute{core},
		Plugins: []PluginRouteSet{{
			Artifact: routeArtifact("matrix.safe", "1.0.0", 'e'),
			Routes:   []extensionmanifest.ManifestRoute{plugin, before},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.SafeMode || len(snapshot.Routes) != 1 || snapshot.Routes[0].Provider.Kind != ProviderCore {
		t.Fatalf("safe snapshot=%#v", snapshot)
	}
	if _, err := registry.Resolve("GET", "/matrix/plugin"); !errors.Is(err, ErrRouteNotFound) {
		t.Fatalf("plugin resolve err=%v", err)
	}
	plan, err := registry.BuildExecutionPlan("GET", "/matrix/health")
	if err != nil {
		t.Fatal(err)
	}
	// Safe Mode 下 core-only 计划无插件链；Dispatcher 应完全放行给 Fiber。
	if len(plan.Chain()) != 1 || plan.Terminal().Provider.Kind != ProviderCore {
		t.Fatalf("safe plan=%#v", plan.Chain())
	}
	dispatcher := NewDispatcher(DispatcherConfig{
		Plans: matrixPlanResolver{registry: registry}, Guard: &dispatchGuard{}, Schemas: &dispatchSchemas{},
	})
	result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{Method: "GET", Path: "/matrix/health"}, &dispatchCoreInvoker{
		invoke: func(context.Context, RouteExecutionStep, DispatchRequest) (DispatchResponse, error) {
			t.Fatal("core-only safe plan must not enter buffered core invoker")
			return DispatchResponse{}, nil
		},
	})
	if err != nil || result.Handled {
		t.Fatalf("safe dispatch=%#v err=%v", result, err)
	}
}

func TestP6RouteUnsafeNoSecondWriterMatrix(t *testing.T) {
	// 与生产 fence 一致：非 GET/HEAD、已开始副作用/响应后，绝不允许 core 成为第二写者。
	tests := []struct {
		name            string
		method          string
		sideEffect      bool
		responseStarted bool
	}{
		{name: "unsafe pristine post", method: "POST"},
		{name: "get after side effect", method: "GET", sideEffect: true},
		{name: "get after response started", method: "GET", responseStarted: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			providers := matrixSelectedReplacement(t, test.method, "/matrix/fence")
			core := &dispatchCoreInvoker{}
			dispatcher := NewDispatcher(DispatcherConfig{
				Plans: providers,
				Steps: &dispatchStepInvoker{invoke: func(_ context.Context, input RouteInvocation) (RouteInvocationResult, error) {
					if input.Step.Action != extensionmanifest.RouteActionReplace || input.Step.Fallback != "readonly_core" {
						t.Fatalf("replace step=%#v", input.Step)
					}
					if test.sideEffect {
						input.Commit.SideEffectStarted()
					}
					if test.responseStarted {
						input.Commit.ResponseStarted()
					}
					return RouteInvocationResult{
						SideEffectStarted: test.sideEffect, ResponseStarted: test.responseStarted,
					}, errors.New("writer failed")
				}},
				Guard: &dispatchGuard{}, Schemas: &dispatchSchemas{},
			})
			result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{Method: test.method, Path: "/matrix/fence"}, core)
			if !errors.Is(err, ErrDispatchTransport) || result.Handled || core.calls != 0 {
				t.Fatalf("result=%#v core=%d err=%v", result, core.calls, err)
			}
		})
	}

	// 对照：GET/HEAD + pristine + readonly_core 允许一次只读 core 回退（已接受契约）。
	for _, method := range []string{"GET", "HEAD"} {
		t.Run("pristine "+method, func(t *testing.T) {
			providers := matrixSelectedReplacement(t, method, "/matrix/fence")
			core := &dispatchCoreInvoker{invoke: func(_ context.Context, step RouteExecutionStep, _ DispatchRequest) (DispatchResponse, error) {
				if step.Action != extensionmanifest.RouteActionReplace || step.TargetID != "core.route.matrix.fence" {
					t.Fatalf("fallback step=%#v", step)
				}
				return DispatchResponse{Status: http.StatusOK, Body: []byte("core")}, nil
			}}
			dispatcher := NewDispatcher(DispatcherConfig{
				Plans: providers,
				Steps: &dispatchStepInvoker{invoke: func(context.Context, RouteInvocation) (RouteInvocationResult, error) {
					return RouteInvocationResult{}, errors.New("plugin unavailable")
				}},
				Guard: &dispatchGuard{}, Schemas: &dispatchSchemas{},
			})
			result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{Method: method, Path: "/matrix/fence"}, core)
			if err != nil || !result.Handled || string(result.Response.Body) != "core" || core.calls != 1 {
				t.Fatalf("readonly fallback=%#v core=%d err=%v", result, core.calls, err)
			}
		})
	}
}

func TestP6RouteTimeoutWhereHarnessSupports(t *testing.T) {
	// 现有 Dispatcher 超时 harness：步骤 TimeoutMS 使 Invoke 上下文截止；
	// fallback=closed 时 fail-closed，不进入 core 第二写者。
	for _, test := range []struct {
		name    string
		parent  func() (context.Context, context.CancelFunc)
		wantErr error
	}{
		{
			name: "step timeout",
			parent: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			wantErr: context.DeadlineExceeded,
		},
		{
			name: "caller cancellation",
			parent: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			wantErr: context.Canceled,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			registry := NewRegistry()
			route := pluginRoute("matrix.timeout.route", "/matrix/timeout", 0, "GET")
			route.TimeoutMS = 5
			if _, err := registry.Publish(Publication{Plugins: []PluginRouteSet{{
				Artifact: routeArtifact("matrix.timeout", "1.0.0", 'f'),
				Routes:   []extensionmanifest.ManifestRoute{route},
			}}}); err != nil {
				t.Fatal(err)
			}
			core := &dispatchCoreInvoker{}
			dispatcher := NewDispatcher(DispatcherConfig{
				Plans: matrixPlanResolver{registry: registry},
				Steps: &dispatchStepInvoker{invoke: func(ctx context.Context, input RouteInvocation) (RouteInvocationResult, error) {
					if input.Step.TimeoutMS != 5 || input.Step.RouteID != route.ID {
						t.Fatalf("timeout step=%#v", input.Step)
					}
					<-ctx.Done()
					return RouteInvocationResult{}, ctx.Err()
				}},
				Guard: &dispatchGuard{}, Schemas: &dispatchSchemas{},
			})
			ctx, cancel := test.parent()
			defer cancel()
			result, err := dispatcher.Dispatch(ctx, DispatchRequest{Method: "GET", Path: "/matrix/timeout"}, core)
			if !errors.Is(err, ErrDispatchTransport) || !errors.Is(err, test.wantErr) || result.Handled || core.calls != 0 {
				t.Fatalf("result=%#v core=%d err=%v", result, core.calls, err)
			}
		})
	}
}

func matrixSelectedReplacement(t *testing.T, method, path string) *ProviderSelectionAPI {
	t.Helper()
	registry := NewRegistry()
	artifact := routeArtifact("matrix.fence", "1.0.0", 'e')
	target := coreRoute("core.route.matrix.fence", method, path)
	replacement := modifierRoute(
		"matrix.fence.replace", target.ID, target.Path,
		extensionmanifest.RouteActionReplace, method, 100,
	)
	replacement.Fallback = "readonly_core"
	snapshot, err := registry.Publish(Publication{
		Core:    []CoreRoute{target},
		Plugins: []PluginRouteSet{{Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{replacement}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Conflicts) != 1 || snapshot.Conflicts[0].Kind != ConflictProviderSelection {
		t.Fatalf("replace conflicts=%#v", snapshot.Conflicts)
	}
	providers := NewProviderSelectionAPI(registry, newMemoryProviderSelectionStore())
	conflicts, err := providers.Conflicts(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 1 || conflicts[0].SelectionStatus != "unselected" {
		t.Fatalf("provider conflicts=%#v", conflicts)
	}
	key := conflicts[0].Key
	if _, err := providers.Select(t.Context(), SelectProviderRequest{
		Key: key, ProviderRouteID: replacement.ID, ProviderContractVersion: replacement.ContractVersion,
		ProviderArtifact: artifact, ActorUserID: 7, AuditEventID: 17,
	}); err != nil {
		t.Fatal(err)
	}
	plan, err := providers.BuildExecutionPlan(t.Context(), method, path)
	if err != nil || plan.Terminal().Action != extensionmanifest.RouteActionReplace ||
		plan.Terminal().Fallback != "readonly_core" || plan.Terminal().Provider.Artifact != artifact {
		t.Fatalf("selected fallback plan=%#v err=%v", plan.Terminal(), err)
	}
	return providers
}

type matrixPlanResolver struct {
	registry *Registry
}

func (r matrixPlanResolver) BuildExecutionPlan(_ context.Context, method, path string) (RouteExecutionPlan, error) {
	return r.registry.BuildExecutionPlan(method, path)
}
