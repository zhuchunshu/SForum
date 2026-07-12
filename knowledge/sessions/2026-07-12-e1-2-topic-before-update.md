# 2026-07-12 Session Handoff — E1.2 topic.before_update

## Changed

- Catalog: `topic.before_update` (`filter`, fail_closed, 2000ms)
  - Payload: `actorUserId`, `topicId`, plus only fields present in the request
    (`categorySlug`, `tagSlugs`, `title`, `content`)
  - Patch allowlist: `categorySlug`, `tagSlugs`, `title`, `content` (same as create)
- Forum `UpdateTopic`: invoke filter after edit permission + author edit window,
  before field validation / render / store commit
- Plugins may patch fields not in the request (e.g. force tags on title-only edit)
- Host re-validates patched values (title length, tags, content, links, etc.)
- Reject → `appevents.RejectedError` → controller 422
- Tests: patch title/tags, force tags, reject, unauthorized skips filter
- Docs: regenerated catalogs; authoring guide filter scenario table
- Plan checklist E1.2 marked done

## Decisions

- Edit window check stays **before** filter so expired author edits never pay
  plugin RPC
- Payload omits unset optional fields; patch can still introduce them
- `TagSlugs: nil` means “not updating tags” in the request; after a tag patch
  the host treats the slice as an explicit update

## Next

1. **E1.3** `user.before_register` (prefer validate-only reject in v1)
2. Or product fork: **E6** storage provider plugin slot

## Open Questions

- Whether partial updates should include current DB values in the filter payload
  for plugins that need “full snapshot” context (v1: request fields only)
