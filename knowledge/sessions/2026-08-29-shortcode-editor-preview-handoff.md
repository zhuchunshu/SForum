# 2026-08-29 Shortcode Editor Preview Handoff

## Changed

- Replaced reference and protected shortcode NodeViews with the approved
  editor-preview treatment: floating type label, quiet bordered body, and
  selection outline, with no embedded replace/locate/unwrap/delete controls.
- User references now render `SFAvatar`; category references render their
  stored icon and safe six-digit hex color in both the selection dialog and
  node preview.
- Extended the composer reference response/cache/OpenAPI contract with safe
  `avatar`, `icon`, and `iconColor` metadata. Raw user email is not returned.
- Fixed direct shortcode typing in `SFEditor`: self-emitted Markdown is now
  recognized before shortcode import preparation, so typing `[` no longer
  reloads the editor with visible `&#91;` text.
- Removed every shortcode-specific visual rule from Core. The protected
  `sforum-shortcodes@1.3.3` candidate now owns published, protected/fallback,
  editor NodeView, and reference-dialog CSS as the exact
  `frontend/public/shortcodes.css` asset.
- Reworked the published surface after live comparison exposed two defects:
  reference rules targeted a DOM shape that user/category renders do not use,
  and protected fallbacks produced a large nested card. Published blocks now
  share the editor-preview radius, border, gradient, density, and floating type
  label while remaining read-only and single-layer.
- Added the provider-neutral public forum-content style catalog and Nuxt
  loader. It admits only enabled, trusted, healthy, digest-bound blocking CSS,
  loads it with SRI on public pages, and clears it on admin routes.
- Kept the hand-written comparison artifact at
  `tmp/demos/shortcode-editor-preview-20260828/`; no generator is retained.

## Decisions

- Editor NodeViews are contextual document previews, not published cards and
  not miniature toolbars. Existing dialog/keyboard flows own editing actions.
- All built-in reference/protected kinds share this geometry while preserving
  identity-specific avatar, icon, and semantic-color cues.
- Shortcode semantics and security remain Host-owned; concrete presentation is
  plugin-owned and follows plugin enable/disable/upgrade/rollback/Safe Mode.
  See `../decisions/2026-08-29-plugin-owned-forum-content-styles.md`.

## Verification

- Focused Go Forum/Host tests passed.
- Full Web suite passed: 945 tests; typecheck and production build passed.
- Bracket-input regression passed in the focused shortcode suite.
- Browser verification on `/topics/new` confirmed that sequentially typing
  `[reply]内容[/reply]` leaves literal brackets in both editor text and DOM.
- Architecture boundary and OpenAPI reference validation passed.
- Target NodeViews contain no `UButton` or legacy action-container classes.

## Next

- Activate the staged `sforum-shortcodes@1.3.3` exact artifact through the
  normal super-admin confirmation, then verify the catalog, SRI stylesheet,
  published topic, and editor NodeViews in the browser. The active runtime is
  still `v1.3.1`, so the corrected publication CSS is not visible yet.
