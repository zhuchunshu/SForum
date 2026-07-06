package extensionsruntime

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

type Starter interface {
	Start(ctx context.Context, extension extensions.Extension) (RouteTarget, error)
	Stop(ctx context.Context, extension extensions.Extension) error
}

type ManagerConfig struct {
	Starter Starter
	HookBus *HookBus
}

type Manager struct {
	mu       sync.RWMutex
	starter  Starter
	statuses map[string]extensions.RuntimeStatus
	targets  map[string]RouteTarget
	running  map[string]extensions.Extension
	hooks    *HookBus
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
	return &Manager{starter: starter, statuses: map[string]extensions.RuntimeStatus{}, targets: map[string]RouteTarget{}, running: map[string]extensions.Extension{}, hooks: hooks}
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
		HookCount:     len(extension.Manifest.Hooks),
		ProviderCount: len(extension.Manifest.Providers),
	})
	target, err := m.starter.Start(ctx, extension)
	if err != nil {
		m.setStatus(extension, extensions.RuntimeStatus{
			State:         extensions.RuntimeFailed,
			LastError:     err.Error(),
			RouteCount:    len(extension.Manifest.Routes),
			HookCount:     len(extension.Manifest.Hooks),
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
		HookCount:     len(extension.Manifest.Hooks),
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
		HookCount:     len(extension.Manifest.Hooks),
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
		HookCount:     len(extension.Manifest.Hooks),
		ProviderCount: len(extension.Manifest.Providers),
	}
}

func (m *Manager) RouteTarget(extensionID string) (RouteTarget, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	target, ok := m.targets[extensionID]
	return target, ok
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
	m.hooks.Emit(ctx, HookInput{Name: name, Payload: payload})
}

func (m *Manager) setStatus(extension extensions.Extension, status extensions.RuntimeStatus) {
	m.mu.Lock()
	m.statuses[extension.ID] = status
	m.mu.Unlock()
}

type localStarter struct{}

func (localStarter) Start(context.Context, extensions.Extension) (RouteTarget, error) {
	return RouteTarget{BaseURL: "http://127.0.0.1:0"}, nil
}

func (localStarter) Stop(context.Context, extensions.Extension) error {
	return nil
}
