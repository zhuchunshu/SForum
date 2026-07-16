# Trusted Plugin And Theme Platform V3 Progress Ledger

Last updated: 2026-07-17

## Progress

- Verified weighted baseline: **62.8586%** (display **62.9%**).
- Phase counts: P0-P5 and P8 complete; P6 **13/18**, P7 **14/22**,
  P8 **18/18**, P9 **4/16**, and P12 **1/22**. P10, P11, and P13 have no
  credited authoritative row yet.
- Completion remains unproven until all 99 target rows, 14 accepted boundaries,
  five reference-plugin classes, 24 Program Definition of Done rows, and final
  gates pass.

## Current Subtask

- **P11 Cache SDK closure:** review and commit the typed plugin helpers for
  namespaced get/set/delete/increment/tag invalidation, cross-RPC locks, and
  distributed `remember` without widening the already committed Host contract.
- The Cache implementation commit is limited to:
  - `apps/api/sdk/plugin/v2/cache.go`
  - `apps/api/sdk/plugin/v2/cache_operations.go`
  - `apps/api/sdk/plugin/v2/cache_test.go`
- Do not credit the P11 Cache task row until the SDK audit confirms the existing
  production Host service, provider policy, inspector, failure matrix, and the
  new helpers together prove the complete row.

## Recent Verified Commits

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

- `cd apps/api && go test ./sdk/plugin/v2 -count=1` passed.
- `cd apps/api && go test -race ./sdk/plugin/v2 -count=1` passed.
- `cd apps/api && go vet ./sdk/plugin/v2` passed.
- `gofmt -d` for the three staged Cache SDK files produced no diff.
- `git diff --cached --check` passed.
- Independent audit found that a blocked Renew RPC could outlive a 100ms lease,
  allowing the old loader to overlap a replacement owner. The SDK now bounds
  Renew and the post-acquire read by the current lease expiry, independently
  cancels the loader at expiry, and has an auto-expiring two-owner regression.
- Cleanup now refreshes the wire deadline, invalid Acquire responses release a
  returned opaque token, remote error messages are discarded, and conditional
  write conflicts plus lease consumption have focused tests.
- The post-fix normal/race/vet, formatting, and worktree diff checks pass. One
  final staged-diff review remains before commit.

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
- `apps/api/app/Support/Extensions/lifecycle_migration_runtime_activation_postgres_integration_test.go`
  is separate P12 migration-proof evidence.
- `docs/extensions/catalogs/manifest-v3.md`, the V3 ADR edit, and every other
  unstaged file remain outside the Cache commit until independently reviewed.

## Exact Next Steps

1. Stage the post-audit versions of only the three Cache SDK files and review
   the complete staged diff.
2. Rerun staged diff check plus focused normal/race/vet if the index differs.
3. Commit only the three Cache SDK files as `feat(sdk): harden cache helpers`.
4. Update this ledger with the resulting hash and exact progress; commit that
   documentation separately.
5. Review and land SEO in contract/transport/Host-policy/bootstrap/reference
   slices, then implement the remaining P6 matrix.

## Rollback, Flags, And Compatibility

- Cache SDK rollback removes only convenience helpers; the committed Protocol
  V2 and Host CacheService contracts remain additive and usable directly.
- Protocol v1 compatibility remains present until P13 removal gates pass.
- Safe Mode remains Host-owned and filters third-party Registry publications.
- No database migration, feature-flag default, legacy deletion, push, tag, PR,
  branch, or worktree change belongs to the current Cache subtask.

## Open Questions

- None for the current Cache SDK boundary. Any audit finding that changes Host
  product semantics must be checked against the V3 ADR before implementation.
