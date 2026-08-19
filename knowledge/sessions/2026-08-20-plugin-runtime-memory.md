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

## Evidence

- Protocol V2 SDK dependencies: 396 -> 164; SMTP: 397 -> 165.
- SMTP Linux/amd64 stripped binary: 23,011,490 -> 15,618,210 bytes.
- Isolated Linux SMTP PSS with THP disabled: 27,360 -> 19,284 KiB. The original
  no-THP-control process measured about 30 MiB.
- All seven module tests and Linux release builds passed. All seven staged
  packages passed digest refresh and `extension test`.
- Focused SDK/Host tests, Host API docs validation, built-in release validation,
  and architecture validation passed.

## Decisions

- Protocol, trust, authorization, process isolation, and lifecycle semantics
  are unchanged; this is a dependency and Go heap mapping optimization.
- Expected production total is about 150-160 MiB rather than 205 MiB, pending
  post-release Linux measurement.

## Next

- Deploy the next release, wait for representative plugin traffic, then compare
  per-plugin 60-second median PSS and `AnonHugePages` with the current baseline.
- Extract the general storage provider helpers into a lightweight SDK package
  so `sforum.storage-fs` and `sforum.storage-s3` stop importing the legacy SDK.
- Replace the temporary Go `disablethp` compatibility switch with an explicit
  Linux THP policy before the Go runtime removes that switch.

## Open Questions

- Confirm whether production hosts can standardize THP mode to `madvise`
  without affecting unrelated workloads.
