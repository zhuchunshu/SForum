# Decision: Runtime Memory Observability And Limits

## Status

Accepted

## Context

The admin dashboard previously mixed instantaneous process RSS with Go runtime
`MemStats.Sys`, polled the process table too often, and displayed an independent
Worker row even when the Worker was embedded in the API. Plugin replacement and
trust refreshes also caused large temporary allocations while hashing immutable
artifacts. Operators therefore saw high, unstable memory values without a clear
owner or measurement basis.

## Decision

- Use one cached `ps` frame per 5 seconds and a 60-second rolling median for API,
  independent Worker, directly owned backend plugins, and total RSS.
- Attribute plugin memory only to direct children of the API/Worker and expose
  per-extension process counts plus same-owner overlap diagnostics.
- Read Linux PSS from `/proc/<pid>/smaps_rollup` only when the complete family is
  available; omit PSS on platforms where it cannot be measured honestly.
- Treat an embedded Worker as part of the API process. The API row says it
  includes the Worker and the Worker row reports concurrency/running jobs, never
  a fabricated standalone MiB value.
- Keep pprof disabled by default and loopback-only when explicitly enabled. Use
  `GOMEMLIMIT` as an optional Go soft heap target, while plugin/container limits
  remain separate controls.
- Stream extension artifact digest hashing through a bounded buffer instead of
  loading the whole artifact into memory.

## Consequences

The dashboard is less sensitive to GC, page-in, and lifecycle replacement spikes,
and its totals are explainable across API/Worker/plugin ownership. RSS remains an
OS process metric rather than a promise of physical footprint; macOS values may
not match Activity Monitor's physical-footprint view. A 60-second median can hide
a very short spike, so pprof and host/container metrics remain the escalation
path for incident diagnosis. Built-in strip flags reduce artifact size but do
not change runtime limits or plugin lifecycle semantics.
