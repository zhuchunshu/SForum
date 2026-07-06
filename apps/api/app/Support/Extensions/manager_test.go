package extensionsruntime

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

func TestManagerTracksStartStopStatusAndRouteTargets(t *testing.T) {
	manager := NewManager(ManagerConfig{})
	extension := runtimeExtension("demo.plugin")

	if err := manager.Start(context.Background(), extension); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	status := manager.Status(context.Background(), extension)
	if status.State != extensions.RuntimeRunning || status.RouteCount != 1 {
		t.Fatalf("unexpected running status: %#v", status)
	}
	target, ok := manager.RouteTarget("demo.plugin")
	if !ok || target.BaseURL == "" {
		t.Fatalf("expected route target, got %#v ok=%v", target, ok)
	}

	if err := manager.Stop(context.Background(), extension); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	status = manager.Status(context.Background(), extension)
	if status.State != extensions.RuntimeStopped {
		t.Fatalf("expected stopped status, got %#v", status)
	}
}

func TestManagerRecordsStartFailure(t *testing.T) {
	manager := NewManager(ManagerConfig{Starter: fakeStarter{err: errors.New("start failed")}})
	extension := runtimeExtension("broken.plugin")
	err := manager.Start(context.Background(), extension)
	if err == nil {
		t.Fatal("expected start failure")
	}
	status := manager.Status(context.Background(), extension)
	if status.State != extensions.RuntimeFailed || status.LastError == "" {
		t.Fatalf("expected failed status, got %#v", status)
	}
}

func TestManagerEmitsFilterEventsInStableOrderAndMergesPatches(t *testing.T) {
	calls := []string{}
	bus := NewHookBus(HookBusConfig{Invoker: HookInvokerFunc(func(_ context.Context, extension extensions.Extension, input HookInput) HookResult {
		calls = append(calls, extension.ID+":"+input.Kind)
		if extension.ID == "alpha.plugin" {
			return HookResult{OK: true, Patch: map[string]any{"title": "patched"}}
		}
		return HookResult{OK: true, Patch: map[string]any{"categorySlug": "general"}}
	})})
	manager := NewManager(ManagerConfig{HookBus: bus})
	beta := runtimeExtension("beta.plugin")
	alpha := runtimeExtension("alpha.plugin")
	for index := range beta.Manifest.Routes {
		beta.Manifest.Routes[index].Access = extensions.RouteAccessLogin
	}
	alpha.Manifest.Events = []extensions.ManifestEvent{{Name: appevents.TopicBeforeCreate, Kind: appevents.KindFilter}}
	beta.Manifest.Events = []extensions.ManifestEvent{{Name: appevents.TopicBeforeCreate, Kind: appevents.KindFilter}}
	if err := manager.Start(context.Background(), beta); err != nil {
		t.Fatalf("start beta: %v", err)
	}
	if err := manager.Start(context.Background(), alpha); err != nil {
		t.Fatalf("start alpha: %v", err)
	}

	result := manager.Emit(context.Background(), appevents.Envelope{
		Name:        appevents.TopicBeforeCreate,
		Kind:        appevents.KindFilter,
		PatchFields: []string{"title", "categorySlug"},
		Payload:     map[string]any{"title": "original"},
	})
	if !result.OK {
		t.Fatalf("expected filter event ok, got %#v", result)
	}
	if !slices.Equal(calls, []string{"alpha.plugin:filter", "beta.plugin:filter"}) {
		t.Fatalf("unexpected call order: %#v", calls)
	}
	if result.Patch["title"] != "patched" || result.Patch["categorySlug"] != "general" {
		t.Fatalf("unexpected merged patch: %#v", result.Patch)
	}
}

func TestManagerAllowsTopicBeforeCreateTagSlugPatchFromCatalog(t *testing.T) {
	bus := NewHookBus(HookBusConfig{Invoker: HookInvokerFunc(func(_ context.Context, _ extensions.Extension, _ HookInput) HookResult {
		return HookResult{OK: true, Patch: map[string]any{"tagSlugs": []string{"patched"}}}
	})})
	manager := NewManager(ManagerConfig{HookBus: bus})
	extension := runtimeExtension("tagger.plugin")
	extension.Manifest.Events = []extensions.ManifestEvent{{Name: appevents.TopicBeforeCreate, Kind: appevents.KindFilter}}
	if err := manager.Start(context.Background(), extension); err != nil {
		t.Fatalf("start plugin: %v", err)
	}

	result := manager.Emit(context.Background(), appevents.NewEnvelope(appevents.TopicBeforeCreate, map[string]any{
		"title":    "original",
		"tagSlugs": []string{"original"},
	}))
	if !result.OK {
		t.Fatalf("expected tagSlugs patch to be allowed, got %#v", result)
	}
	if got, ok := result.Patch["tagSlugs"].([]string); !ok || !slices.Equal(got, []string{"patched"}) {
		t.Fatalf("unexpected tagSlugs patch: %#v", result.Patch["tagSlugs"])
	}
}

func TestManagerRecordsObserveEventDeliveries(t *testing.T) {
	store := &fakeDeliveryStore{}
	bus := NewHookBus(HookBusConfig{Invoker: HookInvokerFunc(func(_ context.Context, _ extensions.Extension, input HookInput) HookResult {
		if input.DeliveryID == 0 || input.CorrelationID == "" {
			return HookResult{OK: false, Reason: "bad.delivery"}
		}
		return HookResult{OK: true}
	})})
	manager := NewManager(ManagerConfig{HookBus: bus, DeliveryStore: store})
	extension := runtimeExtension("demo.plugin")
	extension.Manifest.Events = []extensions.ManifestEvent{{Name: appevents.TopicCreated, Kind: appevents.KindObserve}}
	if err := manager.Start(context.Background(), extension); err != nil {
		t.Fatalf("start: %v", err)
	}

	result := manager.Emit(context.Background(), appevents.Envelope{
		Name:          appevents.TopicCreated,
		Kind:          appevents.KindObserve,
		CorrelationID: "corr-1",
		Payload:       map[string]any{"topicId": int64(42)},
	})
	if !result.OK {
		t.Fatalf("expected observe event ok, got %#v", result)
	}
	if len(store.deliveries) != 1 {
		t.Fatalf("expected one delivery, got %#v", store.deliveries)
	}
	if store.deliveries[0].Status != extensions.DeliverySucceeded || store.deliveries[0].AttemptCount != 1 {
		t.Fatalf("expected succeeded delivery, got %#v", store.deliveries[0])
	}
}

func runtimeExtension(id string) extensions.Extension {
	return extensions.Extension{
		ID:     id,
		Type:   extensions.TypePlugin,
		Status: extensions.StatusEnabled,
		Manifest: extensions.Manifest{
			ID:            id,
			Name:          "Demo Plugin",
			Version:       "1.0.0",
			Type:          extensions.TypePlugin,
			SForumVersion: "^1.0.0",
			Backend:       extensions.ManifestBackend{Entry: "backend/plugin", RPC: "hashicorp-go-plugin", ProtocolVersion: 1},
			Routes:        []extensions.ManifestRoute{{Path: "/hello", Methods: []string{"GET"}, Access: extensions.RouteAccessPublic}},
		},
	}
}

type fakeStarter struct {
	err error
}

func (s fakeStarter) Start(context.Context, extensions.Extension) (RouteTarget, error) {
	if s.err != nil {
		return RouteTarget{}, s.err
	}
	return RouteTarget{BaseURL: "http://127.0.0.1:43210"}, nil
}

func (s fakeStarter) Stop(context.Context, extensions.Extension) error {
	return nil
}

type fakeDeliveryStore struct {
	deliveries []extensions.ExtensionEventDelivery
}

func (s *fakeDeliveryStore) CreateEventDelivery(_ context.Context, input extensions.EventDeliveryInput) (extensions.ExtensionEventDelivery, error) {
	delivery := extensions.ExtensionEventDelivery{
		ID:            int64(len(s.deliveries) + 1),
		ExtensionID:   input.ExtensionID,
		EventName:     input.EventName,
		EventKind:     input.EventKind,
		Status:        input.Status,
		CorrelationID: input.CorrelationID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	s.deliveries = append(s.deliveries, delivery)
	return delivery, nil
}

func (s *fakeDeliveryStore) UpdateEventDelivery(_ context.Context, input extensions.EventDeliveryUpdateInput) error {
	for index := range s.deliveries {
		if s.deliveries[index].ID == input.ID {
			s.deliveries[index].Status = input.Status
			s.deliveries[index].Reason = input.Reason
			s.deliveries[index].Message = input.Message
			s.deliveries[index].AttemptCount = input.AttemptCount
			s.deliveries[index].UpdatedAt = time.Now()
			if input.Completed {
				completed := time.Now()
				s.deliveries[index].CompletedAt = &completed
			}
			return nil
		}
	}
	return extensions.ErrExtensionNotFound
}
