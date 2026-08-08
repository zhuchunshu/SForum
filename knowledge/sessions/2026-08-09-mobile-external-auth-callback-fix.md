# 2026-08-09 Session Handoff

## Changed

- HTTPS OAuth browser-binding cookies now use `SameSite=None; Secure`; HTTP
  development keeps `SameSite=Lax`. The random HttpOnly value, 30-minute TTL,
  and server-side digest comparison remain unchanged.
- External-auth callback feedback now separates the auth surface from the root
  global surface. Errors on `/login` and `/register` remain inline; errors on
  arbitrary return pages render a persistent root `SFAlert`; success uses a
  bounded toast and non-error alerts auto-dismiss after 10 seconds.
- Root/global feedback is derived from the callback query for SSR and client
  rendering, so an OAuth error is visible even when the callback returns to the
  home page or a public error page.

## Decisions

- The browser binding is relaxed only when the effective cookie is Secure. The
  existing state, actor/session checks, one-use transaction, and digest binding
  remain the authoritative CSRF protections.
- The root error alert is intentionally persistent and non-closable in the
  current Nuxt UI development runtime because its existing ToastProvider
  hydration mismatch can prevent Vue close handlers from attaching. Navigation
  or a new authentication attempt leaves the callback surface.

## Next

- Validate a real configured provider on affected mobile browsers, including
  system-browser account switching and any provider-specific POST callback.
- Track the existing Nuxt UI `ToastProvider` hydration mismatch separately; it
  is not introduced by the external-auth repair and can affect client-only
  event handlers in local development.

## Open Questions

- Confirm whether production deployments use only GET callbacks or require
  `SameSite=None` for a provider's form-post callback flow.
