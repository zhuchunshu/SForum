# 2026-07-28 Architecture Boundary Guardrails Handoff

## Changed

- Added mandatory frontend/backend placement and module-boundary rules to
  `AGENTS.md`.
- Added `tests/validate-architecture-boundaries.mjs` plus a current-debt
  baseline for large files, crowded roots/packages, God-object receiver counts,
  and inline fixed admin tabs.
- Wired the validator into `scripts/test.sh` before the full Go test suite.
- Recorded the accepted baseline strategy in
  `knowledge/decisions/2026-07-28-architecture-boundary-guardrails.md`.

## Decisions

- Existing debt is frozen at current values and may only shrink; every
  reduction must lower/remove its baseline in the same change.
- New fixed Core tabs require domain `tabs/` components; dynamic provider and
  extension tabs keep generic runtime renderers.
- Go package extraction follows stable responsibility/dependency boundaries,
  not cosmetic subdirectories.

## Next

- Decompose site settings, forum settings, SEO, personalization, attachments,
  and mail tabs without raising their baselines.
- Split `bootstrap/api_assembly.go` and introduce focused collaborators behind
  the extension `Service`/runtime `Manager` facades.

## Open Questions

- None for the guardrail itself; each large-package extraction still requires a
  scoped dependency and transaction-boundary review.
