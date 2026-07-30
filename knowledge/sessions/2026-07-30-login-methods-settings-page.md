# 2026-07-30 Session Handoff

## Changed

- Moved account login methods out of `/settings/security` into independent
  `/settings/login-methods` with its own sidebar item, route shell, Host page
  component, Page Registry ID `forum.settings.login_methods`, ViewModel, Host
  island, and built-in default/Nocturne templates.
- Account settings navigation now separates login methods, local password,
  account security, personal access tokens, and notification preferences.
- The security page now focuses on active devices and login history; external
  identity linking renders from the login-methods page with return path
  `/settings/login-methods`.
- Built-in theme exact-artifact declarations now include the login-methods,
  password, and tokens settings templates in both `theme.json` and Manifest V3
  `packageFiles`/`templates`.

## Decisions

- Login methods, local password, and personal access tokens are separate
  account settings surfaces instead of sections inside account security. The
  API remains the authority for recent-auth and sensitive actions.

## Verification

- `cd apps/web && bun test tests/framework/pageOutlet.test.ts tests/framework/presentationOwnershipRemaining.test.ts tests/identity/securitySettingsChrome.test.ts tests/identity/accountSecurityM4b.test.ts tests/forum/profileSettingsCanvas.test.ts`
  — 104 passed.
- `cd apps/api && go test ./app/Support/Pages ./app/Support/ThemeCompiler ./app/Models/PageViewModels ./app/Http/Controllers/Identity`
  — passed.
- `go run ./cmd/sforum extension validate ../../extensions/builtin/themes/sforum-default`
  and `.../sforum-nocturne` — passed with 31 V3 template files each.
- Activated default theme digest
  `19a7c3b459772a721eabb6e94cf97c972ea73ef82d91a39447313bb0f2ac0b12` in the
  dev DB. Playwright QA for `/settings/login-methods` at `1440x1000` and
  `390x844` confirmed `data-provider="sforum.default-theme"`,
  `data-template="1"`, no fallback, no console errors, and no horizontal
  overflow.

## Next

- If the active theme is reset or another theme becomes active, reactivate a
  staged artifact that includes `forum.settings.login_methods` before using
  browser evidence for this page.

## Open Questions

- None.
