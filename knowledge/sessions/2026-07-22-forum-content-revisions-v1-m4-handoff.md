# 2026-07-22 Forum Content Revisions V1 M4 Handoff

## Changed

- Added canonical topic/comment revision restore routes. They require the
  matching history-view permission plus `*_edit_any`, mandatory reason, and
  `expectedRevision`; the service re-enters the M3 update pipeline.
- Restore locks and rechecks the selected source revision, appends a new
  `restore` row with `restored_from_revision_id`, and preserves prior ledger
  rows. Legacy incomplete rows restore source only and retain current topic
  metadata and attachment bindings.
- Complete topic snapshots now revalidate current category/tag policy. Historical
  attachment IDs can only be rebound when each selected ID is active/public and
  still belongs to the original author; failures are atomic.
- Added super-admin-only non-current payload redaction. It clears source,
  attachment IDs, and topic snapshot metadata in the same transaction as a
  lightweight audit tombstone. Redacted rows cannot be read or restored.
- Updated OpenAPI and M4 PostgreSQL/service tests for restore, conflicts,
  attachment failure atomicity, audit, redaction, and hard-delete cascade.

## Decisions

- M4 deliberately adds no admin UI, diff UI, public history, or force-overwrite.
- Restore audits are emitted even when the restoring actor is also the author;
  metadata excludes raw source, attachment URLs, and free-form reasons.

## Verification

- `cd apps/api && GOCACHE=/private/tmp/sforum-gocache go test ./...`
- `ruby scripts/validate-openapi-refs.rb`
- `git diff --check`

## Next

- Start M5 only: admin content management UI and its permission-aware tests.

## Open Questions

- PostgreSQL integration tests require the configured local database to execute
  their fixture path; the package remains green when that optional test database
  is unavailable.
