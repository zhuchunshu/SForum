# 2026-07-10 Unified Avatar Rendering Design

## Context

SForum already has a dedicated avatar mechanism: profile responses expose
`AvatarView`, admin settings control the fallback provider, and `SFAvatar`
can render uploaded, Gravatar-compatible, static, or initials avatars.
However, several high-traffic surfaces still show unrelated avatar UI or only
initials because their user summaries do not include `AvatarView`.

## Problem

The current implementation has two separate gaps:

- Presentation gap: navbar/admin chrome still use hand-written or Nuxt UI
  avatar rendering instead of the shared `SFAvatar`.
- Data gap: `CurrentUser` and forum `ForumUser` summaries only contain
  id/username/display name, so feed rows, topic bylines, author cards, comments,
  and reply references cannot render uploaded or configured fallback avatars.

Changing only Vue templates would make the UI shape more consistent, but it
would not let uploaded avatars appear across the forum. The data contract must
carry the same avatar view that the profile pages already use.

## Design

Introduce one shared backend avatar view builder in the Profile model package.
Profile keeps owning avatar semantics because the data lives in
`user_profiles`, attachment references, and `avatar.*` runtime options.
Identity and Forum consume the exported builder rather than duplicating
fallback/hash/attachment URL logic.

Extend user summary contracts:

- `identity.CurrentUser` gains `avatar`.
- `forum.UserSummary` gains `avatar`.
- OpenAPI `CurrentUser` and `ForumUser` reference the existing
  `profile.yaml#/AvatarView`.
- Frontend `CurrentUser` and `ForumUserSummary` types mirror those fields.

Backend stores should populate avatar data in their normal summary queries:

- Identity current-user/session/login responses should load the current user's
  avatar with a single left join against `user_profiles` and `attachments`.
- Forum topic and comment queries should include author and parent-author
  avatar columns while they already join `users`.
- Avatar defaults should still fall back to initials if option resolution or
  attachment lookup is unavailable, so public reads do not fail because avatar
  settings are temporarily unavailable.

Frontend rendering should use `SFAvatar` everywhere an SForum user avatar is
shown:

- Public navbar user button.
- Admin shell footer/dropdown user button.
- Homepage topic rows and participant chip.
- Topic detail byline and author summary card.
- Comment list and nested comments, including reply references where a compact
  avatar is shown later.

## Testing

Use test-first coverage for contract behavior:

- Backend identity tests assert `CurrentUser.avatar` exists and prefers uploaded
  attachment URLs when present.
- Backend forum tests assert topic/comment authors include `avatar`.
- Frontend tests assert navbar/admin/comment/feed surfaces pass `avatar` to
  `SFAvatar` and no longer use `UAvatar` or hand-written navbar avatar spans.
- OpenAPI reference validation runs after schema changes.

## Non-Goals

- No new avatar provider.
- No admin UI redesign.
- No extra per-row profile API request from the frontend.
- No changes to upload rules, image compression, or attachment authorization.
