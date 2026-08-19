# 2026-08-20 Moderation Notification And Publication Visibility

## Changed

- Added Core `moderation_pending` in-app/Web Push notification policy and
  transactional fanout for new or requeued pending topics and comments.
- Recipient resolution uses current effective `moderation.review` RBAC,
  preserves `super_admin` authority, honors ordinary user allow/deny overrides,
  excludes the submitter, and deduplicates by target revision and recipient.
- Pending notification targets recheck review permission and open the stable
  pre-publication workbench query; mail rendering uses the same protected path.
- Approval now invalidates Forum Redis read models after commit. Both frontend
  approval surfaces clear homepage async data and persisted feed state, while
  `/` responses use `Cache-Control: no-store`.

## Decisions

- Keep email disabled by default for queue alerts; in-app and Web Push inherit
  enabled recommended defaults and remain operator/user configurable.
- Keep API Redis generation caching as the homepage scale boundary instead of
  shared HTML caching, because publication visibility must be immediate.

## Verification

- `cd apps/api && go test ./...`
- `cd apps/web && bun test` (893 passed)
- `cd apps/web && bun run typecheck`
- `ruby scripts/validate-openapi-refs.rb`
- `node tests/validate-architecture-boundaries.mjs`
- Local runtime: API health and Nuxt homepage returned 200; the homepage
  rendered through the in-app Browser without console errors and its response
  carried `Cache-Control: no-store`.

## Next

- Optional local two-account Browser smoke: submit pending content, verify the
  reviewer alert/workbench link, approve it, then return to the already-used
  browser homepage and confirm immediate visibility.

## Open Questions

- None.
