# 2026-07-12 Session Handoff — E2.2 forum.comment.actions

## Changed

- Catalog: `forum.comment.actions` (descriptor, payload `extensionRoute`)
  - Optional `requiresAuth` on payload (UX hint; proxy still authoritative)
  - Same method allowlist as topic actions: POST/PUT/PATCH/DELETE
- `CommentList.extensionActions` list-level decoration (not per-comment rows)
- Provider: `ExtensionCommentActionProvider`
- Bootstrap wires comment actions with other public contributions
- OpenAPI: `CommentExtensionAction` + CommentList field
- Web: `ForumCommentExtensionAction`, `forumCommentExtensionActionRequest`,
  `applyCommentExtensionAction`
- Default theme: comment row menus append extension actions; guests hide
  `requiresAuth`; body `{ topicId, commentId }`
- Tests: manifest, provider, service ListComments, frontend taxonomy + theme
- Docs: catalog regenerate + authoring guide E2.2 section

## Decisions

- List-level descriptors avoid per-row plugin RPC and payload bloat
- `requiresAuth` is not a security boundary — only theme visibility
- Reuse topic action payload type (+ requiresAuth) rather than a parallel schema

## Next

1. **E2.3** `forum.nav.items`
2. **E2.4** `forum.topic.list.badges`
3. Product fork: **E6.0** storage provider decision + host interface

## Open Questions

- Whether topic actions should also surface `requiresAuth` for parity
- Whether comment create response should include the list-level actions array
