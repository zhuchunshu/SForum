package extensionsruntime

import (
	"context"
	"sort"
	"sync"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

type HookInput struct {
	Name          string
	Kind          string
	DeliveryID    int64
	CorrelationID string
	Timeout       time.Duration
	Payload       map[string]any
	PatchFields   []string
}

type HookResult struct {
	OK      bool
	Reason  string
	Message string
	Patch   map[string]any
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
	mu      sync.RWMutex
	invoker HookInvoker
	plugins map[string]extensions.Extension
}

func NewHookBus(config HookBusConfig) *HookBus {
	return &HookBus{invoker: config.Invoker, plugins: map[string]extensions.Extension{}}
}

func (b *HookBus) Register(extension extensions.Extension) {
	if extension.Type != extensions.TypePlugin || extension.Status != extensions.StatusEnabled {
		return
	}
	b.mu.Lock()
	b.plugins[extension.ID] = extension
	b.mu.Unlock()
}

func (b *HookBus) Unregister(extensionID string) {
	b.mu.Lock()
	delete(b.plugins, extensionID)
	b.mu.Unlock()
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
	for _, plugin := range b.plugins {
		if declaresHook(plugin, name) {
			plugins = append(plugins, plugin)
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
