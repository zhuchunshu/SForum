# P6 Route Registry Performance Comparison

Date: 2026-07-15

This report compares the V3 selected Route Registry and Dispatcher path with a
same-run v1 namespaced proxy. It supplements the frozen P0 baseline; it does not
replace or reinterpret the original measurements.

## Environment

- macOS, amd64
- Intel Core i7-9750H, benchmark suffix `-12`
- Go 1.26.3
- loopback `httptest.Server`, persistent HTTP connections
- three serial samples per benchmark

No API, Nuxt, PostgreSQL, Redis, or plugin subprocess was required. The
benchmark constructs the complete 218-route Host catalog, an exact plugin
artifact, the Runtime Admission gate, and a loopback runtime in process.

## Reproduction

Run the same-run comparison:

```bash
cd apps/api
go test ./app/Http -run '^$' \
  -bench '^BenchmarkRouteDispatcherV3ProductionPath$' -benchmem -count=3
```

Run the original P0-compatible benchmark implementation separately:

```bash
cd apps/api
go test ./app/Support/Extensions -run '^$' \
  -bench '^BenchmarkRouteGatewayV1Baseline$' -benchmem -count=3
```

Run the allocation regression gate:

```bash
cd apps/api
go test ./app/Http -run '^TestRouteDispatcherV3AllocationBudgets$' -count=1
```

## Same-Run Results

| Path | Median | Range | Median bytes/op | Allocs/op |
| --- | ---: | ---: | ---: | ---: |
| v1 namespaced proxy, comparable fixture | 198.263 us | 192.860-370.041 us | 10,193 | 114 |
| V3 Dispatcher plan/Core bypass | 117.740 us | 68.255-167.857 us | 3,752 | 21 |
| V3 selected plugin HTTP | 606.905 us | 515.985-610.793 us | 33,932 | 352 |
| V3 six-step composed chain HTTP | 2.418 ms | 2.066-2.606 ms | 156,140 | 1,559 |

The comparable v1 and V3 selected cases use the same JSON request and response,
the same persistent loopback server behavior, and the same Runtime Admission
gate acquisition/release. Relative to that v1 case, the V3 selected route is:

- 3.061x latency (`+206.1%`);
- 3.329x allocated bytes (`+232.9%`);
- 3.088x allocation count (`+208.8%`).

The six-step chain executes global, before, filter, wrap, selected replace, and
after as six real HTTP runtime calls. It is reported independently and is not
presented as equivalent to the single-call v1 proxy.

The Core bypass row measures Route Registry resolution, execution-plan
construction, and the Dispatcher's decision to leave a Core-only request on the
existing Fiber path. It excludes Fiber middleware and the Core handler itself,
so it is not a complete Core request latency measurement.

## Covered V3 Layers

The selected V3 benchmark includes:

- the complete reviewed Core Route Catalog;
- immutable Registry resolution and execution-plan construction;
- explicit replace-provider resolution through `ProviderSelectionAPI`;
- exact runtime artifact inspection and Runtime Admission leasing;
- Host guard evaluation;
- exact request and response JSON Schema lookup and validation;
- the buffered HTTP adapter and Host-observed commit fence;
- bounded route trace publication;
- a persistent real loopback HTTP exchange.

The provider-selection store is an in-memory exact selection to keep the Go
benchmark deterministic. Production PostgreSQL selection latency is excluded,
so these results are a lower bound for the current selected-provider path. The
benchmark does not substitute a no-op invoker or schema validator for the key
runtime work.

## P0 Reference

P0 recorded the namespaced route proxy at a 694.333 us median, 15,116-15,159
bytes/op, and 45 allocations/op. That fixture deliberately forced the peer to
close every response and did not make all current comparison layers identical.
On this P6 worktree, the unchanged named P0 benchmark produced 621.316,
746.733, and 839.010 us in the final three-sample run: a 746.733 us median,
18,732 median bytes/op, and 123 allocations/op.

The V3 selected median is 12.6% below the frozen historical P0 median and 18.7%
below the final same-worktree P0 median. Neither comparison is accepted as proof
of an improvement because the connection and admission fixtures differ. The
same-run comparable result above is the authoritative P6 delta.

## Regression Gates And Findings

The ordinary test suite enforces allocation-count ceilings without asserting
wall-clock latency:

| Path | Observed allocs/op | Ceiling |
| --- | ---: | ---: |
| Dispatcher plan/Core bypass | 21 | 64 |
| Selected plugin HTTP | 352 | 480 |
| Six-step chain | 1,559 | 2,100 |

Wall-clock thresholds are deliberately absent from CI because shared runners
and loopback scheduling are noisy. The three-sample benchmark remains the
reviewed latency evidence.

The planning path now reads one internal immutable registry revision instead of
creating one or two caller-owned copies of the complete 218-route catalog per
request. Public Snapshot, Resolve, and execution-plan getters still return deep
copies. Mutation and concurrent-publication race tests cover that boundary.

The optimization removes the earlier full-catalog copy regression, but the
selected path still allocates about 33.9 KB versus 10.2 KB for the comparable v1
path and remains about 3.1 times slower. Production PostgreSQL provider-selection
latency is also still excluded. Provider selection should become part of a
read-optimized immutable runtime view rather than adding mutable-store work to
every selected request. P13 must repeat this benchmark and the
production-preview process/RSS sample; the current results are a P6 comparison,
not a final no-regression claim.
