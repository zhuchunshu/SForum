# 2026-10-16 Route A Layer 6, Loader Transport Gap, And Theme Scaffold Alignment

Hot handoff for `plans/2026-08-22-extension-authoring-speed-and-controlled-power.md`
(M1/M2/M4 closure pass). Companion notes: `2026-10-16-m4-guard-binding-verification-handoff.md`
(M4 detail) and `decisions/2026-10-16-theme-activation-exactness-and-open-questions.md`
(theme activation rules + the two open questions).

## Changed

- **A1 - Loader Gateway transport gap found and pinned (not fixed; needs a human).**
  A Protocol V2 plugin runtime deliberately exposes **no** loopback HTTP base
  (`protocol_v2_client.go:463` returns an empty target; V1 is rejected in
  `protocol_v2_client.go:188`), while the Loader Gateway's only transport *is*
  that loopback base (`Support/Pages/loader_gateway.go:98-108`) and the
  production adapter rejects an empty base
  (`bootstrap/app.go`, `pageRouteTargetAdapter`). So `dataSource: plugin` page
  contributions can never obtain loader data and always fall back with
  `view_model_unavailable`.
  - New runtime test:
    `app/Support/Extensions/IntegrationTests/page_loader_transport_integration_test.go`
    `TestPageLoaderGatewayHasNoTransportForProtocolV2Runtime` - real Protocol V2
    subprocess, real `Manager`, real production target source
    (`bootstrap.NewPageRouteTargetSource`), real gateway -> fallback/503.
  - New unit test:
    `bootstrap.TestPageRouteTargetAdapterDeniesRuntimeWithoutLoopbackTarget`
    (the denial branch had no coverage) and a narrow `PageRouteTargetRuntime`
    interface + exported `NewPageRouteTargetSource` so joined tests reuse the
    production admission rule instead of reimplementing it.
  - `bootstrap/api_assembly_domains.go` now builds the loader with that
    constructor (no behaviour change).
- **C - `make:theme` scaffold vs runtime activation validator, closed.**
  Root cause: the scaffold declared `components[]` with
  `propsSchema: "sforum.page.home@1"` (a Host core schema) and no matching
  `packageFiles[]` schema declaration, so the component-registry preflight
  always failed with `exact package schema declaration is missing` (production
  evidence: `extension_events` id 5865). Fixes: the scaffold no longer declares
  theme components; manifest validation now enforces the same exactness the
  registry compiles (so CLI and activation agree); the misleading
  `extension.build_failed` copy now names the real check categories; fixtures
  that encoded the unactivatable shape were corrected; docs gained the exact
  activation checklist. Tests:
  `apps/api/cmd/sforum/theme_scaffold_activation_test.go` (4 tests, incl. the
  parity test and the disproved coverage hypothesis).
- **B - Route A layer 6 tests.** `controller_layout_test.go`:
  `TestResolvePathReturnsExactThemeShellWhenDeclared` now asserts exactly one
  `<sf-navbar>`, `<sf-footer>`, `<sf-plugin-page-body>` and surviving geometry
  classes on the composed shell; `TestResolvePathLeavesShellEmptyForExplicitBareLayout`
  pins that a declared shell never reaches `bare`/empty/unknown layouts; new
  `TestResolvePathFailsOpenToHostShellWhenDeclaredShellDisappears` pins
  resolve-time fail-open (shell unreadable -> empty `shellHtml`, still 200,
  layout stays `site`).
- **D - M4 verification + doc correction.** Guard tables in
  `docs/extensions/routes.md`, `docs/zh-CN/extensions/routes.md`,
  `docs/en-US/extensions/routes.md` and `docs/extensions/authoring-guide.md` no
  longer claim custom/`raw_request` guards are a later wave. No production code
  changed. See the M4 handoff for the allowed/denied coverage map.
- **A4 - outer artifact identity evidence (runtime, HTTP).**
  `GET /api/v1/site/active-theme/skin` ->
  `sforum.default-theme` 1.0.0 `4691a60541836c57…`, nodeRevision 3.
  `GET /api/v1/pages/resolve-path?path=/probe-default` ->
  `layout: site`, `shellProvider: sforum.default-theme`, `shellVersion: 1.0.0`,
  `shellPackageDigest: 4691a60541836c57…` (identical digest),
  plugin body `provider: acme.iconprobe`,
  `selectedPackageDigest: 9dfe2359330f5b0ef69cdb32ddfe6270b3b3271b33a0afdd8b4e4a9854eb3d6b`,
  `fallback: false`.

## Corrected Claims (my own earlier statements)

1. **"1-page theme rejected for insufficient page coverage" - wrong.**
   `PreflightThemeContributionsReplacing` only rejects add-path signature
   conflicts and an uncovered page is served by the Host fallback.
   `TestActiveThemeDoesNotRequireFullPageCoverage` proves a 1-page scaffold
   replacing the 34-page default theme passes. The real cause was the component
   schema declaration (above).
2. **"`data-theme-region` survives shell compilation" - wrong.**
   The theme-authored shell is rendered through the legacy L1 string path
   (`LoadTemplate` -> `SanitizeTemplateHTML` -> bluemonday theme policy), so
   `data-theme-region` / `data-theme-owned` / `data-page` / `data-theme-shell`
   are **stripped** from `shellHtml` (verified: `GET /pages/resolve-path` returns
   `<div class="sf-page sf-theme--default sf-theme-shell--fullwidth-3col">` with
   no data attributes). Impact is marker-only today: the built-in theme's CSS
   keys on `[data-extension-id]` (carried by the `SFThemeTemplate` wrapper) and
   on `.sf-theme-shell--fullwidth-3col` / `.sf-theme-chrome-body` classes, which
   do survive. A theme that keys its own CSS on `[data-theme-region]` would see
   plugin pages differ from its own pages. Pinned by the new assertion in
   `TestResolvePathReturnsExactThemeShellWhenDeclared`.

## Runtime Browser Evidence (2026-10-16, web :3000, API :8080)

DOM counts use visibility-filtered `querySelectorAll` in the live page:

| Page | `data-registry-shell` | source | visible `header.navbar` | visible `footer.sf-footer` | theme shell node | host chrome |
| --- | --- | --- | --- | --- | --- | --- |
| `/probe-default` | `site` | `theme` | 1 | 1 (element; `display:none` at 1278px by theme CSS) | `DIV.sf-page.sf-theme--default.sf-theme-shell--fullwidth-3col` | 0 |
| `/probe-site` | `site` | `theme` | 1 | 1 | same | 0 |
| `/probe-bare` | `bare` | `none` | 0 | 0 | 0 | 0 |

Shell children on `/probe-default`: `HEADER.navbar`, `NAV.sf-public-mobile-bottom-nav`,
`DIV.sf-theme-chrome-body`, `FOOTER.sf-footer` - exactly one navbar and one
footer, and the plugin body renders inside `.sf-theme-chrome-body`.
Measured viewport in this carrier: **1278x846** (the width is not fixed at 500
in this session; `Emulation.setDeviceMetricsOverride` was not re-tested).

## Theme Switch Runbook (reproducible)

1. `export https_proxy=… http_proxy=… all_proxy=…` (only needed for network commands).
2. Confirm the baseline: `curl -s http://127.0.0.1:8080/api/v1/site/active-theme/skin`
   and `curl -s "http://127.0.0.1:8080/api/v1/pages/resolve-path?path=/probe-default"`
   - note `extensionId`/`packageDigest` and `shellProvider`/`shellPackageDigest`.
3. `cd apps/api && go run ./cmd/sforum make:theme --no-interaction --id acme.shelltest … --out extensions/dev/themes/acme.shelltest`
   then remove the `shell` key from `theme.json` (this is the "no shell" variant).
4. Stage the source tree: `./scripts/build-builtin-plugins.sh`, then restart the
   API (append/remove a comment line in `app/Support/Pages/catalog.go`; air only
   watches non-`_test.go` content writes).
5. Activate through the admin flow (`POST /api/v1/admin/extensions/{id}/activate`
   with the preview digest + token) as `super_admin`.
6. Re-run step 2: `shellProvider` must disappear while `layout` stays `site` and
   the plugin page still shows exactly one navbar and one footer
   (`data-registry-shell-source` moves `theme` -> `host`) with no Nuxt rebuild.
7. Restore the built-in default theme ("restore default theme" action), delete
   the scratch package, re-verify step 2 and the chrome counts.

Automated assertion of the same invariant:
`TestResolvePathReturnsExactThemeShellWhenDeclared` (shell present) plus
`TestResolvePathLeavesShellEmptyWhenThemeDeclaresNone` (theme without a shell ->
host fallback).

## Next

- **A1 blocker (human decision).** Choose between (a) dispatching the page data
  route through the Route Registry over gRPC, (b) adding a page-data RPC, or
  (c) retiring the `dataSource: plugin` path. Until then `dataSource: plugin`
  pages cannot render plugin data and M2 task 5/7's browser evidence cannot be
  produced. Recorded with full evidence in the decision note.
- **A3.** A genuine `390x844` pass still needs a carrier where the viewport is
  controllable; this session measured 1278x846 only.
- **B1 remainder.** A web-side behavioral test for the shell/island branch needs
  a `.vue` SFC mount harness; this repo's bun tests read `.vue` as text or mount
  inline components only. The Go render-layer assertions plus the live DOM
  evidence above stand in for now.
- **M3+.** M3 (page/middleware scaffolds) is unblocked once M2's remaining items
  are judged; M5/M7 are unblocked by M4's closure except for the guard questions.
# 2026-10-16 Route A Layer 6, Loader Transport Gap, And Theme Scaffold Alignment

Hot handoff for `plans/2026-08-22-extension-authoring-speed-and-controlled-power.md`
(M1/M2/M4 closure pass). Companion notes: `2026-10-16-m4-guard-binding-verification-handoff.md`
(M4 detail) and `decisions/2026-10-16-theme-activation-exactness-and-open-questions.md`
(theme activation rules + the two open questions).

## Changed

- **A1 - Loader Gateway transport gap found and pinned (not fixed; needs a human).**
  A Protocol V2 plugin runtime deliberately exposes **no** loopback HTTP base
  (`protocol_v2_client.go:463` returns an empty target; V1 is rejected in
  `protocol_v2_client.go:188`), while the Loader Gateway's only transport *is*
  that loopback base (`Support/Pages/loader_gateway.go:98-108`) and the
  production adapter rejects an empty base
  (`bootstrap/app.go`, `pageRouteTargetAdapter`). So `dataSource: plugin` page
  contributions can never obtain loader data and always fall back with
  `view_model_unavailable`.
  - New runtime test:
    `app/Support/Extensions/IntegrationTests/page_loader_transport_integration_test.go`
    `TestPageLoaderGatewayHasNoTransportForProtocolV2Runtime` - real Protocol V2
    subprocess, real `Manager`, real production target source
    (`bootstrap.NewPageRouteTargetSource`), real gateway -> fallback/503.
  - New unit test:
    `bootstrap.TestPageRouteTargetAdapterDeniesRuntimeWithoutLoopbackTarget`
    (the denial branch had no coverage) and a narrow `PageRouteTargetRuntime`
    interface + exported `NewPageRouteTargetSource` so joined tests reuse the
    production admission rule instead of reimplementing it.
  - `bootstrap/api_assembly_domains.go` now builds the loader with that
    constructor (no behaviour change).
- **C - `make:theme` scaffold vs runtime activation validator, closed.**
  Root cause: the scaffold declared `components[]` with
  `propsSchema: "sforum.page.home@1"` (a Host core schema) and no matching
  `packageFiles[]` schema declaration, so the component-registry preflight
  always failed with `exact package schema declaration is missing` (production
  evidence: `extension_events` id 5865). Fixes: the scaffold no longer declares
  theme components; manifest validation now enforces the same exactness the
  registry compiles (so CLI and activation agree); the misleading
  `extension.build_failed` copy now names the real check categories; fixtures
  that encoded the unactivatable shape were corrected; docs gained the exact
  activation checklist. Tests:
  `apps/api/cmd/sforum/theme_scaffold_activation_test.go` (4 tests, incl. the
  parity test and the disproved coverage hypothesis).
- **B - Route A layer 6 tests.** `controller_layout_test.go`:
  `TestResolvePathReturnsExactThemeShellWhenDeclared` now asserts exactly one
  `<sf-navbar>`, `<sf-footer>`, `<sf-plugin-page-body>` and surviving geometry
  classes on the composed shell; `TestResolvePathLeavesShellEmptyForExplicitBareLayout`
  pins that a declared shell never reaches `bare`/empty/unknown layouts; new
  `TestResolvePathFailsOpenToHostShellWhenDeclaredShellDisappears` pins
  resolve-time fail-open (shell unreadable -> empty `shellHtml`, still 200,
  layout stays `site`).
- **D - M4 verification + doc correction.** Guard tables in
  `docs/extensions/routes.md`, `docs/zh-CN/extensions/routes.md`,
  `docs/en-US/extensions/routes.md` and `docs/extensions/authoring-guide.md` no
  longer claim custom/`raw_request` guards are a later wave. No production code
  changed. See the M4 handoff for the allowed/denied coverage map.
- **A4 - outer artifact identity evidence (runtime, HTTP).**
  `GET /api/v1/site/active-theme/skin` ->
  `sforum.default-theme` 1.0.0 `4691a60541836c57…`, nodeRevision 3.
  `GET /api/v1/pages/resolve-path?path=/probe-default` ->
  `layout: site`, `shellProvider: sforum.default-theme`, `shellVersion: 1.0.0`,
  `shellPackageDigest: 4691a60541836c57…` (identical digest),
  plugin body `provider: acme.iconprobe`,
  `selectedPackageDigest: 9dfe2359330f5b0ef69cdb32ddfe6270b3b3271b33a0afdd8b4e4a9854eb3d6b`,
  `fallback: false`.

## Corrected Claims (my own earlier statements)

1. **"1-page theme rejected for insufficient page coverage" - wrong.**
   `PreflightThemeContributionsReplacing` only rejects add-path signature
   conflicts and an uncovered page is served by the Host fallback.
   `TestActiveThemeDoesNotRequireFullPageCoverage` proves a 1-page scaffold
   replacing the 34-page default theme passes. The real cause was the component
   schema declaration (above).
2. **"`data-theme-region` survives shell compilation" - wrong.**
   The theme-authored shell is rendered through the legacy L1 string path
   (`LoadTemplate` -> `SanitizeTemplateHTML` -> bluemonday theme policy), so
   `data-theme-region` / `data-theme-owned` / `data-page` / `data-theme-shell`
   are **stripped** from `shellHtml` (verified: `GET /pages/resolve-path` returns
   `<div class="sf-page sf-theme--default sf-theme-shell--fullwidth-3col">` with
   no data attributes). Impact is marker-only today: the built-in theme's CSS
   keys on `[data-extension-id]` (carried by the `SFThemeTemplate` wrapper) and
   on `.sf-theme-shell--fullwidth-3col` / `.sf-theme-chrome-body` classes, which
   do survive. A theme that keys its own CSS on `[data-theme-region]` would see
   plugin pages differ from its own pages. Pinned by the new assertion in
   `TestResolvePathReturnsExactThemeShellWhenDeclared`.

## Runtime Browser Evidence (2026-10-16, web :3000, API :8080)

DOM counts use visibility-filtered `querySelectorAll` in the live page:

| Page | `data-registry-shell` | source | visible `header.navbar` | visible `footer.sf-footer` | theme shell node | host chrome |
| --- | --- | --- | --- | --- | --- | --- |
| `/probe-default` | `site` | `theme` | 1 | 1 (element; `display:none` at 1278px by theme CSS) | `DIV.sf-page.sf-theme--default.sf-theme-shell--fullwidth-3col` | 0 |
| `/probe-site` | `site` | `theme` | 1 | 1 | same | 0 |
| `/probe-bare` | `bare` | `none` | 0 | 0 | 0 | 0 |

Shell children on `/probe-default`: `HEADER.navbar`, `NAV.sf-public-mobile-bottom-nav`,
`DIV.sf-theme-chrome-body`, `FOOTER.sf-footer` - exactly one navbar and one
footer, and the plugin body renders inside `.sf-theme-chrome-body`.
Measured viewport in this carrier: **1278x846** (the width is not fixed at 500
in this session; `Emulation.setDeviceMetricsOverride` was not re-tested).

## Theme Switch Runbook (reproducible)

1. `export https_proxy=… http_proxy=… all_proxy=…` (only needed for network commands).
2. Confirm the baseline: `curl -s http://127.0.0.1:8080/api/v1/site/active-theme/skin`
   and `curl -s "http://127.0.0.1:8080/api/v1/pages/resolve-path?path=/probe-default"`
   - note `extensionId`/`packageDigest` and `shellProvider`/`shellPackageDigest`.
3. `cd apps/api && go run ./cmd/sforum make:theme --no-interaction --id acme.shelltest … --out extensions/dev/themes/acme.shelltest`
   then remove the `shell` key from `theme.json` (this is the "no shell" variant).
4. Stage the source tree: `./scripts/build-builtin-plugins.sh`, then restart the
   API (append/remove a comment line in `app/Support/Pages/catalog.go`; air only
   watches non-`_test.go` content writes).
5. Activate through the admin flow (`POST /api/v1/admin/extensions/{id}/activate`
   with the preview digest + token) as `super_admin`.
6. Re-run step 2: `shellProvider` must disappear while `layout` stays `site` and
   the plugin page still shows exactly one navbar and one footer
   (`data-registry-shell-source` moves `theme` -> `host`) with no Nuxt rebuild.
7. Restore the built-in default theme ("restore default theme" action), delete
   the scratch package, re-verify step 2 and the chrome counts.

Automated assertion of the same invariant:
`TestResolvePathReturnsExactThemeShellWhenDeclared` (shell present) plus
`TestResolvePathLeavesShellEmptyWhenThemeDeclaresNone` (theme without a shell ->
host fallback).

## Next

- **A1 blocker (human decision).** Choose between (a) dispatching the page data
  route through the Route Registry over gRPC, (b) adding a page-data RPC, or
  (c) retiring the `dataSource: plugin` path. Until then `dataSource: plugin`
  pages cannot render plugin data and M2 task 5/7's browser evidence cannot be
  produced. Recorded with full evidence in the decision note.
- **A3.** A genuine `390x844` pass still needs a carrier where the viewport is
  controllable; this session measured 1278x846 only.
- **B1 remainder.** A web-side behavioral test for the shell/island branch needs
  a `.vue` SFC mount harness; this repo's bun tests read `.vue` as text or mount
  inline components only. The Go render-layer assertions plus the live DOM
  evidence above stand in for now.
- **M3+.** M3 (page/middleware scaffolds) is unblocked once M2's remaining items
  are judged; M5/M7 are unblocked by M4's closure except for the guard questions.

## Round 3 Follow-up: The Footer Defect Is Fixed, And Two Earlier Statements Were Incomplete

Measured with a **real Chrome over CDP** (`google-chrome --headless=new
--remote-debugging-port`, `Emulation.setDeviceMetricsOverride`), so this is a
genuine 390x844 pass rather than the 500px floor of the sidebar carrier:
`innerWidth 390 / innerHeight 844 / dpr 3 / screen 390x844`. The same harness
also produced a 1440x900 desktop pass.

**Root cause, correctly located this time.** Footer visibility is owned by the
**Host** stylesheet `apps/web/app/assets/css/sforum-theme.css`, not by the theme:
it hides the page-level `.sf-footer` of the default theme's fullwidth-3col shell
(`.sf-theme-template[data-extension-id="sforum.default-theme"]
.sf-page.sf-theme--default.sf-theme-shell--fullwidth-3col > .sf-footer`) because
a theme page's visible footer comes from `.sforum-content-footer` rendered inside
the content column, and the same file suppresses the shell footer whenever
`:has(.sforum-content-footer)` matches to avoid double chrome. A Page Registry
`add` page has no content column, so it fell through both rules and had **no
visible footer**. The theme's own `hybrid-forum.css:503` rule is real but was not
the effective cause; the earlier entry in this note attributed the defect to the
theme alone, which was incomplete.

**Fix (2 files, one contract owner):**

- `apps/web/app/assets/css/sforum-theme.css` - new complementary rule for
  `.sf-page.sf-theme--default.sf-theme-shell--fullwidth-3col:not(:has(.sforum-content-footer)) > .sf-footer { display: block }`.
  The `:has()` branch keeps suppressing the shell footer on theme pages, so the
  geometric contract still has exactly one visible owner per render path.
- `extensions/builtin/themes/sforum-default/templates/plugin-page-shell.html` -
  the shell root gains a stable hook class `sf-theme-shell--plugin-page`
  (markup-only; a theme-side CSS attempt was written first, found to sit inside
  `@media (min-width: 981px)` where it could not help at 390px, and reverted so
  visibility stays with the Host contract).

**Theme artifact flow (hard rule 5) executed twice, ending consistent:**

| Step | Active artifact |
| --- | --- |
| baseline | `4691a60541836c57…` (nodeRevision 3) |
| after first rebuild + admin reactivation | `7464f73f9cd6e1d4…` (nodeRevision 6) |
| after CSS revert + rebuild + admin reactivation | `85ddda6ad7825ada…` (nodeRevision 6) |

Source tree and the newest staged artifact agree:
`storage/extensions/sforum.default-theme/1.0.0/85ddda6ad7825ada…/`.

**Verified at both viewports (fresh headless Chrome, 2026-10-16):**

| Page | viewport | `data-registry-shell` | visible navbar | visible footer | footer elements | host chrome | overflowX | clipped | console errors |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `/` | 390x844 dpr3 | (theme page, no registry shell) | 1 | 1 | 2 (one suppressed by the content-column rule) | 0 | false | 0 | none |
| `/probe-default` | 390x844 dpr3 | `site` / `theme` | 1 | 1 | 1 | 0 | false | 0 | none |
| `/probe-site` | 390x844 dpr3 | `site` / `theme` | 1 | 1 | 1 | 0 | false | 0 | anonymous `GET /api/v1/auth/session` -> 401 |
| `/probe-bare` | 390x844 dpr3 | `bare` / `none` | 0 | 0 | 0 | 0 | false | 0 | none |
| `/`, `/probe-default`, `/probe-site`, `/probe-bare` | 1440x900 dpr1 | same pattern | 1/1/1/0 | 1/1/1/0 | 2/1/1/0 | 0 | false | 0 | same 401 note |

The 401 is the expected anonymous-session response in a fresh profile (the pages
are public); it is not a page or shell error. Screenshots:
`storage/qa/2026-10-16-390x844_probe-default.png`, `…_probe-site.png`,
`…_probe-bare.png`, `…_.png` (gitignored).

DOM evidence, `/probe-default` at 390x844:

```html
<div class="sf-theme-template" data-extension-id="sforum.default-theme"
     data-registry-shell="site" data-registry-shell-source="theme">
  <div class="sf-page sf-theme--default sf-theme-shell--fullwidth-3col sf-theme-shell--plugin-page">
    <header class="navbar" …>
```

DOM evidence, `/probe-bare` at 390x844 (no chrome of any kind):

```html
<main class="sf-public-page sf-registry-add mx-auto w-full max-w-4xl px-4 py-8"
      data-registry-shell="bare" data-registry-shell-source="none">
  <div class="sf-theme-template" data-extension-id="acme.iconprobe">…
```

**Therefore A3 is no longer carrier-blocked:** the sidebar carrier clamps its
viewport to 500x846, but a real Chrome driven over CDP produced the genuine
390x844 pass required by M2 task 7, and both viewports now satisfy "exactly one
visible navbar and one visible footer, no horizontal overflow, no clipped
controls, no page console errors, bare has no chrome".

**Third correction.** The statement written earlier in this same session that
"M2's Browser acceptance is not met on plugin pages" was true when measured and
is now obsolete: it is met after the fix above.

## Round 4: M3 Delivered, And Its Runtime Render Verified

M3 (page/middleware scaffolds) is complete. New flags on `make:plugin`/`make:theme`:
`--public-page <slug>`, `--public-page-layout site|bare`, `--route-middleware`
(plugin + `--backend`), with the admin-page flag reused rather than duplicated and
the same options added to the interactive forms. Implementation lives in
`apps/api/cmd/sforum/generator_authoring_workflow.go` plus small wiring edits in
`generator.go`/`command.go`.

Exit-gate evidence on a fresh scaffold with no hand-editing:

| Command | Result |
| --- | --- |
| `go run ./cmd/sforum extension validate /tmp/m3final` | OK (routes 2, templates 1, packageFiles 2) |
| `... extension test /tmp/m3final` | 0 errors, 1 expected scaffold-stub warning, 7 checks incl. `template.runtime_exact` |
| `... extension build --allow-scaffold /tmp/m3final` | OK, 0 errors |
| generated `routes[]` | both `before`/`after`, `guard: core.guard.inherit`, `fallback: closed`, no `access`/`permission`, only own pointers (`/query/<id>-trace`, `/headers/x-<id>-trace`) |

**Runtime render evidence (not just CLI).** The generated **theme** scaffold was
packaged (`extension package`), installed, and activated in the dev instance
through the real admin flow (exact-artifact trust challenge + `activate` with
`approveCoreReplacements`), then measured in real headless Chrome over CDP at
390x844 (dpr 3):

| Page | active theme | shell/source | visible navbar | visible footer | overflow | clipped | console |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `/notes` (generated `--public-page notes --public-page-layout bare`) | `acme.m3theme2` `fb431b49…` | `bare` / `none` | 0 | 0 | false | 0 | anonymous `/api/v1/auth/session` 401 only |
| `/` (that theme's own home template) | same | (theme page) | 1 | 1 | false | 0 | none |

`GET /pages/resolve-path?path=/notes` returned `provider: acme.m3theme`,
`action: add`, `layout: bare`, `fallback: false`,
`contractVersion: sforum.page.acme_m3theme_notes@1`, and empty `shellHtml`
(bare never receives a shell). The instance was then restored: active theme back
to `sforum.default-theme` `85ddda6a…` (nodeRevision 12), `/probe-default` still
`site` + theme shell at the same digest, and both scratch themes uninstalled.

**Two scaffold defects found by that render, and fixed:**

1. **The theme home template owned no chrome.** The generated
   `templates/home.html` only mounted `<sf-home-page>`, so an activated
   scaffold theme produced a site with no global navbar on its home page. It now
   mounts `sf-navbar` + a body `<main>` + `sf-footer`, matching the contract
   ("every render path has exactly one chrome owner"). Verified above: 1 visible
   navbar + 1 visible footer.
2. **An island tag inside an HTML comment breaks the package.** The first
   version of the new comment mentioned the plugin body island by tag name and
   the exact-template preflight failed with
   `pages: unclosed host island "sf-plugin-page-body"`, because the island
   extractor scans comments too. The comment no longer contains island tags, and
   this trap is recorded in the template comment itself.

**Honest limitation:** the plugin variant of `--public-page` is verified at CLI,
contract-test and exact-template-preflight level only. It cannot render in the
dev instance until its backend binary exists (the scaffold ships a stub), and
plugin page **data** stays blocked by the loader-transport gap above.

## Item Status For The Closure Checklist

- **A1** (executable plugin page with typed loader data): **blocked**, see the
  Next section. Evidence for the blocker is in place; the page cannot be landed
  until a transport decision is made.
- **A2** (missing runtime / disabled plugin): **closed by coverage, no new test
  written.** "Missing runtime" is exercised by
  `TestResolvePathReportsLayoutRegardlessOfBodyRender` (the harness registers no
  runtime snapshot, so the body render fails while `layout` stays `site`) and by
  the new loader-transport test (a real runtime present, transport missing ->
  fallback, still 200/site). "Disabled plugin" is `TestEnableDisableSyncsPageRegistry`
  (`app/Models/Extensions/page_lifecycle_effect_test.go`): disabling clears the
  contribution immediately, so there is no page to give chrome to, and the
  public catch-all's not-found -> 404 mapping in
  `app/web/app/composables/pages/useRegistryPageResolve.ts` handles the rest.
  That mapping is currently pinned by a source-contract assertion, not a
  rendered-behaviour test - residual, not a new synonymous test.
- **A3** (genuine 390x844): **verified complete** - see the round-3 section
  above (real Chrome over CDP, 390x844 dpr3 and 1440x900 dpr1, both satisfying
  the acceptance). The sidebar carrier's clamp to 500x846 is still worth
  knowing when only `ego_*` is available; that is why the earlier entry called
  this carrier-blocked. Original text kept below for the record:

  **carrier-blocked, measured.** In this carrier
  `Emulation.setDeviceMetricsOverride` is accepted but the page viewport is
  pinned at **500x846**: requesting width 390 yields `innerWidth`/`clientWidth`/
  `visualViewport.width` 500, and requesting 1440 also yields 500;
  `screen.width` stays 1920 and `devicePixelRatio` stays 1. Narrow-width QA was
  therefore run at 500x846 (the floor), where `/probe-default` shows exactly one
  visible navbar, one theme shell, the body inside `.sf-theme-chrome-body`, no
  horizontal overflow (`scrollWidth == innerWidth == 500`) and no clipped
  controls, and `/probe-bare` shows 0 chrome of any kind. 390x844 itself is not
  produced and is not claimed.
- **A4** (outer artifact identity): **verified at runtime** (`/site/active-theme/skin`
  and `/pages/resolve-path` agree on `sforum.default-theme` 1.0.0
  `4691a60541836c57…`; probe page `fallback: false`, `layout: site`,
  `shellProvider` = the theme).
- **B1/B2/B3**: layer-6 assertions, bare isolation and resolve-time fail-open are
  automated; the theme-switch runbook is the section above. The remaining B1 gap
  is a web-side `.vue` mount harness (see Next).
- **C1/C2/C3**: closed with tests, docs, message fix and corrected fixtures.
- **D1-D4**: M4 verified and documented; R3 and two guard questions stay open for
  a human. Coverage was mapped, not duplicated.
