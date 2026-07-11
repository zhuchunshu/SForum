# 2026-07-12 V32 暖橙左栏 public theme

## Changed

- Replaced Modern Card Flow homepage/topic shells with V32 structure:
  left 240px category nav, notice + latest tab, dense topic table; topic
  detail dual-column with info side card.
- Updated default theme CSS (`sforum-theme.css`, `sforum-home.css`,
  `sforum-topic.css`), `SFHomeNavigation`, `SFHomeTopicRow`, `SFNavbar`,
  `SFTopicHeading`, new `SFTopicSideCard`, homepage/topic pages, comment
  card radius/branch tokens, zh-CN/en-US strings.
- Updated theme contract tests for V32; avatar contract now expects author
  avatars on homepage rows.

## Decisions

- Accent stays on `appearance.theme` (1A). Warm orange preview: select
  `amber` or `custom:#c2410c` in personalization.
- Participant column/side card shows author only (2A); no fake stacks.
- Unread / ranking / hot / mine / likes / bookmarks / relevance sort not
  rendered (3A).
- `SFTopicProgressRail` kept in package but unmounted from default topic page.

## Verification

- Focused theme contracts (`defaultThemeHomepage` / `Topic` / `Navbar` +
  `unifiedAvatarRendering`): 24 pass, 0 fail.
- `cd apps/web && bun run typecheck`: exit 0 (includes `adminExtensions`
  locale normalization fix).
- Static audit: homepage left-nav + topic table (no hero/right aside);
  topic dual-column + `SFTopicSideCard` (author-only participants);
  category/tag pages share `sforum-home` + `navigation-mode="route"`;
  accent via `appearance.theme` (recommended still `pine_teal`).

## Follow-up (same day)

- Category `/c/:slug` and tag `/tags/:slug` pages now share the V32 left-nav +
  topic-table shell. `SFHomeNavigation` supports `navigationMode="route"` for
  real `/` and `/c/:slug` links outside the homepage filter model.
- Fixed pre-existing `adminExtensions.ts` locale normalization type errors;
  `bun run typecheck` passes.

## Next

- Optional: if product wants warm orange as site default, change
  `recommendedAppearanceTheme` (affects admin Toast primary too).

## Open Questions

- None for the confirmed V32 plan scope.
