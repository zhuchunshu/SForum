# Mail Module

Core owns a provider-neutral asynchronous mail framework. It does not implement
SMTP, log delivery, no-op delivery, authentication, or TLS.

## Core

- `mail.provider` is a first-class extension provider slot.
- `mail_deliveries` records `queued/sending/sent/failed/skipped` state without
  storing provider credentials.
- River job `mail.deliver` carries only a delivery ID and runs on the `mail`
  queue. API and standalone/embedded workers each reconcile enabled plugin
  runtimes.
- No selected provider produces `skipped/provider_unavailable`; in-app
  notifications and forum writes remain successful.
- Password-reset token, delivery, and River job are inserted in one PostgreSQL
  transaction. The public request remains enumeration-safe.
- `settings.manage` protects provider selection, reset, recent deliveries, test
  mail, and mail-provider plugin settings. `extension.manage` still controls
  plugin enable/disable. Disabling the selected plugin clears the selection.

## SMTP Plugin

Protected built-in plugin ID: `sforum.smtp`, under
`extensions/builtin/plugins/sforum-smtp`.

The plugin exclusively owns SMTP host/port/sender settings, password,
SMTP/STARTTLS/implicit TLS, AUTH PLAIN, MIME message assembly, and transport
error classification. Plugin secrets live in `extension_settings`, are masked
from API responses, are preserved by empty updates, and reach only that plugin
process through `SFORUM_SETTING_*` environment variables.

`scripts/build-builtin-plugins.sh` builds the local subprocess before API or
worker dev startup. The API Dockerfile builds the Linux executable into the
built-in package.

## Compatibility

Startup runs idempotent migration marker `mail_provider_plugin_v1` after
built-in sync. Legacy `mail.smtp.*` values are copied without overwriting newer
plugin settings. Legacy `mail.provider=smtp` enables/selects `sforum.smtp`;
`dev_log`, `noop`, blank, and unknown values become unconfigured.
