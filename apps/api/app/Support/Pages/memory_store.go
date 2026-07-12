package pages

import (
	"context"
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
	s.bindings[binding.PageID] = binding
	return nil
}

func (s *MemoryStore) DeleteBinding(_ context.Context, pageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.bindings, pageID)
	return nil
}
