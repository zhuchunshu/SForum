# 2026-07-05 ALTCHA Scenario Settings Handoff

## Changed

- Added public runtime CAPTCHA scenario options for registration, password
  reset, login risk, and posting risk.
- Updated the runtime human-verification service so disabled purposes skip
  challenge and verify requirements while enabled purposes remain enforced by
  the API verifier.
- Updated the admin Site Settings > CAPTCHA tab with scenario switches, numeric
  TTL minutes, suggested TTL/cost buttons, ALTCHA secret generation,
  show/hide-secret control, ALTCHA widget behavior controls (`type`, `auto`,
  `display`, logo/footer visibility, workers, and minimum duration), and an
  ALTCHA configuration summary.
- Registration now renders ALTCHA only when the provider is `altcha` and the
  registration scenario is enabled, and passes the public ALTCHA widget
  behavior settings into the web component.
- Updated OpenAPI option enums, backend tests, frontend helper tests, and i18n
  copy for the new settings.

## Decisions

- Keep scenario switches public so frontend pages can decide whether to render
  ALTCHA widgets, but keep API verification authoritative.
- Default registration verification scenario to enabled and the other planned
  scenarios to disabled until their flows are wired.
- Store ALTCHA challenge TTL as the existing Go duration value but edit it in
  the admin UI as whole minutes for operator clarity.
- Keep widget behavior settings public because they are HTML/web-component
  presentation controls; do not expose unsafe advanced widget knobs such as
  test, mockError, verifyUrl, custom fetch, or debug through admin settings.

## Verification

- `go test ./app/Models/Options ./app/Support/HumanVerify ./app/Http ./bootstrap`
- `bun test`
- `bun run typecheck`
- Playwright with local Chrome checked `/control-panel/settings` on desktop
  `1440x1000` and mobile `390x844`; CAPTCHA tab rendered, scenario text was
  visible, TTL/cost hints were visible, Generate Secret produced a 64-character
  secret, suggestion buttons set TTL to `20` and cost to `3000`, and browser
  console had no warnings/errors.

## Next

- When password reset, login-risk, and posting-risk flows are implemented,
  reuse `HumanVerify` purposes and the scenario switches instead of adding new
  provider flags.
- Tune production ALTCHA cost and TTL on expected low-end client devices.

## Open Questions

- What production ALTCHA cost should be recommended after testing real target
  devices?
