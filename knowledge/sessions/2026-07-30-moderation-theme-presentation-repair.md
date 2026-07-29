# 2026-07-30 Moderation Theme Presentation Repair

## Changed

- Split Page Registry `themeable` presentation from plugin `replaceable`
  behavior ownership; moderation remains Core-owned and plugin-closed.
- Added the reviewed `sf-moderation-review` Host island and exact templates to
  the default and Nocturne built-in themes.
- Moved Host full-width navbar geometry into an explicit scoped `SFNavbar`
  variant, removing the CSS cascade that expanded the compose button across
  the viewport.
- Changed the moderation and profile-settings center columns from the darker
  `--sf-public-bg` canvas token to the foreground `--sf-public-surface` token.
- Registered moderation workbench CSS in the initial Nuxt stylesheet set and
  replaced blank navbar hydration placeholders with stable SSR controls/user
  state, preventing the oversized title and empty utility area on hard refresh.
- Refreshed and passed Manifest V3 digest, validate, and extension-test gates
  for both themes.
- Rebuilt and staged the built-in packages, activated the repaired default
  theme through the normal admin flow, and confirmed `/moderation` and
  `/settings/profile` render with `data-provider="sforum.default-theme"` and
  `data-template="1"`.
- Browser QA passed at desktop and `390x844`: the three-column fallback navbar
  no longer overflows, the center surfaces use the expected lighter token,
  responsive controls remain visible, and the moderation history and settings
  navigation interactions update the rendered state. The operator also
  confirmed the repaired pages in their own browser.

## Decisions

- See `../decisions/2026-07-30-themeable-core-workbench-presentation.md`.

## Next

- Operator hard-refresh verification remains for the first-paint title and
  navbar fallback at desktop and `390x844`. Existing typecheck and architecture
  gate failures are unrelated workstreams documented in the current working
  tree.

## Open Questions

- None.
