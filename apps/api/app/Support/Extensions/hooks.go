package extensionsruntime

import (
	"context"
	"sync"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

type HookInput struct {
	Name    string
	Payload map[string]any
}

type HookResult struct {
	OK      bool
	Reason  string
	Message string
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
	b.mu.RLock()
	plugins := make([]extensions.Extension, 0, len(b.plugins))
	for _, plugin := range b.plugins {
		plugins = append(plugins, plugin)
	}
	b.mu.RUnlock()
	results := []HookResult{}
	for _, plugin := range plugins {
		if !declaresHook(plugin, input.Name) || b.invoker == nil {
			continue
		}
		results = append(results, b.invoker.InvokeHook(ctx, plugin, input))
	}
	return results
}

func declaresHook(extension extensions.Extension, name string) bool {
	for _, hook := range extension.Manifest.Hooks {
		if hook.Name == name {
			return true
		}
	}
	return false
}
