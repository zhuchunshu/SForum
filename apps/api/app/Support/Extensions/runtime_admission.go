package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	ErrRuntimeAdmissionInvalid  = errors.New("extension runtime admission is invalid")
	ErrRuntimeAdmissionDraining = errors.New("extension runtime is draining")
	ErrRuntimeAdmissionForced   = errors.New("extension runtime admission was force-cancelled")
	ErrRuntimeAdmissionBusy     = errors.New("extension runtime lifecycle cleanup is still active")
)

// RuntimeCallClass 区分 drain 时需要独立观测的调用来源。
type RuntimeCallClass string

const (
	RuntimeCallRoute            RuntimeCallClass = "route"
	RuntimeCallPage             RuntimeCallClass = "page"
	RuntimeCallHook             RuntimeCallClass = "hook"
	RuntimeCallProvider         RuntimeCallClass = "provider"
	RuntimeCallService          RuntimeCallClass = "service"
	RuntimeCallHost             RuntimeCallClass = "host_call"
	RuntimeCallJob              RuntimeCallClass = "job"
	RuntimeCallSchedule         RuntimeCallClass = "schedule"
	RuntimeCallLifecycleCleanup RuntimeCallClass = "lifecycle_cleanup"
)

// RuntimeInstanceIdentity 把 gate 固定到一个进程实例，防止旧实例 drain 新实例。
type RuntimeInstanceIdentity struct {
	ExtensionID string
	InstanceID  string
}

// RuntimeAdmissionSnapshot 是某一时刻仍在执行的调用快照。
type RuntimeAdmissionSnapshot struct {
	Identity      RuntimeInstanceIdentity
	Draining      bool
	Forced        bool
	ForceCause    error
	ActiveTotal   int
	ActiveByClass map[RuntimeCallClass]int
}

// RuntimeAdmissionGate 在进程真正停止前关闭新入口并等待已获准调用退出。
type RuntimeAdmissionGate struct {
	identity RuntimeInstanceIdentity

	mu         sync.Mutex
	draining   bool
	forced     bool
	forceCause error
	nextID     uint64
	active     map[uint64]runtimeAdmissionCall
	idle       chan struct{}
}

type runtimeAdmissionCall struct {
	class  RuntimeCallClass
	cancel context.CancelCauseFunc
}

// RuntimeAdmissionLease 持有一次已获准调用；Release 可安全重复调用。
type RuntimeAdmissionLease struct {
	Context context.Context
	Class   RuntimeCallClass

	gate   *RuntimeAdmissionGate
	id     uint64
	cancel context.CancelCauseFunc
	once   sync.Once
}

func NewRuntimeAdmissionGate(identity RuntimeInstanceIdentity) (*RuntimeAdmissionGate, error) {
	identity.ExtensionID = strings.TrimSpace(identity.ExtensionID)
	identity.InstanceID = strings.TrimSpace(identity.InstanceID)
	if identity.ExtensionID == "" || identity.InstanceID == "" {
		return nil, fmt.Errorf("%w: extension id and instance id are required", ErrRuntimeAdmissionInvalid)
	}
	idle := make(chan struct{})
	close(idle)
	return &RuntimeAdmissionGate{
		identity: identity,
		active:   make(map[uint64]runtimeAdmissionCall),
		idle:     idle,
	}, nil
}

func (g *RuntimeAdmissionGate) Identity() RuntimeInstanceIdentity {
	if g == nil {
		return RuntimeInstanceIdentity{}
	}
	return g.identity
}

// Acquire 对普通调用执行 admission；drain 后仅显式 lifecycle cleanup 可进入。
func (g *RuntimeAdmissionGate) Acquire(parent context.Context, class RuntimeCallClass) (*RuntimeAdmissionLease, error) {
	if g == nil || parent == nil || strings.TrimSpace(string(class)) == "" {
		return nil, ErrRuntimeAdmissionInvalid
	}
	if err := parent.Err(); err != nil {
		return nil, err
	}

	g.mu.Lock()
	if g.forced {
		err := runtimeAdmissionForcedError(g.forceCause)
		g.mu.Unlock()
		return nil, err
	}
	if g.draining && class != RuntimeCallLifecycleCleanup {
		g.mu.Unlock()
		return nil, ErrRuntimeAdmissionDraining
	}
	ctx, cancel := context.WithCancelCause(parent)
	if len(g.active) == 0 {
		g.idle = make(chan struct{})
	}
	g.nextID++
	id := g.nextID
	g.active[id] = runtimeAdmissionCall{class: class, cancel: cancel}
	g.mu.Unlock()

	return &RuntimeAdmissionLease{
		Context: ctx,
		Class:   class,
		gate:    g,
		id:      id,
		cancel:  cancel,
	}, nil
}

// BeginDrain 与 Acquire 使用同一把锁；返回后不会再放行 ordinary 调用。
func (g *RuntimeAdmissionGate) BeginDrain() RuntimeAdmissionSnapshot {
	if g == nil {
		return RuntimeAdmissionSnapshot{}
	}
	g.mu.Lock()
	g.draining = true
	snapshot := g.snapshotLocked()
	g.mu.Unlock()
	return snapshot
}

// Resume 仅用于原子发布前的失败补偿；forced gate 和仍在执行 cleanup 的 gate 不可重开。
func (g *RuntimeAdmissionGate) Resume() (RuntimeAdmissionSnapshot, error) {
	if g == nil {
		return RuntimeAdmissionSnapshot{}, ErrRuntimeAdmissionInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.forced {
		return g.snapshotLocked(), runtimeAdmissionForcedError(g.forceCause)
	}
	for _, call := range g.active {
		if call.class == RuntimeCallLifecycleCleanup {
			return g.snapshotLocked(), ErrRuntimeAdmissionBusy
		}
	}
	g.draining = false
	return g.snapshotLocked(), nil
}

// Wait 等待所有已获准调用真实 Release；ctx 超时不会修改 gate 状态。
func (g *RuntimeAdmissionGate) Wait(ctx context.Context) error {
	if g == nil || ctx == nil {
		return ErrRuntimeAdmissionInvalid
	}
	for {
		g.mu.Lock()
		if len(g.active) == 0 {
			g.mu.Unlock()
			return nil
		}
		idle := g.idle
		g.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-idle:
		}
	}
}

// ForceCancel 关闭全部 admission 并取消在途 context；不伪造调用已退出。
func (g *RuntimeAdmissionGate) ForceCancel(cause error) RuntimeAdmissionSnapshot {
	if g == nil {
		return RuntimeAdmissionSnapshot{}
	}
	if cause == nil {
		cause = ErrRuntimeAdmissionForced
	}

	g.mu.Lock()
	if !g.forced {
		g.draining = true
		g.forced = true
		g.forceCause = cause
	}
	cancels := make([]context.CancelCauseFunc, 0, len(g.active))
	for _, call := range g.active {
		cancels = append(cancels, call.cancel)
	}
	snapshot := g.snapshotLocked()
	g.mu.Unlock()

	for _, cancel := range cancels {
		cancel(g.forceCause)
	}
	return snapshot
}

func (g *RuntimeAdmissionGate) Snapshot() RuntimeAdmissionSnapshot {
	if g == nil {
		return RuntimeAdmissionSnapshot{}
	}
	g.mu.Lock()
	snapshot := g.snapshotLocked()
	g.mu.Unlock()
	return snapshot
}

func (l *RuntimeAdmissionLease) Release() {
	if l == nil {
		return
	}
	l.once.Do(func() {
		l.cancel(nil)
		l.gate.release(l.id)
	})
}

func (g *RuntimeAdmissionGate) release(id uint64) {
	g.mu.Lock()
	if _, ok := g.active[id]; ok {
		delete(g.active, id)
		if len(g.active) == 0 {
			close(g.idle)
		}
	}
	g.mu.Unlock()
}

func (g *RuntimeAdmissionGate) snapshotLocked() RuntimeAdmissionSnapshot {
	byClass := make(map[RuntimeCallClass]int)
	for _, call := range g.active {
		byClass[call.class]++
	}
	return RuntimeAdmissionSnapshot{
		Identity:      g.identity,
		Draining:      g.draining,
		Forced:        g.forced,
		ForceCause:    g.forceCause,
		ActiveTotal:   len(g.active),
		ActiveByClass: byClass,
	}
}

func runtimeAdmissionForcedError(cause error) error {
	if cause == nil || errors.Is(cause, ErrRuntimeAdmissionForced) {
		return ErrRuntimeAdmissionForced
	}
	return errors.Join(ErrRuntimeAdmissionForced, cause)
}
