# 2026-07-27 APP URL Runtime Fallback Handoff

## Changed

- `site.url` is now an optional operator override; empty means inherit the
  deployment `APP_URL`.
- Public option reads return the effective URL. Admin reads additionally expose
  `overrideValue`, `fallbackValue`, and `inherited`.
- Basic settings can clear the override in one click and show the active
  environment fallback.
- New installs no longer persist `APP_URL` into `web_options.site.url`.
- Existing materialized `site.url` values equal to the current `APP_URL` are
  normalized back to an empty override; distinct operator values are preserved.

## Decisions

- OAuth callbacks, CSRF trusted origins, and cookie security remain bound to
  trusted environment `APP_URL`; the runtime override is for public product
  URLs such as pages, SEO, and outbound links.

## Next

- Run focused Options tests, Nuxt typecheck, and OpenAPI reference validation.

## Open Questions

- None.
