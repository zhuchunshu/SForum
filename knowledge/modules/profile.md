# Profile Module

Member public profiles and current-user profile settings.

## Backend

- `apps/api/app/Models/Profile` package: types, store interface, service,
  postgres store.
- `user_profiles` table (migration `202607070004_user_profiles.sql`):
  `user_id` PK referencing `users(id)` cascade delete; `bio`, `signature`,
  `location`, `website_url` text; `avatar_attachment_id` nullable reference to
  `attachments(id)` set null on delete; `created_at`/`updated_at`.
- Sparse by design: no background images, custom code, birthday, phone,
  follow counts, or gamification fields.
- Profile responses include an `avatar` view object with `kind`, `url`,
  `attachmentId`, and `alt`. Uploaded avatars win; otherwise the service falls
  back to the configured default provider (`initials`, `gravatar`, or
  `static`). The older `avatarAttachmentId` field remains for compatibility,
  but avatar changes should use the dedicated avatar endpoints.
- Avatar view construction is shared through the neutral
  `apps/api/app/Support/Avatar` builder so Identity and Forum summaries reuse
  the same fallback behavior without importing the Profile model package.

### Endpoints

- `GET /api/v1/profiles/{username}` public profile: user summary + profile +
  public-visible topic/comment counts, 5 recent public topics, and a bounded
  public activity timeline. Counts and timeline rows only include active
  public topics and active comments in public active/locked topics.
- `GET /api/v1/profiles/{username}/activities?kind=topic|comment&page&perPage`
  paginated public activities. Comment items include `commentId` and
  `commentPage` (flat-view page, same formula as
  `GET /topics/{id}/comments/{commentId}/page`) so clients can build
  `/t/.../page/N#comment-{id}` deep links.
- `GET /api/v1/profile` current user's editable profile (login required).
- `PUT /api/v1/profile` update current user's profile (login required,
  current-user only). Validates length limits and website URL shape
  (`http://`/`https://`). It still accepts `avatarAttachmentId` for
  compatibility, but new avatar changes should use the dedicated endpoints.
- `POST /api/v1/profile/avatar` uploads and sets the current user's avatar.
  The actor must be logged in, active, have `attachment.upload`, and the runtime
  option `avatar.allow_upload` must be enabled.
- `DELETE /api/v1/profile/avatar` removes the uploaded avatar and returns the
  profile decorated with the configured fallback avatar.

Profile update is login-required and current-user only; no admin profile
editor in V1.

## Frontend

- `/u/{username}` public profile page uses the default theme three-column
  shell. The center column renders a compact member summary and locale-aware
  daily activity groups for real public topics and replies; the right rail
  renders real public profile fields, public topic/reply stats, and recent
  public topics. It does not show unimplemented social counts, levels, follows,
  private messages, portfolios, or placeholder stats.
- `SFProfileRightRail` owns the shared desktop/mobile rendering of public
  details, real statistics, recent topics, and extension links. On mobile the
  left navigation and this right rail use the default-theme shared drawers.
- Public activity links use existing legal forum routes:
  `/t/{topicId}/{slug}` for topics and
  `/t/{topicId}/{slug}/page/N#comment-{id}` for replies (`/page/N` omitted when
  N=1). Page is pre-baked from API `commentPage` because URL fragments are not
  sent to the server; hash-only links always SSR page 1.
- The edit-profile entry on `/u/{username}` is self-only UI backed by the
  existing login-required current-user profile API; hiding the button is not a
  permission boundary.
- `/settings/profile` current-user settings page with avatar preview,
  upload/remove controls, safe defaults, and a clear save flow (success
  auto-dismisses after 10s; errors do not).
- `/settings/profile` uses the
  `forum.settings.profile` Page Registry surface with a canvas layout: desktop
  left account rail, center editable profile form, right public preview, and
  shared left/right mobile drawers on narrow screens.
- The profile settings page keeps all writes on the existing `useProfileApi`
  contract: `GET /profile`, `PUT /profile`, `POST /profile/avatar`, and
  `DELETE /profile/avatar`. Avatar upload UI requires both public runtime
  option `avatar.allow_upload` and `attachment.upload`; API policy remains
  authoritative.
- Profile form editing is draft-based. Save is disabled until local edits are
  dirty, reset discards unsaved changes with a neutral Toast, save/avatar
  success Toasts auto-dismiss after 10 seconds, field-level validation stays
  beside the relevant inputs, and blocking errors remain visible.
- Navbar user menu links to public profile and profile settings.
- `useProfileApi` composable + `AvatarView`/`ProfileData`/`PublicProfile`
  types.
- `AvatarView` is also used by auth current-user state and forum author
  summaries; first-party UI should pass it into `SFAvatar`.
- `SFAvatar` emits remote `AvatarView` URLs (including Gravatar) directly in
  SSR HTML. It falls back to initials only after the image reports a real load
  error; do not reintroduce client-only preloading that swaps avatars after the
  first paint.
