# 2026-07-15 Trusted Plugin And Theme Platform V3 P6-P8 Progress

## Changed

- Weighted V3 is 49%. P5 is 12/17, P6 is 11/18, P7 is 6/22, and P8 is 10/18.
- Contextual Core Guard production coverage is 114/123. Exact extension trust,
  Options owner policy, public/bootstrap routes, theme assets, Page Registry
  access, public entity metadata, safe custom-role deletion, declared extension
  routes, cookie-bound PAT list/create, and the inert inbound-webhook skeleton
  use immutable Host authority with request-path I/O tests.
- P7 now has production versioned action/filter hooks with exact runtime
  snapshots, deterministic priority, typed contracts, sync/fail policy,
  dependency SemVer, optional fallback, Host revalidation, River delivery, and
  Protocol V2 exact declaration binding.
- P8 has an exact compiled four-level fallback and durable exact publication.
  Two real nodes converge through LISTEN plus authoritative polling; remaining
  work is Page ViewModel closure, all-catalog hot-path proof, and crawler/
  JavaScript-disabled SSR evidence.

## Commits

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

## Verification

- P6 batches passed focused repetition, full HTTP/bootstrap or policy race,
  `go vet ./...`, and `go build ./...` before commit.
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
- P8 fallback passed focused Pages/Extensions/Controller tests, race, vet,
  build, and ThemeCompiler allocation checks before commit.
- P8 exact-preview backend, OpenAPI, admin typecheck/locales, and the additive
  publication migration passed focused gates. Migration `020` additionally
  passed real PostgreSQL Up/Down/reapply and immutable-history scenarios.
- P8 publication passed isolated PostgreSQL activation, compensation,
  concurrency, node lease/ack/retry, LISTEN reconnect, and two-node convergence;
  full Go tests, focused race, vet, and build also passed.
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

- P6: `apps/api/app/Http/core_guard_authorizer*`, extension guard policy, and a
  narrow bootstrap policy injection for the next contextual batch.
- P7: generic Provider Inspector and later selection/reset/probe/health UI.
- P8: Page ViewModel/hot-path audit and crawler/JavaScript-disabled evidence.
- Preserve these groups and stage only one coherent owner at a time.

## Next

1. Continue the remaining nine contextual guards without opening custom/raw
   authority before its product freeze: four executable bootstrap flows, two
   entity-meta value routes, two attachment reads, and one forum comment-create
   route.
2. Audit every P8 Page ViewModel and prove all-catalog hot rendering has no
   theme disk I/O or provider database query.
3. Start API and Nuxt manually, then capture crawler and JavaScript-disabled
   SSR evidence for home, lists, topic, profile, pagination, SEO, and JSON-LD.
4. Continue P5 with signed actor delegation and concrete transactional Host
   Commands, then provider-neutral entitlement persistence.
5. Continue P7 with Provider Inspector, then selection/reset/probe/health UI.

## Open Questions

- Awaiting user freeze: RFC 6901 mutable-field paths, high-priority wrap order,
  after fail-closed semantics, and redirect/canonical policy for remaining P6
  actions.
- An early P8 PostgreSQL fixture wrote append-only test revisions into the local
  configured database. Desired state was restored; historical rows must remain.
  Resume only with uniquely migrated per-test schemas, never destructive cleanup.
