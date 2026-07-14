# P8 Theme Compiler Performance And Memory

Date: 2026-07-15

This report records the P8 small/large template compile and production render
benchmarks. It supplements the frozen P0 baseline and does not reinterpret the
P0 theme-provider lookup as a template compiler measurement.

## Environment

- macOS Darwin, amd64
- Intel Core i7-9750H, benchmark suffix `-12`
- Go 1.26.3
- three serial samples per benchmark

No API, Nuxt, database, Redis, extension process, or package-directory access
is required. Compilation reads a deterministic in-memory `fstest.MapFS` so the
result isolates compiler parsing, inspection, binding, and snapshot creation.

## Reproduction

Run the three-sample benchmark:

```bash
cd apps/api
go test ./app/Support/ThemeCompiler -run '^$' \
  -bench '^(BenchmarkCompileSmall|BenchmarkCompileLarge|BenchmarkRenderSmall|BenchmarkRenderLarge)$' \
  -benchmem -count=3
```

Run the allocation and memory regression gate:

```bash
cd apps/api
go test ./app/Support/ThemeCompiler \
  -run '^TestThemeCompilerAllocationBudgets$' -count=1
```

## Fixtures

- Small compile/render: one typed `forum.home` template, Host SEO title, and
  one declared Host home-page island.
- Large compile: 1,000 repeated static sections plus the same typed binding and
  island declaration.
- Large render: one immutable compiled snapshot and a sealed, exact-digest
  `HomePageViewModel` containing 1,000 topics.
- Render benchmarks use the production `Snapshot.Render` contract. They include
  exact Page ViewModel validation performed before timing, contextual template
  execution, bounded output, HTML segmentation, typed island descriptors, and
  sealed SEO output. They do not use the legacy/internal `renderPassive` path.

The timed render loop excludes ViewModel construction and binding because the
Host constructs and seals that request input independently. It includes all
snapshot execution and output processing performed for each render.

## Three-Sample Results

| Path | Samples | Median | Median bytes/op | Allocs/op |
| --- | --- | ---: | ---: | ---: |
| Compile small | 124.492, 133.591, 165.238 us | 133.591 us | 39,732 | 204 |
| Compile large | 29.037, 32.452, 38.753 ms | 32.452 ms | 2,406,247 | 23,063 |
| Render small | 58.834, 61.234, 62.294 us | 61.234 us | 12,227 | 67 |
| Render large | 14.015, 14.560, 16.190 ms | 14.560 ms | 2,173,164 | 22,850 |

Wall-clock values are evidence, not CI thresholds. Shared-runner load and Go
scheduler variance can move latency without changing implementation behavior.

## Regression Gates

The ordinary Go test suite runs each benchmark through `testing.Benchmark` and
fails when allocated bytes or allocation counts exceed these ceilings:

| Path | Highest observed bytes/op | Bytes/op ceiling | Highest observed allocs/op | Allocs/op ceiling |
| --- | ---: | ---: | ---: | ---: |
| Compile small | 39,733 | 49,152 | 204 | 224 |
| Compile large | 2,406,398 | 2,883,584 | 23,064 | 25,000 |
| Render small | 12,228 | 16,384 | 67 | 80 |
| Render large | 2,173,172 | 2,621,440 | 22,850 | 25,000 |

The ceilings retain limited Go-version and allocator headroom without masking
a new allocation class or a material growth in retained per-operation work.
They were set after the production-contract samples above; they must not be
raised merely to make a regression green.

## P0 Reference And Findings

P0 measured the old in-memory provider-binding resolution at 328.3 ns, 0 B/op,
and 0 allocations/op. That path only chose a provider by version, digest, and
contract. It did not read, inspect, parse, compile, execute, segment, or return
a theme template, so a direct P0-to-P8 multiplier would be invalid.

The large fixtures intentionally expose current scaling costs. Both compile and
render allocate roughly 2.2-2.4 MB for 1,000 sections/topics. The gates prevent
silent regression but do not establish that these figures are final release
targets. P13 must repeat all four benchmarks and gates, then measure theme
rendering inside the isolated seeded production-preview home/topic flows from
the P0 protocol. That final run must also include process RSS, JavaScript-
disabled SSR completeness, and the full Page Controller/ThemeRuntimeSnapshot
path; this compiler-only report is not a final no-regression claim.
