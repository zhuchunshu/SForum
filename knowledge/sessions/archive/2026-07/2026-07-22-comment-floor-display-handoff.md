# 2026-07-22 Session Handoff

## Changed

- Topic detail comment floor badges now display list positions instead of database comment IDs.
- Comment anchors still use stable `#comment-<id>` targets, so deep links and side-card navigation continue to work.
- Reply-target UI hides the floor label when the target is not in the loaded comment page instead of falling back to a raw ID.

## Decisions

- Floor numbering is derived from `ForumCommentList.page` and `perPage`, matching the server-authoritative public pagination.
- `SFComment` only renders a floor badge when the parent surface passes an explicit `floorLabel`.

## Next

- No required follow-up for this bug.

## Open Questions

- None.
