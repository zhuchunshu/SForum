# Decision: API Owns The Background Worker

## Status

Accepted

## Context

The optional standalone Worker duplicated the Go runtime, database and Redis
pools, extension runtime, and every enabled backend plugin process. More
importantly, the API settings surface used SettingsLifecycle and SecretStore
while the standalone Worker runtime could fall back to legacy
`extension_settings`. Production SMTP probes therefore succeeded with the new
credential while `mail.deliver` used stale credentials and failed AUTH.

Keeping a configurable ownership split makes one operator setting capable of
creating two runtime authorities for the same plugin configuration. That is a
larger integrity cost than the isolation benefit for SForum's supported
single-node deployment.

## Decision

- The API process always creates and owns the River Worker outside Safe Mode
  and host recovery mode. Worker ownership is not environment-configurable.
- Development, normal production, and blue/green Compose files expose no
  standalone Worker service or profile.
- Releases no longer publish a standalone Worker image or include its binary in
  the backend runtime bundle.
- API and background jobs share the PostgreSQL pool, Redis clients,
  SettingsLifecycle, SecretStore, and reconciled extension runtime.
- `deploy.sh` and `upgrade.sh` ignore retained `EMBED_WORKER_IN_API` values and
  remove legacy `worker`, `worker-blue`, and `worker-green` containers by exact
  Compose project/service labels before completing an update. Project-name
  resolution follows Compose environment precedence.
- Database restore tooling stops and restarts only the API; it never revives a
  retained legacy Worker container.
- The standalone binary and internal constructor may remain as compatibility
  and test scaffolding, but they are not a supported deployment entrypoint.

## Consequences

- SMTP and other provider jobs observe the same settings and secret authority
  as admin probes.
- Single-node installations remove the duplicate Worker and plugin-process
  memory cost.
- API downtime also pauses queue consumption; River preserves queued work.
- During a blue/green update, candidate and active APIs may briefly both run
  consumers. River owns job locking, and the old consumer stops with its API.
- Independent Worker scaling and process isolation are no longer supported
  deployment modes.
