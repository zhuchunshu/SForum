# 2026-07-23 Content-Addressed Asset Namespace

## Status

Accepted and implemented.

## Context

Theme CSS, fonts, public extension modules, and trusted admin components were
served from `/api/v1` routes. Those handlers correctly owned active-artifact,
trust, MIME, and path checks, but their public URLs mixed JSON control APIs with
immutable byte delivery. Query-based theme versions also did not propagate to
relative `url(...)` references inside CSS.

## Decision

1. Reserve `/_sforum` for Host-owned non-API delivery surfaces. Plugins cannot
   claim this namespace through Route Registry.
2. Public immutable bytes use `/_sforum/assets`; authenticated admin component
   bytes use `/_sforum/private-assets`.
3. Theme URLs include the exact package digest before the package-relative
   path: `/_sforum/assets/themes/{id}/{packageDigest}/{path}`. Relative CSS
   resources therefore inherit the same digest path automatically.
4. Public extension package assets use
   `/_sforum/assets/extensions/{id}/{packageDigest}/{path}`. Trusted admin
   components use `/_sforum/private-assets/extensions/{id}/{digest}/{asset}`.
5. Nuxt streams these same-origin paths to existing Go handlers. Go remains
   authoritative for active artifacts, exact trust, authentication, MIME,
   containment, and response headers; Nuxt never reads extension package files.
6. Existing `/api/v1/...assets...` handlers remain as internal upstream and
   compatibility endpoints until an explicit deprecation removal.
7. Only content-addressed `/_nuxt` and `/_sforum/assets` paths receive public
   one-year immutable caching. A filename extension alone never proves that a
   resource is immutable.
8. Public asset proxy requests omit cookies, bearer credentials, and CSRF
   headers. Successful responses strip `Set-Cookie` and vary only by content
   encoding; non-200 responses are `no-store` to prevent durable negative
   caching.

## Consequences

- Static resource URLs no longer expose the REST API namespace.
- Changing a package changes every emitted resource URL without a manual
  version bump or forced browser refresh.
- Theme CSS can use ordinary relative font and image URLs safely.
- Production can later route `/_sforum/assets` directly to Go, object storage,
  or a CDN without changing descriptors or theme manifests.
- Revocation and theme selection remain control-plane decisions. Previously
  downloaded browser bytes cannot be recalled and must never be treated as the
  authorization boundary for executing extension code.
