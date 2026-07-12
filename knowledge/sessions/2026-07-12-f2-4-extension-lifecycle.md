# 2026-07-12 Session Handoff

## Changed

- **F2.4 Extension lifecycle** (upgrade / uninstall / migration ledger / disable drain).
- Same-id ZIP upload is an upgrade: drain plugin runtime, `SaveInstalled` resets
  status to `installed`, record declared migrations, revoke frontend trust when
  `packageDigest` changes, emit `upgraded` + audit `extension.upgrade`.
- `DELETE /admin/extensions/{id}` uninstall (must disable first; builtin/system
  blocked). Optional `retainPackage`; settings CASCADE delete in v1
  (`retainSettings` audit-only).
- Migration ledger table `extension_migration_ledger`; v1 records path+checksum
  only (no host SQL execution of plugin files).
  `GET/POST .../migrations` and `.../migrations/apply`.
- Disable drains runtime (stop subprocess, clear mail provider selection, emit
  disabled hook) before DB status change.
- Admin UI: upgrade toast metadata; uninstall button + confirm modal on plugins /
  themes / overview.
- OpenAPI + i18n messages for new error codes.

## Decisions

- Plugin migrations stay **record-only** until a safer executor exists; arbitrary
  plugin SQL against the host DB is out of scope for v1.
- Upgrade always re-enable: trust re-approval when digest changes; capability
  confirm still applies on next enable (F2.1).

## Next

- F3 outbox / idempotency / webhooks, or product Iteration A.
- Optional: independent settings backup table so `retainSettings` can truly keep
  config across uninstall.
- Optional: schedule registry cleanup on disable for plugin-declared schedules
  once plugins own schedule grants.

## Open Questions

- Should theme upgrade while active force deactivation UI copy beyond status reset?
- Future migration executor ownership (host-run vs plugin RPC)?
