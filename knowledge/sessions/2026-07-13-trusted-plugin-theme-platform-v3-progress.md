# Trusted Plugin And Theme Platform V3 Progress Ledger

Last updated: 2026-07-17

## Progress

- Verified weighted progress: **63.2336%** (display **63.2%**).
- Phase counts: P0-P5 and P8 complete; P6 **13/18**, P7 **14/22**,
  P8 **18/18**, P9 **4/16**, P11 **1/16**, and P12 **1/22**. P10 and P13
  have no credited authoritative row yet.
- Completion remains unproven until all 99 target rows, 14 accepted boundaries,
  five reference-plugin classes, 24 Program Definition of Done rows, and final
  gates pass.

## Current Subtask

### 2026-07-17 P6 Bound Replay And Terminal Status Checkpoint

- Verified weighted progress remains **63.2336%** (display **63.2%**); P6
  remains **13/18** because the production Dispatcher mutation and joint route-
  policy snapshot rows are not closed yet.
- `dc3e08b52 fix(idempotency): bind rolling aliases to v3 callers` keeps V1
  compatibility and exact V2 legacy reads, but permits a V2 rolling fingerprint
  alias only for the Bound/V3 migration reader. The deprecated V2 writer can no
  longer expand its record identity through caller-supplied aliases.
- `e430135e5 fix(protocol): reject informational route terminals` rejects every
  Protocol V2 terminal 1xx response except the exact streamed WebSocket 101
  upgrade. Unary terminal, prior-response, and stream-close validation share the
  same rule.
- `d9ab64673 fix(routes): reject informational response terminals` adds the
  Host mode-exact terminal-status contract and applies it to RFC 6901 response
  status mutation. HTTP, multipart, SSE, and ordinary streams require 200-599;
  WebSocket requires exactly 101.
- Idempotency focused compatibility tests passed 50 normal and 10 race
  repetitions; its complete normal/race suites and vet passed. Protocol status
  tests passed 50 normal and 10 race repetitions plus vet. Routes status tests
  passed 100 normal and 20 race repetitions plus vet. Formatting, staged diff
  review, and whitespace checks were clean for all three commits.
- The current dirty Bound adapter is not yet credited. It must switch the
  production HTTP controller from `BeginRequiredReplay` to
  `BeginRequiredReplayBound`, add wrong-key HTTP evidence, and land with the
  complete Routes staged-mutation dependency closure.
- Independent review found that lifecycle writers serialize Route and Route
  Schema publication, but request readers can still observe a route snapshot
  and required-idempotency policy snapshot from different revisions. The
  accepted implementation direction is to freeze exact route policies into the
  immutable Routes Registry snapshot and copy the selected policy into the
  execution plan; a writer-only mutex is not sufficient evidence.
- Exact next order: land the staged Dispatcher/Protocol V2 HTTP bridge in
  buildable slices; add response-only and mutable wrong-key fail-closed tests;
  then add plan-bound policy publication with a 64-reader concurrency matrix.

### 2026-07-17 P6 Plugin Response Authority Checkpoint

- Verified weighted progress remains **63.2336%** (display **63.2%**); P6
  remains **13/18**.
- `f0913227d fix(routes): centralize plugin response header policy` replaces
  the divergent legacy and Registry terminal filters with one Routes-owned
  policy. Both paths now remove `Set-Cookie`, `Link`,
  `Idempotency-Replayed`, every `X-SForum-*` field, `Proxy-Connection`, the
  standard hop-by-hop family, and every header nominated by any
  case-insensitive `Connection` field. `Location` and ordinary plugin metadata
  remain available to complete `add`/`replace` terminal responses.
- The shared contract passed fifty focused repetitions. Legacy RouteGateway,
  production Provider, and new buffered/stream consumers passed twenty focused
  repetitions; all four packages passed focused race five times and vet.
  gofmt, staged review, and diff checks were clean before commit.
- The pending stage/mutation draft in `docs/extensions/host-api-v2.md`
  distinguishes Host-owned mutation fields from complete terminal response
  fields: modifiers cannot patch `Location` or `Link`, terminal `add`/`replace`
  may return `Location`, and every plugin terminal/streaming `Link` remains
  stripped. That mixed draft stays uncommitted until its own contract slice.
- This closes a P6 compatibility/security blocker but does not independently
  earn a row. Remaining blockers for the current batch are the statically legal
  but runtime-unusable required-idempotency plus mutable-request combination,
  direct repeated-query evidence for custom/raw guards and streaming, and the
  still-uncommitted HTTP/Dispatcher action and failure matrices.

### 2026-07-17 P6 Legacy Route Proxy Link Checkpoint

- Verified weighted progress remains **63.2336%** (display **63.2%**); P6
  remains **13/18**.
- `85b8d79f7 fix(extensions): reserve Link on legacy route proxy` closes the
  P13-retained namespaced V1 proxy bypass around the Host-final canonical
  policy. The complete plugin `Link` response header is now removed
  case-insensitively while status, body, and unrelated allowed headers remain
  intact.
- A real loopback RouteGateway test covers canonical, preload, pagination,
  mixed relation, quoted-comma, and multi-value forms. A separate Fiber test
  crosses the production Provider adapter, exact route admission lease, and
  legacy gateway. Both focused tests passed twenty repetitions; both packages
  passed focused race five times, full normal tests, vet, gofmt, and diff
  checks.
- The compatibility audit also found that the legacy and Registry terminal
  filters still diverge for Host session/replay/reserved metadata and dynamic
  hop-by-hop headers. The next independent safety slice must centralize that
  output policy and prove parity before P6 can close; this Link-only checkpoint
  earns no row by itself.

### 2026-07-17 P6 Protocol V2 Guard Failure Checkpoint

- Verified weighted progress remains **63.2336%** (display **63.2%**); P6
  remains **13/18** until the complete custom/raw guard, action, mutation,
  cancellation, and production failure matrices pass.
- `7eed8d0d2 feat(protocol): classify plugin guard call failures` adds a typed,
  redacted post-RPC failure contract for deny, crash, timeout, protocol, and
  cancellation outcomes. Pre-RPC binding/authority/schema rejection cannot
  claim runtime execution, while Manager wrapping preserves the typed evidence
  without retaining plugin-controlled error text.
- Focused Protocol V2 and Manager tests passed twenty repetitions; focused race
  passed five repetitions; `go vet ./app/Support/Extensions`, staged diff
  review, and both diff checks passed before the implementation commit.
- This producer-first compatibility commit does not change the P6 score. The
  next atomic slice maps the typed Protocol evidence into the HTTP guard
  adapter, then the Dispatcher slice must prove request/response-stage caller
  cancellation, idempotency abort/complete boundaries, trace/quarantine
  classification, and custom/raw guard failure matrices before receiving any
  row credit.
- The legacy namespaced RouteGateway `Link` bypass has a separate reviewed
  three-file fix waiting for its own commit. Repeated-query wire production,
  params authority, canonical response policy, Identity role suggestions, and
  the user-owned fixture edits remain intentionally outside this checkpoint.

### 2026-07-17 P6 Bounded Route Mutation Engine Checkpoint

- Verified weighted progress remains **63.2336%** (display **63.2%**); P6
  remains **13/18** until production Dispatcher application, immediate schema
  revalidation, true wrap ordering, and the complete action/failure matrix pass.
- `94fd2f074 feat(protocol): preserve repeated route query values` adds the
  additive Protocol V2 field 17 representation with stable key ordering and
  ordered multi-values while retaining legacy field 8 as the first value. The
  generated descriptor and SDK tests pin both field numbers. Buf lint, the
  repository-relative breaking check, regeneration, SDK normal/race, vet, and
  independent Grok review passed. This earns no P6 row before the production
  HTTP producer sends both representations and the Dispatcher applies them.
- `f9749a12d feat(extensions): constrain route mutation paths` freezes the
  direction-specific synthetic request/response documents, exact RFC 6901
  allowlists, raw-request credential rule, impossible-shape rejection, and
  Host-owned/hop-by-hop header policy in Manifest V3 and OpenAPI.
- `d2a3107db feat(routes): add bounded field mutation engine` adds the
  Host-authoritative RFC 6902 `add`/`replace`/`remove` subset using the mature
  `evanphx/json-patch/v5` library. It rejects undeclared or duplicate paths,
  preserves JSON number precision and null/remove distinction, keeps untouched
  multipart/binary/HTML fields opaque, preserves repeated query values, blocks
  credential and dynamic `Connection` headers without exact raw authority, and
  applies 4 MiB patch, 8 MiB body, 8 MiB + 256 KiB document, and 1 MiB metadata
  budgets before accepting a result.
- Host mutation focused tests passed twenty repetitions; focused race passed
  five repetitions; full Routes and ExtensionManifest normal/race plus both-
  package vet and staged diff checks passed before the commit.
- `f45f28eed feat(routes): bind mutable fields to exact guards` removes the
  Registry's divergent generic-pointer validator. Publication now reuses the
  Manifest policy only after resolving the exact custom guard binding, so only
  `core.guard.raw_request` or an exact package `raw_request` guard can declare
  credential fields; forged status/header shapes and oversized array indices
  fail before the immutable snapshot advances.
- The Registry/Manifest policy slice passed full normal tests, twenty focused
  repetitions, five focused race repetitions, vet, and staged diff checks.
- `a03f1f33e fix(routes): preserve route mutation atomicity` fail-closes
  malformed source header names instead of silently deleting an undeclared
  field, freezes root `remove` as clearing request query/params/body while
  rejecting response status removal, and proves source plus patch numbers above
  `2^53` remain byte-exact. Focused normal tests passed twenty repetitions,
  focused race passed five repetitions, and the full Routes/vet gates passed.
- Active uncommitted slices are intentionally separate: Registry publication
  must reuse the Manifest validator after freezing the exact guard binding;
  Protocol V2 request/response patch mapping and repeated-query wire support
  remain agent-owned; its new required action/stage inputs currently expose the
  still-unwired production HTTP producer and therefore must land together with
  that bridge. Dispatcher application and schema revalidation remain the
  main-thread next step; P7 role suggestions are a separate identity slice.
- Never stage the user-owned `apps/api/app/Models/PageViewModels/source_test.go`
  or `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`.
  Protocol V1 compatibility, Safe Mode, prior snapshots, and CLI recovery stay
  intact; no migration, destructive rollback, push, tag, branch, or worktree
  operation belongs to this checkpoint.

### 2026-07-17 P6 Trust Revocation And Guard Closure Checkpoint

- Verified weighted progress remains **63.2336%** (display **63.2%**); P6
  remains **13/18**. The current safety work does not earn a row until omitted
  target guards inherit the exact Core guard and the real WebSocket revoke /
  reauthorization matrix passes end to end.
- Durable revoke now removes the exact runtime member in the same PostgreSQL
  transaction as grant/challenge revocation, performs bounded COMMIT readback,
  treats SQLSTATE `08007`/`40003` and inconclusive transport failures as typed
  unknown outcomes, and preserves builtin/no-grant-history members.
- The initiating process holds the Manager runtime-set barrier across durable
  revoke, drains old admission, captures GuardPolicy even after TTL expiry,
  and fail-closes runtime, public assets, and current/review/staged policy on
  success or unknown COMMIT. Grant-generation tombstones prevent an old live
  grant from being republished while allowing an explicit new grant id.
- Full-set apply rechecks durable latest after the Manager barrier and rejects
  process regression. The coordinator immediately follows only a genuinely
  newer durable revision; `latest == requested < applied` returns to normal
  poll/backoff instead of writing unbounded failed acknowledgements.
- Filtered route/guard transport strips `Cookie`, `Authorization`,
  `X-API-Key`, and `X-Auth-Token`; dynamic `Connection` tokens are collected
  from every case-insensitive header-map key. Raw authority remains bound to
  the exact frozen artifact/route/guard. Protocol V2 SEO now maps gRPC
  deadline/cancellation status before consulting the asynchronously updated
  caller context.
- The shared exact lifecycle fixture is now a valid Manifest V3 executable
  plugin. Lifecycle-state PostgreSQL tests run in a unique private schema with
  the real lifecycle/runtime migrations and drop it with `CASCADE`; the one
  failed exploratory `state.publication.enable.*` public row was identified,
  deleted, and verified absent. No `lsp_*` schema remains.
- Completed gates: full Models/Extensions normal and race; full
  Support/Extensions; full Http; focused trust/revoke/full-set/coordinator and
  credential normal/race; real PostgreSQL `08007`, COMMIT readback, lifecycle
  state normal/race; Manifest full normal/race; SEO deadline 500 repetitions
  and focused race 100 repetitions; vet and `git diff --check`.
- Pending before the first implementation commit: finish the deterministic
  PostgreSQL advisory-lock waiter proof, finish the real TCP WebSocket
  revoke/R+2 test, run sequential Models/Extensions, Support/Extensions,
  Http, and bootstrap gates with `SFORUM_TEST_DATABASE_URL`, then review and
  stage each contract independently.
- Planned commit order: target-route guard inheritance; Protocol and Host
  credential filtering; lifecycle fixtures; retained-runtime stop; full-set
  and coordinator non-regression; serialized/ambiguous durable trust revoke;
  final live-grant publication check; GuardPolicy capture/tombstones; trust
  service and local runtime fence; SEO cancellation; bootstrap trust and SEO
  bindings; final docs checkpoint.
- Never stage the user-owned
  `apps/api/app/Models/PageViewModels/source_test.go` or
  `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`.
  There is no new migration or destructive rollback. Runtime publication and
  Registry histories remain additive/immutable; rollback is the previous
  process snapshot plus the existing Safe Mode and CLI recovery path.

- **Startup recovery is closed:** migration 034 repaired the legacy role-
  approval schema, exact evidence-bound Identity adoption completed, and the
  real API now remains serving with the embedded worker.
- The final startup failure was correctly detected by the P8 exact-artifact
  Registry fence but caused by a P12 ownership gap: historical `SaveBuiltin`
  advanced enabled builtin plugin `active_version_id` values without advancing
  the immutable runtime full-set. Commit `b2ea70227` makes builtin plugin sync
  publish only that Host-owned exact enable/upgrade under the shared producer
  lock. It preserves unrelated immutable members and never re-adds a missing
  member removed by disable or trust revocation.
- The independent P6 authority/replay slice is now committed through exact
  request-authority transport, remote-execution replay fencing, unsafe `after`
  response preservation, durable audit evidence, and process-local exact
  runtime quarantine. P6 remains 13/18 until the complete action/mutation,
  revoke/WebSocket, canonical SEO, and production-matrix exit rows close.
- The P11 SEO production path now has Host-final policy, exact-runtime provider
  resolution, Protocol V2 execution/SDK transport, lifecycle Registry binding,
  and a real subprocess reference fixture. This is verified groundwork only:
  P11/P13 receive no additional row credit until the complete SEO kinds,
  sitemap/route/query/cache/admin/SSR-without-JavaScript/uninstall and failure
  matrices pass.

## Recent Verified Commits

- `94fd2f074 feat(protocol): preserve repeated route query values`
- `5df41f67e docs(extensions): list SEO manifest family`
- `7237dfc2b feat(seo): bind lifecycle registry runtime`
- `b183c3aee test(seo): add protocol v2 reference plugin`
- `35fccd29a feat(seo): add protocol v2 execution bridge`
- `92bc0c474 feat(seo): resolve exact runtime providers`
- `1b8109127 feat(seo): enforce host final policy`
- `508efeac0 fix(extensions): skip page fence without contributions`
- `457d25047 fix(extensions): retire revoked protocol v1 runtimes`
- `b2ea70227 fix(extensions): publish builtin runtime upgrades`
- `80766cc31 feat(api): bind identity adoption verifier`
- `7fc0fefe8 feat(extensions): restore trusted legacy identity publications`
- `2c8923ecf feat(identity): adopt trusted legacy publications`
- `774358466 test(identity): isolate registry postgres fixtures`
- `1b23f3462 fix(identity): distinguish missing durable publication`
- `6b288b489 feat(extensions): validate stored trust impact`
- `14dea1a29 feat(extensions): publish runtime trust revocation`
- `23682fb91 fix(identity): repair drifted role approval schema`
- `a645ac594 feat(routes): audit and quarantine committed modifier failures`
- `365cd0df6 feat(extensions): quarantine exact runtime incidents`
- `70dd7fb7c feat(routes): preserve unsafe response after modifier failure`
- `b3a521e05 fix(routes): preserve replay lease after observed execution`
- `c7cf50c97 fix(routes): enforce exact request authority end to end`
- `2bebc8cae docs(extensions): record startup recovery`
- `0c4f42c84 test(extensions): isolate postgres integration fixtures`
- `cc4ce473f fix(runtime): converge mixed protocol startup set`
- `1f2c2e81a fix(routes): bind raw authority to exact dispatch`
- `7e68fe2b9 feat(protocol): add route request authority fields`
- `c5c7b089c fix(routes): harden loopback request forwarding`
- `fea430020 fix(extensions): gate runtime publication on migration proof`
- `d20d88097 docs(extensions): record cache SDK closure`
- `ba4ebc50c feat(sdk): harden cache helpers`
- `d76531e48 feat(protocol): expose cache get revisions`
- `fec013ce4 feat(api): bind durable identity registry`
- `deba95e06 feat(identity): publish registry lifecycle snapshots`
- `7a2581d4c feat(identity): persist exact registry publications`
- `ec3a44e80 feat(identity): add registry root publication ledger`
- `05718f61a docs(extensions): record P12 runtime ownership`
- `d46fd3597 feat(api): supervise theme runtime convergence`
- `873e48248 feat(themes): fail closed on runtime lease loss`
- `04b159441 feat(themes): seed durable runtime publication`
- `1cc4c4320 feat(cache): add cross-rpc lock leases`

## Verification

- Exact request authority now binds the plan revision, step index, full
  contribution/artifact/guard, request, prior response, invocation stage, and
  commit observer. HTTP, Protocol V2 unary/guard/stream, raw credential, and
  forged fixture matrices pass full Routes/Http/Extensions tests, focused race,
  and vet.
- Required replay now retains the 24-hour pending lease after any observed
  remote execution, including transport failure and response-schema rejection;
  `Finalize` cannot erase late execution evidence. Pre-dispatch failures still
  abort safely, and completion failure remains pending.
- Unsafe committed `after` failures preserve the exact prior response, stop
  later contributions, emit redacted structured evidence, and complete/replay
  deterministic 2xx responses. Guard/request-schema failures are audit-only;
  only observed transport failures and response-schema failures quarantine the
  exact version/digest/instance.
- Exact runtime quarantine is monotonic, does not wait for runtime-set or
  lifecycle transition locks, preserves existing leases, permits lifecycle
  cleanup, rejects every ordinary acquisition and Resume/rollback, keeps the
  first stable cause, and never falls back to an active replacement.
- The production recorder synchronously closes exact admission and sends audit
  writes through one bounded worker/queue. Queue pressure never skips
  quarantine, canceled request contexts do not discard audit, and shutdown is
  bounded before PostgreSQL/runtime close. Routes, Http, Audit, Extensions, and
  bootstrap full tests, focused race tests, vet, formatting, and staged diff
  checks passed.
- Mixed Protocol startup focused normal/race tests, bootstrap normal/race,
  `go vet ./app/Support/Extensions ./bootstrap`, formatting, and staged diff
  checks passed for `cc4ce473f`.
- A real API launch progressed beyond the original open-lifecycle failure, the
  Protocol V2-without-Lifecycle validation failure, and both exact Protocol V1
  members. Identity adoption then converged `sforum.admin-surface-reference`.
- The next failure, `startup page runtime for sforum.content-policy is not
  exact and available`, was an aggregate Registry-restore label around the P8
  exact page/runtime fence. PostgreSQL showed runtime publication revision 1
  still named old builtin digests while `SyncBuiltins` had advanced three
  active versions. The P8 fence was retained unchanged.
- `b2ea70227` covers normal A-to-B builtin sync, the already-active-B/stale-
  publication-A recovery shape, API/worker concurrency, non-resurrection,
  unrelated third-party preservation, new builtins, declaration-only plugins,
  and no-publication genesis against real PostgreSQL. Focused normal/race,
  full Models/Extensions, and `go build ./...` pass.
- Two controlled real launches on port 18080 reached the Fiber listener and
  embedded worker; `/api/v1/health` and `/api/v1/ready` both returned 200.
  The current local revision 2 has four members and every published version ID
  and package digest exactly matches its active artifact.
- `508efeac0` narrows the page/runtime fence to plugins with real page
  contributions. Backend-only plugins no longer fail Host startup merely
  because their runtime is drained, while a page contributor with a non-exact
  or unavailable runtime still fails closed before page or ThemeRuntime
  publication. Focused normal/race and the full Extensions suite pass.
- A post-`508efeac0` controlled launch on port 18081 reached the Fiber listener
  with the embedded worker; both health endpoints returned 200 before a normal
  signal shutdown.
- The real `sforum-seo-reference` Protocol V2 subprocess built from committed
  source, applied its exact-runtime title filter, preserved the Core document on
  provider failure, and fell back when the runtime stopped. Focused
  `Support/Extensions`, SDK SEO, bootstrap SEO/lifecycle tests and commit
  whitespace checks all pass for the six SEO commits above.
- Read-only DB inspection proved zero open lifecycle operations; the three old
  `publication.integration.*` rows are terminal `cancelled`. Runtime genesis
  revision 1 remains immutable historical evidence; revision 2 is the current
  exact four-member full-set.
- Provider-slot, lifecycle journal, and lifecycle jobs isolation passed against
  a uniquely created and dropped PostgreSQL test database. Lifecycle jobs also
  passed focused normal/race tests; `SFORUM_TEST_DATABASE_URL` is now mandatory.
- `cd apps/api && go test ./sdk/plugin/v2 -count=1` passed.
- `cd apps/api && go test -race ./sdk/plugin/v2 -count=1` passed.
- `cd apps/api && go vet ./sdk/plugin/v2` passed.
- `cd apps/api && go test ./app/Support/HostAPI ./sdk/plugin/v2 -count=1`
  passed.
- `cd apps/api && go test -race ./app/Support/HostAPI ./sdk/plugin/v2 -count=1`
  passed.
- `gofmt -d` for the four staged Cache SDK files produced no diff.
- `git diff --cached --check` passed.
- Independent audit found that a blocked Renew RPC could outlive a 100ms lease,
  allowing the old loader to overlap a replacement owner. The SDK now bounds
  Renew and the post-acquire read by the current lease expiry, independently
  cancels the loader at expiry, and has an auto-expiring two-owner regression.
- Cleanup now refreshes the wire deadline, invalid Acquire responses release a
  returned opaque token, remote error messages are discarded, and conditional
  write conflicts plus lease consumption have focused tests.
- The post-fix normal/race/vet, formatting, and staged-diff checks passed.
  Independent `grok-4.5` review exited successfully with no final blocker; its
  intermediate guesses were checked against the code rather than trusted.
- The P12 gate passed the full `app/Support/Extensions` normal and race suites
  against PostgreSQL, `go vet ./app/Support/Extensions`, Models/Extensions and
  bootstrap tests, `go build ./...`, and focused overlap tests repeated under
  the race detector.
- P6 loopback forwarding now refuses every HTTP redirect in both the Route
  Registry invoker and the legacy namespaced gateway, strips standard and
  `Connection`-named hop-by-hop headers, and keeps browser credentials,
  CSRF material, and Host-reserved headers closed. The complete `app/Http` and
  `app/Support/Extensions` normal suites, focused race tests, both-package vet,
  formatting, and staged diff checks passed for `c5c7b089c`.
- Protocol V2 now has additive, typed `filtered`/`raw` request-authority and
  `host`/`custom`/`raw_request` guard-kind fields on unary/guard and stream-open
  envelopes. Buf lint and the repo-relative breaking check, SDK normal/race,
  vet, descriptor assertions, generated-code review, and staged diff checks
  passed for `7e68fe2b9`. The default `scripts/proto.sh breaking` baseline is
  incorrectly relative to `contracts/proto`; the explicit `../../.git` baseline
  passed.

## Active Hardening Commit

- `f522ff28f feat(routes): stamp authorized raw request steps` was created by a
  read-only audit agent that exceeded its role. It is retained for review rather
  than destructively rewritten, but is not sufficient evidence for P6: its
  private enum is derived from exported step fields after any legacy authorizer
  returns success and is not bound to the exact plan, step, request, artifact,
  or authorizer-issued raw decision. `1f2c2e81a` closes those boundaries without
  rewriting history: only the production typed authorizer can return an opaque
  raw proof; legacy authorizers remain filtered; the Dispatcher seals revision,
  index, full step/artifact/guard, request, stage, and commit identity.
- Stream authorization now occurs once at `Open`, so malformed or cross-origin
  WebSocket requests stop before guard/preflight RPC and a prepared dispatch
  cannot be replayed after trust drift. Full Routes/Http normal tests, full
  Routes race, focused authority/WebSocket Http race, both-package vet,
  formatting, and staged diff checks passed for `1f2c2e81a`.

## Accepted Decisions And Assumptions

- P5 uses additive database grants, per-runtime lease roles, short-lived
  Host-signed actor delegation, and the provider-neutral entitlement minimum.
- P6 uses RFC 6901 mutable-field allowlists; higher-priority `wrap` is outermost;
  unsafe committed `after` failures preserve the response and trigger audit plus
  quarantine; redirects allow only 301/308 and default to 308; raw credentials
  require an exact-artifact `raw_request` grant.
- Cache revisions and lease handles are opaque 64-character hexadecimal Host
  capabilities. SDK diagnostics must never render lease tokens.
- Cache `remember` must use a hard contention deadline, never run a loader
  without the Host lease, double-check after acquisition, renew while loading,
  atomically set-and-release, and preserve the earlier caller cancellation cause.

## Dirty Worktree Ownership

- Never stage these user-owned files:
  - `apps/api/app/Models/PageViewModels/source_test.go`
  - `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`
- The uncommitted SEO family is separate from Cache and includes
  `Support/SEORegistry`, SEO Protocol/SDK/runtime/bootstrap files, the
  `sforum-seo-reference` fixture, and its fixture index entry.
- The P12 migration-proof implementation/tests are committed in `fea430020` and
  are no longer dirty ownership.
- Migration 034 and Identity legacy adoption are committed; never edit the
  already-applied migration 029 or mix later Identity work into P6 authority.
- `docs/extensions/catalogs/manifest-v3.md`, the V3 ADR edit, and every other
  unstaged file remain outside the current commit until independently reviewed.

## Exact Next Steps

1. Finish the exact trust revoke/drain, Protocol V1 removal, and WebSocket
   admission slice with allowed, denied, drift, redirect, invalid-WebSocket,
   and race tests.
2. Implement RFC 6901 mutable-field enforcement and close every route action,
   priority, conflict, timeout, crash, multipart/stream, and unsafe matrix row.
3. Land route alias/redirect 301/308 canonical ownership and SEO only in
   independently reviewed contract/transport/Host-policy/
   bootstrap/reference slices; do not credit the SEO row before SSR, sitemap,
   revoke/failure, and Inspector evidence is production-complete.
4. Add full-set/staged-publication quarantine concurrency coverage. Current
   quarantine is intentionally node/process-local; cross-node or restart
   persistence requires an explicit durable incident/clear contract rather
   than overloading lifecycle publication reasons.

## Rollback, Flags, And Compatibility

- Reverting `ba4ebc50c` removes only Cache convenience helpers; the committed
  Protocol V2 and Host CacheService contracts remain additive and usable
  directly.
- Reverting `fea430020` removes the publication proof fence and its tests but
  does not roll back migration tables, proofs, or runtime publication history.
- Protocol v1 compatibility remains present until P13 removal gates pass.
- Safe Mode remains Host-owned and filters third-party Registry publications.
- No database migration, feature-flag default, legacy deletion, push, tag, PR,
  branch, or worktree change belongs to the current P6 subtask.

## Open Questions

- None for the current P6 boundary. The user accepted all recommended P6
  choices, including exact-artifact raw authority and continued filtering for
  every non-raw route.
