# 2026-07-15 Trusted Plugin And Theme Platform V3 P6-P8 Progress

## Changed

- Weighted V3 is 57%. P5/P6/P7/P8/P9 remain parallel; P6 is 13/18, P7 is
  13/22, P8 is 17/18, and P9 is 1/16.
- P6 now closes the OpenAPI policy row: validated operations derive exact
  security/permission, Host client-IP rate-limit, and optional required-replay
  metadata. Production required replay uses a 24-hour CAS-fenced ledger,
  canonical request fingerprints, exact route/artifact/credential scope,
  current-guard reauthorization, bounded 2xx storage, and fail-closed stream or
  storage behavior.
- Two `extension.view` endpoints expose the immutable extension OpenAPI
  aggregate and generated-client operation catalog with a digest ETag. Their
  stable Host identities bring the reviewed route catalog to 227 routes.
- P6 now has real-process multipart, SSE, WebSocket, disconnect cancellation,
  and bounded backpressure evidence through Fiber, Registry, Dispatcher,
  Manager admission, Protocol V2 gRPC, and the plugin SDK.
- P7 now has an exact-artifact Plugin Command Registry and out-of-band CLI
  execution with namespace/conflict, trust, Safe Mode, admission, and audit
  enforcement.
- Contextual Core Guard production coverage is 114/123. Exact extension trust,
  Options owner policy, public/bootstrap routes, theme assets, Page Registry
  access, public entity metadata, safe custom-role deletion, declared extension
  routes, cookie-bound PAT list/create, and the inert inbound-webhook skeleton
  use immutable Host authority with request-path I/O tests.
- P7 now has production versioned action/filter hooks with exact runtime
  snapshots, deterministic priority, typed contracts, sync/fail policy,
  dependency SemVer, optional fallback, Host revalidation, River delivery, and
  Protocol V2 exact declaration binding.
- P7 Provider Slots now add durable exact-artifact selection/reset, active
  candidate probe, default/selected/stale inspection, runtime availability,
  `next`/`closed` fallback enforcement, lifecycle invalidation, append-only
  events, OpenAPI, generated catalogs, and a bilingual management page.
- P7 jobs now freeze payload and execution policy in exact River rows, enforce
  bounded retry/concurrency, publish plugin schedules dynamically through
  River's safe add/remove API, and hold lifecycle admission through the real
  job insert. Embedded and standalone workers publish the same exact snapshot.
- P7 Admin Surfaces now have immutable exact-runtime publication, typed Protocol
  V2 invocation, active-runtime visibility, reviewed HTTP routes, declaration
  permission filtering, redacted catalog output, and durable pre-call audit.
  Publication swaps cannot redirect an audited call or invalidate an admitted
  old call's frozen output validator. Terminal audit is best-effort after the
  mandatory durable attempt.
- P7 Admin Surfaces now also have complete production consumption across the
  twelve manifest kinds and concrete navigation/dashboard/list/filter/row/bulk/
  form/notice/editor/detail/import/export placements. The independent reference
  plugin passed upload, exact trust, Protocol V2 execution, restart restore,
  actor denial, idempotent mutation, export, desktop, and 390px acceptance.
- Production deployment now enables completed P1 exact-artifact trust by
  default for bare production processes and for both Compose API/worker
  services. Development remains opt-in and an explicit compatibility override
  remains available.
- P8 now has the exact compiled four-level fallback, durable convergence,
  all-catalog zero-I/O proof, and crawler/JavaScript-disabled SSR evidence.
  A self-contained production API/Nitro restart and concurrent exact-activation
  exit is accepted. Exact plugin business-data preservation is also accepted;
  only page-specific production Core ViewModel population remains open.

## Commits

- `de333a405 fix(admin): stabilize admin surface outlet rendering`
- `367731bd7 test(admin): add full surface reference plugin`
- `7f1f34bf0 docs(admin): refresh user surface catalog`
- `5c76cbca8 feat(admin): consume list and user surfaces`
- `3bc8ea6e5 feat(admin): consume placed admin surfaces`
- `51dc43d59 feat(extensions): enable exact trust in production`
- `cd1573d5a feat(themes): preserve plugin business contracts`
- `a30c8f757 fix(openapi): normalize exact manifest comparisons`
- `2344a580c feat(idempotency): add fenced required route replay`
- `dd12b755d feat(openapi): derive host route policies`
- `cfff68b2c feat(routes): index exact OpenAPI policies`
- `3e771b149 fix(themes): preserve exact activation across restarts`
- `ae4ca62fd feat(themes): add exact runtime fallback chain`
- `476ca7aca feat(routes): bind exact extension guard policy`
- `5b96b58f2 feat(routes): authorize static option and public guards`
- `24ed8b4d8 feat(routes): authorize page resolution policies`
- `124c151dc feat(manifest): define versioned hook contracts`
- `77674b2fc feat(hooks): add immutable exact runtime registry`
- `4b3f8b82c feat(protocol): bind exact versioned hook calls`
- `32d32ac72 feat(routes): authorize declared extension route guards`
- `306204f98 feat(routes): authorize cookie-bound API token management`
- `9e1b80c35 feat(routes): authorize inert inbound webhook guard`
- `78db19b98 feat(manifest): define typed provider slot contracts`
- `9d540fc8e feat(providers): publish exact provider slot registry`
- `3516bbf7d fix(manifest): bind provider schemas to package`
- `c280bbd25 feat(providers): invoke exact typed provider slots`
- `64164cfb2 feat(pages): atomically switch theme provider bindings`
- `dd7ba898c feat(themes): expose runtime publication revision`
- `da90dbf5b feat(themes): require exact preview activation`
- `f41815ba2 feat(contracts): define exact theme activation preview`
- `5abdea62b feat(admin): bind theme activation to exact preview`
- `457dac0a9 feat(database): add durable theme publication ledger`
- `d513aea77 feat(database): retain prior theme approval authority`
- `4850bc999 feat(routes): authorize identity admin target guards`
- `3bda8e31b feat(routes): authorize identity self resource guards`
- `0f3cd58ca feat(providers): broker typed provider calls`
- `97a499957 feat(themes): publish exact durable activation revisions`
- `a79b04148 feat(themes): persist node publication acknowledgements`
- `0b56bb8e3 feat(themes): converge runtime publications across nodes`
- `7962e5127 feat(bootstrap): run theme publication watcher`
- `c59e60d39 feat(database): issue exact runtime lease credentials`
- `881e08811 feat(database): add exact provider slot selections`
- `a08f68250 feat(providers): persist exact slot selections`
- `5d9afdd28 feat(providers): honor durable slot selection`
- `26175bbdf feat(worker): bind provider slot selections`
- `ca9574339 feat(providers): inspect durable slot selection`
- `0648ba15b feat(providers): probe exact slot candidate`
- `9913e543c feat(admin): manage provider slot selections`
- `123fac2f9 docs(openapi): manage provider slot selections`
- `2868e1d1e feat(web): manage provider slot fallbacks`
- `e4afa1b16 docs(catalog): publish provider slot management`
- `5fb841180 feat(providers): invalidate lifecycle selections`
- `1416aa121 feat(jobs): declare bounded execution policies`
- `0e81befcf feat(jobs): enforce manifest execution policies`
- `1eba53bd1 feat(schedules): publish dynamic River catalog`
- `814e824b3 feat(schedules): trigger exact plugin jobs`
- `cdcd46979 feat(protocol): open bounded route streams`
- `de3627df8 feat(runtime): open exact route streams`
- `33702960d feat(protocol): authenticate streamed route preflights`
- `90333626d feat(routes): fence non-buffered dispatch`
- `6cc634bf6 feat(routes): stream protocol v2 responses`
- `0f185817f fix(routes): extend streamed route leases`
- `81f5d8208 feat(routes): bridge websocket streams`
- `7f9b83cc7 test(routes): verify real streamed transports`
- `a9b08a412 feat(manifest): define trusted plugin CLI commands`
- `53068aea0 feat(extensions): publish exact plugin commands`
- `1becca5b1 feat(protocol): invoke exact plugin commands`
- `cccdf3512 feat(extensions): execute exact plugin commands`
- `9c446c7b8 feat(cli): run exact trusted plugin commands`
- `3dff78a05 feat(extensions): register immutable admin surfaces`
- `00470c41e feat(extensions): invoke exact admin surfaces`
- `8ee782c78 feat(extensions): expose typed admin surfaces`

## Verification

- P6 batches passed focused repetition, full HTTP/bootstrap or policy race,
  `go vet ./...`, and `go build ./...` before commit.
- The P6 OpenAPI policy runtime passed all Go tests; focused replay, limiter,
  controller, Routes, OpenAPI, Extensions, bootstrap, and localization tests;
  race across the five changed runtime packages; vet; build; 1,858 OpenAPI refs;
  Nuxt typecheck; protobuf/documentation drift; and the 227-route/120-UI/99-row
  catalog gate. The final repository script stopped only on the separately owned
  P7 admin registry path `/extensions/route-providers`.
- Cookie-bound PAT management additionally passed ten focused repetitions,
  full HTTP race, Routes/bootstrap tests, vet, build, and staged diff checks.
- Identity-admin target guards passed focused HTTP twenty times, focused race
  five times, full HTTP/Identity/bootstrap, vet, build, and a real PostgreSQL
  two-pool grant/revoke freshness scenario.
- Identity self-resource guards passed focused HTTP twenty times, focused race
  five times, full HTTP/Identity/API-token/bootstrap, vet, build, and a real
  PostgreSQL two-pool ownership freshness scenario.
- P7 passed full ExtensionManifest and Extensions tests, complete Extensions
  race in 119.901 seconds, targeted rollback races, vet, build, gofmt, and diff
  checks.
- The Provider Slot slice separately passed ten focused repetitions, full
  Extensions including a real Protocol V2 subprocess, Extensions race in 94.66
  seconds, vet, build, and package-bound schema checks.
- Provider management passed full Support/Extensions, focused Models/HTTP/
  Routes/bootstrap, Nuxt typecheck, 1,816 OpenAPI refs, and V3 catalog drift.
  Browser QA is pending because the available Chrome session redirects to login.
- Dynamic jobs/schedules passed focused ExtensionManifest, Jobs, Models,
  HostAPI, Extensions, and bootstrap tests plus race for the four runtime
  packages. OpenAPI validation passed 1,817 references across 44 files.
- P6 streamed transports passed focused Protocol V2 repetition, real
  WebSocket TCP upgrade repetition, full Routes/HTTP/Extensions tests, focused
  race, vet, build, and the reviewed 223-route/119-UI/99-trace-row catalog gate.
  The real plugin subprocess preserved a 1,049,078-byte multipart body, emitted
  two SSE events, negotiated and echoed WebSocket traffic, and released exact
  runtime admission after client disconnect.
- P7 Plugin Command Registry focused, package, CLI, and race gates passed.
  Nested builtin-plugin module gates are not fully green: their `go.sum` files
  still lack Goldmark and go-redis entries and must be repaired and rerun.
- P7 Admin Surfaces passed full Support/Extensions, Extensions HTTP, bootstrap,
  Routes, and Localization packages; focused runtime/HTTP race; vet; build;
  1,841 OpenAPI refs across 45 files; and the 225-route catalog drift gate.
  Deterministic regressions cover both exact-artifact publication races.
- P7 production consumption passed all 320 Web tests, focused outlet tests,
  Nuxt typecheck, and the production build. Browser QA covered editor command,
  export download, ordinary-member 403, hydration, desktop, and 390px detail
  layouts without overflow.
- Production trust deployment passed Go config/bootstrap tests, the static
  deployment drift validator, and merged development plus production Compose
  parsing; production resolves the gate to `true` with a five-minute TTL in
  both API and worker services.
- P8 fallback passed focused Pages/Extensions/Controller tests, race, vet,
  build, and ThemeCompiler allocation checks before commit.
- P8 exact-preview backend, OpenAPI, admin typecheck/locales, and the additive
  publication migration passed focused gates. Migration `020` additionally
  passed real PostgreSQL Up/Down/reapply and immutable-history scenarios.
- P8 publication passed isolated PostgreSQL activation, compensation,
  concurrency, node lease/ack/retry, LISTEN reconnect, and two-node convergence;
  full Go tests, focused race, vet, and build also passed.
- P8 production restart passed the fresh-build real API/Nitro E2E in 203.78
  seconds, including exact switch persistence, one-winner concurrent CAS, and a
  second full restart. Browser QA confirmed the Nocturne artifact/skin and menu.
- P8 plugin business-contract preservation passed twenty focused repetitions,
  all affected packages, Extensions and targeted ThemeCompiler race gates,
  allocation budgets, vet, build, and `./scripts/test.sh` before `cd1573d5a`.
- P5 runtime leases passed real PostgreSQL source/target overlap, heartbeat,
  drain, exact session termination, retained target access, core-view-only
  login, additive raw-core access, focused repetition, full Extensions tests,
  race, vet, and build.

## Decisions

- The operator approved all recommended P5 defaults on 2026-07-15: cumulative
  legacy-to-additive database grants, per-runtime lease roles, short-lived
  Host-signed actor delegation, and a provider-neutral entitlement minimum.
  P5 implementation may resume without another product confirmation.
- Do not credit individual guard batches as the inherited/custom/raw guard row;
  custom and raw request/session authority remain a separately confirmed
  high-risk boundary.
- Async hooks support fail-open only. A fail-closed chain cannot be honest after
  an earlier listener job has already been durably enqueued.
- Hook payloads/results/patches are deep-cloned as JSON documents, and every
  authoritative filter patch is revalidated by the Host.
- Ordinary theme managers may activate only an exact visibly previewed L0/add
  artifact. Core replace approval remains explicit and super-admin-only.

## Active Dirty Ownership

- P5 owns only the dirty database runtime-lease/kernel-recovery and migrator
  files. Its two P1 exits are durable two-stage credential revocation and
  migration-time reconciliation of legitimate ledger-backed kernel owners.
- P7 Admin Surface production consumption is committed at `de333a405`; Query,
  Identity/Permission, Auth/Profile, automation authority, and generated SDK
  rows remain.
- P8 now owns only page-specific Core ViewModel production population. P9 owns
  the separate Asset Registry/public L2 slice.
- Do not stage the content-policy manifest, `.playwright-cli/`, or
  `.playwright-p8-nojs.json` with any of these slices.

## Next

1. Finish P5 two-stage durable revoke and physical kernel-owner reconciliation.
   Run the real non-super PostgreSQL restart/overlap/pending/forged/drift matrix,
   then commit those two contracts separately.
2. Resume P6 action/SEO and remaining failure-matrix work, and continue P9
   Navigation/Region plus Component Registry on non-overlapping files.
3. Populate all production page-specific P8 Core ViewModels while P9 proceeds.

## P5 Entitlement Persistence Continuation

- `cff694d39` adds the additive provider-neutral entitlement migration with
  retained lifecycle evidence and fail-closed rollback after facts exist.
- `cdda05554` adds transaction-aware grant/revoke/expire/effective behavior,
  exact idempotent replay, audit coupling, and isolated real PostgreSQL tests.
- P5 stays **12/17 (71%)**: persistence is ready, but the entitlement Host
  Command remains owned by the active HostAPI implementation and must be wired
  before the corresponding authoritative row can close.

## Open Questions

- Awaiting user freeze: RFC 6901 mutable-field paths, high-priority wrap order,
  after fail-closed semantics, and redirect/canonical policy for remaining P6
  actions.
- An early P8 PostgreSQL fixture wrote append-only test revisions into the local
  configured database. Desired state was restored; historical rows must remain.
  Resume only with uniquely migrated per-test schemas, never destructive cleanup.
