# Decision: Compose blue/green updates

## Status

Accepted

## Context

The beginner deployment uses one Docker Compose host with managed PostgreSQL
and Redis. Replacing the directly published API and Web containers creates an
avoidable HTTP outage, but allowing two Workers or applying arbitrary database
migrations during a rolling switch would make job and schema behavior unsafe.

## Decision

Keep one stable Caddy edge and two application slots (`blue` and `green`). Start
and verify the standby API/Web before atomically reloading Caddy, then gracefully
stop the old Worker before starting the new one and retiring the old slot. Both
slots share the managed database, Redis, attachment, and extension volumes.

The target migrator must prove that both embedded Core migrations and River's
migration set exactly match the live database. Any pending or mismatched
migration refuses the zero-downtime path and directs the operator to
`deploy.sh`, which owns backup, migration, and the maintenance window.

Version input defaults to the newest public GitHub Release, including
prereleases, but `upgrade.sh` resolves it to an immutable `vX.Y.Z` tag, prints
the current and target tags, and asks for confirmation before pulling images.
Only `--yes` skips prompts; deployment state never stores `latest`.

## Consequences

- The first conversion from the legacy direct-port topology has a short outage.
- Later migration-free API/Web HTTP updates keep serving through the old slot
  until the candidate is healthy.
- Existing WebSocket connections may reconnect at the Caddy reload boundary.
- Queue consumption pauses during Worker drain/start, but durable jobs remain
  in River and two Worker versions never consume concurrently.
- This is single-host availability, not multi-host high availability; the edge,
  database, Redis, and Docker host remain shared failure domains.
