# 2026-07-05 Session Handoff

## Changed

- Login failure copy is now actionable while still using the generic
  `auth.invalid_credentials` reason.
- Registration validation returns localized field-level messages in
  `data.fields` for `username`, `email`, `password`, and `humanVerification`.
- The Nuxt registration page displays backend field errors next to the relevant
  controls and resets ALTCHA tokens when human verification fails.
- OpenAPI and tests now document/cover the registration field error contract.

## Decisions

- Do not distinguish missing account from wrong password during login.
- Keep the first registration rules minimal: username required, email required
  and basically valid, password at least 12 characters.
- Do not introduce `go-playground/validator` yet; current rules stay local to
  the identity service.

## Next

- Consider adding username length/character rules only after product policy is
  decided.
- If more forms need field-level errors, extract a shared frontend form-error
  helper or component.

## Open Questions

- Should email verification be required before posting or account recovery?
