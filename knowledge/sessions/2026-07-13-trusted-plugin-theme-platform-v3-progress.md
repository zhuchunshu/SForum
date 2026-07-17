# Trusted Plugin And Theme Platform V3 Progress Ledger

Last updated: 2026-07-18

## Progress

- Verified weighted progress: **64.3447%** (display **64.3%**).
- Phase counts: P0-P5 and P8 complete; P6 **15/18**, P7 **14/22**,
  P8 **18/18**, P9 **4/16**, P11 **1/16**, and P12 **1/22**. P10 and P13
  have no credited authoritative row yet.
- Completion remains unproven until all 99 target rows, 14 accepted boundaries,
  five reference-plugin classes, 24 Program Definition of Done rows, and final
  gates pass.

## Current Subtask

### 2026-07-18 Custom/Raw Guard Production-Chain Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**); P6 stays
  **15/18**. Stream lifetime is closed; custom/raw guard **production-chain**
  evidence is now committed, but the custom/raw row is still not credited until
  the joined full P6 behavior matrix and non-HTTP Schema freeze land with it.
- `1fc9226a1 test(routes): cover custom and raw guard production chain` proves:
  1. Fiber + Registry + `ProductionRouteGuardAuthorizer` +
     `RuntimePluginRouteGuardEvaluator` + real Protocol V2 go-plugin subprocess
     for declared `custom` allow/deny and `raw_request` credential forwarding.
  2. Trust revoke (`CurrentArtifactTrusted=false`) fail-closed: no further plugin
     invoke for custom or raw; status BadGateway; admission ActiveTotal 0.
  3. WebSocket custom guard runs only at Open preflight (deny → 403, no lease);
     post-upgrade multi-message traffic does not increase Protocol CallCount;
     trust revoke after open blocks new handshakes without further invokes.
  4. Legacy `HostRouteGuardAuthorizer` on Fiber cannot mint raw for
     `core.guard.raw_request` (403 Forbidden, CallCount unchanged).
- Focused gates: production-chain tests **20** normal + **10** race; `go vet`
  `./app/Http`; `git diff --check`. No sleep/assertion weakening.
- Stream lifetime closure commits (prior checkpoint): `26493c35a`/`6c95b748e`/
  `740962396`/`595dad2b1`/`fd05b0816`/`280a0d31b`/`093a39e2b`/`8be353344`.
- Exact resume point: join and close the **full P6 behavior matrix** across every
  action, priority/conflict, locale/query/body, permission/CSRF, custom guard,
  stream, disconnect, timeout, crash, multipart, and unsafe committed response
  (prefer extending `route_matrix_test.go`, `route_request_authority_matrix_test.go`,
  `route_failure_matrix_test.go`). Then resolve non-HTTP Schema product option
  (opaque / mode envelopes / JSON stream) before any framing implementation.
  Do **not** raise P6 above 15/18 until matrix + Schema product freeze are
  production-proven together with this guard evidence.


### 2026-07-18 Stream Lifetime Closure Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**); P6 stays
  **15/18**. Stream total budget and lifecycle ownership are now committed end
  to end, including the invalid WebSocket preflight Fail-before-Cancel order.
- Lifetime ownership commits:
  - `26493c35a fix(routes): own stream budget and cancel lifetime`
  - `6c95b748e test(routes): cover stream lifetime budget and cancel races`
  - `740962396 fix(routes): publish stream traces before lifetime Done`
  - `595dad2b1 test(routes): assert stream traces precede lifetime Done`
  - `fd05b0816 docs(routes): record stream Done-before-trace fix`
  - `280a0d31b fix(routes): fail WebSocket preflight before lifetime Cancel`
  - `093a39e2b test(routes): assert WebSocket preflight fails before Done`
- Contract held: one Host total budget covers guard, unary preflight, stream
  open, and the full session (`TimeoutMS == 0` → 24h). Outer lifetime owns
  budget timer, caller callback, and WebSocket detach; inner
  `routeV2StreamSession` owns wire cancel and lease release. Active / terminal /
  canceled race is atomic; only terminal publishes Response; cancel preserves
  typed cause; lease cause is captured before `RuntimeAdmissionLease.Release()`.
  Adapters Fail/Complete/StreamFailed before Cancel so traces precede Done.
  Invalid WebSocket preflight now matches Upgrade and detach-error order.
- Focused gates after the preflight order fix: Routes `TestStreamDispatcher`
  **100** normal + **20** race; Http `TestRouteV2Stream|TestRouteDispatcher.*WebSocket`
  **100** normal + **20** race; Fiber real Protocol V2 stream race **10**;
  complete `./app/Support/Routes ./app/Http` normal + race; both-package vet;
  `go build ./...`; `git diff --check` on staged files. No sleep or weakened
  assertions.
- Still open for the streamed-transport row: non-HTTP Schema framing/validation
  product freeze, durable incident source where still open, and the joined full
  P6 behavior matrix. Custom/raw guard production-chain evidence remains open
  and is the immediate next task.
- Exact resume point: audit and close custom/raw guard **production-chain**
  evidence (Fiber + real Protocol V2 guard invoke + trust revoke + raw
  credential boundary + stream Open-only guard), then the complete P6 behavior
  matrix. Do not claim non-HTTP Schema complete until SSE/WebSocket/multipart/
  arbitrary stream JSON framing and Host validation are frozen across manifest,
  Protocol V2, Host, docs, and tests (`DataChunk` remains raw bytes only).


### 2026-07-18 Stream Lifetime Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**); P6 stays
  **15/18**. Stream total budget and lifecycle ownership are committed, but the
  streamed-transport row still needs non-HTTP Schema framing/validation, durable
  incident source closure where still open, and the joined full-route behavior
  matrix. Custom/raw guard production-chain evidence is also still open.
- `26493c35a fix(routes): own stream budget and cancel lifetime` adds one Host
  total budget over guard, unary preflight, stream open, and the full session
  (`TimeoutMS == 0` → 24h). Outer lifetime owns budget timer, caller callback,
  and WebSocket detach; inner `routeV2StreamSession` owns wire cancel and lease
  release with active/terminal/canceled atomic race, cause capture before
  `RuntimeAdmissionLease.Release()`, Fail-before-Cancel adapter order, and
  typed-cause preservation on outer wait.
- `6c95b748e test(routes): cover stream lifetime budget and cancel races` covers
  shared budget, pre/post-execution caller cancel, Host budget timeout,
  ForceCancel cause, terminal-vs-cancel race, DetachCaller independence, and
  outer typed-cause preservation.
- `740962396 fix(routes): publish stream traces before lifetime Done` stops
  outer Recv/budget watch from closing Done. HTTP `streamRouteResponse` always
  Complete/StreamFailed then Cancel (defer), so commit/fail traces land before
  session completion is visible. WebSocket already used that adapter order.
- `595dad2b1 test(routes): assert stream traces precede lifetime Done` probes
  commit and transport-fail traces while Done is still open on the real
  `streamRouteResponse` path.
- Focused gates: Routes stream tests **100** normal + **20** race; Http
  stream/WebSocket/response-order **100** normal + **20** race;
  `TestRouteStreamAcrossFiberManagerAndRealProtocolV2Process` **10** race;
  complete `./app/Support/Routes ./app/Http` normal + race; both-package vet;
  `go build ./...`; `git diff --check`. No sleep/assertion-weakening workarounds.
- Exact resume point: audit and close custom/raw guard **production-chain**
  evidence (Fiber + real Protocol V2 guard invoke + trust revoke + raw
  credential boundary), then the complete P6 behavior matrix. Do not claim
  non-HTTP Schema complete until SSE/WebSocket/multipart/arbitrary stream JSON
  framing and Host validation are frozen across manifest, Protocol V2, Host,
  docs, and tests (current `DataChunk` is raw bytes only).

### 2026-07-18 Stream Correlation Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**); P6 stays
  **15/18** because total lifetime, non-HTTP Schema, incident, and joined matrix
  evidence are still open.
- `4c582422b fix(routes): bind stream preflight correlation` restricts stream
  invocation to exact handler-stage `add`/`replace`, validates mode-specific
  status before opening the wire, preserves repeated query order and empty
  values, and binds unary preflight plus `StreamRoute` to one fresh correlation.
  The real go-plugin subprocess now rejects an empty or mismatched trace id.
- A standalone five-file clone passed complete Extensions/Routes/Http normal
  tests, Routes/Http race, five real-subprocess repetitions under race, three-
  package vet, `go build ./...`, and diff checks.
- Exact resume point: add one Host-owned total stream budget and one session
  lifetime owner that releases on normal EOF, caller cancellation, budget,
  ForceCancel, and WebSocket completion without retaining a 24-hour timer or
  racing post-upgrade caller detachment.

### 2026-07-18 Route Redirect Canonical Checkpoint

- Verified weighted progress advances to **64.3447%** (display **64.3%**);
  P6 is **15/18** after the route alias/redirect SEO row closed.
- `a4cdb6764 feat(routes): bind redirects to host canonical` binds redirects
  to the exact materialized Host path in structured `CanonicalPath`. The Fiber
  writer remains the single canonical `Link` generator, so plugin headers and
  replay payloads cannot become a second authority source. Alias uses its Core
  target while rewrite retains the public source path.
- `07f5311b7 test(routes): cover redirect canonical lifecycle` joins stable-ID
  and literal destinations, 301/308, Unicode escaping, GET/HEAD, source-query
  exclusion, Host-only output, and Safe Mode removal. The existing Nuxt proxy
  test proves 301/308 `Location` and canonical `Link` survive same-origin proxying.
- Focused coverage passed **100** normal and **20** race repetitions. A standalone
  clone containing only the staged evidence patch passed complete Routes/Http
  normal and race suites, two-package vet, `go build ./...`, and diff checks.
  Sitemap/SEO Registry consumers remain P11 work and are not claimed here.
- Exact resume point: repair Stream V2 total budget and lifecycle-owned session
  release without breaking the 24-hour compatibility default, ForceCancel, or
  WebSocket detach. Then close custom/raw guard production evidence and the
  complete P6 behavior matrix.

### 2026-07-18 Host-Owned Link Response Authority Checkpoint

- Verified weighted progress advances to **63.7892%** (display **63.8%**);
  P6 is **14/18** after the Schema/explicit mutable-field row reclosed.
- `00c627301 fix(routes): reserve link response authority` rejects
  `/headers/link` in Manifest V3 and runtime response-patch allowlists. Plugin
  terminal responses over legacy HTTP and Protocol V2 lose every `Link`
  relation, including canonical, preload, and pagination, while a Core route may
  still emit Host-owned relations. OpenAPI now states that both `Location` and
  `Link` are outside plugin response-mutation authority.
- Focused Manifest/Routes/Http tests passed **100/100/50** repetitions and the
  joined race gate passed five repetitions. A standalone clone containing only
  the six-file staged patch passed complete Manifest/Routes/Http normal and race
  suites, three-package vet, `go build ./...`, OpenAPI validation (**1932 refs / 49
  files**), and diff checks. Redirect canonical, status-code, query, and Host API
  documentation drafts remained unstaged.
- Exact resume point: repair Stream V2 total budget and lifecycle-owned session
  cancellation without breaking the 24-hour compatibility default or WebSocket
  post-upgrade boundary. In parallel, revise the rejected custom-guard and P7
  candidates from their independent audits, then close redirect SEO and the full
  P6 behavior matrix.

### 2026-07-18 Response Cancellation And Credit Audit Checkpoint

- Verified weighted progress is **63.2336%** (display **63.2%**). The P7
  Host-owned role-mapping row returns to open until exact decision evidence and
  dirty-draft fencing land. P6 returns to **13/18**: the streamed-transport row
  remains open for one total budget, lease-owned cancellation, non-HTTP Schema,
  durable incident source, and real subprocess correlation; the mutable-field
  row remains open until a response modifier cannot reintroduce Host-owned
  canonical `Link` metadata. The task book checkboxes now match this strict
  production exit rather than provisional unit evidence.
- `19e6ef357 fix(routes): complete replay after caller cancellation` preserves
  the last valid response when its caller disconnects during response-stage
  guard, request/response Schema, plugin transport, patch validation, final
  validation, incident persistence, or required-replay completion. Remaining
  modifiers stop, final Schema/audit/replay work uses a bounded detached
  context, and an invalid schema-less mutation rolls back to its last validated
  checkpoint before persistence.
- Caller cancellation does not create a runtime incident. A runtime-owned
  cancel/timeout while the parent remains active, or a distinguishable crash
  concurrent with caller cancellation, still records the exact incident. When
  the parent and transport return the same cancellation sentinel concurrently,
  caller cause wins: Protocol V2 and legacy HTTP erase the original transport
  source at that boundary, and treating it as a runtime fault would permit false
  quarantine. Exact attribution would require a future cross-layer typed
  failure provenance or a non-quarantine ambiguous audit event.
- Focused response/guard tests passed **100** normal and **20** race
  repetitions. Real required-replay backend CAS tests cover cancellation before,
  during, and after apply. Complete Routes/Http normal and race suites, two-package
  vet, and `go build ./...` passed both in the shared worktree and in a standalone
  clone containing only the staged patch. The independent redirect canonical
  hunk remained unstaged.
- Exact resume point: close the Host-owned `Link` mutable-field gap, then repair
  Stream V2 without losing lifecycle `ForceCancel`, the 24-hour zero-timeout
  compatibility default, or the WebSocket post-upgrade detach boundary. Review
  the isolated custom-guard and P7 role-suggestion candidates before copying any
  hunk, then close redirect SEO and the complete P6 behavior matrix.

### 2026-07-18 P7 Role Suggestion UI Checkpoint

- This provisional **64.7992%** / P7 **15/22** checkpoint was superseded by the
  response-cancellation credit audit above. The backend boundary remains valid,
  but the row stays open until the exact-evidence and dirty-draft UI defects are
  fixed and reverified.
- `4adcba492` adds exact-CAS approve/reject/apply review to the existing roles
  administration screen. Install/enable cannot grant permissions; incomplete
  evidence, denial, stale artifacts, revision conflicts, and missing targets
  cannot emit success or silently retry. Refresh preserves unrelated dirty role
  fields and prevents a later draft save from removing the newly approved key.
- Focused Web tests passed **14/14** with 69 assertions, Nuxt typecheck passed,
  Identity normal/race gates passed, and an isolated clean-HEAD plus staged-only
  Web gate passed. Authenticated Chrome covered the real filter/template
  interactions, eight-second stability, overlay absence, and zero fresh console
  warning/error output after replacing empty select values with UI-only sentinels.
- Exact resume point: close P6 Core execution/cancellation and stream lifetime
  blockers, then continue the remaining P7 Query/Identity/Auth surfaces and P9
  public component policy without crediting partial drafts.

### 2026-07-18 Core Execution Fence Checkpoint

- Verified weighted progress remains **64.7992%** (display **64.7%**); P6 stays
  **15/18** because this closes a retry-safety defect inside already-credited
  unsafe route and required-replay behavior.
- `c685a875c fix(routes): fence observed core execution` gives direct Core and
  `readonly_core` fallback calls one shared commit-evidence boundary. A context
  canceled before delivery remains pristine and may abort its unused replay
  lease. Once Core delivery can no longer be disproved, side-effect evidence is
  monotonic; a successful captured response advances response evidence.
- Unsafe POST alias/rewrite tests prove successful replay does not invoke Core
  twice, Core error/500 and cancellation after delivery leave the exact replay
  pending, and a retry returns in-progress without another Core call. Focused
  tests passed 50 repetitions and 10 race repetitions. An isolated exact-index
  clone passed the complete Routes normal/race suites, Routes vet, and
  `go build ./...`; the independent canonical redirect hunk remained unstaged.
- Exact resume point: preserve and complete an already-valid response when the
  caller cancels during response-stage authorization/plugin/schema handling or
  replay completion, without misclassifying the caller as a runtime incident.
  Then close Stream V2 total budget, automatic lease release, non-HTTP schema,
  incident source, and real subprocess correlation.

### 2026-07-18 Required Replay Response Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**); P6 stays
  **15/18** because this hardens an already-credited required-idempotency path.
- `036cfc4c8` applies current response-header policy and final response Schema
  validation before a stored response can leave the Host. `60d16ae88` records
  the effective response contract rather than guessing the last declaration in
  the plan, including the case where a later modifier rejects its input before
  runtime invocation.
- New encrypted replay payloads use a versioned AAD domain and carry bounded,
  exact step/stage/route/contract/schema provenance. Existing payloads remain
  readable. Drifted or forged provenance, Host replay metadata in validation,
  schema-less invalid mutation, and legacy header injection all fail closed.
- Full Idempotency/Routes/Http tests, focused race, three-package vet, 50
  repetitions for the initial response-policy slice, and isolated clean-HEAD
  staged-patch normal/race/vet gates passed. Exact resume point: close unsafe
  Core execution observation and post-response cancellation, then the Stream V2
  total deadline, automatic lease release, non-HTTP schema boundary, durable
  incident source, and real subprocess correlation evidence.

### 2026-07-18 Extension Settings One-Request Bootstrap Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**). This is
  a committed performance/correctness follow-up for the already-credited
  extension settings surface and earns no new authoritative row yet.
- `ec97c1d3a feat(extensions): add admin page bootstrap` adds
  `GET /api/v1/admin/extensions/:id/page-bootstrap?path=<manifest-page>` and a
  stable route identity `core.route.extensions.page_bootstrap@1`. `Store.Get`
  runs once; only a manifest declaration whose `view` is `settings` reads and
  returns localized masked settings; about/unknown pages return explicit nulls;
  URL text never implies page type; GET does not start extension runtime code.
  Metadata requires `extension.view`, while matching plugin/theme/mail settings
  managers may configure their declared settings page without an accidental
  reverse dependency on `extension.view`.
- `12cab0dc0 feat(openapi): document extension page bootstrap` adds the modular
  path and nullable response contract. Exact patch staging excluded the
  independent Host-owned `Link` description hunk. `bf2516e22 feat(web): consume
  extension page bootstrap` replaces the detail-then-settings waterfall with
  one lazy request. A request-start key binds extension, path, and locale so
  Nuxt reactive-key seeding cannot display or write a previous extension's
  settings while a new request is pending. The list cache may paint the title
  and declared shell immediately; the Host bootstrap remains authoritative.
- Authenticated Chrome desktop evidence reported the page's own warm metric at
  about **217ms** from the default-theme list to the first field (previously
  about **897ms**), **826ms** for SMTP settings (previously about **1.30s**),
  and **314ms** for SMTP-to-theme tab switching. The switched page contained
  one theme field and zero stale SMTP fields. Warm full reload was about
  **1.39s**; a **5.54s** cold reload followed an HMR compile and is not a stable
  production measurement. Theme and SMTP forms, about, unknown-page null state,
  tabs, unchanged-value save plus success toast, overlay absence, and fresh
  console warning/error absence passed. The Browser Chrome backend ignored its
  requested 390x844 override and stayed 1920px wide, so authenticated mobile
  visual evidence remains explicitly unproven rather than being claimed.
- API health was about **3-7ms** and unauthenticated bootstrap rejection about
  **5.6ms**. Four leftover Air watchers were competing for port 8081; three
  Codex-owned duplicates were stopped and one healthy watcher retained. This
  explains the earlier 2.9s/502 restart samples and prevents further duplicate
  hot-reload churn without touching the user's port 3000 frontend.
- Focused Extensions model/controller tests, complete Models/Controller/Http/
  Routes tests, relevant race tests, four-package vet, Bun settings/prebuilt
  tests (**13/13**), Nuxt typecheck and production build, OpenAPI validation
  (**1932 refs / 49 files**), and diff checks passed from a clean
  `bf2516e22` archive. A full-history isolated clone exposed the one remaining
  gate defect: the P0 validator still expected 233 routes after the new route
  was generated. `6bf02611f test(extensions): track page bootstrap route count`
  updates that reviewed invariant; the complete validator now passes with
  **234 routes, 123 UI surfaces, and 99 traceability rows**.
- The first full-history catalog attempt incorrectly pointed an archive at the
  repository `GIT_DIR`; the validator's intentional temporary commit then
  changed local Git metadata and created `c1d0564bc`. Configuration was
  restored, and `a6936de2f` explicitly reverted the fixture commit to the exact
  `60d16ae88` tree without reset, checkout, clean, or loss of existing dirty
  work. Future catalog isolation must use a standalone local clone and must not
  inherit the source repository's `GIT_DIR`/`GIT_WORK_TREE`.
- Independent reviews found no remaining page-bootstrap blocker and confirmed
  the exact dirty-worktree split. The settings checkpoint is fully verified;
  the exact resume point is P6 unsafe Core execution observation and
  post-response cancellation, followed by Stream V2 total-budget, automatic
  lease release, typed schema, incident-source, and real subprocess correlation
  evidence. Mobile/no-JS frontend gates remain required at the later shared/P13
  browser gate.

### 2026-07-17 P6 Stream Evidence Commit

- Verified weighted progress remains **64.3447%** (display **64.3%**); P6 stays
  **15/18** because this corrects production evidence inside already-credited
  streamed transport work rather than closing a remaining authoritative row.
- `78ecad557 fix(routes): preserve stream execution evidence` landed as an
  isolated four-file commit. Exact immutable terminal selection, `add`/`replace`
  fencing, composed-plan rejection, custom/raw guard failure classification,
  mode-exact status checks, pristine pre-admission cancellation, and observed-
  execution cancellation evidence are now joined to the Protocol V2 preflight
  adapter in one buildable dependency set.
- A clean-HEAD archive passed full Routes/Http normal and race suites plus vet.
  Focused cancellation tests passed 50 ordinary and 10 race repetitions. The
  real index retained only the separate P7 role-suggestion candidate after the
  commit; user-owned fixture files were not staged.
- Exact resume point: finish the isolated required-replay response-policy review,
  then close stream preflight timeout/schema evidence and review the real
  WebSocket trust-revoke test. In parallel, remediate and retest the P7 role-
  suggestion UI and continue the P9 public frontend policy slice. Do not credit
  progress until a complete authoritative row passes its production exit.

### 2026-07-17 Extension Settings Performance Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**). This is
  a user-visible correctness/performance repair for already-credited extension
  administration and does not close another authoritative row.
- `5b4b147ec feat(extensions): add exact catalog filtering` gives direct detail
  loads an exact `GET /api/v1/admin/extensions?id=...` path backed by
  `Store.Get`, rather than loading and decorating the complete catalog.
- `6ed44a86c fix(web): speed extension settings rendering` reuses the existing
  `admin-extensions` item when an operator enters from the plugin/theme list,
  falls back to the exact endpoint for direct loads, keeps Settings Document
  client navigation lazy with an explicit loading state, uses shallow async
  data, and removes the unconditional mounted refresh that unmounted and
  remounted the just-hydrated form.
- The page deliberately does not infer `view=settings` from the URL or issue an
  unconditional parallel settings request: Manifest V3 permits arbitrary admin
  page paths, including non-menu pages, so the exact extension declaration
  remains the authority. A future one-request page-bootstrap endpoint is the
  safe route if this remaining dependency ever needs removal.
- Focused Bun settings/prebuilt suites passed **9/9**, Nuxt typecheck and diff
  checks passed, and authenticated Chrome verification covered direct plugin
  load, theme-list navigation, tab interaction, full form rendering, and new
  console warnings/errors. Warm theme rendering reported about **835ms**; the
  SMTP direct reload returned in about **1.66s** on the Nuxt dev server. Cold
  Vite compilation remains development tooling cost rather than a slow Go
  handler; independent service measurements put exact detail at about 4-5ms
  and settings at about 5-29ms.
- Reverting `6ed44a86c` restores the shared full-catalog page load and mounted
  refresh without changing extension state, settings, trust, migrations, or
  package artifacts.

### 2026-07-17 P6 Bound Mutable Replay Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**). This
  completes the production dependency of already-credited route mutation and
  idempotency work; it does not independently close one of the three remaining
  P6 rows.
- `091b3632b feat(routes): bind mutable requests to required replay` switches
  production required-route idempotency to the encrypted Bound/V3 store. The
  fingerprint binds the immutable plan, frozen execution policy, exact artifact
  semantics, ordered query values, content type, body, and original request
  digest while deliberately excluding live credentials and process-local
  runtime instance ids.
- Unsafe HTTP request modifiers now produce a bounded encrypted transcript for
  every request stage. Replay evaluates current guards and request schemas,
  reapplies only Host-validated RFC 6902 operations under current allowlists,
  and never invokes a modifier or terminal plugin a second time. Credential-
  mutating plans, missing/wrong ciphers, permission revocation, transcript/
  schema drift, malformed queries, oversized metadata, and aggregate transcript
  overflow fail closed before returning a stored response or invoking the
  handler.
- Same-artifact runtime restart remains replay-compatible; response-only V1
  records remain readable for a single-step, single-valued-query plan. V1
  mutable records fail closed, and reordered repeated query values no longer
  borrow a legacy sorted fingerprint.
- Focused Routes/HTTP/Idempotency tests passed five repetitions; focused Routes
  and HTTP race tests passed three repetitions. A clean `git archive HEAD` plus
  only the 12-file replay patch at `/tmp/sforum-bound-replay.ddoIZq` passed the
  complete Idempotency, Routes, HTTP, and bootstrap suites, four-package vet,
  and `go build ./...`.
- The untracked `required_replay_publication*.go` pair is a dead duplicate of
  the committed immutable policy binder and must not be staged. Stream,
  WebSocket, canonical redirect, Identity UI, and bootstrap cleanup drafts were
  excluded and preserved.

### 2026-07-17 P4 Disabled Missing-Package Recovery Checkpoint

- `b6a93e959 fix(extensions): allow disabled missing-package recovery` restores
  out-of-band boot recovery when an operator has disabled an uploaded extension
  whose retained executable package is now missing or drifted. The disabled
  artifact is omitted from the immutable Guard Policy Catalog and receives no
  request authority.
- The same package error still fails closed for an enabled extension, and an
  unrelated trust/database source failure still aborts refresh even when the
  extension is disabled. Focused tests passed 20 repetitions, race tests passed
  5 repetitions, and the complete Models/Extensions suite plus vet passed.
- This is a P4 compatibility correction and does not add a newly credited V3
  row. Reverting the commit restores the stricter startup failure without
  changing stored extensions, grants, migrations, or package files.

### 2026-07-17 P6 Lifecycle Route Policy Publication Checkpoint

- Verified weighted progress remains **64.3447%** (display **64.3%**); this
  closes a correctness dependency of already-credited P6 route work and does
  not independently earn another authoritative row.
- `b6eb63afd feat(extensions): bind lifecycle route policies` serializes startup
  and lifecycle Registry publication, binds every Route policy from the live
  Route Schema snapshot, rebuilds bindings on each CAS retry, freezes the exact
  runtime identity, compensates Schema publication when Route publication
  fails, and keeps Safe Mode policy authority nil.
- Composed lifecycle tests cross real Manager admission for startup, enable,
  upgrade, true rollback, disable, and uninstall. Required replay proves
  `host.ip_write@1` plus `required.24h@1`; concurrency tests cover unrelated CAS
  writers, live-schema rebinding, cancellation after lock acquisition, and
  failure compensation.
- Focused lifecycle tests, the complete Extensions normal/race suites,
  Extensions vet, and two `git archive HEAD` clean-source verification runs
  passed before commit. The implementation has no dependency on the separate
  uncommitted HTTP replay or stream drafts.

### 2026-07-17 Extension Settings Hydration Checkpoint

- `a0f461c0b fix(web): render extension settings without hydration mismatch`
  fixes the user-visible plugin/theme settings pane that left the admin shell
  visible while the form area remained blank for many seconds.
- The exact cause was an unqualified wrapper `<template>` compiled as a native
  `HTMLTemplateElement`: SSR flattened its children while client hydration
  placed them in `template.content`, producing `section` versus `div` and parent
  children mismatches. The trusted component and Schema renderer are now direct
  adjacent `v-if`/`v-else` branches.
- Settings async identity now includes extension id, normalized page path, and
  locale. Lazy client navigation exposes the existing loading state instead of
  suspending the complete page, and an absent payload stays undefined so Nuxt
  can fetch rather than treating a default null as hydrated success.
- The focused Bun settings suites passed **13/13**, Nuxt typecheck passed, and
  real logged-in Chrome validation passed direct and SPA navigation for both
  `sforum.content-policy` and `sforum.default-theme`. Both forms rendered, cold
  theme navigation showed an explicit loading state, and new browser console
  warnings/errors were empty. This regression fix does not change the 99-row
  score.

### 2026-07-17 P6 Bidirectional Staged Modifier Checkpoint

- Verified weighted progress is **64.3447%** (display **64.3%**); P6 advances
  to **15/18**. Only committed and verified evidence is credited.
- `5da58f160 feat(routes): execute bidirectional staged modifiers` closes the
  accepted route-action and explicit mutable-field/schema rows. It lands the
  staged request/handler/response sequence, bounded request and response patch
  application, immediate schema revalidation, exact Protocol V2 stage/action
  bridge, lossless repeated-query propagation, Host-issued params proof,
  Protocol V1 modifier fence, stage-aware traces/failures, and the production
  action/guard/failure matrices.
- Its exact index passed full Routes, Http, Extensions, and bootstrap tests;
  Routes/Http race tests; four-package vet; `go build ./...`; a real subprocess
  repeated-query test; and the production Dispatcher benchmark.
- `d55f027a6 test(routes): prove request patch schema revalidation` adds the
  missing negative production proof: the first modifier changes a valid string
  to JSON number `42`, the same schema rejects it on the second validation, the
  second modifier and Core never run, the exact runtime lease drains, and one
  payload-free request-stage trace records `schema_rejected` after remote
  execution. It passed 50 focused repetitions, 10 race repetitions, full Http,
  and vet.
- Current exact next step: bind required-idempotency policy into the same
  immutable Routes snapshot and execution plan, prove a 64-reader publication
  matrix, then land the production Bound replay adapter and wrong-key fail-
  closed tests. Do not stage the parallel Identity, stream/WebSocket, SEO,
  public frontend, or user-owned fixture drafts with that slice.

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

- `6ed44a86c fix(web): speed extension settings rendering`
- `f65c89a8f fix(routes): preserve terminal websocket cleanup`
- `9d2bcb56e fix(routes): require websocket response terminal`
- `5b4b147ec feat(extensions): add exact catalog filtering`
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
- Unowned dirty / untracked inventory after stream lifetime commits (do not
  stage unless a later subtask proves ownership):
  - `apps/api/app/Http/route_action_v2_fiber_integration_test.go`
  - `apps/api/app/Http/route_websocket_trust_revoke_integration_test.go` (??; unowned)
  - `apps/api/app/Models/Extensions/public_frontend_policy.go` (??)
  - `apps/api/app/Models/Extensions/public_frontend_policy_test.go` (??)
  - `apps/api/app/Support/Extensions/admin_surface_reference_plugin_integration_test.go`
  - `apps/api/app/Support/Extensions/protocol_v1_builtins_integration_test.go`
  - `apps/api/app/Support/Routes/route_mutation_test.go`
  - `apps/api/bootstrap/app.go`
  - `apps/api/go.mod`
  - `apps/web/**` route-inspector / public-extension drafts
  - `contracts/openapi/schemas/extension-route-inspector.yaml`
  - `docs/extensions/host-api-v2.md`
  - `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
- The uncommitted SEO family is separate from Cache and includes
  `Support/SEORegistry`, SEO Protocol/SDK/runtime/bootstrap files, the
  `sforum-seo-reference` fixture, and its fixture index entry.
- The P12 migration-proof implementation/tests are committed in `fea430020` and
  are no longer dirty ownership.
- Migration 034 and Identity legacy adoption are committed; never edit the
  already-applied migration 029 or mix later Identity work into P6 authority.
- `docs/extensions/catalogs/manifest-v3.md`, the V3 ADR edit, and every other
  unstaged file remain outside the current commit until independently reviewed.

## Non-HTTP Schema Gap (explicit product options)

`RouteStreamFrame` currently carries only `DataChunk` raw bytes
(`contracts/proto/sforum/plugin/v2/runtime.proto`). There is **no frozen**
JSON framing/validation contract for SSE event fields, WebSocket text/binary
message schema, multipart part headers/boundaries, or arbitrary stream
documents. Options before any implementation:

1. **Opaque bytes only (status quo):** Host validates size/backpressure only;
   plugins own framing; Manifest cannot declare stream response Schema for
   non-HTTP modes. Document this as the supported boundary.
2. **Mode-specific frame envelopes:** add versioned protobuf oneofs
   (`SseEvent`, `WebSocketMessage`, `MultipartPart`) selected by route mode;
   Host validates envelope shape; body payload remains optional Schema-bound
   JSON/bytes.
3. **Single JSON document stream:** each chunk is one Schema-validated JSON
   value; unsuitable for binary WebSocket/multipart without base64 cost.

Do **not** credit non-HTTP Schema or raise P6 until one option is chosen and
wired through Manifest V3, Protocol V2, Host validation, docs, and tests.

## Custom/raw Guard Production Evidence Audit

Already present (unit/integration, not full credit alone):

- Production evaluator: `RuntimePluginRouteGuardEvaluator` bound in
  `bootstrap/app.go` via lifecycle runtime manager + extension guard policy.
- Trust/safe-mode/digest fail-closed matrix in `plugin_guard_runtime_test.go`.
- Raw credential forwarding matrix in `route_request_authority_matrix_test.go`.
- Dispatcher raw stamp sealing (`1f2c2e81a`) and stream raw stamp preservation.
- Plugin guard request/response failure matrices in Routes.

Production-chain evidence now committed in `1fc9226a1` (see Current Subtask).
The four previously open requirements are covered by
`route_guard_production_chain_integration_test.go`. The custom/raw **row** is
still not credited until the joined full P6 behavior matrix and non-HTTP Schema
freeze close with it; do not raise P6 on this evidence alone.

Remaining before any P6 score increase:

1. Full joined behavior matrix across actions/priority/locale/CSRF/guard/stream/
   disconnect/timeout/crash/multipart/unsafe committed response.
2. Non-HTTP Schema product freeze (opaque / mode envelopes / JSON stream) with
   Manifest + Protocol V2 + Host + docs + tests, or an explicit accepted opaque
   boundary decision.


## P6 Behavior Matrix Evidence Inventory (2026-07-18)

Honest inventory only. **Do not credit the matrix row or raise P6** until every
cell below has production-path evidence in one joined regression suite and the
non-HTTP Schema product freeze is decided.

| Cell | Status | Primary evidence |
| --- | --- | --- |
| Action terminals add/alias/redirect/rewrite | present (Registry/Plan) | `Support/Routes/route_matrix_test.go` |
| before/after/filter/wrap/replace/global | present | `route_matrix_test.go`, staged modifier tests |
| Priority order / conflict selection | present | `route_matrix_test.go` |
| Locale path + query/body | present (Fiber) | `route_request_authority_matrix_test.go` |
| Permission + CSRF | present (Fiber) | `route_request_authority_matrix_test.go` |
| Custom guard allow/deny/crash (fake runtime) | present (Fiber) | `route_request_authority_matrix_test.go` |
| Custom/raw production chain (real Protocol V2) | present (Fiber+subprocess) | `route_guard_production_chain_integration_test.go` (`1fc9226a1`) |
| Raw credential + trust revoke | present | production-chain + authority matrix |
| Legacy authorizer cannot mint raw | present (Fiber) | production-chain HostRouteGuardAuthorizer test |
| Stream multipart/SSE/WebSocket/disconnect | present (real subprocess) | `route_dispatcher_stream_integration_test.go` |
| Stream lifetime budget/cancel/ForceCancel | present | `stream_lifetime_test.go` + Http stream tests |
| WebSocket Open-only custom guard | present | production-chain WS test |
| Protocol V2 crash/timeout (Fiber) | present | `route_failure_matrix_test.go` |
| Guard failure classification matrix | present | `dispatcher_guard_failure_matrix_test.go` |
| Unsafe no-second-writer | present | `route_matrix_test.go` |
| Safe mode bypass | present | `route_matrix_test.go` + stream safe-mode tests |
| Non-HTTP Schema framing/validation | **open** | `DataChunk` raw bytes only; product options recorded |
| Joined single-suite matrix across all cells | **open** | cells exist separately; not yet one joined gate |
| Durable incident source for all stream failures | partial | failure sink matrices; not fully joined to stream |

### Exact matrix exit criteria still open

1. One joined regression (or explicitly named suite list in CI) that runs every
   cell above without skipping race/count gates.
2. Non-HTTP Schema product decision recorded in `knowledge/decisions/` and wired
   or explicitly accepted as opaque-bytes boundary across Manifest, Protocol V2,
   Host, docs, and tests.
3. Only then check the plan book rows for streamed transport, custom/raw, and
   the tests matrix, raise P6 from **15/18**, and recompute weighted progress.


## Exact Next Steps

1. Join and close the full P6 behavior matrix across every action, priority/
   conflict, locale/query/body, permission/CSRF, custom guard, stream,
   disconnect, timeout, crash, multipart, and unsafe committed response. Prefer
   extending existing `route_matrix_test.go`,
   `route_request_authority_matrix_test.go`, and
   `route_failure_matrix_test.go` rather than inventing a parallel harness.
2. Resolve non-HTTP Schema product option (opaque / mode envelopes / JSON
   stream) before any framing implementation; record the freeze in
   `knowledge/decisions/` and contracts. Do not claim Schema complete with only
   raw `DataChunk` bytes.
3. Only after matrix + Schema product freeze are production-proven together with
   the committed custom/raw production-chain and stream lifetime evidence, credit
   P6 from **15/18** toward **18/18** and recompute weighted progress. Do not
   raise progress on partial evidence.
4. Keep implementation/tests/docs in separate commits; never stage unowned dirty
   files listed under Dirty Worktree Ownership.
5. Add full-set/staged-publication quarantine concurrency coverage. Current
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
