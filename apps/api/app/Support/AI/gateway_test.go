package ai_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	ai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
)

type fakeSettings struct{ settings ai.Settings }

func (f *fakeSettings) GetSettings(context.Context) (ai.Settings, error) { return f.settings, nil }
func (f *fakeSettings) SaveSettings(_ context.Context, s ai.Settings, _ int64) (ai.Settings, error) {
	return s, nil
}
func (f *fakeSettings) ResetSettings(_ context.Context, s ai.Settings, _ int64) (ai.Settings, error) {
	return s, nil
}

type fakeUsage struct {
	snapshot ai.UsageByScope
	records  []ai.UsageEntry
}

func (f *fakeUsage) Snapshot(context.Context, time.Time, string, int64) (ai.UsageByScope, error) {
	return f.snapshot, nil
}
func (f *fakeUsage) Record(_ context.Context, _ time.Time, entry ai.UsageEntry) error {
	f.records = append(f.records, entry)
	return nil
}

type fakeTraces struct{ records []ai.ExecutionRecord }

func (f *fakeTraces) RecordExecution(_ context.Context, entry ai.ExecutionRecord) error {
	f.records = append(f.records, entry)
	return nil
}
func (f *fakeTraces) ListExecutions(context.Context, int) ([]ai.ExecutionRecord, error) {
	return f.records, nil
}

type fakeCreds struct {
	value string
	err   error
}

func (f *fakeCreds) ResolveAIKey(context.Context, string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.value, nil
}

type fakeInvoker struct {
	calls    int
	response ai.WireResponse
	err      error
	lastWire ai.WireRequest
}

func (f *fakeInvoker) Invoke(_ context.Context, _ ai.Profile, request ai.WireRequest) (ai.WireResponse, error) {
	f.calls++
	f.lastWire = request
	if f.err != nil {
		return ai.WireResponse{}, f.err
	}
	return f.response, nil
}

type fakeCache struct {
	items map[string]ai.CompletionResult
}

func (f *fakeCache) Get(_ context.Context, key string) (ai.CompletionResult, bool) {
	item, ok := f.items[key]
	return item, ok
}
func (f *fakeCache) Put(_ context.Context, key string, result ai.CompletionResult, _ time.Duration) {
	if f.items == nil {
		f.items = map[string]ai.CompletionResult{}
	}
	f.items[key] = result
}

func enabledSettings() ai.Settings {
	settings := ai.RecommendedSettings()
	settings.Enabled = true
	settings.Revision = 7
	for i := range settings.Profiles {
		if settings.Profiles[i].ID == "deepseek" {
			settings.Profiles[i].Enabled = true
		}
	}
	return settings
}

func newGateway(t *testing.T, settings ai.Settings) (*ai.Gateway, *fakeUsage, *fakeTraces, *fakeInvoker, *fakeCache) {
	return newGatewayWithUsage(t, settings, ai.UsageByScope{})
}

func newGatewayWithUsage(t *testing.T, settings ai.Settings, snapshot ai.UsageByScope) (*ai.Gateway, *fakeUsage, *fakeTraces, *fakeInvoker, *fakeCache) {
	t.Helper()
	usage := &fakeUsage{snapshot: snapshot}
	traces := &fakeTraces{}
	invoker := &fakeInvoker{response: ai.WireResponse{
		Text: "ok", StopReason: ai.StopReasonEnd,
		Usage: ai.Usage{InputTokens: 10, OutputTokens: 5},
	}}
	cache := &fakeCache{}
	gateway := ai.NewGateway(ai.GatewayConfig{
		Settings: &fakeSettings{settings: settings},
		Usage:    usage,
		Traces:   traces,
		Creds:    &fakeCreds{value: "sk-test"},
		Invoker:  invoker,
		Cache:    cache,
		Clock:    func() time.Time { return time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC) },
	})
	return gateway, usage, traces, invoker, cache
}

func TestGatewayRejectsWhenDisabledWithoutTouchingProvider(t *testing.T) {
	settings := ai.RecommendedSettings()
	gateway, _, traces, invoker, _ := newGateway(t, settings)
	_, err := gateway.Execute(context.Background(), validRequest())
	var unavailable *ai.UnavailableError
	if !errors.As(err, &unavailable) || !errors.Is(err, ai.ErrGatewayDisabled) {
		t.Fatalf("err = %v", err)
	}
	if unavailable.Posture != ai.FailurePostureOpen {
		t.Fatalf("posture = %q", unavailable.Posture)
	}
	if invoker.calls != 0 {
		t.Fatal("disabled gateway must not call the provider")
	}
	if len(traces.records) != 1 || traces.records[0].Status != ai.ExecutionStatusDenied {
		t.Fatalf("traces = %+v", traces.records)
	}
}

func TestGatewayEnforcesQuotaWithinItsOwnScope(t *testing.T) {
	request := validRequest()
	request.Metadata.CallerExtensionID = "vendor.plugin"

	// 配额耗尽：该插件被拒绝。
	gateway, _, traces, invoker, _ := newGatewayWithUsage(t, enabledSettings(), ai.UsageByScope{
		Extension: ai.UsageSnapshot{DayCalls: 500},
	})
	_, err := gateway.Execute(context.Background(), request)
	var unavailable *ai.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Reason != ai.GateReasonQuotaExceeded {
		t.Fatalf("err = %v", err)
	}
	if invoker.calls != 0 {
		t.Fatal("quota-exhausted caller must not reach the provider")
	}
	if len(traces.records) != 1 || traces.records[0].Status != ai.ExecutionStatusDenied {
		t.Fatalf("traces = %+v", traces.records)
	}
	if traces.records[0].GateScope != ai.GateScopeExtension {
		t.Fatalf("gate scope = %q", traces.records[0].GateScope)
	}

	// 同一份超额用量在另一个插件身上不生效：闸门按发起方隔离。
	other := validRequest()
	other.Metadata.CallerExtensionID = "other.plugin"
	gateway, _, _, invoker, _ = newGatewayWithUsage(t, enabledSettings(), ai.UsageByScope{})
	if _, err := gateway.Execute(context.Background(), other); err != nil {
		t.Fatalf("unrelated plugin must not be gated: %v", err)
	}
	if invoker.calls != 1 {
		t.Fatalf("invoker calls = %d", invoker.calls)
	}
}

func TestGatewayRecordsSuccessfulExecution(t *testing.T) {
	gateway, usage, traces, invoker, cache := newGateway(t, enabledSettings())
	request := validRequest()
	request.Metadata.CallerExtensionID = "vendor.plugin"
	request.PromptVersion = "moderation@3"
	result, err := gateway.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.Text != "ok" || result.StopReason != ai.StopReasonEnd {
		t.Fatalf("result = %+v", result)
	}
	if result.PromptVersion != "moderation@3" || result.ConfigRevision != 7 {
		t.Fatalf("provenance missing: %+v", result)
	}
	if invoker.calls != 1 {
		t.Fatalf("invoker calls = %d", invoker.calls)
	}
	if wire := invoker.lastWire; wire.URL != "https://api.deepseek.com/chat/completions" {
		t.Fatalf("wire url = %s", wire.URL)
	}
	if len(usage.records) != 1 || usage.records[0].InputTokens != 10 {
		t.Fatalf("usage = %+v", usage.records)
	}
	if len(traces.records) != 1 {
		t.Fatalf("traces = %+v", traces.records)
	}
	trace := traces.records[0]
	if trace.Status != ai.ExecutionStatusSucceeded || trace.PromptVersion != "moderation@3" {
		t.Fatalf("trace = %+v", trace)
	}
	if trace.CallerExtensionID != "vendor.plugin" || trace.Model != "deepseek-chat" {
		t.Fatalf("trace attribution = %+v", trace)
	}
	if len(cache.items) != 1 {
		t.Fatalf("cache = %+v", cache.items)
	}
}

func TestGatewayServesSecondCallFromCache(t *testing.T) {
	gateway, usage, _, invoker, _ := newGateway(t, enabledSettings())
	request := validRequest()
	for i := 0; i < 2; i++ {
		if _, err := gateway.Execute(context.Background(), request); err != nil {
			t.Fatalf("call %d failed: %v", i, err)
		}
	}
	if invoker.calls != 1 {
		t.Fatalf("second identical call must hit cache, invoker calls = %d", invoker.calls)
	}
	if len(usage.records) != 1 {
		t.Fatalf("cached hit must not double count usage: %+v", usage.records)
	}
}

func TestGatewayReportsMissingCredential(t *testing.T) {
	gateway, _, traces, invoker, _ := newGateway(t, enabledSettings())
	gateway = ai.NewGateway(ai.GatewayConfig{
		Settings: &fakeSettings{settings: enabledSettings()},
		Creds:    &fakeCreds{err: errors.New("secret not found")},
		Invoker:  invoker,
		Traces:   traces,
	})
	_, err := gateway.Execute(context.Background(), validRequest())
	var unavailable *ai.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Reason != "provider_failed" {
		t.Fatalf("err = %v", err)
	}
	if !errors.Is(err, ai.ErrCredentialMissing) {
		t.Fatalf("expected ErrCredentialMissing in chain: %v", err)
	}
	if invoker.calls != 0 {
		t.Fatal("provider must not be called without a credential")
	}
	if len(traces.records) != 1 || traces.records[0].Status != ai.ExecutionStatusFailed {
		t.Fatalf("traces = %+v", traces.records)
	}
}

func TestGatewayPropagatesProviderFailureWithPosture(t *testing.T) {
	settings := enabledSettings()
	settings.PurposeFailurePosture["moderation.review"] = ai.FailurePostureClosed
	gateway, _, traces, invoker, _ := newGateway(t, settings)
	invoker.err = errors.New("upstream 503")
	_, err := gateway.Execute(context.Background(), validRequest())
	var unavailable *ai.UnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("err = %v", err)
	}
	if unavailable.Posture != ai.FailurePostureClosed {
		t.Fatalf("posture = %q", unavailable.Posture)
	}
	if len(traces.records) != 1 || !strings.Contains(traces.records[0].ErrorSummary, "503") {
		t.Fatalf("traces = %+v", traces.records)
	}
}

func TestGatewayRejectsUnboundProfile(t *testing.T) {
	settings := enabledSettings()
	settings.CostClassProfiles[ai.CostClassEconomy] = ""
	gateway, _, traces, invoker, _ := newGateway(t, settings)
	_, err := gateway.Execute(context.Background(), validRequest())
	var unavailable *ai.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Reason != "profile_unavailable" {
		t.Fatalf("err = %v", err)
	}
	if invoker.calls != 0 {
		t.Fatal("provider must not be called without a resolvable profile")
	}
	if len(traces.records) != 1 || traces.records[0].Status != ai.ExecutionStatusFailed {
		t.Fatalf("traces = %+v", traces.records)
	}
}

func TestGatewayRejectsInvalidRequestBeforeAnySideEffect(t *testing.T) {
	gateway, _, traces, invoker, _ := newGateway(t, enabledSettings())
	request := validRequest()
	request.MaxTokens = 0
	if _, err := gateway.Execute(context.Background(), request); !errors.Is(err, ai.ErrRequestInvalid) {
		t.Fatalf("err = %v", err)
	}
	if invoker.calls != 0 || len(traces.records) != 0 {
		t.Fatal("invalid request must not reach provider or trace store")
	}
}

func TestCacheKeySeparatesCallersAndPromptVersions(t *testing.T) {
	profile := deepseekProfile()
	first := validRequest()
	second := validRequest()
	second.Metadata.CallerExtensionID = "other.plugin"
	if ai.CacheKey(first, profile, 1) == ai.CacheKey(second, profile, 1) {
		t.Fatal("different callers must not share a cache key")
	}
	third := validRequest()
	third.PromptVersion = "moderation@2"
	if ai.CacheKey(first, profile, 1) == ai.CacheKey(third, profile, 1) {
		t.Fatal("different prompt versions must not share a cache key")
	}
	if ai.CacheKey(first, profile, 1) == ai.CacheKey(first, profile, 2) {
		t.Fatal("different config revisions must not share a cache key")
	}
}
