# 2026-10-24 S3-Compatible Storage Cloudflare R2 Compatibility

## Changed

- Built-in `sforum.storage-s3` 1.0.9 → 1.0.10; the plugin now actually works
  against Cloudflare R2 with only endpoint/bucket/API-token inputs.
- Root causes fixed in `extensions/builtin/plugins/sforum-storage-s3/backend/s3.go`:
  - aws-sdk-go-v2 (`service/s3` v1.97.x) defaults `RequestChecksumCalculation`
    to `when_supported`, so every `PutObject` attached `x-amz-checksum-crc32`;
    R2 rejects that header with 400 `NotImplemented`, which failed the probe
    (probe starts with a PutObject) and every upload. The client now uses
    `RequestChecksumCalculation`/`ResponseChecksumValidation = when_required`.
  - SDK default Virtual-host addressing breaks R2
    (`<bucket>.<account>.r2.cloudflarestorage.com` has no DNS). `use_path_style`
    became a tri-state `auto|true|false` select; `auto` (new default) resolves
    Path-style for any configured non-`amazonaws.com` endpoint and
    Virtual-host for AWS. Legacy stored `"true"/"false"` values keep working.
  - Empty region now resolves to `auto` for R2 endpoints (AWS keeps
    `us-east-1` fallback); explicit values always win.
  - `Open` no longer bounds the response body lifetime with the 60s operation
    context: large proxied attachments were truncated mid-stream at 60s. A
    shared `*http.Client` carries dial/TLS/response-header timeouts instead;
    Put is bounded at 120s to match the manifest provider timeout.
- `manifest/settings.json`: `use_path_style` boolean → select (Auto
  recommended / Path-style / Virtual-host) using the existing
  `SFExtensionSettingsField` USelect renderer and Host options validation
  (same shape as the SMTP `encryption` field); region/public-base-URL copy now
  documents R2 (`auto`, custom domain / r2.dev). No Host code changes; the
  provider slot, SecretStore, and instance protocol are untouched.
- Plugin README gained a Cloudflare R2 quick-configuration section.
- Artifact chain: `backend/plugin` rebuilt
  (`go build -trimpath -buildvcs=false -ldflags="-s -w"`), source manifest
  digest refreshed via `extension digest --write` (backend
  `c9b57862…`), `extension validate` + `extension test` pass (6 checks,
  0 errors), `tests/builtin-plugin-release-baseline.json` regenerated with
  `tests/validate-builtin-plugin-versions.mjs --write`.

## Decisions

- `decisions/2026-10-24-s3-compatible-auto-addressing.md` records the tri-state
  addressing + when-required checksum posture and the rejected alternatives
  (provider preset dropdown, global path-style default, multipart/direct
  upload stays out of scope).

## Verification

- New `backend/s3_config_test.go` asserts resolved client options through
  `client.Options()`: R2 endpoint → Path-style + region `auto` + checksums
  when_required; explicit overrides win; legacy booleans parse; AWS
  (no endpoint / `amazonaws.com`) stays Virtual-host; prefix and public URL
  joins unchanged; required-field validation preserved.
- `go test ./...` (plugin backend) green; architecture boundary gate green;
  `./scripts/test.sh --fast` gate run recorded in this session.

## Next

- **Live R2 QA completed (2026-10-24 ~05:50)** with operator credentials in the dev
  browser: R2 bucket `sforum`, endpoint `…r2.cloudflarestorage.com`, region
  `auto`, addressing auto, public base URL `https://assets-sforum.dalao.me`.
  Draft probe returned `ok=true / storage.ok` (real Put/Head/Get/Delete round
  trip, 3.6s); instance saved + activated (`instance:6816e783…` is the current
  writer); real upload through `POST /api/v1/attachments` returned 201 (×3,
  object key `2026/10/24/….png`); `/media/attachments/{publicId}` redirected
  (302) to a valid R2 presigned URL whose SigV4 scope carries the `auto`
  region, and the browser rendered the object from the R2 endpoint; the
  operator's R2 custom domain served the object directly (200 image/png).
  Three test attachment rows were soft-deleted afterwards; their R2 objects
  remain until orphan cleanup retention.
- Diagnostic notes for future sessions: the draft-probe button click must be
  issued atomically (Vue remounts reset draft fields under concurrent edits —
  fill + verify + click in one script); repeated failed probes trip the
  per-extension resilience circuit (5 failures → 30s open,
  `extension.circuit_open`); page-side `fetch()` against `/media/…` fails with
  `TypeError: Failed to fetch` because the route 302-redirects cross-origin to
  the R2 presigned URL without CORS headers — `<img>`/navigation loads work,
  so this is a diagnostic artifact, not a product defect.

## Open Questions

- None. If the operator's live probe reports a different R2 error, handle it
  on top of the raw message surfaced by the probe UI.
