# 2026-08-14 Built-in storage-s3 Release Baseline Drift Fix

## Changed

- Repaired the CI Quality gate failure on `./scripts/test.sh` →
  `tests/validate-builtin-plugin-versions.mjs`: only `sforum.storage-s3` had
  source-contract drift; the other 6 built-ins matched.
- Root cause: the previous S3 EventStream DoS commit (04e5bbee9) committed a
  wrong `sourceDigest` in `tests/builtin-plugin-release-baseline.json`
  (`2ec05393…`), which does not match the actual committed storage-s3 source
  (`6a035d46…` recomputed at 1.0.3). The Bun production runtime migration did
  **not** touch `extensions/builtin`, the shared plugin runtime, or plugin
  business code, so it did not cause the drift.
- Patch-bumped `sforum.storage-s3` from **1.0.3 → 1.0.4** in
  `extensions/builtin/plugins/sforum-storage-s3/sforum.extension.json` so the
  already-pushed 1.0.3 baseline would not read as "plugin source changed
  without a version bump" in the cross-commit gate.
- Regenerated `tests/builtin-plugin-release-baseline.json` with the repo's own
  validator (`node tests/validate-builtin-plugin-versions.mjs --write`), which
  now records `sforum.storage-s3` at **1.0.4** with digest
  `b843c0ea55c8d2c8d4a6a870d2996f04f412c7cc749c4d96c10622b17c906b5c`. No
  digest or validator rule was hand-edited.
- No immutable artifact digest refresh was needed: the version bump only edits
  manifest metadata, the backend binary is unchanged, and the committed
  `backend.digest` is stripped by the validator and regenerated inside the
  Docker image. `./scripts/build-builtin-plugins.sh` passes and leaves source
  tree manifests untouched.

## Decisions

- Did not modify the Bun Dockerfile, Nuxt config, or plugin business code to
  satisfy the gate.
- Did not rewrite `main` history; used a patch-bump plus baseline regeneration.

## Verification

- `node tests/validate-builtin-plugin-versions.mjs`: pass (7 contracts).
- Cross-commit gates vs the previous commit (04e5bbee9) and vs HEAD
  (b38fde4e4): pass.
- `go test ./...` and `govulncheck ./...` in
  `extensions/builtin/plugins/sforum-storage-s3/backend`: pass.
- `./scripts/build-builtin-plugins.sh`: all 7 built-ins build + digest-refresh +
  `extension test` pass.
- `node tests/validate-architecture-boundaries.mjs`: pass.
- `./scripts/test.sh` (CI env, `SFORUM_COMPAT_DATABASE_URL` set): exit 0,
  including Go tests, compat farm, proto/OpenAPI, web typecheck/unit, and V3
  catalogs. Local `DATABASE_URL`-scoped shared-DB integration tests were
  cleaned of stale test schemas/roles first.
- `git diff --check` and `git diff --exit-code` (drift rejection): pass; only
  the extension manifest and the release baseline changed.

## Next

- Push and confirm the full Quality gate passes on GitHub Actions for the new
  commit.
- If the Security workflow still flags GO-2026-5932 (`x/crypto/openpgp`), it is
  the pre-existing unreachable advisory shared by all built-ins, not this fix.

## Open Questions

- None for this fix. The old handoff's claim that the 1.0.3 baseline was
  correctly refreshed is superseded by this correction.
