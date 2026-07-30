# 2026-07-31 Responsive Public Sidebar Handoff

## Changed

- Public desktop and mobile left navigation consume the same
  `public.sidebar.primary` items.
- `SFPublicSidebarContent` is the single renderer for compose, links, active
  state, dynamic categories, counts, contextual slots, and the About entry.
- `usePublicSidebarDrawer` is the one serializable open/owner authority.
  `SFNavbar` opens the current page owner and only mounts the generic public
  navigation fallback when the page has no desktop left sidebar.
- `SFResponsivePublicSidebar` renders one page sidebar DOM as the desktop rail
  above 980px and as the left drawer at narrow widths. Home/search, taxonomy,
  topic, profile, settings, notification, moderation, and system error pages no
  longer maintain duplicate mobile left-sidebar markup or state.
- Independent right information drawers remain page-owned.
- Personalization no longer exposes a separate mobile navigation editor
  location; mobile follows sidebar.
- Legacy `public.mobile.primary` API/document data remains compatible and is
  not deleted.

## Decisions

- Responsive presentation does not create a second navigation placement
  authority. See `decisions/2026-07-31-responsive-public-sidebar.md`.

## Verification

- Focused responsive-sidebar tests: 107 passed, 0 failed.
- Full Web Bun suite: 797 passed, 0 failed.
- Nuxt typecheck passed.
- Production Nuxt build completed successfully.
- Architecture boundary validation passed across 1,525 production files; the
  `SFTopicShowPage.vue` line-count ratchet was lowered from 1261 to 1249.
- Browser QA passed on the active default theme:
  - `390x844` topbar-menu interaction opens exactly one contextual left drawer
    on home, category index/detail, tag index/detail, topic detail/create,
    profile, settings, notifications, and 404 system-error surfaces.
  - Page owner IDs and contextual content match the corresponding desktop
    rails; the Navbar fallback is absent on owned pages.
  - `/guidelines`, which has no desktop left rail, receives exactly one generic
    Navbar fallback drawer.
  - `1440x900` category QA keeps the 230px desktop rail and does not apply a
    drawer class.
  - All checked mobile and desktop pages have no horizontal overflow.
  - No relevant browser console errors or warnings were observed.

## Next

- Preserve `public.mobile.primary` as compatibility-only data until API-LTS and
  extension/theme usage evidence permits removal.
- Keep new public pages on `SFResponsivePublicSidebar` whenever they introduce
  a desktop left rail; do not add a page-specific mobile copy.

## Open Questions

- Remove `public.mobile.primary` only after API-LTS and extension/theme usage
  evidence permits it.
