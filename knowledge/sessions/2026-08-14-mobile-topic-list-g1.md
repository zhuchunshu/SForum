# 2026-08-14 Mobile Topic List G1 + Desktop C1 + Topbar Search

## Changed

- Approved the G1 mobile white-flat topic list and the C1 desktop topic list
  directions from `tmp/demos/topic-list-20260814` (mobile) and
  `tmp/demos/topic-list-pc-20260814` (desktop), then implemented both in the
  shared `SFHomeTopicRow` and the public home feed.
- `SFHomeTopicRow.vue` is now a single responsive row:
  - Mobile (<=720px, G1): badges inline before the title, meta shows
    `[avatar] author ← lastReply · lastActivity` (or `author · created` when no
    replies), category pill in the meta row, neutral reply badge, no tags.
  - Desktop (>=721px, C1): badges on their own line above the title, title line
    is `[category pill] title`, meta left shows
    `[avatar] author ← lastReply · lastActivity`, meta right shows up to three
    tags plus the neutral reply badge. The old five-column table grid was
    removed. Desktop inline meta avatar is 28px.
- Mobile home feed list switched from the D soft-panel cards (14px radius,
  shadows, grid gap) to a flat white list with hairline separators
  (`sforum-home.css` `@media (max-width:720px)`). Theme artifact
  `hybrid-forum.css` mobile topbar-height and desktop row rules updated to match.
- Mobile search moved from an always-visible 52px bar below the topbar to a
  topbar search icon (`navbar__mobile-search-trigger`) that opens a fixed
  slide-down panel (`SFPublicMobileSearchBar` gains `open`/`close`). The mobile
  navbar is now a single 54px row; `--sf-public-topbar-height` at `<=980px` was
  reduced from 108px to 54px in `sforum-theme.css` and the theme artifact.

## Decisions

- The approved directions are G1 (mobile) and C1 (desktop), not the earlier D
  soft-panel cards or the old five-column desktop table. Mobile topbar order is
  search, appearance, sidebar/avatar; compose stays in the fixed bottom
  navigation.
- Mobile rows no longer show tags (approved demo decision); category remains a
  link pill. Desktop C1 shows at most three tags in the meta right side.

## Verification

- `cd apps/web && bun test` — 883 passed, 0 failed (incl. updated
  `defaultThemeHomepage.test.ts` row contract for the C1 desktop structure).
- `cd apps/web && bun run typecheck` — passed.
- `cd apps/web && bun run build` — passed.
- `node tests/validate-architecture-boundaries.mjs` — passed (SFNavbar 941
  lines, under the 1000 gate).
- `go test ./cmd/sforum -run 'TestValidateTemplateRuntime|TestExtension'` and
  `sforum extension test extensions/builtin/themes/sforum-default` — passed.
- Headless Chrome at 430px and 1280px against the port-3000 dev server
  confirmed: 54px navbar, search trigger visible, slide-down panel opens on
  click, flat rows (0 radius, transparent bg, 1px hairline), replied rows show
  the `←` arrow, mobile keeps G1 layout (badges inline, category pill, no
  tags), desktop shows C1 (badges line, category before title, tags, 28px
  avatar), and no horizontal overflow.

## Next

- Rebuild built-in theme staging
  (`./scripts/build-builtin-plugins.sh`) is already run; the running air API
  needs a restart so `SyncBuiltins` stages the new theme digest, then the
  staged default theme must be re-activated through the admin flow for the
  `hybrid-forum.css` changes to take effect in the immutable artifact. Host CSS
  (`sforum-home.css` / `sforum-theme.css`) already applies live in the web dev
  server.

## Open Questions

- None for the approved G1 mobile / C1 desktop topic list directions.

