# 2026-07-05 Extension Admin Submenus

## Changed

- Moved extension management out of the System sidebar folder into its own
  admin sidebar folder.
- Added extension admin submenu pages for Overview, Plugins, Themes, Settings,
  and Event Log, all guarded by the existing `extension.manage` permission.
- Extracted frontend extension types/helpers and a shared admin extension
  manager composable so extension pages reuse the same API calls.

## Decisions

- This change does not add backend APIs. The first submenu release reuses
  `/api/v1/admin/extensions`, lifecycle endpoints, and per-extension event
  endpoints.
- Extension settings are currently read-only manifest declarations until a
  dedicated settings CRUD API is designed.

## Next

- Implement extension settings CRUD from manifest-declared settings.
- Add upgrade, rollback, uninstall, marketplace, and signature/trust metadata.
- Replace the reserved plugin route boundary with a real runtime supervisor and
  proxy.

## Open Questions

- Which plugin RPC protocol shape should become the public SDK contract.
- Whether theme activation should run synchronously from admin actions or via a
  queued rebuild/health-check worker.
