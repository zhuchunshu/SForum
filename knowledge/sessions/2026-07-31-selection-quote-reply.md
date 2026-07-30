# 2026-07-31 Selection Quote Reply

## Changed

- Topic and comment rich content now exposes native text-selection quote targets.
- Selecting eligible text shows a compact `引用并回复` action anchored in the
  topic scroll container's document coordinates.
- Topic selections open the existing advanced topic reply; comment selections
  open the existing direct-comment reply with a safe Markdown blockquote draft.
- Quote normalization caps selections at 500 Unicode characters, preserves
  paragraph breaks, and escapes HTML-significant text before editor parsing.

## Decisions

- Reuse browser `Selection`/`Range`; no selection dependency was added.
- Reuse `SFTopicCommentComposerDrawer` and current comment submission paths.
- Reply permission, locked-topic policy, moderation, cooldown, CAS, and
  transactional notification fanout remain server-authoritative and unchanged.
- The action is `position: absolute` inside the topic scroll owner, so it moves
  away with the selected content instead of following the viewport.

## Verification

- `bun test tests/forum/selectionQuoteReply.test.ts tests/forum/defaultThemeTopicPage.test.ts`
- `bun run typecheck`
- Desktop Chrome on `/t/87`: real comment selection showed `引用并回复`;
  scrolling the content column from 0 to 360 moved the toolbar top from 530.7
  to 170.7; clicking opened the existing drawer with a visible Tiptap
  blockquote containing `这次一定`; no relevant console warnings/errors.
- Architecture validation passes with the topic route kept at its existing
  1249-line legacy cap.

## Next

- Repeat rendered selection QA at `390x844` when the connected browser backend
  can enforce its advertised viewport override.
- A two-actor publish/recipient-notification smoke test is optional; notification
  creation itself was not changed and remains covered by Notification V2.

## Open Questions

- None for the implementation contract.
