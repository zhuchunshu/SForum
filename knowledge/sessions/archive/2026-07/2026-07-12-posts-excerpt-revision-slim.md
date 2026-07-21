# 2026-07-12 Session Handoff

## Changed

- Dropped stored `posts.excerpt`; list/detail/moderation excerpts are derived
  from `plain_text` at read time using `forum.reading.excerpt_rune_limit`.
- Slimmed `post_revisions` to source-only snapshots: `raw_content` plus
  source/editor/render metadata and `content_hash` (no html/plain/excerpt).
- Migration: `202607120008_posts_drop_excerpt_slim_revisions.sql`.
- Forum store/service, profile recent topics, and moderation workbench SQL
  updated; OpenAPI response `excerpt` fields remain (derived, not storage).

## Decisions

- Keep `raw_content` + `html_content` + `plain_text` on `posts` for now
  (read-heavy forum path). Only remove the fully derivative `excerpt` column
  and stop duplicating derived fields in revisions.

## Next

- If disk becomes a concern at scale: revision retention policy, then
  optional html cache outside the main row.

## Open Questions

- None for this slice.
