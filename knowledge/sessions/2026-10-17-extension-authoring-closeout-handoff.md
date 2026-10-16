# 2026-10-17 Session Handoff — Extension Authoring Closeout (M6/M9/M10)

Supersedes `sessions/2026-10-16-m6-s9-session-delegation-handoff.md` as the hot
handoff for the extension-authoring workstream (its content is the audit trail
for how S9 landed; this file records what this round added on top).

Companion to **Revision 4** of
`knowledge/decisions/2026-10-16-extension-authoring-contracts-freeze.md` and to
**Revision 4** of `knowledge/sessions/2026-10-16-m10-joined-release-gate-matrix.md`.

## Outcome in one paragraph

The workstream is **closed** (all ten M10 rows evidenced; the plan moved to
`plans/archive/2026-08/`). Same-day addendum (below, "Row 7 closed"): the
previously open row 7 (true multi-node) was closed with real two-node evidence
on the published `v3.0.9` images — two API nodes sharing one durable
PostgreSQL + Redis, dual registration with live leases, identical revision
acks, cross-node disable/enable converge within the poll window with
byte-identical artifact paths, graceful node-depart/rejoin — plus two measured
cold-start races documented with mitigations in the runbook. Before that: T1
gave the Identity/Extensions erasure hooks their own tests (real PostgreSQL on
the isolated cluster + real lifecycle), T2 built the M9 session-state workflow
reference package (`sforum-session-checkout-reference`, reference gate 10 →
**11** green), T3 rewrote the joined end-to-end test from a stub gRPC plugin to
the **compiled fixture binary over the production go-plugin transport**, T4
delivered the `privacy:erase-user` CLI (decision D2) and measured it live end
to end, T5 closed M10 row 5 clause-by-clause with the anonymous clause closed
by decision **D1** (unavailable by construction), and T6 closed row 9's two
residuals on the real registry tag and the **production Caddy reverse-proxy/TLS
layer**. Two production defects were found and fixed mid-round (see
Corrections).

## Row 7 closed (same-day addendum, matrix Revision 5)

Two real API nodes (independent containers, independent `hostname`/`boot_id`)
from the **published** `ghcr.io/zhuchunshu/sforum-api:v3.0.9` images shared one
disposable PostgreSQL + Redis (`sforum-multinode` compose project; torn down
afterwards). Durable machinery verified live (`plugin_runtime_nodes` boot
leases 45s/10s heartbeat, immutable `plugin_runtime_publications` revisions,
per-node per-revision `plugin_runtime_publication_acks`, 5s poll + LISTEN
convergence):

```text
P1 双节点注册   : node-a/api + node-b/api, distinct boot ids, lease_live=t
P2 同修订双 ack : revision 1 applied by both (17:35:20 / 17:37:42)
P3 跨节点禁用   : disable sforum.web-push on node A → revision 2 → both ack applied;
                  node B's web-push subprocess gone (ps verified)
P4 跨节点启用   : revision 3 → node B revives from the identical immutable artifact
                  path (…/1.0.2/a982b361…/backend/plugin) on both nodes
P5 节点下线     : stop api-b → lease expires ≤45s; node A health 200 throughout
P6 节点重入     : fresh boot_id acks the latest revision within seconds; runtimes revived
```

Two cold-start races were **measured and mitigated in the runbook** (not
coordinator defects; the documented deploy path never hits them):
simultaneous first-boot migrations deadlock (goose session advisory lock +
`CREATE INDEX CONCURRENTLY` mutual wait), and the builtin-sync/legacy-mail
adoption ordering race. Mitigations: sequential node boots (migrate first, or
one node to `health 200` before the second; `MIGRATE_ON_STARTUP=false` on
subsequent nodes), never omit `BUILTIN_EXTENSION_ROOT`.

## Changed (all green)

### T1 — erasure-hook tests (the matrix's per-clause demand)

- `app/Models/Identity/plugin_session_state_postgres_integration_test.go` (new,
  real PostgreSQL via `scripts/test-api-isolated.sh`): `RevokeSession` erases
  exactly the ended sid; `RevokeOtherSessions` keeps the current sid's state;
  `RevokeUserSessions` erases all; `EnforceMaxSessions` erases exactly the
  evicted pair; `ConfirmPasswordResetAtomic` erases all; idempotent re-revoke of
  an already-revoked-but-existing sid still erases; a missing sid erases
  nothing; erasure **error and panic** never fail the revoke; no invalidator →
  revoke works; other users are untouched. 10 tests, all PASS on the isolated
  cluster.
- `app/Models/Extensions/plugin_session_state_lifecycle_test.go` (new): real
  `Disable` erases exactly the disabled extension id; `Uninstall` erases it
  again; erasure error/panic never fails the lifecycle; no invalidator →
  lifecycle works. 4 tests.

### T2 — the M9 session-state workflow reference package (last M9 item)

`extensions/fixtures/plugins/sforum-session-checkout-reference/`: real Protocol
V2 backend (hashicorp-go-plugin), two declared routes over
`Host.SessionState(ctx)` (`state` GET reads the checkout step, `advance` POST
writes step+1), request/response schemas + OpenAPI fragments, README. Honest
notes kept: `delegationEcho` is an integration-test surface (echoes the opaque
delegation so the Host can assert single-invocation semantics after settle; the
token carries no session identity, S8). `bash scripts/ci/reference-packages-gate.sh`
→ **11** packages, all pass. CLI: `digest --write` / `validate` OK /
`test` 0 errors. Docs: the fixture README (bilingual) plus the EN
authoring-guide Reference 5 table.

### T3 — joined test now drives the real plugin binary

`app/Support/Extensions/plugin_session_state_joined_test.go` **replaced** its
stub gRPC plugin with the compiled fixture: stage → build → digest-align →
`ProtocolStarter`/`Manager` start (exact production shape: the **Gateway is the
HostAPIRegistrar** the starter type-asserts for the S9 issuer) → mint → plugin
echo over real gRPC → real service → real store → settle → retire → **replay
denied** with the very token the plugin echoed → cross-session invisibility →
host-side erasure makes the same workflow restart at step 1. The old
stub-based test is superseded (correction recorded; its authority-level
assertions all transferred).

### T4 — `privacy:erase-user` (decision D2's entry)

- `bootstrap/privacy_registry.go` (new): `NewPrivacyRegistry(pool)` — the one
  constructor for the production privacy registry (real RBAC gate + PostgreSQL
  audit), now shared by the API assembly and the CLI; plus
  `NewPrivacyRegistryForOutband(...)` which registers the Host-owned
  plugin-session-state contribution through the **same** registration path as
  the API assembly and **fails closed with a named error when Redis is
  unreachable** (measured live: `NOAUTH Authentication required`).
- `cmd/sforum/privacy_erase_user.go` (new): `--actor-user-id` must be a real
  super_admin/`user.manage` account (audit-recorded); `--user-id`/`--user-email`
  pick the target; report carries extension ids, success, and external-retention
  warnings only. Registered at the root command; usage-contract tests added;
  bilingual CLI docs updated (`validate-docs` green).
- **Live evidence** (disposable migrated+seeded database + real Redis DB 14):
  planted a plugin-session key for a probe session → the run erased it (`GET` →
  empty), printed `core.host.plugin-session-state erased=true`, wrote
  `erase|user:1|50|ok` to `privacy_operation_audit`; a non-super_admin actor was
  denied live without running hooks; the dev database was never touched.
  Also measured live on the dev DB: missing-target path (`user not found`).

### T5 — M10 row 5 closed (matrix Revision 4)

Clause-by-clause evidence table in the matrix; anonymous clause closed by
decision **D1** as unavailable-by-construction (fail closed), explicitly not
satisfied behaviour. Suite citations re-run green: `PluginSessionState -race`
(19 tests), `AuthSession`, `HostAPI -run Session`, plus the new T1 suites.

### T6 — row 9 residuals closed

- **Real registry tag**: `SFORUM_VERSION=v3.0.9` pulled from ghcr with
  `pull_policy: always`, clean `--env-file` (dev `.env` never read) — health 200
  `environment:production`, web `/` 200 (82,148-byte SSR), `/c/general` 200,
  plugin backends running from immutable digest paths.
- **Reverse proxy/TLS**: production `deploy/caddy/Caddyfile` (host-network
  Caddy 2.10.2) in front of `compose.yaml+compose.prod.yaml` pinning the same
  `v3.0.9` images: HTTP→HTTPS 308; HTTPS `/` 200 SSR with the H4 headers
  (HSTS/nosniff/DENY); HTTPS `/api/v1/health` 200; a TLS WebSocket Upgrade
  reaches the Go API (401 authored by the API, not the proxy); unauth SSE also
  reaches the API. Recorded fact: access must use the `APP_DOMAIN` (SNI); bare
  IP fails the TLS handshake as expected.
- Both smoke stacks and volumes torn down cleanly (`down -v` with the same
  `-f`/env, per the runbook).

### T7 — row 7 closed (matrix Revision 5, live two-node evidence above)

## Decisions

- **D1**: anonymous plugin session state stays unavailable by construction; no
  anonymous session persistence will be built without a new ADR (ADR Rev 4.1).
- **D2**: the privacy-erase entry is the super_admin-gated CLI
  `privacy:erase-user`; an admin UI is a separate, undecided question
  (ADR Rev 4.2).

## Corrections (推翻先前结论)

1. **The S9 mint was not production-wired.** Revision 3 and the previous handoff
   said the Host mints the delegation on HTTP-initiated plugin invocations.
   Tests injected the delegation authority into the client config directly;
   production passes the bare `*hostapi.Gateway` to `ProtocolStarter`, and the
   issuer is resolved by type assertion — which the Gateway **did not
   implement** (verified by a compile-failing type assertion). Every scoped
   Host-initiated invocation failed at mint. **Fixed**: the Gateway implements
   `Mint/RetireProtocolV2SessionDelegation`, delegating to the bound session
   service's authority (unbound = named fail-closed error; Safe Mode does not
   block minting — the boundary stays on the three operations, so plugins
   degrade per-call instead of the whole invocation failing). Pinned by
   `TestGatewaySessionDelegationIssuerFailsClosedBeforeBinding`,
   `TestGatewaySessionDelegationIssuerRoundTripAfterBinding`, and the
   production-shape subprocess joined test. Lesson recorded in the ADR: verify
   a "served" capability through the assembly path, not only through tests that
   wire pieces directly.
2. **The first privacy-erase CLI would have reported success without erasing
   plugin session state** (fresh registry, no contribution). Fixed by
   `NewPrivacyRegistryForOutband` before anything shipped.
3. The joined test's plugin side is no longer a stub — the previous handoff
   described it honestly ("stub gRPC server, not a compiled plugin binary");
   that limitation is now gone.

## Next

- This workstream is **closed and archived** (`plans/archive/2026-08/
  2026-08-22-extension-authoring-speed-and-controlled-power.md`). Nothing
  further is scheduled for it.
- Separate workstreams (out of scope here): Built-in Shortcodes M10B release
  gate, V3 production rewire, P13/APILTS residuals, AI Assist remaining items,
  GitHub login re-review, sticker platform, navigation platform M7,
  content-addressed residuals.

## Open Questions

- Whether the two cold-start races deserve hardening beyond the runbook
  mitigation (fail-fast instead of mutual wait) — recorded for the V3
  production-rewire ledger rather than this archived plan.
- Whether the privacy-erase admin UI (beyond the CLI) is wanted (D2 deferred it).
