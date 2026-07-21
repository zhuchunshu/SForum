# Password Policy Settings And Registration Strength Feedback Design

## Goal

Make SForum's password policy configurable from the admin settings area and show
real-time password qualification feedback below password inputs, starting with
registration.

The backend remains authoritative. The frontend reads the public policy only to
guide users before submission.

## Context

Current behavior is hard-coded:

- Registration validation requires passwords to be at least 12 characters.
- `HashPassword` repeats the same 12-character guard.
- Password reset confirmation also depends on `HashPassword`, so it currently
  inherits the same hard-coded rule.
- The default theme registration page displays a fixed password hint.
- Runtime options already support public/admin-safe settings through
  `web_options`, `GET /api/v1/web-options`, and admin batch updates guarded by
  `settings.manage`.

This feature belongs in core identity and runtime options rather than a plugin:
password policy is part of the account security boundary, and every password
creation path must share the same authoritative policy.

## Library Survey

No new package is needed for the first release.

- NIST SP 800-63B emphasizes minimum length, accepting full passwords without
  truncation, allowing printable/unicode characters, guidance to help users
  choose strong passwords, and not imposing composition rules by default:
  <https://pages.nist.gov/800-63-4/sp800-63b.html>
- OWASP's Authentication Cheat Sheet recommends application-enforced minimum
  length, at least 64 characters for maximum length, no silent truncation, and a
  password strength meter:
  <https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html>

SForum should therefore ship a length-first recommended policy. Optional
composition toggles are available for operators that need them, but the
recommended defaults keep them disabled.

## Chosen Approach

Add a small password policy model to core identity:

- Minimum length: configurable, default 12 for compatibility with the current
  product behavior.
- Maximum length: configurable, default 128, with validation ensuring it is at
  least the minimum and at least 64.
- Optional requirements: lowercase letter, uppercase letter, number, and symbol.
  All are disabled by default.
- Public policy read: included in public web options so registration and
  password reset can render accurate guidance.
- Admin policy edit: added to the existing admin settings basic tab under an
  account security section.
- Restore recommended defaults: one-click reset in the admin UI restores the
  recommended password policy defaults without affecting unrelated site fields.

Rejected alternatives:

- Preset-only policies are simpler but too limited for operators.
- Raw regex policies are flexible but unfriendly and easy to misconfigure.

## Runtime Options

Add these public runtime options:

```text
identity.password.min_length = 12
identity.password.max_length = 128
identity.password.require_lowercase = disabled
identity.password.require_uppercase = disabled
identity.password.require_number = disabled
identity.password.require_symbol = disabled
```

Validation rules:

- `min_length` must be within `8..128`.
- `max_length` must be within `64..512`.
- `max_length >= min_length`.
- Boolean-like options normalize to `enabled` or `disabled`.
- Missing or invalid stored rows fall back to recommended defaults.

Permission boundary:

- Updating these options requires `settings.manage`, matching the existing
  site-level identity/security configuration surface.
- API policy checks stay authoritative; frontend admin visibility is only UX.

## Backend Design

Add a `PasswordPolicy` value type to the identity module. It should expose:

- `RecommendedPasswordPolicy() PasswordPolicy`
- `ValidatePassword(password string) FieldMessages` or equivalent
- `Describe(locale string)` only if backend localized field messages need policy
  parameters in one place

The validator should count runes for length, matching current behavior and NIST
guidance that each Unicode code point counts as one character for length
evaluation.

The identity service receives a policy resolver instead of reading options
directly inside low-level password hashing. Suggested boundary:

```go
type PasswordPolicyResolver interface {
    PasswordPolicy(ctx context.Context) (PasswordPolicy, error)
}
```

Registration flow:

- `ValidateRegister` loads the runtime policy and validates username, email, and
  password before human verification is consumed.
- `Register` repeats the same policy validation inside the bootstrap transaction
  path before hashing.
- Existing conflict checks and initial `super_admin` behavior stay unchanged.

Password reset flow:

- `ConfirmPasswordReset` validates `NewPassword` against the same runtime policy
  before updating credentials.
- The controller maps policy failures to the existing field-aware validation
  response when possible. If password reset only has a top-level error today, it
  may return the stable password policy reason first and add field mapping in a
  follow-up if needed.

Hashing:

- `HashPassword` should no longer own mutable policy decisions. It should hash
  the password it is given and keep only cryptographic concerns.
- Callers that establish or change passwords validate policy before hashing.

## Frontend Design

Add shared password policy helpers in the web app:

- Resolve the runtime policy from `useWebOptions()`.
- Compute requirement rows for a password.
- Compute progress as passed requirements divided by total requirements.
- Generate a concise localized hint string from the current policy.

Registration page behavior:

- Replace the fixed hint with a compact progress indicator below the password
  input.
- Show a small progress bar and requirement rows:
  - length range
  - lowercase if enabled
  - uppercase if enabled
  - number if enabled
  - symbol if enabled
- Use icon-library icons through Nuxt Icon/Lucide, not emoji.
- Keep field-level API errors visible below the same input.
- Set `aria-describedby` to include the progress/help text and error when
  present.

Password reset page:

- Use the same policy helper for `canSubmit`, placeholder/hint text, and
  progress feedback, so account creation and password reset do not drift.

Admin settings page:

- Add an "Account security" section to the existing basic settings tab.
- Controls:
  - numeric input for minimum length
  - numeric input for maximum length
  - toggles/checkboxes for lowercase, uppercase, number, symbol
  - a restore recommended defaults button for only the password policy fields
- The recommended path should be visually obvious: length-first, optional
  composition disabled.
- Save through the existing `saveMany` admin web-options batch path.
- Success uses the existing toast style.

## API And Contract Design

Update modular OpenAPI:

- `contracts/openapi/schemas/options.yaml` option name enums include the new
  `identity.password.*` names.
- Identity registration validation examples mention configurable policy instead
  of hard-coded 12 characters.
- If password reset starts returning field-level password policy errors, update
  the relevant identity schema and path examples.

Run `ruby scripts/validate-openapi-refs.rb` after editing contract files.

## Testing Strategy

Backend tests:

- Options service lists new password options publicly with recommended defaults.
- Options service validates min/max bounds and boolean normalization.
- Options service rejects `max_length < min_length` in merged updates.
- Registration accepts a password matching a customized policy.
- Registration rejects passwords that violate each enabled requirement.
- Password reset confirmation uses the same customized policy.
- `HashPassword` still produces verifiable Argon2id hashes after policy removal.

Frontend tests:

- `useWebOptions` resolves recommended password policy defaults.
- Password policy helper computes progress and requirement rows correctly.
- Registration page rendering test confirms the password guidance is SSR-safe.
- Existing auth route rendering tests continue to pass.

Manual/browser QA:

- Registration route loads without framework overlay.
- Typing into the password field updates progress and requirement rows.
- A policy configured in admin settings changes the registration/reset hints
  after options refresh.
- Admin reset restores recommended password policy defaults and shows a toast.

## Knowledge Base Updates

After implementation:

- Update `knowledge/modules/identity.md` with the configurable password policy
  behavior.
- Update `knowledge/modules/options.md` with the new public runtime options.
- Add a decision record if the final implementation changes the recommended
  default from the current compatibility value of 12.
- Add a short session handoff with changed files, verification, and remaining
  risks.
