# 2026-07-30 Session Handoff

## Changed

- Added authenticated `PUT /api/v1/auth/locale` to persist `users.locale`.
  Empty input resolves to runtime `site.default_locale`; explicit input must be
  an enabled site locale.
- Added backend `identity.Service.UserLocale` and frontend `useUserLanguage`.
  The latter maps API BCP 47 values to Nuxt i18n keys, applies stored language
  after session restoration, and is reused by the navbar menu.
- Added the private default-language field to `/settings/profile` with Chinese
  and English copy and OpenAPI coverage.
- Added explicit imports for the nested `useUserLanguage` composable in the app
  startup and navbar language menu, avoiding Nuxt's non-recursive composable
  auto-import boundary and the resulting SSR 500.
- Made `useUserLanguage` import all project-owned composable dependencies
  explicitly, including the nested `useAuthSession` dependency.
- Added public runtime option `site.domain` to Site Settings. Both the admin
  form and Options service strip HTTP(S) protocols and trailing slashes before
  the normalized domain is saved and returned.
- Replaced the hard-coded `sforum.dev/u/` username prefix on
  `/settings/profile` with `{site.domain}/u/`; fresh defaults derive the domain
  from the trusted `site.url` host.
- Updated `/settings/security` login history to render by default instead of
  behind a collapsed toggle. It now requests the existing session API with
  `includeHistory=true&page&perPage=10` and shows `SFPagination` when needed.

## Decisions

- Reused the existing private `users.locale` field. No schema migration or
  public-profile API expansion is required.

## Next

- Manually verify changing the setting updates the active UI language and
  persists across a new session; verify the reset-to-empty API path uses the
  configured site default.
- Manually verify a protocol/trailing-slash domain entry is redisplayed in its
  normalized form and immediately updates the username prefix.
- Manually verify login history pagination on `/settings/security` with more
  than 10 session records.

## Open Questions

- None.
