# 2026-07-09 Session Handoff - Default Theme Three-Column Homepage

## Changed

- Redesigned the built-in default theme homepage into a high-density three-column forum layout.
- Added a table-oriented `SFFeedRow` variant for compact topic scanning.
- Added default-theme shell CSS for sticky rails, topic table chrome, dark mode, and mobile collapse.
- Added static contract tests for the homepage structure and feed-row variant.
- Fixed responsive cascade issues found in browser QA so desktop hides mobile filters and mobile keeps the reply-count column without horizontal overflow.

## Decisions

- Kept all data sources on existing forum APIs; no backend endpoint was added.
- Kept unavailable Hot/Featured/Following tabs disabled until backend sorting/feed semantics exist.
- Kept mobile category and active-tag filters near the topic table, matching the accepted taxonomy-first mobile design.
- Preserved unrelated worktree changes that appeared during QA instead of reverting or staging them.

## Verification

- `bun test apps/web/tests/defaultThemeHomepage.test.ts apps/web/tests/forumTaxonomy.test.ts apps/web/tests/forumTopic.test.ts`
- `bun run typecheck` from `apps/web`
- Browser QA on desktop through the in-app Browser at `http://127.0.0.1:3000/`
- Mobile QA through Playwright CLI fallback at `390 x 844`, because the in-app Browser viewport override stayed at `1280 x 720`

## QA Notes

- Desktop rendered the sticky left taxonomy rail, central topic table, and sticky right utility rail with no relevant in-app Browser console errors.
- Mobile rendered a single-column topic table with side rails hidden, `scrollWidth=390`, mobile taxonomy filters visible, and only reply stats retained.
- Playwright CLI reported existing hydration mismatch warnings around category/topic counts (`0` on server, populated on client). The rendered layout recovered, but the warning should be investigated separately if SSR data consistency becomes a focus.
- Pagination controls rendered and accepted clicks, but the page-2 data state returned an empty list during QA. This appears data/API-state related rather than a layout regression.

## Next

- Consider applying the same table variant to category and tag listing pages if the homepage direction feels good in real use.
- Add backend-supported hot/featured/following feeds before enabling those tabs.
- Investigate SSR hydration count mismatches and page-2 empty data separately from layout work.

## Open Questions

- Should right-rail stats later read a dedicated public forum overview endpoint instead of deriving from loaded categories/topics?
