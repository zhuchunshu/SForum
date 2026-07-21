# 2026-07-16 Trusted Plugin And Theme Platform V3 Query/Identity/P12 Checkpoint

## Progress

- Exact weighted progress remains **63.1414%**; the durable displayed total is
  **63%** and the user-facing progress bar may show **63.1%**.
- P6 remains **14/18**, P7 **14/22**, P8 **18/18**, and P9 **4/16**.
- P10-P13 remain uncredited and Program Definition of Done remains **0/24**.
- Identity and Query foundations do not receive task-book credit before their
  production lifecycle, execution, persistence, Inspector, and reference exits.

## Committed

- `57ebbd266` adds the immutable Query Registry foundation with exact artifact
  CAS, Safe Mode, Host-sealed Core authority, admission fences, permission/cost
  callbacks, cache isolation, and race coverage.
- `78c5564fb` normalizes and bounds Identity Manifest role/risk declarations,
  rejects `super_admin` suggestions, and fences foreign session/risk authority.
- `eec809599` adds the immutable Identity Registry foundation with append-only
  ownership tombstones, exact artifact CAS, Safe Mode, deterministic providers,
  Host-only permission assignment semantics, strict SemVer, graph bounds, and
  Core-forgery/race tests.
- Focused Identity Registry and Manifest tests, race tests, vet, and diff checks
  passed before those commits.
- `16ddb69b1`, `faeb9d69f`, and `04e364b11` add exact Query publication
  conversion, lifecycle/startup/Safe Mode reconciliation, and one shared
  bootstrap Registry binding. Query remains **14/22** because the Host executor,
  result-schema/release fence, composition, result filters, Inspector, and
  reference exit are still absent.
- `9fc0ac885` adds the P12 plugin runtime convergence ledger with full-set
  digests, plugin-only exact members, API/worker boot leases, acknowledgement
  CAS, applied runtime evidence, monotonic node progress, and commit-time
  PostgreSQL notification. Real PostgreSQL empty/multi-member, retry, lease,
  digest, worker, Down/reapply, and concurrency coverage passed.
- `4a727eee9`, `9e93b09dc`, and `c62d757a3` align Manifest Go validation,
  embedded JSON Schema, and modular OpenAPI for content IDs, contracts,
  handlers, schema paths, backend requirements, and declaration bounds.
- `d212def13` adds an immutable Content Registry declaration leaf with exact
  artifact CAS, append-only ID/contract definition tombstones, exact-package
  history across Safe Mode/remove/restart, Core seals, deterministic snapshots,
  and race/security coverage. It receives no P10 task credit without lifecycle,
  render/editor/media/entity execution, persistence, Inspector, and references.
- `273e75799`, `f928c0ad5`, `ef57c3bae`, and `c364776f4` make production ZIP
  upload operator-build-free and race-safe: typed plugin/theme permissions,
  4096-entry/50 MiB bounds, owned cross-process content-addressed snapshots,
  exact durable reference retention, transactional SaveInstalled reads, and
  safe cleanup only for newly-created unreferenced bytes. Normal/race/vet,
  Windows compile, real PostgreSQL, full API build/tests, and OpenAPI refs passed.
- `8e25276c1`, `9fa698661`, and `891c33b83` add durable Identity ownership/
  declaration history, Host-reviewed role suggestions, a PostgreSQL repository,
  and a `role.manage`-guarded Host service which records review evidence without
  mutating `role_permissions`. These commits are under independent main review
  and do not close P7 until lifecycle publication, bootstrap/HTTP/admin UI,
  auth/profile providers, and allowed/denied integration exits exist.

## Active Owned Work

- P12 repository work is restricted to new
  `apps/api/app/Models/Extensions/plugin_runtime_publication*.go` files and
  matching tests. It may implement transactional full-set publication, node
  boot/lease, acknowledgement/applied-member CAS, and LISTEN/poll primitives;
  it must not yet edit lifecycle/bootstrap or claim Manager application.
- Cache Registry is an untracked P11 leaf under `Support/CacheRegistry`; it must
  pass main review and tests and receives no credit before lifecycle plus a real
  Host CacheService enforces namespace, actor, tag, lock, and provider policy.
- The only unrelated dirty files remain the two user-owned paths listed below.

## P12 Repository Review Requirements

Before the repository may be committed, it must preserve the ledger invariants
and enforce/test:

- canonical four-field full-set digest matching the migration trigger;
- monotonic `last_applied_revision` and rejection of non-newer apply;
- a live boot lease for acknowledgement and applied evidence writes;
- plugin-only desired members, excluding theme versions;
- an applied-set digest recomputed from exact applied members;
- empty and multi-member sets, wrong digest, failed retry, API/worker roles,
  lease expiry, LISTEN reconnect/wake-only behavior, missed-notification poll,
  and out-of-order/concurrent apply behavior.

Do not add `source_revision`, lifecycle operation, trust grant, protocol/backend,
or duplicate migration-proof columns to this first migration. Migration fencing
is an application/coordinator follow-up that must reuse existing lifecycle
migration-once evidence.

## Audit Findings

- P9 CSP and Navigation/Region remain blocked on product contracts: Host CSP
  baseline and merge rules, cross-policy SPA reload, stable targets/placement,
  permission visibility, SSR payload, and cache revision consumption.
- P10 is still declaration scaffolding around strong Core forum primitives; no
  production Content/Media/Entity/Taxonomy registry is currently wired.
- P11 is still zero: Core Redis/SEO/settings/SSRF helpers exist, but plugin
  Cache/SEO/Secret/File/HTTP/Localization services and recovery matrices do not.
- P13 references and legacy deletion cannot start before P6-P12 exits and final
  gates. All files under `extensions/fixtures` are intentional tracked source
  fixtures used by tests/docs; the directory must not be globally gitignored.

## Preserve

These pre-existing user changes are unrelated and must not be staged:

- `apps/api/app/Models/PageViewModels/source_test.go`
- `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`

## Next

1. Complete independent review of the three Identity persistence/review commits;
   fix rather than over-credit any product-semantic or SQL issue.
2. Review and commit P12 repository primitives in small transaction/listener
   slices, then separately wire Manager apply and API/worker bootstrap.
3. Implement the Query Host executor/outlet, permission/cost/result-schema
   release fences, result filters, composition, Inspector, and reference plugin.
4. Either add durable ownership and a real Host CacheService to Cache Registry
   or leave the declaration leaf uncommitted; no P11 credit for names/types.
5. Continue P6 route composition/custom guard/SEO matrix and P9 component/
   navigation/CSP aggregation in parallel before starting P13 legacy deletion.
