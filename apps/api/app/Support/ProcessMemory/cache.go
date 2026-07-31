package processmemory

import (
	"sync"
	"time"
)

// cachedSampler 让所有调用方共享一次昂贵的进程表读取。
// 它不启动后台 goroutine；没有管理端请求时不会产生采样开销。
type cachedSampler struct {
	inner Sampler
	ttl   time.Duration
	now   func() time.Time

	mu        sync.Mutex
	expiresAt time.Time
	samples   []Sample
	err       error
}

func NewCachedSampler(inner Sampler, ttl time.Duration) Sampler {
	if ttl <= 0 {
		ttl = DefaultSampleInterval
	}
	return &cachedSampler{inner: inner, ttl: ttl, now: time.Now}
}

func (s *cachedSampler) List() ([]Sample, error) {
	if s == nil || s.inner == nil {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if !s.expiresAt.IsZero() && now.Before(s.expiresAt) {
		return cloneSamples(s.samples), s.err
	}

	samples, err := s.inner.List()
	for index := range samples {
		if samples[index].CapturedAt.IsZero() {
			samples[index].CapturedAt = now
		}
	}
	s.samples = cloneSamples(samples)
	s.err = err
	s.expiresAt = now.Add(s.ttl)
	return cloneSamples(s.samples), s.err
}

func cloneSamples(samples []Sample) []Sample {
	if len(samples) == 0 {
		return nil
	}
	return append([]Sample(nil), samples...)
}
