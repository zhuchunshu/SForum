# Admin Language Settings And Runtime Locale Packs Design

## Goal

Add a language settings page under the admin system settings menu. Operators can upload multilingual ZIP packages at runtime, enable or disable them, and choose which languages are available to users without writing uploaded package content back into Git-tracked frontend locale files.

The first release enables uploaded packages for frontend UI messages only. The package format and database model reserve backend message files for a later release, but backend API envelope messages, emails, and notifications continue to use the built-in backend localization catalog for now.

## Context

SForum already has these relevant foundations:

- Built-in frontend catalogs under `apps/web/i18n/locales` for `zh-CN` and `en-US`.
- Backend locale normalization and localized API envelope messages under `apps/api/app/Support/Localization`.
- Runtime site options in `web_options`, including `site.default_locale` and `site.supported_locales`.
- A registry-driven admin shell under `apps/web/app/config/adminModules.ts`.
- Extension ZIP upload and runtime package storage patterns under `apps/api/app/Models/Extensions`.

The design keeps source-controlled built-in catalogs as the product baseline and treats uploaded language packs as runtime data.

## Library Survey

The existing stack is enough for this feature:

- `@nuxtjs/i18n` supports lazy-loaded translation files and runtime i18n behavior, but its route locale list is fundamentally build-time configuration.
- `vue-i18n` supports runtime message injection through APIs such as `mergeLocaleMessage` or equivalent composition APIs.
- Go's standard `archive/zip`, `encoding/json`, and filesystem packages are enough for package parsing and safe extraction, following the existing extension upload approach.

No new internationalization framework is needed. SForum should add package management, validation, state, permissions, and runtime API boundaries around the existing Nuxt/Vue i18n stack.

References:

- Nuxt i18n lazy loading: <https://i18n.nuxtjs.org/docs/guide/lazy-load-translations>
- Vue I18n composition API: <https://vue-i18n.intlify.dev/api/composition.html>

## Chosen Approach

Use a runtime language pack repository:

- API uploads and validates ZIP packages.
- Package files live under a runtime storage root, not inside `apps/web/i18n/locales`.
- PostgreSQL stores package metadata, versions, provided locales, status, and events.
- Public runtime APIs expose available locales and enabled frontend message overlays.
- Nuxt/Vue loads built-in catalogs normally, then merges enabled runtime messages when a user selects a runtime language or when a built-in locale is explicitly overridden.

This is preferred over storing whole translation JSON blobs only in the database because uploaded package files are easier to back up, inspect, export, and roll back. It also matches the existing extension system's package management shape.

## Storage And Git Isolation

Add `LOCALE_PACK_ROOT`, defaulting to `storage/locale-packs`.

Uploaded ZIP files and extracted message files are written only under:

```text
<LOCALE_PACK_ROOT>/<pack_id>/<version>/
  package.zip
  sforum.locale.json
  files/
    messages/<locale>.json
    backend/<locale>.json
```

Git isolation requirements:

- Add `storage/` or at minimum `storage/locale-packs/` to `.gitignore`.
- Add a `locale_packs` Compose volume.
- Mount the volume into API containers for upload, extraction, and message reads. The web container should fetch runtime messages through the public API and does not need direct filesystem access unless a later server-side file loader is introduced.
- Do not copy uploaded files into `apps/web/i18n/locales`.
- Treat language packs as persisted runtime data and include them in backup and restore documentation.

## Data Model

Add dedicated localization tables instead of adding large blobs to `web_options`.

### `locale_packs`

- `id TEXT PRIMARY KEY`
- `name TEXT NOT NULL`
- `status TEXT NOT NULL CHECK (status IN ('installed', 'enabled', 'disabled'))`
- `active_version_id BIGINT`
- `allow_builtin_override BOOLEAN NOT NULL DEFAULT false`
- `installed_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

### `locale_pack_versions`

- `id BIGSERIAL PRIMARY KEY`
- `pack_id TEXT NOT NULL REFERENCES locale_packs(id) ON DELETE CASCADE`
- `version TEXT NOT NULL`
- `manifest JSONB NOT NULL`
- `package_path TEXT NOT NULL`
- `messages_path TEXT NOT NULL`
- `installed_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- `UNIQUE (pack_id, version)`

### `locale_pack_locales`

- `id BIGSERIAL PRIMARY KEY`
- `version_id BIGINT NOT NULL REFERENCES locale_pack_versions(id) ON DELETE CASCADE`
- `locale_code TEXT NOT NULL`
- `display_name TEXT NOT NULL`
- `frontend_path TEXT NOT NULL`
- `backend_path TEXT NOT NULL DEFAULT ''`
- `message_key_count INTEGER NOT NULL DEFAULT 0`
- `missing_key_count INTEGER NOT NULL DEFAULT 0`
- `overrides_builtin BOOLEAN NOT NULL DEFAULT false`
- `UNIQUE (version_id, locale_code)`

### `locale_pack_events`

- `id BIGSERIAL PRIMARY KEY`
- `pack_id TEXT NOT NULL REFERENCES locale_packs(id) ON DELETE CASCADE`
- `actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL`
- `action TEXT NOT NULL`
- `message TEXT NOT NULL DEFAULT ''`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`

`site.default_locale` and `site.supported_locales` stay in `web_options`, but their validation changes from "built-in only" to "built-in plus currently enabled runtime locales."

## Package Format

A language pack ZIP must contain `sforum.locale.json`.

Example:

```json
{
  "id": "japanese-community-locale",
  "name": "Japanese Community Locale",
  "version": "1.0.0",
  "sforumVersion": "^0.1.0",
  "locales": [
    {
      "code": "ja-JP",
      "name": "日本語",
      "frontend": "messages/ja-JP.json",
      "backend": "backend/ja-JP.json"
    }
  ],
  "allowBuiltinOverride": false
}
```

Expected ZIP layout:

```text
sforum.locale.json
messages/
  ja-JP.json
backend/
  ja-JP.json
```

The `backend` file is optional and reserved in the first release. If present, it must be valid JSON, but it is not used for backend API messages yet.

## Validation Rules

Archive validation:

- Reject empty archives and archives above the configured maximum size.
- Reject paths that are absolute, contain `../`, escape the extraction root, or represent unsafe metadata.
- Require exactly one root manifest named `sforum.locale.json`.
- Limit total uncompressed size to prevent zip bombs.

Manifest validation:

- `id` matches the extension-style pattern: `^[a-z0-9][a-z0-9._-]{1,80}$`.
- `name`, `version`, and `sforumVersion` are required.
- Each locale entry has a BCP 47-style code such as `ja-JP`, `ko-KR`, or `fr-FR`.
- Each `frontend` path is present, safe, and points to a JSON object.
- `backend` paths are optional, safe, and JSON object-shaped when present.
- Reinstalling the same `id` and `version` is rejected.

Message validation:

- Frontend message JSON supports nested objects.
- The service flattens nested keys into dot paths for coverage analysis.
- New runtime languages may miss keys; missing keys are counted and shown in the admin UI.
- Runtime fallback is `zh-CN`.
- Built-in locale override requires both `manifest.allowBuiltinOverride = true` and an explicit admin enable action with `allowBuiltinOverride = true`.
- At most one enabled runtime pack version may provide a given non-built-in locale. Enabling a pack that conflicts with another enabled pack returns a validation error until the conflicting pack is disabled or replaced.

## Permissions And Policies

Add `locale.manage`.

- Seed it into `permissions`.
- Grant it to `super_admin` by default.
- Use it for all admin language pack and language settings endpoints.
- Keep API policy checks authoritative. Admin navigation visibility is only a user-experience helper.

This permission is intentionally separate from `settings.manage` because language packs can change every user-facing label on the site and may override trusted built-in strings.

## API Design

Public APIs:

- `GET /api/v1/locales`
  - Returns default locale, enabled locales, route locales, runtime locales, and source metadata.
- `GET /api/v1/locale-messages/:locale`
  - Returns enabled runtime frontend messages for one locale plus `version` or `etag` metadata.

Admin APIs:

- `GET /api/v1/admin/locale-packs`
- `POST /api/v1/admin/locale-packs`
- `POST /api/v1/admin/locale-packs/:id/versions/:version/enable`
- `POST /api/v1/admin/locale-packs/:id/disable`
- `GET /api/v1/admin/locale-packs/:id/events`
- `PUT /api/v1/admin/locales/settings`

Expected error reasons:

- `locale_pack.archive_invalid`
- `locale_pack.manifest_invalid`
- `locale_pack.messages_invalid`
- `locale_pack.not_found`
- `locale_pack.override_denied`
- `locale_pack.locale_conflict`

OpenAPI changes should live in:

- `contracts/openapi/paths/localization.yaml`
- `contracts/openapi/schemas/localization.yaml`

The root `contracts/openapi.yaml` remains an index.

## Admin UI

Add a new admin page:

- Page id: `/locales`
- Label key: `admin.nav.locales`
- Icon: `i-lucide-languages`
- Component name: `AdminLocales`
- Required permission: `locale.manage`
- Sidebar location: "System" folder, next to Settings, SEO, and Extensions.

Page sections:

- Recommended defaults: show `zh-CN` default and built-in `zh-CN`/`en-US` as a safe starting point, with one-click restore.
- Enabled languages: manage default locale and enabled locale list.
- Language packs: upload ZIP, inspect package coverage, enable, disable, and view events.
- Built-in overrides: a guarded switch for allowing uploaded packages to override `zh-CN` or `en-US` messages.

Move default language and supported language controls from the basic settings tab to this page. The basic settings tab should keep general site identity settings such as site name and site URL.

## Frontend Runtime Behavior

Built-in route locales stay build-time:

- `zh-CN` remains the default unprefixed route locale.
- English remains `/en/*`.

Uploaded runtime languages do not create new localized routes in the first release. For example, uploading `ja-JP` enables Japanese UI strings but does not add `/ja/*` SEO routes.

Runtime loading flow:

1. App startup loads public `/locales`.
2. Built-in locales continue to use Nuxt i18n catalogs.
3. When a user selects a runtime locale, the frontend fetches `/locale-messages/:locale`.
4. The frontend injects messages with Vue I18n runtime APIs.
5. The selected runtime locale is stored in cookie and, later, user profile preference.
6. If runtime message loading fails, keep the current built-in language and show a non-blocking failure state where appropriate.

Navigation behavior:

- Built-in English uses `switchLocalePath('en')`.
- Runtime languages switch UI locale without changing the route prefix.
- The language picker should display both route languages and runtime languages without exposing implementation details in normal UI copy.

## Backend Message Boundary

The first release does not let uploaded packages change backend API envelope messages, email templates, notification templates, or seed/admin labels.

The package format reserves backend files so a later release can add:

- backend message catalog loading,
- per-package backend message validation,
- cache invalidation across API and worker processes,
- policy for backend built-in message overrides.

Until then, backend messages continue to come from `apps/api/app/Support/Localization/messages.go`.

## Error Handling

- Upload and enable operations are transactional at the metadata level.
- A failed enable attempt does not disable or change the currently active language pack version.
- Enable failures are recorded in `locale_pack_events`.
- Public message reads return 503 when an enabled package references missing or unreadable files.
- Frontend runtime loading failures do not blank the page.
- Disabling a package removes its runtime locales from `site.supported_locales`. If the default locale is removed, it falls back to `zh-CN`.

## Tests

Backend service tests:

- Safe ZIP path handling.
- Manifest validation.
- Locale code validation.
- Duplicate version rejection.
- Message JSON validation and key flattening.
- Built-in override double-confirm behavior.
- Enable and disable state transitions.

Controller tests:

- Unauthenticated admin requests return 401.
- Authenticated users without `locale.manage` return 403.
- Upload success and upload failure paths.
- Enable, disable, and event list paths.
- Public `/locales` and `/locale-messages/:locale` output.

Options tests:

- `site.supported_locales` accepts built-in plus enabled runtime locales.
- Unknown locales are rejected.
- Disabling a package coerces supported/default locale values safely.

Frontend tests:

- Runtime locale list loading.
- Runtime message loading and caching.
- Language picker handling for route vs runtime locales.
- API failure fallback.
- Built-in override merge behavior.

Contract and integration:

- Run `ruby scripts/validate-openapi-refs.rb` after OpenAPI edits.
- Run `./scripts/test.sh` for full feature verification.

## Operations

- Production deployments must persist `locale_packs` alongside the database.
- API instances need shared access to enabled package files. Web instances should use the public message API and do not need the package volume unless a future direct file-loading path is added.
- Backup and restore docs must include database plus `locale_packs` volume.
- Operators should be warned that deleting runtime package files outside the app can make enabled languages temporarily unavailable.

## Out Of Scope

- SEO route generation for uploaded locales.
- Rebuilding Nuxt when a language pack is uploaded.
- Backend message overrides.
- Translating user-generated content.
- Machine translation or automatic language pack generation.
