# 2026-07-11 Public Forum Hybrid Redesign

## Changed

- Completed the approved C / SForum Hybrid redesign for the shared public
  navbar, homepage, topic detail page, and recursive comment presentation.
- Homepage filters are URL-backed, SSR-first, debounce search input, reject
  stale responses, and preserve infinite scrolling. Query variants disable
  Nitro payload caching to avoid the root-route `EISDIR` cache collision.
- Topic detail now uses an unframed 820px reading column and 190px sticky
  progress rail. Focused heading, action, progress, stream-control, and report
  components reduce route-template concentration without moving policy.
- `SFComment` now exposes explicit tree/flat and depth contracts, uses one
  desktop branch inset, collapses deep descendants once, and visually flattens
  all depths on mobile without losing ancestry or reply context.
- Corrected the homepage after visual review to match the selected C demo:
  74px dark icon rail, pulse-card discussion stream, and 310px real-data
  activity dock. The earlier 208px two-column table interpretation was removed.
- A second side-by-side fidelity pass rendered the original demo and production
  at 1440x900. Production now uses the demo's exact canvas, border, text, and
  muted colors; 19px outer gutter; 360px command search; always-visible real
  category chips; API topic excerpts; two-cell dock stats; and a functional
  login-to-post fallback for guests.

## Decisions

- Keep complete tree data authoritative; responsive flattening is presentation
  only and must not rewrite stored relationships or API paging semantics.
- Use the existing Nuxt/Vue/Nuxt UI/Icon stack. No new dependency was needed.
- Mutation errors remain persistent. Successful user-triggered mutations use
  the existing theme-aware 10-second toast convention.

## Verification

- Browser QA passed at 1440x900, 1280x720, and 390x844 on `/t/5998` using the
  active default theme. Desktop columns measured 820px + 190px, the progress
  rail was sticky, and desktop descendants used only root/branch left edges.
- Mobile comments shared one 343px-wide column and document `scrollWidth`
  stayed equal to `clientWidth`. Disclosure changed from `aria-expanded=false`
  to `true`, exposed all descendants, and tree/flat switching worked.
- Focused frontend tests passed: 79 tests, 0 failures. Nuxt typecheck passed.
- `./scripts/test.sh` was blocked before frontend checks by unrelated working
  tree text `jixu` at `apps/api/app/Models/Extensions/service_test.go:758`.

## Next

- Re-run `./scripts/test.sh` after the unrelated extension-service test edit is
  completed or removed by its owner.
- Revisit category and tag listing pages later if they should adopt the same
  compact Hybrid row treatment.

## Open Questions

- None for the approved homepage, topic-detail, and comment scope.
