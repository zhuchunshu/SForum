# 2026-07-16 Trusted Plugin And Theme Platform V3 P8-P9 Handoff

## Progress

- Overall weighted progress is **58%** after P8 reached **18/18 (100%)**.
- P9 remains **1/16 (6%) accepted** until its production exits pass. Several
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

## Active Work

- `apps/api/app/Support/NavigationRegistry/` is an uncommitted P9 foundation
  under main-thread and Grok 4.5 review. Do not stage until exact-artifact
  removal, namespace, ordering, and dependency semantics are accepted.
- `extensions/fixtures/themes/sforum-public-l2-e2e-theme/` plus
  `apps/api/bootstrap/public_l2_*_e2e_test.go` are an uncommitted isolated
  upload/trust/activate/mount/restart/revoke production fixture and harness.
- Public L2 remains production-default off until that E2E and the remaining P9
  boundaries pass.

## Known Gaps

- `PublicComponent` must consume the active Component Registry plan so a stale,
  hidden, losing, or dependency-invalid contribution cannot receive a browser
  descriptor merely because its manifest still declares an L2 entry.
- Asset CSP declarations are validated and returned but are not yet aggregated
  into Nuxt SSR response headers. CSP scope/conflict policy remains a separate
  production boundary; an empty fixture declaration does not close it.
- Asset Registry still needs authoritative lifecycle/bootstrap publication.
- Navigation/Region still needs production manifest mapping, lifecycle restore,
  SSR ViewModel consumption, inspector UI, and exact target-contract evolution.
- P9 template fragments, plugin theme overrides, inspectors, and high-traffic
  desktop/mobile browser checks remain open.

## Dirty Ownership

- Preserve and do not stage `apps/api/app/Models/PageViewModels/source_test.go`,
  `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`,
  `.playwright-cli/`, and `.playwright-p8-nojs.json` with V3 commits.

## Next

1. Finish Navigation/Region review and commit only the self-contained registry.
2. Run and review the isolated public L2 upload-to-revoke production E2E.
3. Bind Component Registry winner/admission checks to public descriptors.
4. Implement scoped CSP aggregation and Asset Registry lifecycle publication.
