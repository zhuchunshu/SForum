# 2026-07-22 Topic Reply Always Expanded

## Changed

- Content-page reply composer (`SFTopicReplyComposer`) is always expanded; no collapsed CTA / open toggle.
- Removed `replyComposerOpen` state and `:open` / `@open` wiring from `SFTopicShowPage`.
- Cancel clears draft + reply target but keeps the editor mounted.
- Topic action “回复主题” still scrolls to `#topic-reply-editor`.
- Updated `apps/web/tests/defaultThemeTopicPage.test.ts` contract.
- Compact editor footer buttons: SFButton icon/label slots, aligned cancel/submit styling
  with topic action buttons (public theme tokens + icon leading slot).

## Decisions

- Always-on editor is product intent for content pages; click-to-mount was only a navigation optimization and is reversed for UX.

## Next

- None for this request.

## Open Questions

- None.
