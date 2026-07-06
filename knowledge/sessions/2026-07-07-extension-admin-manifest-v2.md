# 2026-07-07 Extension Admin Manifest v2

## Changed

- Added `admin.entry` and `admin.pages[]` support to extension manifests.
- Kept legacy `adminPages` compatible for existing packages.
- Added `menu` to admin page declarations, defaulting to false.
- Updated extension admin navigation to include only explicit `menu: true`
  pages from enabled plugins and the active theme.
- Updated list-row Manage actions to resolve through manifest entry,
  `/settings`, first declared page, or generated `/about`.
- Updated OpenAPI and `sforum make:*` scaffolds for the v2 admin shape.

## Decisions

- New scaffolds should emit `admin.entry` and `admin.pages[]`, not legacy
  `adminPages`.
- Generated `/about` remains a management fallback but is not a sidebar item.
- `admin.entry` must stay inside the host admin shell and target `/about` or a
  declared admin page.

## Next

- Add safe manifest content pages when the host-rendered content model is
  designed.
- Start the `mail.provider` vertical slice after plugin observability and
  failed-enable rollback are planned.

## Open Questions

- Whether legacy `adminPages` should emit a deprecation warning during upload
  once extension package diagnostics exist.
