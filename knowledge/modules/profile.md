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

### Endpoints

- `GET /api/v1/profiles/{username}` public profile: user summary + profile +
  topic/comment counts + 5 recent public topics.
- `GET /api/v1/profile` current user's editable profile (login required).
- `PUT /api/v1/profile` update current user's profile (login required,
  current-user only). Validates length limits and website URL shape
  (`http://`/`https://`). Avatar upload requires an existing attachment owned
  by the actor and writes an `attachment_references` row (future).

Profile update is login-required and current-user only; no admin profile
editor in V1.

## Frontend

- `/u/{username}` public profile page with summary and recent topics.
- `/settings/profile` current-user settings page with safe defaults and a
  clear save flow (success auto-dismisses after 10s; errors do not).
- Navbar user menu links to public profile and profile settings.
- `useProfileApi` composable + `ProfileData`/`PublicProfile` types.
