# 2026-07-15 Trusted Plugin And Theme Platform V3 P8 Handoff

## Changed

- P8 is **15/18 (83%)** after reconciling every authoritative row.
- `1c85d27b4` proves all 23 compiled catalog renders perform no filesystem I/O
  and provider resolution performs no request-path binding Store reads.
- `e125a47fc` gives legacy navigation islands safe empty props and disables
  unsafe shared SWR caching for query-paginated category/tag detail routes.
- `650e36802` extracts comment action presentation so the topic page remains
  below the repository's 1,000-line hard warning without moving permission
  ownership out of the route/API boundary.
- The authoritative checklist now marks the benchmark, compiler security
  matrix, crawler/no-JavaScript evidence, and hot-path task/test rows complete.

## Decisions

- Category/tag detail SWR stays disabled until its cache key includes normalized
  query and theme revision. The reproduced cache served a 6,004-topic payload
  beside current 50-topic SSR HTML, causing deterministic hydration mismatches.
- The Page ViewModel row was reopened. Defining 23 schemas is not sufficient
  while `BuildCorePageViewModel` leaves most page-specific product fields empty.
- Do not credit the restart row from backend convergence alone. It still needs
  one exact theme switch that survives API and Nitro restart plus concurrent
  activation under the V3 runtime.

## Verification

- `bun test` in `apps/web`: 310 passed.
- P8-focused Web superset: 111 passed; narrower touched-file slice: 27 passed.
- `bun run typecheck` and `bun run build`: passed.
- `go test ./app/Support/ThemeCompiler ./app/Support/Pages ./app/Http/Controllers/Pages`:
  passed.
- In-app Browser: category page 2 -> 3, topic/comments, profile, and plugin add
  page passed with correct canonical state, no overlay, and no relevant logs.
- JavaScript-disabled Playwright: home page 1 -> 2, category page 2 -> 3,
  topic/comments, profile topics, and plugin add page all rendered real content.
- Baiduspider SSR parse: five routes returned 200 with title, content, links,
  canonical, robots, five hreflang links, and valid JSON-LD; home/category
  exposed pagination anchors.

## Next

1. Populate every production Core Page ViewModel with its typed page-specific
   product data and test allowed plus denied actor projections.
2. Prove plugin Page ViewModel/business-data contracts remain unchanged through
   plugin templates and active-theme overrides.
3. Run the exact API/Nitro restart and concurrent theme activation exit test.

## Open Questions

- None. The three remaining rows have explicit implementation/test exits.
