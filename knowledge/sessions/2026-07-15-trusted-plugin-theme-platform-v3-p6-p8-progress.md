# 2026-07-15 Trusted Plugin And Theme Platform V3 P6-P8 Progress

## Changed

- Weighted V3 is 48%. P5 is 11/17, P6 is 11/18, P7 is 4/22, and P8 is 9/18.
- Contextual Core Guard production coverage is 105/123. Exact extension trust,
  Options owner policy, public/bootstrap routes, theme assets, Page Registry
  access, public entity metadata, safe custom-role deletion, declared extension
  routes, and cookie-bound PAT list/create use immutable Host authority with
  request-path I/O tests.
- P7 now has production versioned action/filter hooks with exact runtime
  snapshots, deterministic priority, typed contracts, sync/fail policy,
  dependency SemVer, optional fallback, Host revalidation, River delivery, and
  Protocol V2 exact declaration binding.
- P8 has an exact compiled four-level theme fallback implementation, but its
  authoritative fallback/switch rows remain open while add pages, visible exact
  preview, stale binding cleanup, concurrent activation, and multi-node
  convergence are unfinished.

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

## Verification

- P6 batches passed focused repetition, full HTTP/bootstrap or policy race,
  `go vet ./...`, and `go build ./...` before commit.
- Cookie-bound PAT management additionally passed ten focused repetitions,
  full HTTP race, Routes/bootstrap tests, vet, build, and staged diff checks.
- P7 passed full ExtensionManifest and Extensions tests, complete Extensions
  race in 119.901 seconds, targeted rollback races, vet, build, gofmt, and diff
  checks.
- P8 fallback passed focused Pages/Extensions/Controller tests, race, vet,
  build, and ThemeCompiler allocation checks before commit.

## Decisions

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
- P8: theme migration/store/service/Page Registry/runtime, Extensions and Pages
  controllers, OpenAPI, admin composable/i18n, and related tests.
- Preserve these groups and stage only one coherent owner at a time.

## Next

1. Continue the remaining 18 contextual guards without opening custom/raw
   authority before its product freeze: five target-dependent identity admin,
   three self resource-dependent identity, four executable bootstrap flows, two
   entity-meta value routes, two attachment reads, one inbound webhook, and one
   forum comment-create route.
2. Land the P8 additive migration separately, then exact preview/CAS, stale
   binding cleanup, runtime convergence, OpenAPI, and visible admin preview in
   separate buildable commits.
3. Recalculate P8 only after concurrent activation/restart/multi-node and
   JavaScript-disabled evidence meet the authoritative row.
4. Resume P5 immediately after the user approves the recommended product
   boundaries.

## Open Questions

- Awaiting user freeze: cumulative additive database grants, per-runtime lease
  roles, short-lived Host-signed actor delegation, and provider-neutral
  entitlement lifecycle.
- Awaiting user freeze: RFC 6901 mutable-field paths, high-priority wrap order,
  after fail-closed semantics, and redirect/canonical policy for remaining P6
  actions.
