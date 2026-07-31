# 2026-07-31 Alpha.10 Clean-host Deploy

## Changed

- Repaired the exact route/UI catalog counts and added the missing
  `systemUpdates` extension surface entry required by the alpha.8 main CI.
- Bound Trivy to each release matrix platform so arm64 digest scans do not
  silently request an amd64 child image.
- Replaced the deployment entrypoint with an image-only, beginner-friendly
  install/update flow and a secure bilingual production configuration wizard.
- Added clean/existing database state handling, failure-safe successful-version
  persistence, strict dotenv reads for backup/restore, deployment regressions,
  and bilingual operator documentation.
- Follow-up review added pre-database configuration/port/image identity checks,
  `--pull never` after prefetch, a deployment lock, stable service sampling,
  and explicit `recovery_required` evidence for every post-stop failure.
- The alpha.9 release run exposed that the production migrator did not receive
  four mandatory runtime secrets. Compose now passes them explicitly and a
  real merged-config regression test protects the contract.
- Removed Windows from the release asset matrix and asset tooling. SForum's
  supported CLI release platforms are Linux and macOS on amd64/arm64.

## Decisions

- `v3.0.0-alpha.8` and `v3.0.0-alpha.9` remain immutable failed candidates.
  Alpha.9 built all image candidates but failed before promotion because of the
  migrator environment contract and the now-removed Windows asset jobs. The
  repaired first deployable candidate is `v3.0.0-alpha.10`.
- The beginner path supports only Compose-managed PostgreSQL and Redis. External
  services remain an advanced future mode rather than an unsafe partial toggle.
- The generated deployment-local Marketplace verifier keeps unknown indexes
  locked until an official public key and key ID are configured.
- Windows is not a supported deployment or CLI platform now or in the planned
  release contract.

## Verification

- Deployment wizard, state-machine, production Compose, PostgreSQL safety,
  release asset, route catalog, V3 catalog, and release workflow focused checks
  pass.
- `./scripts/test.sh` passes with the required local compatibility PostgreSQL.
- `go build ./...` and the Nuxt production build pass.

## Next

- Push the repair, require the exact main SHA's CI to pass, then publish and
  deploy `v3.0.0-alpha.10` from all four anonymous-readable GHCR images.
- After release deployment evidence is complete, continue the signed release
  manifest and `sforum-manager` install/update operation engine.

## Open Questions

- None for the managed-Compose alpha deployment path.
