# 2026-08-18 Verified Release Bootstrap Handoff

## Changed

- Added `sforum-bootstrap.sh` as the verified install/update entrypoint. It
  resolves an immutable target, refreshes itself from that Release, verifies
  the complete matching deploy bundle, and hands off to `deploy.sh` or
  `upgrade.sh`.
- Existing installations preserve configuration, deployment state, runtime
  router state, data, and volumes. Release-owned tooling is backed up under
  `.sforum/tooling-backups/` before atomic per-file promotion and restored if
  promotion fails.
- Release assets now include a standalone bootstrap and include it inside the
  deploy bundle. Finalization restores executable modes after GitHub Artifact
  transfer and includes the bootstrap in `SHA256SUMS` and provenance inputs.
- README, Chinese/English deployment guides, generated Release Notes, and the
  deploy-bundle README now recommend the bootstrap. Direct remote shell pipes
  are rejected; local maintenance remains available through `deploy.sh`.
- Added local fake-Release coverage for fresh install, existing-instance
  adoption, state preservation, tooling backup, self-refresh, argument handoff,
  and checksum failure. Documentation and release-asset gates enforce the new
  contract.

## Decisions

- Installation and update refresh the whole target Release toolkit, not only a
  standalone updater. See
  `knowledge/decisions/2026-08-18-verified-release-bootstrap.md`.

## Verification

- `deploy/scripts/bootstrap_test.sh`
- `scripts/ci/release_assets_test.sh`
- `scripts/ci/generate-release-notes_test.sh`
- `node tests/validate-docs.mjs`
- `tests/validate-docs_test.sh`
- `node tests/validate-architecture-boundaries.mjs`
- `bash -n sforum-bootstrap.sh deploy.sh upgrade.sh deploy/scripts/*.sh scripts/ci/*.sh`

`./scripts/test.sh` passed the release, bootstrap, deployment, architecture,
documentation, built-in version, and ordinary Go suites. Without a database
URL it stopped at the required compatibility farm. With only the repository's
database URL mapped to `SFORUM_TEST_DATABASE_URL`, the compatibility farm
passed, but the pre-existing PostgreSQL integration test
`TestP7HostOwnedRoleMappingJoined` failed with `extension lifecycle registry
publication dependency is unavailable`; an isolated uncached rerun reproduced
the same failure. The remaining full gate was interrupted after other Go
integration packages stayed silent for several minutes. This failure has no
bootstrap/deployment call path.

## Next

- After merge, publish a new immutable prerelease tag and verify the Release
  contains `sforum-bootstrap.sh`, `sforum-deploy.tar.gz`, and exact checksum and
  provenance records before advertising the adoption command.

## Open Questions

- None.
