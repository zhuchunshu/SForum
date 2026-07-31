# 2026-07-31 Runtime Memory Governance

## Changed

- API resource sampling now shares a 5-second cached process frame and reports
  60-second RSS medians for API, Worker, plugins, and total; Linux PSS is shown
  only when complete. Plugin rows are sorted by RSS and identify process overlap.
- Embedded Worker accounting is explicit: API includes Worker memory, while the
  Worker row reports `running/concurrency` instead of a made-up MiB value.
- Extension artifact digest validation is streaming; seven built-in backend
  binaries use stripped linker output, reducing the local aggregate by about
  69 MiB.
- pprof is opt-in and loopback-only (`6060` API, `6061` standalone Worker).
- `/control-panel` resource polling is 5 seconds. The mobile toolbar now stacks
  controls and has no horizontal overflow at `390x844`.

## Evidence

- `go test ./...` passed in the real host environment; the concurrent identity
  CAS test passed with `-count=5`.
- Linux `CGO_ENABLED=0` API build and architecture boundary validation passed.
- Browser QA passed at `1440x900` and `390x844`: API/Worker labels, 7-plugin
  descending details popover, no blank/overlay, no horizontal overflow, and no
  fresh-tab console errors or warnings.
- Profile comparison showed the main allocation hotspot was repeated whole-file
  `io.ReadAll` in GuardPolicy digest refresh, not a growing live object set.

## Next

- Keep the pprof flags absent in normal development/production environments.
- For a real incident, enable one loopback profile briefly, collect it, then
  remove the flag and restart; inspect container/plugin limits separately from
  Go `GOMEMLIMIT`.

## Open Questions

- RSS semantics still differ from macOS physical-footprint reporting; operators
  needing host-level truth should pair the dashboard with platform/container
  metrics.
