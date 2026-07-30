# 2026-07-31 Responsive Public Sidebar Handoff

## Changed

- Public desktop and mobile left navigation consume the same
  `public.sidebar.primary` items.
- `SFPublicSidebarContent` is the single renderer for compose, links, active
  state, dynamic categories, counts, contextual slots, and the About entry.
- The mobile drawer loads Forum taxonomy only while open and renders the shared
  sidebar content in route mode.
- Personalization no longer exposes a separate mobile navigation editor
  location; mobile follows sidebar.
- Legacy `public.mobile.primary` API/document data remains compatible and is
  not deleted.

## Decisions

- Responsive presentation does not create a second navigation placement
  authority. See `decisions/2026-07-31-responsive-public-sidebar.md`.

## Verification

- Focused Bun tests: 47 passed, 0 failed.
- Follow-up taxonomy and visual-matrix regression tests: 11 passed, 0 failed.
- Full Web Bun suite: 796 passed, 0 failed.
- Nuxt typecheck passed.
- Production Nuxt build completed successfully.
- Architecture boundary validation passed across 1,523 production files.
- Browser QA passed on the active default theme:
  - Desktop and `390x844` home sidebars have the same ordered entries.
  - The mobile drawer is 320px wide, scrolls internally, and has no page-level
    horizontal overflow.
  - Navigating to Categories closes the mobile drawer and reaches
    `/categories`.
  - Desktop and mobile `/settings/profile` sidebars match, including the
    contextual account-settings navigation.
  - Desktop and mobile `/control-panel/personalization?tab=nav` expose Topbar,
    Sidebar, and Footer only; there is no separately editable mobile menu.
  - No relevant browser console errors or warnings were observed.

## Next

- Preserve `public.mobile.primary` as compatibility-only data until API-LTS and
  extension/theme usage evidence permits removal.

## Open Questions

- Remove `public.mobile.primary` only after API-LTS and extension/theme usage
  evidence permits it.
