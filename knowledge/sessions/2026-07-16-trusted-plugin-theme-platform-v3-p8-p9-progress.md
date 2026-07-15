# 2026-07-16 Trusted Plugin And Theme Platform V3 P8-P9 Handoff

## Progress

- Overall weighted progress is **58%** after P8 reached **18/18 (100%)**.
- P9 is **2/16 (12%) accepted** after the package-local public L2 production
  exit. Several
  implementation slices are ahead of the accepted percentage and must not be
  credited as complete merely because unit tests pass.
- Current branch is `main`; do not push, create a feature branch, or stage the
  unrelated dirty files listed below.

## Changed

- `d10cc2093` registers the package-local public frontend route in the reviewed
  230-route catalog; the 121-UI/99-trace-row catalog gate passes.
- `d64e9177b` publishes Component Registry graphs atomically across restore,
  Safe Mode, upgrade, rollback, disable, exact package identity, required
  cross-plugin targets, and concurrent readers. Host component identities are
  domain-separated by `sforum.component.package-publication@1`.
- `57d1c2958` creates and retains the shared production Component Registry in
  bootstrap.
- `867bdc6c3` adds the exact-artifact trust dialog for executable/L2 theme
  activation. Ordinary L0/L1 themes remain operator-buildless and issue no
  challenge. Stale/replayed/expired failures discard the token and refresh both
  the trust document and activation preview tuple.
- `92969a5f8` adds the public L2 browser runtime: Host-issued descriptor parsing,
  same-origin immutable package URLs, byte verification, native ESM and linked
  CSS with relative-resource bases, exact request headers, lease revalidation,
  session quarantine, SSR fallback, reference-counted CSS cleanup, and bounded
  descriptor/import/mount/unmount timeouts.
- `b081898c5`, `d8d6d5205`, and `bf49c2aa9` bind generated L2 entries to exact
  target contracts and exact Asset/Component Registry publications.
- `b164b30bd`, `c0e2fa855`, and `86d112ef5` preserve emergency L1 output, add a
  buildless author fixture, and commit the isolated production upload-to-revoke
  E2E harness.

## Verification

- P8 isolated production API + Nitro evidence passed all **18/18** rows: three
  restarts, exact theme switch, concurrent single winner, final recovery,
  JavaScript-disabled catalog output, and no residual processes or port 3000
  interference.
- Component Registry: full `Support/Extensions`, focused race, bootstrap,
  `go vet ./...`, and `go build ./...` passed.
- Web: admin focused **21/21**, Public L2 focused **53/53**, full Web **353/353**,
  Nuxt typecheck, offline Page Registry validation, and production Nuxt/Nitro
  build passed. Only pre-existing build warnings remain.
- The public L2 production exit passed in **238.30 seconds**: fresh API/Nitro
  artifacts, inert ZIP upload, actor/artifact-bound trust, activation without
  API restart, SSR fallback, Chromium native ESM plus relative chunk and nested
  CSS, interaction, API/Nitro restart, revoke-to-404, DOM/CSS cleanup, and no
  residual test processes. The fixture contains no `package.json` and runs no
  package-local build step.

## Active Work

- Asset Registry authoritative lifecycle/bootstrap publication is under
  controlled Grok 4.5 implementation and main-thread review. Public requests
  must stop rebuilding the graph through `Store.List`.
- Navigation/Region registry foundations are committed, but production mapping
  is paused at explicit target-contract, placement, region-rendering, scope, and
  Nitro cache-invalidation product boundaries rather than inventing semantics.
- Public L2 remains production-default off until scoped SSR CSP and the
  remaining P9 composition/inspection boundaries pass.

## Known Gaps

- Asset CSP declarations are validated and returned but are not yet aggregated
  into Nuxt SSR response headers. CSP scope/conflict policy remains a separate
  production boundary; an empty fixture declaration does not close it.
- Asset Registry still needs authoritative lifecycle/bootstrap publication.
- Navigation/Region still needs production manifest mapping, lifecycle restore,
  SSR ViewModel consumption, inspector UI, and exact target-contract evolution.
- P9 template fragments, plugin theme overrides, inspectors, and high-traffic
  desktop/mobile browser checks remain open.

## Dirty Ownership

- Preserve and do not stage `apps/api/app/Models/PageViewModels/source_test.go`
  and `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`
  with V3 commits. Local Playwright artifacts are ignored by `.gitignore`.

## Next

1. Review and land Asset Registry lifecycle/bootstrap publication without
   request-time `Store.List` rebuilds.
2. Implement strict, page-scoped CSP aggregation and Nitro SSR headers.
3. Resolve Navigation/Region product contracts before production mapping.
4. Implement SSR fragments, theme plugin overrides, inspectors, and responsive
   browser exits.
