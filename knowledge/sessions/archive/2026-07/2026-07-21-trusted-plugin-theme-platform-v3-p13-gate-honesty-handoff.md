# 2026-07-21 P13 Gate Honesty / Dead-Code Hygiene Handoff

## Changed

- Retargeted `tests/validate-moderation-workbenches.ts` to Host body islands after
  presentation ownership (`7ca7353e1`).
- Wired `validate-public-seo.ts`, `validate-seo-workbench.ts`, and moderation
  workbench into `scripts/test.sh` (`71d36a021`).
- Dropped false completeness requirement for dead chrome layouts/partials
  (`282bb6927`); deleted those stubs (`b5b087225`).
- Removed inert default theme Nuxt `layer/` tree (`12c0ef995`).
- Web presentation tests assert `<sf-navbar>` / `<sf-footer>` on non-auth shells
  (`ffbec31e6`).
- Clarified Agents.md: `test.sh` runs product validators it wires, not demos.

## Decisions

- Do **not** delete LoadTemplate / Protocol V1 / fail-closed SFPageOutlet until
  RemoveAfter ≈ 2026-11-28 + zero-shim telemetry + LTS checklist.
- Goal harness "remaining P10" is stale; P10 is complete.

## Next

- Wait for LTS window; keep optional operator re-evidence of full `./scripts/test.sh`
  when env free.
- No further implementable V3 product residual known after this hygiene pass.

## Open Questions

- None product-boundary; LTS deletion date remains policy-bound.
