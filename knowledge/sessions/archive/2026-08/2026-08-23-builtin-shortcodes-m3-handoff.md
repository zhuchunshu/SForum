# 2026-08-23 Built-in Shortcodes M3 Handoff

## Status

M0-M3 are complete. M4 has not started and requires explicit user confirmation.

## Changed

- Published the sealed Core Forum target `sforum.core.content.post-body@1`
  before lifecycle replay and retained it through Safe Mode and restart.
- Replaced production identity-only Forum filtering with registry-revision
  cached bindings from exact active `render_filter`/`sanitizer` declarations.
- Added `content.runtime@1` Protocol V2 dispatch and SDK-owned actorless typed
  content DTOs. Host admission and invocation both verify declaration fields,
  artifact digest/version row, runtime instance, and acquired lease.
- Kept Host permission/schema checks, typed-segment sanitizer, timeout/output/
  depth/concurrency limits, quarantine, and bounded source-free traces
  authoritative. Any runtime, stale, invalid, or limit failure preserves Host
  HTML; Host plain/search/excerpt/hash never accept plugin output.
- Added bounded ordered `ExecuteBatch` with shared concurrency and aggregate
  output ceilings. Empty content graphs remain identity.
- Patch-bumped all seven existing built-ins because `sdk/plugin/v2` is their
  tracked shared release contract, then regenerated the release baseline.
- Added no concrete shortcode, Host Query projection, plugin package, editor
  command, or M4 behavior.

## Verification

- Focused Content Registry, HostAPI, SDK, Extensions, Bootstrap, and real
  Protocol V2 subprocess suites passed.
- Real-process matrix passed normal, XSS, invalid identity/schema, oversized
  output, timeout, crash, disable/stale lease, Safe Mode, restart/recovery,
  upgrade, and rollback scenarios.
- Targeted Content Registry/publication integration race tests passed.
- `bun test` passed 922 tests; Nuxt typecheck, OpenAPI refs, architecture
  ratchet, built-in release contracts, and `git diff --check` passed.
- `SFORUM_COMPAT_DATABASE_URL=... ./scripts/test.sh` passed, including both
  required real PostgreSQL compatibility-farm cells.

## Next

Wait for explicit confirmation, then execute M4 only. Start from:

- `apps/api/app/Support/HostAPI/query_registry_core.go`
- `apps/api/app/Support/HostAPI/query_registry_core_schema.go`
- `apps/api/app/Support/HostAPI/query_registry_protocol_v2.go`
- `apps/api/app/Support/QueryRegistry/execution_types.go`
- `apps/api/sdk/plugin/v2/query_runtime.go`
- `apps/api/app/Models/Forum/public_read_service.go`
- `apps/api/app/Models/Forum/postgres_store_ops.go`
- `apps/api/app/Models/Forum/store.go`
- `apps/api/bootstrap/api_assembly_domains.go`

M4 must expose only minimal versioned batch projections and Host-owned
visibility decisions. Raw DB/content/session/email/IP/moderation/private-avatar
data remain closed. Do not create `sforum-shortcodes` or start M5.

## Open Questions

- None for M3. Frozen M0 contract changes still require user confirmation and
  a replacement decision record.
