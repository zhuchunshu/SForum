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

Page-specific settings, notification, profile, moderation, and forum drawers
already reuse their desktop sidebar components. The remaining divergence was
the navbar-owned public mobile drawer and its separate editor location.

## Decision

`public.sidebar.primary` is the only product source for the public left
navigation on desktop and mobile. The public web client requests sidebar items
once and renders them through one `SFPublicSidebarContent` component inside the
desktop rail or the mobile drawer container.

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
mode differ. They must not introduce a second content, ordering, permission,
or dynamic-block authority.

## Consequences

- Desktop and mobile left navigation share labels, ordering, visibility,
  dynamic categories, counts, compose behavior, active state, and the About
  entry.
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
