# Trusted Plugin And Theme Platform V3 Progress Ledger

Date: 2026-07-18
Overall progress: **66.0%**
Active phase: **P7/P9 accepted work plus P10-P12 production closure slices**

This ledger is the durable percentage and context-compaction checkpoint for the
V3 program. Update it before context compression, at every phase boundary, and
after any commit that materially changes the completion calculation.

## Percentage Model

Progress is weighted by implementation risk and verified phase exits. A phase
may contribute a fraction of its weight while work is in progress, but docs,
scaffolding, or demo-only code cannot satisfy a runtime exit criterion.

| Phase | Weight | Current completion | Earned |
| --- | ---: | ---: | ---: |
| P0 Governance | 3% | 100% | 3% |
| P1 Trust/recovery | 6% | 100% | 6% |
| P2 Manifest/contracts | 7% | 100% | 7% |
| P3 Host API v2 | 8% | 100% | 8% |
| P4 Lifecycle/dependencies | 7% | 100% | 7% |
| P5 Database/commands | 8% | 100% | 8% |
| P6 Routes/middleware | 10% | 100% | 10.00% |
| P7 Workflow/admin/query/identity | 10% | 73% | 7.27% |
| P8 Theme compiler/runtime | 8% | 100% | 8% |
| P9 Components/assets/L2 | 8% | 25% | 2.00% |
| P10 Content/media/data | 8% | 0% | 0% |
| P11 Platform services | 6% | 6% | 0.38% |
| P12 Operations/ecosystem | 6% | 5% | 0.27% |
| P13 References/removal/final gates | 5% | 0% | 0% |

Displayed overall progress is the floor of earned weighted progress until the
program reaches 100% and every final gate passes.

## Completed P0 Evidence

- Read all required authority, module notes, and latest handoffs.
- Confirmed clean `main` baseline at `d72c9ac2c` before V3 edits.
- Generated the exact 27-theme + 72-plugin traceability matrix.
- Inventoried 204 API registrations and 113 Nuxt page/component surfaces with
  stable IDs and contract versions.
- Inventoried events, contribution points, provider slots, schedules, job kinds,
  cache, content/data, and 33 admin pages.
- Published the eleven-family Extension Surface Matrix for current core modules.
- Froze namespaces, migration flags, raw DB/custom guard policy, and rollback
  rules.
- Added reproducible v1 enable/theme-resolve/route-proxy/plugin-RPC benchmarks.
- Linked superseded historical decisions to the active V3 target.
- Passed catalog drift, focused Go tests/benchmarks, OpenAPI validation, Nuxt
  typecheck/build, and the full repository gate.
- Closed every P0 task and verification checkbox without changing production
  behavior.

## Compression Checkpoint Protocol

Before expected context compression:

1. Update this file's percentage, evidence, last commit, dirty files, and next
   command.
2. Update the active session handoff under `knowledge/sessions/`.
3. Run the focused tests for the current slice.
4. Inspect `git diff` and `git diff --cached`; stage only V3-owned hunks.
5. Commit every coherent buildable slice. If a slice cannot be committed,
   record the exact dirty files and why.
6. Resume from the recorded next command, not from conversation recollection.

Context usage must be monitored throughout the long-running goal. Do not wait
for an automatic compression warning when a coherent slice can be checkpointed:
write the current evidence and exact resume point here, update the session
handoff, run the focused gate, and commit the checkpoint first. Every progress
update to the user must include the displayed overall percentage and active
phase percentage.

## Execution Acceleration Policy

Until V3 P0-P13 and every final gate are complete, the user explicitly requires
the primary agent to keep useful Codex sub-agent slots occupied and to delegate
additional bounded work to Codex CLI and Grok Build whenever tasks can proceed
independently. Token cost is not a constraint. Codex CLI uses `gpt-5.6-sol`
with `high` or stronger reasoning; Grok Build uses `grok-4.5` unless a smaller
task has a justified faster model. Every delegated task must have an exact goal,
file scope, constraints, and verification commands; delegates must not stage or
commit, and the primary agent must inspect their diffs and tests before accepting
work. This policy remains active across context compression and should be removed
or treated as expired only after the complete V3 goal is achieved.

## Completed P1 Evidence

- Added additive trust-recovery persistence, including one-use challenge
  digests, durable exact-artifact grants, revocation, and startup-attempt state.
- Added a default-off exact-artifact trust service. Challenge tokens are stored
  as SHA-256 hashes, returned in plaintext once, bound to the actor and complete
  impact document, and expire after at most five minutes.
- Exact identity covers package, backend, admin frontend, and migration
  digests. A granted unchanged digest survives restart without another prompt.
- Added super-admin trust inspection, challenge, and revoke HTTP contracts;
  delegated extension managers may store inert packages but cannot authorize
  execution.
- Added Host-owned `SFORUM_SAFE_MODE=1` before plugin startup. It blocks plugin
  lifecycle/settings mutations, routes, contributions, navigation, providers,
  Host API capabilities, trusted frontend assets, and non-default theme
  resolution while keeping health and recovery available.
- Added PostgreSQL-only `sforum extension list`, `disable`, and `disable-all`
  recovery commands. They do not load packages, boot HTTP/Nuxt, or initialize
  plugin runtime code.
- Added startup-attempt containment: a failed, incomplete `starting`, or
  `skipped` attempt for the same digest is skipped on the next boot; a manual
  enable or changed digest may retry.
- Unified backend and admin-frontend execution under the whole-artifact grant;
  legacy frontend-only grants cannot bypass V3 exact-artifact checks.
- Added a shared admin impact dialog with every canonical impact category,
  persistent blocking errors, delegated preview-only behavior, a two-step
  challenge/enable flow, and a theme-aware 10-second success Toast.
- Covered package/backend bytes, migration bytes/declarations, admin frontend
  bytes/contracts, routes, permissions, features, authority, Host/Frontend
  contracts, and dependencies in the trust invalidation matrix.
- Covered wrong actor, missing, expired, replayed, and stale challenges through
  the HTTP boundary, plus trust audit and delegated static-preview behavior.
- Verified PostgreSQL challenge concurrency produces one grant and one replay;
  isolated Safe Mode boot left the executable-plugin sentinel absent; malformed
  package recovery disabled the extension; two isolated boots executed a
  failing plugin exactly once (`failed`, then `skipped`).

## Last Durable Checkpoint

### 2026-07-18 Reference Query Plugin Joined Gates (no P7 credit)

- Exact weighted progress remains `66.9205%`; floored display progress remains
  **66.0%**. P7 stays **16/22**. Query task and joined Query test row uncredited.
- `9b94a088a` HostFeatures offer `query.runtime@1` for executable queries/filters;
  `3763aaf70` reduced query.runtime context; `7427b35b5` composite Registry+Core
  Schema validator in bootstrap; `620daf681` export
  `BuildLifecycleQueryPublication`; `ee4cd412b` real-subprocess reference plugin
  gates (pagination/filter/login/cost/Schema/provider-fail/disable/Safe Mode).
  Redis cache still blocked; production lifecycle coordinator + upgrade gate
  still required before P7 credit.
- Focused Extensions package + QueryRegistry (skip blocked redis tests) passed.
- Next: production lifecycle/bootstrap joined path, resumable Redis invalidation,
  upgrade/replace artifact proof; do not advance score for this reference slice.

### 2026-07-18 Protocol V2 Query Execution Wiring (no P7 credit)

- Exact weighted progress remains `66.9205%`; floored display progress remains
  **66.0%**. P7 stays **16/22**. Query task and joined Query test row uncredited.
- `92f30c76f` Host Protocol V2 InvokeQuery/FilterQueryResult client; `225313dc1`
  composable provider resolvers + ResultFilterSource; `c98128925` Core-then-V2
  composite provider + Registry filter source wired in production bootstrap.
  Lifecycle metadata (`b77271613`) already publishes Handler/Identity/DefaultSort
  and queryResultFilters. Redis cache candidate still blocked and uncommitted.
- Focused Extensions/QueryRegistry/bootstrap normal+race+vet passed.
- Superseded for next-step detail by the Reference Query Plugin Joined Gates
  checkpoint above.

### 2026-07-18 P7 Host-Owned Role Mapping Joined Closure

- Exact weighted progress is `66.9205%`; floored display progress remains
  **66.0%**. P7 advances from **15/22** to **16/22** after the Host-owned
  permission-assignment task passed one joined production-path proof.
- `d07129dd5` drives an exact Manifest V3 lifecycle plugin through the real
  lifecycle Registry boundary and PostgreSQL Identity store. Publication creates
  only one pending `operator` recommendation, persists exact owner/declaration/
  catalog/root evidence, and restores the same graph after a fresh Host restart
  without adding a role mapping or grant.
- The real Fiber boundary rejects the operator, bearer-only, and mixed bearer +
  cookie attempts. A cookie-authenticated `super_admin` decision then adds one
  mapping, immutable grant, exact audit event, and no duplicate evidence on
  replay while preserving the operator's existing permission.
- Focused PostgreSQL normal/race, complete Identity controller/model/registry
  normal/race, complete Extensions normal, build, vet, formatting, diff review,
  and an independent `NO BLOCKER` review pass. Query and executable Identity/
  Auth/Profile production chains remain open and receive no credit here.

### 2026-07-18 P7 Execution Policy Matrix Closure

- Exact weighted progress is `66.4659%`; floored display progress remains
  **66.0%**. P7 advances from **14/22** to **15/22** after its complete
  priority/timeout/failure-policy/version/dependency/provider-fallback test row
  passed production-path evidence and named joined gates.
- `e29394694`, `ec6698136`, and `68baa6bcd` freeze synchronous fail-open
  isolation, exact provider timeout lifecycle ownership, and the real Manifest
  timeout applied to versioned listener contexts. `1f74bdbe1` adds stable
  ExtensionManifest/Extensions/Protocol V2/HostAPI joined gates.
- Focused normal/race, joined normal/race, full related-package normal/race,
  vet, formatting, staged-diff review, and independent credit reviews pass.
- The Host-owned role-suggestion UI/backend checkpoint below was provisional.
  Strict audit in `1c9f7bbf8` withdrew that row's credit because it lacks a
  lifecycle-publication-to-HTTP-apply joined proof; it remains the next P7 task
  and must not be counted a second time.

### 2026-07-18 P7 Host-Owned Role Suggestion Review

- Exact weighted progress is `64.7992%`; displayed progress is **64.7%**. P7
  advances from **14/22** to **15/22** after closing the Host-owned permission
  assignment row.
- `4adcba492 feat(identity): add host-owned role suggestion review` connects the
  existing exact-artifact Identity Registry approval/audit path to the admin
  roles screen. Install and enable remain preview-only; only an authenticated
  Host decision with the exact expected revision can approve/reject and apply
  the one declared permission-to-role mapping.
- Contradictory or incomplete response evidence never produces a success state.
  Stale artifact, revision conflict, missing target, denied access, concurrent
  decisions, list-generation races, and unmapped protocol reasons are handled
  without retrying an obsolete decision or exposing raw internal reasons.
  Approving a mapping refreshes Host data while preserving unrelated unsaved
  role edits and merges the exact approved permission into a dirty draft so a
  later save cannot revoke it accidentally.
- Both role-review and existing template selects use non-empty UI sentinels,
  avoiding the Reka UI runtime 500 while preserving the API's omitted all-filter
  and empty template semantics. Focused Bun tests passed **14/14** with 69
  assertions, Nuxt typecheck passed, Identity controller/model/registry normal
  tests and focused race passed, and isolated clean-HEAD plus staged-only Web
  tests/typecheck passed.
- Authenticated Chrome evidence covered the real roles page, five roles, one
  pending suggestion, the all-state filter, no-template selection, an eight-
  second stability window, no error overlay, and zero new console warnings or
  errors. Browser mutation was deliberately not performed against operator data;
  exact CAS allowed/denied/conflict/evidence behavior is covered at the API and
  composable boundaries.

### 2026-07-18 Core Execution Observation Fence

- Exact weighted progress remains `64.7992%`; displayed progress remains
  **64.7%**. `c685a875c fix(routes): fence observed core execution` closes an
  unsafe retry defect inside the already-credited P6 route/replay rows.
- Direct Core alias/rewrite execution and `readonly_core` fallback now share a
  pre-delivery cancellation check plus monotonic side-effect/response evidence.
  An unused canceled request may abort; once Core delivery is possible, error,
  500, or cancellation leaves required replay pending instead of authorizing a
  second Core writer.
- POST alias/rewrite success, replay, error, 500, cancellation-before-delivery,
  and cancellation-after-delivery are covered. Focused tests passed 50 normal
  and 10 race repetitions; an isolated exact-index clone passed full Routes
  normal/race, vet, and `go build ./...`.
- Next: detach completion of an already-valid response from caller cancellation
  without hiding runtime incidents, then finish the Stream V2 deadline, lease,
  schema, incident-source, and real-process correlation matrix.

### 2026-07-18 Required Replay Final-Response Closure

- Exact weighted progress remains `64.3447%`; displayed progress remains
  **64.3%**. This closes replay correctness defects inside already-credited
  required idempotency behavior and does not independently close one of the
  three remaining P6 rows.
- `036cfc4c8 fix(routes): revalidate required replay responses` reapplies the
  current plugin-response header policy to historical records, removes Host
  replay evidence from guard/Schema inputs, rejects invalid stored terminal
  status, and runs first execution plus replay through a final response-contract
  boundary before finalization or persistence.
- `60d16ae88 fix(routes): persist effective replay response contract` stores the
  exact handler/response contract that actually governed the completed payload.
  A later modifier whose input Schema rejected the prior response is no longer
  mistaken for the effective contract on replay. New encrypted payloads use a
  versioned AAD domain; existing V1 payloads remain readable, while older
  binaries fail closed on the new payload version.
- A schema-less unsafe `after` mutation that violates the effective contract
  preserves the latest contract-valid response and records the exact committed
  failure. Safe Core fallback is not validated against a plugin handler contract
  that never produced it. Forged, drifted, malformed, or unknown provenance
  fails before output or a second plugin invocation.
- Full Idempotency, Routes, and Http tests passed with focused replay race and
  three-package vet. Both commits also passed isolated clean-HEAD plus exact
  staged-patch normal/race/vet gates. Unsafe Core execution observation,
  post-response caller cancellation, and the stream lifetime/lease/schema matrix
  remain separate P6 work.

### 2026-07-17 P6 Exact Stream Execution Evidence

- Exact weighted progress remains `64.3447%`; displayed progress remains
  **64.3%**. This hardens already-credited streamed transport behavior and does
  not independently close one of the three remaining P6 rows.
- `78ecad557 fix(routes): preserve stream execution evidence` restricts the
  non-buffered Dispatcher to the immutable terminal and exact plugin `add` or
  `replace` handler, rejects composed/drifted stream plans, classifies remote
  custom/raw guard failures correctly, and applies mode-exact terminal status
  validation. Caller cancellation before remote execution remains payload-free
  and pristine; cancellation or failure after observed execution remains a
  transport failure with `side_effect_started` evidence.
- The Protocol V2 adapter now requires a commit observer, checks the acquired
  lease context immediately before the remote preflight, and advances the
  execution fence before a preflight whose delivery cannot be disproved. This
  dependency landed atomically with removal of the old premature unconditional
  `RequestStarted` mark.
- A clean `git archive HEAD` plus only the four-file staged patch passed the
  complete Routes and Http normal/race suites and both-package vet. The focused
  caller-cancellation matrix passed 50 normal and 10 race repetitions before
  commit; the committed diff passed `git show --check`.
- Stream route-timeout enforcement across the unary preflight, Host validation
  semantics for required non-HTTP response schemas, and real SDK correlation
  evidence remain part of the final P6 behavior-matrix exit. Required replay
  response revalidation is under a separate isolated review. P7 role-suggestion
  UI and P9 public frontend policy drafts remain outside this commit.

### 2026-07-17 P6 Bidirectional Staged Modifier Closure

- Exact weighted progress is `64.3447%`; displayed progress is **64.3%**.
  P6 advances from **13/18** to **15/18** after closing the complete accepted
  route-action family and request/response/filter schema plus explicit mutable-
  field rows.
- `5da58f160` executes global/before/filter/wrap request stages in deterministic
  priority order, the selected Core or plugin handler, and wrap/filter/after
  response stages in reverse composition order. Protocol V2 carries exact
  action/stage, ordered repeated query values, bounded RFC 6901 patches, typed
  guard failures, and Host-proven params authority; Protocol V1 modifiers fail
  before runtime admission.
- `d55f027a6` proves a request patch is immediately revalidated against the
  same exact route schema. A rejected patched body returns 422, skips every
  later modifier and Core, drains the exact runtime lease, emits one redacted
  request-stage `schema_rejected` trace, and does not misuse the response-stage
  committed-after failure contract.
- The exact implementation index passed full Routes, Http, Extensions, and
  bootstrap tests; Routes/Http race tests; four-package vet; `go build ./...`;
  real subprocess repeated-query verification; and the production Dispatcher
  benchmark. The schema revalidation proof additionally passed 50 focused
  repetitions, 10 race repetitions, full Http, and vet.
- Required replay with request mutation remains fail-closed before `Begin`.
  The next production slice must freeze exact required-idempotency policy into
  the immutable Route snapshot and execution plan, prove 64-reader publication
  atomicity, then switch HTTP to Bound replay with wrong-key evidence. Custom/
  raw guard revoke-WebSocket, route SEO, and the complete P6 matrix remain open.

### 2026-07-17 P6 Trust Revocation And Guard Closure In Progress

- Exact weighted progress remains `63.2336%`; displayed progress remains
  **63.2%**, and P6 remains **13/18**. Durable/local trust revocation,
  process-ahead convergence, four-credential filtering, lifecycle fixture
  isolation, omitted target-guard inheritance, and SEO cancellation fixes are
  implemented or under final integration verification, but no authoritative
  P6 row is credited early.
- Full Models normal/race, full Support and Http, focused real-PostgreSQL
  unknown-COMMIT/lifecycle-state tests, Manifest normal/race, focused
  coordinator/credential/revoke race tests, vet, repetition, and diff checks
  pass. The exact resume point is the real TCP WebSocket revoke/R+2 proof,
  deterministic advisory-lock waiter proof, sequential package/bootstrap
  gates, scoped diff review, and fine-grained commits.
- The exact dirty-file grouping, rollback state, user-owned exclusions, and
  commit sequence are recorded in
  `knowledge/sessions/2026-07-13-trusted-plugin-theme-platform-v3-progress.md`.

### 2026-07-17 P6 Exact Authority, Replay, And Failure Evidence

- Exact weighted progress remains `63.2336%`; displayed progress remains
  **63.2%**. P6 remains **13/18** because this closes critical production
  boundaries inside open rows but does not yet prove the full route-action,
  RFC 6901 mutation, revoke/WebSocket, canonical SEO, or complete behavior
  matrix exits.
- `c7cf50c97` carries Host-issued filtered/raw authority through HTTP and
  Protocol V2 unary/guard/stream, binds it to the exact plan/step/request/prior
  response/artifact, forwards credentials only for exact `raw_request`, and
  rejects forged direct invocations.
- `b3a521e05` makes remote execution evidence monotonic across `Finalize` and
  prevents required-idempotency Abort after an observed call, transport crash,
  or response-schema rejection.
- `70dd7fb7c` preserves the prior response when an unsafe plugin `after`
  contribution fails, stops later modifiers, emits a stable payload-free event,
  and completes deterministic 2xx replay.
- `365cd0df6` adds exact version/digest/instance quarantine without waiting for
  runtime-set/lifecycle locks. Existing calls finish, cleanup remains available,
  ordinary admission closes, and Resume/full-set rollback cannot reopen the
  exact gate.
- `a645ac594` production-wires a bounded audit recorder and stable
  `routes.committed_after_failure` evidence. Guard/request-schema failures and
  unobserved Host transport failures do not quarantine; observed transport and
  response-schema failures do. Queue pressure cannot skip quarantine, and
  shutdown is bounded before runtime/PostgreSQL close.
- Full Routes, Http, Audit, Extensions, and bootstrap tests passed, along with
  focused race suites, vet, formatting, and staged-diff checks. The exact
  quarantine state is intentionally process/node-local; durable cross-node or
  restart persistence remains an explicit open contract.

Exact resume command:

```bash
cd apps/api && go test ./app/Support/Routes ./app/Http ./app/Support/Extensions ./bootstrap
```

Then finish Identity startup adoption, strict WebSocket/revoke closure, RFC 6901
route mutation/action semantics, canonical redirect SEO, and the full P6 matrix.

### 2026-07-17 P12 Migration Publication Proof Fence

- Exact weighted progress remains `63.2336%`; displayed progress remains
  **63.2%**. P12 stays **1/22** because this safety fix does not yet prove the
  full multi-node install/upgrade/rollback row across every producer.
- `fea430020` makes install, upgrade, and rollback runtime publication load and
  lock the current operation's canonical exact-plan `target_ready` proof in the
  same transaction before desired-runtime publication and marker binding.
  Missing, failed, malformed, or drifted proof evidence fails closed with the
  retryable lifecycle migration error; enable/deactivate remain compatible.
- Real PostgreSQL tests prove eight-way migration-once, a proof-row lock overlap
  with eight committers, atomic source/target revision behavior, failed SQL,
  plan drift, and new-pool replay. The whole Extensions package normal/race,
  vet, Models/bootstrap tests, `go build ./...`, diff checks, Codex review, and
  a bounded read-only `grok-4.5` review passed.

### 2026-07-17 P11 Cache SDK Closure

- Exact weighted progress is `63.2336%`; displayed progress is **63.2%**.
  P11 earns its first verified row (**1/16**) while P6 remains **13/18**, P7
  **14/22**, P8 **18/18**, P9 **4/16**, and P12 **1/22**.
- `ba4ebc50c` adds typed namespaced Cache SDK operations, opaque CAS revisions,
  tag invalidation, cross-RPC lease helpers, and a distributed `remember` path
  that double-checks after acquisition, renews while loading, cancels at exact
  lease expiry, and commits through atomic set-and-release.
- Focused SDK and Host Cache normal/race tests, SDK vet, formatting, staged diff
  checks, and an independent `grok-4.5` read-only audit passed. The external
  audit's final result reported no blocker; its intermediate speculation was
  not accepted without matching local code and test evidence.
- This closes only P11's first Cache task. Cache provider policy/filters,
  inspector metrics/revision awareness, the full failure test row, and every
  other P11 service remain open.

### 2026-07-16 P12 Theme And Plugin Runtime Ownership Closure

- Exact weighted progress is `62.8586%`; displayed progress is **62.9%**.
  P6 remains **13/18**, P7 **14/22**, P8 **18/18**, P9 **4/16**, and
  P12 now earns its first production row (**1/22**). The previously checked
  broken-system-extension recovery test remains P1 evidence and is not counted
  twice in the weighted ledger.
- `04b159441` creates an exact immutable Theme genesis when an upgraded database
  has a valid active theme but no desired-state publication. It shares the
  activation advisory lock, preserves approval evidence, recovers ambiguous
  commits by exact revision, and rejects invalid mutable state. Real PostgreSQL
  normal/race tests cover 32 concurrent producers and activation races.
- `873e48248` makes Theme node ownership fail closed: heartbeat runs independently
  with a deadline, cancels in-flight apply on ownership uncertainty, catches up
  every revision committed during initial apply before readiness, validates
  applying/applied acknowledgements, and closes process-local theme mutation
  admission before restoring the protected default.
- `d46fd3597` removes double initialization and gives the API process bounded,
  supervised Theme ownership. Theme and plugin terminal failures share one API
  failure source; fallback completes before failure publication, startup rejects
  an already-dead runtime, and HTTP drains before Redis/PostgreSQL close.
- Focused Models/bootstrap/cmd normal and race, vet, `go build ./...`, three real
  PostgreSQL convergence runs, and a real PostgreSQL race run passed. This closes
  only P12 task 1. Staged/canary rollout, migration-once rolling activation, the
  full multi-node upgrade/rollback test row, and all later P12 work remain open.

Dirty ownership at this checkpoint:

- Identity owns migration `202607160033`, IdentityRegistry root/leaf publication,
  Extensions lifecycle wiring, and its PostgreSQL tests. Do not stage it with
  Theme or Cache work.
- Cache owns Protocol V2/Host/SDK work, including hard Remember bounds, opaque
  token redaction/cleanup, hit revisions, and ergonomic helpers.
- `docs/extensions/catalogs/manifest-v3.md` belongs to the pending Identity/SEO
  generated-doc synchronization commit.
- Never stage `apps/api/app/Models/PageViewModels/source_test.go` or
  `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`.

Exact resume command:

```bash
git status --short
```

Then finish Identity root durability and production bootstrap, repair the Cache
SDK review findings, and continue the SEO provider-to-Host-policy vertical.

### 2026-07-16 Query/P12/SEO Production-Slice Checkpoint

- Exact weighted progress remains `62.5859%`; displayed progress remains
  **62.6%**. P6 is **13/18**, P7 **14/22**, P8 **18/18**, and P9 **4/16**.
  P10-P13 remain uncredited because none of their open production rows is yet
  complete end to end.
- `0815d2bce`, `7a8401e47`, and `4127c1c4b` make legacy Service enable/disable
  and out-of-band CLI recovery publish one transaction-scoped immutable plugin
  runtime desired full-set. Exact genesis, actor/reason evidence, lock order,
  malformed-package recovery, concurrent commands, rollback, and uncertain
  COMMIT readback passed real PostgreSQL and race tests. P12 remains open:
  `startPluginRuntimeCoordinator` and the API theme watcher still lack complete
  production ownership/application evidence.
- `1b8a8064e` prevents drained Route providers and modifiers from reappearing in
  selected plans or blocking Core fallback. It does not close the P6 action row:
  real Protocol V2 `filter`, `after`, and `wrap` semantics still require frozen
  mutable-field, nesting, response, failure, redirect, and canonical rules.
- `776b9e089`, `a873e3a59`, `81e8f732d`, and `f83d10b6b` add query actor wire
  delegation, an actor/runtime/query/revision-bound one-use Query Registry Host
  outlet, SDK helpers, and fail-closed reflected-token rejection. Normal/race/
  vet passed for HostAPI, QueryRegistry, SDK, and the broker. P7 Query remains
  uncredited until API bootstrap binds the outlet to live RBAC and exact active
  runtime admission; the frozen Manifest still has no plugin query handler or
  declared result-filter transport.
- `e5df1fcf8`, `f1dfd7efc`, and `1c6dcd10b` add strict SEO Manifest declarations,
  exact trust impact, lifecycle plan `@5`, startup/Safe Mode publication, and
  OpenAPI schemas. SEO Registry, Extensions normal/race/vet, Manifest repetition,
  Models, and all 1,900 OpenAPI references passed. The last commit also contains
  the lifecycle files because a delegate committed the shared staged index;
  content is verified, but the commit message is narrower than its file set.
  P11 SEO remains open without provider transport, Host final policy, Core SEO
  baseline, SSR/sitemap/JavaScript-disabled consumers, Inspector, and reference
  failure evidence.
- The strict P10-P13 audit keeps P10 at 0/15, P11 at 0/16, and P13 fully open.
  The fastest real production exits are Query bootstrap, P12 plugin/theme
  convergence ownership, then the SEO provider-to-SSR vertical. Do not start
  legacy deletion before those dependencies close.

Dirty ownership at this checkpoint:

- Query production wiring may edit focused bootstrap files only and must not
  invent plugin query handler/result-filter/relation semantics.
- `apps/api/app/Models/PageViewModels/source_test.go` and
  `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json` are
  unrelated/user-owned and must never be staged.
- External Grok/Codex CLI repository delegation remains blocked by the managed
  private-repository disclosure policy. Do not retry or bypass it; keep local
  Codex slots on bounded tasks.

Exact resume command:

```bash
git status --short --branch
```

Then review Query production wiring, run bootstrap normal/race/vet, and commit
it independently before starting the P12 convergence `app.go` changes.

### 2026-07-16 P6 Production WebSocket Ingress

- Exact weighted progress is `62.5859%`; displayed progress is **62.6%**. P6 is
  **13/18**, P7 **14/22**, P8 **18/18**, and P9 **4/16**.
- The exact formula is
  `39 + 10*(13/18) + 10*(14/22) + 8 + 8*(4/16) = 62.5859`.
- `1144a78dc`, `e22844ecb`, and `d2704ef38` close the arbitrary-path row and
  restore the production transport evidence: public, admin, and API Registry
  paths reach the Go dispatcher; ordinary HTTP streams through Nuxt; real
  WebSocket Upgrade requests route from Caddy to the loopback Host API while
  preserving Host, Origin, Cookie, Authorization, and Upgrade authority.
- Unknown WebSocket paths fail closed before bearer, actor, or plugin runtime
  admission. `vite-hmr` remains Host-owned and stays on the Nuxt upstream.
- Independent gates passed the complete `app/Http` package, focused race tests,
  Nuxt proxy tests, the production proxy validator, Caddy 2.11.3 validation,
  Compose expansion, and shell syntax checks. `9bbf4fdbc` separately removed a
  flaky order assumption from the Host `Vary` authority regression.
- P6 now has five open rows: complete action semantics, inherited/custom/raw
  authority, explicit mutable fields, alias/redirect SEO integration, and the
  complete route/locale/request/authority/transport/failure matrix.

Dirty ownership remains family-scoped. Never stage the user-owned
`apps/api/app/Models/PageViewModels/source_test.go` or
`extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`.

Exact resume command:

```bash
git status --short
```

Then land Component, Cache, Content, SEO, Media, and Query as independently
reviewed slices before production lifecycle/bootstrap composition.

### 2026-07-16 P6 Production Credit Correction

- Exact weighted progress is `62.0303%`; displayed progress is **62%** and the
  user-facing progress bar may show **62.0%**. P6 is **12/18**, P7 **14/22**,
  P8 **18/18**, and P9 **4/16**.
- The exact formula is
  `39 + 10*(12/18) + 10*(14/22) + 8 + 8*(4/16) = 62.0303`.
- Two previously checked P6 rows were reopened after a production-path audit.
  `app/Http/server.go` mounts `routeDispatcherMiddleware` only on the
  `/api/v1` Fiber group, so Registry declarations for arbitrary public/admin
  root paths cannot reach the production dispatcher. Unit tests that install
  the middleware directly on a test app do not prove the real topology.
- Route request/response schema validation exists, but the route Manifest,
  Registry, Protocol V2 request, and OpenAPI contract have no declaration-bound
  mutable-field policy. `Support/Routes/route_matrix_test.go` explicitly records
  mutable fields as an open semantic boundary.
- Historical Route Inspector credit remains valid. The correction removes only
  the arbitrary-path and explicit-mutable-field rows; it does not rewrite the
  evidence of the commits that originally accepted the Inspector.
- P6 now has six open rows: arbitrary production path mounting, complete action
  semantics, inherited/custom/raw authority, explicit mutable fields,
  alias/redirect SEO/canonical integration, and the complete route matrix.

Dirty ownership remains family-scoped. Never stage the user-owned
`apps/api/app/Models/PageViewModels/source_test.go` or
`extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`.

Exact resume command:

```bash
git status --short
```

Then land the already-running P12 and Query slices independently before shared
Protocol edits, and repair the two reopened P6 rows with production HTTP and
contract tests.

### 2026-07-16 Query/Identity/P12 Candidate Checkpoint

- Exact weighted progress remains `63.1414%`; displayed progress remains
  **63%**. P6 is **14/18**, P7 **14/22**, P8 **18/18**, and P9 **4/16**.
- `78c5564fb` and `eec809599` commit the Identity Manifest safety boundary and
  immutable Identity Registry leaf. They add no P7 credit before lifecycle,
  durable ownership, Host mapping approval, EntityMeta integration, provider
  execution, Inspector, and reference evidence close.
- Query lifecycle publication, the P12 plugin runtime publication migration,
  and the untracked Cache Registry remain under review. A scoped external task
  may create `Support/ContentRegistry`; it remains uncredited declaration code.
- P12 review rejects duplicate source/lifecycle/trust/protocol/migration fields
  but requires monotonic applied revisions, live leases, plugin-only members,
  applied-set digest verification, and expanded PostgreSQL behavior tests.
- P10/P11 audits confirm both phases remain at zero. P13 deletion/reference
  work remains dependency-blocked. `extensions/fixtures` contains tracked source
  fixtures and must not be globally ignored.

Dirty ownership at this checkpoint:

- Query owns its Models/Support/bootstrap lifecycle publication files.
- P12 owns migration `202607160027` and its two tests.
- Cache and Content candidates own only their new Registry directories.
- `apps/api/app/Models/PageViewModels/source_test.go` and the content-policy
  Manifest remain unrelated/user-owned and must never enter V3 commits.

Exact resume command:

```bash
git status --short
```

Then finish Query lifecycle, review P12 constraints, and land each coherent
slice separately before starting shared lifecycle edits for Identity.

### 2026-07-16 P6 Route Inspector Ledger Correction

- Overall advances to **63%** after flooring. P6 advances to **14/18 (78%)**
  and earns **7.78%** of its 10% weight. P7 remains **14/22 (64%)**, P8
  remains complete, and P9 remains **4/16 (25%)**.
- Exact earned weight is `39 + 10*(14/18) + 10*(14/22) + 8 + 8*(4/16)` =
  `63.1414`; the displayed total is the floor, **63%**.
- This is an accounting correction backed by already committed production
  evidence, not credit for a disconnected foundation. The progress history
  accepted the production Inspector in `3b017173c` and `61da559d5`: one shared
  bounded trace ring reports exact chain, provider, guard, contract, fallback,
  timing, outcome, and commit state through the permissioned HTTP endpoint.
  `0c9fc5cbc` subsequently added the complete bilingual admin UI and its typed,
  fail-closed response parser.
- Fresh verification passed `go test ./app/Support/Routes
  ./app/Http/Controllers/Extensions -count=1` with an isolated Go cache,
  `bun test apps/web/tests/adminRouteInspector.test.ts` (14 tests), and all
  1,879 modular OpenAPI references.
- P6 still has four open rows: complete action semantics, inherited plus
  custom/raw guards, alias/redirect SEO/canonical integration, and the full
  route action/locale/request/authority/transport/failure matrix.
- A read-only Grok P12 audit confirmed that P12 remains uncredited. Its first
  production packet should generalize the proven theme desired/node/ack model
  to plugin runtime convergence while reusing, not duplicating, lifecycle CAS
  and migration-once evidence. Wakeup transport and canary semantics remain
  product decisions and are not silently selected here.

Dirty ownership at this checkpoint:

- `apps/api/app/Http/route_dispatcher.go` and
  `apps/api/app/Http/route_request_authority_matrix_test.go` belong to the P6
  request-authority task.
- `apps/api/app/Support/QueryRegistry/` belongs to the P7 production-wiring
  task and remains uncredited until lifecycle/bootstrap/Host execution closes.
- `apps/api/Dockerfile` and `apps/api/cmd/sforum/test_extension_test.go` belong
  to the active extension packaging task.
- `apps/api/app/Models/PageViewModels/source_test.go` and the content-policy
  manifest remain unrelated/user-owned and must not be staged with V3 docs.

Exact resume command:

```bash
git status --short
```

Then review and land each owned code slice separately. Queue Grok P10/P11/P13
audits one at a time because the external endpoint rejected excess concurrent
requests; never treat a rate-limited partial report as implementation evidence.

### 2026-07-16 P7 SDK And P9 Asset Lifecycle Closure

- Overall advances to **62%** after flooring. P7 advances to **14/22 (64%)**
  and earns **6.36%** of its 10% weight. P9 advances to **4/16 (25%)** and
  earns **2.00%** of its 8% weight. P6 remains **13/18 (72%)**.
- Exact earned weight is `39 + 10*(13/18) + 10*(14/22) + 8 + 8*(4/16)` =
  `62.5859`; the displayed total is the floor, **62%**.
- `e92016366` closes the P7 hook/service/provider/job/schedule/command
  SDK/catalog row. Callable families expose typed registries or clients and
  source-derived limits. Schedules are explicitly Host-owned Manifest
  declarations; the unregistered generated wire client is documented instead
  of being wrapped in a nonfunctional List/Trigger helper.
- `2a8c0d3e6`, `d8d6d5205`, and `55063b1a3` establish the immutable bounded
  Asset Registry, deterministic dependency plans, exact artifact/revision CAS,
  quarantine closure, cleanup, and race fences. `cf5636927` binds immutable
  frontend bytes and descriptors to live exact authority without request-time
  Store rebuilds.
- `44cfb67dc` binds plugin/theme lifecycle transitions and compensation to the
  exact Asset graph. `f5ed19d2c` adds durable lifecycle publication plans,
  startup and Safe Mode restore, and one shared Registry late-bound after
  authoritative reconciliation. `86d112ef5` remains the production inert ZIP,
  browser mount, restart, revoke, unmount, CSS cleanup, and SSR fallback proof.
- The converged Asset tree passed Models and Support/Extensions/bootstrap
  normal plus race tests, focused repetitions, vet, `go build ./...`, and diff
  checks. The SDK slice passed normal/race, vet, generated documentation drift,
  and diff checks.
- `4c2911122` strengthens route action ordering, replacement selection, Safe
  Mode, fallback fencing, cancellation, and timeout coverage. It does **not**
  close a P6 row: locale/query/body, permission, CSRF, custom/raw guards,
  crash, redirect SEO, and open composition semantics remain incomplete.
- Page-scoped CSP aggregation into Nuxt SSR headers remains open even though
  Asset declarations validate and return bounded CSP values. Public L2 stays
  production-default off. Navigation/Region production contracts also remain
  open.

Dirty ownership at this checkpoint:

- `apps/api/app/Support/QueryRegistry/` is owned by the active P7 production
  wiring task. It is untracked and receives no progress credit before
  lifecycle/bootstrap/Host execution and production tests land.
- `apps/api/app/Models/PageViewModels/source_test.go` and
  `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`
  remain unrelated/user-owned and must not enter this checkpoint commit.
- The checkpoint owns only `knowledge/index.md`, this progress ledger, the V3
  task book, and
  `knowledge/sessions/2026-07-16-trusted-plugin-theme-platform-v3-asset-checkpoint.md`.

Exact resume command:

```bash
git diff --check -- knowledge/index.md \
  knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3.md \
  knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3-progress.md \
  knowledge/sessions/2026-07-16-trusted-plugin-theme-platform-v3-asset-checkpoint.md
```

Then inspect those four files, commit only their hunks, and continue the P7
Query production path without crediting the disconnected foundation.

### 2026-07-16 P5 Database And Host Command Closure

- Overall advances to **61%** after flooring. P5 reaches **17/17 (100%)** and
  earns its full 8% weight; the five open checkboxes were stale accounting, not
  missing production implementations.
- Host-owned `sforum_core_v1` views, the immutable Host Query catalog, and six
  production Host Commands cover identity, topic visibility, entity metadata,
  moderation, provider-neutral entitlement, and attachment workflows before
  any plugin broker starts in API, embedded worker, and standalone worker.
- Exact per-runtime database lease roles implement the approved additive
  `own_schema`, `core_views`, `host_commands`, `raw_core`, and `kernel` powers.
  Physical `raw_core` ACLs allow disclosed Core DML while denying DDL,
  ownership, role inheritance, River internals, arbitrary functions, PUBLIC
  escape, and foreign-owned objects. Source/target overlap and exact fenced
  revoke preserve rolling upgrades.
- `7c00a7fff` adds the final entitlement Host Command production evidence:
  eight-way same-key replay, changed-payload conflict, expected-revision
  rejection, revoke commit/replay, and actor-delegation rejection for actorless
  service authority.
- Real PostgreSQL Host Command, raw-core, compatibility, entitlement repository,
  migration-once, policy, idempotency, audit, receipt, storage-failure, and
  rollback gates pass in isolated schemas. Risk disclosure passed desktop and
  390px acceptance. See
  `knowledge/sessions/2026-07-16-trusted-plugin-theme-platform-v3-p5-closure.md`.

### 2026-07-16 P9 Buildless Public L2 Production Exit

- Overall remains **58%** after flooring. P9 advances to **12% (2 of 16
  rows)** and earns **1.00%** of its 8% weight.
- `b081898c5`, `d8d6d5205`, and `bf49c2aa9` bind generated L2 entries to exact
  contracts, require exact Asset Registry publications, and publish exact
  Component Registry theme transitions. A manifest declaration alone no longer
  admits stale, hidden, losing, or dependency-invalid browser code.
- `b164b30bd` preserves Host L1 output when an extension render fails, while
  `c0e2fa855` adds the author-prebuilt fixture with no `package.json` or
  package-local build command.
- `86d112ef5` proves the isolated production chain from an inert admin ZIP
  upload through actor/artifact-bound trust and activation, immediate descriptor
  availability without API restart, Nitro SSR fallback, native ESM relative
  chunk and nested CSS loading, browser interaction, API/Nitro restart,
  trust revocation, descriptor denial, DOM unmount, and stylesheet lease
  cleanup. The opt-in production test passed in **238.30 seconds** and left no
  test processes behind.
- Fixture `extension validate` and `extension test`, 29 focused Web emergency
  tests, Nuxt typecheck, and the focused Go public L2 tests passed. Public L2
  remains default-off until Asset lifecycle publication, scoped SSR CSP, and
  the remaining P9 composition/inspection exits close.

### 2026-07-15 P7 Admin Surface Production Closure

- Overall advances to **57%** after flooring. P7 advances to **59% (13 of 22
  rows)** and earns **5.91%** of its 10% weight.
- `3dff78a05`, `00470c41e`, `8ee782c78`, `3bc8ea6e5`, `5c76cbca8`,
  `7f1f34bf0`, `367731bd7`, and `de333a405` close both the production Admin
  Surface Registry task and its independent full-family reference-plugin exit.
- Exact active-artifact publication, placement-specific schemas, permission-
  filtered discovery, Protocol V2 query/command invocation, actor and
  idempotency propagation, durable pre-call audit, lifecycle rollback/restart,
  and list/form/dashboard/editor/detail/import/export consumers are wired.
- Browser acceptance covered editor mutation, export download, authoritative
  ordinary-member denial, desktop and 390px layouts, and hydration. All 320 Web
  tests, focused tests, Nuxt typecheck, and the production build passed.
- `51dc43d59` separately makes the completed P1 exact-artifact trust boundary
  production-default for bare processes and Compose API/worker deployments;
  development remains opt-in and the compatibility override is retained.

### 2026-07-15 P8 Plugin Business-Contract Preservation Checkpoint

- Overall remains **56%** after flooring. P8 advances to **94% (17 of 18
  rows)** and earns **7.56%** of its 8% weight; only production population of
  all page-specific Core ViewModels remains open.
- `cd1573d5a` freezes a versioned, exact-artifact plugin Page ViewModel at
  lifecycle publication and passes that sealed DTO through plugin template,
  theme override, default-theme fallback, and emergency fallback resolution.
- Theme code can change presentation only. It cannot replace the plugin data
  schema or coerce numeric business values; exact loader/runtime admission and
  invalid-payload fallback preserve the prior published artifact.
- Verification passed twenty focused repetitions, all affected Go packages,
  Extensions and targeted ThemeCompiler race gates, compiler allocation
  budgets, vet, build, and the full repository test script before commit.
- Active dirty ownership after the commit is P5 database credential/kernel
  recovery and P7 Admin Surface browser consumption. The content-policy
  manifest and `.playwright*` artifacts remain excluded from V3 staging.

### 2026-07-15 P6 Exact OpenAPI Policy Runtime Checkpoint

- Overall advances to **56%** after flooring. P6 advances to **72% (13 of 18
  rows)** and earns **7.22%** of its 10% weight.
- The Host now derives rate-limit, security, and idempotency facts from each
  validated OpenAPI operation. Unsafe methods use the existing Fiber
  `host.ip_write@1` client-IP limiter; undeclared replay remains disabled; a
  required standard `Idempotency-Key` becomes `required.24h@1` only on bounded
  unsafe HTTP routes. Safe and non-buffered declarations fail static validation,
  with an independent streaming runtime fence against publication drift.
- Required replay is Host-owned and fail-closed. The 24-hour Redis/memory ledger
  scopes a hashed key to actor or anonymous client, cookie/bearer source, exact
  artifact route contract, and method; fingerprints canonical path, sorted query
  values, normalized content type, and raw body; fences completion/abort by CAS;
  and stores only bounded 2xx responses. Replays rerun current Host guards without
  invoking the plugin. Authorization and Cookie data are neither fingerprinted
  nor persisted.
- Production bootstrap consumes the lifecycle-owned policy snapshot and shared
  Redis replay store. Permissioned `extension.view` endpoints expose the exact
  aggregate document and generated-client operation metadata from one immutable
  publication revision with a digest ETag.
- The reviewed Host catalog now contains **227 routes**. Verification passed all
  Go tests, focused HTTP and package tests, race across Idempotency/Routes/HTTP/
  controller/OpenAPI, vet, build, 1,858 OpenAPI references across 47 files,
  Nuxt typecheck, protobuf and documentation drift, and the 227-route/120-UI/
  99-row catalog gate. The final repository script then stopped on the separately
  owned P7 admin registry path `/extensions/route-providers`; P6 did not modify
  that failure.

### 2026-07-15 P8 Production Restart And Concurrent Activation Checkpoint

- Overall remains **55%** after flooring. P8 advances to **89% (16 of 18
  rows)** and earns **7.11%** of its 8% weight.
- `3e771b149` fixes startup synchronization so a valid selected uploaded theme
  survives builtin discovery. Fallback to the default now occurs only when no
  theme is active or the active builtin was removed from the release.
- `TestThemeSwitchSurvivesProductionAPIAndNitroRestartAndConcurrentActivation`
  builds and starts the real API and Nitro applications against a fresh
  PostgreSQL database, switches exact artifacts, restarts both processes,
  races two exact CAS activations, verifies one winner and one stale preview,
  and restarts both processes again to prove the winner persists.
- The self-contained production exit passed in 203.78 seconds. Browser QA
  independently confirmed the winning Nocturne artifact and skin marker,
  absence of a blocking overlay, and working menu navigation. The only console
  diagnostic was the expected canonical-base warning from the isolated port.
- The two remaining P8 rows are complete page-specific production ViewModel
  population and exact plugin business-data contract preservation across theme
  presentation overrides.

### 2026-07-15 P9 Stable Component Identity Checkpoint

- Overall advances to **55%** after flooring. P9 is **6% (1 of 16 rows)**
  and earns **0.50%**; P5 remains **71% (12 of 17 rows)**.
- `a805cbe01` adds the neutral, standard-library-only `ComponentCatalog` leaf
  with 119 generated active Core targets, stable contracts, page/component
  kinds, explicit public/admin ownership, immutable caller copies, and exact
  lookup. A future Component Registry can import both this leaf and Manifest V3
  without creating an import cycle.
- Manifest V3 now binds every nonempty component `targetId` to a required
  `targetContractVersion`. Core targets must exactly match the Host catalog;
  cross-plugin targets are syntactically versioned for later exact Registry
  resolution. Themes cannot target admin-only Core surfaces, and non-component
  `core.*` targets fail closed.
- UI identities use explicit active/retired state. Retired IDs/contracts remain
  reserved in an append-only, immutable-path tombstone ledger checked against
  full reachable Git history and a generated reservation artifact, so deletion
  plus regeneration and unrelated reuse are rejected.
- Verification passed 14-output generator drift, the 225-route/119-UI/99-row
  catalog validator and its collision/owner/source/retirement transitions,
  focused ComponentCatalog and Manifest tests plus race, downstream extension
  tests, vet, build, scoped diff checks, and 1,842 OpenAPI references.

### 2026-07-15 P8 Hot Path, SSR, And Crawler Checkpoint

- Overall advances to **54%** after flooring. P8 is **83% (15 of 18 rows)**;
  P5 is **71% (12 of 17)**, P6 is **67% (12 of 18)**, and P7 is **50%
  (11 of 22)** at this checkpoint.
- The row audit corrects two stale checkboxes without inflating progress:
  production Page ViewModel construction is reopened because it leaves most
  page-specific product fields empty, while the already-accepted small/large
  compiler benchmark is now marked complete.
- `1c85d27b4` proves all 23 catalog templates perform no filesystem opens after
  compilation and 100 complete catalog provider-resolution passes perform one
  startup binding-list read and zero request-path binding reads. Existing typed
  render output removes regex island parsing and repeated template sanitizing.
- The compiler security matrix now covers dynamic XSS and URL contexts, missing
  values, recursive/invalid partials, unknown helpers, bounded output and
  cancellation, exact digest/schema binding, and the compiled fallback chain.
- `e125a47fc` makes the legacy plugin navigation island safe with empty defaults
  and disables query-insensitive Nitro SWR for paginated category/tag detail
  routes. Browser reproduction proved the old cached payload contained 6,004
  topics while current SSR/API contained 50; page 2 now hydrates without
  warnings and linked page 3 navigation changes content and canonical URL.
- JavaScript-disabled Playwright exercised home page 2, category page 2 -> 3,
  topic body/comments, profile topics, and the plugin add page. Independent
  Baiduspider parsing returned 200 SSR HTML with title, content, links,
  canonical, robots, five hreflang links, and valid JSON-LD on all five routes;
  home and category included real pagination anchors.
- Verification passed 27 focused Web tests, a 111-test P8 superset, all 310 Web
  tests, Nuxt typecheck/build, focused ThemeCompiler/Pages/Pages Controller Go
  tests, desktop browser screenshots, and the JavaScript-disabled browser run.
- The three open P8 rows are production page-specific ViewModel population,
  end-to-end plugin business-data contract preservation for theme overrides,
  and one exact API/Nitro restart plus concurrent-activation exit test.

### 2026-07-15 P7 Admin Surface HTTP Checkpoint

- Overall remains **54%** after flooring and P7 remains **50% (11 of 22
  rows)**. The Admin Surface production row is deliberately still open; this
  registry/transport/HTTP slice does not earn another row by itself.
- `3dff78a05` and `00470c41e` add the immutable exact-artifact Admin Surface
  Registry, lifecycle publication/rollback, active-runtime visibility fence,
  typed Protocol V2 invocation, frozen schema validation, and runtime admission.
- `8ee782c78` exposes the permissioned/redacted catalog at
  `GET /api/v1/admin/admin-surfaces` and exact typed invocation at
  `POST /api/v1/admin/admin-surfaces/:surfaceId/invoke`. Core Route guards
  require `admin.access`; the controller rechecks each declaration's Host-owned
  permission and hides modifiers whose target is not visible.
- Invocation requires a durable exact-artifact audit attempt before plugin code.
  Compatible publication swaps cannot redirect the call to a newer artifact,
  while an already-admitted old call retains its frozen validator and completes
  against the audited old instance. Input/output documents are never audited.
  The terminal success/failure append is best-effort; the mandatory attempted
  row is the durable gate that prevents unaudited plugin execution.
- Stable localized HTTP errors cover invalid/missing input, permission, missing,
  conflict/stale, timeout, transport, and runtime-unavailable outcomes. Public
  responses omit artifact digest, runtime instance, handler, and permission
  internals. The reviewed catalog now contains 225 Core routes.
- Verification passed full Support/Extensions, Extensions HTTP, bootstrap,
  Routes, and Localization packages; focused runtime and HTTP race tests; vet;
  build; V3 catalog drift; and 1,841 OpenAPI references across 45 files. The
  nested builtin-plugin Goldmark/go-redis `go.sum` gaps remain a separate open
  full-module gate.
- Do not close the Admin Surface row until Manifest V3 freezes kind-specific
  layout/placement, props and result use distinct schemas, the admin shell has
  concrete list/form/dashboard/editor/detail/import/export consumers with
  Host-attested actor/idempotency where mutation is possible, and an independent
  reference admin plugin exercises the complete surface family.

### 2026-07-15 P6 Streamed Transports And P7 Plugin Commands Checkpoint

- Overall is **52%**. P5 is **71% (12 of 17 rows)**, P6 is **67% (12
  of 18)**, P7 is **50% (11 of 22)**, and P8 is **56% (10 of 18)**.
- `cdcd46979`, `de3627df8`, `33702960d`, `90333626d`, `6cc634bf6`,
  `0f185817f`, `81f5d8208`, and `7f9b83cc7` close the P6 transport row with
  bounded Protocol V2 streams, authenticated preflight, exact-runtime leases,
  HTTP streaming, multipart, SSE, WebSocket, cancellation, and blocking
  backpressure pumps. The real-process test crosses Fiber, the immutable Route
  Registry, Dispatcher, Manager admission, gRPC, and the plugin SDK.
- Multipart delivered 1,049,078 bytes intact with a 652,000-byte observed
  maximum chunk below the 1 MiB limit. Real SSE delivered two events with the
  required media type; real WebSocket covered subprotocol negotiation and
  bidirectional messages; client disconnect returned exact runtime admission
  to zero.
- `a9b08a412`, `53068aea0`, `1becca5b1`, `cccdf3512`, and `9c446c7b8`
  close the P7 Plugin Command Registry row. Manifest commands publish through
  an immutable exact-artifact registry, invoke over Protocol V2 under runtime
  admission, and run from the out-of-band `sforum` CLI with namespace,
  conflict, trust, Safe Mode, and audit enforcement. The independent Safe Mode
  CLI recovery exit row is also closed.
- P6 passed focused stream repetition, real WebSocket TCP upgrade repetition,
  full Routes/HTTP/Extensions tests, focused race, vet, build, and the
  223-route/119-UI/99-trace-row catalog gate. P7 command focused, package,
  CLI, and race coverage passed; full nested builtin-plugin module gates remain
  open because their `go.sum` files lack Goldmark and go-redis entries. Do not
  report the nested-module gate as green until those dependency sums are fixed
  and the tests are rerun.
- Exact resume point: P6 continues the remaining inherited/custom/raw guard and
  action/SEO/OpenAPI work without freezing the six unresolved product
  semantics. P7 continues with Admin Surface Registry, then Query and Identity;
  first repair and rerun the nested builtin-module dependency gates.

### 2026-07-15 P7 Dynamic Jobs And Schedules Checkpoint

- Overall is **51%** and P7 is **41% (9 of 22 rows)**. Dynamic typed jobs now
  freeze exact artifact, job contract, payload schema version, retry policy,
  maximum attempts, fixed/exponential backoff choice, and concurrency limit in
  every River row.
- Manifest V3, embedded JSON Schema, and OpenAPI expose bounded policy values
  with deterministic compatibility defaults. The worker enforces per-artifact
  concurrency before plugin code and delegates exponential retry timing to
  River while preserving declared fixed retry delays.
- Plugin schedules use River's native `PeriodicJobs.AddSafely`/`RemoveByID`
  through a Host-owned dynamic publisher. Periodic leaders enqueue only an
  exact-runtime trigger marker; the trigger worker acquires the same schedule
  admission lease used by disable/upgrade drain before it writes the real
  versioned plugin job.
- Embedded workers reuse the API lifecycle snapshot. Standalone workers rebuild
  the same exact snapshot from active runtime identity, immutable Manifest, and
  executable trust. Safe Mode publishes no third-party periodic work.
- Commits: `1416aa121`, `0e81befcf`, `1eba53bd1`, and `814e824b3`.
- Passed ExtensionManifest, Jobs, Models/Extensions, HostAPI, Extensions, and
  bootstrap focused tests; Jobs/HostAPI/Extensions/bootstrap race passed, with
  the full Extensions race taking 136.425 seconds. OpenAPI validation passed
  1,817 references across 44 files.
- Exact resume point: continue P7 with the Plugin Command Registry, then Admin
  Surface Registry. Provider management browser QA remains pending after login.

### 2026-07-15 P7 Provider Management Checkpoint

- Overall is **50%** and P7 is **32% (7 of 22 rows)**. Generic versioned
  Provider Slots now have durable exact-artifact selection, reset, active
  candidate probe, runtime health/availability, deterministic fallback
  inspection, append-only events, and a bilingual management UI.
- Migration `023` stores exact contract-owner and candidate extension-version
  identities with revision CAS. A same-version digest replacement, owner or
  candidate upgrade, disable, or uninstall cannot inherit the choice.
- Runtime invocation honors the selected candidate first. `fallback=next`
  retains the declared deterministic remainder; `fallback=closed` tries only
  the selected exact candidate and fails closed when the binding is stale.
- API and standalone worker use the same PostgreSQL store. Lifecycle cleanup
  invalidates provider-slot choices before route and legacy provider cleanup,
  preserving actor/audit evidence.
- Commits: `881e08811`, `a08f68250`, `5d9afdd28`, `26175bbdf`, `ca9574339`,
  `0648ba15b`, `9913e543c`, `123fac2f9`, `2868e1d1e`, `e4afa1b16`, and
  `5fb841180`.
- Passed full `Support/Extensions`, targeted Models/HTTP/Routes/bootstrap,
  complete Go tests during the Inspector contract change, Nuxt typecheck,
  1,816 OpenAPI references, and the 12-file V3 catalog gate. PostgreSQL-only
  integration tests skip when no database URL is exported.
- Authenticated browser QA remains a final gate: Chrome redirects the new page
  to login and the in-app browser is unavailable. The login tab is retained as
  a handoff; do not inspect cookies or session storage.
- Exact resume point: run the final provider UI browser pass after login, then
  continue P7 with dynamic typed jobs, schedules, retries, concurrency, and
  payload-version drain behavior.

### 2026-07-14 P4 Convergence Checkpoint

- Overall remains **27%** and P4 remains **47% (7 of 15)**. The underlying P4
  implementation is substantially ahead of the accepted rows, but no row is
  promoted until the production bootstrap, exact migration engine, cleanup
  finalizer, recovery UI, and repository gates converge.
- `ee5a58fc1`, `41b8dc7b0`, `b9b52caf1`, `c2dd53941`, `af32d68cb`, and
  `af2408f0c` complete authoritative uninstall coordination, the PostgreSQL
  production fence, exact River reconciliation, immutable recovery authority,
  sanitized history failures, and target-admission ordering across the shared
  publication marker.
- `38b84f538` and `9ade60966` add the Database Registry ledger and scoped
  schema/role credentials. `cfa416e7c` synchronizes the Goose parser dependency
  for all three built-in protocol-v1 backend compatibility modules.
- `0cd41344f` exposes V2 uninstall results and retry/skip/forced recovery over
  authenticated HTTP/OpenAPI. Focused Controller, Models, Localization,
  OpenAPI-ref, and staged-contract gates passed.
- `b707fa47f` adds exact-artifact migration source/target proof and static
  preflight. Focused, race, and six real-PostgreSQL scenarios passed.
- Active dirty ownership is isolated: exact migration engine and role-integrity
  work under `app/Support/Extensions/extension_database_migration_*`; production
  assembly under `apps/api/bootstrap/{app.go,extension_lifecycle*}`; and
  management recovery/removal UI under `apps/web`. User-owned `.reasonix`,
  `.zcode`, and `CLAUDE.md` deletions remain untouched.
- Release blockers: migration-role pre-existing membership/settings/ownership
  rejection; failed advisory-unlock connection isolation; non-nil production
  migration engine; exact cleanup finalizer/purger with durable database
  disposition; full bootstrap/service binding; three removal modes plus
  retry/skip/forced UI; real PostgreSQL/race/vet/OpenAPI/Nuxt/browser/full-repo
  gates.
- Exact resume order: land the exact migration engine; land additive database
  disposition migration and API; land bootstrap and cleanup finalizer/purger;
  land the management UI; run the complete P4 gate; then update the 15-row
  ledger and begin P5 without reimplementing already-landed Database Registry
  prerequisites.

### 2026-07-14 P4 Registry And Service Production-Path Checkpoint

- Overall remains **27%** and P4 remains **47% (7 of 15)**. The newly landed
  Registry and Service slices are production prerequisites, but Jobs,
  migrations, bootstrap, uninstall/recovery, and the complete P4 gates have not
  yet converged as one production path.
- `8163f1673`, `02771edc2`, `687d84905`, `efa053d33`, and `acec487f8` bind Page,
  Route, Hook, and Service snapshots to immutable revisions and exact Manager
  runtime admission. A drained or staged target stays invisible, stale Hook or
  Service teardown cannot remove its replacement, and caller-owned inspection
  snapshots cannot mutate live Registry state.
- `392344a17` and `7ed4e15c7` persist the aggregate Registry publication phase
  and deterministically reconcile Page, Route foundation, Hook, and Service
  families behind the shared lifecycle publication fence. The failure test
  covers Route target publication followed by a conflicting Page snapshot;
  ordinary target calls remain closed and frozen-source recovery is required.
- `8dfb53009` and `be73a5eb5` prove and implement idempotent Protocol V2 service
  republication from the startup-frozen handshake after compensation removed
  the process-local Service set.
- `064b24f4b`, `548a4826e`, and `ccaa03c08` route trusted install/enable,
  disable, staged upgrade, and exact historical rollback through the durable
  coordinator with actor-bound stable idempotency keys, static preflight before
  position zero, retained authority for deactivation/rollback, HTTP/OpenAPI
  contracts, and admin client key propagation. A retry after state publication
  resolves the original operation before consulting mutated extension state.
- Clean `git archive HEAD` tests passed Models/Extensions, Pages, Routes,
  HostAPI, and Support/Extensions. Registry focused repetition, race, vet, and
  a real PostgreSQL Registry publication test also passed.
- Active dirty ownership is isolated: Jobs/shared-fence plus composed-boundary
  ordering; preflight/migration engine plus additive migration `013` Database
  Registry; bootstrap production assembly; and Service uninstall/recovery. The
  user-owned `.reasonix`, `.zcode`, and `CLAUDE.md` deletions remain untouched.
- Next: land the Jobs fence and post-marker reconciliation, migration `013`
  independently before its P5 implementation, then migration/preflight and
  bootstrap adapters. Finish uninstall preserve/export/remove, forced recovery,
  retry/skip UI, and denied/allowed tests before recalculating P4 progress.

### 2026-07-14 P4 Cleanup And Exact State Publication Checkpoint

- Overall remains **27%** and P4 remains **47% (7 of 15)**. The production
  cleanup and extension-state delegates are complete, but jobs/schedules,
  aggregate registries, migration proof, bootstrap construction, and Service
  routing are not yet one verified production path.
- `72806f0ad`, `cfdf2b246`, and `83162467b` add the retained cleanup schema,
  exact-artifact staging/finalization contracts, and real PostgreSQL coverage.
  Disable and retired-source recovery records remain retained. Uninstall only
  stages a durable tombstone; physical identity/package/data purge requires a
  terminally succeeded operation plus an idempotent exact receipt. Evidence
  retention is distinct from physical resource presence, and both purge/commit
  crash windows converge without inventing success.
- `341b38f36`, `afa7ea044`, and `71a64dfc8` add the durable extension-state
  publication intent, PostgreSQL transaction, and composed-boundary adapter.
  All six operations use operation-first locking, immutable source/target
  vectors, exact version CAS, marker-controlled restore, and restart-safe
  inspection. Upgrade restores its staged candidate; rollback preserves an
  unrelated staged pointer; a committed publication can never restore source.
- `da13783f4` updates the coordinator lease test to count every canonical Host
  gate. `1e71190f9` makes first trusted install promote a newer never-executed
  staged candidate atomically while retaining the old inert active/staged
  vector for compensation.
- Clean `git archive HEAD` plus staged-patch gates passed full Models and
  Support/Extensions tests, focused repetition, real PostgreSQL, race, and vet.
  Migration `009` passed isolated `Up -> evidence-protected Down refusal ->
  empty Down -> Up` against PostgreSQL.
- Three parallel uncommitted slices are active and isolated by ownership:
  `010` jobs/schedules plus HostAPI expected-plan fencing and post-marker
  reconciliation; `011` aggregate registry publication plus exact Page/Route/
  Hook/Service admission; and `012` static preflight plus durable migration
  source-resume proof. Their temporary shared-package compile failures are not
  committed and must be cleared before review.
- Mandatory review findings already sent to the jobs owner: post-marker River
  reconciliation must run before target admission opens; expected plan
  comparison must occur inside the same River transaction before irreversible
  mutation; crash after River commit but before evidence marking must recover
  from durable facts; arbitrary plugin/SQL error text must not be persisted.
- Next: review and land `010`, `011`, and `012` migrations independently, then
  their adapters; construct the exact Manager/runtime/Host/coordinator stack in
  bootstrap; run the same static preflight before position-zero process staging;
  and route first trusted enable through deferred install plus atomic
  publication. User-owned `.reasonix`, `.zcode`, and `CLAUDE.md` deletions remain
  untouched.

### 2026-07-14 P4 Atomic Lifecycle Publication Boundary Checkpoint

- Latest committed slices are `bd8afbfb8 feat(extensions): compose atomic
  lifecycle boundary` and `c938e74db feat(extensions): persist atomic
  publication decisions`. Overall remains **27%** and P4 remains **47% (7 of
  15)** because the production state/jobs/registry/migration adapters and
  Service/bootstrap path are not yet wired.
- The exact Host now drains job/schedule admission before runtime admission,
  rebuilds process-local source/target instances only during explicit
  revalidation, and reopens a failed early drain only after the canonical
  publication marker and durable migration compatibility proof both allow it.
- The composed boundary covers install, enable, disable, upgrade, rollback,
  and uninstall. Target runtime publication remains drained until durable
  extension state, jobs/schedules, and the aggregate registries all inspect as
  the exact target. The journal marker commits only after that convergence;
  post-marker failures remain closed and converge forward.
- PostgreSQL publication decisions are fenced by operation, canonical Host
  step, mode, exact source/target versions and digests, and attempt. Runtime
  instance ids may be rebound after process restart without changing artifact
  authority. Commit-unknown recovery reads a fresh durable marker, and marker
  evidence survives deletion of the mutable extension row.
- Focused `count=10`, race, vet, migration tests, full Support/Extensions tests,
  and real PostgreSQL concurrency/restart/commit-unknown tests passed. A clean
  `git archive HEAD` test also proved the commits do not depend on parallel
  uncommitted files.
- In-flight files are isolated to three parallel P4 slices: cleanup tombstones
  and finalization (`202607140008`), exact extension-state publication
  (`202607140009`), and production jobs/schedules publication (reserved
  `202607140010` if persistence is required). User-owned `.reasonix`, `.zcode`,
  and `CLAUDE.md` deletions remain untouched.
- Next: review and land migrations `008` and `009` independently, then their
  adapters; finish jobs/schedules and aggregate registry transactions; build
  the production preflight/migration adapter; construct the lifecycle stack in
  bootstrap; and route first trusted enable through deferred install plus
  atomic publication.

### 2026-07-14 P4 Host Gates, P6 Route Snapshot, And P8 Compiler Checkpoint

- Latest committed slice: `74fd5f367 test(themes): cover compiler security
  boundaries`. Overall remains **27%** and P4 remains **47% (7 of 15)**. P6
  and P8 foundations are not wired into production request/activation paths,
  so no authoritative row or displayed percentage is advanced.
- `891bcdbe5`, `110f752c2`, and `2794eb5eb` fixed the coordinator's five
  recovery defects, bound source/target exact runtimes, and exposed durable
  allowlisted lifecycle action results to Host gates.
- `a7a0d80ff` exposes a Manager-owned exact coordinator adapter without leaking
  or mismatching its private `ProtocolStarter`; `0b13bf3e3` dispatches all 32
  Host gate positions and revalidates stage/inspect/health/drain snapshots.
- `6f9dab6eb`, `5418fa1b5`, and `110a6941f` provide durable audit ids, scoped
  lifecycle history queries, and Service allowlist DTOs. Authority snapshots,
  opaque checkpoints, input/result documents, error metadata, and lease tokens
  cannot cross that Service inspection boundary.
- `ec5ed7fee`, `42ff24177`, and `1a37d7f41` add the P6 immutable API-route
  snapshot foundation: deterministic specificity/priority, revision CAS,
  exact runtime-instance artifact bindings, Safe Mode, conflicts, GET-to-HEAD,
  defensive activation validation, and explicit inherited-core-guard syntax.
  It still lacks Nuxt public/admin route defaults, execution/proxy semantics,
  provider selection, guard/schema/fallback enforcement, OpenAPI aggregation,
  Inspector/UI, and production lifecycle publication.
- `4969a6872` and `74fd5f367` add the P8 `html/template` compiler foundation:
  layouts/partials/control actions, restricted helpers, contextual escaping,
  tokenizer-backed static XSS rejection, passive ViewModel enforcement,
  recursion/source/output/deadline limits, immutable binding revisions, and no
  render-time filesystem access. Page ViewModels, explicit SafeHTML, typed
  segments/SEO, fallback, publication/restart convergence, and theme migration
  remain open.
- Focused normal/repeated/race/vet gates passed for Models/Extensions,
  Support/Extensions, Routes, ThemeCompiler, Pages, and Pages controllers.
  User-owned `.reasonix`, `.zcode`, and `CLAUDE.md` deletions remain untouched.
- In-flight dirty files belong to two isolated agent slices:
  `app/Support/Extensions/lifecycle_composed_boundary*.go` and lifecycle
  inspection controller/OpenAPI files. Neither is committable until its own
  failure-injection/contract gates finish.
- Next: commit the composed publication/compensation boundary and safe
  inspection HTTP contract; bootstrap the real repository/runtime/Host/
  coordinator; then route first trusted enable through deferred install and
  atomic activation before disable/upgrade/rollback/uninstall recovery UI.

### 2026-07-14 P4 Exact Runtime Publication And Call Barriers Checkpoint

- Last implementation commit: `11d12ed82 feat(extensions): run lifecycle on
  exact runtime instances`.
- P4 remains 7 of 15 rows complete. These commits close prerequisite runtime
  publication and call-admission contracts, but the lifecycle Service/HTTP path
  does not invoke them yet, so no authoritative task row or percentage was
  advanced.
- `ba0107459` gives Protocol V2 real staged, published, and retained physical
  processes. Unpublished start defers readiness until lifecycle work completes;
  exact health/publish/stop/discard and lifecycle calls never fall back to the
  active extension. Retained instances can be republished for rollback, stale
  stop cannot unregister a replacement, and V1 remains a hard replacement.
- `5d2cb6574` adds a Host-owned exact-runtime schedule admission registry.
  Publish, trigger acquire, drain, wait, failed-activation compensation, and
  retained rollback share one linearization boundary. The repository still has
  no manifest schedule trigger owner, so this is not claimed as production
  schedule integration.
- `3be41740a` binds protocol-v2 job enqueue to the active exact Manager
  instance. The lease spans the River insert, stale/draining identities fail
  closed, forced drain and caller cancellation remain distinguishable, and
  bootstrap installs the production adapter without creating an import cycle.
- `c20a87cde` binds the Service Registry winner's exact extension/instance
  identity to a Manager `RuntimeCallService` lease. Unary invocation and the
  complete bidirectional stream run on the lease context; stale, draining, and
  unavailable winners fail closed without trying a lower-priority provider,
  while forced drain remains distinguishable from caller cancellation.
- `fab179571` adds Manager-owned candidate start, exact health/readiness,
  publish, drain/wait, stop, and discard orchestration. Publication requires
  the old active instance to be drained and idle, fences both sides during the
  transition, fails closed across the ProtocolStarter/Manager switch, and
  preserves retained runtimes for exact rollback publication.
- `11d12ed82` adds the coordinator's exact-runtime adapter. Every lifecycle
  action validates its canonical step, source/target role and binding, frozen
  authority, plan, removal mode, and forced authority before acquiring a
  Manager lifecycle-cleanup lease and calling `RunLifecycleInstance`; a stale
  binding cannot fall back to the active process.
- Focused normal/repeated/race tests and vet passed for ProtocolStarter,
  Manager, HostAPI, Jobs, bootstrap, service admission, and exact lifecycle
  execution. Real subprocess coverage passed for staged publication, active
  plus retained lifecycle calls, stale-binding rejection, and forced-drain
  cancellation.
- The uncommitted Models coordinator slice passed its first normal/race/vet
  gate, then a mandatory self-audit found five recovery defects: final-gate
  success ordering, local-clock lease TOCTOU, incomplete source/target
  revalidation coverage, non-canonical marker ids, and side-effectful skipped
  terminal semantics. It is being corrected before commit.
- The exact-instance coordinator adapter, service-provider admission, and
  Manager staged-runtime API are committed. The Models coordinator corrections
  remain pending and uncommitted; the five self-audit defects must be fixed and
  its normal/race/vet gates rerun before that slice can land. Unrelated
  `.reasonix`, `.zcode`, and `CLAUDE.md` deletions remain untouched.
- Next: land the corrected Models coordinator; implement the production Host
  gate against the committed Manager stage/health/publish/drain/stop API;
  construct the coordinator in bootstrap; then move first trusted enable from
  the legacy `store.Enable -> runtime.Start` path into the durable transaction.

### 2026-07-14 P4 Exact Version And Runtime Instance Foundations Checkpoint

- Last implementation commit: `04c8b5d75 test(extensions): validate staged
  management contracts`.
- P4 is 7 of 15 rows complete. The 34-scenario real PostgreSQL matrix closes
  crash/retry at every lifecycle boundary; the broader idempotency/resume task
  remains open until Service and HTTP invoke the coordinator.
- Commits since the prior checkpoint: `f2d0ba93f` boundary recovery,
  `e01b10e8c` retry checkpoint inheritance, `075145fa7` coordinator,
  `a19ff67bb` lease migration, `397a0667c` forced wire authority,
  `0cb14b9f4` job migration ledger, `16dd33fbe` queued-job planner,
  `0a38bddb8` lease repository CAS, `b3adee77b` protocol-v2 runtime adapter,
  `3a6ccbb8f` leased Host-gate compatibility migration, and `71135e942`
  PostgreSQL/River job reconciliation. Later coherent slices are `dbb8f6a88`
  staged-version schema, `a2966f50c` inert store staging, `e5be7695b` runtime
  admission gate, `870fb4e34` coordinator lease execution, `e62a4c99c` staged
  trust review, `71340249f` inert Service upload semantics, `022fc843e`
  lifecycle authority snapshots, `468a67af1` exact staged promotion/discard,
  `fc0419b48` Manager exact-instance accounting, `125b0bbdd` inert staging
  OpenAPI, `e6fe6b3ef` staged-candidate admin display, and `04c8b5d75` the
  cross-layer management contract validator.
- The coordinator preserves stable steps/attempts/checkpoints/progress and
  detached terminal writes. Protocol-v2 now carries all eleven actions, exact
  forced authority, live progress, result JSON, and typed remote failures.
- Step lease ownership uses owner/revision/expiry CAS and PostgreSQL statement
  time, so lock waits cannot create already-expired grants. Real concurrent
  claims produce one winner; stale owners cannot heartbeat, persist, or close.
- Queued-job migration has an exact source/target/trust ledger, pure
  transactional replacement planner, and real pgx/River adapter. Replacement
  uses River's public `InsertTx` and `JobCancelTx`, conditionally links the Host
  ledger, and never updates River's private args storage directly.
- Host gates now have the independent additive `lifecycle_action = 'host.gate'`
  identity required to share the step-lease path without impersonating plugin
  actions. The reversible Down constraint retains historical Host-gate rows
  while preventing old binaries from writing new ones.
- Coordinator execution now claims and heartbeats every plugin action, Host
  gate, and forced skip. Exact lease revision fences progress and terminal
  writes; blocked heartbeats are cancellable; all terminal writes use a bounded
  detached context; Host failure recovery retains the original typed failure.
- Static uploads now persist immutable staged candidates without stopping the
  active process, changing enabled state, selecting providers, revoking the
  active exact-artifact grant, or writing migration execution history. Trust
  review and challenges bind the staged artifact while the active grant remains
  valid. The first-trusted-enable transaction is not yet wired, so the related
  P4 task remains open.
- An instance-bound admission gate now provides atomic ordinary-call closure,
  lifecycle-cleanup exemption, inflight wait, forced cancellation, and exact
  residual counters. Manager now retains exact instance snapshots and fences
  stale admission/stop bookkeeping, but ProtocolStarter still has one physical
  process slot per extension. Therefore dual-runtime execution, production
  route/job/provider admission, and the drain task remain open.
- Static upload responses and extension resources now expose an immutable
  staged candidate without leaking its database id. The admin list/details and
  bilingual success Toast distinguish staging from activation; OpenAPI refs,
  Nuxt typecheck/build, and the dedicated cross-layer validator pass.
- A P4 audit confirmed that the initial `planned` Host gate is currently
  skipped and disable/upgrade/uninstall lack required final Host gates. The
  accepted boundary requires exact source/target runtime binding, durable Host
  checkpoints, an upgrade activation gate, and final cleanup gates before the
  coordinator can own production lifecycle operations.
- Active parallel work: V2-only physical retained runtimes, lifecycle Host-gate
  path/request corrections, and exact historical-version rollback CAS. Unrelated
  `.reasonix`, `.zcode`, and `CLAUDE.md` deletions remain untouched.
- Next: land those three independent prerequisites, wire real admission and job
  barriers, execute first trusted install/enable through the coordinator, then
  implement upgrade/rollback/uninstall recovery API/UI and run the full P4 gate.

### 2026-07-14 P4 Exact-Artifact Plugin Job Checkpoint

- Last implementation commit: `d38e81d42 feat(jobs): execute exact-artifact
  plugin jobs`.
- P4 is 6 of 15 rows complete. River rows already persist the exact envelope;
  the worker now resolves the live extension and trust grant, rejects legacy or
  incompatible rows permanently, and rechecks the running and startup-frozen
  Manifest immediately before protocol-v2 dispatch.
- Job progress validates response identity, job id, monotonic counters,
  terminal state, typed failure/cancellation, and the absence of an undeclared
  result. A runtime change between resolution and dispatch maps to a permanent
  `runtime_changed` cancellation, so old code cannot receive the job.
- The deterministic upgrade policy defines execute, drain, declared migration,
  and cancel outcomes. Lifecycle-driven enumeration/migration of River rows is
  still part of the broader coordinator/drain work and is not claimed here.
- Verification passed focused repeated tests, relevant package tests, race,
  vet, `go test ./...`, `go build ./...`, staged diff review, and a parent
  targeted rerun.
- Current uncommitted ownership: crash-resumable lifecycle coordinator and the
  exhaustive PostgreSQL boundary recovery matrix. Unrelated `.reasonix`,
  `.zcode`, and `CLAUDE.md` deletions remain untouched.
- Next: review and land the boundary matrix and coordinator independently, then
  wire first trusted enable and lifecycle-driven drain into Service/HTTP.

### 2026-07-14 P4 State Machine And Durable Ledger Checkpoint

- Last implementation commit: `a3c4f75dc feat(extensions): persist lifecycle
  operations`.
- P4 is 4 of 15 rows complete. The Host-owned pure state machine fixes the
  authoritative ten states, six operations, eleven actions, recommended safety
  gates, terminal behavior, and the sole failed/cancelled retry path through
  recovery. Forced execution is uninstall-only; skippable plugin cleanup never
  bypasses Host safety gates.
- Additive PostgreSQL operation and step-attempt ledgers persist the exact
  artifact/authority snapshot, idempotency fingerprint, stable step ids,
  checkpoints, monotonic progress, typed errors, retries, actor/audit snapshots,
  and all three removal modes. Extension and audit retention cannot delete the
  lifecycle history accidentally.
- The repository serializes acquisition per extension, enforces one open
  operation, uses revision/state CAS, reuses an existing idempotency key,
  resumes failed/cancelled operations, and allocates monotonic step attempts.
- Verification passed real PostgreSQL migration Down/Up, migration/migrator
  tests, concurrent acquire/CAS/stable-step tests, restart/resume/retry tests,
  full Models/Extensions race detection, focused repeated tests, and vet.
- Current uncommitted ownership: versioned plugin-job runtime execution;
  crash-resumable lifecycle coordinator; exhaustive repository/state-machine
  boundary recovery tests. Existing `.reasonix`, `.zcode`, and `CLAUDE.md`
  deletions are unrelated and must remain untouched.
- Next: land the job and coordinator slices independently, wire the coordinator
  into exact-artifact first trusted enable, then implement drain/uninstall
  cleanup and operator retry/skip/forced-removal APIs and UI.

### 2026-07-14 P3 Completion And P4 Lifecycle Transport Checkpoint

- Last implementation commit: `e70dd677e feat(extensions): add typed lifecycle
  runtime transport`.
- P3 is 13 of 13 rows complete. Runtime streaming helpers cover route, file,
  lifecycle progress, and job streams; immutable service discovery is exercised
  across two real plugin subprocesses; the complete compatibility matrix and
  SMTP/storage/content-policy v1 package gates pass; transactional Host Commands
  retain rollback coverage.
- P4 is 3 of 15 rows complete. Activation now resolves required/optional/
  conflict/provides relationships before runtime start, with cycle, version,
  ambiguity, stale-candidate, and no-start failure coverage.
- All eleven lifecycle v2 actions cross the real gRPC subprocess transport.
  The Host freezes the lifecycle declaration at Start, rejects a stale or forged
  caller manifest, validates exact request/response runtime identity and result
  schema, preserves typed cancellation/retry metadata, and exposes the declared
  checkpoint schema while treating the current string checkpoint as opaque.
- Lifecycle transport verification passed the complete Extensions package,
  ten repeated real-subprocess runs, race detection, vet, and diff checks.
- Current uncommitted files belong to three active parallel slices: versioned
  plugin-job execution/drain compatibility, the additive lifecycle ledger
  migration, and the PostgreSQL lifecycle operation/step repository. Do not
  stage those groups together.
- Next: review and commit each parallel slice independently, run the P3/P4
  repository gates, then wire the lifecycle state machine and first-trusted-
  enable transaction to the durable ledger and frozen runtime contract.

### 2026-07-14 Service Broker And Reference Checkpoint

- Last implementation commit: `876836f7a fix(deploy): bind built-in plugin
  digest in image`.
- P3 is 9 of 13 rows complete. The dedicated CI drift row and built-in V2
  migration with real V1 rollback are now closed.
- Committed immutable Service Registry snapshots, exact SemVer/build selection,
  conflict inspection, Host List/Resolve/Invoke/Stream, schema enforcement,
  SDK service dispatch, idle timeout, and handshake freeze.
- ProtocolStarter now publishes Manifest-matched handshake services, serializes
  each extension lifecycle, removes exact instances on Stop, replaces instances
  on restart, and reaps registrations after unexpected process exit.
- Host authorization is the only caller-authority decision. Provider runtime
  grants are no longer confused with caller grants. Plugin-supplied Actor is
  rejected/cleared pending a Host-attested delegation contract.
- V2 hooks now bind full Manifest event id/name/kind/contract/input/result and
  derived patch schemas. Contract drift fails closed.
- `sforum.content-policy` defaults to Protocol V2, publishes a typed reusable
  service, enforces typed hook contracts, and retains a buildable/runnable V1
  source and Manifest. Linux image builds refresh and validate the exact binary
  digest after reproducible compilation.
- New focused gates passed: HostAPI and Extensions race/vet, content-policy
  race/vet, real crash/restart/concurrent-start subprocess tests, CLI Linux
  double-build reproducibility, built-in build script, proto drift, and Host API
  V2 documentation drift.
- Full Docker image build remains externally unverified because Docker Hub's
  anonymous-token request timed out over IPv6 before build steps began.
- Current uncommitted ownership: transactional Host Command implementation in
  `apps/api/app/Support/HostAPI/v2{,_command}*.go`; two-plugin E2E and v1 package
  compatibility agents are running in separate test files.
- Next: finish transactional rollback integration, real two-plugin service
  discovery matrix, SMTP/storage V1 package gates, then implement remaining
  route/file/progress/job streaming before the P3 full repository exit gate.

- Last implementation commit: `756c33738 test(extensions): cover protocol v2
  concurrency gate`.
- P3 commits so far: `b4d50005f`, `bda361626`, `1b9923372`, `ff4661103`,
  `7afa0c174`, `063f9897a`, `7320324e6`, `ef3ac6288`, `1bd1988c2`, and
  `01156b709`, `d5eef0127`, `297f6b92f`, `7dcac6eab`, `fc1fef8f4`, and
  `756c33738`.
- The library survey selected latest HashiCorp go-plugin `v1.8.0`, latest
  protobuf-go `v1.36.11`, pinned Buf `v1.71.0`, and protoc-gen-go-grpc `v1.6.2`;
  the isolated `tools/proto` module keeps tool dependencies out of the API
  runtime graph.
- `sforum.protocol.v2`, `sforum.plugin.v2`, and `sforum.host.v2` define 18 gRPC
  services and 147 message/enum declarations. They cover handshake, health,
  readiness, lifecycle, routes, hooks, query/transactional commands, database,
  cache, jobs, schedules, services, secrets, files, HTTP, admin, identity,
  permissions, media, navigation, audit, and tracing.
- Every dynamic document binds a schema id/version. Request context carries
  actor, locale, trace/request ids, deadline, exact extension identity, runtime
  epoch, trust grant, and disclosed authority. Transactional commands bind
  version, idempotency, dry-run impact, policy decisions, atomic outcome, audit,
  revision, and typed result.
- Generated Go code is committed under `apps/api/sdk/plugin/v2/gen`. Descriptor
  tests lock the complete service catalog, envelope fields, command result, and
  twelve required streaming modes. `scripts/test.sh` now runs Buf lint,
  generation, and drift detection.
- Runtime identity now exposes the exact grant, artifact digest, runtime token,
  epoch, and disclosed authority required by the v2 request envelope.
- The generated SDK provides a protocol-v2 plugin server with exact token,
  artifact, and epoch binding; health/readiness; 4 MiB message limits;
  deadlines; and a concurrency gate.
- Runtime startup selects transport exactly from the trusted Manifest:
  protocol v1 uses net/rpc, protocol v2 uses gRPC with AutoMTLS, and neither
  mismatch direction silently downgrades. Real subprocess tests exercise the
  v2 handshake, health, readiness, and hook path.
- Runtime status and the admin plugin list expose protocol version, transport,
  deprecation, start count, RPC call count, and last call. Protocol v1 remains
  operational and visibly deprecated until its P13 removal gate.
- Each v2 process now receives a unique `go-plugin.GRPCBroker` channel. Host
  calls require the exact runtime token in gRPC metadata plus the current
  artifact/grant/epoch/instance identity, disclosed authority, request id, and
  deadline; v2 no longer starts or receives the v1 loopback HTTP gateway.
- The Go SDK exposes all generated Host clients and builds bounded Host request
  contexts that preserve locale and trace while replacing runtime identity and
  authority. Actor is cleared until a Host-attested delegation exists.
- Host Query own-settings (unary and server stream), Permission, safe Identity,
  declared Job enqueue, and namespaced Audit append adapt the existing
  authoritative v1 services. Unsupported resource policy, job options,
  provider calls, cancellation, and list surfaces fail explicitly rather than
  silently discarding fields.
- Real subprocess tests cover Host callbacks, AutoMTLS broker streaming, stale
  identity, forged authority, expired deadline, cancellation, 4 MiB message
  rejection, concurrency saturation, and stop/start broker rebinding. Focused
  race detection passed.
- Verification passed: `go test ./...`, `go build ./...`,
  `./scripts/proto.sh check`, 1,607 OpenAPI refs across 40 files, Nuxt
  typecheck, and all 277 web validation tests.
- Rendered admin-page browser QA remains pending because both available local
  browser sessions were unauthenticated; the attempted route rendered the
  theme 404/dev overlay rather than the protected admin page.
- Working tree was clean at `756c33738` before this documentation checkpoint.
- This older checkpoint's Service Discovery next step is superseded by the
  2026-07-14 checkpoint above.

### 2026-07-14 Lifecycle Inspection, Generated Route Catalog, And SafeHTML Checkpoint

- Latest committed slice: `9100b078c feat(themes): add host-produced safe HTML
  values`. Overall remains **27%** and P4 remains **47% (7 of 15)**. The new
  P6/P8 prerequisites are still not production-published and therefore do not
  advance an authoritative row.
- `74c13e64f` and `ce30b306c` expose allowlisted lifecycle history/detail over
  authenticated `extension.view`, contract it in OpenAPI, and add stable route
  identities. The HTTP response excludes exact authority, idempotency,
  checkpoint, input/result, lease, and opaque error metadata.
- `2fe465eea` generates all 209 reviewed core API route identities into a
  caller-owned Go catalog from the same P0 source generator. Runtime code no
  longer needs to read `docs/` or maintain a handwritten duplicate when the
  Route Registry is production-wired.
- `9100b078c` bumps the Theme Compiler contract to
  `sforum.theme-compiler@2` and adds an opaque Host-produced `SafeHTML` value.
  Only the explicit `safeHTML` helper accepts it; ordinary strings remain
  context-escaped, Go trusted-content aliases remain rejected, and URL or
  attribute contexts retain `html/template` filtering.
- Composed lifecycle review found three release-blocking crash boundaries now
  being corrected before commit: target admission must remain drained until
  DB/jobs/registries are exact; uninstall position 6 may only stage a durable
  pending purge until the operation terminal is committed; and publication
  requires a durable exact-operation journal so restart can converge partial
  DB/jobs/registry writes instead of relying on process-local compensation.
- Source drain must close jobs, schedules, and route-facing admission before
  waiting on the exact runtime. A missing production drainer fails closed.
- Recovery decision persistence is in flight against migration `202607140006`.
  It keeps the original exact-artifact authority immutable while recording the
  actor/audit/reason for every retry, skip, and forced-uninstall escalation.
- User-owned `.reasonix`, `.zcode`, and `CLAUDE.md` deletions remain untouched.

## P4 Exit Checkpoint

- P4 is complete at **15 of 15 rows**. Overall V3 progress is **31%** and P5 is
  active at **0 of 17 rows**.
- Production bootstrap binds Service mutations to the PostgreSQL lifecycle
  repository, exact protocol-v2 runtime, static preflight, migration engine,
  River jobs, schedule admission, route/service/page registries, durable state,
  publication journal, cleanup tombstone, database disposition, and terminal
  physical-purge finalizer.
- Static install remains inert. `install.plan` and `install` first run only in
  the exact-artifact trusted enable transaction. Disable, upgrade, rollback,
  and uninstall retain frozen authority and exact runtimes until their declared
  hooks and Host-owned cleanup boundaries finish.
- Preserve, export-then-remove, and complete removal have real PostgreSQL
  coverage. Repeated uninstall, external cleanup failure, forced skip/recovery,
  original actor/audit retention, crash replay, and incompatible queued jobs
  are covered across Service, coordinator, boundary, and full-chain tests.
- `9895221a8` makes the failed machine snapshot and typed error one durable
  transition before terminal completion. `3b6480fb9` marks a successfully
  claimed step lease `running`, so the exact PostgreSQL fence cannot reject its
  own Host gate. `1982525d6` proves Service -> PostgreSQL coordinator -> real
  protocol-v2 subprocess -> composed deactivation -> cleanup/finalizer.
- Authenticated Chrome QA passed on desktop and 390px mobile for lifecycle
  history, close behavior, long-id wrapping, and all three uninstall modes;
  refreshed console warn/error output was empty.
- Final gates passed: real PostgreSQL lifecycle suites, related five-package
  race detection, five-package vet, and complete `./scripts/test.sh`, including
  1,677 OpenAPI references, Nuxt typecheck, 212 routes, 117 UI surfaces, and 99
  traceability rows.
- P4 owns schedule admission and drain only. Dynamic plugin periodic trigger
  registration remains a P7 deliverable and should use River
  `PeriodicJobs.AddSafely` / `RemoveByID`; no current production trigger caller
  bypasses the P4 admission gate.
- P5 must reuse the already-landed deterministic Database Registry, scoped
  credentials, exact migration engine, migration ledger, and uninstall data
  disposition. Resume by auditing the 17 P5 rows against current production
  code before implementing only the missing Host Query/Command and database
  authority boundaries.

## P5 Implementation Checkpoint

- P5 is **65% complete (11 of 17 rows)**. Weighted overall V3 progress is
  **36%**. This count credits only production behavior or exact integration
  evidence, not protobuf shapes or unwired foundations.
- Completed rows: deterministic safe identifiers; exact migration discovery,
  checksum, lock, read-only dry-run, transaction policy, ledger/progress/failure;
  isolated Goose parsing/history; uninstall database disposition; broad
  migration/credential/disposition tests; own-schema isolation; and
  multi-process migration-once. Core-upgrade compatibility blocking and
  database query tracing/slow-query observability are now also complete.
- `c913127c8` adds a public exact-artifact migration preflight. It uses a
  PostgreSQL read-only transaction and the same Goose parser as execution,
  reports statement/checksum digests, transaction modes, non-transactional
  warnings, and backup strategies, and proves no SQL execution, resource
  provisioning, or ledger writes.
- `d75de7751` runs the same exact migration plan from two independent test
  processes and independent PostgreSQL pools. The durable advisory lock and
  ledger converge to one plan, one applied step, and one state row.
- `dd0e08e74` enforces eight connections, a five-second statement timeout, and
  a fifteen-second idle-transaction timeout on every own-schema runtime role;
  real PostgreSQL tests prove over-budget rejection and slow-query cancellation.
- `c9ca0d797` publishes Host-owned, security-barrier, non-updatable
  `sforum_core_v1` views for safe identities, public forum content, entity
  metadata, and attachment metadata without granting base-table access.
- `801052f2e`, `0c2fcdfec`, and `eedaabb15` add the durable Host Command receipt
  ledger, server-attested exact broker identity, and a PostgreSQL backend that
  keeps domain writes, audit, and replay evidence in one transaction. Concrete
  production domain command definitions remain required before the command
  rows can close.
- `368e48e4d` production-binds an immutable Host Query catalog for safe user,
  public topic, and public attachment reads. Exact enabled artifacts and live
  trust grants are resolved server-side; forged request identity/authority,
  stale artifacts, PII fields, unsafe shapes, and oversized pages fail closed.
- `c142426a2` promotes database authority, compatibility, backup/retention,
  migration digest, and transaction policy in the exact-artifact trust flow.
  Authenticated admin-page QA plus a temporary production-component fixture
  passed on desktop and 390px with no overflow or app console errors; ordinary
  backup guidance expires after ten seconds while high-risk warnings persist.
- `ebc8c3919` runs an exact-trust raw-authority compatibility check before any
  Goose core migration. A real PostgreSQL test proves incompatible target
  versions block while revoked grants no longer do.
- `86ed767d8` and `36e143386` add bounded, redacted Host Query tracing with
  slow classification and document the direct-role PostgreSQL logging boundary.
  Real PostgreSQL, race, vet, and build gates passed.
- `c13da29d1`, `14086dee2`, and `926ed2ff2` add the exact-artifact
  DatabaseService core, real own-schema transaction/isolation/replay/revocation
  proof, and frozen gRPC registration.
- `42a9d895d` adds the Manifest V3 source for that catalog: namespaced positive-
  version operations bind exact `database_operation` package files and digests,
  typed parameter/result schemas, column allowlists, and HostAPI-aligned limits.
  Inline SQL, authority mismatch, query/execute field mixing, undeclared local
  schemas, and JSON Schema drift fail closed.
- `e5bb88c3b` and `4a1fb0fb3` complete the plugin-owned transaction row:
  API and standalone worker bootstrap bind exact SQL catalogs before any
  broker registration; active, disabled/installed, staged, upgrade, and
  rollback artifacts are supported without mutating already-running broker
  snapshots. DatabaseService trace output is bounded and redacted.
- `bc591e691`, `09df90982`, and `dc49dcf6d` add strict modular plugin OpenAPI
  aggregation plus the durable route-provider selection Store/API and V2
  cleanup invalidation. They do not advance P6 yet because production
  bootstrap, admin conflict UI/API, and the Fiber dispatcher remain open.
- `e5eea90ab`, `7dbe3e3c0`, and `43be1b5d5` begin P6 without inflating its
  completion count: exact runtime publication fencing, wildcard conflict
  inspection, the 212-row production core catalog, Safe Mode filtering, and an
  immutable fail-closed execution plan are present, but Fiber has no Registry
  dispatcher yet.
- Partial rows: runtime credential delivery; physical `database.core.full`
  grants; stable views/typed query-command publication; and concrete PostgreSQL
  Host Command atomicity. Missing rows include six concrete Host Commands and
  raw-authority behavior tests.
- The authority audit found a real public-contract conflict. ADR prose says
  database powers are disclosed in tiers, while Manifest, JSON Schema,
  OpenAPI, validation, and the database ledger all store one mutually
  exclusive authority. Do not silently invent cumulative semantics.
- Product decision requested: prefer additive composable database grants plus
  exact-artifact declared operation catalogs and bounded batch transactions.
  Direct per-process credentials remain an alternative, but the current
  one-role-per-extension rotation model terminates old sessions and is unsafe
  for rolling upgrade or multi-node runtime use.
- Additional product boundaries remain for trusted actor delegation into
  actor-scoped Host Commands and the provider-neutral entitlement lifecycle.
  Continue independent stable-view, connection-budget, migration-guidance,
  and backend-ledger work while awaiting those decisions.
- PostgreSQL development dependencies are available. Do not assume frontend or
  API dev servers remain running; start them explicitly before browser QA.

## P6 Route Inspector And Dispatcher Integration Checkpoint

- Weighted V3 progress remains **36%**. P5 remains **65% (11 of 17 rows)** and
  P6 remains **0% by authoritative exit rows**. P6 foundations are materially
  ahead of zero, but no row is promoted until its production transport,
  authority, schema, and fallback boundaries close together.
- Latest committed slice: `738b8a30b feat(routes): inspect exact execution
  chains`. The detached Inspector reports one immutable Registry revision,
  exact execution chain, selected/stale/unselected provider state, guard and
  contract metadata, and relevant bounded timing/fallback traces. It never
  chooses a provider by priority and filters unrelated route conflicts/traces.
- `RouteTraceRing` is concurrency-safe and circular, defaults to 256 records,
  hard-caps at 4096, and accepts no request, response, header, query, body,
  secret, actor, idempotency-key, or raw-error fields. Forged exact-artifact
  attribution is rejected.
- Inspector focused, race, vet, build, and diff checks passed. The committed
  core is not yet exposed through HTTP/UI and the Dispatcher has not yet
  published trace events, so the P6 Inspector task remains open.
- Authenticated Chrome initially rendered
  `/control-panel/extensions/route-providers` with the expected super-admin
  identity, navigation, route-provider copy, counters, and loading state with
  no console warnings/errors. API hot reload temporarily disappeared and the
  page received 502 responses; after API recovery, a full reload found the
  login session expired. Desktop/mobile selection/reset QA therefore remains
  incomplete and must not be reported as passed.
- Active dirty ownership is isolated: Fiber buffered dispatcher and production
  bootstrap in `app/Http`, `Support/Routes/dispatcher*`, and `bootstrap/app.go`;
  Inspector HTTP/OpenAPI/catalog generation in Extensions controllers,
  contracts, and generated route catalogs; Page ViewModel/compiler work in
  `Support/ThemeCompiler`. Do not stage these groups together.
- Dispatcher review found boundaries that must stay explicit: pure core plans
  must bypass buffering to preserve existing streaming/download behavior;
  inherited core guards need executable Host guard metadata; declared schemas
  need an exact-artifact schema catalog; and safe fallback needs Host-observed
  side-effect evidence rather than trusting only a plugin response header.
  SSE, WebSocket, multipart, streaming, and backpressure remain unimplemented.
- P8 review rejected a ThemeCompiler-only `forum.search` identity because the
  current Page Registry has no standalone search page. Search remains state in
  the existing home/list ViewModel until a real core page identity is added.
- Exact resume order: review and land the pure-core-safe Fiber adapter; review
  and land Inspector HTTP/OpenAPI separately; review and land the catalog-only
  Page ViewModel slice; then implement Dispatcher trace publication, exact
  schema catalogs, inherited/custom guard contracts, Host-observed side-effect
  fencing, and the non-buffered transport modes before recalculating P6.

## P6 First Accepted Rows Checkpoint

- P6 is now **33% complete (6 of 18 authoritative rows)** and the weighted V3
  total is **39%**. The active phase is P6; P5 remains partially open at 65%
  only for the three recorded product decisions and their dependent rows.
- Accepted task rows: all 218 core routes have generated stable ids/contracts;
  snapshots are immutable with deterministic specificity/priority; declared
  plugin routes may target arbitrary public/admin/API paths and methods; and
  exact replace-provider selection plus conflict API/UI is production-bound.
- Accepted test rows: Safe Mode excludes all third-party route snapshots while
  preserving Host routes, and strict plugin OpenAPI aggregation rejects
  collisions and unsafe package references.
- `9bcaad539` exposes a permissioned, detached Route Inspector snapshot over
  modular OpenAPI without returning the raw inspected path/query. `ace782c5b`
  production-binds the buffered HTTP dispatcher to the exact lifecycle
  Registry/Runtime Manager while leaving pure core streams untouched and
  filtering request authority, Host headers, and plugin `Set-Cookie` output.
- These commits do not close full action semantics, inherited/custom/raw
  guards, exact schema catalogs, Host-observed side-effect fencing, SEO alias/
  redirect integration, live trace publication, streaming transports, the
  complete action/disconnect/crash matrix, or the new-vs-v1 benchmark row.

## P6 Production Trace Checkpoint

- P6 is now **39% complete (7 of 18 authoritative rows)** and weighted V3 is
  **40%**. The newly accepted row is the production Route Inspector with exact
  chain/provider/guard/contract metadata plus bounded timing and fallback
  traces.
- `3b017173c` injects one concurrency-safe trace ring into both the production
  Dispatcher and permissioned Inspector. Plugin steps publish redacted denied,
  schema-rejected, transport-failed, fallback-used, succeeded, and committed
  outcomes with exact artifact attribution and commit state. Pure core routes
  remain untraced and keep their existing streaming path.
- Independent review found that a non-handler `readonly_core` fallback was
  allowed to continue without a fallback/commit trace. `61da559d5` closes that
  audit gap and adds the before-plugin -> core fallback regression.
- Verification passed 20 repeated Route package runs, Route/HTTP race tests,
  focused vet, bootstrap/controller/provider regressions, `go build ./...`, and
  `git diff --check`.
- Active dirty groups remain isolated: the exact-artifact OpenAPI route schema
  catalog under `Support/ExtensionOpenAPI`, and P8 Page ViewModel/typed render
  work under `Support/ThemeCompiler`. P8 independent review found nested-island
  structure and required Host-form-island blockers, so no P8 row is credited.
- Next: review and land the schema catalog foundation, then production-publish
  it from exact lifecycle artifacts; complete Host-observed side-effect fencing,
  inherited/custom/raw guards, remaining action semantics, streaming transports,
  SEO integration, complete failure matrix, and the v1 comparison benchmark.

## P6 Host-Observed Fallback Fence Checkpoint

- P6 is now **56% complete (10 of 18 authoritative rows)** and weighted V3 is
  **41%**. Newly accepted rows are safe GET/fail-closed unsafe fallback,
  preventing fallback after Host-observed request/response commitment, and the
  unsafe replacement failure test that proves Core is never a second writer.
- `caa158402` removes the plugin-controlled side-effect response header as an
  authority source. The Host transport records request headers/request write
  and first response byte through `net/http/httptrace`; after the request leaves
  the Host, crash, disconnect, cancellation, timeout, oversized response, or
  partial response all remain fail closed.
- A pristine safe-method dial failure may still use declared `not_found` or
  `readonly_core` fallback. Unsafe methods never fallback. Tests cover accepted
  GET crash, partial headers/body, accepted POST, timeout/cancellation, forged
  side-effect headers, exact commit-state precedence, trace publication, and
  zero Core calls after any possible plugin write.
- Verification passed 10 repeated HTTP/Route/bootstrap runs, race detection,
  focused vet, `go build ./...`, and staged diff checks.
- The schema catalog remains uncommitted after a second independent review
  found missing status/media-type identity, duplicate JSON-key rejection, and
  bounded/cancellable validation. P8 typed Page ViewModel output was committed
  in `d9268872a` but remains at 0 authoritative rows until production
  construction and runtime publication are proven.
- Next: close and production-bind exact schemas; implement executable inherited
  and separately trusted custom/raw guards; then complete action protocol,
  alias/rewrite, streaming transports, SEO integration, failure matrix, and
  the v1 comparison benchmark.

## P6 Typed Guard And Exact Schema Foundation Checkpoint

- P6 remains **56% complete (10 of 18 authoritative rows)** and weighted V3
  remains **41%**. Neither foundation is promoted until its production runtime
  and lifecycle publication are complete.
- `1fcfdbf05` adds reviewed typed guards for all 218 Core routes, exact inherited
  guard resolution, fail-closed missing/plugin targets, immutable permission
  slices across Registry/Match/Inspector/plan boundaries, and source-verified
  generated catalog policy. The production Guard Authorizer is still active
  work; custom/raw guard authority remains separate high-risk trust.
- `6667e630f` adds an exact artifact/route/method/contract/action/direction/
  schema/media/status catalog using `jsonschema/v6`. It isolates operation
  variants, rejects duplicate/trailing/deep/oversized JSON, shares compiled
  schemas, and bounds decode plus validation with slots, deadlines, structural
  budgets, and exact HEAD-to-GET response admission.
- Schema focused count-10, split race, vet, and `go build ./...` gates passed.
  Production still injects a nil catalog. Correct publication must join Route
  and Schema restore under the durable lifecycle fence; restart currently
  restores only Core routes.
- Production has no source for the Host-owned OpenAPI route policy tuple
  (security, rate-limit, idempotency). The ADR requires Host authority but does
  not select defaults or a persistence/resolution contract. Do not invent
  plugin-controlled or inferred policy values merely to publish the catalog.
- P5 remains 11/17: its remaining rows depend on composable database grants,
  actor delegation, provider-neutral entitlement semantics, and direct database
  credential rolling-upgrade policy. These do not block independent P6/P8 work.
- Active parallel work: production typed Guard Authorizer, Dispatcher allocation
  regression/benchmark, and P8 production Page ViewModel construction/publication.

## P6 Performance Comparison Checkpoint

- P6 is now **61% complete (11 of 18 authoritative rows)** and weighted V3 is
  **42%**. The accepted row is the reproducible performance comparison against
  the current namespaced proxy baseline; the measured V3 regression remains
  explicit and must be remeasured at P13.
- `e38d91f7a` binds planning to one internal immutable Registry revision instead
  of copying all 218 routes one or two times per request. Public Snapshot,
  Resolve, Match, and plan getters remain caller-owned deep copies with mutation
  and concurrent-publication race coverage.
- Allocation gates pass at Core 21/64, selected HTTP 352/480, and six-step chain
  1,557/2,100 allocations. The same-run medians are v1 198.263 us, Core 117.740
  us, selected 606.905 us, and composed 2.418 ms.
- Selected HTTP is still +206.1% latency, +232.9% bytes, and +208.8%
  allocations versus the comparable v1 fixture. Production PostgreSQL provider
  lookup is excluded, so this is not a no-regression claim.
- `24ae6a3a2` production-binds the typed Core Guard Authorizer with an exact
  plan/step/request fence. Eleven explicit evaluator ids cover 22 contextual
  routes; the remaining 101 contextual routes and custom/raw authority stay
  fail closed and therefore do not close the guard row.
- `b84920a03` separates schema-only aggregation from policy publication. Schema
  compilation strips policy fields; public OpenAPI aggregation still requires
  exact Host-owned security, rate-limit, and idempotency policies.

## P6 V2 Transport And P8 Runtime Checkpoint

- Weighted V3 progress is now **44%**. P6 remains **61% (11 of 18 rows)**;
  Protocol V2 unary transport is a required production slice but does not by
  itself close the complete route-action or streaming rows. P8 is accepted at
  **33% (6 of 18 rows)** after production review.
- `c64fa56f8` adds revision-fenced immutable Route Schema publication. A
  prepared candidate is publishable only when its base revision, caller
  expectation, and current revision all match; stale and foreign writers fail.
- `0a1997578` and `2a180cb07` build and production-bind exact theme runtime
  snapshots for 23 of 23 Core Page ViewModels. Snapshot-covered requests use
  compiled templates without Store or filesystem access, stale artifact
  identity fails closed, and PAT-scoped viewer authority cannot regain full
  permissions from the user projection.
- `d1d42a130` measures production `Snapshot.Render` and compile paths for small
  and large fixtures and adds allocation ceilings. The report explicitly
  defers full request-chain, RSS, and JavaScript-disabled comparison to P13.
- `6a41bbcd9` routes Protocol V2 HTTP-mode steps through real gRPC
  `InvokeRoute`, exact retained instances, one Manager admission lease,
  Host-authored actor authority, frozen route/schema identity, bounded typed
  responses, and fail-closed `stream_follows`. Review removed a lifecycle mutex
  that serialized all same-plugin requests and made trace ids unique per call.
- P8 rows accepted: all catalog Page ViewModel contracts, the bounded compiler,
  standard template control actions, sealed SafeHTML, immutable exact runtime
  snapshots, and the compile/render performance row. Install-time compilation,
  all-catalog zero-I/O, typed frontend consumption, four-level fallback,
  plugin business ViewModels, multi-node convergence, and crawler/JavaScript-
  disabled evidence remain open.
- Verification passed focused repetition, race detection, full Pages,
  Extensions, Pages Controller, HTTP and bootstrap packages, vet, all API tests,
  `go build ./...`, and staged diff checks.
- Active parallel work is isolated: immutable Page provider resolution under
  `Support/Pages`, and production Route Schema lifecycle/bootstrap publication.
  Next integrate Route + Schema restore, then continue contextual/custom/raw
  guards and freeze full action semantics before streaming transports.

## P6 Schema Lifecycle And P8 Admin Isolation Checkpoint

- Weighted V3 progress is now **45%**. P6 remains **61% (11 of 18 rows)**;
  P8 advances to **39% (7 of 18 rows)** after accepting public-theme/admin
  style isolation. P5 remains partially open at **65% (11 of 17 rows)** and
  does not block independent P6/P8 delivery.
- `69ac44074` fences exact Route Schema replacement artifacts, and `69266b051`
  publishes schemas before Route Registry exposure, restores both immutable
  publications after runtime reconciliation, keeps Safe Mode Core-only with an
  empty plugin schema catalog, and injects the live publication into the
  production dispatcher validator.
- `ecf860984` adds notification-recipient contextual guard evaluation. Explicit
  evaluators now cover 26 of 123 contextual routes; the remaining 97 routes,
  custom guards, and raw request/session authority continue to fail closed.
- `6dda8c9e0` resolves Page providers from immutable snapshots without a
  PostgreSQL lookup on the production request path.
- `9e369ffc8` keeps public theme skins out of admin desktop/mobile routes while
  restoring them on SPA navigation back to public pages. Browser QA found two
  exact public links, zero admin links, and no relevant console errors.
- Verification passed focused/race package tests, vet, `go build ./...`, 290
  web tests, Nuxt typecheck, production build, and desktop/mobile browser QA.
- Next close the remaining contextual guard catalog and independently trusted
  custom/raw guards, then implement the frozen route-action semantics and real
  multipart/streaming/SSE/WebSocket transports. P8 next owns typed frontend
  render segments, four-level fallback, and all-catalog zero-I/O evidence.

## P8 Install-Time Template Safety Checkpoint

- Weighted V3 remains **45%** after flooring. P8 advances to **44% (8 of 18
  rows)** by closing install-time static template safety; P6 remains **61% (11
  of 18 rows)** and P5 remains **65% (11 of 17 rows)**.
- `b54b8f541` runs bounded preflight over every uploaded theme L1 page plus its
  shared layouts and partials before the package enters the authoritative
  Store. Static dangerous HTML, forbidden helpers, recursive/deep template
  graphs, and contextual escaping failures are rejected even in unused pages.
- Activation still recompiles the exact artifact. A failed candidate cannot be
  staged or published, and the prior immutable runtime continues serving.
- Focused and repeated package tests, targeted race tests, Models race tests,
  vet, `go build ./...`, and ordinary ThemeCompiler allocation budgets passed.
  Full ThemeCompiler under `-race` exceeds pre-existing allocation ceilings
  because of race instrumentation and is not treated as an allocation result.

## P5 Remaining-Boundary Audit

- P5 remains **65% (11 of 17 rows)**. The six open rows are credential delivery
  to plugin processes; stable views plus typed Query/Command publication; six
  production Host Commands; physical `database.core.full`; raw-authority
  behavior/compatibility; and complete Host Command atomicity evidence.
- The current database registry issues one shared runtime role per extension.
  Password rotation or revocation changes that role and terminates every
  session using it, so wiring the current credential directly into plugin
  startup would break rolling upgrades and multi-node operation.
- Product freeze is required before implementation: map legacy single-choice
  `database.authority` to additive grants, and choose per-runtime lease roles
  versus Host-mediated DatabaseService-only access. The recommended V3 path is
  cumulative compatibility grants plus per-runtime lease roles, with old/new
  credentials coexisting until the old exact runtime drains.
- Migration read-only preflight, retry, advisory locking, and failure recovery
  are already accepted. More tests in those completed areas cannot be used to
  inflate the six remaining rows.

## P6 Guard, P7 Service Dependency, And P8 Typed Render Checkpoint

- Weighted V3 progress is now **46%**. P6 remains **61% (11 of 18 rows)**;
  P7 begins at **5% (1 of 22 rows)**; P8 advances to **50% (9 of 18 rows)**.
- `28b3dbf74`, `40aab8be5`, `e60324989`, `7b93272a0`, `ff85c6fa3`,
  `a8a59df5b`, and `afaaa6eb8` expand exact contextual guard execution from 26
  to 68 of 123 routes. Fifty-five resource- or policy-dependent routes remain
  fail closed; custom and raw request/session authority also remain closed.
- `82a16c65e` closes typed plugin-to-plugin service dependency/version checks.
  Protocol V2 atomically publishes exact caller/provider identity, services,
  required/optional ID or capability SemVer, and provided capabilities. List,
  Resolve, Invoke, and Stream share one immutable snapshot with no database hot
  path; stale, disabled, upgraded, missing, version-drifted, and ambiguous
  runtimes fail closed.
- `615a9496f` closes typed frontend RenderOutput consumption. Public registry
  routes prefer typed HTML segments and island descriptors; parse5 rebuilds the
  same nested HTML5 AST in SSR and hydration, while the legacy L1 compatibility
  path also no longer uses regex or `v-html` island splitting.
- Typed output rejects missing/duplicate/forged placeholders, unknown
  components, cross-type props, dangerous elements/attributes/URLs, and
  correctly restores Go `omitempty` zero values. Focused tests passed 14/14,
  all web tests 304/304, Nuxt typecheck and production build passed, and real
  3002 SSR/reload QA reported no placeholder leakage, alert, or console errors.
- `e11c518bb` fixes startup Route/Schema restoration so legacy provider-only
  plugins are skipped while exact runtimes with declared Routes/OpenAPI remain
  eligible without requiring lifecycle hooks. A real API boot completed
  migrations, runtime reconciliation, publication restore, embedded worker
  startup, and listen on 8081 before a clean shutdown.

## P6 Guard, P7 Hook Registry, And P8 Fallback Checkpoint

- Weighted V3 progress is now **48%**. P6 remains **61% (11 of 18 rows)**;
  P7 advances to **18% (4 of 22 rows)**; P8 remains **50% (9 of 18 rows)**
  until the full theme-switch row, add-page SSR path, and multi-node evidence
  converge.
- `476ca7aca`, `5b96b58f2`, and `24ed8b4d8` production-bind immutable
  exact-artifact extension policy, static option/public policy, and Page
  Registry access policy. Contextual guard execution now covers **101 of 123**
  routes; 22 resource-dependent routes plus custom/raw authority remain closed.
- `ae4ca62fd` implements the compiled active-override -> plugin -> active theme
  -> default theme -> Host emergency render plan, exact template digest and
  ViewModel binding, default-theme prewarm, typed attempt evidence, and zero
  request-time package I/O. The P8 fallback row remains open until plugin add
  pages and the complete switch/convergence contract are verified.
- `124c151dc`, `77674b2fc`, and `4b3f8b82c` close the first three P7 tasks:
  Manifest V3 versioned namespaced action/filter hooks; immutable exact-runtime
  priority/schema/failure-policy composition; and mandatory Host revalidation
  for authoritative filters. Plugin dependency SemVer, optional-provider
  downgrade, River exact delivery, nested payload isolation, lifecycle
  rollback, and Protocol V2 declaration binding fail closed.
- Hook validation rejects async fail-closed contracts because individually
  durable enqueues cannot be rolled back, and bounds Host listener deadlines to
  1-5000 ms. Full ExtensionManifest/Extensions tests, complete Extensions race,
  focused rollback races, vet, and build passed before the three commits.
- Active uncommitted ownership is isolated: P6 continues the remaining guards;
  P8 owns additive active-theme migration, exact-preview/current-theme CAS,
  stale binding reconciliation, runtime skin revision, OpenAPI, and admin UI.
  Do not stage P8 files with guard or documentation commits.
- P5 remains **65% (11 of 17 rows)** pending explicit product freeze for
  additive grants, per-runtime lease roles, actor delegation, and the
  provider-neutral entitlement minimum. Independent P6-P8 work continues.

## P6 Declared Route And Cookie Credential Checkpoint

- Weighted V3 remains **48%**. P6 remains **61% (11 of 18 authoritative
  rows)** because these guard batches do not yet close the complete inherited/
  custom/raw guard row.
- `32d32ac72` authorizes declared extension route guards against immutable
  exact-artifact policy, and `306204f98` adds Host-derived cookie/bearer
  credential provenance for PAT list/create. Client header text is not trusted;
  the dispatcher reads the authenticated PAT context after Bearer middleware.
- Contextual guard execution now covers **105 of 123** routes. The remaining 18
  are five target-dependent identity admin routes, three self resource-dependent
  identity routes, four executable bootstrap flows, two entity-meta value
  routes, two attachment reads, one inbound webhook, and one forum comment
  creation route. Custom/raw request and session authority remain fail closed.
- The cookie-bound slice passed ten focused repetitions, full HTTP race,
  Routes/bootstrap tests, `go vet ./...`, `go build ./...`, gofmt, staged diff,
  and whitespace checks.
- Resume by closing only routes whose ownership/policy can be proven from an
  immutable Host snapshot with zero request-path Store I/O. Do not credit route
  counts as a completed P6 row until the inherited/custom/raw guard exit gate
  passes as a whole.

## P6 Credential, P7 Provider, And P8 Publication Checkpoint

- Weighted V3 remains **48%**. The active phase is **P6 at 61% (11 of 18
  rows)**; P5 is independently open at **65% (11 of 17)**, P7 is **18% (4 of
  22)**, and P8 is **50% (9 of 18)**. Intermediate production slices below do
  not inflate a row until its complete authoritative exit gate passes.
- `9e1b80c35` authorizes the current inert inbound-webhook skeleton from an
  immutable Host policy. It does not claim webhook-signature verification.
  Contextual Core Guard execution is now **106 of 123**; 17 resource-dependent
  routes plus separately frozen custom/raw request and session authority remain.
- `78db19b98`, `9d540fc8e`, `3516bbf7d`, and `c280bbd25` add Manifest-bound
  typed provider slots, immutable exact-runtime publication, package-closure
  schema binding, fixed typed invocation, bounded deadlines, deep cloning, Host
  request/response validation, and explicit fallback. The P7 provider row stays
  open until a real Plugin B -> Host broker -> Plugin A consumption path and
  lifecycle rollback evidence pass.
- `64164cfb2`, `dd7ba898c`, `da90dbf5b`, `f41815ba2`, and `5abdea62b` make theme
  activation swap provider bindings atomically, expose a process-local runtime
  revision, require exact visible-preview CAS, publish the HTTP/OpenAPI contract,
  and bind the admin confirmation flow. Runtime failure restores the previous
  database theme, runtime, Page publication, and exact approval actor.
- `457dac0a9` independently adds the durable PostgreSQL theme-publication
  ledger, boot-scoped node leases, per-node acknowledgements, commit-time wakeup,
  and database-enforced append-only history. Real PostgreSQL tests covered
  constraints, mutation rejection, acknowledgement transitions, retention,
  history-protected Down, and reapply. PostgreSQL remains authoritative;
  `NOTIFY` is only a wakeup hint.
- Active ownership remains isolated: P6 owns the remaining contextual guards;
  P7 owns the real cross-plugin provider broker fixture; P8 owns publication
  repository/activation integration and the watcher. Resume P8 with repository
  publication in the activation transaction, then LISTEN plus poll/reconnect,
  heartbeat, apply/failed acknowledgements, and two-node convergence tests.
- P5 can resume only after the product freeze for cumulative additive database
  grants, per-runtime lease roles, short-lived Host-signed actor delegation, and
  the provider-neutral entitlement minimum. The recommended choice is all four
  ADR defaults; P5 does not block independent P6-P8 work.

## P6 Identity Target And P8 Approval Authority Checkpoint

- Weighted V3 remains **48%** and P6 remains **61% (11 of 18 rows)**. Guard
  batches are not credited as the inherited/custom/raw row until that complete
  high-risk boundary closes.
- `4850bc999` authorizes five target-dependent identity-admin routes: user
  update, client-IP clearing, permission overrides, role replacement, and
  session revocation. Contextual Core Guard production coverage is now **111 of
  123**; three self-resource routes, four executable bootstrap flows, two
  entity-meta value routes, two attachment reads, and comment creation remain.
- These five low-frequency authorized requests deliberately perform one narrow
  PostgreSQL target lookup. A process-local target-role cache was rejected
  because another API node could grant or revoke `super_admin` without
  invalidating it. Unauthorized requests perform no Store I/O. Two isolated
  PostgreSQL pools proved a grant and revoke are observed by the next guard.
- P6 verification passed focused HTTP twenty times, focused race five times,
  full HTTP/Identity/bootstrap tests, `go vet ./...`, and `go build ./...`.
- `d513aea77` independently extends the theme publication ledger with the exact
  prior Core-replacement approval actor. Database constraints reject approval
  without an actor/source tuple and reject an actor without approval; historical
  rows protect Down. P8 activation/compensation code remains uncommitted until
  all active-theme mutation paths and real PostgreSQL atomicity tests pass.
- Resume P6 with the three identity self-resource routes. P7 owns exact schema-
  validated Plugin B -> Host broker -> Plugin A evidence. P8 next commits the
  activation/compensation repository slice before starting the watcher.

### 2026-07-15 Identity Self-Resource Continuation

- `3bda8e31b` authorizes session revocation, PAT revocation, and PAT rotation
  against current PostgreSQL ownership. Session revocation accepts the existing
  cookie/PAT credential contract; PAT management remains cookie-only. Foreign,
  missing, forged-param/body/query requests fail before side effects.
- Contextual Core Guard coverage is now **114 of 123**. The nine remaining
  routes are four executable bootstrap flows, two entity-meta value routes, two
  attachment reads, and forum comment creation. P6 remains **11 of 18 rows**.
- Two independent PostgreSQL pools proved cross-node ownership freshness;
  focused tests passed twenty repetitions, focused race five repetitions, and
  full HTTP/Identity/API-token/bootstrap, vet, and build gates passed.
- An early version of the P8 real-PostgreSQL test fixture wrote append-only test
  revisions to the configured local development database before isolation was
  reviewed. The current desired revision was restored, but historical test rows
  intentionally remain because destructive cleanup would violate publication
  evidence. Do not delete or mutate them. All new P8 publication tests must run
  in a uniquely migrated schema and drop that schema after the test.

## 2026-07-15 P5 Product-Boundary Freeze

- The operator explicitly approved all four recommended P5 boundaries. P5 is
  no longer product-blocked, but remains **65% (11 of 17 rows)** until the six
  production rows and their exit evidence are implemented. Weighted V3 remains
  **48%**; a decision alone does not earn implementation credit.
- Database powers are additive grants. Legacy `database.authority` expands
  cumulatively through `own_schema`, `core_views`, `host_commands`, `raw_core`,
  and `kernel`; new manifests may select the exact additive set. The normalized
  set is exact-artifact trust authority.
- Direct credentials use exact per-runtime lease roles. Rolling source and
  target leases coexist, and draining one runtime revokes only its lease.
- Actor-scoped Host Commands use short-lived Host-signed delegation created by
  a core route/admin invocation and rechecked by the Host. Background work uses
  explicit actorless service authority.
- The entitlement minimum is provider-neutral: subject, resource/capability,
  active/revoked/expired state, source reference, validity window, idempotency,
  and audit. It deliberately contains no billing or gateway semantics.
- Resume P5 in buildable boundaries: additive manifest compatibility first;
  additive persistence migration independently; runtime lease issuance and
  drain revocation; signed delegation and six concrete Host Commands;
  entitlement persistence; physical raw-core grants and complete real
  PostgreSQL atomicity/compatibility evidence.

## 2026-07-15 P5 Lease, P7 Provider, And P8 Convergence Checkpoint

- Weighted V3 is now **49%**. P5 is **71% (12 of 17 rows)**, P6 is **61%
  (11 of 18)**, P7 is **27% (6 of 22)**, and P8 is **56% (10 of 18)**.
- `c59e60d39` issues one short-lived PostgreSQL role and credential per exact
  runtime instance. Rolling source/target leases overlap, heartbeat and drain
  use CAS revisions, revocation terminates only the selected runtime, plaintext
  secrets are not persisted, and additive `own_schema`, `core_views`, and
  `raw_core` powers remain effective together. Focused repetition, full
  Extensions tests, race, vet, build, and real PostgreSQL tests passed.
- `0f3cd58ca` completes the real Plugin B -> Host broker -> Plugin A Provider
  Slot path. Exact dependency/version identity, request and response schemas,
  timeout, disable, upgrade, rollback, and fallback behavior are exercised
  through two Protocol V2 subprocesses.
- `97a499957`, `a79b04148`, `0b56bb8e3`, and `7962e5127` publish exact theme
  revisions atomically, persist boot-scoped node leases and acknowledgements,
  recover LISTEN disconnects, poll missed notifications, and converge two real
  Service/ThemeRuntime nodes on one artifact. Safe Mode bypass and startup/
  shutdown watcher ownership are Host-controlled.
- P8 still needs complete Page ViewModel acceptance, all-catalog no-I/O proof,
  and crawler/JavaScript-disabled API plus Nitro browser evidence before its
  remaining rows may close. P7 provider selection/reset/probe/health UI remains
  separate from the now-complete provider discovery and cross-plugin call rows.

## 2026-07-15 P5 Provider-Neutral Entitlement Persistence Checkpoint

- Weighted V3 remains **50%** and P5 remains **71% (12 of 17 rows)**. The
  persistence boundary is complete, but the transactional Host Command row is
  not credited until the entitlement command is production-bound through the
  exact-artifact Host API and passes its full atomicity exit tests.
- `cff694d39` adds the provider-neutral entitlement schema and append-only event
  evidence. It models generic subjects, resource or capability scopes,
  `active`/`revoked`/`expired`, generic source references, half-open validity
  windows, global idempotency keys, request fingerprints, actor/audit links,
  and no billing, currency, checkout, gateway, or provider-transaction fields.
- `cdda05554` implements transaction-aware grant, revoke, expire, get, and
  effective checks. Advisory locks serialize identical idempotency keys;
  identical payloads replay while changed action, actor, or payload conflicts.
  `GrantTx`, `RevokeTx`, and `ExpireTx` let Host Commands compose the lifecycle
  inside one caller-owned PostgreSQL transaction.
- Real isolated PostgreSQL tests cover grant/revoke/expire, effective windows,
  same-key replay, changed-payload conflict, eight-way concurrent replay,
  audit-write failure rollback, and outer-transaction rollback. Focused tests,
  vet, and the real PostgreSQL race run passed.
- Next P5 work is the production entitlement Host Command binding, followed by
  the remaining concrete commands, physical exact-artifact raw-core grants, and
  their complete atomicity/compatibility evidence.

## 2026-07-16 P8 Closure And P9 Runtime Checkpoint

- Overall advances to **58%** after flooring. P8 is **18/18 (100%)** and earns
  its full 8% weight. P9 was **1/16 (6%) accepted** at this checkpoint; see the
  newer P9 buildless public L2 production exit above.
- P8 passed the isolated fresh API/Nitro production matrix with three restarts,
  exact theme switching, one concurrent winner, final recovery, all 23 product
  Page ViewModels, JavaScript-disabled catalog output, and no residual process
  or port 3000 interference.
- `d64e9177b` and `57d1c2958` add atomic Component Registry lifecycle restore,
  exact Host package identity, Safe Mode clearing, and one shared production
  registry instance.
- `867bdc6c3` preserves zero-challenge buildless L0/L1 theme activation while
  requiring the actor-bound one-use exact-artifact flow for executable themes.
- `92969a5f8` implements the package-local public L2 browser runtime with
  immutable native ESM/CSS, digest checks, exact leases, cleanup, quarantine,
  SSR fallback, and bounded failure timeouts. Full Web **353/353**, typecheck,
  Page Registry offline validation, and the production build passed.
- Resume from
  `knowledge/sessions/2026-07-16-trusted-plugin-theme-platform-v3-p8-p9-progress.md`.
  Do not enable the production L2 default before the upload-to-revoke E2E,
  Component Registry descriptor admission, and scoped CSP response policy pass.
