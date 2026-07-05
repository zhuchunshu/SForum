package extensionsruntime

import (
	"context"
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
}

type Manager struct {
	mu       sync.RWMutex
	starter  Starter
	statuses map[string]extensions.RuntimeStatus
	targets  map[string]RouteTarget
}

func NewManager(config ManagerConfig) *Manager {
	starter := config.Starter
	if starter == nil {
		starter = localStarter{}
	}
	return &Manager{starter: starter, statuses: map[string]extensions.RuntimeStatus{}, targets: map[string]RouteTarget{}}
}

func (m *Manager) Check(context.Context, extensions.Extension) error {
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
	m.statuses[extension.ID] = extensions.RuntimeStatus{
		State:         extensions.RuntimeRunning,
		StartedAt:     &now,
		RouteCount:    len(extension.Manifest.Routes),
		HookCount:     len(extension.Manifest.Hooks),
		ProviderCount: len(extension.Manifest.Providers),
	}
	m.mu.Unlock()
	return nil
}

func (m *Manager) Stop(ctx context.Context, extension extensions.Extension) error {
	err := m.starter.Stop(ctx, extension)
	m.mu.Lock()
	delete(m.targets, extension.ID)
	m.statuses[extension.ID] = extensions.RuntimeStatus{
		State:         extensions.RuntimeStopped,
		RouteCount:    len(extension.Manifest.Routes),
		HookCount:     len(extension.Manifest.Hooks),
		ProviderCount: len(extension.Manifest.Providers),
	}
	m.mu.Unlock()
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
	for _, item := range items {
		if item.Type == extensions.TypePlugin && item.Status == extensions.StatusEnabled && item.Manifest.Backend.Entry != "" {
			_ = m.Start(ctx, item)
		}
	}
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
