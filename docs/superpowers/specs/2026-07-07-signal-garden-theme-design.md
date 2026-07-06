# Signal Garden Theme Design

## Goal

Build a second SForum public theme as an installable Nuxt Layer package under
`extensions/dev/themes/sforum-signal-garden`. The theme should be suitable for
upload-and-activate testing without changing core admin, auth/session, API, SEO,
i18n, or permission logic.

## Direction

Signal Garden uses a bright community style: natural green surfaces, warm yellow
activity highlights, coral alert accents, dense but friendly forum rows, and
8px-or-less component radii. It should feel lighter and more social than the
default Pine Teal theme while remaining readable and operational.

## Scope

- Add a theme manifest with `id=sforum.signal-garden`.
- Add a Nuxt Layer with public default/auth layouts.
- Add a custom public navbar and footer.
- Override homepage, login, and registration presentation.
- Keep existing host composables and API calls for auth, SEO, web options,
  i18n, locale switching, and color mode.
- Add a theme validation script.

## Boundaries

The theme must not declare backend runtime, plugin routes, hooks, events, jobs,
providers, migrations, permissions, or any core override outside the Nuxt Layer
contract. Uploaded theme activation remains asynchronous through the existing
theme release worker.

## Verification

Run `node tests/validate-signal-garden-theme.js` for theme package structure.
Run frontend build or typecheck with `SFORUM_THEME_LAYER` when the concurrent
workspace changes are stable enough to exercise the host Nuxt app.
