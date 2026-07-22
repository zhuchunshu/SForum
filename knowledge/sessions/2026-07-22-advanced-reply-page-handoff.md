# 2026-07-22 Advanced Reply Page

## Changed

- Comment composer header gains top-right text link **高级回复** / Advanced reply.
- New Page Registry page `forum.topic.reply` at `/topics/reply?topic=&parent=`.
- Host island `SFTopicReplyPage` renders full (non-compact) `SFEditor`.
- Draft handoff via `sessionStorage` key `sforum.advanced-reply.draft.{topicId}`.
- Default + nocturne theme shells: `templates/topic-reply.html`.
- API catalog / viewmodel / island bindings updated for the new page.

## Decisions

- Advanced reply is a login-required replaceable page (same family as topic create).
- Compact composer stays on the topic page; advanced mode is navigation, not a modal.

## Next

- None for this request.

## Open Questions

- None.
