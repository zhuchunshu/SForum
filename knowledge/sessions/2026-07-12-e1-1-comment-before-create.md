# 2026-07-12 Session Handoff — E1.1 comment.before_create

## Changed

- Catalog: `comment.before_create` (`filter`, fail_closed, 2000ms)
  - Payload: `actorUserId`, `topicId`, `parentId`, `content`
  - Patch allowlist: `content` only
- Forum `CreateComment`: invoke filter after permission + topic active checks,
  before content limits / render / store commit
- Reject → existing `appevents.RejectedError` → controller 422
- Tests: allow+patch, reject (no store), parentId in payload, locked topic
  skips filter
- Docs: regenerated `docs/extensions/catalogs/*`; authoring guide “which
  filter for which scenario” table
- Plan checklist E1.1 marked done

## Decisions

- Filter runs after topic-active check so locked/closed topics never pay
  plugin RPC cost
- Host re-validates patched content (length, links, mentions) after filter
- `parentId` is not patchable in v1 (tree rules stay host-owned)

## Next

1. **E1.2** `topic.before_update` (same pattern)
2. Or product fork: **E6.0** storage provider plugin slot after E1.1

## Open Questions

- Whether a fixture plugin should also declare `comment.before_create` for
  contract CI (optional; service tests already cover host wiring)
