# 2026-08-02 Browser Notification Subscription Fix

## Changed

- `GET /web-push/subscriptions` now returns active current-user devices only;
  revoked rows remain available to server-side audit and delivery relations.
- The Web UI reports enabled only when the live browser subscription matches an
  active Host subscription id.
- Untouched Core reply, mention, and moderation Web Push policies now use the
  enabled recommendation. `admin_test` remains disabled, operator-edited rows
  are preserved, and restore-default behavior uses the same recommendation.
- OpenAPI and bilingual notification documentation describe the corrected
  active-device and policy-default behavior.
- Notification SSE uses one Web Locks leader per authenticated user/origin and
  broadcasts recipient revisions to same-user tabs. Leader exit hands the
  connection to a waiting tab; unsupported browsers retain the previous
  per-tab connection and all tabs retain visible-page REST reconciliation.
- The destructive formal SEO ZIP lifecycle test now creates, migrates, and
  drops an isolated PostgreSQL database instead of publishing temporary
  executable fixture paths into the shared development database.

## Verification

- Full Web suite: 875 passed; Nuxt typecheck passed.
- Cross-tab regressions cover six same-user tabs with one EventSource, leader
  handoff, and different-user isolation.
- Notification controller/model tests and the real-PostgreSQL migration test
  passed; the migration is idempotent and preserves operator-edited rows.
- Full API `go test ./...`, the isolated formal ZIP lifecycle chain,
  architecture validation, and `git diff --check` passed.
- Runtime health and readiness are 200 through both API and Nuxt proxy.
  Authenticated Chrome shows permission `已允许` plus one active matching FCM
  subscription. With seven same-user pages open, Nuxt held one SSE upstream to
  the API and no browser console 429/error was observed.

## Decisions

- A revoked subscription is history, not a user-manageable device. Keep it in
  persistence for auditability but exclude it from the personal settings list.
- Browser permission alone is insufficient to claim Web Push is enabled; the
  Host must also recognize the exact active subscription.
- A previously granted browser permission does not prompt again. The settings
  badge is the authoritative permission signal; the device row is the
  authoritative subscription signal.
- Per-user Web Locks leadership is a client connection optimization. The API's
  bounded per-user/global connection limits remain authoritative safeguards.

## Next

- None for the browser-subscription and SSE 429 fixes.

## Open Questions

- None for the subscription/default-policy fix.
