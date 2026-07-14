package jobs

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	ErrPluginScheduleInvalid      = errors.New("plugin schedule declaration is invalid")
	ErrPluginScheduleRuntimeStale = errors.New("plugin schedule runtime is stale")
	ErrPluginScheduleDraining     = errors.New("plugin schedule runtime is draining")
	ErrPluginScheduleNotDeclared  = errors.New("plugin schedule is not declared")
)

// PluginScheduleRuntimeIdentity 固定触发器归属的 exact runtime artifact。
type PluginScheduleRuntimeIdentity struct {
	ExtensionID      string
	ExtensionVersion string
	ArtifactDigest   string
	InstanceID       string
}

// PluginScheduleDeclaration 是已验证 manifest 发布给宿主调度器的不可变契约。
// 本 registry 只管理 admission；它不自行创建 cron goroutine 或 River constructor。
type PluginScheduleDeclaration struct {
	ScheduleID   string
	JobName      string
	JobContract  string
	Cron         string
	Timezone     string
	Contract     PluginJobContract
	TrustGrantID string
}

type PluginScheduleRuntime struct {
	Identity  PluginScheduleRuntimeIdentity
	Schedules []PluginScheduleDeclaration
}

type PluginScheduleSnapshot struct {
	Identity       PluginScheduleRuntimeIdentity
	Active         bool
	Draining       bool
	ActiveTriggers int
	Schedules      []PluginScheduleDeclaration
}

type pluginScheduleRuntimeState struct {
	identity  PluginScheduleRuntimeIdentity
	schedules map[string]PluginScheduleDeclaration
	draining  bool
	active    int
	idle      chan struct{}
}

// PluginScheduleAdmissionRegistry 让发布、触发 admission 与 drain 共用一把锁。
// 生命周期可以先 BeginDrain，再 WaitDrain，保证返回后不再出现新的触发写入。
type PluginScheduleAdmissionRegistry struct {
	mu       sync.Mutex
	active   map[string]PluginScheduleRuntimeIdentity
	runtimes map[PluginScheduleRuntimeIdentity]*pluginScheduleRuntimeState
	periodic *PluginSchedulePeriodicPublisher
}

type PluginScheduleTriggerLease struct {
	Context context.Context

	registry *PluginScheduleAdmissionRegistry
	identity PluginScheduleRuntimeIdentity
	cancel   context.CancelFunc
	once     sync.Once
}

func NewPluginScheduleAdmissionRegistry() *PluginScheduleAdmissionRegistry {
	return &PluginScheduleAdmissionRegistry{
		active:   make(map[string]PluginScheduleRuntimeIdentity),
		runtimes: make(map[PluginScheduleRuntimeIdentity]*pluginScheduleRuntimeState),
	}
}

// PublishActive 原子发布新实例，并在同一线性化边界关闭旧实例的新触发入口。
func (r *PluginScheduleAdmissionRegistry) PublishActive(runtime PluginScheduleRuntime) (PluginScheduleSnapshot, error) {
	identity, schedules, err := normalizePluginScheduleRuntime(runtime)
	if err != nil {
		return PluginScheduleSnapshot{}, err
	}
	if r == nil {
		return PluginScheduleSnapshot{}, ErrPluginScheduleInvalid
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureMapsLocked()
	currentIdentity, hasCurrent := r.active[identity.ExtensionID]
	var previousState *pluginScheduleRuntimeState
	var previousRuntime *PluginScheduleRuntime
	if hasCurrent && currentIdentity != identity {
		previousState = r.runtimes[currentIdentity]
		if previousState != nil {
			value := r.runtimeLocked(previousState)
			previousRuntime = &value
		}
	}
	if existing := r.runtimes[identity]; existing != nil {
		if !samePluginSchedules(existing.schedules, schedules) {
			return r.snapshotLocked(existing), fmt.Errorf("%w: exact runtime declarations changed", ErrPluginScheduleInvalid)
		}
		if r.periodic != nil && (!hasCurrent || currentIdentity != identity) {
			if err := r.periodic.Replace(previousRuntime, PluginScheduleRuntime{Identity: identity, Schedules: mapPluginSchedules(schedules)}); err != nil {
				return r.snapshotLocked(existing), err
			}
		}
		if previousState != nil {
			previousState.draining = true
		}
		// Failed activation compensation and exact rollback both republish the
		// retained immutable declaration set through this same lock boundary.
		existing.draining = false
		r.active[identity.ExtensionID] = identity
		return r.snapshotLocked(existing), nil
	}
	if r.periodic != nil {
		if err := r.periodic.Replace(previousRuntime, PluginScheduleRuntime{Identity: identity, Schedules: mapPluginSchedules(schedules)}); err != nil {
			return PluginScheduleSnapshot{}, err
		}
	}
	if previousState != nil {
		previousState.draining = true
	}

	idle := make(chan struct{})
	close(idle)
	state := &pluginScheduleRuntimeState{identity: identity, schedules: schedules, idle: idle}
	r.runtimes[identity] = state
	r.active[identity.ExtensionID] = identity
	return r.snapshotLocked(state), nil
}

// AcquireTrigger 同时校验活动 exact runtime 与 schedule declaration。
func (r *PluginScheduleAdmissionRegistry) AcquireTrigger(
	parent context.Context,
	identity PluginScheduleRuntimeIdentity,
	scheduleID string,
) (PluginScheduleDeclaration, *PluginScheduleTriggerLease, error) {
	identity = normalizePluginScheduleIdentity(identity)
	scheduleID = strings.TrimSpace(scheduleID)
	if r == nil || parent == nil || !identity.valid() || scheduleID == "" {
		return PluginScheduleDeclaration{}, nil, ErrPluginScheduleInvalid
	}
	if err := parent.Err(); err != nil {
		return PluginScheduleDeclaration{}, nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	state := r.runtimes[identity]
	if state == nil || r.active[identity.ExtensionID] != identity {
		return PluginScheduleDeclaration{}, nil, ErrPluginScheduleRuntimeStale
	}
	if state.draining {
		return PluginScheduleDeclaration{}, nil, ErrPluginScheduleDraining
	}
	declaration, ok := state.schedules[scheduleID]
	if !ok {
		return PluginScheduleDeclaration{}, nil, ErrPluginScheduleNotDeclared
	}
	if state.active == 0 {
		state.idle = make(chan struct{})
	}
	state.active++
	ctx, cancel := context.WithCancel(parent)
	return declaration, &PluginScheduleTriggerLease{
		Context: ctx, registry: r, identity: identity, cancel: cancel,
	}, nil
}

// BeginDrain 与 AcquireTrigger 线性化；返回后该 exact runtime 不再获得新 lease。
func (r *PluginScheduleAdmissionRegistry) BeginDrain(identity PluginScheduleRuntimeIdentity) (PluginScheduleSnapshot, error) {
	identity = normalizePluginScheduleIdentity(identity)
	if r == nil || !identity.valid() {
		return PluginScheduleSnapshot{}, ErrPluginScheduleInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	state := r.runtimes[identity]
	if state == nil {
		return PluginScheduleSnapshot{}, ErrPluginScheduleRuntimeStale
	}
	state.draining = true
	if r.periodic != nil {
		r.periodic.Remove(r.runtimeLocked(state))
	}
	return r.snapshotLocked(state), nil
}

// BindPeriodicPublisher publishes already-active snapshots when a worker's
// River client becomes available. Each worker process binds its own bundle so
// leader changes keep the same dynamic catalog.
func (r *PluginScheduleAdmissionRegistry) BindPeriodicPublisher(publisher *PluginSchedulePeriodicPublisher) error {
	if r == nil || publisher == nil {
		return ErrPluginScheduleInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.periodic != nil {
		return fmt.Errorf("%w: periodic publisher already bound", ErrPluginScheduleInvalid)
	}
	published := make([]PluginScheduleRuntime, 0, len(r.active))
	for _, identity := range r.active {
		state := r.runtimes[identity]
		if state == nil || state.draining {
			continue
		}
		runtime := r.runtimeLocked(state)
		if err := publisher.Replace(nil, runtime); err != nil {
			for _, rollback := range published {
				publisher.Remove(rollback)
			}
			return err
		}
		published = append(published, runtime)
	}
	r.periodic = publisher
	return nil
}

func (r *PluginScheduleAdmissionRegistry) Snapshot(identity PluginScheduleRuntimeIdentity) (PluginScheduleSnapshot, error) {
	identity = normalizePluginScheduleIdentity(identity)
	if r == nil || !identity.valid() {
		return PluginScheduleSnapshot{}, ErrPluginScheduleInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	state := r.runtimes[identity]
	if state == nil {
		return PluginScheduleSnapshot{}, ErrPluginScheduleRuntimeStale
	}
	return r.snapshotLocked(state), nil
}

func (r *PluginScheduleAdmissionRegistry) WaitDrain(ctx context.Context, identity PluginScheduleRuntimeIdentity) error {
	identity = normalizePluginScheduleIdentity(identity)
	if r == nil || ctx == nil || !identity.valid() {
		return ErrPluginScheduleInvalid
	}
	for {
		r.mu.Lock()
		state := r.runtimes[identity]
		if state == nil {
			r.mu.Unlock()
			return ErrPluginScheduleRuntimeStale
		}
		if state.active == 0 {
			r.mu.Unlock()
			return nil
		}
		idle := state.idle
		r.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-idle:
		}
	}
}

func (l *PluginScheduleTriggerLease) Release() {
	if l == nil {
		return
	}
	l.once.Do(func() {
		if l.cancel != nil {
			l.cancel()
		}
		if l.registry != nil {
			l.registry.releaseTrigger(l.identity)
		}
	})
}

func (r *PluginScheduleAdmissionRegistry) releaseTrigger(identity PluginScheduleRuntimeIdentity) {
	r.mu.Lock()
	state := r.runtimes[identity]
	if state != nil && state.active > 0 {
		state.active--
		if state.active == 0 {
			close(state.idle)
		}
	}
	r.mu.Unlock()
}

func (r *PluginScheduleAdmissionRegistry) snapshotLocked(state *pluginScheduleRuntimeState) PluginScheduleSnapshot {
	schedules := make([]PluginScheduleDeclaration, 0, len(state.schedules))
	for _, declaration := range state.schedules {
		schedules = append(schedules, declaration)
	}
	sort.Slice(schedules, func(i, j int) bool { return schedules[i].ScheduleID < schedules[j].ScheduleID })
	return PluginScheduleSnapshot{
		Identity: state.identity, Active: r.active[state.identity.ExtensionID] == state.identity,
		Draining: state.draining, ActiveTriggers: state.active, Schedules: schedules,
	}
}

func (r *PluginScheduleAdmissionRegistry) runtimeLocked(state *pluginScheduleRuntimeState) PluginScheduleRuntime {
	return PluginScheduleRuntime{Identity: state.identity, Schedules: mapPluginSchedules(state.schedules)}
}

func mapPluginSchedules(schedules map[string]PluginScheduleDeclaration) []PluginScheduleDeclaration {
	result := make([]PluginScheduleDeclaration, 0, len(schedules))
	for _, declaration := range schedules {
		result = append(result, declaration)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ScheduleID < result[j].ScheduleID })
	return result
}

func (r *PluginScheduleAdmissionRegistry) ensureMapsLocked() {
	if r.active == nil {
		r.active = make(map[string]PluginScheduleRuntimeIdentity)
	}
	if r.runtimes == nil {
		r.runtimes = make(map[PluginScheduleRuntimeIdentity]*pluginScheduleRuntimeState)
	}
}

func normalizePluginScheduleRuntime(runtime PluginScheduleRuntime) (PluginScheduleRuntimeIdentity, map[string]PluginScheduleDeclaration, error) {
	identity := normalizePluginScheduleIdentity(runtime.Identity)
	if !identity.valid() || len(runtime.Schedules) == 0 {
		return PluginScheduleRuntimeIdentity{}, nil, ErrPluginScheduleInvalid
	}
	schedules := make(map[string]PluginScheduleDeclaration, len(runtime.Schedules))
	for _, declaration := range runtime.Schedules {
		declaration.ScheduleID = strings.TrimSpace(declaration.ScheduleID)
		declaration.JobName = strings.TrimSpace(declaration.JobName)
		declaration.JobContract = strings.TrimSpace(declaration.JobContract)
		declaration.Cron = strings.TrimSpace(declaration.Cron)
		declaration.Timezone = strings.TrimSpace(declaration.Timezone)
		declaration.Contract = declaration.Contract.Normalized()
		declaration.TrustGrantID = strings.TrimSpace(declaration.TrustGrantID)
		if declaration.ScheduleID == "" || declaration.JobName == "" || declaration.JobContract == "" || declaration.Cron == "" || declaration.Timezone == "" {
			return PluginScheduleRuntimeIdentity{}, nil, fmt.Errorf("%w: complete schedule contract is required", ErrPluginScheduleInvalid)
		}
		if _, exists := schedules[declaration.ScheduleID]; exists {
			return PluginScheduleRuntimeIdentity{}, nil, fmt.Errorf("%w: duplicate schedule %q", ErrPluginScheduleInvalid, declaration.ScheduleID)
		}
		schedules[declaration.ScheduleID] = declaration
	}
	return identity, schedules, nil
}

func normalizePluginScheduleIdentity(identity PluginScheduleRuntimeIdentity) PluginScheduleRuntimeIdentity {
	identity.ExtensionID = strings.TrimSpace(identity.ExtensionID)
	identity.ExtensionVersion = strings.TrimSpace(identity.ExtensionVersion)
	identity.ArtifactDigest = strings.TrimSpace(identity.ArtifactDigest)
	identity.InstanceID = strings.TrimSpace(identity.InstanceID)
	return identity
}

func (i PluginScheduleRuntimeIdentity) valid() bool {
	return i.ExtensionID != "" && i.ExtensionVersion != "" && i.ArtifactDigest != "" && i.InstanceID != ""
}

func (i PluginScheduleRuntimeIdentity) Valid() bool {
	return normalizePluginScheduleIdentity(i).valid()
}

func samePluginSchedules(left, right map[string]PluginScheduleDeclaration) bool {
	if len(left) != len(right) {
		return false
	}
	for id, declaration := range left {
		if right[id] != declaration {
			return false
		}
	}
	return true
}
