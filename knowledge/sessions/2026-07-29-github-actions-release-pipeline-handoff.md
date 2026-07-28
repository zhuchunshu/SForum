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
- Hardened the Route Registry's final redirect response boundary to reject
  backslashes, preventing browser normalization of `/\\host/...` into an
  external network-path redirect even if an invalid execution plan bypasses
  declaration-time path validation.
- Replaced unchecked database pool `int` to `int32` casts with bounded parsing
  for API and worker min/max connection environment variables. Values outside
  `1..2147483647` now follow the existing invalid-config fallback contract.
- Hardened stored Argon2id hash verification: `m`, `t`, and `p` now use
  unsigned width checks plus the current Host cost ceilings before derivation,
  and salt/key lengths are rejected before Base64 allocation when they do not
  match the generated format. Values that previously wrapped back to valid
  `uint32`/`uint8` parameters no longer reach Argon2.
- Removed unpatched `github.com/disintegration/imaging v1.6.2` from Core and
  the media optimization fixture. JPEG/PNG transforms now use
  `golang.org/x/image/draw`; bounded `rwcarlsen/goexif` orientation reads retain
  phone-photo rotation, and TIFF is explicitly rejected before transform.
- Hardened uploaded extension ZIP entry handling for CodeQL alert #12
  (`go/zipslip`). The archive reader now rejects `..` directly at the tainted
  entry-name boundary, plus NUL, absolute/UNC, and Windows drive paths before
  normalization. Shared manifest path validation rejects cross-platform
  absolute forms, while immutable snapshot normalization remains an
  independent second boundary against traversal, links, special files,
  duplicates, and file/directory collisions. ZIP reading and shared archive
  path helpers now live in focused same-package files, lowering both affected
  legacy architecture baselines instead of growing the oversized files.
- Dismissed CodeQL alert #13 (`go/weak-sensitive-data-hashing`) as a false
  positive. The reported value is a CSPRNG-generated 256-bit PostgreSQL role
  credential; SHA-256 is retained only as a non-secret audit/rotation
  fingerprint and is never used to verify a user-selected password.
- Dismissed CodeQL alert #14 (`go/weak-sensitive-data-hashing`) as a false
  positive. The short-lived ALTCHA solution has already passed HMAC signature,
  expiry, and proof-of-work validation; SHA-256 only derives the Redis `SETNX`
  replay key so the raw payload is not stored. A slow password hash would add
  attacker-controlled CPU cost without strengthening this boundary.
- Resolved CodeQL alerts #15 and #16 (`go/allocation-size-overflow`) at the
  shared public-asset publication source. Initial declaration capacity now
  uses the fixed 256-item publication limit instead of adding two manifest
  slice lengths; the existing fail-closed output limit is covered by a new
  over-limit regression test.

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
- The redirect backslash regression test, full `Support/Routes` package, and
  HTTP redirect/alias/rewrite focused tests passed.
- Database pool maximum-bound and overflow regression tests passed together
  with the full config and PostgreSQL pool packages.
- The full Identity model package passed with regression coverage for Argon2
  zero, cost-limit, integer-wrap, minimum-memory, salt-length, and key-length
  rejection while normal and wrong-password verification remain green.
- Core Attachments and the media optimization fixture tests and `go vet`
  passed. Tests cover eight EXIF transforms, a real orientation-bearing JPEG,
  output dimensions, and TIFF rejection. Both focused `govulncheck` runs found
  zero reachable vulnerabilities, and neither module graph contains
  `disintegration/imaging`. The real media optimization plugin subprocess
  integration test also rebuilt, started, and completed successfully.
- The complete `Models/Extensions`, `ExtensionManifest`, and
  `ExtensionPackage` test packages passed after the Zip Slip hardening, as did
  focused `go vet`. Regression cases cover POSIX traversal and absolute paths,
  backslash traversal, UNC paths, Windows drive paths, NUL, and the deliberate
  strict rejection of entry names containing embedded `..`.
- The focused public-asset allocation regression, full `Models/Extensions`
  package, and focused `go vet` passed after resolving alerts #15/#16.
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
- Push the Zip Slip boundary change and confirm CodeQL alert #12 closes on the
  next scan; it remains open remotely until that scan observes this code.
- Push the public-asset allocation change and confirm CodeQL alerts #15/#16
  close on the next scan; they remain open remotely until that scan observes
  this code.
- Review CodeQL alert #17 (settings lifecycle clone allocation size)
  separately, after operator confirmation.

## Open Questions

- Automatic recording of the previously deployed image digest and guarded
  one-click application rollback remain open. Database down-migration is not
  implied by image rollback.
