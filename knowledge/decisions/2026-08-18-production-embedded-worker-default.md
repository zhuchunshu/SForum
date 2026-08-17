# Decision: Embed The Production Worker By Default

## Status

Accepted

## Context

SForum previously embedded the River worker only in development. A normal
production Compose installation started separate API and Worker containers.
The split Worker created another Go runtime, PostgreSQL/Redis pools, extension
runtime, and plugin subprocess set. That isolation is valuable under sustained
background load, but its fixed memory cost is disproportionate for the small,
single-node self-hosted installations targeted by the recommended deployment.

The existing bootstrap already has explicit shared-resource ownership and
graceful shutdown coverage for an embedded Worker. Runtime observability also
represents this mode without inventing a separate Worker memory line.

## Decision

- `EMBED_WORKER_IN_API` defaults to `true` in every environment.
- Normal production Compose keeps `worker` behind the opt-in `split-worker`
  profile. `deploy.sh` activates that profile only when the flag is `false`.
- The API receives the worker concurrency and shutdown settings in embedded
  mode and shares its PostgreSQL pool and reconciled extension runtime.
- Standalone Worker images and binaries remain release artifacts for operators
  that need process isolation or independent scaling.
- Blue/green slot APIs explicitly set `EMBED_WORKER_IN_API=false`. A drainable
  standalone Worker remains necessary to avoid tying queue handoff to the HTTP
  slot lifetime.

## Consequences

- Recommended single-node production uses less memory and fewer database/plugin
  processes while retaining durable River semantics.
- API restarts and failures also pause the embedded Worker. Background CPU,
  memory, and database work can compete with HTTP requests.
- Operators with heavy queues, multiple API replicas, independent resource
  limits, or failure-isolation requirements should set
  `EMBED_WORKER_IN_API=false`; the deployment script then pulls, verifies, and
  starts the standalone Worker.
- Manually starting the standalone Worker while embedding is enabled is an
  invalid topology; deployment verification rejects it.
