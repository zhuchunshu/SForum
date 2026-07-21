package secretstore

import (
	"context"
	"sort"
	"sync"
)

// MemoryStore is a process-local Secret Store backend for unit tests only.
// Production must use PostgresStore.
type MemoryStore struct {
	mu   sync.Mutex
	rows map[string][]Row // key = namespace\x00secretID, ascending versions
}

// NewMemoryStore builds an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: make(map[string][]Row)}
}

func memoryKey(namespace, secretID string) string {
	return namespace + "\x00" + secretID
}

// Append implements Store with process-local locking for version uniqueness.
func (s *MemoryStore) Append(_ context.Context, row Row) (Row, error) {
	if s == nil {
		return Row{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := memoryKey(row.Namespace, row.SecretID)
	existing := s.rows[key]
	version := int64(1)
	if len(existing) > 0 {
		version = existing[len(existing)-1].Version + 1
	}
	row.Version = version
	copied := append([]Row(nil), existing...)
	copied = append(copied, cloneRow(row))
	s.rows[key] = copied
	return cloneRow(row), nil
}

// Latest implements Store.
func (s *MemoryStore) Latest(_ context.Context, namespace, secretID string, includeRevoked bool) (Row, bool, error) {
	if s == nil {
		return Row{}, false, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := s.rows[memoryKey(namespace, secretID)]
	for i := len(rows) - 1; i >= 0; i-- {
		if includeRevoked || !rows[i].Revoked {
			return cloneRow(rows[i]), true, nil
		}
	}
	return Row{}, false, nil
}

// GetVersion implements Store.
func (s *MemoryStore) GetVersion(_ context.Context, namespace, secretID string, version int64) (Row, bool, error) {
	if s == nil || version <= 0 {
		return Row{}, false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, row := range s.rows[memoryKey(namespace, secretID)] {
		if row.Version == version {
			return cloneRow(row), true, nil
		}
	}
	return Row{}, false, nil
}

// ListNamespace implements Store.
func (s *MemoryStore) ListNamespace(_ context.Context, namespace string) ([]Row, error) {
	if s == nil {
		return nil, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	latest := make(map[string]Row)
	for key, rows := range s.rows {
		if len(rows) == 0 {
			continue
		}
		ns, id, ok := splitMemoryKey(key)
		if !ok || ns != namespace {
			continue
		}
		for i := len(rows) - 1; i >= 0; i-- {
			if !rows[i].Revoked {
				latest[id] = cloneRow(rows[i])
				break
			}
		}
	}
	out := make([]Row, 0, len(latest))
	for _, row := range latest {
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SecretID != out[j].SecretID {
			return out[i].SecretID < out[j].SecretID
		}
		return out[i].Version < out[j].Version
	})
	return out, nil
}

// MemoryAuditStore is a process-local audit ring for tests.
type MemoryAuditStore struct {
	mu     sync.Mutex
	events []AuditEvent
}

// NewMemoryAuditStore builds an empty audit ring.
func NewMemoryAuditStore() *MemoryAuditStore {
	return &MemoryAuditStore{}
}

// AppendAudit implements AuditStore.
func (s *MemoryAuditStore) AppendAudit(_ context.Context, event AuditEvent) error {
	if s == nil {
		return ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	if len(s.events) > MaxAuditRing*4 {
		s.events = append([]AuditEvent(nil), s.events[len(s.events)-MaxAuditRing*2:]...)
	}
	return nil
}

// ListRecentAudit implements AuditStore.
func (s *MemoryAuditStore) ListRecentAudit(_ context.Context, limit int) ([]AuditEvent, error) {
	if s == nil {
		return nil, nil
	}
	if limit <= 0 || limit > MaxAuditRing {
		limit = MaxAuditRing
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.events) == 0 {
		return nil, nil
	}
	start := 0
	if len(s.events) > limit {
		start = len(s.events) - limit
	}
	return append([]AuditEvent(nil), s.events[start:]...), nil
}

func cloneRow(row Row) Row {
	out := row
	if len(row.Purposes) > 0 {
		out.Purposes = append([]string(nil), row.Purposes...)
	}
	return out
}

func splitMemoryKey(key string) (namespace, secretID string, ok bool) {
	for i := 0; i < len(key); i++ {
		if key[i] == 0 {
			return key[:i], key[i+1:], true
		}
	}
	return "", "", false
}
