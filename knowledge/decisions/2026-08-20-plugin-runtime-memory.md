# Decision: Plugin Runtime Memory Baseline

Date: 2026-08-20

## Status

Accepted

## Context

A Linux production site reported seven backend plugin processes using about
205 MiB PSS in total, normally 28-29 MiB per process and about 32 MiB for the
AWS SDK-backed S3 provider. Development screenshots used macOS RSS and were not
directly comparable. Release builds already used `-trimpath`,
`-buildvcs=false`, and `-ldflags="-s -w"`.

The Protocol V2 author SDK imported the legacy Host runtime package. A simple
SMTP plugin therefore compiled a 397-package non-standard dependency graph,
including Fiber, pgx, Goose, River, and bluemonday. Linux measurements also
showed transparent huge pages adding variable fixed anonymous PSS to each
small Go plugin process.

## Decision

- The public `sdk/plugin/v2` package owns plugin-side go-plugin startup, gRPC
  message limits, concurrency limits, deadlines, and the runtime-token metadata
  key. Host transport code depends on this SDK boundary; the SDK cannot import
  `app/Support/Extensions`.
- Keep `pluginv2.Serve`, `ServeOptions`, Bootstrap ABI v1, application Protocol
  V2, Host API V2, trust, authorization, isolation, and lifecycle behavior
  unchanged. The old Host names remain narrow compatibility adapters.
- Plugin-side SEO and provider wire projections use SDK-owned stable DTOs and
  constants. The SMTP implementation uses its own decoded mail request instead
  of a Host runtime type. The obsolete Protocol V1 content-policy adapter is
  removed.
- Every plugin child receives `GODEBUG=disablethp=1`. Host `GODEBUG` values are
  not inherited. This Go runtime compatibility setting affects the plugin heap
  on Linux and is a no-op on other platforms.
- Architecture validation rejects any future Protocol V2 SDK import of the
  legacy Host runtime. Built-in source-contract versions and exact package
  digests change with this runtime boundary.
- Storage provider sessions, payload DTOs, key validation, and the Protocol V2
  known-slot adapter live in the lightweight `sdk/plugin/storageprovider`
  package. The root SDK keeps a compatibility facade for existing authors;
  FS/S3 import only the lightweight package.

## Consequences

- Protocol V2 SDK non-standard dependencies fall from 396 to 164. SMTP falls
  from 397 to 165; the five pure Protocol V2 built-ins now build at about
  15-16 MiB on Linux/amd64.
- The stripped SMTP Linux binary falls from 23,011,490 to 15,618,210 bytes.
  In isolated Linux containers with `disablethp=1`, idle SMTP PSS falls from
  27,360 KiB to 19,284 KiB. Compared with the original approximately 30 MiB
  process, the combined measured reduction is about 10 MiB.
- The storage extraction reduces FS dependencies to 166 and S3 dependencies to
  256 including AWS SDK v2. Linux/amd64 stripped binaries fall from 22,589,602
  to 15,564,962 bytes for FS and from 31,088,802 to 24,948,898 bytes for S3.
- In matched isolated Linux samples, FS PSS falls from 26,112 to 19,072 KiB and
  S3 from 29,172 to 23,868 KiB, saving 12,344 KiB across the two processes.
- The expected seven-plugin production total is approximately 138-148 MiB,
  but acceptance requires a Linux release measurement after deployment and a
  representative warm workload.
- Go documents `disablethp` as temporary compatibility behavior. Before the
  runtime removes it, production Linux should use an explicit THP policy such
  as `madvise` and repeat the same PSS measurement.
