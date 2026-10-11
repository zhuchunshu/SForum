# Decision: Plugins May Route Jobs To The `ai` Queue (Closed Set)

## Status

Accepted (2026-10-12). Implemented in the same change.

## Context

Every plugin job was enqueued to `QueueDefault`. A plugin doing slow, model-calling
work therefore competed with ordinary plugin work for the default workers and
ignored `JOB_QUEUE_AI_WORKERS`, which exists precisely because AI generation is
slow and hits vendor rate limits. The queue the Host already reserved for that
work was unreachable from a manifest.

## Decision

### 1. The manifest declares the lane; the vocabulary is closed

`jobs[].queue` accepts only `"default"` (or absent) and `"ai"`. The reason is
ownership, not ergonomics: `critical`, `mail`, `notifications`, `search`, and
`maintenance` carry Core's own delivery, and letting an extension write
`queue: "mail"` would let it decide how long a site waits for its mail. The
validator rejects anything outside the set, and the mapping function
(`jobs.QueueForPluginDeclaration`) collapses anything unexpected back to
`default`, so a hand-edited manifest cannot smuggle a job into a Core lane.

### 2. It is a lane choice, not a privilege

Queue selection changes nothing else: admission, exact-artifact contract checks,
retry policy, and the per-extension concurrency limit are untouched. Landing on
`ai` does not mean more parallelism — it means the job is capped by
`JOB_QUEUE_AI_WORKERS` (default 2) alongside Core's own AI jobs.

### 3. Both enqueue paths honour it, with different guarantees

- The versioned path (`EnqueueVersionedPluginJob`, used by the v2
  `jobs.enqueue` command and by plugin schedules) reads the queue from the exact
  manifest contract it already persists, so the choice is part of the frozen
  evidence of what was authorized.
- The legacy v1 path carries no contract, so the enqueuer resolves one
  best-effort through an injected `PluginJobContractSource` and falls back to
  `default` when the lookup fails. A queue lookup failure must not fail an
  enqueue that the v1 service already validated.

Scheduled plugin jobs enqueue through the versioned path, so a periodic
model-calling job inherits the declared lane instead of silently landing on the
default queue.

### 4. Two smaller housekeeping moves

- `manifestHasPermission` moved from `manifest.go` to `v3_validate_platform.go`:
  its only callers are validators, and the legacy file needed the room for the
  `queue` field. `manifest.go` is now at 916 lines, and its baseline was lowered
  920 → 916 in the same change.
- `Support/Extensions` flat-directory cap was raised 145 → 147 for the two files
  this workstream added (`protocol_v2_ai_tool.go`, `events_publish.go`).
  **Reduction condition:** when the provider-call transport family is moved into
  a `Support/ExtensionProtocol`-style subpackage, lower this cap again.

## Consequences

- A plugin that declares `queue: "ai"` no longer delays Core's or other plugins'
  ordinary jobs while waiting on a model.
- Operators who lower `JOB_QUEUE_AI_WORKERS` now apply that budget to plugin AI
  work too, which is the behaviour the configuration name promised.
- The declaration is part of the manifest, so the trust review sees it: an
  operator approving an artifact sees which lane its jobs may use.
