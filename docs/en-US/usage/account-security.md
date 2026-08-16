# Account & security

[← Usage](./README.md)

For operators and end users: email verification, passwords, sessions, access
tokens, external login bindings, personal appearance, and notification
settings.

## Email verification

- When the site enforces email verification (Site Settings → Account security),
  unverified users are redirected to the waiting page `/email-verification`.
- Verification links are one-use hashed links and stop working once used. The
  waiting page sends mail only on explicit user action, after completing the
  operator-configured ALTCHA challenge.
- Verified users can publish topics/comments/attachments normally; otherwise
  site policy decides.
- Administrators can inspect verification state and verify/reset a user's email
  status from the admin UI (resets invalidate all old links).
- API: `POST /auth/email-verification/request`,
  `POST /auth/email-verification/confirm`.

## Password recovery and change

- **Recovery**: login page → "Forgot password" (`/forgot-password`); enter the
  email to receive a reset link; the link leads to `/reset-password` to set a
  new password. The flow is non-enumerating.
- **Send limits**: request/resend is limited by server-side cooldowns and
  per-target/per-IP limits (Site Settings → Account security); the page shows
  the countdown.
- **Change**: after login, use Settings → Password (`/settings/password`) to
  set or change the local password; changing requires recent authentication.
- Accounts without a local password (for example external-login-only accounts)
  can set one there.
- API: `POST /auth/password-reset/request`,
  `POST /auth/password-reset/confirm`, `POST /auth/password`.

## Sessions / revocation

Settings → Security (`/settings/security`) lists the account's login sessions:

- view the current session and session history (browser/device info);
- revoke one device session;
- "revoke other sessions" signs out every session except the current device;
- after sensitive operations such as a password change, revoke other sessions.

API: `GET /auth/sessions`, `DELETE /auth/sessions/{sessionId}`,
`POST /auth/sessions/revoke-others`.

## Personal Access Tokens (PAT)

Settings → Tokens (`/settings/tokens`) creates tokens for scripts and external
services:

- tokens look like `sft_<publicId>.<secret>`; the secret is shown once at
  creation — save it immediately;
- scopes are limited to permission keys you already hold;
- tokens can be revoked or rotated at any time; managing tokens requires a
  cookie session (token management APIs reject Bearer auth with
  `api_token.cookie_required`).

How to call APIs: [API usage](../development/api.md).

## External login bindings

- Sites may enable external logins (for example the built-in GitHub login); pick
  the provider on the login page.
- Unbound external accounts enter `/auth/continue`: prove an existing local
  account to auto-bind, or register and auto-bind; emails are never used to
  match accounts automatically.
- Settings → Login methods (`/settings/login-methods`) lists and unbinds
  external identities.
- API: `GET /auth/providers`, `GET /auth/external-identities`, etc.; see
  `contracts/openapi.yaml`.

## Personal appearance and notification settings

- **Appearance**: Settings → Appearance (`/settings/appearance`) chooses
  light/dark/system color mode and personal background palettes, with
  immediate preview and "restore site default".
- **Notifications**: Settings → Notifications (`/settings/notifications`)
  manages in-site, email, and browser (Web Push) preferences; administrators
  may constrain the available options.
- **Profile**: Settings → Profile (`/settings/profile`) manages username, bio,
  avatar, and interface language preference.

## Related

- Admin-side policy (enforcement, cooldowns, password policy, notification
  policy): [admin guide](./admin.md).
- API contract: [API usage](../development/api.md).
