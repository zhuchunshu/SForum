package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

// pluginRuntimeFullSetStarter reuses managerStagedStarter and adds deterministic
// health, publish, stop and concurrency fault injection for full-set tests.
type pluginRuntimeFullSetStarter struct {
	inner        *managerStagedStarter
	mu           sync.Mutex
	healthErrors map[RuntimeInstanceIdentity]error
	// healthErrorsByExtensionID 按 extension ID 注入一次性 health 故障，
	// 避免全局 failNextHealth 误伤已 prestart/reuse 的 Protocol V1。
	healthErrorsByExtensionID map[string]error
	failNextHealth            error
	failPublishAfter          int
	publishCount              int
	stopErrors                map[RuntimeInstanceIdentity]error
	startErrors               map[string]error
	acquireSetError           error
	revalidateSetError        error
	revalidateSetHook         func()
	finalFenceError           error
	finalFenceHook            func()
	startDelay                time.Duration
	inFlightStarts            atomic.Int32
	maxInFlightStarts         atomic.Int32
	startedExtensions         map[RuntimeInstanceIdentity]extensions.Extension
	legacyStartCount          atomic.Int32
	legacyStopCount           atomic.Int32
	// stagedStartCount 统计 Protocol V2 StartInstance 调用次数（用于证明真正进入 V2 stage）。
	stagedStartCount atomic.Int32
}

func newPluginRuntimeFullSetStarter() *pluginRuntimeFullSetStarter {
	return &pluginRuntimeFullSetStarter{
		inner:                     newManagerStagedStarter(),
		healthErrors:              make(map[RuntimeInstanceIdentity]error),
		healthErrorsByExtensionID: make(map[string]error),
		stopErrors:                make(map[RuntimeInstanceIdentity]error),
		startErrors:               make(map[string]error),
		startedExtensions:         make(map[RuntimeInstanceIdentity]extensions.Extension),
	}
}

func (s *pluginRuntimeFullSetStarter) Start(ctx context.Context, extension extensions.Extension) (RouteTarget, error) {
	s.mu.Lock()
	if err := s.startErrors[extension.ID]; err != nil {
		s.mu.Unlock()
		return RouteTarget{}, err
	}
	s.mu.Unlock()
	target, err := s.inner.Start(ctx, extension)
	if err != nil {
		return target, err
	}
	if manifestProtocolVersion(extension) != 1 {
		return target, nil
	}
	s.legacyStartCount.Add(1)
	identity := RuntimeInstanceIdentity{ExtensionID: extension.ID, InstanceID: target.InstanceID}
	s.inner.mu.Lock()
	snapshot := s.inner.instances[identity]
	snapshot.ProtocolVersion = 1
	snapshot.ReadinessChecked = false
	s.inner.instances[identity] = snapshot
	s.inner.mu.Unlock()
	s.mu.Lock()
	s.startedExtensions[identity] = extension
	s.mu.Unlock()
	return target, nil
}

func (s *pluginRuntimeFullSetStarter) Stop(ctx context.Context, extension extensions.Extension) error {
	if manifestProtocolVersion(extension) == 1 {
		s.legacyStopCount.Add(1)
	}
	return s.inner.Stop(ctx, extension)
}

func (s *pluginRuntimeFullSetStarter) failStart(extensionID string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startErrors[extensionID] = err
}

// failHealthByExtensionID 按 exact extension ID 注入一次性 HealthInstance 故障。
func (s *pluginRuntimeFullSetStarter) failHealthByExtensionID(extensionID string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthErrorsByExtensionID[extensionID] = err
}

func (s *pluginRuntimeFullSetStarter) StartInstance(ctx context.Context, extension extensions.Extension) (RouteTarget, error) {
	cur := s.inFlightStarts.Add(1)
	for {
		prev := s.maxInFlightStarts.Load()
		if cur <= prev || s.maxInFlightStarts.CompareAndSwap(prev, cur) {
			break
		}
	}
	defer s.inFlightStarts.Add(-1)
	if s.startDelay > 0 {
		select {
		case <-ctx.Done():
			return RouteTarget{}, ctx.Err()
		case <-time.After(s.startDelay):
		}
	}
	target, err := s.inner.StartInstance(ctx, extension)
	if err != nil {
		return RouteTarget{}, err
	}
	s.stagedStartCount.Add(1)
	s.mu.Lock()
	s.startedExtensions[RuntimeInstanceIdentity{ExtensionID: extension.ID, InstanceID: target.InstanceID}] = extension
	s.mu.Unlock()
	return target, nil
}

func (s *pluginRuntimeFullSetStarter) InspectInstance(identity RuntimeInstanceIdentity) (ProtocolRuntimeInstanceSnapshot, error) {
	return s.inner.InspectInstance(identity)
}

func (s *pluginRuntimeFullSetStarter) HealthInstance(ctx context.Context, identity RuntimeInstanceIdentity) (PluginHealth, error) {
	snapshot, _ := s.inner.InspectInstance(identity)
	s.mu.Lock()
	if snapshot.State == ProtocolRuntimePublished && s.revalidateSetError != nil {
		err := s.revalidateSetError
		hook := s.revalidateSetHook
		s.revalidateSetError = nil
		s.revalidateSetHook = nil
		s.mu.Unlock()
		if hook != nil {
			hook()
		}
		return PluginHealth{}, err
	}
	if err := s.failNextHealth; err != nil {
		s.failNextHealth = nil
		s.mu.Unlock()
		return PluginHealth{}, err
	}
	if err := s.healthErrorsByExtensionID[identity.ExtensionID]; err != nil {
		delete(s.healthErrorsByExtensionID, identity.ExtensionID)
		s.mu.Unlock()
		return PluginHealth{}, err
	}
	if err := s.healthErrors[identity]; err != nil {
		s.mu.Unlock()
		return PluginHealth{}, err
	}
	s.mu.Unlock()
	return s.inner.HealthInstance(ctx, identity)
}

func (s *pluginRuntimeFullSetStarter) PublishInstance(ctx context.Context, identity RuntimeInstanceIdentity) (ProtocolRuntimeInstanceSnapshot, error) {
	s.mu.Lock()
	s.publishCount++
	count := s.publishCount
	if s.failPublishAfter > 0 && count == s.failPublishAfter+1 {
		s.failPublishAfter = 0
		s.mu.Unlock()
		return ProtocolRuntimeInstanceSnapshot{}, fmt.Errorf("forced publish failure on call %d", count)
	}
	s.mu.Unlock()
	return s.inner.PublishInstance(ctx, identity)
}

func (s *pluginRuntimeFullSetStarter) PublishInstanceSet(
	ctx context.Context,
	identities []RuntimeInstanceIdentity,
) ([]ProtocolRuntimeInstanceSnapshot, *ProtocolRuntimeSetLease, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	desired, err := normalizeProtocolRuntimeInstanceSet(identities)
	if err != nil {
		return nil, nil, err
	}

	// Run every deterministic fault/barrier before mutating the fake's active
	// map, matching the production complete-set failure guarantee.
	for _, identity := range desired {
		if s.inner.activeIdentity(identity.ExtensionID) == identity {
			continue
		}
		if s.inner.publishStarted != nil {
			s.inner.publishStarted <- struct{}{}
			<-s.inner.publishContinue
		}
		s.mu.Lock()
		s.publishCount++
		count := s.publishCount
		if s.failPublishAfter > 0 && count == s.failPublishAfter+1 {
			s.failPublishAfter = 0
			s.mu.Unlock()
			return nil, nil, fmt.Errorf("forced publish failure on call %d", count)
		}
		s.mu.Unlock()
	}

	s.inner.mu.Lock()
	defer s.inner.mu.Unlock()
	for _, identity := range desired {
		if err := s.inner.publishErrors[identity]; err != nil {
			return nil, nil, err
		}
		snapshot, ok := s.inner.instances[identity]
		if !ok {
			return nil, nil, ErrRuntimeInstanceNotFound
		}
		if !snapshot.Ready {
			return nil, nil, ErrProtocolInstanceNotReady
		}
	}
	desiredSet := make(map[string]RuntimeInstanceIdentity, len(desired))
	for _, identity := range desired {
		desiredSet[identity.ExtensionID] = identity
	}
	for extensionID, previous := range s.inner.active {
		if desiredSet[extensionID] == previous {
			continue
		}
		snapshot := s.inner.instances[previous]
		snapshot.State = ProtocolRuntimeRetained
		s.inner.instances[previous] = snapshot
	}
	nextActive := make(map[string]RuntimeInstanceIdentity, len(desired))
	result := make([]ProtocolRuntimeInstanceSnapshot, 0, len(desired))
	for _, identity := range desired {
		if previous := s.inner.active[identity.ExtensionID]; previous.InstanceID != "" && previous != identity {
			snapshot := s.inner.instances[previous]
			snapshot.State = ProtocolRuntimeRetained
			s.inner.instances[previous] = snapshot
		}
		snapshot := s.inner.instances[identity]
		snapshot.State = ProtocolRuntimePublished
		s.inner.instances[identity] = snapshot
		nextActive[identity.ExtensionID] = identity
		result = append(result, snapshot)
	}
	s.inner.active = nextActive
	s.mu.Lock()
	s.revalidateSetError = s.acquireSetError
	s.acquireSetError = nil
	s.mu.Unlock()
	lease := &ProtocolRuntimeSetLease{release: func() {}}
	lease.restore = func(restoreCtx context.Context, restoreIdentities []RuntimeInstanceIdentity) ([]ProtocolRuntimeInstanceSnapshot, error) {
		restored, nested, restoreErr := s.PublishInstanceSet(restoreCtx, restoreIdentities)
		if nested != nil {
			nested.Release()
		}
		return restored, restoreErr
	}
	lease.validate = func(validateCtx context.Context, validateIdentities []RuntimeInstanceIdentity) error {
		s.mu.Lock()
		err := s.finalFenceError
		hook := s.finalFenceHook
		s.finalFenceError = nil
		s.finalFenceHook = nil
		s.mu.Unlock()
		if hook != nil {
			hook()
		}
		if err != nil {
			return err
		}
		if !s.RuntimeInstanceSetVisible(validateCtx, validateIdentities) {
			return ErrProtocolInstanceNotReady
		}
		return nil
	}
	return result, lease, nil
}

func (s *pluginRuntimeFullSetStarter) RuntimeInstanceSetVisible(ctx context.Context, identities []RuntimeInstanceIdentity) bool {
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	desired, err := normalizeProtocolRuntimeInstanceSet(identities)
	if err != nil {
		return false
	}
	s.inner.mu.Lock()
	defer s.inner.mu.Unlock()
	if len(s.inner.active) != len(desired) {
		return false
	}
	for _, identity := range desired {
		if s.inner.active[identity.ExtensionID] != identity {
			return false
		}
		snapshot, ok := s.inner.instances[identity]
		if !ok || snapshot.State != ProtocolRuntimePublished {
			return false
		}
	}
	return true
}

func (s *pluginRuntimeFullSetStarter) RunLifecycleInstance(
	ctx context.Context,
	identity RuntimeInstanceIdentity,
	extension extensions.Extension,
	invocation LifecycleInvocation,
) (LifecycleRunResult, error) {
	return s.inner.RunLifecycleInstance(ctx, identity, extension, invocation)
}

func (s *pluginRuntimeFullSetStarter) StopInstance(ctx context.Context, identity RuntimeInstanceIdentity) error {
	s.mu.Lock()
	err := s.stopErrors[identity]
	s.mu.Unlock()
	if err != nil {
		return err
	}
	return s.inner.StopInstance(ctx, identity)
}

func (s *pluginRuntimeFullSetStarter) DiscardInstance(ctx context.Context, identity RuntimeInstanceIdentity) error {
	return s.inner.DiscardInstance(ctx, identity)
}

func (s *pluginRuntimeFullSetStarter) failStop(identity RuntimeInstanceIdentity, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopErrors[identity] = err
}

func (s *pluginRuntimeFullSetStarter) resetPublishFault(failAfter int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.publishCount = 0
	s.failPublishAfter = failAfter
}

func (s *pluginRuntimeFullSetStarter) startCount() int {
	s.inner.mu.Lock()
	defer s.inner.mu.Unlock()
	return s.inner.next
}

func (s *pluginRuntimeFullSetStarter) startedExtension(identity RuntimeInstanceIdentity) (extensions.Extension, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	extension, ok := s.startedExtensions[identity]
	return extension, ok
}

func TestPluginRuntimeFullSetKeepsProtocolV1AndV2LTSMembersTogether(t *testing.T) {
	starter := newPluginRuntimeFullSetStarter()
	manager := NewManager(ManagerConfig{Starter: starter})
	inventory := newPluginRuntimeFullSetTestInventory()
	legacy := inventory.seed(t, "lts.protocol.v1", 1, "1.0.0", "lts-protocol-v1")
	current := inventory.seed(t, "lts.protocol.v2", 2, "2.0.0", "lts-protocol-v2")
	markPluginRuntimeFullSetProtocolV1(t, inventory, legacy.ID, "1.0.0", "lts-protocol-v1")

	// 普通路径：已存在 exact V1 时与 V2 同集收敛。
	if err := manager.Start(context.Background(), pluginRuntimeExactExtension(
		inventory.extensions[legacy.ID],
		inventory.versions[inventory.versionKey(legacy.ID, "1.0.0", pluginRuntimeFullSetDigest("lts-protocol-v1"))],
		inventory.member(legacy.ID, 1, "1.0.0", "lts-protocol-v1"),
	)); err != nil {
		t.Fatalf("start Protocol V1 compatibility runtime: %v", err)
	}
	legacyBefore, err := manager.ActiveRuntimeInstance(legacy.ID)
	if err != nil {
		t.Fatal(err)
	}

	applier := mustNewPluginRuntimeFullSetApplier(t, manager, inventory)
	applied, err := applier.ApplyPluginRuntimeFullSet(context.Background(), inventory.publication(
		extensions.PluginRuntimePublicationEnable,
		inventory.member(legacy.ID, 1, "1.0.0", "lts-protocol-v1"),
		inventory.member(current.ID, 2, "2.0.0", "lts-protocol-v2"),
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 2 {
		t.Fatalf("applied LTS set = %#v", applied)
	}
	legacyActive, err := manager.ActiveRuntimeInstance(legacy.ID)
	if err != nil || legacyActive.ExtensionVersion != "1.0.0" || legacyActive.Identity != legacyBefore.Identity {
		t.Fatalf("Protocol V1 active = %#v, %v", legacyActive, err)
	}
	currentActive, err := manager.ActiveRuntimeInstance(current.ID)
	if err != nil || currentActive.ExtensionVersion != "2.0.0" {
		t.Fatalf("Protocol V2 active = %#v, %v", currentActive, err)
	}
	if !starter.RuntimeInstanceSetVisible(context.Background(), []RuntimeInstanceIdentity{legacyActive.Identity, currentActive.Identity}) {
		t.Fatal("Protocol V1 and V2 members were not visible in one complete LTS set")
	}
}

func TestPluginRuntimeFullSetRejectsMissingProtocolV1WithoutPartialActivation(t *testing.T) {
	starter := newPluginRuntimeFullSetStarter()
	manager := NewManager(ManagerConfig{Starter: starter})
	inventory := newPluginRuntimeFullSetTestInventory()
	legacy := inventory.seed(t, "lts.missing.v1", 1, "1.0.0", "lts-missing-v1")
	markPluginRuntimeFullSetProtocolV1(t, inventory, legacy.ID, "1.0.0", "lts-missing-v1")

	applier := mustNewPluginRuntimeFullSetApplier(t, manager, inventory)
	_, err := applier.ApplyPluginRuntimeFullSet(context.Background(), inventory.publication(
		extensions.PluginRuntimePublicationEnable,
		inventory.member(legacy.ID, 1, "1.0.0", "lts-missing-v1"),
	))
	if !errors.Is(err, ErrProtocolInstanceTransitionBlocked) {
		t.Fatalf("missing Protocol V1 error = %v", err)
	}
	if _, activeErr := manager.ActiveRuntimeInstance(legacy.ID); !errors.Is(activeErr, ErrRuntimeInstanceNotFound) {
		t.Fatalf("failed full-set activated Protocol V1: %v", activeErr)
	}
	if active := starter.inner.activeIdentity(legacy.ID); active != (RuntimeInstanceIdentity{}) {
		t.Fatalf("failed full-set started a Protocol V1 process: %#v", active)
	}
	if starter.legacyStartCount.Load() != 0 {
		t.Fatalf("ordinary applier started Protocol V1 processes: %d", starter.legacyStartCount.Load())
	}
}

func TestInitialBootstrapProtocolV1AppliesMixedExactSet(t *testing.T) {
	starter := newPluginRuntimeFullSetStarter()
	manager := NewManager(ManagerConfig{Starter: starter})
	inventory := newPluginRuntimeFullSetTestInventory()
	legacyA := inventory.seed(t, "boot.v1.a", 1, "1.0.0", "boot-v1-a")
	legacyB := inventory.seed(t, "boot.v1.b", 2, "1.0.0", "boot-v1-b")
	current := inventory.seed(t, "boot.v2", 3, "2.0.0", "boot-v2")
	disabled := inventory.seed(t, "boot.v1.disabled", 4, "1.0.0", "boot-v1-disabled")
	markPluginRuntimeFullSetProtocolV1(t, inventory, legacyA.ID, "1.0.0", "boot-v1-a")
	markPluginRuntimeFullSetProtocolV1(t, inventory, legacyB.ID, "1.0.0", "boot-v1-b")
	markPluginRuntimeFullSetProtocolV1(t, inventory, disabled.ID, "1.0.0", "boot-v1-disabled")

	publication := inventory.publication(
		extensions.PluginRuntimePublicationStartupReconcile,
		inventory.member(legacyB.ID, 2, "1.0.0", "boot-v1-b"),
		inventory.member(current.ID, 3, "2.0.0", "boot-v2"),
		inventory.member(legacyA.ID, 1, "1.0.0", "boot-v1-a"),
	)
	applier := mustNewInitialBootstrapPluginRuntimeFullSetApplier(t, manager, inventory)
	applied, err := applier.ApplyPluginRuntimeFullSet(context.Background(), publication)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 3 {
		t.Fatalf("applied mixed set = %#v", applied)
	}
	for _, id := range []string{legacyA.ID, legacyB.ID, current.ID} {
		if _, activeErr := manager.ActiveRuntimeInstance(id); activeErr != nil {
			t.Fatalf("member %s not active: %v", id, activeErr)
		}
	}
	if _, err := manager.ActiveRuntimeInstance(disabled.ID); !errors.Is(err, ErrRuntimeInstanceNotFound) {
		t.Fatalf("non-member Protocol V1 started: %v", err)
	}
	if starter.legacyStartCount.Load() != 2 {
		t.Fatalf("Protocol V1 starts = %d want 2", starter.legacyStartCount.Load())
	}
	// 成功 Apply 后窗口必须单调关闭：新增缺失 V1 不得再 cold-start。
	later := inventory.seed(t, "boot.v1.after-success", 5, "1.0.0", "boot-v1-after-success")
	markPluginRuntimeFullSetProtocolV1(t, inventory, later.ID, "1.0.0", "boot-v1-after-success")
	_, err = applier.ApplyPluginRuntimeFullSet(context.Background(), inventory.publication(
		extensions.PluginRuntimePublicationEnable,
		inventory.member(legacyA.ID, 1, "1.0.0", "boot-v1-a"),
		inventory.member(legacyB.ID, 2, "1.0.0", "boot-v1-b"),
		inventory.member(current.ID, 3, "2.0.0", "boot-v2"),
		inventory.member(later.ID, 5, "1.0.0", "boot-v1-after-success"),
	))
	if !errors.Is(err, ErrProtocolInstanceTransitionBlocked) {
		t.Fatalf("post-success missing Protocol V1 error = %v", err)
	}
	if _, activeErr := manager.ActiveRuntimeInstance(later.ID); !errors.Is(activeErr, ErrRuntimeInstanceNotFound) {
		t.Fatalf("post-success cold-started Protocol V1: %v", activeErr)
	}
}

// TestInitialBootstrapProtocolV1StartNFailureRollsBackPrior 证明成员 N 启动失败时
// 已启动的 1..N-1 由外层 defer 逆序回滚，且不留下 applied evidence。
func TestInitialBootstrapProtocolV1StartNFailureRollsBackPrior(t *testing.T) {
	starter := newPluginRuntimeFullSetStarter()
	manager := NewManager(ManagerConfig{Starter: starter})
	inventory := newPluginRuntimeFullSetTestInventory()
	// id 升序：a、b 先成功 cold-start，c 失败后必须回滚 a/b。
	first := inventory.seed(t, "boot.rollback.a", 1, "1.0.0", "boot-rollback-a")
	second := inventory.seed(t, "boot.rollback.b", 2, "1.0.0", "boot-rollback-b")
	third := inventory.seed(t, "boot.rollback.c", 3, "1.0.0", "boot-rollback-c")
	markPluginRuntimeFullSetProtocolV1(t, inventory, first.ID, "1.0.0", "boot-rollback-a")
	markPluginRuntimeFullSetProtocolV1(t, inventory, second.ID, "1.0.0", "boot-rollback-b")
	markPluginRuntimeFullSetProtocolV1(t, inventory, third.ID, "1.0.0", "boot-rollback-c")
	starter.failStart(third.ID, errors.New("forced third V1 start failure"))

	applier := mustNewInitialBootstrapPluginRuntimeFullSetApplier(t, manager, inventory)
	applied, err := applier.ApplyPluginRuntimeFullSet(context.Background(), inventory.publication(
		extensions.PluginRuntimePublicationStartupReconcile,
		inventory.member(first.ID, 1, "1.0.0", "boot-rollback-a"),
		inventory.member(second.ID, 2, "1.0.0", "boot-rollback-b"),
		inventory.member(third.ID, 3, "1.0.0", "boot-rollback-c"),
	))
	if err == nil {
		t.Fatal("expected third Protocol V1 start failure")
	}
	if applied != nil {
		t.Fatalf("failed start-N apply must not return applied evidence: %#v", applied)
	}
	for _, id := range []string{first.ID, second.ID, third.ID} {
		if _, activeErr := manager.ActiveRuntimeInstance(id); !errors.Is(activeErr, ErrRuntimeInstanceNotFound) {
			t.Fatalf("Protocol V1 %s not rolled back: %v", id, activeErr)
		}
	}
	if starter.legacyStartCount.Load() != 2 {
		t.Fatalf("Protocol V1 starts before failure = %d want 2", starter.legacyStartCount.Load())
	}
	if starter.legacyStopCount.Load() != 2 {
		t.Fatalf("expected reverse-order rollback of 1..N-1, stops=%d", starter.legacyStopCount.Load())
	}

	// 失败不得关闭窗口：清除故障后重试应能 cold-start 全部 V1。
	starter.mu.Lock()
	delete(starter.startErrors, third.ID)
	starter.mu.Unlock()
	retryApplied, retryErr := applier.ApplyPluginRuntimeFullSet(context.Background(), inventory.publication(
		extensions.PluginRuntimePublicationStartupReconcile,
		inventory.member(first.ID, 1, "1.0.0", "boot-rollback-a"),
		inventory.member(second.ID, 2, "1.0.0", "boot-rollback-b"),
		inventory.member(third.ID, 3, "1.0.0", "boot-rollback-c"),
	))
	if retryErr != nil {
		t.Fatalf("retry after start-N failure: %v", retryErr)
	}
	if len(retryApplied) != 3 {
		t.Fatalf("retry applied = %#v", retryApplied)
	}
}

func TestInitialBootstrapProtocolV1RollsBackOnLaterV2Failure(t *testing.T) {
	starter := newPluginRuntimeFullSetStarter()
	manager := NewManager(ManagerConfig{Starter: starter})
	inventory := newPluginRuntimeFullSetTestInventory()
	legacy := inventory.seed(t, "boot.later.v1", 1, "1.0.0", "boot-later-v1")
	current := inventory.seed(t, "boot.later.v2", 2, "2.0.0", "boot-later-v2")
	markPluginRuntimeFullSetProtocolV1(t, inventory, legacy.ID, "1.0.0", "boot-later-v1")
	// 按 V2 extension ID 注入 health 故障，确保 V1 reuse health 先通过。
	starter.failHealthByExtensionID(current.ID, errors.New("forced V2 health failure"))

	applier := mustNewInitialBootstrapPluginRuntimeFullSetApplier(t, manager, inventory)
	applied, err := applier.ApplyPluginRuntimeFullSet(context.Background(), inventory.publication(
		extensions.PluginRuntimePublicationEnable,
		inventory.member(legacy.ID, 1, "1.0.0", "boot-later-v1"),
		inventory.member(current.ID, 2, "2.0.0", "boot-later-v2"),
	))
	if err == nil {
		t.Fatal("expected later Protocol V2 health failure")
	}
	if applied != nil {
		t.Fatalf("failed V2 stage must not return applied evidence: %#v", applied)
	}
	if !strings.Contains(err.Error(), "forced V2 health failure") {
		t.Fatalf("error should carry V2 health cause: %v", err)
	}
	// 必须真正 stage 过 V2 候选，而不是在 V1 recheck 上假失败。
	if starter.stagedStartCount.Load() < 1 {
		t.Fatalf("expected Protocol V2 candidate stage before health failure, staged=%d", starter.stagedStartCount.Load())
	}
	if starter.legacyStartCount.Load() != 1 {
		t.Fatalf("Protocol V1 starts = %d want 1", starter.legacyStartCount.Load())
	}
	if _, activeErr := manager.ActiveRuntimeInstance(legacy.ID); !errors.Is(activeErr, ErrRuntimeInstanceNotFound) {
		t.Fatalf("newly started Protocol V1 not rolled back: %v", activeErr)
	}
	if _, activeErr := manager.ActiveRuntimeInstance(current.ID); !errors.Is(activeErr, ErrRuntimeInstanceNotFound) {
		t.Fatalf("failed apply left Protocol V2 active: %v", activeErr)
	}
	if starter.legacyStopCount.Load() < 1 {
		t.Fatalf("expected Protocol V1 rollback stop, stops=%d", starter.legacyStopCount.Load())
	}

	// 失败后窗口仍 armed：重试应能重新 cold-start V1。
	retryApplied, retryErr := applier.ApplyPluginRuntimeFullSet(context.Background(), inventory.publication(
		extensions.PluginRuntimePublicationEnable,
		inventory.member(legacy.ID, 1, "1.0.0", "boot-later-v1"),
		inventory.member(current.ID, 2, "2.0.0", "boot-later-v2"),
	))
	if retryErr != nil {
		t.Fatalf("retry after V2 failure: %v", retryErr)
	}
	if len(retryApplied) != 2 {
		t.Fatalf("retry applied = %#v", retryApplied)
	}
}

func TestInitialBootstrapProtocolV1ReusesExistingAndKeepsItOnFailure(t *testing.T) {
	starter := newPluginRuntimeFullSetStarter()
	manager := NewManager(ManagerConfig{Starter: starter})
	inventory := newPluginRuntimeFullSetTestInventory()
	legacy := inventory.seed(t, "boot.reuse.v1", 1, "1.0.0", "boot-reuse-v1")
	current := inventory.seed(t, "boot.reuse.v2", 2, "2.0.0", "boot-reuse-v2")
	markPluginRuntimeFullSetProtocolV1(t, inventory, legacy.ID, "1.0.0", "boot-reuse-v1")

	exact := pluginRuntimeExactExtension(
		inventory.extensions[legacy.ID],
		inventory.versions[inventory.versionKey(legacy.ID, "1.0.0", pluginRuntimeFullSetDigest("boot-reuse-v1"))],
		inventory.member(legacy.ID, 1, "1.0.0", "boot-reuse-v1"),
	)
	if err := manager.Start(context.Background(), exact); err != nil {
		t.Fatal(err)
	}
	before, err := manager.ActiveRuntimeInstance(legacy.ID)
	if err != nil {
		t.Fatal(err)
	}
	startsBefore := starter.legacyStartCount.Load()
	stopsBefore := starter.legacyStopCount.Load()
	// 仅让 V2 健康检查失败；已存在 V1 必须通过 reuse health。
	starter.failHealthByExtensionID(current.ID, errors.New("forced V2 health failure after reuse"))

	applier := mustNewInitialBootstrapPluginRuntimeFullSetApplier(t, manager, inventory)
	_, err = applier.ApplyPluginRuntimeFullSet(context.Background(), inventory.publication(
		extensions.PluginRuntimePublicationEnable,
		inventory.member(legacy.ID, 1, "1.0.0", "boot-reuse-v1"),
		inventory.member(current.ID, 2, "2.0.0", "boot-reuse-v2"),
	))
	if err == nil {
		t.Fatal("expected Protocol V2 failure")
	}
	if !strings.Contains(err.Error(), "forced V2 health failure after reuse") {
		t.Fatalf("error should carry V2 health cause: %v", err)
	}
	if starter.stagedStartCount.Load() < 1 {
		t.Fatalf("expected Protocol V2 stage after V1 reuse, staged=%d", starter.stagedStartCount.Load())
	}
	after, err := manager.ActiveRuntimeInstance(legacy.ID)
	if err != nil || after.Identity != before.Identity {
		t.Fatalf("reused Protocol V1 changed: before=%#v after=%#v err=%v", before, after, err)
	}
	if starter.legacyStartCount.Load() != startsBefore {
		t.Fatalf("reused Protocol V1 was restarted: starts before=%d after=%d", startsBefore, starter.legacyStartCount.Load())
	}
	if starter.legacyStopCount.Load() != stopsBefore {
		t.Fatalf("reused Protocol V1 was stopped on failure: stops before=%d after=%d", stopsBefore, starter.legacyStopCount.Load())
	}
}

func TestInitialBootstrapProtocolV1MismatchedActiveFailsWithoutReplacement(t *testing.T) {
	starter := newPluginRuntimeFullSetStarter()
	manager := NewManager(ManagerConfig{Starter: starter})
	inventory := newPluginRuntimeFullSetTestInventory()
	legacy := inventory.seed(t, "boot.mismatch.v1", 1, "1.0.0", "boot-mismatch-v1")
	inventory.seed(t, legacy.ID, 2, "1.0.1", "boot-mismatch-v1-other")
	markPluginRuntimeFullSetProtocolV1(t, inventory, legacy.ID, "1.0.0", "boot-mismatch-v1")
	markPluginRuntimeFullSetProtocolV1(t, inventory, legacy.ID, "1.0.1", "boot-mismatch-v1-other")

	running := pluginRuntimeExactExtension(
		inventory.extensions[legacy.ID],
		inventory.versions[inventory.versionKey(legacy.ID, "1.0.0", pluginRuntimeFullSetDigest("boot-mismatch-v1"))],
		inventory.member(legacy.ID, 1, "1.0.0", "boot-mismatch-v1"),
	)
	if err := manager.Start(context.Background(), running); err != nil {
		t.Fatal(err)
	}
	before, err := manager.ActiveRuntimeInstance(legacy.ID)
	if err != nil {
		t.Fatal(err)
	}

	applier := mustNewInitialBootstrapPluginRuntimeFullSetApplier(t, manager, inventory)
	_, err = applier.ApplyPluginRuntimeFullSet(context.Background(), inventory.publication(
		extensions.PluginRuntimePublicationEnable,
		inventory.member(legacy.ID, 2, "1.0.1", "boot-mismatch-v1-other"),
	))
	if !errors.Is(err, ErrProtocolInstanceTransitionBlocked) {
		t.Fatalf("mismatched Protocol V1 error = %v", err)
	}
	after, err := manager.ActiveRuntimeInstance(legacy.ID)
	if err != nil || after.Identity != before.Identity || after.ExtensionVersion != "1.0.0" {
		t.Fatalf("mismatched active was replaced: before=%#v after=%#v err=%v", before, after, err)
	}
}

func TestInitialBootstrapProtocolV1RetryThenDisarm(t *testing.T) {
	starter := newPluginRuntimeFullSetStarter()
	manager := NewManager(ManagerConfig{Starter: starter})
	inventory := newPluginRuntimeFullSetTestInventory()
	legacy := inventory.seed(t, "boot.retry.v1", 1, "1.0.0", "boot-retry-v1")
	current := inventory.seed(t, "boot.retry.v2", 2, "2.0.0", "boot-retry-v2")
	later := inventory.seed(t, "boot.retry.later.v1", 3, "1.0.0", "boot-retry-later-v1")
	markPluginRuntimeFullSetProtocolV1(t, inventory, legacy.ID, "1.0.0", "boot-retry-v1")
	markPluginRuntimeFullSetProtocolV1(t, inventory, later.ID, "1.0.0", "boot-retry-later-v1")

	publication := inventory.publication(
		extensions.PluginRuntimePublicationEnable,
		inventory.member(legacy.ID, 1, "1.0.0", "boot-retry-v1"),
		inventory.member(current.ID, 2, "2.0.0", "boot-retry-v2"),
	)
	applier := mustNewInitialBootstrapPluginRuntimeFullSetApplier(t, manager, inventory)

	// 按 V2 ID 注入故障，证明 V1 cold-start + reuse health 已通过后才失败。
	starter.failHealthByExtensionID(current.ID, errors.New("first convergence health failure"))
	if _, err := applier.ApplyPluginRuntimeFullSet(context.Background(), publication); err == nil {
		t.Fatal("expected first apply failure")
	}
	if starter.stagedStartCount.Load() < 1 {
		t.Fatalf("first failure must reach Protocol V2 stage, staged=%d", starter.stagedStartCount.Load())
	}
	if _, err := manager.ActiveRuntimeInstance(legacy.ID); !errors.Is(err, ErrRuntimeInstanceNotFound) {
		t.Fatalf("failed first apply left Protocol V1 active: %v", err)
	}

	// 窗口仍 armed：重试可重新 cold-start 缺失 V1 并收敛。
	applied, err := applier.ApplyPluginRuntimeFullSet(context.Background(), publication)
	if err != nil {
		t.Fatalf("retry apply: %v", err)
	}
	if len(applied) != 2 {
		t.Fatalf("retry applied = %#v", applied)
	}

	// 成功后新的/缺失 V1 不得再 cold-start（窗口已 disarm）。
	_, err = applier.ApplyPluginRuntimeFullSet(context.Background(), inventory.publication(
		extensions.PluginRuntimePublicationEnable,
		inventory.member(legacy.ID, 1, "1.0.0", "boot-retry-v1"),
		inventory.member(current.ID, 2, "2.0.0", "boot-retry-v2"),
		inventory.member(later.ID, 3, "1.0.0", "boot-retry-later-v1"),
	))
	if !errors.Is(err, ErrProtocolInstanceTransitionBlocked) {
		t.Fatalf("post-success missing Protocol V1 error = %v", err)
	}
	if _, activeErr := manager.ActiveRuntimeInstance(later.ID); !errors.Is(activeErr, ErrRuntimeInstanceNotFound) {
		t.Fatalf("post-success cold-started Protocol V1: %v", activeErr)
	}
}

// TestInitialBootstrapProtocolV1RollbackCorruptedActivePointerFailsClosed 证明
// active 指针仍存在但实例表查找失败时 fail-closed：返回 conflict 且绝不调用 Stop。
func TestInitialBootstrapProtocolV1RollbackCorruptedActivePointerFailsClosed(t *testing.T) {
	starter := newPluginRuntimeFullSetStarter()
	manager := NewManager(ManagerConfig{Starter: starter})
	inventory := newPluginRuntimeFullSetTestInventory()
	legacy := inventory.seed(t, "boot.corrupt.v1", 1, "1.0.0", "boot-corrupt-v1")
	markPluginRuntimeFullSetProtocolV1(t, inventory, legacy.ID, "1.0.0", "boot-corrupt-v1")

	exact := pluginRuntimeExactExtension(
		inventory.extensions[legacy.ID],
		inventory.versions[inventory.versionKey(legacy.ID, "1.0.0", pluginRuntimeFullSetDigest("boot-corrupt-v1"))],
		inventory.member(legacy.ID, 1, "1.0.0", "boot-corrupt-v1"),
	)
	if err := manager.Start(context.Background(), exact); err != nil {
		t.Fatal(err)
	}
	active, err := manager.ActiveRuntimeInstance(legacy.ID)
	if err != nil {
		t.Fatal(err)
	}
	stopsBefore := starter.legacyStopCount.Load()

	// 故意制造不一致 Manager 状态：active 指针保留，实例表条目删除。
	manager.mu.Lock()
	delete(manager.runtimeInstances[legacy.ID], active.Identity.InstanceID)
	if len(manager.runtimeInstances[legacy.ID]) == 0 {
		delete(manager.runtimeInstances, legacy.ID)
	}
	if manager.activeInstances[legacy.ID] != active.Identity.InstanceID {
		manager.mu.Unlock()
		t.Fatalf("precondition: active pointer missing after corruption setup")
	}
	manager.mu.Unlock()

	applier := mustNewInitialBootstrapPluginRuntimeFullSetApplier(t, manager, inventory)
	err = applier.rollbackInitialProtocolV1Starts([]initialProtocolV1StartedMember{{
		identity:  active.Identity,
		extension: exact,
		member:    inventory.member(legacy.ID, 1, "1.0.0", "boot-corrupt-v1"),
	}})
	if !errors.Is(err, ErrPluginRuntimeFullSetConflict) {
		t.Fatalf("corrupted active pointer rollback error = %v, want conflict", err)
	}
	if !errors.Is(err, ErrRuntimeInstanceNotFound) {
		t.Fatalf("corrupted active pointer should wrap instance lookup failure: %v", err)
	}
	if !strings.Contains(err.Error(), legacy.ID) || !strings.Contains(err.Error(), active.Identity.InstanceID) {
		t.Fatalf("conflict should include extension/instance ids: %v", err)
	}
	if starter.legacyStopCount.Load() != stopsBefore {
		t.Fatalf("corrupted active pointer must not call Stop: stops before=%d after=%d",
			stopsBefore, starter.legacyStopCount.Load())
	}
}

func TestInitialBootstrapProtocolV1UsesExactPublicationArtifact(t *testing.T) {
	starter := newPluginRuntimeFullSetStarter()
	manager := NewManager(ManagerConfig{Starter: starter})
	inventory := newPluginRuntimeFullSetTestInventory()
	legacy := inventory.seed(t, "boot.exact.v1", 11, "1.0.0", "boot-exact-v1")
	markPluginRuntimeFullSetProtocolV1(t, inventory, legacy.ID, "1.0.0", "boot-exact-v1")

	// 污染可变 active 元数据；full-set 必须仍解析 publication 的 exact 历史版本。
	inventory.mu.Lock()
	metadata := inventory.extensions[legacy.ID]
	metadata.Version = "9.9.9"
	metadata.PackageDigest = pluginRuntimeFullSetDigest("mutable-active-digest")
	metadata.ActiveVersionID = 999
	metadata.Manifest.Version = "9.9.9"
	inventory.extensions[legacy.ID] = metadata
	inventory.mu.Unlock()

	wantDigest := pluginRuntimeFullSetDigest("boot-exact-v1")
	applier := mustNewInitialBootstrapPluginRuntimeFullSetApplier(t, manager, inventory)
	applied, err := applier.ApplyPluginRuntimeFullSet(context.Background(), inventory.publication(
		extensions.PluginRuntimePublicationStartupReconcile,
		inventory.member(legacy.ID, 11, "1.0.0", "boot-exact-v1"),
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 1 || applied[0].ExtensionVersion != "1.0.0" || applied[0].PackageDigest != wantDigest {
		t.Fatalf("applied exact evidence = %#v", applied)
	}
	active, err := manager.ActiveRuntimeInstance(legacy.ID)
	if err != nil || active.ExtensionVersion != "1.0.0" || active.ArtifactDigest != wantDigest {
		t.Fatalf("active exact artifact = %#v, %v", active, err)
	}
	started, ok := starter.startedExtension(active.Identity)
	if !ok || started.Version != "1.0.0" || started.PackageDigest != wantDigest || started.ActiveVersionID != 11 {
		t.Fatalf("started extension used mutable metadata: %#v ok=%v", started, ok)
	}
}

func markPluginRuntimeFullSetProtocolV1(
	t *testing.T,
	inventory *pluginRuntimeFullSetTestInventory,
	id, version, digestLabel string,
) {
	t.Helper()
	inventory.mu.Lock()
	defer inventory.mu.Unlock()
	key := inventory.versionKey(id, version, pluginRuntimeFullSetDigest(digestLabel))
	exact, ok := inventory.versions[key]
	if !ok {
		t.Fatalf("missing version %s@%s", id, version)
	}
	exact.Manifest.Backend.ProtocolVersion = 1
	exact.Manifest.Backend.HostAPIVersion = ""
	exact.Manifest.Lifecycle = nil
	inventory.versions[key] = exact
	metadata := inventory.extensions[id]
	metadata.Manifest.Backend.ProtocolVersion = 1
	metadata.Manifest.Backend.HostAPIVersion = ""
	metadata.Manifest.Lifecycle = nil
	inventory.extensions[id] = metadata
}

func mustNewInitialBootstrapPluginRuntimeFullSetApplier(
	t *testing.T,
	manager *Manager,
	inventory PluginRuntimeFullSetInventory,
) *ManagerPluginRuntimeFullSetApplier {
	t.Helper()
	applier, err := NewInitialBootstrapManagerPluginRuntimeFullSetApplier(manager, inventory)
	if err != nil {
		t.Fatal(err)
	}
	return applier
}

// Full-set apply must accept exact Protocol V2 process members that omit optional
// Manifest.lifecycle (for example genesis admin-surface-reference plugins).
func TestPluginRuntimeFullSetAppliesProtocolV2WithoutLifecycle(t *testing.T) {
	starter := newPluginRuntimeFullSetStarter()
	manager := NewManager(ManagerConfig{Starter: starter})
	inventory := newPluginRuntimeFullSetTestInventory()
	plugin := inventory.seed(t, "sforum.admin-surface-reference", 7, "1.0.0", "admin-surface-no-lifecycle")

	inventory.mu.Lock()
	key := inventory.versionKey(plugin.ID, plugin.Version, plugin.PackageDigest)
	version := inventory.versions[key]
	version.Manifest.Lifecycle = nil
	inventory.versions[key] = version
	metadata := inventory.extensions[plugin.ID]
	metadata.Manifest.Lifecycle = nil
	inventory.extensions[plugin.ID] = metadata
	inventory.mu.Unlock()

	if err := validateManagedStagedExtension(pluginRuntimeExactExtension(
		inventory.extensions[plugin.ID], version, inventory.member(plugin.ID, 7, "1.0.0", "admin-surface-no-lifecycle"),
	)); err != nil {
		t.Fatalf("desired Protocol V2 without lifecycle: %v", err)
	}

	applier := mustNewPluginRuntimeFullSetApplier(t, manager, inventory)
	applied, err := applier.ApplyPluginRuntimeFullSet(context.Background(), inventory.publication(
		extensions.PluginRuntimePublicationEnable,
		inventory.member(plugin.ID, 7, "1.0.0", "admin-surface-no-lifecycle"),
	))
	if err != nil {
		t.Fatalf("full-set apply Protocol V2 without lifecycle: %v", err)
	}
	if len(applied) != 1 || applied[0].ExtensionID != plugin.ID || applied[0].ExtensionVersion != "1.0.0" {
		t.Fatalf("applied = %#v", applied)
	}
	active, err := manager.ActiveRuntimeInstance(plugin.ID)
	if err != nil || active.ExtensionVersion != "1.0.0" || active.ArtifactDigest != plugin.PackageDigest {
		t.Fatalf("active without lifecycle = %#v, %v", active, err)
	}
}

var _ StagedRuntimeStarter = (*pluginRuntimeFullSetStarter)(nil)
var _ StagedRuntimeSetStarter = (*pluginRuntimeFullSetStarter)(nil)
