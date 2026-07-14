# 2026-07-15 Trusted Plugin And Theme Platform V3 P6-P8 Progress

## Changed

- Weighted V3 is 48%. P5 is 11/17, P6 is 11/18, P7 is 4/22, and P8 is 9/18.
- Contextual Core Guard production coverage is 106/123. Exact extension trust,
  Options owner policy, public/bootstrap routes, theme assets, Page Registry
  access, public entity metadata, safe custom-role deletion, declared extension
  routes, cookie-bound PAT list/create, and the inert inbound-webhook skeleton
  use immutable Host authority with request-path I/O tests.
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

## Verification

- P6 batches passed focused repetition, full HTTP/bootstrap or policy race,
  `go vet ./...`, and `go build ./...` before commit.
- Cookie-bound PAT management additionally passed ten focused repetitions,
  full HTTP race, Routes/bootstrap tests, vet, build, and staged diff checks.
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
- P7: Protocol V2 broker transport and real cross-plugin provider consumption.
- P8: publication repository/activation transaction, watcher, heartbeats,
  acknowledgements, and multi-node convergence tests.
- Preserve these groups and stage only one coherent owner at a time.

## Next

1. Continue the remaining 17 contextual guards without opening custom/raw
   authority before its product freeze: five target-dependent identity admin,
   three self resource-dependent identity, four executable bootstrap flows, two
   entity-meta value routes, two attachment reads, and one forum comment-create
   route.
2. Bind the durable P8 publication repository to activation, then implement
   LISTEN plus poll/reconnect recovery, heartbeat, acknowledgements, and the
   two-node convergence gate in separate buildable commits.
3. Recalculate P8 only after concurrent activation/restart/multi-node and
   JavaScript-disabled evidence meet the authoritative row.
4. Resume P5 immediately after the user approves the recommended product
   boundaries.
5. Finish the P7 real Plugin B -> Host broker -> Plugin A provider path,
   lifecycle rollback, race, vet, and build evidence before crediting its row.

## Open Questions

- Awaiting user freeze: cumulative additive database grants, per-runtime lease
  roles, short-lived Host-signed actor delegation, and provider-neutral
  entitlement lifecycle.
- Awaiting user freeze: RFC 6901 mutable-field paths, high-priority wrap order,
  after fail-closed semantics, and redirect/canonical policy for remaining P6
  actions.
