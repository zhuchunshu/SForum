# 2026-07-29 CI Quality Gate Fix

## Changed

- Repaid architecture ratchets exposed by Actions instead of raising their
  limits: split Identity role-suggestion decisions and Navbar language-menu
  state into focused owners, moved settings restart coordination into the
  existing lifecycle owner, and lowered baselines for files that shrank.
- Updated stale lifecycle, uninstall, search-route, Page ViewModel, and Navbar
  tests to exercise the current exact-artifact and Page Registry contracts.
- Recalibrated ThemeCompiler allocation-count budgets for the measured
  `golang.org/x/net v0.56.0` cost while preserving byte budgets.
- Added PostgreSQL 17 to the CI quality job. The service uses the repository's
  required-test port `15432`, is migrated before the gate, and exposes only the
  isolated compatibility-farm URL during the full test script.

## Decisions

- Do not raise architecture limits to hide responsibility growth.
- Do not weaken exact-artifact lifecycle validation to preserve obsolete test
  fixtures.
- Do not export `DATABASE_URL` or `SFORUM_TEST_DATABASE_URL` across the full CI
  Go suite; those variables opt into broader destructive integration tests.

## Verification

- Passed architecture validation, actionlint, Nuxt typecheck, focused Bun
  suites, focused Go packages, and local PostgreSQL compatibility farm.
- A fresh temporary PostgreSQL database accepted all 128 Core and 7 River
  migrations, then all six remotely failing tests passed across
  `Models/Extensions`, `Support/Extensions`, and `bootstrap`. The temporary
  database was removed after verification.

## Next

- Commit and push the final workflow correction, then confirm the replacement
  GitHub Actions CI run passes the quality and container jobs.

## Open Questions

- None.
