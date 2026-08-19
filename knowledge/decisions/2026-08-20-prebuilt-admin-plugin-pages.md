# 2026-08-20 Prebuilt Admin Plugin Pages

## Status

Accepted and implemented for admin page bodies.

## Context

Manifest V3 can declare extension admin paths and inject their navigation into
the Host admin shell, but the shared route renders only Host-owned `about` and
`settings` views. The existing prebuilt Admin Micro-frontend API is restricted
to the Settings Document, so a plugin cannot ship an ordinary dashboard,
table, inspector, or workflow page without adding Core code.

Operators should retain an upload, trust, and enable workflow without running a
Nuxt build. Plugin authors should be able to write Vue pages and publish their
prebuilt browser output while the Host continues to own admin chrome,
authorization, exact-artifact trust, and failure isolation.

## Decision

- Add `view: component` to `admin.pages[]`. Such a page declares an Admin
  Micro-frontend API v1 component with `id`, `apiVersion`, package-local
  prebuilt `.mjs` entry, and optional `.css`.
- Keep the Host-owned dynamic admin route and `layout: admin`; an extension
  component owns only the active page body and cannot replace the sidebar,
  topbar, tabs, route middleware, or Nuxt configuration.
- Extend the immutable `adminFrontendDigest` over every declared settings and
  page component, including its surface identity, contract, entry bytes, and
  CSS bytes. Component IDs are unique within one extension.
- Serve page assets through the private content-addressed namespace with an
  explicit component ID. Retain the settings component's legacy `entry` and
  `style` aliases for compatibility.
- Reuse exact-artifact executable trust. A page component mounts only while the
  plugin is enabled, Safe Mode is off, the exact package is trusted, the actor
  can view extensions, and any page-specific permission is granted.
- Add a page-specific API v1 bridge for locale, appearance, page identity,
  namespaced extension requests, navigation, and Host Toasts. It exposes no
  Cookie or Authorization values.
- Import, CSS, mount, cleanup, or repeated browser-session failures remain
  isolated to the component boundary and never remove the Host admin shell.

## Consequences

- A plugin may own multiple first-class admin pages without becoming a Nuxt
  Layer or triggering a Host rebuild.
- Plugin authoring tools may compile Vue SFC source into this contract; the
  installed production package consumes only prebuilt output.
- Direct imports from SForum's private Vue components and composables remain
  unsupported. A separate versioned Plugin UI SDK can build on this page
  boundary without exposing Host internals as an ABI.
- Public plugin page shell inheritance remains a separate follow-up because it
  must preserve active-theme SSR and chrome ownership.
