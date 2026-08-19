# 2026-08-20 Plugin SDK Distribution Handoff

## Changed

- Added exact npm package verification for `@sforum/admin-sdk@1` and
  `@sforum/plugin-ui@1`: metadata/license checks, bridge-major and scaffold
  version alignment, tarball integrity, and an offline Vite consumer build.
- Added an OIDC-only Release job with provenance. It skips an exact existing
  version, rejects different content under the same version, and blocks image
  promotion when SDK publication fails.
- Added `sforum extension build` with locked Bun install, frontend build,
  digest refresh, full validation, and contract tests. `--skip-install` and
  `--allow-scaffold` support the normal author workflow without weakening other
  checks.
- Added the reviewed V3 route and UI identities omitted when component-level
  admin page assets first landed, then regenerated the route/component catalogs
  and raised their exact reviewed counts.
- Updated generated plugin README text and the bilingual CLI/build references.
- Published `@sforum/admin-sdk@1.0.0` and `@sforum/plugin-ui@1.0.0` publicly
  through the one-time interactive 2FA bootstrap.
- Bound both npm packages to the GitHub Actions Trusted Publisher
  `zhuchunshu/SForum`, workflow `release.yml`, no environment, with only
  `npm publish` allowed.

## Decisions

- SDK npm majors stay aligned with Admin Micro-frontend API versions, while SDK
  releases remain independent from the SForum application version.
- Package scripts execute only in the explicit author CLI command. See
  `../decisions/2026-08-20-plugin-sdk-distribution.md`.

## Verification

- `go test ./...` passed, including the complete CLI package and the parallel
  process-memory work present in the shared tree.
- Nuxt typecheck and all 903 Web tests passed. The packed SDK consumer emitted
  135.57 kB ESM and 5.81 kB CSS without registry access.
- Documentation/OpenAPI/architecture validators, Release and deployment
  scripts, actionlint, publisher retry tests, Host/V3 trust checks, and the V3
  catalog gate passed (`343` routes, `291` UI surfaces, `99` traceability rows).
- `./scripts/test.sh` passed through the full Go suite, then stopped at the
  required compatibility farm because this shell had no `DATABASE_URL` or
  `SFORUM_TEST_DATABASE_URL`. The remaining non-database gates were run
  separately and passed.
- Registry `latest` resolves to `1.0.0` for both packages. Fresh registry
  downloads exactly match the reviewed local tarballs: admin SDK SHA-256
  `2b211dfaee509f33752589956115f9564e1d5c5e87182161f1d5d99da89b15cc` and
  Plugin UI SHA-256
  `2ffd111212d455f7521ee9dbaefe3656995a56f255ea861620dc50cdc4523d35`.
- npm Settings showed a successful Trusted Publisher connection for both
  packages with the exact repository, workflow, and allowed action.

## Next

- A future prerelease tag may confirm exact-version retry behavior and image
  promotion ordering in the real Release workflow.
- Confirm npm provenance when an SDK content change produces the next package
  version. The interactive bootstrap cannot retroactively add provenance to
  `1.0.0`.

## Open Questions

- Whether to switch both npm package publishing-access settings from the
  bypass-token-compatible option to npm's recommended 2FA-only option. Trusted
  Publishing works with either setting; the repository stores no npm token.
