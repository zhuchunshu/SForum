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
  topic/comment counts + 5 recent public topics.
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

- `/u/{username}` public profile page with summary and recent topics.
- `/settings/profile` current-user settings page with avatar preview,
  upload/remove controls, safe defaults, and a clear save flow (success
  auto-dismisses after 10s; errors do not).
- Navbar user menu links to public profile and profile settings.
- `useProfileApi` composable + `AvatarView`/`ProfileData`/`PublicProfile`
  types.
- `AvatarView` is also used by auth current-user state and forum author
  summaries; first-party UI should pass it into `SFAvatar`.
- `SFAvatar` emits remote `AvatarView` URLs (including Gravatar) directly in
  SSR HTML. It falls back to initials only after the image reports a real load
  error; do not reintroduce client-only preloading that swaps avatars after the
  first paint.
