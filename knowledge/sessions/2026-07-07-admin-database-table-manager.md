# 2026-07-07 Session Handoff

## Changed

- Added a read-only admin database table manager backed by Go `pgx` catalog
  queries and a Nuxt admin page.
- Added `database.manage`, a migration granting it to `super_admin`, and
  frontend admin navigation gated by that permission.
- Added OpenAPI contract files for database table listing, detail, rows,
  per-cell reveal, and CSV export.

## Decisions

- Keep v1 read-only and omit SQL console or row mutation.
- Mask sensitive columns in row lists and CSV exports; reveal only one
  sensitive cell at a time when a primary key row key is available.
- Keep the tool scoped to the current SForum PostgreSQL database.

## Next

- If write operations are requested later, design them as a separate feature
  with audit events and stricter confirmations.

## Open Questions

- Whether future production deployments should hide this module behind an
  additional environment switch beyond `database.manage`.
