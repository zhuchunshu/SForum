# V3 P0 Performance And Memory Baseline

Date: 2026-07-13
Git baseline: `d72c9ac2c5df9a635444175ed5c2f419eca9092d`

This is the migration baseline, not a release performance claim. It records the
current v1 paths before V3 snapshots and protocols change them.

## Environment

- macOS Darwin 23.6.0, x86_64
- Intel Core i7-9750H, benchmark parallelism suffix `-12`
- Go 1.26.3, Node 23.11.0, Bun 1.3.14
- API and Nuxt dev servers were already running; P0 did not restart them
- Three enabled plugin subprocesses: content-policy, storage-fs, and SMTP

## Reproducible Go Benchmarks

Command:

```bash
cd apps/api
go test ./app/Models/Extensions ./app/Support/Pages ./app/Support/Extensions \
  -run '^$' -bench 'V1Baseline' -benchmem -count=3
```

| Path | Median | Observed range | Bytes/op | Allocs/op | Coverage |
| --- | ---: | ---: | ---: | ---: | --- |
| Extension enable v1 | 26.769 us | 24.015-28.630 us | 2,384 | 17 | package/Manifest recheck, store transition, event, runtime decoration; no backend start |
| Theme resolve v1 | 328.3 ns | 293.5-335.1 ns | 0 | 0 | in-memory binding plus version/digest/contract match |
| Namespaced route proxy v1 | 694.333 us | 341.600-951.232 us | 15,116-15,159 | 45 | loopback HTTP; benchmark peer closes each response |
| Plugin RPC v1 health | 145.720 us | 128.970-158.224 us | 336 | 9 | real HashiCorp go-plugin subprocess and net/rpc call |

The route gateway currently constructs a new `fasthttp.HostClient` per proxy
call. A first unbounded run without a peer `Connection: close` reached loopback
timeouts after several thousand calls. P0 records this behavior without changing
production code; P6 must compare a pooled/stream-capable implementation against
both the stable numbers and this exhaustion failure mode.

## Live SSR And Process Sample

The existing development server returned the home page at 43,340 bytes. After
one cold/dev compilation request, 12 warm serial `curl` samples were:

- median 487.885 ms;
- range 271.967-584.707 ms;
- all HTTP 200.

The user's local database contained no topics, so P0 created a disposable
`sforum_v3_p0_benchmark` database, applied all migrations, seeded 3 topics/2
users/6 comments, and ran an isolated API on 18081 plus Nuxt dev on 13001. The
processes, database, temporary extension root, binaries, and Nuxt build directory
were removed after sampling; the existing 3000/8081 processes and database were
not modified.

For a canonical topic page (49,547 bytes), 12 warm serial samples were:

- median 303.779 ms;
- range 220.264-471.284 ms;
- all HTTP 200 with complete SSR title/body output.

Following the compatibility redirect from `/t/3` to the configured `id_slug`
canonical path produced a median 511.779 ms (range 448.759-771.419 ms) across
12 HTTP 200 final responses. The cold dev-compile request took 26.220 seconds
and is excluded from warm medians.

Observed resident memory after the sample (development processes, not production
steady state):

| Process | RSS |
| --- | ---: |
| Nuxt dev | 584,428 KiB |
| API | 44,364 KiB |
| content-policy plugin | 11,332 KiB |
| storage-fs plugin | 10,848 KiB |
| SMTP plugin | 10,616 KiB |

In the isolated seeded topology, API RSS was 43,540 KiB. The second Nuxt dev
compiler reached 2,307,248 KiB RSS and is recorded only as a warning about
parallel dev/HMR overhead, not as a production runtime target.

Dev HMR/compiler caches make Nuxt RSS and SSR latency noisy. P13 must reproduce
the same Go benchmarks and use an isolated seeded production-preview topology
for home/topic SSR, JavaScript-disabled output, and memory regression evidence.
