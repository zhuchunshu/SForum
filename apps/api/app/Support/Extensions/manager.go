package extensionsruntime

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

type Starter interface {
	Start(ctx context.Context, extension extensions.Extension) (RouteTarget, error)
	Stop(ctx context.Context, extension extensions.Extension) error
}

type MailProviderInvoker interface {
	SendMail(context.Context, string, MailProviderRequest) (MailProviderResponse, error)
}

type ManagerConfig struct {
	Starter       Starter
	HookBus       *HookBus
	DeliveryStore DeliveryStore
	Dispatcher    EventDispatcher
}

type Manager struct {
	mu            sync.RWMutex
	starter       Starter
	statuses      map[string]extensions.RuntimeStatus
	targets       map[string]RouteTarget
	running       map[string]extensions.Extension
	hooks         *HookBus
	deliveryStore DeliveryStore
	dispatcher    EventDispatcher
}

type DeliveryStore interface {
	CreateEventDelivery(ctx context.Context, input extensions.EventDeliveryInput) (extensions.ExtensionEventDelivery, error)
	UpdateEventDelivery(ctx context.Context, input extensions.EventDeliveryUpdateInput) error
}

type EventDispatcher interface {
	Enqueue(ctx context.Context, args EventDeliveryArgs, opts supportjobs.EnqueueOptions) error
}

func NewManager(config ManagerConfig) *Manager {
	starter := config.Starter
	if starter == nil {
		starter = localStarter{}
	}
	hooks := config.HookBus
	if hooks == nil {
		var invoker HookInvoker
		if candidate, ok := starter.(HookInvoker); ok {
			invoker = candidate
		}
		hooks = NewHookBus(HookBusConfig{Invoker: invoker})
	}
	return &Manager{
		starter:       starter,
		statuses:      map[string]extensions.RuntimeStatus{},
		targets:       map[string]RouteTarget{},
		running:       map[string]extensions.Extension{},
		hooks:         hooks,
		deliveryStore: config.DeliveryStore,
		dispatcher:    config.Dispatcher,
	}
}

func (m *Manager) Check(_ context.Context, extension extensions.Extension) error {
	if extension.Manifest.Backend.Entry == "" {
		return nil
	}
	path, ok := extensions.InstalledFilePathForRuntime(extension, extension.Manifest.Backend.Entry)
	if !ok {
		return extensions.ErrInvalidManifest
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return fmt.Errorf("backend entry %s is not available", extension.Manifest.Backend.Entry)
	}
	return nil
}

func (m *Manager) Start(ctx context.Context, extension extensions.Extension) error {
	m.setStatus(extension, extensions.RuntimeStatus{
		State:         extensions.RuntimeStarting,
		RouteCount:    len(extension.Manifest.Routes),
		HookCount:     len(extensions.DeclaredManifestEvents(extension.Manifest)),
		EventCount:    len(extensions.DeclaredManifestEvents(extension.Manifest)),
		ProviderCount: len(extension.Manifest.Providers),
	})
	target, err := m.starter.Start(ctx, extension)
	if err != nil {
		m.setStatus(extension, extensions.RuntimeStatus{
			State:         extensions.RuntimeFailed,
			LastError:     err.Error(),
			RouteCount:    len(extension.Manifest.Routes),
			HookCount:     len(extensions.DeclaredManifestEvents(extension.Manifest)),
			EventCount:    len(extensions.DeclaredManifestEvents(extension.Manifest)),
			ProviderCount: len(extension.Manifest.Providers),
		})
		return err
	}
	now := time.Now().UTC()
	m.mu.Lock()
	m.targets[extension.ID] = target
	m.running[extension.ID] = extension
	m.statuses[extension.ID] = extensions.RuntimeStatus{
		State:         extensions.RuntimeRunning,
		StartedAt:     &now,
		RouteCount:    len(extension.Manifest.Routes),
		HookCount:     len(extensions.DeclaredManifestEvents(extension.Manifest)),
		EventCount:    len(extensions.DeclaredManifestEvents(extension.Manifest)),
		ProviderCount: len(extension.Manifest.Providers),
	}
	m.mu.Unlock()
	m.hooks.Register(extension)
	return nil
}

func (m *Manager) Stop(ctx context.Context, extension extensions.Extension) error {
	err := m.starter.Stop(ctx, extension)
	m.mu.Lock()
	delete(m.targets, extension.ID)
	delete(m.running, extension.ID)
	m.statuses[extension.ID] = extensions.RuntimeStatus{
		State:         extensions.RuntimeStopped,
		RouteCount:    len(extension.Manifest.Routes),
		HookCount:     len(extensions.DeclaredManifestEvents(extension.Manifest)),
		EventCount:    len(extensions.DeclaredManifestEvents(extension.Manifest)),
		ProviderCount: len(extension.Manifest.Providers),
	}
	m.mu.Unlock()
	m.hooks.Unregister(extension.ID)
	return err
}

func (m *Manager) Status(_ context.Context, extension extensions.Extension) extensions.RuntimeStatus {
	m.mu.RLock()
	status, ok := m.statuses[extension.ID]
	m.mu.RUnlock()
	if ok {
		return status
	}
	return extensions.RuntimeStatus{
		State:         extensions.RuntimeStopped,
		RouteCount:    len(extension.Manifest.Routes),
		HookCount:     len(extensions.DeclaredManifestEvents(extension.Manifest)),
		EventCount:    len(extensions.DeclaredManifestEvents(extension.Manifest)),
		ProviderCount: len(extension.Manifest.Providers),
	}
}

func (m *Manager) RouteTarget(extensionID string) (RouteTarget, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	target, ok := m.targets[extensionID]
	return target, ok
}

func (m *Manager) SendMail(ctx context.Context, extensionID string, request MailProviderRequest) (MailProviderResponse, error) {
	invoker, ok := m.starter.(MailProviderInvoker)
	if !ok {
		return MailProviderResponse{}, extensions.ErrRuntimeUnavailable
	}
	m.mu.RLock()
	_, running := m.running[extensionID]
	m.mu.RUnlock()
	if !running {
		return MailProviderResponse{}, extensions.ErrRuntimeUnavailable
	}
	return invoker.SendMail(ctx, extensionID, request)
}

func (m *Manager) RefreshMailProvider(ctx context.Context, extensionID string) error {
	m.mu.RLock()
	extension, ok := m.running[extensionID]
	m.mu.RUnlock()
	if !ok {
		return extensions.ErrRuntimeUnavailable
	}
	if err := m.starter.Stop(ctx, extension); err != nil {
		return err
	}
	_, err := m.starter.Start(ctx, extension)
	return err
}

func (m *Manager) Reconcile(ctx context.Context, items []extensions.Extension) {
	enabled := map[string]extensions.Extension{}
	for _, item := range items {
		if item.Type == extensions.TypePlugin && item.Status == extensions.StatusEnabled && item.Manifest.Backend.Entry != "" {
			enabled[item.ID] = item
			_ = m.Start(ctx, item)
		}
	}
	m.mu.RLock()
	running := make([]extensions.Extension, 0, len(m.running))
	for id, item := range m.running {
		if _, ok := enabled[id]; !ok {
			running = append(running, item)
		}
	}
	m.mu.RUnlock()
	for _, item := range running {
		_ = m.Stop(ctx, item)
	}
}

func (m *Manager) Close(ctx context.Context) {
	m.mu.RLock()
	running := make([]extensions.Extension, 0, len(m.running))
	for _, item := range m.running {
		running = append(running, item)
	}
	m.mu.RUnlock()
	for _, item := range running {
		_ = m.Stop(ctx, item)
	}
}

func (m *Manager) EmitHook(ctx context.Context, name string, payload map[string]any) {
	m.Emit(ctx, appevents.NewEnvelope(name, payload))
}

func (m *Manager) Emit(ctx context.Context, envelope appevents.Envelope) appevents.Result {
	envelope = normalizeEnvelope(envelope)
	definition, ok := appevents.FindDefinition(envelope.Name)
	if !ok {
		return appevents.Result{OK: false, Reason: "extension.event_unknown", Message: "Unknown extension event."}
	}
	if envelope.Kind == "" {
		envelope.Kind = definition.Kind
	}
	if envelope.Kind != definition.Kind {
		return appevents.Result{OK: false, Reason: "extension.event_kind_invalid", Message: "Extension event kind does not match the event definition."}
	}
	if envelope.Kind == appevents.KindObserve {
		return m.emitObserve(ctx, envelope)
	}
	return m.emitSync(ctx, envelope)
}

func (m *Manager) Deliver(ctx context.Context, extensionID string, deliveryID int64, envelope appevents.Envelope) appevents.Result {
	envelope = normalizeEnvelope(envelope)
	extension, ok := m.runningExtension(extensionID)
	if !ok {
		result := appevents.Result{OK: false, Reason: "extension.runtime_unavailable", Message: "Plugin runtime is not available."}
		m.finishDelivery(ctx, deliveryID, extensions.DeliveryFailed, result, 1)
		return result
	}
	m.updateDelivery(ctx, extensions.EventDeliveryUpdateInput{
		ID:           deliveryID,
		Status:       extensions.DeliveryRunning,
		AttemptCount: 1,
	})
	result := hookResultToEventResult(m.invoke(ctx, extension, hookInputFromEnvelope(envelope, deliveryID)))
	status := extensions.DeliverySucceeded
	if !result.OK {
		status = extensions.DeliveryFailed
	}
	m.finishDelivery(ctx, deliveryID, status, result, 1)
	return result
}

func (m *Manager) emitObserve(ctx context.Context, envelope appevents.Envelope) appevents.Result {
	listeners := m.hooks.Listeners(envelope.Name)
	for _, listener := range listeners {
		deliveryID := int64(0)
		if m.deliveryStore != nil {
			delivery, err := m.deliveryStore.CreateEventDelivery(ctx, extensions.EventDeliveryInput{
				ExtensionID:   listener.ID,
				EventName:     envelope.Name,
				EventKind:     envelope.Kind,
				Status:        extensions.DeliveryQueued,
				CorrelationID: envelope.CorrelationID,
			})
			if err != nil {
				return appevents.Result{OK: false, Reason: "extension.delivery_record_failed", Message: err.Error()}
			}
			deliveryID = delivery.ID
		}
		args := EventDeliveryArgs{
			DeliveryID:    deliveryID,
			ExtensionID:   listener.ID,
			EventName:     envelope.Name,
			EventKind:     envelope.Kind,
			CorrelationID: envelope.CorrelationID,
			Payload:       envelope.Payload,
			PatchFields:   envelope.PatchFields,
		}
		if m.dispatcher != nil {
			if err := m.dispatcher.Enqueue(ctx, args, args.EnqueueOptions()); err != nil {
				result := appevents.Result{OK: false, Reason: "extension.delivery_enqueue_failed", Message: err.Error()}
				m.finishDelivery(ctx, deliveryID, extensions.DeliveryFailed, result, 0)
				return result
			}
			continue
		}
		m.Deliver(ctx, listener.ID, deliveryID, envelope)
	}
	return appevents.Result{OK: true}
}

func (m *Manager) emitSync(ctx context.Context, envelope appevents.Envelope) appevents.Result {
	result := appevents.Result{OK: true}
	for _, listener := range m.hooks.Listeners(envelope.Name) {
		current := hookResultToEventResult(m.invoke(ctx, listener, hookInputFromEnvelope(envelope, 0)))
		if !current.OK {
			return current
		}
		if envelope.Kind == appevents.KindFilter && len(current.Patch) > 0 {
			if !patchAllowed(current.Patch, envelope.PatchFields) {
				return appevents.Result{OK: false, Reason: "extension.patch_forbidden", Message: "Plugin returned a patch field that is not allowed for this event."}
			}
			if result.Patch == nil {
				result.Patch = map[string]any{}
			}
			for key, value := range current.Patch {
				result.Patch[key] = value
			}
		}
	}
	return result
}

func (m *Manager) invoke(ctx context.Context, extension extensions.Extension, input HookInput) HookResult {
	if m.hooks == nil || m.hooks.invoker == nil {
		return HookResult{OK: true}
	}
	if input.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, input.Timeout)
		defer cancel()
	}
	return m.hooks.invoker.InvokeHook(ctx, extension, input)
}

func (m *Manager) runningExtension(extensionID string) (extensions.Extension, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	extension, ok := m.running[extensionID]
	return extension, ok
}

func (m *Manager) updateDelivery(ctx context.Context, input extensions.EventDeliveryUpdateInput) {
	if m.deliveryStore == nil || input.ID == 0 {
		return
	}
	_ = m.deliveryStore.UpdateEventDelivery(ctx, input)
}

func (m *Manager) finishDelivery(ctx context.Context, deliveryID int64, status string, result appevents.Result, attempts int) {
	m.updateDelivery(ctx, extensions.EventDeliveryUpdateInput{
		ID:           deliveryID,
		Status:       status,
		Reason:       result.Reason,
		Message:      result.Message,
		AttemptCount: attempts,
		Completed:    true,
	})
}

func (m *Manager) setStatus(extension extensions.Extension, status extensions.RuntimeStatus) {
	m.mu.Lock()
	m.statuses[extension.ID] = status
	m.mu.Unlock()
}

func normalizeEnvelope(envelope appevents.Envelope) appevents.Envelope {
	if envelope.ID == "" {
		envelope.ID = appevents.NewID()
	}
	if envelope.CorrelationID == "" {
		envelope.CorrelationID = envelope.ID
	}
	if envelope.OccurredAt.IsZero() {
		envelope.OccurredAt = time.Now().UTC()
	}
	if envelope.Kind == "" {
		if definition, ok := appevents.FindDefinition(envelope.Name); ok {
			envelope.Kind = definition.Kind
			envelope.PatchFields = append(envelope.PatchFields, definition.PatchFields...)
		}
	}
	return envelope
}

func hookResultToEventResult(result HookResult) appevents.Result {
	return appevents.Result{
		OK:      result.OK,
		Reason:  result.Reason,
		Message: result.Message,
		Patch:   result.Patch,
	}
}

func patchAllowed(patch map[string]any, allowed []string) bool {
	if len(patch) == 0 {
		return true
	}
	allowedSet := map[string]bool{}
	for _, field := range allowed {
		allowedSet[field] = true
	}
	for field := range patch {
		if !allowedSet[field] {
			return false
		}
	}
	return true
}

type localStarter struct{}

func (localStarter) Start(context.Context, extensions.Extension) (RouteTarget, error) {
	return RouteTarget{BaseURL: "http://127.0.0.1:0"}, nil
}

func (localStarter) Stop(context.Context, extensions.Extension) error {
	return nil
}
