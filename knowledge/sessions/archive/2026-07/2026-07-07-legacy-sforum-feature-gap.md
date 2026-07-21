# 2026-07-07 Session Handoff

## Changed

- Added `knowledge/legacy-sforum-feature-gap.md`, a cross-module inventory of
  SForum-old features that are missing or only partially implemented in the
  current rewrite.
- Linked the new inventory from `knowledge/index.md`.

## Decisions

- Treat the old SForum feature set as migration input, not as architecture to
  copy directly.
- Delay SForumData importer implementation until enough target modules exist
  to avoid silent data loss.
- Classify old capabilities by migration impact: migration-blocking, high-value
  product gaps, plugin/framework gaps, lower-priority legacy features, and admin
  operations.

## Next

- Use the suggested build order in `knowledge/legacy-sforum-feature-gap.md`
  when selecting the next implementation slice.
- Before importing data, define explicit mappings for old `user_class`,
  `topic_keywords`, attachments, password hashes, and unsupported commerce or
  private-message records.

## Open Questions

- Which old deployment data matters most: core forum content only, or full
  social/commerce/private-message history?
- Should legacy keywords become tags, search metadata, or a separate feature?
- Should paid posts and wallet balances remain product scope for the rewrite,
  or be archived/exported without reimplementation?
