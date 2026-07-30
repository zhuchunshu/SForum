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
- Stabilized the drawer-owned full editor as a toolbar/canvas/status grid.
  Enlarging the drawer now grows the canvas instead of leaving blank space
  below the status row; shrinking it preserves the complete editor and moves
  overflow to the drawer body instead of clipping the status row.

## Decisions

- Desktop defaults to a bottom drawer around 76dvh; mobile defaults around
  90dvh. Both clamp manual height changes to the current viewport. All advanced,
  reply, and edit modes reuse the full `SFEditor` and drawer-owned footer actions.
- The drawer uses a six-row editor baseline aligned with a 180px desktop and
  190px mobile canvas minimum. This is a local host layout contract and does
  not change `SFEditor` sizing on topic, settings, or admin surfaces.
- Existing API authorization, cooldown, moderation, expected-revision CAS, and
  cross-author audit-reason behavior remain authoritative.

## Verification

- `bun test tests/forum/defaultThemeTopicPage.test.ts tests/forum/forumCooldownFeedback.test.ts`:
  10 passed.
- `bun run build`: passed through client, SSR, and Nitro output.
- `bun run typecheck`: passed.
- Browser QA on `/t/87/topic-1oa3k5k` passed inline focus without drawer open,
  bottom-drawer identity, pointer resizing, keyboard resizing, console checks,
  and no horizontal overflow at `1280x720` and `390x844`. Desktop height moved
  from about 547px to 627px; mobile moved from about 760px to 660px.
- Follow-up Browser QA on `/t/65` passed default, near-full-height, minimum-height
  scrolling, and `390x844` mobile states. The status row remained intact, extra
  height entered only the canvas, console logs were clean, and horizontal
  overflow remained absent.
- Architecture validation passed.

## Next

- No feature work remains. A real create/update submission was not performed
  during Browser QA to avoid mutating the user's development data.

## Open Questions

- None.
