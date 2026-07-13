package extensionsruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/go-plugin"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
)

var (
	ErrProtocolInstanceUnsupported       = errors.New("protocol runtime instance operation is unsupported")
	ErrProtocolInstancePublished         = errors.New("protocol runtime instance has been published")
	ErrProtocolInstanceTransitionBlocked = errors.New("protocol v1 and v2 runtime transition requires an explicit stop")
	ErrProtocolInstanceUnhealthy         = errors.New("protocol runtime instance is unhealthy")
	ErrProtocolInstanceNotReady          = errors.New("protocol runtime instance is not ready")
)

type ProtocolRuntimeInstanceState string

const (
	ProtocolRuntimeStaged    ProtocolRuntimeInstanceState = "staged"
	ProtocolRuntimePublished ProtocolRuntimeInstanceState = "published"
	ProtocolRuntimeRetained  ProtocolRuntimeInstanceState = "retained"
)

// ProtocolRuntimeInstanceSnapshot 只暴露宿主可安全检查的 exact process 元数据。
type ProtocolRuntimeInstanceSnapshot struct {
	Identity         RuntimeInstanceIdentity
	ExtensionVersion string
	ArtifactDigest   string
	ManifestDigest   string
	ProtocolVersion  int
	Target           RouteTarget
	State            ProtocolRuntimeInstanceState
	Healthy          bool
	Ready            bool
	ReadinessChecked bool
	StartedAt        time.Time
}

// StagedRuntimeStarter 是 lifecycle coordinator 使用的 V2 exact-process 最小契约。
// 普通 Starter.Start/Stop 继续保留兼容语义，不代表 V1 支持双实例。
type StagedRuntimeStarter interface {
	StartInstance(context.Context, extensions.Extension) (RouteTarget, error)
	InspectInstance(RuntimeInstanceIdentity) (ProtocolRuntimeInstanceSnapshot, error)
	HealthInstance(context.Context, RuntimeInstanceIdentity) (PluginHealth, error)
	PublishInstance(context.Context, RuntimeInstanceIdentity) (ProtocolRuntimeInstanceSnapshot, error)
	RunLifecycleInstance(context.Context, RuntimeInstanceIdentity, extensions.Extension, LifecycleInvocation) (LifecycleRunResult, error)
	StopInstance(context.Context, RuntimeInstanceIdentity) error
	DiscardInstance(context.Context, RuntimeInstanceIdentity) error
}

type protocolRuntimeInstance struct {
	identity         RuntimeInstanceIdentity
	extensionVersion string
	artifactDigest   string
	manifestDigest   string
	protocolVersion  int
	target           RouteTarget
	client           *plugin.Client
	protocol         PluginProtocol
	registrations    []hostapi.ServiceRegistration
	healthy          bool
	ready            bool
	readinessChecked bool
	published        bool
	everPublished    bool
	startedAt        time.Time
}

// Start 保留 Starter 兼容语义：V2 内部完成 stage+publish，V1 继续硬替换。
// 需要跨 health/drain 边界编排时，V2 调用方应使用 StartInstance + PublishInstance。
func (s *ProtocolStarter) Start(ctx context.Context, extension extensions.Extension) (RouteTarget, error) {
	if s == nil {
		return RouteTarget{}, extensions.ErrRuntimeUnavailable
	}
	unlock := s.lockExtensionLifecycle(extension.ID)
	defer unlock()
	version := manifestProtocolVersion(extension)
	if s.protocolTransitionBlockedLocked(extension.ID, version) {
		return RouteTarget{}, ErrProtocolInstanceTransitionBlocked
	}
	return s.startProtocolInstanceLocked(ctx, extension, true)
}

// StartInstance 启动并验证一个未发布的 V2 进程；不会改变活动指针或 Registry。
func (s *ProtocolStarter) StartInstance(ctx context.Context, extension extensions.Extension) (RouteTarget, error) {
	if s == nil {
		return RouteTarget{}, extensions.ErrRuntimeUnavailable
	}
	if manifestProtocolVersion(extension) != 2 {
		return RouteTarget{}, ErrProtocolInstanceUnsupported
	}
	unlock := s.lockExtensionLifecycle(extension.ID)
	defer unlock()
	if s.protocolTransitionBlockedLocked(extension.ID, 2) {
		return RouteTarget{}, ErrProtocolInstanceTransitionBlocked
	}
	return s.startProtocolInstanceLocked(ctx, extension, false)
}

func (s *ProtocolStarter) InspectInstance(identity RuntimeInstanceIdentity) (ProtocolRuntimeInstanceSnapshot, error) {
	if s == nil {
		return ProtocolRuntimeInstanceSnapshot{}, extensions.ErrRuntimeUnavailable
	}
	identity, err := normalizeRuntimeInstanceIdentity(identity)
	if err != nil {
		return ProtocolRuntimeInstanceSnapshot{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	instance := s.runtimeInstanceLocked(identity)
	if instance == nil {
		return ProtocolRuntimeInstanceSnapshot{}, protocolInstanceNotFound(identity)
	}
	return protocolRuntimeSnapshot(instance), nil
}

// HealthInstance 对 exact V2 进程重新执行 health+readiness，不读取活动回退实例。
func (s *ProtocolStarter) HealthInstance(ctx context.Context, identity RuntimeInstanceIdentity) (PluginHealth, error) {
	if s == nil {
		return PluginHealth{}, extensions.ErrRuntimeUnavailable
	}
	identity, err := normalizeRuntimeInstanceIdentity(identity)
	if err != nil {
		return PluginHealth{}, err
	}
	unlock := s.lockExtensionLifecycle(identity.ExtensionID)
	defer unlock()
	instance := s.protocolInstance(identity)
	if instance == nil {
		return PluginHealth{}, protocolInstanceNotFound(identity)
	}
	if instance.protocolVersion != 2 {
		return PluginHealth{}, ErrProtocolInstanceUnsupported
	}
	return s.healthProtocolInstanceLocked(ctx, instance)
}

// PublishInstance 健康检查后原子替换 V2 Service Registry，再切换活动 transport 指针。
// 调用方在此边界外负责关闭 Manager admission，避免跨 Registry 的新 ordinary 调用。
func (s *ProtocolStarter) PublishInstance(ctx context.Context, identity RuntimeInstanceIdentity) (ProtocolRuntimeInstanceSnapshot, error) {
	if s == nil {
		return ProtocolRuntimeInstanceSnapshot{}, extensions.ErrRuntimeUnavailable
	}
	identity, err := normalizeRuntimeInstanceIdentity(identity)
	if err != nil {
		return ProtocolRuntimeInstanceSnapshot{}, err
	}
	unlock := s.lockExtensionLifecycle(identity.ExtensionID)
	defer unlock()
	instance := s.protocolInstance(identity)
	if instance == nil {
		return ProtocolRuntimeInstanceSnapshot{}, protocolInstanceNotFound(identity)
	}
	if instance.protocolVersion != 2 {
		return ProtocolRuntimeInstanceSnapshot{}, ErrProtocolInstanceUnsupported
	}
	return s.publishProtocolInstanceLocked(ctx, identity, true)
}

// StopInstance 停止任意 staged/published/retained V2 exact process。
func (s *ProtocolStarter) StopInstance(_ context.Context, identity RuntimeInstanceIdentity) error {
	if s == nil {
		return extensions.ErrRuntimeUnavailable
	}
	identity, err := normalizeRuntimeInstanceIdentity(identity)
	if err != nil {
		return err
	}
	unlock := s.lockExtensionLifecycle(identity.ExtensionID)
	defer unlock()
	return s.stopProtocolInstanceLocked(identity, false)
}

// DiscardInstance 只销毁从未发布的 V2 候选，不能误删活动或 rollback 保留实例。
func (s *ProtocolStarter) DiscardInstance(_ context.Context, identity RuntimeInstanceIdentity) error {
	if s == nil {
		return extensions.ErrRuntimeUnavailable
	}
	identity, err := normalizeRuntimeInstanceIdentity(identity)
	if err != nil {
		return err
	}
	unlock := s.lockExtensionLifecycle(identity.ExtensionID)
	defer unlock()
	instance := s.protocolInstance(identity)
	if instance == nil {
		return protocolInstanceNotFound(identity)
	}
	if instance.protocolVersion != 2 {
		return ErrProtocolInstanceUnsupported
	}
	if instance.published || instance.everPublished {
		return fmt.Errorf("%w: %s/%s", ErrProtocolInstancePublished, identity.ExtensionID, identity.InstanceID)
	}
	s.removeProtocolInstanceLocked(identity)
	instance.client.Kill()
	return nil
}

func (s *ProtocolStarter) retainProtocolInstanceLocked(instance *protocolRuntimeInstance) error {
	if instance == nil || instance.client == nil || instance.protocol == nil {
		return ErrRuntimeAdmissionInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runtimeInstances == nil {
		s.runtimeInstances = make(map[string]map[string]*protocolRuntimeInstance)
	}
	instances := s.runtimeInstances[instance.identity.ExtensionID]
	if instances == nil {
		instances = make(map[string]*protocolRuntimeInstance)
		s.runtimeInstances[instance.identity.ExtensionID] = instances
	}
	if instances[instance.identity.InstanceID] != nil {
		return fmt.Errorf("%w: %s/%s", ErrRuntimeInstanceConflict, instance.identity.ExtensionID, instance.identity.InstanceID)
	}
	instances[instance.identity.InstanceID] = instance
	return nil
}

// Caller holds the extension lifecycle lock.
func (s *ProtocolStarter) publishProtocolInstanceLocked(
	ctx context.Context,
	identity RuntimeInstanceIdentity,
	recheckHealth bool,
) (ProtocolRuntimeInstanceSnapshot, error) {
	if ctx == nil {
		return ProtocolRuntimeInstanceSnapshot{}, ErrRuntimeAdmissionInvalid
	}
	if err := ctx.Err(); err != nil {
		return ProtocolRuntimeInstanceSnapshot{}, err
	}
	instance := s.protocolInstance(identity)
	if instance == nil {
		return ProtocolRuntimeInstanceSnapshot{}, protocolInstanceNotFound(identity)
	}
	if instance.published && s.activeProtocolInstanceID(identity.ExtensionID) == identity.InstanceID {
		if recheckHealth {
			if _, err := s.healthProtocolInstanceLocked(ctx, instance); err != nil {
				return ProtocolRuntimeInstanceSnapshot{}, err
			}
		}
		return s.InspectInstance(identity)
	}
	if instance.protocolVersion == 2 {
		if s.activeProtocolVersion(identity.ExtensionID) == 1 {
			return ProtocolRuntimeInstanceSnapshot{}, ErrProtocolInstanceTransitionBlocked
		}
		if recheckHealth {
			if _, err := s.healthProtocolInstanceLocked(ctx, instance); err != nil {
				return ProtocolRuntimeInstanceSnapshot{}, err
			}
		}
		registry := protocolV2ServiceRegistryFor(s.hostAPI)
		if registry == nil {
			if len(instance.registrations) > 0 {
				return ProtocolRuntimeInstanceSnapshot{}, fmt.Errorf("protocol v2 service registry is not configured")
			}
		} else if err := registry.ReplaceProtocolV2Services(identity.ExtensionID, instance.registrations); err != nil {
			return ProtocolRuntimeInstanceSnapshot{}, fmt.Errorf("register protocol v2 services: %w", err)
		}
	} else {
		if s.hasProtocolV2Instance(identity.ExtensionID) {
			return ProtocolRuntimeInstanceSnapshot{}, ErrProtocolInstanceTransitionBlocked
		}
		if registry := protocolV2ServiceRegistryFor(s.hostAPI); registry != nil {
			registry.UnregisterProtocolV2Services(identity.ExtensionID)
		}
	}

	var previous *protocolRuntimeInstance
	s.mu.Lock()
	current := s.runtimeInstanceLocked(identity)
	if current != instance {
		s.mu.Unlock()
		return ProtocolRuntimeInstanceSnapshot{}, protocolInstanceNotFound(identity)
	}
	previousID := s.activeRuntimeInstances[identity.ExtensionID]
	if previousID != "" && previousID != identity.InstanceID {
		previous = s.runtimeInstances[identity.ExtensionID][previousID]
		if previous != nil {
			previous.published = false
			if instance.protocolVersion == 1 {
				delete(s.runtimeInstances[identity.ExtensionID], previousID)
			}
		}
	}
	instance.published = true
	instance.everPublished = true
	s.activeRuntimeInstances[identity.ExtensionID] = identity.InstanceID
	s.clients[identity.ExtensionID] = instance.client
	s.protocols[identity.ExtensionID] = instance.protocol
	snapshot := protocolRuntimeSnapshot(instance)
	s.mu.Unlock()

	// V1 沿用单实例硬替换；V2 previous 则保留供 drain/rollback。
	if instance.protocolVersion == 1 && previous != nil && previous.client != nil {
		previous.client.Kill()
	}
	return snapshot, nil
}

// Caller holds the extension lifecycle lock.
func (s *ProtocolStarter) healthProtocolInstanceLocked(ctx context.Context, instance *protocolRuntimeInstance) (PluginHealth, error) {
	if ctx == nil {
		return PluginHealth{}, ErrRuntimeAdmissionInvalid
	}
	if err := ctx.Err(); err != nil {
		return PluginHealth{}, err
	}
	if instance == nil || instance.client == nil || instance.client.Exited() {
		return PluginHealth{}, ErrProtocolInstanceUnhealthy
	}
	health, healthErr := instance.protocol.Health()
	ready := false
	readinessChecked := false
	var readinessErr error
	if healthErr == nil && health.OK {
		readiness, ok := instance.protocol.(interface{ Readiness(context.Context) error })
		if !ok {
			readinessErr = ErrProtocolInstanceUnsupported
		} else {
			readinessChecked = true
			readinessErr = readiness.Readiness(ctx)
			ready = readinessErr == nil
		}
	}
	healthy := healthErr == nil && health.OK
	s.mu.Lock()
	if current := s.runtimeInstanceLocked(instance.identity); current == instance {
		current.healthy = healthy
		current.ready = ready
		current.readinessChecked = readinessChecked
	}
	s.mu.Unlock()
	if !healthy {
		if healthErr != nil {
			return health, fmt.Errorf("%w: %v", ErrProtocolInstanceUnhealthy, healthErr)
		}
		return health, ErrProtocolInstanceUnhealthy
	}
	if !ready {
		if readinessErr != nil {
			return health, fmt.Errorf("%w: %v", ErrProtocolInstanceNotReady, readinessErr)
		}
		return health, ErrProtocolInstanceNotReady
	}
	return health, nil
}

// Caller holds the extension lifecycle lock.
func (s *ProtocolStarter) stopProtocolInstanceLocked(identity RuntimeInstanceIdentity, allowV1 bool) error {
	instance := s.protocolInstance(identity)
	if instance == nil {
		return protocolInstanceNotFound(identity)
	}
	if instance.protocolVersion != 2 && !allowV1 {
		return ErrProtocolInstanceUnsupported
	}
	s.removeProtocolInstanceLocked(identity)
	if instance.protocolVersion == 2 {
		s.unregisterProtocolV2Services(identity.ExtensionID, instance.protocol)
	} else if s.hostAPI != nil {
		s.hostAPI.UnregisterExtension(identity.ExtensionID)
	}
	instance.client.Kill()
	return nil
}

func (s *ProtocolStarter) protocolInstance(identity RuntimeInstanceIdentity) *protocolRuntimeInstance {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.runtimeInstanceLocked(identity)
}

// Caller holds s.mu.
func (s *ProtocolStarter) runtimeInstanceLocked(identity RuntimeInstanceIdentity) *protocolRuntimeInstance {
	return s.runtimeInstances[identity.ExtensionID][identity.InstanceID]
}

func (s *ProtocolStarter) removeProtocolInstanceLocked(identity RuntimeInstanceIdentity) *protocolRuntimeInstance {
	s.mu.Lock()
	defer s.mu.Unlock()
	instance := s.runtimeInstanceLocked(identity)
	if instance == nil {
		return nil
	}
	delete(s.runtimeInstances[identity.ExtensionID], identity.InstanceID)
	if len(s.runtimeInstances[identity.ExtensionID]) == 0 {
		delete(s.runtimeInstances, identity.ExtensionID)
	}
	if s.activeRuntimeInstances[identity.ExtensionID] == identity.InstanceID {
		delete(s.activeRuntimeInstances, identity.ExtensionID)
		delete(s.clients, identity.ExtensionID)
		delete(s.protocols, identity.ExtensionID)
	}
	return instance
}

func (s *ProtocolStarter) activeProtocolInstanceID(extensionID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeRuntimeInstances[extensionID]
}

func (s *ProtocolStarter) activeProtocolVersion(extensionID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	instanceID := s.activeRuntimeInstances[extensionID]
	instance := s.runtimeInstances[extensionID][instanceID]
	if instance == nil {
		return 0
	}
	return instance.protocolVersion
}

func (s *ProtocolStarter) hasProtocolV2Instance(extensionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, instance := range s.runtimeInstances[extensionID] {
		if instance.protocolVersion == 2 {
			return true
		}
	}
	return false
}

// Caller holds the extension lifecycle lock.
func (s *ProtocolStarter) protocolTransitionBlockedLocked(extensionID string, version int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if version == 1 {
		for _, instance := range s.runtimeInstances[extensionID] {
			if instance.protocolVersion == 2 {
				return true
			}
		}
		return false
	}
	if version == 2 {
		activeID := s.activeRuntimeInstances[extensionID]
		active := s.runtimeInstances[extensionID][activeID]
		return active != nil && active.protocolVersion == 1
	}
	return false
}

func protocolRuntimeSnapshot(instance *protocolRuntimeInstance) ProtocolRuntimeInstanceSnapshot {
	state := ProtocolRuntimeStaged
	if instance.published {
		state = ProtocolRuntimePublished
	} else if instance.everPublished {
		state = ProtocolRuntimeRetained
	}
	return ProtocolRuntimeInstanceSnapshot{
		Identity: instance.identity, ExtensionVersion: instance.extensionVersion,
		ArtifactDigest: instance.artifactDigest, ManifestDigest: instance.manifestDigest, ProtocolVersion: instance.protocolVersion,
		Target: instance.target, State: state, Healthy: instance.healthy,
		Ready: instance.ready, ReadinessChecked: instance.readinessChecked, StartedAt: instance.startedAt,
	}
}

func manifestProtocolVersion(extension extensions.Extension) int {
	if extension.Manifest.Backend.ProtocolVersion == 0 {
		return 1
	}
	return extension.Manifest.Backend.ProtocolVersion
}

func newProtocolRuntimeInstanceID() string {
	return "legacy-" + appevents.NewID()
}

func protocolInstanceNotFound(identity RuntimeInstanceIdentity) error {
	return fmt.Errorf("%w: %s/%s", ErrRuntimeInstanceNotFound, identity.ExtensionID, identity.InstanceID)
}

func protocolRuntimeManifestDigest(manifest extensions.Manifest) (string, error) {
	document, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(document)
	return hex.EncodeToString(digest[:]), nil
}

var _ StagedRuntimeStarter = (*ProtocolStarter)(nil)
