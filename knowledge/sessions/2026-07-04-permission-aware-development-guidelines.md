# 2026-07-04 Permission-Aware Development Guidelines

## Changed

- Added a `Permission-Aware Development` section to `AGENTS.md`.
- Added identity module rules for designing and testing permission-protected
  behavior.
- Added a backend route-registration reminder that protected API operations must
  implement backend authorization, with frontend guards used only for usability.
- Updated the knowledge index so future sessions notice the new development
  discipline.

## Decisions

- Treat authorization as part of feature design for new protected routes,
  mutations, admin screens, exports, setting updates, and background action
  triggers.
- Keep API policy checks authoritative. Frontend visibility and route guards
  may mirror permissions, but they are not security boundaries.
- Add new permission keys only for distinct admin-grantable capabilities, and
  update seed data, catalog text, frontend labels, contracts, and knowledge
  notes when doing so.

## Next

- Apply this checklist to forum category/topic/post, moderation, settings,
  search indexing, and future export/import workflows as they are implemented.

## Open Questions

- Which forum-domain actions should receive first-class permission keys during
  the next module buildout?
