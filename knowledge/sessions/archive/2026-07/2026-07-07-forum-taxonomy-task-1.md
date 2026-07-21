# 2026-07-07 Forum Taxonomy Task 1 Handoff

## Changed

- Added migration `202607070003_forum_taxonomy.sql` for category groups,
  category ordering/default sort, tags, topic-tag joins, forum taxonomy runtime
  options, and the `tag.manage` permission grant for `super_admin`.
- Added `tag.manage` to the identity seed permission catalog.
- Added forum runtime option names, defaults, public/admin visibility, permission
  boundaries, and validation in the Options service.
- Added focused tests for forum option defaults/public exposure/admin
  permissions/validation, migration embedding, and the tag permission seed.

## Decisions

- `tag.manage` is inserted by migration as well as the Go seed catalog so
  existing databases receive the new permission without relying on test-only
  seed data.
- `forum.default_category_slug` is validated as slug-shaped text in Options.
  Category existence resolution is deferred to the Forum settings/domain task.

## Verification

- `go test ./app/Models/Options ./database/migrations ./app/Models/Identity`

## Next

- Continue with Task 2 in
  `docs/superpowers/plans/2026-07-07-real-forum-categories-tags.md`.

## Open Questions

- None for Task 1.
