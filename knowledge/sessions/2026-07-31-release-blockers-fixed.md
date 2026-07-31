# 2026-07-31 Release Blocker Remediation

## Changed

- Historical schema-declared `enc::` secrets migrate through the production
  SettingsLifecycle binding to SecretStore before reads or writes. Migration
  failures preserve the old document and fail closed; memory and PostgreSQL
  regressions cover the first write.
- Production Compose, example env, release smoke, and bilingual deployment docs
  now require the Marketplace Ed25519 public key and key ID.
- SystemTier priority reaches the real plugin full-set start plan. RuntimeRollout
  no longer creates a fictional healthy `api-local` Ack, and active plans remain
  discoverable for rollback.
- Marketplace actors resolve from `user:<id>` through Identity and require an
  active `super_admin`. Until real node health is wired before publication,
  Marketplace activation is staged-only and ordinary lifecycle rollout is
  deliberately unbound.
- PostgreSQL backup failure now stops deploy. CompatFarm RPC errors fail their
  cell. Built-in version/catalog validators and the admin resource gauge were
  corrected; generated V3 catalogs now cover 335 routes and 281 UI surfaces.
- Release-review follow-up makes restore fail fast in one transaction inside a
  temporary database, validates it, and atomically swaps database names while
  API/Worker are stopped. Backups and `.env.production` are mode `0600`, and
  deployment completion now waits for API readiness plus the Web root.
- GitHub Release creation now depends on credential-free pulls of all four
  promoted GHCR version tags. Public docs keep distributions prerelease while
  production-rewire M3/M5/M6/M7 remain open.
- Navbar Enter search, long homepage category labels, and registration legal
  links were repaired and covered by focused frontend regressions.

## Verification

- `./scripts/test.sh` passed with the real local PostgreSQL compatibility URL.
- `go vet ./...`, `bun test` (837 pass), Nuxt typecheck through the full gate,
  and Nuxt production build passed.
- Focused race-enabled SettingsLifecycle, RuntimeRollout, Extensions,
  CompatFarm, Models, and bootstrap Go tests passed.
- `/control-panel` passed Browser QA at 1440x900 and 390x844: refresh worked,
  the corrected 72x72 gauge rendered without clipping, console/overlay were
  clean, and neither viewport overflowed horizontally.
- Deployment safety and anonymous-image scripts passed their executable
  regressions. Browser QA confirmed `/search?q=上线` from a real Enter key,
  non-overlapping desktop topic columns, working `/terms` and `/privacy` links,
  and no 390px registration overflow.

## Decisions

- Strict signed-index key policy remains the production/staging default.
- Do not present post-terminal bookkeeping as a multi-node rollout gate. The
  unfinished node-health/promotion integration stays open under M3/M5.
- The clean-host backup-before-PostgreSQL behavior was deliberately not changed;
  deployment entrypoint replacement is owned by the operator's planned rewrite.

## Next

- Implement the real pre-publication RuntimeRollout node-health gate and the
  supported Marketplace/Privacy HTTP or CLI consumers.
- Finish CompatFarm matrix/single-run consolidation and commerce Dispatcher
  coverage, then run the Page Registry live smoke.
- Run `scripts/ci/release-smoke.sh IMAGE_TAG EXPECTED_VERSION EXPECTED_COMMIT`
  against the immutable release-candidate images once they exist.

## Open Questions

- None for the repaired P0s. Stable release approval still depends on the
  immutable candidate-image smoke and the explicitly open remediation scope.
