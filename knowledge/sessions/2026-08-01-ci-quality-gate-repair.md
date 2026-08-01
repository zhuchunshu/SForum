# 2026-08-01 CI Quality Gate Repair

## Changed

- Fixed GitHub Actions run `30704611966`, job `91381454014`: the architecture
  gate rejected `SFNavbar.vue` after it grew from 967 to 1132 lines.
- Extracted the mobile search/compose row into `SFPublicMobileSearchBar` and
  the fixed home/post/notification dock into
  `SFPublicMobileBottomNavigation`. `SFNavbar.vue` is now 948 lines.
- Registered both public components in the reviewed V3 identity and generated
  component catalogs.
- Made `authRouteRendering.test.ts` independent from full-suite test order by
  installing its own `useSForumSeo` runtime stub, and updated stale reviewed
  route/UI catalog counts exposed after the architecture failure was removed.
- Added actionable file-size preflight rules to `AGENTS.md` and the bilingual
  testing guides: inspect handwritten files above 500 lines, do not cross 1000
  or grow a legacy cap, and run the architecture gate before commit/push.

## Decisions

- Mobile search and the fixed mobile dock are focused public-navigation
  components; `SFNavbar` remains the shared chrome orchestrator.
- Do not add or raise a legacy large-file baseline to admit ordinary feature
  growth. Extract a coherent responsibility instead.

## Verification

- Architecture validation passed: 1587 production files scanned.
- Focused navigation/notification regression: 42 passed, 0 failed.
- Full Web suite: 852 passed, 0 failed.
- Nuxt typecheck passed.
- `go build ./...` and the Nuxt production build passed.
- Exact focused CI Web slice: 36 passed, 0 failed.
- Full `./scripts/test.sh` passed outside the sandbox, including Go tests,
  required compatibility cells, OpenAPI, typecheck, product validators, and V3
  catalogs (338 routes, 285 UI surfaces, 99 traceability rows).
- In-app Browser passed at 1280x800 and 390x844: one navbar/footer, selected
  theme provider/template identity, no horizontal overflow, desktop mobile
  controls hidden, mobile search plus three-item dock visible, guest post link
  routed to `/login`, and no console warnings/errors.

## Next

- Commit and push the repair, then confirm the new `main` CI run succeeds.

## Open Questions

- None.
