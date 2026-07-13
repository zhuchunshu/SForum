package extensionsruntime

import (
	"context"
	"sync"
	"time"
)

// F2.3 默认韧性参数。后续可接 runtime option，当前用稳定宿主默认。
const (
	DefaultMaxConcurrentRPC   = 4
	DefaultFailureThreshold   = 5
	DefaultCircuitOpenSeconds = 30
	DefaultMailTimeoutMS      = 15000
	// 存储 Put/Open 可跨多块 RPC；单次调用默认上限 120s（E6.2）。
	DefaultStorageTimeoutMS = 120000
)

// ResilienceConfig 控制每扩展 RPC 闸门与熔断。
type ResilienceConfig struct {
	MaxConcurrent         int
	FailureThreshold      int
	CircuitOpenFor        time.Duration
	DefaultMailTimeout    time.Duration
	DefaultStorageTimeout time.Duration
}

func (c ResilienceConfig) withDefaults() ResilienceConfig {
	if c.MaxConcurrent <= 0 {
		c.MaxConcurrent = DefaultMaxConcurrentRPC
	}
	if c.FailureThreshold <= 0 {
		c.FailureThreshold = DefaultFailureThreshold
	}
	if c.CircuitOpenFor <= 0 {
		c.CircuitOpenFor = time.Duration(DefaultCircuitOpenSeconds) * time.Second
	}
	if c.DefaultMailTimeout <= 0 {
		c.DefaultMailTimeout = time.Duration(DefaultMailTimeoutMS) * time.Millisecond
	}
	if c.DefaultStorageTimeout <= 0 {
		c.DefaultStorageTimeout = time.Duration(DefaultStorageTimeoutMS) * time.Millisecond
	}
	return c
}

// extensionGate 单扩展的并发槽 + 熔断状态。
type extensionGate struct {
	sem               chan struct{}
	mu                sync.Mutex
	consecutiveFails  int
	circuitOpenUntil  time.Time
	lastFailureReason string
	lastFailureAt     time.Time
	hasLastFailure    bool
	activeCalls       int
}

type gateSnapshot struct {
	CircuitOpen         bool
	CircuitOpenUntil    *time.Time
	ConsecutiveFailures int
	LastFailureReason   string
	LastFailureAt       *time.Time
	ActiveCalls         int
	MaxConcurrent       int
}

// resilienceHub 管理所有扩展的闸门。
type resilienceHub struct {
	cfg   ResilienceConfig
	mu    sync.Mutex
	gates map[string]*extensionGate
}

func newResilienceHub(cfg ResilienceConfig) *resilienceHub {
	return &resilienceHub{
		cfg:   cfg.withDefaults(),
		gates: map[string]*extensionGate{},
	}
}

func (h *resilienceHub) gate(extensionID string) *extensionGate {
	h.mu.Lock()
	defer h.mu.Unlock()
	g, ok := h.gates[extensionID]
	if !ok {
		g = &extensionGate{sem: make(chan struct{}, h.cfg.MaxConcurrent)}
		h.gates[extensionID] = g
	}
	return g
}

func (h *resilienceHub) remove(extensionID string) {
	h.mu.Lock()
	delete(h.gates, extensionID)
	h.mu.Unlock()
}

// tryEnter 在熔断开启时拒绝；否则占用一个并发槽（可被 ctx 取消）。
// failOpen=true 时熔断拒绝用 skip 语义（reason 仍为 circuit_open）。
func (h *resilienceHub) tryEnter(ctx context.Context, extensionID string) (release func(success bool, reason string), rejectedReason string) {
	g := h.gate(extensionID)
	now := time.Now()

	g.mu.Lock()
	if !g.circuitOpenUntil.IsZero() && now.Before(g.circuitOpenUntil) {
		g.mu.Unlock()
		return nil, "extension.circuit_open"
	}
	// half-open：冷却结束清零 openUntil，允许探测。
	if !g.circuitOpenUntil.IsZero() && !now.Before(g.circuitOpenUntil) {
		g.circuitOpenUntil = time.Time{}
	}
	g.mu.Unlock()

	select {
	case g.sem <- struct{}{}:
		// 占用成功
	case <-ctx.Done():
		return nil, "extension.hook_timeout"
	}

	g.mu.Lock()
	g.activeCalls++
	g.mu.Unlock()

	released := false
	return func(success bool, reason string) {
		if released {
			return
		}
		released = true
		<-g.sem
		g.mu.Lock()
		if g.activeCalls > 0 {
			g.activeCalls--
		}
		if success {
			g.consecutiveFails = 0
			g.circuitOpenUntil = time.Time{}
		} else {
			g.consecutiveFails++
			g.lastFailureReason = reason
			g.lastFailureAt = time.Now().UTC()
			g.hasLastFailure = true
			if g.consecutiveFails >= h.cfg.FailureThreshold {
				g.circuitOpenUntil = time.Now().UTC().Add(h.cfg.CircuitOpenFor)
			}
		}
		g.mu.Unlock()
	}, ""
}

func (h *resilienceHub) snapshot(extensionID string) gateSnapshot {
	g := h.gate(extensionID)
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	snap := gateSnapshot{
		ConsecutiveFailures: g.consecutiveFails,
		LastFailureReason:   g.lastFailureReason,
		ActiveCalls:         g.activeCalls,
		MaxConcurrent:       h.cfg.MaxConcurrent,
	}
	if g.hasLastFailure {
		t := g.lastFailureAt
		snap.LastFailureAt = &t
	}
	if !g.circuitOpenUntil.IsZero() && now.Before(g.circuitOpenUntil) {
		snap.CircuitOpen = true
		until := g.circuitOpenUntil
		snap.CircuitOpenUntil = &until
	}
	return snap
}
