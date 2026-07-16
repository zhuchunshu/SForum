# 2026-07-16 Trusted Plugin And Theme Platform V3 Identity/Producer Checkpoint

## Progress

- Exact weighted progress remains **63.1414%**; display **63.1%**.
- P6 remains **14/18**, P7 **14/22**, P8 **18/18**, and P9 **4/16**.
- P10-P13 and Program Definition of Done remain uncredited.
- The work below closes prerequisite authority and producer contracts, but does
  not receive a phase row before lifecycle, bootstrap, HTTP/admin UI, reference,
  and multi-node exits are proven.

## Committed

- `474792090` publishes the complete database-derived enabled executable plugin
  desired set under a shared advisory fence. Static plugins and themes are
  excluded; same sets reuse the revision; exact COMMIT recovery supports
  `MaxConns=1`; canceled lock waiters destroy their physical backend.
- `e086de0b3` adds migration 029 for declaration-bound permission catalog rows,
  immutable role-grant evidence, actor authorization, privacy-safe audit
  retention, truthful `rolePermissionAdded`, and authority-protected Down.
- `284efe816` recovers an exact committed producer revision even when physical
  close reports an error after `Hijack`; missing revisions retain commit,
  release, and close evidence.
- `d228ecddf` implements actor-bound role-suggestion approval/rejection replay,
  additive grants, legacy review-only apply, bounded detached COMMIT readback,
  stable keyset listing, service permission mapping, and production
  `ReplaceRolePermissions` race coverage.

## Verification

- Producer real PostgreSQL normal `x5`, focused race `x3`, and vet passed.
- Producer dual-fault normal `x10`, race, vet, gofmt, and diff checks passed.
- Migration static `x20`, real PostgreSQL full suite, focused race `x2`, and vet
  passed. Grok and Codex independent migration reviews returned `PASS`.
- Identity Registry full real PostgreSQL, focused race, Models full suite,
  production replacement race `-race x3`, vet, gofmt, and diff checks passed.
- Identity review findings were fixed before commit: exact audit actor asserts,
  non-self-referential provenance asserts, production replacement coverage, and
  exact existing-grant evidence comparison.

## Active Ownership

- P12 full-set work owns `Support/Extensions` manager/protocol/full-set files and
  `Support/HostAPI` service runtime-set files. It must still close cancellable
  writer-fence admission and real queued-writer rollback tests before commit.
- Query work owns `Support/QueryRegistry` and new `HostAPI/query_registry_*`.
- Component work owns `Support/Extensions/component_registry*` and
  `component_composition*` only.
- Navigation work owns `Support/NavigationRegistry` plus the explicitly scoped
  `Models/SiteChrome/navigation_registry_*` files.
- Content work owns `Support/ContentRegistry` plus new
  `HostAPI/content_registry_*` files.
- Cache work owns `Support/CacheRegistry` plus new
  `HostAPI/cache_registry_*` files.
- All external CLI output remains untrusted until exact file scope, diff,
  normal/race/vet, and product-contract review pass. No slice may be committed
  together with another Registry family.

## Preserve

- Never stage `apps/api/app/Models/PageViewModels/source_test.go`.
- Never stage
  `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`.
- Do not push, tag, open a PR, create a branch/worktree, reset, clean, or rewrite
  history.

## Next

1. Resolve P12's cancellable Service writer fence and production rollback race,
   then split HostAPI compatibility, Protocol lease, full-set coordinator, and
   bootstrap wiring into independently tested commits.
2. Collect and terminate completed external CLI sessions; review Query first,
   then Component, Navigation, Content, and Cache without cross-family staging.
3. Add lifecycle/bootstrap/HTTP/OpenAPI/admin/frontend bindings and reference
   plugin evidence before crediting P7/P9/P10/P11/P12 rows.
4. Keep progress at **63.1%** until an authoritative task-book row fully exits.
