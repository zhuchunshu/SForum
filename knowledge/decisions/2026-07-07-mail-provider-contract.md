# Decision: Mail Provider Contract

## Status

Accepted

## Context

SForum needs outbound mail for password reset and future notifications. The
core framework should expose a stable provider contract so deployment-specific
mail delivery (SMTP, transactional services) can live behind a single
interface, with safe defaults that work out of the box for development and
fail loudly (not silently) in production when SMTP is unconfigured.

AGENTS.md requires plugin-first development for deployment-specific systems:
mail delivery is a provider concern. Core should expose the contract and safe
defaults, not bundle vendor logic.

## Decision

Define a `mail.Provider` interface in `app/Support/Mail`:

```go
type Provider interface {
    Send(ctx context.Context, message Message) error
    Name() string
}
```

Ship three built-in providers:

- `noop`: silently discards. Used when mail is explicitly disabled so
  production never pretends delivery succeeded.
- `dev_log`: writes structured log entries. Development default; works with
  Mailpit/console inspection.
- `smtp`: standard-library `net/smtp` with `none`/`starttls`/`tls`. Chosen
  over a third-party SMTP library to avoid a new dependency for V1; the
  standard library covers STARTTLS + AUTH PLAIN for the vast majority of
  SMTP servers. A future plugin can add a richer provider.

A `mail.Service` resolves the active provider from runtime options
(`mail.*` in `web_options`) on each send, so operators can switch providers
without redeploying. Options are registered in the Options service with
validation and coercion to recommended defaults; `mail.smtp.password` is a
secret option preserved on empty updates.

## Consequences

- Core owns the contract and safe defaults; vendor SMTP/transactional
  providers remain plugin territory.
- Password-reset request never reveals whether an email exists: the endpoint
  always returns success, and mail-send failures inside that path are
  swallowed (logged) rather than surfaced.
- `net/smtp` is sufficient for V1 but lacks some advanced features (DKIM
  signing, connection pooling). A plugin can replace the SMTP provider when
  those are needed.
