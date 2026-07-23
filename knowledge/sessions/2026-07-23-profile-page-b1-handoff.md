# 2026-07-23 Session Handoff

## Changed

- Reworked the default-theme public profile page `/u/{username}` to the B1
  daily timeline design while keeping the homepage/topic three-column shell and
  shared forum navigation.
- Public profile API now returns authoritative public activity rows in addition
  to real public counts and recent public topics.
- Added locale-stable frontend activity grouping/link helpers plus focused Go
  and Bun tests for mapping, grouping, empty/error states, and self-only edit UI.

## Decisions

- Profile timeline links are plain anchors rather than `NuxtLink`; Browser
  verification found one client navigation path could update the URL while
  leaving the profile shell stale, while full navigation reliably loads the
  target topic/comment page.
- The page intentionally does not invent follows, followers, likes, levels,
  portfolios, private messages, or client-computed stats.

## Next

- Watch for conflicts with any parallel default-theme work touching public
  layout/CSS, `SFHomeNavigation`, Page Registry defaults, or knowledge files.
- Repo-wide `apps/web bun test` still has unrelated existing dependency/alias
  failures; profile-targeted tests and typecheck pass.

## Open Questions

- None for the B1 profile implementation.
