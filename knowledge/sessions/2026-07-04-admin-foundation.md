# 2026-07-04 Session Handoff - Admin Foundation

## Changed

- Added configurable admin route prefix support in Nuxt runtime config:
  `NUXT_PUBLIC_ADMIN_ROUTE_PREFIX` defaults to `/control-panel`.
- Added admin route helpers in `apps/web/app/utils/adminRoutePrefix.ts` and
  `apps/web/app/composables/useAdminRoutes.ts`.
- Added `apps/web/app/layouts/admin.vue` using Nuxt UI Dashboard components and
  Nuxt Icon lucide icons.
- Migrated the existing admin overview and roles pages to the admin layout and
  Nuxt UI components.
- Updated login/register post-auth redirects to use the configurable admin
  route helper.
- Added `tests/validate-admin-framework.ts` and wired it into
  `scripts/test.sh`.

## Decisions

- Public admin URLs are configurable while source files stay under
  `pages/admin`.
- Admin UI should use Nuxt UI Dashboard components directly; custom `SF*`
  components remain for public/forum UI.
- Admin links should not use `localePath()` directly because Nuxt i18n route
  resources are generated from source paths before the admin prefix rewrite.

## Next

- Add real admin user-management pages under the existing shell.
- Add moderation/audit/settings pages when their backend contracts are ready.
- Run browser visual QA once the web service is reachable on the expected local
  port.

## Open Questions

- Whether production should keep `/control-panel` or set a deployment-specific
  admin prefix.
