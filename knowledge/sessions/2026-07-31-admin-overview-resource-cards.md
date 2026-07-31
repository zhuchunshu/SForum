# 2026-07-31 Session Handoff

## Changed

- Added admin overview runtime resource data for API, independent Worker,
  backend plugins, total memory/CPU, and current filesystem disk usage.
- Updated `/control-panel` with one resource row for backend memory, API CPU,
  and 1/5/15-minute system load, followed by one community KPI row for posts,
  users, and pending work.
- Replaced the CPU card's horizontal metric strip with a two-column grid so
  it has no horizontal scrollbar.

## Changed (real-time)

- Added `/admin/overview/resources` lightweight endpoint (process + disk +
  system load only).
- Dashboard now polls:
  - Resource cards (memory/CPU/disk): every **2 seconds**
  - Full overview / KPI cards: every **30 seconds**
- Background polling respects page visibility and KeepAlive; manual refresh still works.
- Omitted fields in resource patch preserve previous values (no flicker to "unavailable").

## Decisions

- Embedded Worker memory/CPU is included in API usage and displayed as
  `并入 API` / `In API`; independent Worker processes remain separate.
- Existing `memoryBytes` and `familyMemoryBytes` response fields remain for
  compatibility. New categorized values are optional under `runtime.resources`.
- Host system load is an optional top-level runtime field and is shown as
  1/5/15-minute run-queue averages, never as a CPU percentage.

## Next

- User will manually verify the dashboard in the running local app.

## Open Questions

- None.
