# 2026-07-31 Alpha.9 Clean-host Deploy

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

## Decisions

- `v3.0.0-alpha.8` remains immutable and failed before publishing images; the
  repaired first deployable candidate is `v3.0.0-alpha.9`.
- The beginner path supports only Compose-managed PostgreSQL and Redis. External
  services remain an advanced future mode rather than an unsafe partial toggle.
- The generated deployment-local Marketplace verifier keeps unknown indexes
  locked until an official public key and key ID are configured.

## Verification

- Deployment wizard, state-machine, PostgreSQL safety, shell syntax, ShellCheck,
  route catalog, V3 catalog, and release workflow focused checks pass.
- `./scripts/test.sh` passes with the required local compatibility PostgreSQL.
- `go build ./...` and the Nuxt production build pass.

## Next

- Push the repair, require the exact main SHA's CI to pass, then publish and
  deploy `v3.0.0-alpha.9` from all four anonymous-readable GHCR images.
- After release deployment evidence is complete, continue the signed release
  manifest and `sforum-manager` install/update operation engine.

## Open Questions

- None for the managed-Compose alpha deployment path.
