# Notification Host API reference fixture

This Protocol V2 fixture declares one inert, namespaced notification type and
uses the Go SDK `EmitNotification` helper from an observe hook. The Host owns
exact-artifact admission, payload schema validation, recipient eligibility,
policy, idempotency, rate limits, persistence, and redacted audit.

The package is test-only. Build `backend/plugin`, replace the two digest
placeholders in `sforum.extension.json.tmpl`, and install the resulting exact
artifact through the normal extension trust flow.
