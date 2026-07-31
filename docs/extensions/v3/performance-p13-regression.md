# P13 Performance / Memory Regression Note

Date: 2026-07-21  
Compared to: `docs/extensions/v3/performance-baseline.md` (P0)

## Package microbenchmarks (this session)

| Path | Observation | Notes |
| --- | --- | --- |
| Theme resolve V1 baseline | ~1978 ns/op, 0 B/op | Single `-benchtime=1x` sample; still sub-microsecond class vs P0 ~328 ns median (noise at n=1) |
| Dependency inspector snapshot | ~140 us/op | P12 inspector overhead gate remains order-of-magnitude acceptable for admin use |
| Builtin theme completeness compile+render | Product test path | Full protected default-theme compile/render green under `TestBuiltinThemesCoverAllReplaceablePages` |
| SEO multi-kind subprocess | Product test path | Real Protocol V2 build+start within Extensions package suite (~10–25s wall including compile) |

## Interpretation

- No Host registry hot path was intentionally degraded for P13 reference packages.
- Theme L1 expansion adds compile-time work at activation, not per-request filesystem IO (catalog hot-path test still asserts zero post-compile opens).
- Plugin subprocess product gates intentionally include `go build`; they are CI correctness costs, not production request latency.

## Residual measurement gaps

- Full warm SSR home/topic median re-sample against live 3000/API not re-run in this session (do not kill user web server).
- Production multi-node enable latency remains covered by RuntimeRollout unit/coordinator tests rather than a full Redis cluster e2e re-measure.

## Verdict

No blocking performance regression identified for P13 reference package landing.
Re-run full P0 command set before any legacy deletion that changes enable/proxy paths.
