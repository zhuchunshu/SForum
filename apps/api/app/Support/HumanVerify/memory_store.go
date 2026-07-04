package humanverify

import (
	"context"
	"sync"
	"time"
)

type MemoryStore struct {
	mu      sync.Mutex
	used    map[string]time.Time
	buckets map[string]memoryRateBucket
}

type memoryRateBucket struct {
	count     int
	expiresAt time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		used:    map[string]time.Time{},
		buckets: map[string]memoryRateBucket{},
	}
}

func (s *MemoryStore) MarkUsed(_ context.Context, key string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if expiresAt, ok := s.used[key]; ok && now.Before(expiresAt) {
		return ErrReplayed
	}
	s.used[key] = now.Add(ttl)
	return nil
}

func (s *MemoryStore) IncrementRate(_ context.Context, key string, window time.Duration, limit int) (bool, error) {
	if limit <= 0 {
		return false, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	bucket := s.buckets[key]
	if bucket.expiresAt.IsZero() || now.After(bucket.expiresAt) {
		bucket = memoryRateBucket{expiresAt: now.Add(window)}
	}
	bucket.count++
	s.buckets[key] = bucket

	return bucket.count > limit, nil
}
