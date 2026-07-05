# 2026-07-06 Extension Plugin Runtime

## Changed

- Added plugin runtime implementation for declared plugin routes, lifecycle
  hooks, HashiCorp go-plugin startup, and provider slot registry.
- Surfaced plugin runtime status and route/hook/provider counts in the admin
  plugin list.

## Decisions

- Plugins cannot override arbitrary core API routes.
- Provider replacement is allowed only through named core-owned slots.
- Uploaded theme activation remains separate from plugin runtime.

## Next

- Design uploaded theme activation worker separately.

## Open Questions

- None.
