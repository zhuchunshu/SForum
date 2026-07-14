package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

type HookInput struct {
	DeclarationID   string
	Name            string
	Kind            string
	ContractVersion string
	InputSchema     string
	ResultSchema    string
	FailurePolicy   string
	DeliveryID      int64
	CorrelationID   string
	Timeout         time.Duration
	Payload         map[string]any
	PatchFields     []string
}

type HookResult struct {
	OK      bool
	Reason  string
	Message string
	Patch   map[string]any
	Result  map[string]any
}

type HookInvoker interface {
	InvokeHook(ctx context.Context, extension extensions.Extension, input HookInput) HookResult
}

type HookInvokerFunc func(context.Context, extensions.Extension, HookInput) HookResult

func (fn HookInvokerFunc) InvokeHook(ctx context.Context, extension extensions.Extension, input HookInput) HookResult {
	return fn(ctx, extension, input)
}

type HookBusConfig struct {
	Invoker HookInvoker
}

type HookBus struct {
	mu            sync.RWMutex
	invoker       HookInvoker
	plugins       map[string]hookRuntimeRegistration
	registry      *VersionedHookRegistry
	providerSlots *VersionedProviderSlotRegistry
}

type hookRuntimeRegistration struct {
	extension  extensions.Extension
	instanceID string
}

// HookRuntimeSnapshot binds the visible hook declarations to one exact
// runtime. InstanceID is empty only for the legacy compatibility path.
type HookRuntimeSnapshot struct {
	Extension  extensions.Extension
	InstanceID string
}

func NewHookBus(config HookBusConfig) *HookBus {
	return &HookBus{
		invoker: config.Invoker, plugins: map[string]hookRuntimeRegistration{},
		registry: NewVersionedHookRegistry(), providerSlots: NewVersionedProviderSlotRegistry(),
	}
}

func (b *HookBus) Register(extension extensions.Extension) {
	if extension.Status != extensions.StatusEnabled {
		return
	}
	_ = b.RegisterRuntime(extension, "")
}

// RegisterRuntime atomically replaces one extension's hook declarations with
// the declarations owned by an exact process. Manager admission remains the
// execution gate, so registering a drained target does not open ordinary calls.
func (b *HookBus) RegisterRuntime(extension extensions.Extension, instanceID string) error {
	if extension.Type != extensions.TypePlugin {
		return nil
	}
	b.mu.RLock()
	previous, hadPrevious := b.plugins[extension.ID]
	b.mu.RUnlock()
	if publishesVersionedHookSnapshot(extension, instanceID) {
		if err := b.providerSlots.ReplaceRuntime(extension, instanceID); err != nil {
			return err
		}
		if err := b.registry.ReplaceRuntime(extension, instanceID); err != nil {
			if hadPrevious && publishesVersionedHookSnapshot(previous.extension, previous.instanceID) {
				_ = b.providerSlots.ReplaceRuntime(previous.extension, previous.instanceID)
			} else {
				_, _ = b.providerSlots.RemoveRuntime(extension.ID, instanceID)
			}
			return err
		}
	}
	b.mu.Lock()
	b.plugins[extension.ID] = hookRuntimeRegistration{extension: cloneHookExtension(extension), instanceID: instanceID}
	b.mu.Unlock()
	return nil
}

func (b *HookBus) Unregister(extensionID string) {
	b.mu.RLock()
	current, ok := b.plugins[extensionID]
	b.mu.RUnlock()
	if ok {
		_ = b.UnregisterRuntime(extensionID, current.instanceID)
	}
}

// UnregisterRuntime removes declarations only while they still belong to the
// stopping process. A retained source can therefore never clear its published
// replacement.
func (b *HookBus) UnregisterRuntime(extensionID, instanceID string) bool {
	removed, _ := b.unregisterRuntime(extensionID, instanceID)
	return removed
}

func (b *HookBus) unregisterRuntime(extensionID, instanceID string) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	current, ok := b.plugins[extensionID]
	if !ok || current.instanceID != instanceID {
		return false, nil
	}
	if publishesVersionedHookSnapshot(current.extension, current.instanceID) {
		if err := b.providerSlots.ValidateRemoveRuntime(extensionID, instanceID); err != nil {
			return false, err
		}
		if err := b.registry.ValidateRemoveRuntime(extensionID, instanceID); err != nil {
			return false, err
		}
		removed, err := b.registry.RemoveRuntime(extensionID, instanceID)
		if err != nil || !removed {
			return false, err
		}
		providerRemoved, providerErr := b.providerSlots.RemoveRuntime(extensionID, instanceID)
		if providerErr != nil || !providerRemoved {
			_ = b.registry.ReplaceRuntime(current.extension, current.instanceID)
			return false, providerErr
		}
	}
	delete(b.plugins, extensionID)
	return true, nil
}

func (b *HookBus) validateUnregisterRuntime(extension extensions.Extension, instanceID string) error {
	b.mu.RLock()
	current, ok := b.plugins[extension.ID]
	b.mu.RUnlock()
	if !ok || current.instanceID != instanceID {
		if publishesVersionedHookSnapshot(extension, instanceID) {
			return fmt.Errorf("%w: exact hook runtime %s/%s is not published", ErrHookRegistryConflict, extension.ID, instanceID)
		}
		return nil
	}
	if !publishesVersionedHookSnapshot(current.extension, current.instanceID) {
		return nil
	}
	return errors.Join(
		b.registry.ValidateRemoveRuntime(extension.ID, instanceID),
		b.providerSlots.ValidateRemoveRuntime(extension.ID, instanceID),
	)
}

func (b *HookBus) restoreRuntime(targetID, targetInstanceID string, previous HookRuntimeSnapshot, hadPrevious bool) error {
	if hadPrevious {
		return b.RegisterRuntime(previous.Extension, previous.InstanceID)
	}
	_, err := b.unregisterRuntime(targetID, targetInstanceID)
	return err
}

func (b *HookBus) VersionedRegistry() *VersionedHookRegistry { return b.registry }

func (b *HookBus) ProviderSlots() *VersionedProviderSlotRegistry { return b.providerSlots }

func (b *HookBus) RuntimeSnapshot(extensionID string) (HookRuntimeSnapshot, bool) {
	b.mu.RLock()
	current, ok := b.plugins[extensionID]
	b.mu.RUnlock()
	if !ok {
		return HookRuntimeSnapshot{}, false
	}
	return HookRuntimeSnapshot{Extension: current.extension, InstanceID: current.instanceID}, true
}

func hasVersionedPluginHooks(extension extensions.Extension) bool {
	for _, hook := range extension.Manifest.Hooks {
		if isVersionedPluginHook(hook) {
			return true
		}
	}
	return false
}

func hasVersionedProviderSlots(extension extensions.Extension) bool {
	for _, provider := range extension.Manifest.Providers {
		if isVersionedProviderSlot(provider) {
			return true
		}
	}
	return false
}

func hasVersionedRuntimeContracts(extension extensions.Extension) bool {
	return hasVersionedPluginHooks(extension) || hasVersionedProviderSlots(extension)
}

func publishesVersionedHookSnapshot(extension extensions.Extension, instanceID string) bool {
	return instanceID != "" && extension.Manifest.Backend.ProtocolVersion == 2 &&
		(strings.TrimSpace(extension.PackageDigest) != "" || hasVersionedRuntimeContracts(extension))
}

func runtimeManifestCounts(manifest extensions.Manifest) (hooks, events int) {
	// events 不吸收 legacy hooks，避免同一声明同时出现在两个运行状态计数。
	return len(manifest.Hooks), len(manifest.Events)
}

func (b *HookBus) Emit(ctx context.Context, input HookInput) []HookResult {
	plugins := b.listeners(input.Name)
	results := []HookResult{}
	for _, plugin := range plugins {
		if b.invoker == nil {
			continue
		}
		results = append(results, b.invoker.InvokeHook(ctx, plugin, input))
	}
	return results
}

func (b *HookBus) Listeners(name string) []extensions.Extension {
	return b.listeners(name)
}

func (b *HookBus) listeners(name string) []extensions.Extension {
	b.mu.RLock()
	plugins := make([]extensions.Extension, 0, len(b.plugins))
	for _, registration := range b.plugins {
		if declaresHook(registration.extension, name) {
			plugins = append(plugins, registration.extension)
		}
	}
	b.mu.RUnlock()
	sort.Slice(plugins, func(left int, right int) bool {
		return plugins[left].ID < plugins[right].ID
	})
	return plugins
}

func declaresHook(extension extensions.Extension, name string) bool {
	for _, event := range extensions.DeclaredManifestEvents(extension.Manifest) {
		if event.Name == name {
			return true
		}
	}
	return false
}

func hookInputFromEnvelope(envelope appevents.Envelope, deliveryID int64) HookInput {
	timeout := time.Duration(0)
	if definition, ok := appevents.FindDefinition(envelope.Name); ok && definition.TimeoutMS > 0 {
		timeout = time.Duration(definition.TimeoutMS) * time.Millisecond
	}
	return HookInput{
		Name:          envelope.Name,
		Kind:          envelope.Kind,
		DeliveryID:    deliveryID,
		CorrelationID: envelope.CorrelationID,
		Timeout:       timeout,
		Payload:       envelope.Payload,
		PatchFields:   envelope.PatchFields,
	}
}
