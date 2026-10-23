# Profile Module

Member public profiles and current-user profile settings.

## Backend

- `apps/api/app/Models/Profile` package: types, store interface, service,
  postgres store.
- `Service.SearchPublicUsers` / `Store.SearchPublicUsers` is the bounded public
  member lookup (default 8, max 20, query max 80 runes) used by the AI reply
  bot's `user-search` tool. It requires a non-empty query (an empty one would
  become a `%%` LIKE pattern and list every member), escapes LIKE
  metacharacters, matches `username_lower` and `lower(display_name)` on active
  accounts only, and returns `UserSearchHit{UserID, Username, DisplayName}`
  with no email/status/role field. It has no HTTP route of its own; the
  authenticated composer picker (`/api/v1/composer/references?kind=user`) stays
  the only browser-facing user lookup. Rationale:
  `decisions/2026-10-11-ai-reply-endpoint-identity-and-user-search.md`.
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
- Uploaded avatar views use the stable browser media URL
  `/media/avatars/{publicId}`. The Nuxt Host proxies that URL to the
  versioned attachment content API, keeping API versioning out of rendered
  profile and forum image URLs.
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
- `PUT /api/v1/auth/locale` updates the logged-in user's private default
  language. Empty input restores the runtime `site.default_locale`; other
  values must be among `site.supported_locales`.

Profile update is login-required and current-user only; no admin profile
editor in V1.

### Nickname（昵称）

- 昵称是展示标签，不是身份：`users.display_name` **不唯一**，唯一性只属于
  `username_lower`。决策见 `decisions/2026-10-20-nickname-policy.md`。
- `PUT /api/v1/profile` 接受 `displayName`（空串 = 回退用户名），成功时返回完整
  `PublicProfile`（含 `nickname` 状态），与 `GET /profile` 同形状。
- `GET /api/v1/profile/nickname/availability?value=` 只读预检，返回 `available`、
  `duplicates`（重名只提示）与 `conflict`（i18n key）。路由
  `core.route.profile.nickname_availability` 走 `core.guard.profile.self`。
- 策略来自 `profile.nickname.*` 公开选项（开关、冷却分钟、最小/最大长度、
  曾用名是否公开）；保留词复用 `identity.username.reserved`；昵称不得与另一位成员的
  `username` 完全同名（大小写不敏感）。
- 自助改名在 Identity 的 `NicknameService` 里完成：校验 → 冷却（事务内行锁 + 数据库
  时钟）→ 写入 `users.display_name` / `nickname_changed_at` → `user_nickname_changes`
  历史 → `audit_events(action=identity.nickname.change)`。管理员改名走既有 admin 事务，
  只写历史（`source=admin`），不写 `nickname_changed_at`，因此不占用成员的冷却。
- 改名成功后由 `PostgresSiteEngine.SyncAuthorDisplayName` 回写
  `search_documents.author_display_name` 与 `metadata_tsv`；失败只记日志，不回滚改名。
- 公开主页的 `previousDisplayNames` 只在 `profile.nickname.history_public` 开启时出现，
  最多 1 条；后台用户详情始终可看最近 20 条（`AdminUserDetail.nicknameHistory`）。

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
  auto-dismisses after 5s; errors do not).
- 昵称输入由 `components/settings/SFProfileNicknameField.vue` 拥有：本地格式/长度
  预检、400ms 防抖的 `nickname/availability` 预检、分钟级冷却倒计时（30s 刷新）、
  重名提示与字段级错误。保存仍由资料页统一提交；成功后页面更新
  `profile.value.displayName` 与 auth session 的 `displayName`，导航栏即时同步。
  组件只在昵称真的变化时才提交 `displayName`。
- `/settings/profile` uses the
  `forum.settings.profile` Page Registry surface with a canvas layout: desktop
  left account rail, center editable profile form, right public preview, and
  shared left/right mobile drawers on narrow screens.
- The three-column settings chrome (left `SFHomeNavigation` +
  `SFSettingsAccountNav`, page head with drawer toggles, right rail, mobile
  drawers, `SFContentColumnFooter`, category-group fetch) is owned by the
  shared `SFSettingsShell` component; profile, personal appearance, login
  methods, local password, security, personal access tokens, and notification
  settings pages only fill its `default` / `#rail` / `#head-actions` slots. The
  rail slot renders once and is reused by both the desktop aside and the mobile
  right drawer. New account settings pages should add a link in
  `SFSettingsAccountNav` and wrap content in `SFSettingsShell`.
- The profile settings page keeps all writes on the existing `useProfileApi`
  contract: `GET /profile`, `PUT /profile`, `POST /profile/avatar`, and
  `DELETE /profile/avatar`. Avatar upload UI requires both public runtime
  option `avatar.allow_upload` and `attachment.upload`; API policy remains
  authoritative.
- Profile form editing is draft-based. Save is disabled until local edits are
  dirty, reset discards unsaved changes with a neutral Toast, save/avatar
  success Toasts auto-dismiss after 5 seconds, field-level validation stays
  beside the relevant inputs, and blocking errors remain visible.
- The profile identity section includes a private default-language selector.
  `useUserLanguage` maps API BCP 47 values (such as `en-US`) to Nuxt locale
  keys (such as `en`), persists a signed-in choice, and applies it to i18n.
- The read-only username field displays its public path prefix from the runtime
  `site.domain` option (`{domain}/u/`) instead of a hard-coded product domain.
- Navbar user menu links to public profile and profile settings.
- `useProfileApi` composable + `AvatarView`/`ProfileData`/`PublicProfile`
  types.
- `AvatarView` is also used by auth current-user state and forum author
  summaries; first-party UI should pass it into `SFAvatar`.
- `SFAvatar` emits remote `AvatarView` URLs (including Gravatar) directly in
  SSR HTML. It falls back to initials only after the image reports a real load
  error; do not reintroduce client-only preloading that swaps avatars after the
  first paint.
- Uploaded avatars are already normalized by the backend and render through a
  plain `<img>` at `/media/avatars/{publicId}`. Do not send these API-backed
  relative URLs through Nuxt IPX: IPX treats them as local filesystem assets
  and returns `IPX_FILE_NOT_FOUND`.
