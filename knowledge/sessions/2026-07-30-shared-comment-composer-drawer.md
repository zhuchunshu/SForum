# 2026-07-30 Shared Comment Composer Drawer

## Changed

- Kept top-level quick reply in the page with the compact editor and direct
  submission; focusing it no longer opens the advanced composer.
- Advanced reply, comment reply, and comment edit use one
  `SFTopicCommentComposerDrawer`; the standalone advanced-reply page remains a
  compatibility redirect.
- The shared drawer now opens from the bottom on desktop and mobile. Its top
  handle supports pointer dragging plus Arrow/Home/End keyboard height changes.
- Added `useTopicCommentComposerDrawer` for mode, draft, legacy-query,
  submission, revision-CAS, cross-author reason, and close/discard state.
- Reduced `SFTopicShowPage.vue` from 1475 to 1273 lines and lowered its
  architecture ratchet from 1363 to 1273.
- Kept `/topics/reply?topic=&parent=` as a compatibility redirect that opens
  the shared drawer and then removes the one-shot query parameters.
- Delayed comment-anchor flash state until mounted so legacy reply redirects
  hydrate without a server/client class mismatch.

## Decisions

- Desktop defaults to a bottom drawer around 76dvh; mobile defaults around
  90dvh. Both clamp manual height changes to the current viewport. All advanced,
  reply, and edit modes reuse the full `SFEditor` and drawer-owned footer actions.
- Existing API authorization, cooldown, moderation, expected-revision CAS, and
  cross-author audit-reason behavior remain authoritative.

## Verification

- `bun test tests/forum/defaultThemeTopicPage.test.ts tests/forum/forumCooldownFeedback.test.ts`:
  10 passed.
- `bun run build`: passed through client, SSR, and Nitro output.
- `bun run typecheck`: this change adds no type errors; the worktree still has
  unrelated pre-existing errors in attachment settings, personalization,
  language, search SEO, and admin-surface typing.
- Browser QA on `/t/87/topic-1oa3k5k` passed inline focus without drawer open,
  bottom-drawer identity, pointer resizing, keyboard resizing, console checks,
  and no horizontal overflow at `1280x720` and `390x844`. Desktop height moved
  from about 547px to 627px; mobile moved from about 760px to 660px.
- Architecture validation reports only unrelated concurrent API baselines.

## Next

- No feature work remains. A real create/update submission was not performed
  during Browser QA to avoid mutating the user's development data.

## Open Questions

- None.
