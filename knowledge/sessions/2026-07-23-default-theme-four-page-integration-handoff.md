# 2026-07-23 Default Theme Four-Page Integration Handoff

## Changed

- Merged the completed `/categories`, `/notifications`, `/u/{username}`, and
  topic-create worktrees into `main` as four separate feature commits plus
  merge commits.
- Fixed notification navigation permission UX to use the canonical
  `FORUM_PERMISSIONS.topicCreate` check and the active appearance primary Toast.
- Fixed the notification mobile detail drawer so its API-backed unread summary,
  selected detail, and empty state remain visible below the desktop breakpoint.
- Split the public profile right rail into `SFProfileRightRail` and reused it in
  the new mobile information drawer; the shared mobile navigation drawer is now
  wired on the profile page as well.
- Removed stale and duplicated profile-page rules from the default theme L0
  stylesheet. Host `sforum-profile.css` is the single owner of Host island
  presentation defaults.
- Split topic-composer CSS out of the Vue SFC, scoped local drafts by account,
  made controlled tags fail closed, and made error Toasts persistent.

## Decisions

- The four original worktrees and branches remain intact for audit/recovery;
  integration did not delete or reset any other task worktree.
- Notification type filters remain honest client-side filters over loaded rows.
  No API or fabricated statistics were added.
- Public profile and notification mobile right rails reuse the same data and
  presentation components as desktop instead of maintaining parallel mock UI.

## Verification

- Integrated focused Bun suite: 35 tests passed before final QA fixes; the
  notification/profile subset passed again after the fixes.
- Nuxt typecheck passed after integration and after the final component split.
- Profile Go model/controller tests and modular OpenAPI reference validation
  passed.
- Browser QA passed at 1440x1000 and 390x844 for layout, overflow, category
  sorting/group focus/filter, notification empty/detail drawer, self profile
  edit entry/timeline/info drawer, composer editor/category summary/fixed dock,
  mobile drawers, light/dark tokens, and a clean console on fresh SSR keys.
- The current account had no notification rows, so single/all-read rendered
  mutations were covered by focused optimistic-update/rollback tests rather
  than destructive Browser setup.

## Next

- Resume the surviving tags, profile-settings, moderation, and system-error
  worktrees from their existing branches. Do not recreate clean worktrees or
  reset their uncommitted changes.

## Open Questions

- Nuxt dev SWR can retain pre-change anonymous `/categories` and `/u/**` HTML in
  the generated cache across restarts. Fresh cache keys produced clean SSR and
  zero hydration warnings; deployment starts with a clean runtime cache.
