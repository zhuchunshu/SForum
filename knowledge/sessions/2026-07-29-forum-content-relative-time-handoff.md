# 2026-07-29 Forum Content Relative Time Handoff

## Changed

- Topic detail and comment read models now expose optional `editedAt`, sourced
  from the current accepted revision rather than lifecycle-sensitive
  `topics.updated_at` / `comments.updated_at`.
- Edit-mark settings mask both `edited` and `editedAt`; topic/comment cache
  namespaces were advanced so old payloads cannot hide the new field.
- The topic detail UI now labels publish/update times, uses seconds, minutes,
  hours, days, then one month for the first 30 days, and displays older values
  as site-timezone `Y-m-d H:i:s`. A single client-side clock recalculates the
  visible labels every second without refetching topic or comment data.
- OpenAPI, frontend types, bilingual copy, and focused unit/static contracts
  were updated.

## Decisions

- 28-30 days render as `1 month ago`; values older than 30 days render as an
  absolute timestamp. Other site-wide date formatting behavior is unchanged.

## Next

- Run focused Go/Bun tests, OpenAPI reference validation, and manual topic
  detail checks for unedited/edited topic and comment timestamps.

## Open Questions

- None.
