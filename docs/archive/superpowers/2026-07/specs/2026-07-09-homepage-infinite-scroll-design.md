# Homepage Infinite Scroll Design

## Goal

Replace the default theme homepage's manual pagination controls with automatic infinite scrolling, and make both desktop side rails reliably sticky while the central topic feed grows.

## Scope

- Change only the built-in default theme homepage and its theme-level CSS.
- Keep the existing forum topic list and search APIs. No backend endpoint, permission, OpenAPI, or data model change is required.
- Keep `SFPagination` available for category pages, tag pages, comments, admin tables, and component previews.

## Interaction Design

- The first topic page remains SSR-loaded through `useAsyncData`.
- On the client, the homepage stores loaded topics in a feed array. A bottom sentinel observed by `IntersectionObserver` requests the next API page when it enters view.
- Search, category, or tag changes reset the feed to page 1, replace the loaded topics with the fresh first page, and resume automatic pagination from page 2.
- The bottom of the feed shows one of four states:
  - loading more topics
  - retry button after a load-more failure
  - end-of-list when all results are loaded
  - no footer when there are no topics

## Layout Design

- Desktop side rails stay sticky with the current visual design.
- Rails get a viewport-bounded max height and internal overflow so long category/tag/sidebar content remains usable without pushing the central feed.
- Tablet/mobile keep the existing single-column behavior and mobile filter chips.

## Testing

- Update `apps/web/tests/defaultThemeHomepage.test.ts` before implementation.
- The test should assert that the homepage no longer renders `SFPagination`, includes an infinite-scroll sentinel/state area, uses `IntersectionObserver`, and gives sticky rails viewport-bounded overflow styles.
- Run the homepage test first to observe the expected failure, then implement the minimal page and CSS changes.
- After implementation, run the homepage test, relevant forum frontend tests, and typecheck when practical.

