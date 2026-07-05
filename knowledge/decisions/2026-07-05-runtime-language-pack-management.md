# 2026-07-05 Runtime Language Pack Management

## Status

Accepted.

## Context

SForum already supports built-in `zh-CN` and `en-US` locale catalogs and stores default/enabled locale choices as runtime options. Operators now need an admin language settings page where they can upload language packs without changing Git-tracked source files.

Uploaded translations can change a large part of the user-facing interface, and future backend messages may use the same package format. The design needs clear storage, permission, validation, and rollback boundaries.

## Decision

Use runtime ZIP language packs managed by the API.

- Store package files under `LOCALE_PACK_ROOT`, defaulting to `storage/locale-packs`.
- Keep uploaded files out of `apps/web/i18n/locales` and out of Git.
- Store package metadata, versions, provided locales, and events in dedicated `locale_pack*` tables.
- Add `locale.manage` for language pack and language settings administration.
- Add a system-menu admin page at `/locales`.
- Keep built-in `zh-CN` and `en-US` catalogs as the source-controlled product baseline.
- First release enables uploaded packages only for frontend runtime UI messages. Backend message files are reserved in the package format but not applied yet.
- Runtime languages do not create new SEO route prefixes in the first release. Built-in route locales remain `zh-CN` and `en-US`.

Built-in locale override is allowed only when both the manifest and the admin enable action explicitly permit it.

## Consequences

- Language packs become persisted runtime data and must be included in backups.
- `site.supported_locales` validation must expand from built-in-only to built-in plus enabled runtime locales.
- Public frontend runtime APIs are needed for locale metadata and runtime message loading.
- Multi-instance API deployments need shared access to `LOCALE_PACK_ROOT` until a future storage backend moves message delivery elsewhere. Web instances should fetch runtime messages through the public API.
- Later backend localization work can reuse the package metadata and reserved backend message file path.
