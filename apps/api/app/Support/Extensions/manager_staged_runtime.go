package extensionsruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

// StageRuntimeInstance starts one inert Protocol V2 process and retains its
// Manager admission gate without changing the active runtime pointer.
func (m *Manager) StageRuntimeInstance(ctx context.Context, extension extensions.Extension) (RuntimeInstanceSnapshot, error) {
	if m == nil || ctx == nil {
		return RuntimeInstanceSnapshot{}, ErrRuntimeAdmissionInvalid
	}
	if err := ctx.Err(); err != nil {
		return RuntimeInstanceSnapshot{}, err
	}
	frozen, err := cloneManagedRuntimeExtension(extension)
	if err != nil {
		return RuntimeInstanceSnapshot{}, err
	}
	if err := validateManagedStagedExtension(frozen); err != nil {
		return RuntimeInstanceSnapshot{}, err
	}
	starter, ok := m.starter.(StagedRuntimeStarter)
	if !ok {
		return RuntimeInstanceSnapshot{}, ErrProtocolInstanceUnsupported
	}

	unlock := m.lockRuntimeLifecycle(frozen.ID)
	defer unlock()
	target, err := starter.StartInstance(ctx, frozen)
	if err != nil {
		return RuntimeInstanceSnapshot{}, err
	}
	identity, err := normalizeRuntimeInstanceIdentity(RuntimeInstanceIdentity{
		ExtensionID: frozen.ID,
		InstanceID:  target.InstanceID,
	})
	if err != nil {
		return RuntimeInstanceSnapshot{}, discardStagedRuntimeAfterFailure(starter, identity, err)
	}
	protocolSnapshot, err := starter.InspectInstance(identity)
	if err != nil {
		return RuntimeInstanceSnapshot{}, discardStagedRuntimeAfterFailure(starter, identity, err)
	}
	if err := validateManagedProtocolSnapshot(protocolSnapshot, frozen, identity, ProtocolRuntimeStaged); err != nil {
		return RuntimeInstanceSnapshot{}, discardStagedRuntimeAfterFailure(starter, identity, err)
	}

	gate, err := NewRuntimeAdmissionGate(identity)
	if err != nil {
		return RuntimeInstanceSnapshot{}, discardStagedRuntimeAfterFailure(starter, identity, err)
	}
	m.mu.Lock()
	instances := m.runtimeInstances[frozen.ID]
	if instances == nil {
		instances = make(map[string]*managedRuntimeInstance)
		m.runtimeInstances[frozen.ID] = instances
	}
	if instances[identity.InstanceID] != nil {
		m.mu.Unlock()
		return RuntimeInstanceSnapshot{}, discardStagedRuntimeAfterFailure(
			starter, identity, fmt.Errorf("%w: %s/%s", ErrRuntimeInstanceConflict, identity.ExtensionID, identity.InstanceID),
		)
	}
	target.InstanceID = identity.InstanceID
	instance := &managedRuntimeInstance{
		extension: frozen, extensionVersion: frozen.Version, artifactDigest: frozen.PackageDigest,
		target: target, gate: gate,
	}
	instances[identity.InstanceID] = instance
	if m.activeInstances[frozen.ID] == "" {
		m.statuses[frozen.ID] = managedRuntimeStatus(frozen, extensions.RuntimeStarting, nil)
	}
	snapshot := m.runtimeInstanceSnapshotLocked(identity, instance)
	m.mu.Unlock()
	return snapshot, nil
}

// HealthRuntimeInstance performs health and readiness against one exact
// staged/published/retained process and verifies the frozen artifact again.
func (m *Manager) HealthRuntimeInstance(ctx context.Context, identity RuntimeInstanceIdentity) (ProtocolRuntimeInstanceSnapshot, error) {
	if m == nil || ctx == nil {
		return ProtocolRuntimeInstanceSnapshot{}, ErrRuntimeAdmissionInvalid
	}
	identity, err := normalizeRuntimeInstanceIdentity(identity)
	if err != nil {
		return ProtocolRuntimeInstanceSnapshot{}, err
	}
	starter, ok := m.starter.(StagedRuntimeStarter)
	if !ok {
		return ProtocolRuntimeInstanceSnapshot{}, ErrProtocolInstanceUnsupported
	}
	unlock := m.lockRuntimeLifecycle(identity.ExtensionID)
	defer unlock()
	extension, err := m.managedRuntimeExtension(identity)
	if err != nil {
		return ProtocolRuntimeInstanceSnapshot{}, err
	}
	if _, err := starter.HealthInstance(ctx, identity); err != nil {
		return ProtocolRuntimeInstanceSnapshot{}, err
	}
	snapshot, err := starter.InspectInstance(identity)
	if err != nil {
		return ProtocolRuntimeInstanceSnapshot{}, err
	}
	if err := validateManagedProtocolSnapshot(snapshot, extension, identity, ""); err != nil {
		return ProtocolRuntimeInstanceSnapshot{}, err
	}
	if !snapshot.Healthy || !snapshot.Ready || !snapshot.ReadinessChecked {
		return ProtocolRuntimeInstanceSnapshot{}, ErrProtocolInstanceNotReady
	}
	return snapshot, nil
}

// PublishRuntimeInstance switches ProtocolStarter first while the old Manager
// gate is closed, then publishes the same exact identity to Manager callers.
// The tiny cross-registry window is fail-closed because service/job admission
// still resolves the drained old Manager pointer until this method commits.
func (m *Manager) PublishRuntimeInstance(ctx context.Context, identity RuntimeInstanceIdentity) (RuntimeInstanceSnapshot, error) {
	if m == nil || ctx == nil {
		return RuntimeInstanceSnapshot{}, ErrRuntimeAdmissionInvalid
	}
	identity, err := normalizeRuntimeInstanceIdentity(identity)
	if err != nil {
		return RuntimeInstanceSnapshot{}, err
	}
	if err := ctx.Err(); err != nil {
		return RuntimeInstanceSnapshot{}, err
	}
	starter, ok := m.starter.(StagedRuntimeStarter)
	if !ok {
		return RuntimeInstanceSnapshot{}, ErrProtocolInstanceUnsupported
	}
	unlock := m.lockRuntimeLifecycle(identity.ExtensionID)
	defer unlock()

	m.mu.Lock()
	instance, err := m.runtimeInstanceLocked(identity)
	if err != nil {
		m.mu.Unlock()
		return RuntimeInstanceSnapshot{}, err
	}
	if instance.transitioning {
		m.mu.Unlock()
		return RuntimeInstanceSnapshot{}, fmt.Errorf("%w: %s/%s", ErrRuntimeInstanceBusy, identity.ExtensionID, identity.InstanceID)
	}
	activeID := m.activeInstances[identity.ExtensionID]
	var activeInstance *managedRuntimeInstance
	if activeID == identity.InstanceID {
		snapshot := m.runtimeInstanceSnapshotLocked(identity, instance)
		extension := instance.extension
		m.mu.Unlock()
		protocolSnapshot, err := starter.PublishInstance(ctx, identity)
		if err != nil {
			return RuntimeInstanceSnapshot{}, err
		}
		if err := validateManagedProtocolSnapshot(protocolSnapshot, extension, identity, ProtocolRuntimePublished); err != nil {
			return RuntimeInstanceSnapshot{}, err
		}
		return snapshot, nil
	}
	if activeID != "" {
		activeInstance = m.runtimeInstances[identity.ExtensionID][activeID]
		if activeInstance == nil {
			m.mu.Unlock()
			return RuntimeInstanceSnapshot{}, fmt.Errorf("%w: active runtime pointer is missing", ErrRuntimeAdmissionInvalid)
		}
		activeAdmission := activeInstance.gate.Snapshot()
		if !activeAdmission.Draining || activeAdmission.ActiveTotal != 0 {
			m.mu.Unlock()
			return RuntimeInstanceSnapshot{}, fmt.Errorf("%w: %s/%s", ErrRuntimeInstanceNotDrained, identity.ExtensionID, activeID)
		}
	}
	candidateAdmission := instance.gate.Snapshot()
	candidateWasDraining := candidateAdmission.Draining
	if candidateAdmission.ActiveTotal != 0 {
		m.mu.Unlock()
		return RuntimeInstanceSnapshot{}, fmt.Errorf("%w: %s/%s", ErrRuntimeInstanceBusy, identity.ExtensionID, identity.InstanceID)
	}
	if candidateAdmission.Draining {
		if _, err := instance.gate.Resume(); err != nil {
			m.mu.Unlock()
			return RuntimeInstanceSnapshot{}, err
		}
	}
	instance.transitioning = true
	if activeInstance != nil {
		activeInstance.transitioning = true
	}
	extension := instance.extension
	m.mu.Unlock()

	protocolSnapshot, publishErr := starter.PublishInstance(ctx, identity)
	if publishErr == nil {
		publishErr = validateManagedProtocolSnapshot(protocolSnapshot, extension, identity, ProtocolRuntimePublished)
	}
	if publishErr != nil {
		m.mu.Lock()
		if current := m.runtimeInstances[identity.ExtensionID][identity.InstanceID]; current == instance {
			current.transitioning = false
			if candidateWasDraining {
				current.gate.BeginDrain()
			}
		}
		if current := m.runtimeInstances[identity.ExtensionID][activeID]; current == activeInstance && current != nil {
			current.transitioning = false
		}
		m.mu.Unlock()
		return RuntimeInstanceSnapshot{}, publishErr
	}

	now := time.Now().UTC()
	m.mu.Lock()
	current, currentErr := m.runtimeInstanceLocked(identity)
	if currentErr != nil || current != instance || m.activeInstances[identity.ExtensionID] != activeID {
		if current == instance {
			current.transitioning = false
		}
		if activeInstance != nil {
			activeInstance.transitioning = false
		}
		m.mu.Unlock()
		if currentErr != nil {
			return RuntimeInstanceSnapshot{}, currentErr
		}
		return RuntimeInstanceSnapshot{}, fmt.Errorf("%w: runtime publication changed concurrently", ErrRuntimeInstanceConflict)
	}
	instance.transitioning = false
	if activeInstance != nil {
		activeInstance.transitioning = false
	}
	m.activeInstances[identity.ExtensionID] = identity.InstanceID
	m.targets[identity.ExtensionID] = instance.target
	m.running[identity.ExtensionID] = extension
	m.statuses[identity.ExtensionID] = managedRuntimeStatus(extension, extensions.RuntimeRunning, &now)
	snapshot := m.runtimeInstanceSnapshotLocked(identity, instance)
	m.mu.Unlock()
	m.hooks.Register(extension)
	return snapshot, nil
}

// StopRuntimeInstance stops one exact V2 process after its gate is drained and
// idle. Stopping a retained instance cannot clear the active replacement.
func (m *Manager) StopRuntimeInstance(ctx context.Context, identity RuntimeInstanceIdentity) error {
	return m.removeManagedProtocolRuntime(ctx, identity, false)
}

// DiscardRuntimeInstance destroys only an unpublished inactive candidate.
func (m *Manager) DiscardRuntimeInstance(ctx context.Context, identity RuntimeInstanceIdentity) error {
	return m.removeManagedProtocolRuntime(ctx, identity, true)
}

func (m *Manager) removeManagedProtocolRuntime(ctx context.Context, identity RuntimeInstanceIdentity, discard bool) error {
	if m == nil || ctx == nil {
		return ErrRuntimeAdmissionInvalid
	}
	identity, err := normalizeRuntimeInstanceIdentity(identity)
	if err != nil {
		return err
	}
	starter, ok := m.starter.(StagedRuntimeStarter)
	if !ok {
		return ErrProtocolInstanceUnsupported
	}
	unlock := m.lockRuntimeLifecycle(identity.ExtensionID)
	defer unlock()

	m.mu.Lock()
	instance, err := m.runtimeInstanceLocked(identity)
	if err != nil {
		m.mu.Unlock()
		return err
	}
	active := m.activeInstances[identity.ExtensionID] == identity.InstanceID
	admission := instance.gate.Snapshot()
	if instance.transitioning || admission.ActiveTotal != 0 {
		m.mu.Unlock()
		return fmt.Errorf("%w: %s/%s", ErrRuntimeInstanceBusy, identity.ExtensionID, identity.InstanceID)
	}
	if discard && active {
		m.mu.Unlock()
		return fmt.Errorf("%w: %s/%s", ErrRuntimeInstanceActive, identity.ExtensionID, identity.InstanceID)
	}
	if !discard && (!admission.Draining || admission.ActiveTotal != 0) {
		m.mu.Unlock()
		return fmt.Errorf("%w: %s/%s", ErrRuntimeInstanceNotDrained, identity.ExtensionID, identity.InstanceID)
	}
	instance.transitioning = true
	extension := instance.extension
	m.mu.Unlock()

	if discard {
		err = starter.DiscardInstance(ctx, identity)
	} else {
		err = starter.StopInstance(ctx, identity)
	}
	if err != nil {
		m.mu.Lock()
		if current := m.runtimeInstances[identity.ExtensionID][identity.InstanceID]; current == instance {
			current.transitioning = false
		}
		m.mu.Unlock()
		return err
	}

	m.mu.Lock()
	if current := m.runtimeInstances[identity.ExtensionID][identity.InstanceID]; current == instance {
		delete(m.runtimeInstances[identity.ExtensionID], identity.InstanceID)
		if len(m.runtimeInstances[identity.ExtensionID]) == 0 {
			delete(m.runtimeInstances, identity.ExtensionID)
		}
		if active && m.activeInstances[identity.ExtensionID] == identity.InstanceID {
			delete(m.activeInstances, identity.ExtensionID)
			delete(m.targets, identity.ExtensionID)
			delete(m.running, identity.ExtensionID)
			m.statuses[identity.ExtensionID] = managedRuntimeStatus(extension, extensions.RuntimeStopped, nil)
		} else if m.activeInstances[identity.ExtensionID] == "" && len(m.runtimeInstances[identity.ExtensionID]) == 0 {
			m.statuses[identity.ExtensionID] = managedRuntimeStatus(extension, extensions.RuntimeStopped, nil)
		}
	}
	m.mu.Unlock()
	if active {
		m.hooks.Unregister(identity.ExtensionID)
		if m.resilience != nil {
			m.resilience.remove(identity.ExtensionID)
		}
	}
	return nil
}

func (m *Manager) managedRuntimeExtension(identity RuntimeInstanceIdentity) (extensions.Extension, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	instance, err := m.runtimeInstanceLocked(identity)
	if err != nil {
		return extensions.Extension{}, err
	}
	return instance.extension, nil
}

func validateManagedProtocolSnapshot(
	snapshot ProtocolRuntimeInstanceSnapshot,
	extension extensions.Extension,
	identity RuntimeInstanceIdentity,
	wantState ProtocolRuntimeInstanceState,
) error {
	manifestDigest, err := protocolRuntimeManifestDigest(extension.Manifest)
	if err != nil {
		return fmt.Errorf("%w: encode frozen runtime manifest: %v", ErrRuntimeInstanceConflict, err)
	}
	if snapshot.Identity != identity || snapshot.Target.InstanceID != identity.InstanceID ||
		snapshot.ExtensionVersion != extension.Version || snapshot.ArtifactDigest != extension.PackageDigest ||
		snapshot.ManifestDigest != manifestDigest ||
		snapshot.ProtocolVersion != 2 || (wantState != "" && snapshot.State != wantState) {
		return fmt.Errorf("%w: protocol snapshot does not match exact artifact", ErrRuntimeInstanceConflict)
	}
	return nil
}

func validateManagedStagedExtension(extension extensions.Extension) error {
	if extension.ID == "" || extension.ID != strings.TrimSpace(extension.ID) ||
		extension.Version == "" || extension.Version != strings.TrimSpace(extension.Version) ||
		extension.PackageDigest == "" || extension.PackageDigest != strings.TrimSpace(extension.PackageDigest) ||
		extension.Type != extensions.TypePlugin || extension.Manifest.ID != extension.ID ||
		extension.Manifest.Version != extension.Version || extension.Manifest.Type != extensions.TypePlugin ||
		extension.Manifest.Backend.ProtocolVersion != 2 || extension.Manifest.Lifecycle == nil ||
		strings.TrimSpace(extension.Manifest.Lifecycle.ContractVersion) == "" ||
		strings.TrimSpace(extension.Manifest.Lifecycle.ContractVersion) != extension.Manifest.Lifecycle.ContractVersion {
		return fmt.Errorf("%w: exact Protocol V2 lifecycle artifact is required", ErrRuntimeAdmissionInvalid)
	}
	return nil
}

func cloneManagedRuntimeExtension(extension extensions.Extension) (extensions.Extension, error) {
	document, err := json.Marshal(extension.Manifest)
	if err != nil {
		return extensions.Extension{}, fmt.Errorf("%w: encode runtime manifest: %v", ErrRuntimeAdmissionInvalid, err)
	}
	var manifest extensions.Manifest
	if err := json.Unmarshal(document, &manifest); err != nil {
		return extensions.Extension{}, fmt.Errorf("%w: clone runtime manifest: %v", ErrRuntimeAdmissionInvalid, err)
	}
	clone := extension
	clone.Manifest = manifest
	clone.CapabilityGrants = append([]extensions.CapabilityGrant(nil), extension.CapabilityGrants...)
	clone.StagedVersion = nil
	if extension.Runtime != nil {
		status := *extension.Runtime
		clone.Runtime = &status
	}
	return clone, nil
}

func managedRuntimeStatus(extension extensions.Extension, state string, startedAt *time.Time) extensions.RuntimeStatus {
	return extensions.RuntimeStatus{
		State: state, StartedAt: startedAt,
		RouteCount: len(extension.Manifest.Routes), HookCount: len(extensions.DeclaredManifestEvents(extension.Manifest)),
		EventCount: len(extensions.DeclaredManifestEvents(extension.Manifest)), ProviderCount: len(extension.Manifest.Providers),
	}
}

func discardStagedRuntimeAfterFailure(starter StagedRuntimeStarter, identity RuntimeInstanceIdentity, cause error) error {
	if identity.ExtensionID == "" || identity.InstanceID == "" {
		return cause
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := starter.DiscardInstance(ctx, identity); err != nil && !errors.Is(err, ErrRuntimeInstanceNotFound) {
		return errors.Join(cause, fmt.Errorf("discard failed staged runtime: %w", err))
	}
	return cause
}
