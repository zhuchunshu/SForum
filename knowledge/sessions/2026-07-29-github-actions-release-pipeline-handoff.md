# 2026-07-29 GitHub Actions Release Pipeline Handoff

## Changed

- Added reusable PR/main CI with the full repository gate, complete Web tests,
  production builds, drift rejection, and API/worker/migrate/Web image builds.
- Added CodeQL, dependency review, and `govulncheck` coverage for Core and all
  protected built-in Go plugins.
- Added tag-driven GHCR publication for four `linux/amd64` + `linux/arm64`
  images, including SBOM, provenance, attestation, Trivy blocking scan, exact
  candidate Compose smoke, verified tag promotion, and GitHub Release creation.
- Added `compose.release.yaml` and `deploy.sh --version vX.Y.Z`; release deploys
  pull one exact version and never rebuild application images locally.
- Production Compose now passes `IDENTITY_SUBJECT_HMAC_SECRET`, internal
  PostgreSQL defaults to the actual non-TLS Compose network, and the API image
  includes the protected Web Push plugin.
- Resolved the first Security run failures by upgrading Core and all protected
  built-in Go plugins to `google.golang.org/grpc v1.82.1` and
  `golang.org/x/text v0.39.0`; the GitHub auth plugin also resolved
  `golang.org/x/oauth2 v0.36.0`. All workflow checkouts now use the pinned
  `actions/checkout` v6 commit instead of the Node 20-based v4 action.
- Upgraded all 18 source Go modules from Fiber `v3.0.0-rc.3` to the current
  stable `v3.4.0`. The Host session cleanup boundary now clears authentication
  data explicitly before commit-unknown delete/save compensation because
  Fiber 3.4 preserves in-memory session data when storage deletion fails.
- Upgraded the Buf/Protobuf tool module's indirect `github.com/google/cel-go`
  dependency from `v0.28.1` to `v0.29.0`, resolving the Dependabot CEL field
  visibility advisory. The same tool graph now selects patched
  `golang.org/x/text v0.39.0` and `github.com/klauspost/compress v1.18.7`.

## Decisions

- GitHub publishes artifacts but never deploys to operator-owned hosts.
- Stable aliases move only after scan and real runtime smoke pass.
- See `decisions/2026-07-29-ghcr-multi-platform-release-pipeline.md`.

## Verification

- Passed: actionlint 1.7.12, `bash -n`, `git diff --check`, release Compose
  config/merge (all four application services had `build=none`), Go build,
  Nuxt typecheck, Nuxt production build, architecture validation, the focused
  Route Registry test, and the focused taxonomy navigation test.
- After the dependency remediation, `govulncheck ./...` passed independently
  for Core and all six protected built-in plugin modules with zero reachable
  vulnerabilities. The only module-level advisory is the unmaintained
  `golang.org/x/crypto/openpgp` package, which SForum does not import or call.
- All 18 module graphs select Fiber `v3.4.0`; all 17 plugin, fixture, and
  compatibility module test commands passed. Core `AuthSession` passed after
  adapting the cleanup semantics. The full Core test run compiled and ran the
  HTTP stack but remained non-green because of unrelated dirty-worktree tests
  (extension missing-artifact routing and theme search schema), allocation
  ceilings, and the sandbox-blocked `/bin/ps` test. `git diff --check` passed.
- The tool module contains no ordinary Go packages, so its three declared
  tools were built and executed directly: Buf `1.71.0`, protoc-gen-go
  `v1.36.11`, and protoc-gen-go-grpc `1.6.2`. Binary `govulncheck` on Buf with
  `cel-go v0.29.0`, `x/text v0.39.0`, and `compress v1.18.7` found zero
  reachable vulnerabilities; the CEL, x/text, and S2 advisories are absent.
- A full repository gate rerun reached the PostgreSQL integration suite but was
  interrupted by one transient migration error (`could not open relation with
  OID`) in `TestNotificationReferencePluginEmitsThroughRealBroker`. The exact
  integration test passed on retry in 13.5 seconds. The full gate was not run
  to completion again, so this handoff does not claim a final all-green gate.
- Local Buildx parsed the Dockerfiles but Docker Desktop could not reach Docker
  Hub auth to finish `--check`. Docker Hub metadata confirmed the pinned Go,
  Bun, and Alpine base tags all publish amd64 and arm64 manifests. The first
  GitHub release run remains the authoritative multi-platform build proof.

## Next

- Merge the workflows, then push the first new `v*` tag from a protected tag
  ruleset and observe the complete release run.
- After the first run creates GHCR packages, confirm all four are public and
  linked to `zhuchunshu/SForum`.
- Make the CI and Security checks required after the initial `main` baseline is
  green.
- Decide how to handle the unpatched `disintegration/imaging` TIFF panic
  advisory.

## Open Questions

- Automatic recording of the previously deployed image digest and guarded
  one-click application rollback remain open. Database down-migration is not
  implied by image rollback.
