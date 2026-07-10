# Auth Return Navigation Design

## Goal

Make login and registration navigation preserve the user's forum context. An
authenticated visitor who opens an auth page should return to a safe internal
source page, while a guest sent to login from a protected route should return
to that original destination after authentication.

## Current Behavior

- Protected route middleware redirects guests to `/login` without preserving
  the requested route.
- Login and registration always navigate to the admin home for users with
  `admin.access`, otherwise to the forum home.
- The default and Signal Garden themes implement this behavior separately.

## Approaches Considered

1. Use `history.back()`. This is compact but can return to an external site,
   another auth page, or an unusable browser-history entry, and it does not
   work reliably during SSR navigation.
2. Use only an explicit `redirect` query parameter. This reliably restores
   protected routes but cannot preserve context when an authenticated user
   manually opens an auth page from an ordinary forum page.
3. Use a validated explicit target with a same-origin referrer fallback. This
   covers both flows while keeping navigation deterministic and safe.

Use approach 3.

## Design

Add a shared frontend utility or composable that owns auth return navigation.
It resolves destinations in this order:

1. A validated `redirect` query value.
2. A validated same-origin referrer captured when the auth page is entered in
   the browser.
3. The localized forum home page.

The resolver accepts only local absolute paths beginning with `/`. It rejects
protocol-relative paths, absolute URLs, malformed values, and auth-entry paths
such as login and registration in every supported locale. Query strings and
fragments on otherwise valid internal destinations are preserved.

The global auth middleware and admin middleware append the requested `to.fullPath`
as `redirect` when sending a guest to login. They must not overwrite an existing
safe destination or create an auth-page loop.

Login and registration pages in both bundled themes use the shared resolver:

- On entry, once session state is known, an authenticated user immediately
  navigates to the resolved destination.
- After successful login or registration, the same destination is used.
- No admin-specific fallback is applied. A user with admin access returns to
  the admin area only when that area was the explicit original destination;
  otherwise they return to their forum context or the forum home.

The auth pages remain SSR-rendered. Referrer fallback is client-only because
the server cannot reliably know browser history. Explicit `redirect` handling
and the home fallback work during SSR.

## Security And Failure Handling

- Redirect validation is frontend defense in depth; no API authorization rule
  changes.
- External and invalid targets fall back silently to the localized forum home.
- Auth service unavailability keeps the existing behavior: the page remains
  usable and does not treat an unknown session as authenticated.
- Repeated redirects are prevented by excluding auth-entry destinations.

## Testing

Add focused unit coverage for the shared resolver:

- Accept a local path with query and fragment.
- Reject external, protocol-relative, malformed, and auth-entry targets.
- Prefer explicit redirect over referrer and use home as the final fallback.
- Preserve locale-aware internal paths.

Update route rendering/source tests to verify both auth middlewares preserve
`to.fullPath` and both bundled themes use the shared navigation behavior for
already-authenticated visits and successful login/registration. Run the
frontend auth test set and Nuxt typecheck.

## Scope

This change is frontend-only. It adds no API endpoint, permission, persistent
setting, or dependency. Password reset pages keep their current behavior.
