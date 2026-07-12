# 2026-07-12 Session Handoff — F4.3 Contribution point expansion

## Changed

- Catalog points + payload types in `ExtensionManifest`:
  - `forum.composer.toolbar` → `extensionRoute`
  - `forum.profile.tabs` → `profileSection` (`extensionRoute` | `hostLink`)
  - `admin.dashboard.widgets` → `dashboardLink` (`adminLink` + admin route)
  - `system.health.checks` → `healthDescriptor` (`static` | `extensionRuntime`)
- Runtime providers in `app/Providers/*_contributions.go` (+ forum composer)
- API: `GET /composer/toolbar`; profile/overview fields; `/ready` merge
- Web: composer toolbar buttons, profile tabs, admin overview widgets
- OpenAPI + regenerated `docs/extensions/catalogs/contribution-points.md`
- Authoring guide table for F4.3 points
- Extension contribution-point list tests updated for 11 catalog points

## Decisions

- No executable JSON; all new points are descriptor-only.
- Health checks never invoke plugin RPC on the ready path (runtime state map only).
- Dashboard widgets only allow admin-shell relative routes (no external URLs).

## Next

- F4.4 entity meta / custom fields
- F4.5 feature flags vs permissions
- Or product Iteration A / settings Wave 3

## Open Questions

- Whether composer toolbar should later support trusted editor components
  instead of only extensionRoute actions.
