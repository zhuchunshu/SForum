# 2026-08-20 Plugin Runtime Memory

## Changed

- Moved plugin-side Protocol V2 go-plugin/gRPC serving into
  `apps/api/sdk/plugin/v2`; Host transport now depends on the SDK boundary.
- Removed every production `sdk/plugin/v2` import of the legacy
  `app/Support/Extensions` package and added an architecture ratchet.
- Replaced Host-owned SEO/provider/mail projections used in plugin processes
  with SDK- or plugin-owned stable types without changing JSON or gRPC shape.
- Removed the unused Protocol V1 content-policy adapter.
- Plugin child environments now force `GODEBUG=disablethp=1` and reject an
  inherited Host `GODEBUG` value.
- Tidied all seven built-in modules, patch-bumped all seven versions, refreshed
  the built-in release baseline, and rebuilt staging exact digests.
- Extracted storage provider behavior into `sdk/plugin/storageprovider`, kept a
  root SDK compatibility facade, and migrated FS/S3 off the legacy root SDK.
- Extended admin runtime diagnostics with complete-frame 60-second PSS medians,
  per-extension PSS medians, and Linux `AnonHugePages` attribution.

## Evidence

- Protocol V2 SDK dependencies: 396 -> 164; SMTP: 397 -> 165.
- SMTP Linux/amd64 stripped binary: 23,011,490 -> 15,618,210 bytes.
- Isolated Linux SMTP PSS with THP disabled: 27,360 -> 19,284 KiB. The original
  no-THP-control process measured about 30 MiB.
- Storage FS dependencies: 396-class legacy graph -> 166; Linux binary
  22,589,602 -> 15,564,962 bytes; isolated PSS 26,112 -> 19,072 KiB.
- Storage S3 dependencies: 256 after retaining AWS SDK v2; Linux binary
  31,088,802 -> 24,948,898 bytes; isolated PSS 29,172 -> 23,868 KiB.
- All seven module tests and Linux release builds passed. All seven staged
  packages passed digest refresh and `extension test`.
- Focused SDK/Host tests, Host API docs validation, built-in release validation,
  and architecture validation passed.
- Process-memory aggregation/window tests, OpenAPI references, admin overview
  frontend tests, and Nuxt typecheck cover the production acceptance fields.

## Decisions

- Protocol, trust, authorization, process isolation, and lifecycle semantics
  are unchanged; this is a dependency and Go heap mapping optimization.
- Expected production total is about 138-148 MiB rather than 205 MiB, pending
  post-release Linux measurement.

## Next

- Deploy the next release, wait for representative plugin traffic, then compare
  per-plugin 60-second median PSS and `AnonHugePages` with the current baseline.
- Replace the temporary Go `disablethp` compatibility switch with an explicit
  Linux THP policy before the Go runtime removes that switch.

## Open Questions

- Confirm whether production hosts can standardize THP mode to `madvise`
  without affecting unrelated workloads.
