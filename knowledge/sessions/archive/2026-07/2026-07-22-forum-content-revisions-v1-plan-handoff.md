# 2026-07-22 Forum Content Revisions V1 Plan Handoff

## Changed

- Added ready task book:
  `knowledge/plans/2026-07-22-forum-content-revisions-v1.md`.
- Scoped V1 to staff editing, immutable self/staff revisions, CAS conflict
  prevention, admin history/diff, safe restore, and super-admin payload redaction.
- Reused existing edit-any permissions and added planned history-view permissions.

## Decisions

- Evolve `post_revisions`; keep `posts` as the hot current read model.
- Full accepted source snapshots, no canonical delta chains or derived HTML in
  history.
- Restore creates a new revision and reruns current Host policy.
- History is admin-only in V1; collaboration and notifications are deferred.
- Large existing datasets require additive schema plus batched resumable backfill.

## Next

- Start M0 only: re-audit current code, write the ADR and contract-test matrix,
  verify the diff dependency, and keep any executable checkpoint green before
  schema work.

## Open Questions

- None at product-scope level. Implementation may adjust names to repository
  conventions without weakening the frozen semantics in the task book.
