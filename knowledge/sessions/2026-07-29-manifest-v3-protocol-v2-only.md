# 2026-07-29 Manifest V3 / Protocol V2 Handoff

## Changed

- Manifest loading and validation now require explicit V3; missing or older
  versions fail installation.
- Executable startup now requires Protocol V2 and Host API V2. V1 runtime,
  SDK, telemetry, full-set bootstrap, CLI scaffold, fixture, built-in binary,
  build-tag, and rollback-manifest paths were removed.
- Built-ins, fixtures, author docs, extension governance, tests, and knowledge
  records now describe one V3/V2 baseline.
- Shared Controller, Models, ExtensionPackage, transition, trust, and archive
  fixtures were migrated to valid V3 exact-artifact declarations.

## Decisions

- No compatibility window is retained before the first public release.
- Old packages must be rebuilt and repackaged; the Host never normalizes or
  downgrades them.
- Independent theme-loader APILTS rules remain unchanged.

## Evidence

- Focused Manifest, ExtensionPackage, Models/Extensions, Controllers, runtime,
  SDK, CLI, and CompatFarm suites pass.
- The formal SEO ZIP install/trust/enable/restart/upgrade/rollback/disable/
  uninstall chain passes against a clean migrated PostgreSQL database with a
  real Protocol V2 subprocess.
- The shared development database was not modified; its older active default
  theme row explains why the same formal test fails against that dirty state.
- `git diff --check` passes. The full `go test ./...` run completed but is not
  green: packages sharing one PostgreSQL test database raced over schema and
  publication fixtures, `/bin/ps` is denied by the sandbox, and concurrent
  Identity/Lifecycle work has unrelated failures. The protocol-focused suites
  pass when run independently.
- Final focused reruns pass for `ExtensionManifest`, `ExtensionRuntime`, both
  plugin SDK packages, `HostAPI`, `APILTS`, and `CompatFarm`. Static scanning
  finds no V1 executable, manifest, SDK serve, or rollback-manifest file.
- The architecture gate now has no Manifest/Protocol baseline finding; its four
  remaining findings belong to concurrent Attachments, Options, Lifecycle, and
  Pages edits. The V3 catalog gate is blocked by the unrelated new
  `POST /api/v1/admin/site/brand-assets` route lacking a stable identity map.

## Next

- Resolve the repository-wide shared-database test isolation and unrelated
  dirty-worktree architecture/catalog failures before expecting the aggregate
  gate to pass.
- Operators with pre-switch development data should activate the staged V3
  default theme or reset disposable data before expecting dependency preflight
  to pass.

## Open Questions

- None for protocol compatibility.
