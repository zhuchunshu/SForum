# 2026-07-09 Session Handoff

## Changed

- Fixed topic detail route lookup after URL mode switches. The catch-all
  detail page now uses ordered lookup candidates so old `id`, `id_slug`, and
  `slug` links can resolve and then canonicalize to the active
  `seo.topic_url_mode`.
- Fixed a topic detail initialization regression where the canonical redirect
  `watchEffect` could read `isEditing` before the computed was declared,
  causing valid topic pages to fail with `Cannot access 'isEditing' before
  initialization`.
- Added `topicPathLookupCandidates` in `apps/web/app/utils/forumTaxonomy.ts`
  with regression tests for old id+slug links under slug mode, ambiguous
  numeric slug/id paths, and old slug links under id-based modes.
- Added a default-theme topic page regression test to keep edit-mode state
  declared before immediate effects and SEO callbacks read it.

## Decisions

- Only 404 lookup failures fall through to the next candidate. Network/API
  failures still surface as errors so operational outages are not hidden as
  "topic not found".

## Verification

- Browser verification with real seeded topic data passed for
  `/t/6002/topic-11gqufm-6002`: the page canonicalized to `/t/6002`, rendered
  topic/comment content, and switching the comment view produced no framework
  overlay or `isEditing` error.

## Next

- No immediate topic-route fallback follow-up is known beyond the numeric slug
  ambiguity question below.

## Open Questions

- Numeric slug-only paths such as `/t/42` prefer slug lookup in `slug` mode,
  then fall back to ID lookup when the slug is missing. Confirm this remains
  the desired ambiguity rule if operators allow fully numeric topic titles.
