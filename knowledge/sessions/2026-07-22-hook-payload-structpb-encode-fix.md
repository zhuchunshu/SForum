# 2026-07-22 Session Handoff — extension.hook_failed on post/comment

## Changed

- Root cause: Protocol V2 `protocolV2Document` used `structpb.NewStruct` on
  raw Host payload maps. Forum write path puts `forum.ContentInput` and
  `[]string` into filter events; protobuf rejects those types →
  `extension.hook_failed` under fail_closed `topic.before_create` /
  `comment.before_create` (sforum.content-policy).
- Fix: JSON-normalize via `cloneHookDocument` before `structpb.NewStruct`
  in `apps/api/app/Support/Extensions/protocol_v2_client.go`.
- Content-policy `payloadString` now reads content objects
  (`plainText` / `rawContent`) so keyword scan still works after
  normalization.
- Tests: `protocol_v2_document_test.go`; content-policy `TestPayloadStringContentObject`.
- Rebuilt staging plugin under `storage/builtin-dev` (source tree digests
  unchanged by design).

## Decisions

- Normalize at Host encode boundary (not each Emit call site) so all V2
  hooks/providers share the same JSON-friendly document path.

## Follow-up (same day)

- After hook encode fix, Tiptap posts still 500: `posts_source_format_check`
  only allowed markdown/html/json while Host writes `editor-document`.
- Migration: `202607220048_posts_source_format_editor_document.sql` (applied).

## Next

- Manual smoke from the web UI: create topic + comment with the Tiptap editor.

## Open Questions

- None.
