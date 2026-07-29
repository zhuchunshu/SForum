# Extension Stable Package Boundaries

Date: 2026-07-28

## Status

Accepted

## Context

`app/Support/Extensions` accumulated process lifecycle, wire protocol,
database authority, registry composition, product adapters, and compatibility
shims in one Go package. M9 and M10 established focused collaborators and
single mutable state owners, but moving every implementation mechanically
would recreate cycles or duplicate runtime locks. The temporary protocol
compatibility surface described at the time was removed before public release.

## Decision

- `ExtensionRuntime` owns runtime admission, exact-instance identity/snapshot
  contracts, and narrow product-facing runtime adapter contracts. It owns no
  SQL.
- `ExtensionProtocol` owns transport-only DTOs. It does not own lifecycle
  policy and cannot import Runtime or the legacy implementation package.
- `ExtensionDatabase` owns Host database identity contracts and is the target
  package for database-local transaction implementations as their interfaces
  stabilize.
- `ExtensionComposition` owns redacted cross-registry inspection contracts.
  Runtime implementations project into these DTOs by whitelist.
- Stable packages cannot import Models, HTTP Controllers, bootstrap, or
  `Support/Extensions`.
- Product Models cannot import `Support/Extensions`; they use consumer-owned
  or stable interfaces.
- Bootstrap is the concrete assembly layer. The legacy package may retain
  Manager, ProtocolStarter, V2 Host integration, and lifecycle composition as
  named implementations, but new
  production imports are forbidden by an exact allowlist ratchet.

## Consequences

- New feature work has stable dependency directions without copying runtime
  state or widening product models.
- Existing SDK, CLI, HTTP, worker, and APILTS consumers continue to compile.
- Removing a legacy consumer tightens the allowlist immediately; adding one
  fails architecture validation.
- Later physical moves must preserve the same contracts and can proceed in
  smaller slices without changing application behavior.
