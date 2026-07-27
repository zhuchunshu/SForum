# Plugin Restart Is A Host-Owned Lifecycle Operation

Date: 2026-07-28
Status: Accepted

## Context

The admin UI previously implemented plugin restart by calling the enable
endpoint. That was only safe while the active artifact remained the lifecycle
authority. A protected built-in could have a legacy active artifact and a
staged Lifecycle V2 artifact after source synchronization. Reusing enable then
started the legacy process without first retiring its durable registry
publication, causing an exact-fence conflict and compensating the plugin to
disabled.

The resulting `extension.trust_not_required` response was also misleading:
protected built-ins intentionally do not require the uploaded-artifact trust
grant, and the V3 migration gate must not turn that normal state into a 409.

## Decision

- Restart uses `POST /admin/extensions/{id}/restart` with a stable
  `Idempotency-Key`.
- The Host preflights the exact target, dependencies, package identity,
  features, trust, and required capability confirmation before stopping the
  current runtime.
- Restart fully disables the source before enabling the target so registry
  publications are quarantined through their normal lifecycle boundaries.
- When an enabled legacy artifact has a staged Lifecycle V2 candidate, the
  Host disables the source, promotes the exact staged tuple through CAS while
  inert, then enables it through Lifecycle V2.
- Existing Lifecycle V2 staged updates continue through the native upgrade
  ledger.
- Trust preview and challenge endpoints accept `target=staged` so restart
  recovery can bind the exact candidate even after a failed attempt has left
  the plugin disabled. Without that explicit target, disabled-plugin trust
  review continues to bind the active artifact.
- Capability confirmation is resolved before downtime for legacy bridges,
  Lifecycle V2 upgrades, and disabled recovery. Uploaded executable targets
  use their exact trust impact; built-ins and other non-trust-required targets
  still require explicit capability confirmation when the target changes.
- Restart phase keys are deterministic derivatives of the caller key. A
  failure after disable remains disabled; retrying the same request resumes
  the exact committed target.
- Trust status returns `200` with `trustRequired=false` for built-ins and
  declarative artifacts. The migration-gate 409 remains only for uploaded
  executable artifacts that actually require exact trust.

## Consequences

- Restart can involve bounded downtime and fails closed rather than restoring
  a partially published runtime.
- The UI must use the dedicated endpoint and review staged target
  capabilities before invoking it. A disabled plugin with an exact staged
  artifact remains restartable so operators can recover a failed bridge.
- Lifecycle and audit history distinguish restart from enable, while the
  actual disable/enable phases remain inspectable.
- Recovery never rewrites immutable package or Identity Registry history.
