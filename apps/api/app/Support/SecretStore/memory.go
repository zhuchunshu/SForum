package secretstore

import (
	"context"
	"sort"
	"sync"
	"time"
)

// memoryRow is one immutable version of a secret.
type memoryRow struct {
	namespace string
	secretID  string
	version   int64
	cipher    string
	mediaType string
	purposes  []string
	updatedAt time.Time
	updatedBy string
	revoked   bool
}

// MemoryStore is a process-local Secret Store backend for tests and dev.
type MemoryStore struct {
	mu   sync.Mutex
	rows map[string][]memoryRow // key = namespace\x00secretID, ascending versions
}

// NewMemoryStore builds an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: make(map[string][]memoryRow)}
}

func memoryKey(namespace, secretID string) string {
	return namespace + "\x00" + secretID
}

func (s *MemoryStore) put(_ context.Context, row memoryRow) error {
	if s == nil {
		return ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := memoryKey(row.namespace, row.secretID)
	existing := s.rows[key]
	// Append-only versions; never mutate prior ciphertext.
	copied := append([]memoryRow(nil), existing...)
	copied = append(copied, row)
	s.rows[key] = copied
	return nil
}

func (s *MemoryStore) latest(_ context.Context, namespace, secretID string, includeRevoked bool) (memoryRow, bool) {
	if s == nil {
		return memoryRow{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := s.rows[memoryKey(namespace, secretID)]
	for i := len(rows) - 1; i >= 0; i-- {
		if includeRevoked || !rows[i].revoked {
			return cloneRow(rows[i]), true
		}
	}
	return memoryRow{}, false
}

func (s *MemoryStore) getVersion(_ context.Context, namespace, secretID string, version int64) (memoryRow, bool) {
	if s == nil || version <= 0 {
		return memoryRow{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, row := range s.rows[memoryKey(namespace, secretID)] {
		if row.version == version {
			return cloneRow(row), true
		}
	}
	return memoryRow{}, false
}

func (s *MemoryStore) listNamespace(_ context.Context, namespace string) []memoryRow {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Latest non-revoked per secret_id.
	latest := make(map[string]memoryRow)
	for key, rows := range s.rows {
		if len(rows) == 0 {
			continue
		}
		// key is namespace\x00secretID
		ns, id, ok := splitMemoryKey(key)
		if !ok || ns != namespace {
			continue
		}
		for i := len(rows) - 1; i >= 0; i-- {
			if !rows[i].revoked {
				latest[id] = cloneRow(rows[i])
				break
			}
		}
	}
	out := make([]memoryRow, 0, len(latest))
	for _, row := range latest {
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].secretID != out[j].secretID {
			return out[i].secretID < out[j].secretID
		}
		return out[i].version < out[j].version
	})
	return out
}

func (s *MemoryStore) nextVersion(_ context.Context, namespace, secretID string) int64 {
	if s == nil {
		return 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := s.rows[memoryKey(namespace, secretID)]
	if len(rows) == 0 {
		return 1
	}
	return rows[len(rows)-1].version + 1
}

func cloneRow(row memoryRow) memoryRow {
	out := row
	if len(row.purposes) > 0 {
		out.purposes = append([]string(nil), row.purposes...)
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
