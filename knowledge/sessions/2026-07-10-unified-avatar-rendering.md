# 2026-07-10 Unified Avatar Rendering Handoff

## Changed

- Added a neutral `Support/Avatar` view builder and reused it from Profile,
  Identity, and Forum without introducing model import cycles.
- `identity.CurrentUser` and `forum.UserSummary` now expose `avatar` as the
  shared `AvatarView` JSON shape.
- Forum topic/comment SQL loads profile avatar source data for authors and
  parent reply authors, then decorates summaries with the configured avatar
  fallback.
- Nuxt auth and forum types now include `AvatarView`; navbar, admin shell,
  homepage, category/tag lists, topic author blocks, and comments pass the view
  into `SFAvatar`.
- OpenAPI documents `CurrentUser.avatar` and `ForumUser.avatar`.

## Decisions

- Keep avatar fallback semantics in the neutral support package so packages
  that already depend on each other do not import Profile only for rendering
  helpers.
- Treat `SFAvatar :avatar="..."` as the required first-party user-avatar path.
  Name-only `SFAvatar` remains acceptable for demos and generic fallbacks.

## Verification

- `go test ./app/Models/Profile ./app/Support/Avatar ./app/Models/Identity ./app/Models/Forum ./app/Http/Controllers/Identity ./app/Http/Controllers/Forum`
- `bun test tests/unifiedAvatarRendering.test.ts`
- `ruby scripts/validate-openapi-refs.rb`

## Next

- Run broader frontend typecheck/build once the active worktree's unrelated
  in-progress UI/auth changes are ready for a full gate.
