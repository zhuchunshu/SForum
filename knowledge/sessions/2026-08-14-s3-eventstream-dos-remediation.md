# 2026-08-14 S3 EventStream DoS Remediation Handoff

## Changed

- Upgraded `extensions/builtin/plugins/sforum-storage-s3/backend` AWS SDK v2:
  `service/s3` v1.88.6 → **v1.97.3**, transitive `aws/protocol/eventstream`
  v1.7.2 → **v1.7.8**, core `aws-sdk-go-v2` v1.39.6 → v1.41.5, smithy-go
  v1.23.2 → v1.24.2 via `go get` + `go mod tidy` (no manual eventstream pin).
- Fixed GO-2026-5764 (AWS EventStream decoder DoS) in the Govulncheck
  `storage-s3` matrix cell that the 1.26.6 toolchain upgrade surfaced.
- Patch-bumped `sforum.storage-s3` to **1.0.3** and refreshed
  `tests/builtin-plugin-release-baseline.json` because `backend/go.mod` +
  `go.sum` are part of the plugin source contract digest.
  **Correction (2026-08-14):** the committed `sourceDigest` for 1.0.3 was
  wrong (`2ec05393…`); it did not match the actual committed storage-s3 source
  (`6a035d46…`), which broke the next CI Quality gate. The release baseline
  was regenerated correctly at **1.0.4** — see
  `sessions/2026-08-14-builtin-s3-baseline-drift-fix.md`.
- No business code changes: the backend's Put/Get/Head/Delete/Presign usage
  compiles and vets unchanged against the new SDK.

## Decisions

- Upgraded the whole AWS SDK graph with the official service/s3 floor instead of
  only patching the indirect eventstream dependency.
- Did not bypass the scan: no `continue-on-error`, no module exclusions, no
  stdlib exclusions, no govulncheck downgrade.

## Verification

- `go test ./...` in the backend: pass (no test files).
- `govulncheck ./...` (Go 1.26.6): 0 reachable vulnerabilities; the single
  remaining module-level finding is the pre-existing unreachable
  GO-2026-5932 `x/crypto/openpgp` advisory shared by all built-ins.
- `./scripts/build-builtin-plugins.sh`: all 7 built-ins build + digest-refresh +
  `extension test` pass; storage-s3 staged at 1.0.3.
- `node tests/validate-architecture-boundaries.mjs`: pass.
- `./scripts/test.sh`: exit 0 (release gates, Go tests, compat farm,
  proto/OpenAPI, web typecheck/unit, V3 catalogs).
- Final versions confirmed in go.mod/go.sum: `service/s3` v1.97.3,
  `eventstream` v1.7.8.

## Next

- Push and confirm the full Security workflow (Govulncheck matrix incl.
  `storage-s3`) passes on GitHub Actions.
