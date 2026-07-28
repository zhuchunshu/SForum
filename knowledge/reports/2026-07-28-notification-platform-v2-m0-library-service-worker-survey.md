# Notification Platform V2 M0 - Library And Service-Worker Survey

Date: 2026-07-28

## Scope And Result

This is an M0 evidence record only. It makes no production-code, dependency,
migration, or runtime-behavior change.

Result: use a mature Web Push library rather than implementing encryption;
implement SSE with the installed Fiber v3 stream API rather than adding an SSE
dependency; and keep the Web Push worker as a Host-owned, narrowly scoped,
static worker. The worker boundary is viable **only after** M6 reserves its
`/_sforum` namespace in the Page Registry as specified below.

## Repository Evidence

| Concern | Current evidence | Consequence for the frozen design |
| --- | --- | --- |
| HTTP framework | `apps/api/go.mod` pins `github.com/gofiber/fiber/v3 v3.0.0-rc.3`; its local source exposes `Ctx.SendStreamWriter(func(*bufio.Writer))`. | A small Fiber handler is sufficient for SSE. |
| HTTP compression | `apps/api/app/Http/server.go` installs `compress.New` globally. Fiber's installed compression middleware skips a response with `Cache-Control: no-transform`. | The future SSE response must set `Cache-Control: no-store, no-transform` before streaming. |
| Fiber request cancellation | Installed Fiber `ctx.go` documents `Ctx.Done()` as a no-op; its `Context()` defaults to `context.Background()`. fasthttp's `RequestCtx.Done()` closes for server shutdown, not a normal client disconnect. | The SSE loop must treat `bufio.Writer.Flush()` failure as disconnect, emit a bounded heartbeat to discover it, and also stop on process shutdown. Do not claim a request context supplies disconnect cancellation. |
| Existing delivery persistence | Migration `202607110016_notifications_mail_deliveries.sql` defines separate `notifications` and `mail_deliveries`. `PostgresStore.CreateBundleTx` creates both in one transaction; `mail.deliver` remains a River job. | Preserve `mail_deliveries`, `mail.deliver`, and Mail admin APIs. Do not reinterpret old rows as generic deliveries. |
| Current browser Host namespace | Nuxt routes `/_sforum/assets/**` through a Host proxy, and `pluginRouteProxy.isHostReservedPath` rejects all `/_sforum/**` plugin-route proxy attempts. | A Host static worker can live under `/_sforum/notifications/`, separate from extension assets and plugin routing. |
| Missing Page Registry reservation | `apps/api/app/Support/Pages/reserved.go` reserves `/__sforum`, but not `/_sforum`. | Before adding the worker, M6 must add `/_sforum` to `ReservedPrefixes` and add a regression test. The current route-proxy guard alone is insufficient as the complete namespace proof. |

## Web Push Library Survey

### Candidates

| Candidate | Maintenance and license evidence | Protocol and API fit | Decision |
| --- | --- | --- | --- |
| [`SherClockHolmes/webpush-go`](https://github.com/SherClockHolmes/webpush-go) `v1.4.0` | GitHub API checked 2026-07-28: 445 stars, pushed 2026-04-22; latest release `v1.4.0` published 2025-01-02; [MIT](https://raw.githubusercontent.com/SherClockHolmes/webpush-go/v1.4.0/LICENSE). | Implements RFC 8291 `aes128gcm` payload encryption and VAPID. Source provides `SendNotificationWithContext`, an injectable `HTTPClient`, `TTL`, `Topic`, `Urgency`, VAPID keys, `MaxRecordSize=4096`, and an over-size error. Therefore the Host can impose both `context.WithTimeout` and `http.Client.Timeout`. | **Selected for M6**, pinned at `v1.4.0` after the normal proxy-enabled dependency command. |
| [`marknefedov/go-webpush/v2`](https://github.com/marknefedov/go-webpush) | MIT, active upstream (pushed 2026-07-20), but GitHub API reports only 3 stars. | Its documented client API has context, custom `http.Client`, RFC 8188/8291/8292/8030, multi-record and receipt metadata. It is technically richer than V2 needs. | Not selected: considerably less adoption/maturity evidence for a security-sensitive first reference. Revisit only if the selected library cannot meet a tested protocol requirement. |
| [`akyoto/webpush-go`](https://github.com/akyoto/webpush-go) | A 2019 fork of the first library with 0 stars. | Older API has no context-aware send and is not an independent maintained alternative. | Rejected. |
| Hand-written RFC 8291/VAPID implementation | No upstream maintenance or interoperability corpus. | Would require owning ECDH, HKDF, AES-GCM, VAPID JWT, endpoint variants, HTTP status classification, and payload edge cases. | Rejected by the library-first rule. |

### Selected-library integration contract

M6 must use `webpush.SendNotificationWithContext`, a dedicated reusable
`http.Client{Timeout: 10 * time.Second}`, and a per-attempt context deadline no
longer than that client timeout. It must always close a non-nil response body.
The plugin translates only bounded Host input into the library's subscription
and `Options`; VAPID private material stays in SecretStore and never enters
generic delivery rows, audits, logs, APIs, or River arguments.

The standard requires `TTL` ([RFC 8030 section 5.2](https://www.rfc-editor.org/rfc/rfc8030#section-5.2)); VAPID requires correctly scoped `aud` and an `exp` no more than 24 hours ahead ([RFC 8292 section 2](https://www.rfc-editor.org/rfc/rfc8292#section-2)). M6 must set both through the selected library and classify endpoint responses without exposing the endpoint or subscription keys.

RFC 8291 does not require a push service to support more than 4096 encrypted
octets ([section 4](https://www.rfc-editor.org/rfc/rfc8291#section-4)). The
library's default record size is also 4096. Freeze the Host-to-provider display
payload at **3072 UTF-8 bytes after JSON serialization**. Reject a larger
payload before projection with a stable, redacted reason such as
`notification.payload_too_large`; do not rely on the library error as the
product limit. The V2 worker receives only the bounded standard display model,
never an arbitrary provider payload.

## SSE Decision

Do not add an SSE library. Fiber's installed `SendStreamWriter` wraps
fasthttp's body stream writer and supports the required buffered write/flush
model. A `net/http` SSE handler would require an adapter and would not improve
the authoritative recipient-revision protocol.

M5 handler requirements frozen by this survey:

- authenticate and bind the recipient before opening the stream;
- set `Content-Type: text/event-stream; charset=utf-8`,
  `Cache-Control: no-store, no-transform`, and `X-Accel-Buffering: no`;
- emit only an event id/revision refresh signal, not a notification payload;
- write and flush each event, heartbeat at a bounded interval, stop on flush
  error or application shutdown, and enforce the per-user/process connection
  limit outside plugin replacement authority;
- read `Last-Event-ID` and/or the explicit durable cursor only as a hint;
  PostgreSQL recipient revision remains authoritative and REST performs the
  refresh after a mismatch.

This is Fiber-native, standards-compliant SSE. The known cancellation behavior
above is an implementation test requirement, not a reason to introduce an
unneeded dependency.

## Host-Owned Minimal Service Worker Proof

### Exact boundary to implement in M6

| Boundary | Frozen requirement |
| --- | --- |
| Script URL | `/_sforum/notifications/sw.js`, shipped from the Host web application as `apps/web/public/_sforum/notifications/sw.js`; it is not an extension artifact, Page Registry template, or Route Registry contribution. |
| Registration | Host-owned settings/browser code calls `navigator.serviceWorker.register('/_sforum/notifications/sw.js', { scope: '/_sforum/notifications/', updateViaCache: 'none' })` after a user gesture and permission flow. |
| Scope | Exactly `/_sforum/notifications/`. Do **not** set `Service-Worker-Allowed: /`, and do not register root scope. The default scope equals the script directory; explicitly repeat that value as a regression-resistant contract. |
| Static-response headers | Nuxt exact `routeRules` for the script must set `Cache-Control: no-store, max-age=0, must-revalidate`, `Service-Worker-Allowed: /_sforum/notifications/`, `X-Content-Type-Options: nosniff`, and `Content-Security-Policy: default-src 'none'; script-src 'none'; connect-src 'none'; img-src 'none'; style-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'`. The existing Caddy site headers add HSTS, `X-Frame-Options: DENY`, `nosniff`, and `Referrer-Policy: strict-origin-when-cross-origin`. The settings document must permit the Host worker with `worker-src 'self'` whenever it sends a CSP. |
| Script capabilities | Register handlers only for `push` and `notificationclick`. No `fetch`, `message`, `sync`, cache API, `importScripts`, dynamic import, plugin script URL, arbitrary action buttons, arbitrary icon URL, or arbitrary click target. |
| Push model | Strictly parse a versioned, size-limited Host display envelope. Call `showNotification` with bounded title/body/tag and data fixed to the relative Host route `/notifications`; invalid/missing data is dropped. |
| Click model | Close the notification and focus an existing same-origin `/notifications` client or `clients.openWindow('/notifications')`. Never navigate to a plugin-provided or external URL. |
| Namespace ownership | M6 must add `/_sforum` to `apps/api/app/Support/Pages.ReservedPrefixes`, retain Nuxt's `isHostReservedPath` rejection, and add Page Registry plus proxy tests. A plugin must not be able to claim any worker script, scope, or registration page under the prefix. |

The Service Workers specification states that a script normally controls only
its directory path and that `Service-Worker-Allowed` can broaden that path
([origin restriction](https://www.w3.org/TR/service-workers/#origin-restriction)).
The proposed worker neither uses that broadening mechanism nor controls an
application document path. The Push API associates a subscription with a
specific service-worker registration and wakes that worker for delivery
([Push API](https://www.w3.org/TR/push-api/#dfn-associated-service-worker-registration));
therefore a root scope is not needed for standardized push/display/click
handling. This proves the architectural boundary, subject to the required
`ReservedPrefixes` fix and browser verification in M6.

## Generic Delivery Persistence Decision

Use a new additive **`notification_deliveries`** table for V2 external
`notification.channel` projections. Do not overload `mail_deliveries`:
its recipient email/template contract and existing Mail UI/API are intentionally
mail-owned, while a channel delivery needs recipient/type/channel/provider
artifact identity and does not necessarily have an email address or template.

The initial table contract is:

| Field or index | Frozen purpose |
| --- | --- |
| `id BIGSERIAL PRIMARY KEY` | Stable redacted admin/audit reference. |
| `notification_id BIGINT NOT NULL REFERENCES notifications(id) ON DELETE CASCADE` | The canonical inbox intent; user erasure cascades through the recipient-owned notification. |
| `recipient_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE` | Recipient-scoped inspection and erasure; must equal the notification recipient in the creation transaction. |
| `type_id TEXT NOT NULL`, `payload_version INTEGER NOT NULL` | Immutable projection identity without re-rendering a changed plugin type. |
| `channel TEXT NOT NULL`, `provider_extension_id TEXT NOT NULL`, `provider_artifact_digest TEXT NOT NULL` | Independent channel/provider selection and exact-artifact audit; use snapshots rather than a restrictive extension FK so historical failures survive lifecycle changes. |
| `target_ref TEXT NOT NULL` | Bounded opaque subscription/target reference, never endpoint/key/secret. |
| `idempotency_key TEXT NOT NULL UNIQUE` | One deterministic projection per `(recipient, notification intent, channel)` across transaction/River retries and provider retries. Artifact changes do not generate a second delivery. |
| `status`, `attempt_count`, `reason`, `error_summary`, timestamps | Same bounded lifecycle vocabulary/redaction model as `mail_deliveries`; terminal transitions are idempotent. |
| Indexes | `(recipient_user_id, created_at DESC, id DESC)`, `(status, created_at DESC, id DESC)`, and `(channel, provider_extension_id, status, created_at DESC, id DESC)`. |

The generic worker/job loads the canonical notification and the opaque target
reference, resolves the currently usable exact provider artifact, and writes a
redacted classification. It must not persist raw Web Push endpoint/key material
or provider secrets in the row or job arguments. Rows for unavailable,
disabled, revoked, drifted, or Safe-Mode providers transition to the documented
skipped/failed state rather than silently rerouting.

**Email compatibility rule:** `mail_deliveries` remains the durable authority
for email, including existing notification email and non-notification identity
mail. No historic mail row is copied, renamed, or deleted. M6's generic
delivery table begins with external channel providers such as `web_push`; a
future explicit adapter may link a newly created mail projection, but that is
not needed to establish the generic contract and must not create dual delivery
authorities. `/api/v1/admin/mail/deliveries`, `/api/v1/admin/mail/policy`,
Mail provider selection, and `mail.deliver` retain their API-LTS behavior.

## Sources And Commands

Read-only repository evidence commands completed successfully unless noted:

```text
git status --short
sed -n '1,420p' knowledge/plans/2026-07-27-notification-platform-v2.md
rg -n -i "service worker|service-worker|..." apps/api apps/web deploy scripts compose*.yaml
sed -n '1,260p' apps/api/app/Http/server.go
sed -n '1,140p' apps/web/nuxt.config.ts
sed -n '1,260p' deploy/caddy/Caddyfile
sed -n '1,120p' apps/api/database/migrations/202607110016_notifications_mail_deliveries.sql
sed -n '1,260p' apps/api/app/Models/Notifications/postgres_store.go
sed -n '1,180p' apps/api/app/Jobs/Notifications/deliver_mail.go
sed -n '1,160p' apps/web/server/routes/_sforum/assets/[...path].ts
sed -n '1,90p' apps/web/server/utils/pluginRouteProxy.ts
sed -n '1,260p' apps/api/app/Support/Pages/reserved.go
sed -n '1,260p' $GOMODCACHE/github.com/gofiber/fiber/v3@v3.0.0-rc.3/middleware/compress/compress.go
sed -n '1,162p' $GOMODCACHE/github.com/gofiber/fiber/v3@v3.0.0-rc.3/ctx.go
```

External source checks completed successfully:

```text
curl -fsSL https://api.github.com/repos/SherClockHolmes/webpush-go
curl -fsSL https://api.github.com/repos/SherClockHolmes/webpush-go/releases/latest
curl -fsSL https://raw.githubusercontent.com/SherClockHolmes/webpush-go/v1.4.0/webpush.go
curl -fsSL https://raw.githubusercontent.com/SherClockHolmes/webpush-go/v1.4.0/LICENSE
curl -fsSL https://api.github.com/repos/marknefedov/go-webpush
curl -fsSL https://raw.githubusercontent.com/marknefedov/go-webpush/master/README.md
curl -fsSL https://www.w3.org/TR/service-workers/
curl -fsSL https://www.w3.org/TR/push-api/
curl -fsSL https://www.rfc-editor.org/rfc/rfc8291.txt
curl -fsSL https://www.rfc-editor.org/rfc/rfc8292.txt
curl -fsSL https://www.rfc-editor.org/rfc/rfc8030.txt
curl -fsSL -o /dev/null -w 'webpush-repo=%{http_code}\\n' https://github.com/SherClockHolmes/webpush-go
curl -fsSL -o /dev/null -w 'webpush-doc=%{http_code}\\n' https://pkg.go.dev/github.com/SherClockHolmes/webpush-go
curl -fsSL -o /dev/null -w 'service-workers=%{http_code}\\n' https://www.w3.org/TR/service-workers/
curl -fsSL -o /dev/null -w 'push-api=%{http_code}\\n' https://www.w3.org/TR/push-api/
curl -fsSL -o /dev/null -w 'rfc8291=%{http_code}\\n' https://www.rfc-editor.org/rfc/rfc8291
git diff --check
```

One best-effort MDN request for the `Service-Worker-Allowed` page failed with a
TLS `SSL_ERROR_SYSCALL`; the normative W3C Service Workers specification and
MDN registration page both supplied the scope evidence used above. No dependency
download/install command, database inspection, application server, migration,
or test suite was run by this survey. The five explicit link checks all returned
HTTP 200, and `git diff --check` passed.

## M6 Verification Additions

Before declaring the reference provider complete, M6 must add focused tests for:

1. Page Registry and Nuxt proxy rejection of every `/_sforum/**` plugin claim.
2. The exact worker script response headers, content type, route ownership,
   cache behavior, and absence of any import/fetch handler.
3. Browser registration reports exactly `/_sforum/notifications/` scope; a root
   scope registration is rejected by the application contract.
4. Valid push displays only the bounded Host envelope; malformed, oversized,
   arbitrary URL/action, and external URL payloads cannot change behavior.
5. Notification click only focuses/opens same-origin `/notifications`.
6. Selected-library fake endpoint coverage: VAPID/TTL headers, context timeout,
   payload preflight, 2xx success, 404/410 subscription invalidation, 429/5xx
   retry classification, and no secret/key/endpoint in records or logs.
