# Plugin-Owned Forum Content Styles

Date: 2026-08-29

## Context

Structured shortcode output needs stable Host-owned semantic wrappers for
sanitization, authorization, fallback, and theme interoperability. Its concrete
visual treatment, however, belongs to the extension that declares and renders
the shortcode. Keeping shortcode CSS in Core would make plugin disable,
upgrade, rollback, and replacement visually dishonest and would accumulate
product-specific presentation in the host framework.

## Decision

- Core owns only semantic wrapper contracts, generic editor/runtime mounting
  points, exact-artifact validation, and a provider-neutral public content
  stylesheet catalog.
- A plugin may declare a Manifest V3 `style` asset for
  `core.surface.forum-content`. The asset must be an exact package file,
  digest-bound, same-origin, blocking stylesheet.
- The public catalog returns styles only from enabled, trusted, healthy exact
  artifacts. Safe Mode, trust revocation, package drift, invalid CSP, script
  dependencies, or unsupported loading modes fail closed.
- Public Nuxt rendering loads the returned immutable links with SRI. Admin
  routes clear this public style surface.
- The `sforum-shortcodes` plugin owns all concrete shortcode presentation CSS:
  published wrappers, protected/fallback states, editor NodeViews, and the
  reference picker. Core must not add `.sf-shortcode`, `.sf-editor-fallback`,
  `.sf-shortcode-node`, `.sf-protected-node`, or `.sf-shortcode-dialog` visual
  rules.
- Core may continue emitting bounded semantic class names. Plugins still may
  not return arbitrary class, style, or data attributes through content render
  output.

## Consequences

- Enabling, disabling, upgrading, rolling back, or distrusting the plugin also
  controls its visual presentation through the same exact-artifact lifecycle.
- Other plugins can use the same generic style surface without adding their CSS
  to Core.
- If the plugin style cannot be admitted, content remains safe and readable via
  semantic HTML/fallback text, but no plugin-specific decoration is applied.

