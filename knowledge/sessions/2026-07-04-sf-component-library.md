# 2026-07-04 Session Handoff

## Changed

- Split the Pine Teal forum component demo into reusable Nuxt components under
  `apps/web/app/components/` with uppercase `SF` prefixes.
- Added `SFAlert`, `SFBadge`, `SFToast`, `SFTabs`, and supporting forum
  primitives such as `SFButton`, `SFCard`, `SFInput`, `SFAvatar`, `SFFeedRow`,
  `SFComment`, `SFSearch`, `SFEditor`, `SFPagination`, `SFProgress`,
  `SFSkeleton`, `SFEmptyState`, and `SFToggle`.
- Expanded `apps/web/app/assets/css/sforum-components.css` with Pine Teal
  component styles and a responsive component preview layout.
- Added a dev-only `/components` page that previews the component library.
- Added `tests/validate-sf-components.js` and updated the demo validation file
  list to match the current demo assets.

## Decisions

- `SF` is the canonical component prefix. Older `Sf` references are superseded.

## Next

- Use `SF*` components in actual forum pages once page skeletons are added.
- Decide which components should eventually wrap Nuxt UI primitives.

## Open Questions

- Whether the component preview page should gain copyable code snippets again
  after the API settles.
