# 2026-07-22 Forum Content Revisions V1 M3 Handoff

## Changed

- PATCH topic/comment now require `expectedRevision`; service-level early CAS
  prevents stale requests from reaching synchronous update filters, and the
  PostgreSQL transaction locks the resource and shared post before rechecking.
- Effective topic/comment edits append exactly one complete accepted revision
  with final normalized content, metadata, topic snapshot/attachment IDs, and
  increment `posts.current_revision` once. The old superseded-before-overwrite
  helper is no longer used by canonical edits.
- Locked final-state comparison returns no-op without touching timestamps,
  cache/search invalidation, audit, moderation, or the revision ledger.
- Cross-author edits require a trimmed reason of at most 500 Unicode runes;
  successful cross-author writes append generic audit through the new shared
  transaction-aware audit writer entry point.
- Added `comment.before_update` and `comment.updated`; `topic.updated` now
  carries safe revision metadata only. Public topic/comment editors submit the
  loaded revision token. OpenAPI and generated extension event/hook docs match.

## Decisions

- M3 does not add restore, historical attachment restore, redaction, admin UI,
  diff UI, or force-overwrite behavior. Those remain M4-M7 work.
- A backfill-complete ledger remains the operational prerequisite for legacy
  `current_revision=0` rows; run `sforum revisions backfill --loop` before
  relying on edits for old content.

## Verification

- `GOCACHE=/private/tmp/sforum-gocache go test ./...`
- `set -a; . ../../.env; set +a; GOCACHE=/private/tmp/sforum-gocache go test ./app/Models/Forum -run 'TestRevisionLedgerVersioned' -count=1`
- `cd apps/web && bun run typecheck`
- `ruby scripts/validate-openapi-refs.rb`
- `cd apps/api && GOCACHE=/private/tmp/sforum-gocache go run ./cmd/sforum extension docs generate --check`

## Next

- Start M4 only: restore through the current write pipeline, narrow historical
  attachment validation, and `super_admin` redaction with audit.

## Open Questions

- The independent V3 catalog check is still blocked by M2 admin content routes
  missing reviewed stable identity mappings in `scripts/v3-catalog`; do not
  hand-edit generated inventory or treat that as an M3 failure.
