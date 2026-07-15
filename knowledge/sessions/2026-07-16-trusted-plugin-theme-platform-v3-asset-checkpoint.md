# 2026-07-16 Trusted Plugin And Theme Platform V3 Asset Checkpoint

## Progress

- Overall weighted V3 progress is **62%** after flooring.
- P6 remains **13/18 (72%)**. The additional route matrix strengthens already
  implemented actions and failure fences but does not close the complete P6
  action/failure row.
- P7 advances from **13/22 to 14/22 (64%)**. Commit `e92016366` closes the
  six-family SDK/catalog row with typed callable surfaces and an explicit
  Host-owned schedule declaration boundary. The unconnected `QueryRegistry` is
  still not production evidence.
- P8 remains **18/18 (100%)**.
- P9 advances from **2/16 to 4/16 (25%)** by accepting the complete Asset
  Registry row and the exact frontend-safety/rollback row.

Exact earned-weight formula:

```text
P0-P5                 = 39.0000
P6  10 * 13 / 18      =  7.2222
P7  10 * 14 / 22      =  6.3636
P8                     =  8.0000
P9   8 *  4 / 16      =  2.0000
P10-P13                =  0.0000
total                  = 62.5859 -> displayed 62%
```

The checked P12 Safe Mode fixture remains cross-phase evidence and does not earn
P12 weight before that phase has a production exit.

## Authoritative Counts

The generated traceability matrix still contains exactly **99** one-to-one
targets: **27 theme rows + 72 plugin rows**. It has no missing phase mapping.

| Phase | 99-row targets |
| --- | ---: |
| P0 | 1 |
| P1 | 4 |
| P2 | 1 |
| P3 | 1 |
| P4 | 4 |
| P5 | 6 |
| P6 | 13 |
| P7 | 12 |
| P8 | 19 |
| P9 | 12 |
| P10 | 8 |
| P11 | 9 |
| P12 | 7 |
| P13 | 2 |
| **Total** | **99** |

Task/test acceptance rows after this checkpoint are:

| Phase | Accepted | Total | Open |
| --- | ---: | ---: | ---: |
| P0 | 18 | 18 | 0 |
| P1 | 17 | 17 | 0 |
| P2 | 19 | 19 | 0 |
| P3 | 13 | 13 | 0 |
| P4 | 15 | 15 | 0 |
| P5 | 17 | 17 | 0 |
| P6 | 13 | 18 | 5 |
| P7 | 14 | 22 | 8 |
| P8 | 18 | 18 | 0 |
| P9 | 4 | 16 | 12 |
| P10 | 0 | 15 | 15 |
| P11 | 0 | 16 | 16 |
| P12 | 1 | 22 | 21 |
| P13 | 0 | 49 | 49 |

The 99 comparison rows and phase task rows are not one-to-one completion
counters. One task can prove several comparison rows, while a comparison row
may require multiple production exits. Do not publish an invented `x/99`
completion number; the fixed weighted phase formula above remains authoritative.

## Asset Exit Evidence

The following committed chain closes the two P9 rows:

- `2a8c0d3e6` introduced the immutable Asset Registry.
- `d8d6d5205` and `55063b1a3` added bounded graph validation, exact ownership,
  revision/artifact CAS, deterministic dependency planning, quarantine closure,
  immutable snapshots, race fences, and cleanup.
- `cf5636927` binds frontend descriptors and bytes to live exact artifact trust,
  stable package file descriptors, immutable digests, dependency-owner trust,
  and request-path Registry reads without `Store.List` rebuilds.
- `44cfb67dc` binds plugin and theme enable/disable/uninstall/rollback to exact
  Asset publications, one-theme graphs, trust-aware compensation, and stale
  writer fencing.
- `f5ed19d2c` publishes the graph through durable lifecycle registry plans,
  startup/Safe Mode reconciliation, and one shared Registry that is late-bound
  to Service, trust, and frontend consumers only after authoritative restore.
- `86d112ef5` remains the production browser proof for inert ZIP upload, exact
  trust, native relative ESM/CSS, SSR fallback, restart, revoke, unmount, and
  complete stylesheet cleanup without an operator build.

Adjacent accepted evidence does not inflate P6/P9 accounting:

- `e92016366` closes only the P7 generated family SDK/catalog row. Callable
  hooks/services/providers/jobs/commands have typed registries or clients;
  schedules intentionally remain Host-owned Manifest declarations. The
  generated wire client is documented as unregistered, and no fake List/Trigger
  helper was added.
- `4c2911122` strengthens action ordering, replacement selection, Safe Mode,
  fallback fencing, cancellation, and timeout tests. It does not close the full
  P6 matrix because locale/query/body/policy/custom-guard/crash and open product
  semantics remain missing.

Accepted Asset behavior covers handles, contract versions, dependencies,
module/loading mode, integrity, CSP declarations, page/component scope,
deduplication, exact cleanup, byte drift, symlink/FIFO safety, dependency-owner
revocation, concurrent publication, restart, Safe Mode, and rollback conflict.

Verification on the converged tree passed:

- `go test ./app/Models/Extensions -count=1`
- `go test -race ./app/Models/Extensions -count=1`
- `go test ./app/Support/Extensions ./bootstrap -count=1`
- `go test -race ./app/Support/Extensions ./bootstrap -count=1`
- `go vet ./app/Models/Extensions ./app/Support/Extensions ./bootstrap`
- `go build ./...`
- `git diff --check`

Page-scoped CSP aggregation into Nuxt SSR response headers is still open. The
Registry validates and returns CSP declarations, but that is not equivalent to
enforcing a response policy. Public L2 therefore remains production-default off.

## Fourteen Final Boundaries

The ADR still contains exactly fourteen accepted boundaries. Current evidence
must be read as phase progress, not as a claim that the program is complete.

| # | Boundary | Current evidence |
| ---: | --- | --- |
| 1 | Complete Route Registry action set | **Partial:** P6 action/SEO/full failure matrix remains open. |
| 2 | Inherited and trusted custom guards | **Partial:** inherited/custom foundations exist; raw request/session policy and remaining resource guards are open. |
| 3 | Trusted raw core database access | **Accepted P5:** exact grants, physical ACLs, compatibility blocking, and production tests are complete. |
| 4 | Public/admin component replacement | **Partial:** stable identities and immutable graphs exist; complete composition/render exits remain P9. |
| 5 | Plugin-defined hooks/components/services/providers | **Partial:** hooks, services, and providers are accepted; component extension remains P9. |
| 6 | Per-module Extension Surface Matrix | **Accepted P0 governance:** CI catalogs exist; P13 must still rerun the final five-plugin matrix. |
| 7 | Admin/Query/Identity/Media/Navigation registries | **Partial:** Admin is accepted; Query is unconnected and Identity, Media, and production Navigation remain open. |
| 8 | Plugins declare, Host assigns permissions | **Partial:** manifest policy is frozen, but P7 role-mapping approval and no-implicit-grant reference evidence remain open. |
| 9 | Theme overrides preserve plugin business contracts | **Partial:** Page ViewModel preservation is accepted in P8; complete plugin template/component override exits remain P9. |
| 10 | Primary SEO content is SSR | **Partial:** P8 and public L2 fallback prove current pages; P9/P11/P13 component and SEO reference exits remain open. |
| 11 | Plugin uninstall hooks lead cleanup | **Accepted P4:** resumable uninstall plans/hooks and Host fallback cleanup are production-bound. |
| 12 | Executable extensions are honestly full-trust | **Accepted P1:** inert install plus exact actor/artifact confirmation is production-default. |
| 13 | Safe Mode, CLI disable, snapshot rollback | **Accepted foundation:** P1/P4 and the Asset exit cover current override families; P12/P13 rerun the system-tier and final recovery matrix. |
| 14 | Five independent reference plugins | **Open:** SEO, identity, custom-content, media, and commerce/workflow references are P13 gates. |

The Program Definition of Done remains **0/24 checked**. Phase evidence must not
be copied into those final checkboxes before the complete reference, production,
performance, compatibility, and security gates pass.

## Not Credited

- `apps/api/app/Support/QueryRegistry/` is an untracked foundation with no
  lifecycle/bootstrap/Host execution path. P7 Query remains open.
- The route matrix does not cover locale paths, query/body, permission, CSRF,
  custom/raw guards, crash, composed streaming middleware, or redirect SEO.
  P6 remains 13/18.
- The accepted SDK/catalog row does not make the wire-only schedule RPC
  callable; schedule admission and triggering remain Host-owned by contract.
- Navigation/Region foundations have no frozen production placement, target,
  visibility, SSR consumption, or cache-invalidation contract.

## Next

1. Commit only the four synchronized knowledge files for this checkpoint;
   preserve the unrelated Page ViewModel test and content-policy manifest.
2. Keep public L2 default-off while implementing page-scoped CSP aggregation.
3. Finish the P6 guard/action/SEO/failure exits and P7 Query/Identity/Auth
   production paths without crediting disconnected foundations.
4. Resolve Navigation/Region product contracts, then continue P9 component
   composition, fragments, theme overrides, inspectors, and visual exits.

## Open Questions

- P6 still requires product freezes for mutable route fields, wrap ordering,
  after-failure behavior, redirect/canonical policy, and raw request/session
  authority.
- Navigation/Region still requires explicit target/placement, visibility,
  permission, SSR region, and cache invalidation semantics before production
  mapping.
