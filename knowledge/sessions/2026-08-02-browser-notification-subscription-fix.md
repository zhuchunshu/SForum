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

## Verification

- Focused notification Web tests: 8 passed.
- Notification controller/model tests and the real-PostgreSQL migration test
  passed; the migration is idempotent and preserves operator-edited rows.
- OpenAPI references and `git diff --check` passed.
- The architecture gate currently reports only concurrent growth in
  `Options/service.go` (line and receiver-method legacy caps), outside this
  notification change.

## Decisions

- A revoked subscription is history, not a user-manageable device. Keep it in
  persistence for auditability but exclude it from the personal settings list.
- Browser permission alone is insufficient to claim Web Push is enabled; the
  Host must also recognize the exact active subscription.

## Next

- Recheck the provider-unavailable alert after the concurrent email-
  verification redirect/API runtime work is stable. Earlier runtime evidence
  confirmed the selected exact artifact, VAPID settings, secret, and plugin
  process; no executable extension activation was performed.

## Open Questions

- None for the subscription/default-policy fix.
