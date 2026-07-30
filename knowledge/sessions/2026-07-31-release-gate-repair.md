# 2026-07-31 Release Gate Repair

## Changed

- Declared `@vue/compiler-sfc` as a direct Web development dependency so fresh
  Bun installs can resolve the SFC compiler imported by focused UI tests.
- Made the exact-main-CI waiter normalize GitHub's empty in-progress conclusion
  and parse API fields with a non-whitespace separator; its regression fixture
  now reproduces the empty conclusion returned during `v3.0.0-alpha.3`.

## Verification

- `bun install --frozen-lockfile` completed without changes.
- The four previously failing frontend test files passed: 18 tests, 0 failures.
- `scripts/ci/verify-main-ci_test.sh`, Bash syntax, and diff whitespace checks
  passed. Full repository and release workflows were deliberately not run.

## Next

- Commit and push the repair, require the new main SHA's CI to pass, then publish
  `v3.0.0-alpha.4`; do not move or reuse the failed `v3.0.0-alpha.3` tag.

## Open Questions

- None.
