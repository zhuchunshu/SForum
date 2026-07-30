# 2026-07-29 GHCR Multi-Platform Release Pipeline

## Status

Accepted and implemented.

## Context

SForum already had production Dockerfiles and Compose deployment, but releases
were source-built on each operator host. The repository had no general PR gate,
no reproducible container publication, and no exact-image runtime smoke test.
`deploy.sh` could not consume a published version, so building images alone
would not improve deployment or establish a rollback artifact.

## Decision

1. Pull requests and `main` use one reusable CI workflow that runs the canonical
   repository gate, all Web unit tests, Go/Web production builds, and one native
   container build for every runtime target.
2. `vMAJOR.MINOR.PATCH[-prerelease]` tags must resolve to a commit reachable
   from `main` and reuse the exact commit's successful `main` push CI result.
   Release waits for that run on GitHub instead of executing the same gate a
   second time.
3. GitHub Container Registry is the canonical image registry. Releases publish
   `sforum-api`, `sforum-worker`, `sforum-migrate`, and `sforum-web` for
   `linux/amd64` and `linux/arm64`.
4. A release first publishes only a commit-addressed `sha-<commit>` candidate.
   Trivy scans the exact manifest and Compose starts those exact API, worker,
   migration, and Web images against fresh PostgreSQL and Redis services.
5. Only a verified candidate is promoted to the `vX.Y.Z` and `X.Y.Z` tags.
   Stable releases additionally update `X.Y` and `latest`; prereleases do not.
6. Published images include OCI source/version/revision labels, BuildKit SBOM
   and provenance, plus a GitHub artifact attestation. Actions are commit-SHA
   pinned and use job-scoped least privilege.
7. GitHub Release assets include six `sforum` CLI archives for Linux, macOS,
   and Windows on amd64/arm64, plus Linux amd64/arm64 backend bundles. Backend
   bundles reuse API/worker/migrate binaries and protected built-ins from the
   exact scanned candidate images; only the CLI is cross-compiled separately.
   Every archive is covered by `SHA256SUMS` and artifact provenance.
8. The Linux backend archive deliberately excludes the Nuxt Web runtime,
   PostgreSQL, and Redis. Docker Compose remains the complete production
   distribution, while archives serve CLI, recovery, and advanced deployments.
9. `compose.release.yaml` removes local builds and selects one explicit version
   for every application process. `deploy.sh --version` pulls that version,
   runs its migration image, and starts it without rebuilding.
10. GitHub Actions never deploys to operator-owned hosts. Operators retain
   authority over backup, migration timing, startup, and rollback.
11. The maintainer release helper returns after pushing the tag by default.
   Synchronous terminal monitoring remains available through explicit `--wait`.

## Consequences

- A GitHub Release is created only after all images pass scan and runtime smoke.
- Tag publication does not duplicate a gate already proven for the exact main
  commit, and release image builds can restore the corresponding CI image cache.
- GitHub Release publication is atomic with respect to verified images and the
  complete eight-archive asset matrix: checksums and provenance complete before
  versioned image aliases move. Partial or byte-different existing asset sets
  fail closed instead of being overwritten.
- Failed candidates remain addressable by commit for diagnosis but cannot move
  stable aliases.
- GHCR packages require a one-time administrator check that visibility is
  public and repository linkage is correct.
- Application-image rollback is now possible by redeploying a prior version,
  but automatic rollback metadata and database down-migration remain separate
  future work.
- The release Compose override requires Docker Compose 2.24.4 or newer because
  it uses the standard `!reset` merge tag.
