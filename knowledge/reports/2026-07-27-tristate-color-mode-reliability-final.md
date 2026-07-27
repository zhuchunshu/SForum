# Tri-State Color Mode Reliability Final Report

Date: 2026-07-28

## Scope And Architecture

- Completed M0-M5 without adding dependencies or changing the accepted
  browser-local persistence decision.
- `useColorModePreference` owns `system | light | dark` normalization, option
  metadata, writes, and resolved `light | dark` access.
- Public Host and admin menus consume that authority. The previous public/admin
  `MutationObserver`, local resolved refs, and binary writers were removed.
- Extension bridges receive readonly resolved appearance only.
- Development canonicalization is a development-only H3 middleware. It accepts
  only validated loopback `APP_URL`, HTML GET/HEAD requests, and supported local
  aliases on the canonical port; the redirect target never uses request Host.

## Behavior Matrix

| Behavior | Evidence |
| --- | --- |
| Automatic/Light/Dark normalization and setters | composable tests, 6 pass / 23 expectations |
| Automatic versus live resolution; explicit override | behavioral tests with changing resolved values |
| Public/admin option order, selected state, accessible labels | focused surface tests; authenticated admin DOM |
| Dark refresh and client navigation | authenticated admin Browser QA |
| Automatic on current system-light environment | trigger `外观：自动`, root class `light`, media query false |
| Selected-theme public and 404 surfaces | Browser rendered active theme without relevant console errors |
| Extension resolved-only contract | focused public/prebuilt extension tests |
| Alias canonicalization | `localhost/categories?group=core` -> `127.0.0.1` with path/query preserved |

The available Browser control surface could read the operating-system media
query but could not emulate system-dark or change it live. Those transitions
are covered by the composable behavioral test, not by rendered browser evidence.
The active selected theme owns its successful public navbar and does not render
Host `SFNavbar`; Host public desktop/mobile menu presentation is therefore
covered by focused source contracts rather than a forced runtime fallback.

## Origin And Persistence

- `APP_URL=http://127.0.0.1:3000` remains authoritative.
- Live alias HTML probe returned `307 Temporary Redirect`, `Cache-Control:
  no-store`, and the exact configured authority.
- Canonical requests do not loop. Vite `/_nuxt/@vite/client` on the alias was
  served directly; pure helper tests cover `/api`, `/_nuxt`, `/_sforum`, and
  `/health` exact/descendant exclusions plus malformed hosts/configuration.
- Browser UI selection wrote through Nuxt Color Mode. No direct storage edit,
  cookie migration, account sync, or new persistence key was introduced.

## SSR And Cache

- Anonymous `/` and `/categories` kept `s-maxage=600,
  stale-while-revalidate`; the topic probe kept `s-maxage=60`.
- The HTML 404 probe returned `404` and `Cache-Control: no-store`.
- Every probed HTML document had a preference-neutral `<html>` tag, no server
  `.light`/`.dark` class or serialized preference, and the same early script
  reading `localStorage` key `nuxt-color-mode`.
- No color/theme preference cookie was emitted. The invalid synthetic session
  cookie was intentionally not treated as authenticated; authenticated cache
  headers were not extracted because Browser policy prohibits reading session
  cookies.

## Verification

- `bun test tests/appearance/colorModePreference.test.ts tests/appearance/colorModeSurfaces.test.ts tests/framework/canonicalLocalOrigin.test.ts tests/themes/defaultThemeNavbar.test.ts tests/admin/adminProfessionalMode.test.ts tests/admin/prebuiltSettingsComponent.test.ts tests/extensions/publicExtensionMount.test.ts` -> PASS (45 tests, 300 expectations).
- `bun run typecheck` -> PASS.
- `bun run build` -> PASS (existing chunk/sourcemap/import-attribute warnings).
- `ruby scripts/validate-openapi-refs.rb` -> PASS (2287 refs, 54 files).
- `git diff --check` -> PASS.
- `bun test` -> FAIL (658 pass, 13 unrelated failures: stale auth component
  test paths plus existing page-outlet, proxy, topic-path, and moderation static
  contracts).
- `./scripts/test.sh` -> PARTIAL: architecture validation and all Go tests
  passed; compat farm then failed because `DATABASE_URL` /
  `SFORUM_TEST_DATABASE_URL` was absent.

## Residual Risks And Deferred Work

- Repeat rendered system-dark and live OS-switch QA on a browser surface that
  exposes media emulation; current behavioral coverage is deterministic but not
  pixel/runtime evidence.
- Repeat authenticated SSR cache-header inspection in a test harness that can
  provide a disposable session without exposing a user's browser cookie.
- The 13 unrelated full-web failures and compat-farm environment prerequisite
  remain repository gate debt outside this task.
- Account/database preference synchronization and cross-device synchronization
  remain explicitly deferred.
