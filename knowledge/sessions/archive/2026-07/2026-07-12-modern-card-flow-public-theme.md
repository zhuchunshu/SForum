# 2026-07-12 Modern Card Flow Public Theme

## Changed

- Replaced the default public Hybrid workbench with the screenshot-aligned Modern Card Flow shell.
- Added a real-data homepage overview, 768px card feed, 300px activity sidebar, and API-backed topic cards.
- Changed topic detail to a 760px reading card with a 240px sticky contents/progress rail and card-based top-level replies.
- Reused topic cards on category/tag pages and unified remaining public page backgrounds and containers.

## Decisions

- Work directly on `main` using atomic commits while excluding concurrent admin-extension changes.
- Do not add missing likes, online counts, weekly rankings, subscriptions, or last-replier data.
- Keep the runtime appearance preset authoritative for color while matching screenshot layout and density.

## Verification

- Focused theme tests: 43 passed, 0 failed.
- Browser QA passed at 1440x900 and 390x844 with no relevant console warnings or errors.
- Desktop homepage measured 1140px overall, 768px + 300px tracks, 150px cards, and a 106px overview; topic detail measured 760px + 240px.
- Mobile document width matched client width; the topic rail hid and comment cards measured 343px.
- Full gate passed Go tests and 1140 OpenAPI references, then stopped on unrelated `apps/web/app/utils/adminExtensions.ts:373,375` type errors.

## Next

- Re-run `./scripts/test.sh` after the concurrent admin-extension task resolves its type errors.
- Remember that an activated uploaded Layer may intentionally override these default-theme pages.

## Open Questions

- None for the Modern Card Flow default-theme scope.
