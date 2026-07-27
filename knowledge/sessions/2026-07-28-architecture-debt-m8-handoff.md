# 2026-07-28 Architecture Debt M8 Handoff

## Changed

- Split production API assembly into infrastructure, extension platform,
  extension restore, domain service, and HTTP finishing stages. All stage files
  are below 500 lines and preserve the existing staged cleanup handoffs.
- Split the Identity HTTP Controller into public authentication, admin access,
  session, and shared controller files.
- Split Identity PostgresStore into current-user, permission, admin-user, role,
  and bootstrap transaction responsibilities.
- Split Forum PostgresStore into taxonomy, topic query, topic mutation, and
  topic lifecycle responsibilities.
- Split Options normalization into dispatch, option-set validation, scalar,
  SEO/footer, and forum content-limit responsibilities.
- Replaced Forum constructor permutations with the single
  `NewService(ServiceConfig)` entry and migrated all callers.
- Removed all five repaid legacy large-file baselines.

## Decisions

- M8 stays inside existing Go packages; stable package extraction waits for
  collaborator interfaces to settle in M9-M10.
- API stage ownership preserves the prior close/rollback order. The HTTP stage
  receives handed-off resources and the final `API.close` remains authoritative.
- Forum optional dependencies are declared in one config instead of adding
  another derived constructor.

## Evidence

- Focused Identity Controller, Identity, Forum, Options, Provider, and Forum
  Controller tests passed.
- Full bootstrap package tests passed.
- CLI compile-only test passed.
- `go list ./...` passed.
- Architecture validation passed: 1394 production files scanned, 163 files
  remain above the 500-line review threshold.
- `git diff --check` passed and repository search found no old Forum constructor
  definitions or calls.

The full CLI test suite contains an orphan-process test that invokes `/bin/ps`;
the managed sandbox rejects that process listing with `operation not permitted`.
This is an environment restriction, not an M8 failure, and was not rerun.

## Next

- M9 extracts Catalog, Lifecycle, Theme, and Settings collaborators inside the
  existing Extensions package while retaining `Service` as a compatibility
  facade.
- Keep the production Go file count at or below 95; reuse existing files.
- Theme owns `themeActivationMu`, `assetPublicationMu`, and
  `themeRuntimeUnavailable`. Other collaborators use narrow boundaries and do
  not copy locks.
- Lower the `Service` receiver cap from 151 to the final facade count and add
  collaborator-specific caps.

## Open Questions

- None. If collaborator extraction reveals a transaction or lock-order
  ambiguity, pause M9 for an ADR rather than inventing a package boundary.
