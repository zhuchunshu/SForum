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
- Core admin route `/settings/mail` owns mail provider configuration, testing,
  and mail delivery history. Notification policy moved to the dedicated
  `/control-panel/settings/notifications` admin surface; existing `/admin/mail/policy`
  routes remain compatibility projections over the Notification resolver.
- Provider selection and settings navigation remain extension-generic.
  Deliveries owns status, template, and reason localization. Queued test mail
  is not presented as synchronously delivered.
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
SettingsLifecycle-backed secret references are resolved through SecretStore for
runtime settings and provider probes; API responses keep `value` empty while
preserving `secretSet=true` so operators can tell that a password/app password
is saved.

`scripts/build-builtin-plugins.sh` builds the local subprocess before API or
worker dev startup. The API Dockerfile builds the Linux executable into the
built-in package.

SMTP is RPC/provider-only: `RouteTarget` is empty (no HTTP proxy base). The
host treats empty/`disabled`/`none` as “no route target” so SSRF loopback
validation does not mark the runtime failed.

The plugin owns all SMTP product copy through the shared buildless Settings
Document and Provider Probe contract:

- Manifest `settings` labels/descriptions/placeholders/groups/options use
  multi-locale maps (`zh-CN` / `en-US`); the host settings API resolves the
  document via `Accept-Language` and `SFExtensionSettingsRenderer` renders it.
- The declared `provider_probe` Settings Action calls the plugin's bounded
  `ProviderProbe` RPC through host permission/lifecycle/input/timeout/audit
  policy. A disabled plugin uses the restricted short-lived Probe runtime and
  does not register normal routes, jobs, hooks, events, or provider slots.
- SMTP ships no executable admin frontend or custom settings SFC. Core chrome
  has no SMTP-specific field or port branching; secrets stay
  encrypted/masked, blank updates preserve them, and recommended restore is
  still host-owned.
- The core mail provider select defaults visually to the selected provider, or
  to enabled `sforum.smtp` when unconfigured, but selection is only persisted
  after the operator clicks save.

## Compatibility

Startup runs idempotent migration marker `mail_provider_plugin_v1` after
built-in sync. Legacy `mail.smtp.*` values are copied without overwriting newer
plugin settings. Legacy `mail.provider=smtp` enables/selects `sforum.smtp`;
`dev_log`, `noop`, blank, and unknown values become unconfigured.

## Architecture Debt Checkpoint

M6 focused Bun tests, architecture validation, Nuxt typecheck, production
build, full repository gate, and desktop/mobile browser closeout passed. The
completed program is archived at
`../plans/archive/2026-07/2026-07-28-architecture-boundary-debt-repayment.md`.
