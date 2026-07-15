package extensionsruntime

// AdminSurfaceSnapshot returns only contracts whose exact runtime instance is
// active and accepting ordinary calls. The immutable registry may already hold
// a drained target during lifecycle publication, but consumers must not observe
// it before the durable lifecycle marker opens admission.
func (m *Manager) AdminSurfaceSnapshot(kind string) AdminSurfaceRegistrySnapshot {
	if m == nil || m.hooks == nil || m.hooks.adminSurfaces == nil {
		return AdminSurfaceRegistrySnapshot{}
	}
	snapshot := m.hooks.adminSurfaces.Snapshot(kind)
	visible := snapshot.Surfaces[:0]
	for _, surface := range snapshot.Surfaces {
		if m.RuntimeInstanceAvailable(RuntimeInstanceIdentity{
			ExtensionID: surface.ExtensionID,
			InstanceID:  surface.InstanceID,
		}) {
			visible = append(visible, surface)
		}
	}
	snapshot.Surfaces = visible
	return snapshot
}

// ResolveAdminSurface applies the same exact-runtime visibility fence as the
// catalog read. Invocation still acquires admission after resolution so a
// concurrent drain fails closed.
func (m *Manager) ResolveAdminSurface(id string) (AdminSurfaceContract, error) {
	if m == nil || m.hooks == nil || m.hooks.adminSurfaces == nil {
		return AdminSurfaceContract{}, ErrAdminSurfaceNotFound
	}
	contract, err := m.hooks.adminSurfaces.Resolve(id)
	if err != nil {
		return AdminSurfaceContract{}, err
	}
	if !m.RuntimeInstanceAvailable(RuntimeInstanceIdentity{
		ExtensionID: contract.ExtensionID,
		InstanceID:  contract.InstanceID,
	}) {
		return AdminSurfaceContract{}, ErrAdminSurfaceNotFound
	}
	return contract, nil
}
