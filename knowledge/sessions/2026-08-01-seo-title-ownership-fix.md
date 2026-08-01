# 2026-08-01 Session Handoff

## Changed

- Removed the root-level SEO title transformation from `apps/web/app/app.vue`.
  A resolved page title is preserved; the root only falls back to the site name
  when no title exists.
- Made `useSForumSeo` the public-page title/robots authority and migrated
  authentication, legal, topic authoring, and account settings surfaces away
  from manual site-name concatenation.
- Kept `seo.meta_title_template` as a compatibility fallback when resolving
  `seo.page.title_template`; it is no longer applied a second time by the root.
- Added regression tests for title fallback, single site-name emission, and
  the profile settings source contract.

## Verification

- `cd apps/web && bun run typecheck` passed.
- `cd apps/web && bun test tests/framework tests/seo tests/identity tests/forum`
  passed: 467 tests, 0 failures, 2799 assertions.
- `cd apps/web && bun run build` passed.
- The existing port-3000 runtime was not restarted. It is an older published
  instance and still serves `SForum - SForum`; deploy/restart the new Web image
  before treating runtime HTML as updated.

## Next

- Restart or redeploy the actual Nuxt/Caddy runtime, then verify representative
  SSR pages contain one `<title>` and matching `og:title` without duplication.

## Open Questions

- None for the source fix. Keep the legacy admin SEO field until a separate
  migration/deprecation decision removes the compatibility option.
