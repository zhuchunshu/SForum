# Decision: One Responsive Public Sidebar Source

## Status

Accepted

## Date

2026-07-31

## Context

The configurable navigation V1 contract introduced separate
`public.sidebar.primary` and `public.mobile.primary` placements. In practice,
both surfaces are the same left navigation at different viewport widths.
Independent operator configuration and separate renderers allowed ordering,
dynamic categories, counts, compose actions, and footer links to drift.

Even after the data placement was unified, rendering still had two owners.
`SFNavbar` opened a generic navigation drawer, while forum, settings,
notification, profile, moderation, and error pages kept separate state and a
second copy of their desktop sidebar markup. The topbar therefore showed
generic categories where the desktop page showed filters, account sections,
notification types, composer progress, or moderation queues.

## Decision

`public.sidebar.primary` is the only product source for the public left
navigation on desktop and mobile. A page with a desktop left sidebar claims the
shared drawer owner and renders its existing sidebar DOM through
`SFResponsivePublicSidebar`: it is a grid rail above 980px and the left drawer
at 980px or below. The page does not render a second mobile copy.

`SFNavbar` only toggles `usePublicSidebarDrawer`. The shared state contains
serializable `{ open, owner }` data, never components, VNodes, slots, or
callbacks. Owner tokens prevent an old route instance from clearing a newer
page owner during unmount. When no page claims the owner, Navbar mounts
`SFPublicMobileNavigation` as the generic fallback; otherwise the current page
sidebar is authoritative.

Page-specific right information rails remain independent drawers because they
are a different surface and action. They do not share the left-sidebar owner.

The Personalization navigation editor exposes topbar, sidebar, and footer.
Its sidebar description makes the responsive behavior explicit. Operators no
longer create, move, reorder, reset, or maintain a separate mobile placement
through the product UI.

`public.mobile.primary` remains accepted in the V1 API types, stored documents,
snapshots, and portable imports for compatibility. Existing data is inert in
the current public renderer and is not deleted or silently copied into the
sidebar. Removing the compatibility location requires the normal API-LTS and
extension/theme compatibility evidence.

Themes may style the mobile drawer differently because its geometry and input
mode differ. They must not introduce a second content, state, ordering,
permission, or dynamic-block authority.

## Consequences

- Desktop and mobile left navigation share labels, ordering, visibility,
  dynamic categories, counts, compose behavior, active state, and the About
  entry.
- Contextual page sections such as taxonomy filters, account settings,
  notification types, composer progress, and moderation queues are identical
  on desktop and mobile because they are the same rendered DOM.
- Pages without a desktop left sidebar continue to receive the generic public
  navigation fallback from Navbar.
- The public chrome fetches topbar, sidebar, and footer locations; it no longer
  requests an independent mobile location.
- Older backups and clients remain parseable during the compatibility window.
- The legacy mobile placement can only be removed after its compatibility
  users and theme declarations have been audited.

## Supersedes

This decision supersedes only the independent mobile-placement product behavior
in `2026-07-27-operator-owned-public-navigation.md`. The canonical document,
plugin contribution, revision, permission, recovery, and theme-presentation
boundaries remain unchanged.
