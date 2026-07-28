# 2026-07-29 Public Search Route Handoff

## Changed

- Added canonical public `/search?q=...` routing with stable Page Registry ID
  `forum.search` and contract `sforum.page.search@1`.
- Reused the existing forum feed implementation with search-specific route,
  pagination, empty-state, Host island, regions, ViewModel, and SEO behavior.
- Added default and Nocturne search templates, changed navbar/error recovery/
  Schema.org search targets, and redirected legacy `/?q=...` links.
- Added focused source tests and V3 UI catalog identities/derived entries.

## Decisions

- Search is a distinct replaceable public page, not an alias of `forum.home`.
- Search result pages remain `noindex,follow`; empty `/search` does not fall
  through to the ordinary topic list.

## Next

- Tests and Browser QA were intentionally not run at operator request.
- The V3 catalog generator is currently blocked before writes by an unrelated
  missing reviewed identity for `GET /api/v1/admin/site/navigation`; search UI
  catalog outputs were updated in the generator's canonical format.
- Rebuild built-in themes, restart the API, activate the staged default theme
  digest, then verify `/search?q=%E5%95%8A` resolves with
  `data-provider="sforum.default-theme"` and `data-template="1"`.
- Verify navbar and mobile search, hard refresh, pagination/filter retention,
  empty search, and the legacy `/?q=content` redirect.

## Open Questions

- None.
