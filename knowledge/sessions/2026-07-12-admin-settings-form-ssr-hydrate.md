# 2026-07-12 Session Handoff

## Changed

- Fixed admin settings forms showing default values instead of saved options
  after SSR hydrate. Root cause: `useAsyncData` handlers that mutated local
  `reactive` form state only ran on the server; the client reused payload and
  re-initialized forms to defaults, so `<select>` and similar controls looked
  unloaded.
- Pattern: return data from `useAsyncData`, apply to form via
  `watch(data, apply, { immediate: true })` (same as `admin/seo.vue`).
- Updated pages:
  - `apps/web/app/pages/admin/settings/index.vue`
  - `apps/web/app/pages/admin/personalization.vue`
  - `apps/web/app/pages/admin/forum/settings.vue`
  - `apps/web/app/pages/admin/attachments.vue`

## Decisions

- Keep SSR for admin pages; do not flip to client-only. Prefer payload +
  watch form hydration over side effects inside the fetch handler.

## Next

- When adding new admin option forms, never put form mutations only inside
  `useAsyncData` handlers.

## Open Questions

- None.
