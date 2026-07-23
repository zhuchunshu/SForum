# 2026-07-23 Theme-Defined System Error Pages Completion Handoff

## Changed

- Completed the broader system error page workstream after the public-resource
  404 precursor. `system.forbidden`, `system.not_found`,
  `system.rate_limited`, and `system.server_error` are virtual Page Registry
  surfaces for 403, 404, 429, and 500/502/503/504.
- Host owns normalized status, localized safe copy, actions/retry,
  `no-store`, `noindex,nofollow`, and the non-recursive Core emergency page.
  Themes own only selected-theme L0/L1 presentation.
- Added request-local system-error context, short single-attempt resolve,
  exact-artifact L0/L1 commitment, narrow Host islands, built-in default and
  Nocturne templates/styles, OpenAPI/admin virtual-page exposure, docs, and ADR.
- Plugin replacement of `system.*` is rejected, and public L2 widgets are
  rejected on system error templates.
- Final cleanup fixed default-theme mobile system-error layout so the hidden
  sidebar/right-rail state collapses to a single full-width column below
  960 px.
- Integration review restricted all system-error Host islands to `system.*`
  templates and made right-rail count formatting locale-stable across SSR and
  hydration.

## Decisions

- 401 remains login/redirect behavior, and API JSON error envelopes remain out
  of scope.
- 502/504 map to `system.server_error` even though current QA data does not
  expose natural full-page producers for those statuses.
- No test-only browser routes were committed for 429/5xx; those paths are
  covered by mapping, resolver, ViewModel, compiler/runtime, and fallback tests.

## Verification

- Focused web: `cd apps/web && bun test tests/errorPage.test.ts
  tests/pageOutlet.test.ts tests/presentationOwnershipRemaining.test.ts`
  passed (`70` tests, `443` assertions).
- Focused Go: `cd apps/api && go test ./app/Support/Pages
  ./app/Support/ThemeCompiler ./app/Models/PageViewModels
  ./app/Http/Controllers/Pages` passed.
- OpenAPI refs passed: `ruby scripts/validate-openapi-refs.rb` (`2165` refs
  across `54` files).
- `git diff --check` passed after final knowledge cleanup.
- Browser QA covered default-theme real unknown-route 404 on desktop/mobile,
  dark mode, English locale, selected-theme markers, original 404 status, no
  framework overlay, and recovery search navigation into the real home search.
  The mobile CSS source fix may require rebuilding/reactivating the immutable
  default theme artifact before an already-running QA DB serves the new digest.

## Next

1. Human review and merge of the existing worktree changes; no Git commit has
   been made.
2. If reviewers want fresh visual proof of the mobile CSS fix, rebuild or
   reactivate the default theme in the QA database so it serves a new immutable
   digest, then repeat the mobile 404 Browser check.

## Open Questions

- None for implementation. Future product routes that naturally emit full-page
  429/5xx can add browser fixtures without changing the current ownership
  model.
