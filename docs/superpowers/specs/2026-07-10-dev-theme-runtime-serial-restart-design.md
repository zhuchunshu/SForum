# 2026-07-10 Dev Theme Runtime Serial Restart Design

## Context

SForum uses `theme-releases/current.json` as the shared theme-selection signal.
Production `runtime.mjs` switches between built Nitro artifacts, while local
`dev-theme-runtime.mjs` starts `nuxt dev` with the selected `layerPath` so theme
authors retain Vite HMR.

The local supervisor currently tries to apply the production blue-green model:
it starts a second `nuxt dev`, waits for health, switches the proxy, and then
stops the old process. An isolated reproduction on an alternate port showed
that Nuxt 4.4.8 rejects the candidate before it can become healthy:

```text
Another Nuxt dev server is already running
Set NUXT_IGNORE_LOCK=1 to bypass this check.
```

The candidate shares the active process's build directory. Even if the lock
were bypassed, two development instances would also compete for generated
files, Vite caches, and HMR ports. Parallel `nuxt dev` processes are therefore
not a sound local runtime boundary.

## Decision

Local theme switching may have a brief development-only outage. Replace local
blue-green switching with one serial `nuxt dev` lifecycle:

1. Stop the active process group.
2. Wait until every process in that group has exited and Nuxt has released its
   lock and HMR port.
3. Start one new process using the latest `current.json` selection.
4. Restore proxy traffic after the new process passes its health check.

Production keeps its existing Nitro blue-green behavior. This decision does
not weaken production availability and avoids adding isolated build slots and
HMR-port allocation solely for local development.

## Runtime Model

The development runtime tracks explicit selection and lifecycle state:

- `activeSelection`: the selection served by the healthy child.
- `desiredSelection`: the latest value read from `current.json`.
- `child`: the currently owned child process, if any.
- `restarting`: whether the serial restart loop is running.
- `restartRequested`: whether a newer selection arrived during a restart.
- `shuttingDown`: prevents crash recovery during supervisor shutdown.

A selection is an object with `mode` and resolved `layerPath`; the default
theme is represented explicitly instead of overloading `null`. Selection
comparison uses a stable key derived from both fields. This prevents a pending
switch back to the default theme from being mistaken for "no pending work."

## Restart Flow

`fs.watch` continues watching the release root so atomic replacement of
`current.json` remains visible. The callback filters unrelated filenames and
debounces matching events. Each accepted event updates `desiredSelection` and
sets `restartRequested`.

The restart coordinator processes one selection at a time:

1. Snapshot the latest desired selection.
2. Return when that selection is already healthy and no newer request exists.
3. Clear the proxy target so requests during the accepted restart window fail
   explicitly instead of being sent to a stopped upstream.
4. Signal the owned process group with `SIGTERM` and wait for the group to
   disappear. After a bounded grace period, use `SIGKILL` and wait again.
5. Spawn `bun run dev:plain` with `PORT=0` and the snapshot's
   `SFORUM_THEME_LAYER` value.
6. Parse the child listening address and wait for the existing TCP health
   check.
7. Set the proxy target and publish the child as active only after health
   succeeds.
8. If `current.json` changed while the child was starting, repeat the serial
   loop with the newest selection.

Only the callback belonging to the currently owned child may clear `child` and
`activeSelection`. An exit from an older or intentionally stopped process is
ignored for active-state mutation. Unexpected active-child exits clear the
proxy target and schedule crash recovery from the latest selection.

## Failure Handling

Before a runtime switch, the coordinator retains the last healthy selection.
If the requested selection fails to start or fails health checking, it starts
that last healthy selection again. This preserves the current production
semantic that an invalid candidate must not leave the operator without the
previously working theme, while accepting the local restart interval.

Initial startup has no last healthy selection and remains fail-fast. If both a
requested runtime switch and its last-known-good recovery fail, the supervisor
logs both errors, closes the proxy, stops owned process groups, and exits
non-zero instead of remaining alive in an ambiguous state.

Rapid changes are coalesced to the latest desired selection. A candidate that
became obsolete while starting is not treated as convergence; the coordinator
immediately performs the next serial restart.

## Module Boundaries

- `apps/web/scripts/dev-theme-runtime.mjs` remains the executable entrypoint
  and owns environment resolution, file watching, process signals, and logs.
- A focused development lifecycle module owns explicit selection state,
  stop-and-wait behavior, restart serialization, rollback, and identity-safe
  exit handling. Its process start/stop and proxy operations are injected so
  concurrency behavior can be tested without starting Nuxt.
- `apps/web/scripts/theme-proxy.mjs` remains the shared HTTP/WebSocket proxy and
  health-check implementation.
- `apps/web/scripts/runtime.mjs` remains the production Nitro blue-green
  supervisor and is not converted to serial restart.

## Testing

Implementation follows test-first development. Automated coverage must prove:

- The active process group is fully stopped before the replacement is spawned.
- An uploaded-theme switch followed quickly by a default-theme switch
  converges on the default selection.
- A delayed exit from the old child cannot clear the replacement child.
- Multiple watch events for the same selection do not create duplicate child
  processes.
- A failed requested selection restarts the last healthy selection.
- An unexpected active-child crash schedules one recovery, while intentional
  shutdown does not.
- Initial startup still waits for health before the public proxy listens.
- Existing HTTP forwarding and Vite HMR WebSocket proxy tests remain green.

An isolated real-Nuxt smoke test uses an alternate public port and temporary
release root. It must switch from the default theme to an uploaded layer and
back without `Another Nuxt dev server is already running`, while the user's
port 3000 process remains untouched.

## Documentation

Update the frontend and extension knowledge notes to state that production
theme activation is blue-green, while local development intentionally performs
an automatic serial Nuxt restart. Add a session handoff with the reproduced
failure, the compatibility reason, and verification commands.

## Non-Goals

- No parallel local Nuxt build slots.
- No `NUXT_IGNORE_LOCK` bypass.
- No change to the `current.json` backend contract.
- No change to production theme build or activation jobs.
- No promise that local requests remain available during a theme switch.
- No changes to ordinary source-file Vite HMR while the active dev child is
  running.
