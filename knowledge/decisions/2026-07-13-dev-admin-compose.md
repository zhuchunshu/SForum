# Local lightweight admin/theme compose for `bun run dev` (P1)

Date: 2026-07-13

## Context

Trusted admin extension frontends and theme layers were only visible in local
dev after a full Web Release (copy snapshots, frozen install, Nuxt production
build, pointer). That made everyday edits to `ThemeSettingsPage.vue` / SMTP
admin / locales require 「扩展 → Web 发布 → 立即重建」, which is the right
gate for production but too heavy for builtin source work.

## Decision

1. **`bun run dev` defaults to lightweight compose** of
   `extensions/builtin/**` packages that declare `frontend.admin` and
   `admin.extension.settings.*` contributions.
2. Compose output lives at `storage/theme-releases/dev-compose/` and reuses the
   same Nuxt env contract: `SFORUM_ADMIN_REGISTRY_ROOT`, `SFORUM_THEME_LAYER`,
   `SFORUM_WEB_RELEASE_ID=dev-local`.
3. **Symlink** admin roots and theme layers to source so Vite HMR works.
4. **Inline** locales into registry `metadata.ts` (same as Web Release).
5. Watch builtin themes/plugins; recompose on change; **restart Nuxt only when
   `compositionHash` changes** (locales/manifest/contributions), not on every
   `.vue` save.
6. **`SFORUM_DEV_USE_RELEASE=1`** restores the previous “consume full Web
   Release `current.json`” path for validating production packages.
7. Production Web Release pipeline is unchanged.

## Non-goals (this decision)

- Uploaded / non-builtin extension packages in default dev-compose.
- Full dependency install isolation or artifact digests for local compose.
- P2 “Nuxt extends source trees without any compose directory”.

## Consequences

- Daily theme/plugin admin UI work: edit source → refresh (HMR); no manual
  Web Release for builtin packages.
- Enabling plugins in admin still uses API lifecycle; custom UI presence for
  builtin packages no longer depends on the latest production release id.
- Operators validating uploaded themes must set `SFORUM_DEV_USE_RELEASE=1`.
