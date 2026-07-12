package pages

import (
	"context"
	"strings"
	"sync"
)

// MemoryStore 进程内绑定存储（测试与轻量启动）；生产用 PostgresStore。
type MemoryStore struct {
	mu       sync.RWMutex
	bindings map[string]ProviderBinding
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{bindings: map[string]ProviderBinding{}}
}

func (s *MemoryStore) ListBindings(_ context.Context) ([]ProviderBinding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ProviderBinding, 0, len(s.bindings))
	for _, b := range s.bindings {
		out = append(out, b)
	}
	return out, nil
}

func (s *MemoryStore) GetBinding(_ context.Context, pageID string) (ProviderBinding, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.bindings[pageID]
	return b, ok, nil
}

func (s *MemoryStore) UpsertBinding(_ context.Context, binding ProviderBinding) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Fiber 等框架可能复用请求缓冲；持久化前克隆字符串，避免 map key 被后续请求改写。
	binding = cloneBinding(binding)
	s.bindings[binding.PageID] = binding
	return nil
}

func (s *MemoryStore) DeleteBinding(_ context.Context, pageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.bindings, pageID)
	return nil
}

func cloneBinding(b ProviderBinding) ProviderBinding {
	b.PageID = strings.Clone(b.PageID)
	b.ExtensionID = strings.Clone(b.ExtensionID)
	b.ContributionID = strings.Clone(b.ContributionID)
	b.Version = strings.Clone(b.Version)
	b.PackageDigest = strings.Clone(b.PackageDigest)
	b.ContractVersion = strings.Clone(b.ContractVersion)
	b.TemplatePath = strings.Clone(b.TemplatePath)
	return b
}
