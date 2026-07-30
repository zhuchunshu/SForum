# 2026-07-31 Release Gate Repair

## Changed

- Declared `@vue/compiler-sfc` as a direct Web development dependency so fresh
  Bun installs can resolve the SFC compiler imported by focused UI tests.
- Restored Nuxt's standard `nuxt prepare` postinstall lifecycle so fresh Bun
  installs generate `.nuxt/tsconfig.json` before unit tests resolve `~` and `@`
  imports. The isolated `.nuxt-typecheck` directory remains unchanged.
- Relaxed the system-error ownership source assertion to accept attributes on
  the non-themeable `<UApp v-else>` branch, preserving the active toaster
  configuration while keeping the ownership boundary under test.
- Made the exact-main-CI waiter normalize GitHub's empty in-progress conclusion
  and parse API fields with a non-whitespace separator; its regression fixture
  now reproduces the empty conclusion returned during `v3.0.0-alpha.3`.

## Verification

- `bun install --frozen-lockfile` completed without changes.
- The four previously failing frontend test files passed: 18 tests, 0 failures.
- Run `30585464069` was reproduced from commit `53027e8be`: without `.nuxt`,
  the two focused identity entries failed on unresolved Nuxt aliases; after
  `nuxt prepare`, all 12 tests passed.
- A fresh isolated `bun install --frozen-lockfile` ran the new postinstall,
  generated the `~/*` alias in `.nuxt/tsconfig.json`, and changed no packages.
- The seven Web files exercised by the repository quality gate passed with 35
  tests; the complete Web suite passed with 819 tests, and Nuxt typecheck
  passed.
- `git diff --check` passed. Architecture boundary validation is currently
  blocked by unrelated in-progress Forum changes that grew `service.go` from
  1087 to 1096 lines and `service_ops.go` from 1257 to 1260 lines.
- `scripts/ci/verify-main-ci_test.sh`, Bash syntax, and diff whitespace checks
  passed. Full repository and release workflows were deliberately not run.

## Next

- Commit and push the repair, require the new main SHA's CI to pass, then publish
  `v3.0.0-alpha.4`; do not move or reuse the failed `v3.0.0-alpha.3` tag.

## Open Questions

- None.
