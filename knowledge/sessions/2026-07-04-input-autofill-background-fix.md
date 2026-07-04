# 2026-07-04 Input Autofill Background Fix

## Changed

- Added WebKit autofill overrides for `SFInput`, `SFSearch`, and the standalone
  login/register `.auth-input` styles so browser-saved credentials keep the
  intended white input background and dark text.
- Preserved existing focus rings for autofilled focused fields by combining the
  inset autofill shadow with the focus shadow.
- Added autofill coverage checks to `tests/validate-sf-components.js` and
  `tests/validate-identity-ui.js`.
- Wired `tests/validate-sf-components.js` into `scripts/test.sh`.

## Verification

- `node tests/validate-sf-components.js` failed before the CSS fix and passed
  after it.
- `node tests/validate-identity-ui.js` failed before the auth page CSS fix and
  passed after it.
- `bun run typecheck` passed in `apps/web`.
- `./scripts/test.sh` passed.
- Existing dev server at `http://127.0.0.1:3000/login` returned the login page;
  compiled scoped CSS included the `.auth-input:-webkit-autofill` rules.

## Notes

- Browser plugin validation was blocked because the required `node_repl` JS
  control tool was not exposed in this session.
