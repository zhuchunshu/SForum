# 2026-07-30 Shared Comment Composer Drawer

## Changed

- Replaced the inline comment editor and standalone advanced-reply page with
  one `SFTopicCommentComposerDrawer` for advanced reply, comment reply, and
  comment edit.
- Added `useTopicCommentComposerDrawer` for mode, draft, legacy-query,
  submission, revision-CAS, cross-author reason, and close/discard state.
- Reduced `SFTopicShowPage.vue` from 1475 to 1273 lines and lowered its
  architecture ratchet from 1363 to 1273.
- Kept `/topics/reply?topic=&parent=` as a compatibility redirect that opens
  the shared drawer and then removes the one-shot query parameters.
- Delayed comment-anchor flash state until mounted so legacy reply redirects
  hydrate without a server/client class mismatch.

## Decisions

- Desktop uses a right drawer up to 640px; mobile uses a bottom drawer around
  90dvh. All modes reuse the full `SFEditor` and drawer-owned footer actions.
- Existing API authorization, cooldown, moderation, expected-revision CAS, and
  cross-author audit-reason behavior remain authoritative.

## Verification

- `bun test tests/forum/defaultThemeTopicPage.test.ts tests/forum/forumTopic.test.ts`:
  31 passed.
- `bun run typecheck`: passed.
- `bun run build`: passed through client, SSR, and Nitro output.
- Browser QA on `/t/87/topic-1oa3k5k` passed advanced reply, comment reply,
  edit, dirty-discard, legacy redirect, and console checks at `1440x1000` and
  `390x844`. Mobile measured 390x759.6 with no horizontal overflow.
- Architecture validation reports only unrelated concurrent API baselines.

## Next

- No feature work remains. A real create/update submission was not performed
  during Browser QA to avoid mutating the user's development data.

## Open Questions

- None.
