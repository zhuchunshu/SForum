# 2026-07-28 Architecture Debt M10 Handoff

## Changed

- Extracted `RuntimeSupervisor`, `InstanceAdmission`, `RuntimeInvoker`, and
  `RuntimeEventsProviders` inside `Support/Extensions`.
- Reduced `Manager` to a 72-method compatibility facade while keeping one
  `NewManager(ManagerConfig)` construction path.
- Centralized runtime maps, exact active-instance selection, runtime-set
  transitions, and lifecycle admission in one `runtimeAdmissionState`.
- Centralized hooks, delivery, resilience, and provider selections in one
  `runtimeEventsProviderState`.
- Reduced `manager.go` below the 1000-line hard warning and added receiver
  ratchets for every collaborator and the shared internal core.

## Decisions

- M10 remains a same-package transition; stable import paths are an M11 concern.
- The parent Manager retains compatibility entry points but does not own a
  second copy of mutable runtime state.
- Nil compatibility adapters remain fail closed, including
  `(*Manager)(nil).NewExactLifecycleCoordinatorRuntimeAdapter()`.

## Evidence

- Full `go test ./app/Support/Extensions` passed.
- Race-focused restart fencing, runtime-set publication barriers, and exact
  admission boundary tests passed.
- Architecture boundary validation and `git diff --check` passed.
- Receiver ratchets: Manager 72, RuntimeSupervisor 7, InstanceAdmission 23,
  RuntimeInvoker 28, RuntimeEventsProviders 11, internal core 45.

## Next

- M11 extracts stable Runtime, Protocol, Database, and Composition boundaries.
- Bootstrap remains the concrete assembly owner; product Models must not
  import the legacy `Support/Extensions` implementation package.

## Open Questions

- None. APILTS V1 and public SDK consumers may retain named compatibility
  adapters until their declared removal window.
