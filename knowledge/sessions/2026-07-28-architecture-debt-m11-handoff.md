# 2026-07-28 Architecture Debt M11 Handoff

## Changed

- Added stable `ExtensionRuntime`, `ExtensionProtocol`,
  `ExtensionDatabase`, and `ExtensionComposition` packages.
- Moved runtime admission and exact-instance contracts to
  `ExtensionRuntime`; the legacy package retains type/error aliases.
- Moved route proxy/target DTOs to `ExtensionProtocol`; Provider and bootstrap
  runtime interfaces now consume the stable types directly.
- Moved Host-owned database identifier resolution to `ExtensionDatabase` and
  kept legacy naming functions as compatibility adapters.
- Added a redacted `ExtensionComposition.Inspector` contract. The runtime
  adapter projects only whitelisted conflict/trace fields, and the HTTP layer
  no longer returns package artifacts, schemas, runtime identities, steps, or
  raw renderer errors.
- Removed the Attachments model's concrete dependency on
  `Support/Extensions` through a stable storage adapter-factory boundary.

## Decisions

- `Manager`, `ProtocolStarter`, Protocol V2 Host integration, and lifecycle
  registry orchestration remain named compatibility implementations. Moving
  them now would either duplicate M10's single mutable state owners or create
  stable-to-legacy cycles.
- Protocol V1 remains on the legacy import surface until its APILTS removal
  date and zero-shim conditions are both satisfied.
- Stable packages must not import Models, HTTP, bootstrap, or the legacy
  runtime package. Bootstrap owns concrete assembly.

## Evidence

- Focused runtime admission, database identifier, composition redaction,
  HTTP inspector, and attachment plugin-adapter tests passed.
- Stable packages, legacy Extensions, Controllers, Providers, Attachments,
  and bootstrap compile together.
- Architecture validation and `git diff --check` passed.

## Next

- M12 records the compatibility consumer allowlist, final ADR, module notes,
  and one final full repository gate.

## Open Questions

- Protocol V1 physical implementation extraction remains governed by APILTS;
  it is not part of this debt program's deletion scope.
