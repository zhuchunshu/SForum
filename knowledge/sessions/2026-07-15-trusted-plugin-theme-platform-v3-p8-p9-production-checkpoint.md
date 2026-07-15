# 2026-07-15 Trusted Plugin And Theme Platform V3 P8-P9 Production Checkpoint

## Status

- Weighted V3 remains **57%**. P8 remains **17/18 (94%)** until the final
  production build and browser acceptance run; P9 remains **1/16 (6%)** until
  its currently dirty Asset Registry/public L2 work is reviewed and committed.
- All 23 Page Registry catalog ids now have a production frontend outlet. The
  seven protected auth/settings/composer pages resolve through themes only with
  their exact Host-owned mutation island; non-404 Host errors stay outside the
  replaceable `system.not_found` surface.
- Core Page ViewModels now carry service-backed product data. Private resolve
  responses are `private, no-store` and vary by Cookie, Authorization, and
  Accept-Language. Catalog feature gates, guest-read policy, 404 robots,
  Unicode route decoding, pagination/filter SEO, and topic URL-mode canonical
  behavior fail closed.
- The public Profile API now enforces the same `forum.guest.read` snapshot
  before loading recent topics; a missing policy snapshot returns 503.

## Commits

- `eec40b3dd fix(pages): prevent shared caching of resolved pages`
- `d27d46d25 fix(pages): enforce core view model policies`
- `199c3e099 feat(themes): render protected pages through host islands`
- `2cfa9a79d fix(pages): decode dynamic route parameters`
- `f434f9d98 feat(pages): connect remaining catalog outlets`
- `e5544e059 fix(pages): align view model SEO canonicals`
- `89367ef6a fix(profile): enforce guest read policy`

## Verification

- Focused Pages, Profile, PageViewModels, ThemeCompiler, and Providers Go tests
  pass on the shared worktree.
- The committed Host-island slice passed 321/321 Web tests, Nuxt typecheck,
  Page/Theme focused Go tests, and offline Page Registry validation.
- The catalog-outlet slice passed its focused Web tests, Nuxt typecheck, and SF
  component validation. A later full Web run has one expected failure because
  the dirty P9 worker changed `SFExtensionWidget` from disabled to executable
  before updating the old assertion; P9 must restore a green full gate.
- Do not count the current public L2 implementation as production evidence yet.
  OpenAPI, exact revoke/upgrade invalidation, real SSR/L1 component fallback,
  dependency cleanup/restart behavior, CSP handling, and browser E2E are still
  under review.

## Dirty Ownership

- P5 owns `database/migrator`, `database/coreauthority`, migration `026`, and
  the Extension database identity/runtime-lease files. Its migration must be a
  separate additive commit before authority/runtime wiring.
- P9 owns AssetRegistry, public frontend manifest/service/controller/config/
  OpenAPI files, `SFExtensionWidget`, `SFThemeTemplate`, public-extension Web
  runtime/tests, and L2 environment/Compose flags.
- Preserve the user's content-policy manifest, `.playwright-cli/`, and
  `.playwright-p8-nojs.json`; never stage them with V3 slices.

## Next

1. Review and split P5 into migration, shared identity, and migrator/runtime
   commits; prove same-version startup does not churn the ready revision.
2. Make P9 public L2 retain a real component SSR/L1 fallback, close dependency
   cleanup/restart and exact revoke/upgrade gaps, synchronize OpenAPI, and split
   manifest/registry/API/Web commits.
3. Run a clean production API/Nitro upload, trust, enable, render, restart,
   revoke, fallback, desktop/mobile, and JavaScript-disabled acceptance matrix.
   Only then close P8 or credit additional P9 rows.
