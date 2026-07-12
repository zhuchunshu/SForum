# Mail Module

Core owns a provider-neutral asynchronous mail framework. It does not implement
SMTP, log delivery, no-op delivery, authentication, or TLS.

## Core

- `mail.provider` is a first-class extension provider slot.
- `mail_deliveries` records `queued/sending/sent/failed/skipped` state without
  storing provider credentials.
- River job `mail.deliver` carries only a delivery ID and runs on the `mail`
  queue. Standalone workers reconcile enabled plugin runtimes themselves;
  when the worker is embedded in the API it reuses the API extension runtime
  (single plugin process per enabled backend extension).
- No selected provider produces `skipped/provider_unavailable`; in-app
  notifications and forum writes remain successful.
- Password-reset token, delivery, and River job are inserted in one PostgreSQL
  transaction. The public request remains enumeration-safe.
- `settings.manage` protects provider selection, reset, recent deliveries, test
  mail, and mail-provider plugin settings. `extension.manage` still controls
  plugin enable/disable. Disabling the selected plugin clears the selection.
- Core admin route `/settings/mail` is the visible **Mail and Notifications**
  center. It owns provider selection, custom-recipient test mail, notification
  policy, self-test notification, and delivery history; queued test mail is not
  presented as synchronously delivered.
- `POST /admin/mail/test` recipient resolution: explicit JSON `recipient` first,
  otherwise `site.admin_email` via `Options.AdminEmail`. Both empty → `422
  mail.test_recipient_required`. Response `data.recipient` echoes the resolved
  address. The admin UI prefills from admin web-options when available.

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

SMTP is RPC/provider-only: `RouteTarget` is empty (no HTTP proxy base). The
host treats empty/`disabled`/`none` as “no route target” so SSRF loopback
validation does not mark the runtime failed.

The plugin owns all SMTP product copy and optional custom settings UI:

- Manifest `settings` labels/descriptions/placeholders/groups/options use
  multi-locale maps (`zh-CN` / `en-US`); the host settings API resolves them
  via `Accept-Language` into plain strings for the generic form fallback.
- `frontend.admin` ships `SmtpSettingsPage.vue` plus plugin locale JSON.
- Contribution `admin.extension.settings.page` replaces the host generic form
  for this extension only. Core chrome has no SMTP-specific field or port
  branching; secrets stay preserved on empty update and recommended restore.

## Compatibility

Startup runs idempotent migration marker `mail_provider_plugin_v1` after
built-in sync. Legacy `mail.smtp.*` values are copied without overwriting newer
plugin settings. Legacy `mail.provider=smtp` enables/selects `sforum.smtp`;
`dev_log`, `noop`, blank, and unknown values become unconfigured.
