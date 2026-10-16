# 2026-10-16 M6 handoff — S9 delegation, the served session service, and the SDK

> Archived 2026-10-17: superseded as the hot handoff by
> `knowledge/sessions/2026-10-17-extension-authoring-closeout-handoff.md`.
> Kept because ADR Revision 3 and the M10 matrix cite it as the record of how
> S9 landed. Note (recorded 2026-10-17): its "evidence" section ran the mint
> through tests that injected the delegation authority directly; the production
> Gateway was missing the S9 issuer and was fixed the next day — see ADR
> Revision 4.3 and the closeout handoff's Corrections.

Companion to **Revision 2 + Revision 3** of
`knowledge/decisions/2026-10-16-extension-authoring-contracts-freeze.md` and to
milestone M6 of
`knowledge/plans/2026-08-22-extension-authoring-speed-and-controlled-power.md`.
Supersedes the "Phase 4/5" sections of
`sessions/2026-10-16-extension-authoring-closeout-handoff.md` as the current
handoff for this workstream.

## Outcome in one paragraph

S9 is **implemented, wired, served and tested**: a Host-signed, single-invocation
session delegation travels on the plugin invocation (`RequestContext.host_session_delegation`),
the plugin echoes it on `PluginSessionService` `get`/`set`/`delete`, and the Host
resolves the session scope from its own mint ledger, retires the token when the
invocation settles, and refuses everything else. `PluginSessionService` is now
**registered** on the production Host API gateway (safe mode + exact-runtime
admission + installation identity), the Host API v2 documentation gate and
`docs/extensions/host-api-v2.md` document it, and the plugin SDK ships
`Host.SessionState(ctx)`. **M10 row 5 stays open**: the privacy-erase consumer
(T4), the M9 reference package (T5), a joined end-to-end test, and the anonymous
clause are not done — see "Still open".

## Changed (all green)

### 1. Wire (S9)

- `contracts/proto/sforum/protocol/v2/common.proto`: new `HostSessionDelegation`
  message (one opaque `token` field) and `RequestContext.host_session_delegation = 11`
  (singular — one invocation has exactly one session). Regenerated:
  `bash scripts/proto.sh generate` → exit 0.
- `sdk/plugin/v2/session_contract_test.go` (new) pins the S7 property at
  **descriptor level**, not by eye: the three request messages have exactly
  `context` + `namespace_key` (+ `value`/`ttl_ms`), the delegation message has
  exactly `token`, and a scan over every field of every method's input/output in
  `PluginSessionService` rejects any name containing
  `extension_id|user_id|session_id|session_fingerprint|actor|sid|scope`.
  `TestRequestContextSessionDelegationIsTheOnlyNewField` pins the field number,
  kind and cardinality.

### 2. Mint/validate authority

- `app/Support/HostAPI/plugin_session_state_delegation.go` (new):
  `ProtocolV2SessionDelegationAuthority` — boot-scoped HS256 key, 2-minute TTL
  (≤5 min), separate audience `sforum.host-session.v2`, bounded mint ledger, and
  `Mint` / `Resolve` / `Retire`. The **session fingerprint, both deadlines and the
  actor id live only in the ledger record**; the token carries the runtime binding
  only (Revision 3 §3.1).
- `app/Support/Extensions/protocol_v2_actor_delegation.go`: `invocationRequestContext`
  now returns a retire function and attaches the session delegation; the actor
  delegation logic moved to `attachInvocationActorDelegation`. The session
  delegation is attached **independently of the actor**, because an authenticated
  actor is not required for a session scope.
- `app/Support/Extensions/protocol_v2_host.go`: the inbound interceptor now
  *bounds* (and accepts) a session delegation, while command/query delegations stay
  rejected as Host-to-plugin only.
- Mint sites: unary routes, streaming routes (retired in the stream's `closeOnce`),
  and admin-surface invocations. Jobs/schedules/queue consumers carry no scope in
  their Go context and therefore mint nothing.
- `app/Http/plugin_session_scope.go` (new) + `app/Http/server.go` +
  `app/bootstrap/api_assembly_http.go`: `PluginSessionScopeSource` resolves the
  request's Core session scope and publishes it on the request context before the
  route dispatcher runs.

### 3. Erasure hooks are now actually wired

The three erasure consumers existed since the previous round but had **no
production caller**. `app/bootstrap/plugin_session_state.go` (new) now builds the
store (`Support/PluginSessionState` over the shared Redis client + durable
installation id) and injects it into `AuthSession`, `Models/Identity` and
`Models/Extensions` (`WithPluginSessionStateInvalidator`). Fingerprinting the raw
`sid` happens in exactly one place (that adapter), as the AuthSession comment
promised.

### 4. Service + registration (S7)

- `app/Support/HostAPI/plugin_session_state_service.go` (new):
  `ProtocolV2SessionServiceServer` with `Get`/`Set`/`Delete`; extension identity
  from the broker-attested context, session scope from the delegation, installation
  from the Host; exact-runtime admission lease held across the store call; Safe Mode
  = silent read miss + refused writes; typed, non-leaking error details.
- `app/Support/HostAPI/v2.go` + `gateway.go`: optional registration (nil = not
  served) and `BindProtocolV2SessionService` before any broker registers.
  `app/bootstrap/api_assembly_extensions.go` binds it with the production admission
  source and `cfg.SafeMode`.

### 5. Plugin SDK (S9/T3)

- `sdk/plugin/v2/plugin_session_state.go` (new): `Host.SessionState(parent)` →
  `Get`/`Set`/`Delete`, returning `ErrPluginSessionDelegationUnavailable` when the
  invocation carried no delegation, `HostPluginSessionError{Code, Reason, ...}` for
  typed failures, and rejecting a >24 h request TTL locally. `Host.PluginSession`
  client added; `Host.RequestContext` now clears the session delegation too (it is
  consumed only by the matching helper).
- The reduced-context validators (`identity_runtime.go`, `query_runtime.go`,
  `content.go`) now also forbid delegation material on those reduced contexts.

### 6. Documentation (only after the service is really served)

- `tests/validate-host-api-v2-docs.mjs`: `session.proto` added to `hostProtoPaths`
  plus assertions for `PluginSessionService`, the three request messages,
  `namespace_key`, `host_session_delegation`, `HostSessionDelegation` and
  `Actor.session_id` → the gate now fails if the docs drop any of them.
- `docs/extensions/host-api-v2.md`: new "Plugin session state (S9)" section
  (shape, delegation flow, lifetime/limits table, failure/lifecycle table, SDK
  usage).
- `docs/extensions/migrating-from-sforum-v2.md` + `docs/zh-CN/...`: the
  "Session state: **Not available**" row was **stale** and is corrected in both
  languages (capability described, Core session *mutation* still closed).

### 7. Privacy erase consumer (T4, done)

`app/bootstrap/plugin_session_state_privacy.go` (new) registers the first
`Support/Privacy` consumer in the repository: a Host-owned contribution
(`core.host.plugin-session-state`) whose erase hook walks **all** of the erased
user's sessions (`ListUserSessions(..., includeHistory=true)`, paged, deduped by
sid) and calls the store's idempotent `InvalidateSession` for each. The report
carries counts only; values and namespace keys never reach the report, the audit
ledger, or the logs. Export is deliberately unsupported (ephemeral plugin bytes
are not a portable user record). Wired from
`app/bootstrap/api_assembly_extensions.go` after the P12 ops registry exists.

### 8. Joined end-to-end test (T6's biggest gap)

`app/Support/Extensions/plugin_session_state_joined_test.go` runs the whole S9 flow in
one test instead of at its boundaries: the Host resolves a session scope → the client
mints a delegation onto the invocation → **real gRPC** to a stub plugin runtime → the
plugin echoes the delegation on `PluginSessionService` (**real gRPC** back into a real
`Gateway` registration of the real service) → the service resolves the scope from its
own ledger and writes through the real `Store` → the invocation settles and the
delegation is retired → a replay of the same token on the real service is refused with
`host.plugin_session_scope_unattested` and reads nothing. It also asserts the state
lands under exactly that session's fingerprint and is invisible to another session.
The only stand-ins are the Redis backend and the runtime admission lease (the store's
own opt-in tests cover real Redis).

### 9. Architecture ratchet

`protocol_v2_client.go` was exactly at its 1107-line baseline, so the identity/
authority projection + clone helpers moved to `protocol_v2_identity_runtime.go` and
the baseline was **lowered to 1050** (never raised). `Support/Extensions` stays at
its 148-file flat cap (no new file was added there).

## Evidence

```text
cd apps/api && go build ./...                                    → exit 0
cd apps/api && gofmt -l app cmd config sdk                        → 4 known files only
cd apps/api && go test ./app/Support/PluginSessionState/ -race     → ok
cd apps/api && go test ./app/Support/HostAPI/ -run Session         → ok (11 tests, 16 subtests)
cd apps/api && go test ./... -count=1        → 127 packages ok, 1 FAIL:
        app/Support/Pages TestIslandAttributesAllowDeclaredPropsOnly (the concurrent
        workflow's in-flight table disagreement; reproduced WITHOUT this session's
        changes — see "Attribution" below)
cd apps/web && bun run typecheck             → exit 0
cd apps/web && bun test                      → 1213 pass / 0 fail
node tests/validate-architecture-boundaries.mjs  → passed (1803 files)
node tests/validate-docs.mjs                     → all checks passed
node tests/validate-public-page-surface-catalog.mjs → OK (36 pages / 42 host islands)
node tests/validate-v3-production-trust.mjs      → passed
node scripts/v3-catalog/generate.mjs --check     → current (18 files)
ruby scripts/validate-openapi-refs.rb            → OK (2797 refs)
cd apps/api && go test ./app/Support/Extensions/ -run TestPluginSessionStateJoinedHostDispatchToStore → PASS
cd apps/api && go test ./bootstrap/ -run PluginSessionState → ok (3 tests)
bash scripts/ci/reference-packages-gate.sh       → 10/10 OK
bash scripts/proto.sh lint / generate            → exit 0
```

`bash scripts/proto.sh check` still reports the two **new, untracked** generated
files (`host/v2/session.pb.go`, `host/v2/session_grpc.pb.go`) as "out of date",
because the script judges with `git status --porcelain`. Real drift is zero
(two consecutive `generate` runs produce the same aggregate hash).

## Corrections recorded that round

1. **The stale catalog drift was the round's own, and it was fixed** (source
   line numbers only; 363 routes unchanged).
2. **`app/Http/Controllers/Pages` was red mid-round and is green now** (reverted
   to reproduce; not the round's regression).
3. **S5's anonymous row is not satisfiable today** — implemented as fail-closed
   and recorded in Revision 3 §3.4.
4. **Fiber's stored absolute-expiry is wiped by the login `Reset()`**; the scope
   anchors the bound at `created_at + AbsoluteTimeout`.

## Still open (closed on 2026-10-17, see the closeout handoff)

- T5 M9 reference package → `sforum-session-checkout-reference` landed.
- Anonymous session state → decision D1: unavailable by construction.
- M10 row 5 → closed with per-clause evidence.
- M10 row 7 → still operator-owned.
