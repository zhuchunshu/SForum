# 2026-10-16 M6 Implementation Guide — Plugin-Namespaced Session State

Companion to **Revision 2** of
`knowledge/decisions/2026-10-16-extension-authoring-contracts-freeze.md` (decisions S1–S8)
and to milestone **M6** of
`knowledge/plans/2026-08-22-extension-authoring-speed-and-controlled-power.md`.

Why this file exists: M6 was held for weeks because the contract did not exist, and the
plan forbids a milestone from inventing one. The contract now exists, so this file records
**what to build, where, and what proves it** — so the next session starts from the decision
instead of re-deriving it. Nothing here is implemented; nothing here may be reported as
working until M6's tests pass.

## Scope

Build Host-owned, plugin-namespaced **session** state so a plugin can hold small bounded
browser-workflow state for the current Host session **without** receiving any Core session
authority. The plugin never supplies, and the Host never trusts, an extension id, a user id,
or a session id.

## Verified starting points (measured 2026-10-16)

| Thing | Where |
| --- | --- |
| Session storage: raw session id as the Redis key, no prefix, DB 0 (the only unprefixed key in the app) | `bootstrap/api_assembly_infrastructure.go:214`, `Support/Redis/session.go:41` |
| A login issues a **fresh** `sid`; the previous one is captured as `previousSID` in the issue path | `Support/AuthSession/manager.go:198,222` |
| Only `Regenerate()` (24 h rotation) changes the key and it preserves `sid` | `Support/AuthSession/renewal.go:104` |
| Revocation is a per-request PostgreSQL check, **not** a Redis delete; device-cap eviction revokes other sids while their keys live | `Models/Identity/sessions.go:42,218` |
| Session destroy / logout | `Support/AuthSession/manager.go:390` (`Destroy`) |
| Namespace + limits envelope to reuse | `Support/HostAPI/cache_registry_service.go:20-39` (`sforum:host-cache:v1:installation:…`, key ≤512 B, value ≤1 MiB, TTL ≤24 h), `CacheRegistry/validation.go:22` (`maxTagsPerCache=64`) |
| Privacy contribution shape (export + erase hooks) | `Support/Privacy/hooks.go:201,343` |
| Host API v2 service surface (add a service here) | `contracts/proto/sforum/host/v2/data_work.proto` (`CacheService`), `contracts/proto/sforum/host/v2/platform.proto`; runtime adapters live in `app/Support/HostAPI/` |
| Shared Redis client available for a new store | `bootstrap/api_assembly_infrastructure.go:47,120` |
| `Actor.session_id` is declared but a populated value is rejected — leave it unpopulated | `sdk/plugin/v2/identity_runtime.go:235` |

## What to build

1. **Store** (new focused owner, not `AuthSession`). Keys per S1; values opaque bytes with a
   size cap; TTL per S3; caps per S6. Two index sets per S4. Put it in a domain package (for
   example `app/Support/PluginSessionState/`) rather than in `Support/AuthSession` — the
   session package owns identity, and M6 must not add a second responsibility to it.
2. **Host API v2 service** per S7: `get` / `set` / `delete`, request carries **only**
   `namespace_key`, `value`, `ttl_ms`. Register it in the host v2 surface and in whatever
   capability/declaration table the other host services use (check `app/Support/HostAPI/`
   for the provider/admission wiring that `CacheService` uses and mirror it).
3. **Authority derivation**: extension id + runtime instance + session all come from the
   Host-attested runtime context (the same context `CacheRegistry` and the guard evaluator
   already use). Do not accept them from the request.
4. **Invalidation hooks** per S4, each with its own test:
   `InvalidateSession` on logout/destroy, on revoke, on device-cap eviction, on the fresh-`sid`
   login issue (invalidate the captured `previousSID`), and on absolute expiry;
   `InvalidateExtension` on disable and uninstall.
5. **Privacy**: register a contribution whose erase hook walks the erased user's sessions and
   invalidates each. This is the §5.2(5) requirement and it has no consumer today.
6. **Safe Mode**: reads miss, writes refused (fail closed) — never a window in which a
   quarantined extension can read state.
7. **SDK helpers** for the three operations (no Fabric/Nuxt/Redis types on the plugin-facing
   side), plus bilingual docs and one reference workflow (this is also M9's last unmet item —
   `extensions/fixtures/plugins/` — so build them together).

## Required tests (M6's exit gate; every row of the S5 matrix is one)

- Allowed path: set → get → delete round trip through the **real** Host API v2 surface.
- **Cross-plugin denial**: plugin A cannot read or write plugin B's namespace.
- **Cross-session denial**: a second session cannot see the first session's state.
- **Caller-supplied authority is unrepresentable**: a request carrying an extension id / user
  id / session id is rejected (and the request shape should not have the fields at all).
- **Rotation**: state survives `Regenerate()` (same `sid`).
- **Replacement**: a new login does not inherit the previous session's state.
- **Logout / revoke / device-cap eviction**: `InvalidateSession` reaches the state, and a
  revoked-but-Redis-alive session cannot read it.
- **Expiry**: TTL bounds per S3 — including the case where the requested TTL exceeds the
  session's remaining lifetime (the effective TTL must shrink, never extend, and plugin state
  must never extend the Core session).
- **Caps**: value > 1 MiB, key > 512 B, > 64 keys per session, > 4096 per extension all fail
  with typed errors and no partial write.
- **Privacy erase**: the erase hook removes the user's plugin state, and the run is audited
  **without** values (never log values).
- **Concurrency**: parallel set/get on the same namespace key does not corrupt state or leak
  across sessions.
- **Safe Mode**: reads miss and writes are refused.
- Full `AuthSession` and external-auth suites stay green — this must not perturb identity.

## Constraints that must not be relaxed

- Plugin state never lives in the session payload (adding a key there would make every request
  rewrite the whole-key TTL and `clearSessionData` would wipe it on unrelated failures).
- It is never keyed by the raw session id, and the raw sid never leaves the Core session key.
- `Actor.session_id` stays unpopulated.
- No value appears in audit, logs, public API, or the extension inventory.
- Architecture ratchets **decrease** if M6 extracts anything; never raise a baseline to fit.
