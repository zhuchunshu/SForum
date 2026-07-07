# Mail Module

Mail provider contract and password-reset mail flow.

## Backend

- `apps/api/app/Support/Mail` package: `Provider` interface
  (`Send(ctx, Message) error`, `Name() string`), `Service` resolver, and three
  providers.
- Providers: `noop` (silent discard), `dev_log` (structured log for
  development/Mailpit), `smtp` (standard library `net/smtp`, supports
  `none`/`starttls`/`tls`).
- `Service.Send` resolves the provider from runtime options and delegates.
  No provider failure is swallowed silently except in the password-reset
  request path (privacy: never reveal whether the email exists).

### Runtime Options (web_options)

- `mail.provider`: `dev_log` (development default) / `smtp` / `noop`.
- `mail.from_address`, `mail.from_name`.
- `mail.smtp.host`, `.port`, `.username`, `.password` (secret),
  `.encryption` (`none`/`starttls`/`tls`).

Options are registered in `app/Models/Options` with validation
(`normalizeMailProvider`, `normalizeMailEncryption`, `normalizeMailPort`) and
coercion to recommended defaults. The `mail.smtp.password` is a secret option:
empty updates preserve the stored value; reset clears it and the UI states
that clearly.

### Endpoints

- `POST /api/v1/admin/mail/test` sends a test mail to a submitted recipient
  (requires `settings.manage`).
- `POST /api/v1/auth/password-reset/request` always returns success regardless
  of email existence. Rate-limited; optionally protected by human verification
  when the `password_reset` purpose is enabled.
- `POST /api/v1/auth/password-reset/confirm` consumes a single-use token,
  checks expiry, updates the password hash, and invalidates the token.

`password_reset_tokens` table (migration `202607070005`): `token_hash` unique
(sha256 of raw token), `expires_at`, `consumed_at`, `request_ip_hash`.

`PasswordResetService` (in `app/Models/Identity`) coordinates the flow. Token
lifetime defaults to 30 minutes. Password policy requires at least 12
characters (argon2id).

## Frontend

- `/forgot-password` and `/reset-password` pages in the default theme layer.
- Login page links to `/forgot-password`.
- `/admin/settings/mail` page (System folder, `settings.manage`) with provider
  select, SMTP config, secret-preserve password field, one-click test mail,
  and restore-to-defaults guidance. Does not extend
  `apps/web/app/pages/admin/settings/index.vue`.
