# 2026-07-21 Plugin process RSS on admin lists

## Changed

- `runtime.memoryBytes` on extension List/Detail (plugin backend subprocess OS RSS)
- Shared sampler: `apps/api/app/Support/ProcessMemory` (AdminOverview family KPI reuses it)
- Admin Plugins list badge + extensions index detail row
- OpenAPI `ExtensionRuntimeStatus.memoryBytes`

## Decisions

- Basic only: show when a owned `backend/plugin` child is visible; omit otherwise
- One `ps` sample per list request; attribute by path `storage/extensions/<id>/…`
- Not full host-side memory attribution (caches in API process stay on the host)

## Next

- Optional: PID, refresh interval, or package disk size
- Optional: themes remain “no process” forever unless a backend theme appears

## Open Questions

- None for this basic surface
