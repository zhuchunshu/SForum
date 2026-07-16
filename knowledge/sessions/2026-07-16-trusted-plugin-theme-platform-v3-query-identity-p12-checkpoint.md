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

## Active Owned Work

- Query lifecycle publication is active in `Models/Extensions`,
  `Support/Extensions`, and `bootstrap`; it remains uncommitted until startup,
  Safe Mode, exact-runtime replacement/removal, and focused tests converge.
- P12 additive migration `202607160027_plugin_runtime_publications.sql` and its
  two tests are active. The schema should keep desired full sets, exact members,
  boot leases, acknowledgement CAS, applied runtime evidence, and PostgreSQL
  `NOTIFY` without duplicating lifecycle/trust/migration ledgers.
- Cache Registry is an untracked P11 leaf under `Support/CacheRegistry`; it must
  pass main review and tests and receives no credit before lifecycle plus a real
  Host CacheService enforces namespace, actor, tag, lock, and provider policy.
- A Grok task owns only the new `Support/ContentRegistry` directory. Its output
  is a candidate P10 declaration foundation, not production completion.

## P12 Review Requirements

Before the migration may be committed, it must enforce and test:

- monotonic `last_applied_revision` and rejection of non-newer apply;
- a live boot lease for acknowledgement and applied evidence writes;
- plugin-only desired members, excluding theme versions;
- an applied-set digest recomputed from exact applied members;
- empty and multi-member sets, wrong digest, failed retry, worker role, lease
  expiry, and out-of-order/concurrent apply behavior.

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

1. Review and atomically commit Query lifecycle publication.
2. Review the corrected P12 migration and commit it independently.
3. Review and commit Cache/Content leaf foundations only if their contracts and
   tests are sound, without awarding production progress.
4. Add Identity lifecycle/tombstone persistence only after Query has released
   the shared lifecycle files.
5. Continue with Query executor/Host outlet and P11 Host CacheService rather
   than accumulating disconnected registries.
