# 2026-07-05 Extension System Foundation

## Changed

- Added extension backend model, manifest validation, safe ZIP extraction,
  install/enable/disable lifecycle, local preflight boundaries, and event logs.
- Added extension database migration and `extension.manage` seed permission.
- Added admin API controller/provider/bootstrap wiring for extension routes.
- Added `EXTENSION_ROOT` config and environment examples.
- Added admin "Extensions" page with upload, list, enable/disable, capability
  summary, and lifecycle event display.
- Updated admin module registry, i18n catalogs, OpenAPI, and knowledge base.

## Decisions

- Extension packages are stored under `EXTENSION_ROOT`, not the attachment
  system.
- Extension settings use dedicated extension tables, not `web_options`.
- The first implementation reserves RPC/plugin route and Nuxt theme builder
  interfaces but does not yet run third-party child processes or web rebuilds.

## Next

- Implement plugin child-process RPC supervisor and route proxying.
- Implement theme activation rebuild/health-check/rollback worker.
- Add extension settings CRUD, uninstall, upgrade, and rollback actions.

## Open Questions

- Which RPC protocol shape should become the stable plugin SDK surface.
- Whether extension packages should require signatures before a public
  marketplace exists.
