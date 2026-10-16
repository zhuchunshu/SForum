# 2026-10-16 M0 Freeze + M1 Shell Increment Handoff

## Changed

- **M0 (contract freeze) completed** for
  `knowledge/plans/2026-08-22-extension-authoring-speed-and-controlled-power.md`.
  ADR: `knowledge/decisions/2026-10-16-extension-authoring-contracts-freeze.md`.
  Inventory:
  `knowledge/sessions/2026-10-16-m0-threat-and-inventory-report.md`.
- **M1 first increment implemented and verified**: `layout: site|bare` on public
  page contributions, with `bare` as the backward-compatible default.

## Decisions

- `layout` default is **`bare`**. Page contributions bind exact artifacts, and no
  current plugin template mounts global chrome, so defaulting to `site` would
  retroactively add a chrome layer to already-approved artifacts.
- `layout: site` resolves to the **Host fallback shell** (`SFHostPublicChrome`)
  until theme-authored shell artifacts exist. That is the contract's defined
  fallback branch, not a stub.
- The plugin page body moved into one reviewed outlet
  (`apps/web/app/components/extensions/public/SFRegistryPageBody.vue`); the route
  shell owns only shell selection. Two shells therefore cannot duplicate the body
  branches, and "exactly one navbar/footer" follows from "exactly one shell".
- **Descriptor points are not content surfaces** (recorded in the ADR). The M4
  milestone "`l2Widget` on every descriptor point" was **cancelled**: every other
  point is a link/action/badge descriptor, so accepting a widget there would
  create a validated, documented, never-rendered capability. Third premise
  correction in this program, after the M2 registry-merge and the M3
  `hero`/`composer_after`/`reply` rejections.

## Two Corrections Found By Independent Verification

1. **The custom-guard runtime is live, not closed.** Docs
   (`docs/extensions/routes.md:283,284,399`,
   `docs/extensions/authoring-guide.md:741`) claim a later wave, while
   `bootstrap/api_assembly_domains.go:618` injects a live evaluator and
   `app/Http/plugin_guard_runtime.go:32` invokes plugin code behind exact-artifact
   trust gates. The docs were written *after* the code went live.
2. **`core.guard.raw_request` grants raw credentials with zero guard code.**
   `plugin_guard_runtime.go:55-57` returns allow on an empty binding, that binding
   is *required* for `core.guard.raw_request`
   (`Support/Routes/request_authority.go:189-191`), and it mints
   `RequestAuthorityRaw`, under which credential headers are forwarded
   (`Support/Extensions/protocol_v2_request_authority.go:129-131`).

An earlier revision of my own ADR and report asserted the **opposite** of (2) - it
trusted the code comment "Credential forwarding remains closed" that sits directly
above the contradicting short-circuit. Behaviour was re-verified directly and the
documents were corrected inline. Intent behind (2) is **not** determinable from the
repository and needs a human decision.

## Verification

- `bun test` in `apps/web`: 1206 pass / 0 fail. `bun run typecheck` clean.
- `go build ./...` clean; pages/controllers/manifest/region focused suites green.
- `node scripts/v3-catalog/generate.mjs --check` green (16 files).
- `node tests/validate-architecture-boundaries.mjs` green, with two baselines
  **lowered** rather than raised: `Support/Pages/registry.go` 1055 -> 1017 and
  `Http/Controllers/Pages/controller.go` 1113 -> 1075, via same-package
  extractions (`validateContribution`/`normalizePublicPath` into
  `registry_contribution_validation.go`; transport DTOs and response helpers into
  `controller_pages_transport.go`).
- Rendered evidence: the probe plugin declares `/probe-bare` (undeclared layout)
  and `/probe-site` (`layout: site`) at the same digest. Browser DOM and SSR HTML
  both show 0 navbar / 0 footer / unchanged `max-w-4xl` class on `/probe-bare`,
  and exactly 1 navbar / 1 footer inside `sf-host-public-chrome` with the plugin
  body rendered in the shell on `/probe-site`. An invalid layout
  (`"fancy"`) is rejected at package preflight.

## M2 Progress (tasks 1-4 + the default flip)

- Tasks 1-4 done. Resolution and the 401/403/404 mapping moved into
  `app/composables/pages/useRegistryPageResolve.ts`; both catch-alls are now thin
  shells differing only in path source and cache key, with a contract test
  asserting neither re-inlines `/pages/resolve-path` or `createError`. The shell
  decision moved into one shared component
  (`app/components/extensions/public/SFRegistryPageShell.vue`), over the single
  reviewed body outlet.
- **Default flipped to `site`** (see the ADR revision and the plan for the
  evidence: 0/5 plugin page templates mount global chrome, 34/34 theme templates
  do). `bare` is the explicit opt-out.
- Runtime evidence, one probe digest, three pages: `/probe-default` (undeclared)
  -> `site` shell with 1 navbar + 1 footer; `/probe-site` -> same; `/probe-bare`
  -> `bare` with 0 + 0 and the unchanged `max-w-4xl` container. Narrow viewport
  (500x846): 1 navbar, 1 footer, no horizontal overflow.

## M2 Exit Gate 3 Verified At Runtime (theme switch)

The third M2 exit gate is "an active-theme change updates the shell without
rebuilding Nuxt". That is now runtime evidence, not a source assertion:

1. Forked the default theme into a scratch package under `extensions/dev/themes/`
   that is structurally identical **except** it declares no `shell` (namespace fully
   renamed so its contribution ids could not collide).
2. Activated it through the normal admin exact-artifact flow. Active theme moved
   `sforum.default-theme` `4691a605...` -> `acme.switchtest` `69d5acc74a1c...`.
3. Plugin pages then reported `data-registry-shell-source="host"` (was `theme`) while
   still rendering **exactly 1 navbar and 1 footer**, and `bare` stayed at 0/0. So the
   shell source follows the active theme per request, the shared geometry contract
   survives the switch, and the dev server was never rebuilt.
4. Restored `sforum-default` via the built-in "restore default theme" action, deleted
   the scratch package, and re-verified: active `4691a605...`, plugin pages back to
   `source=theme` with 1 navbar + 1 footer, home page chrome normal.

Recorded along the way because it will bite the next person: a **forged full-theme
fork** replaces the same 34 core pages, so restoring the default theme reports 34
`冲突` entries against the fork. Activation still succeeds (the conflict is disclosed,
not blocking), but the leftover fork must be removed or it keeps shadowing core pages.

Also observed: a scaffold produced by `sforum make:theme` passes `extension validate`
and `extension test` (0 errors) but the **runtime activation validator rejects it**
("当前无法安全激活主题 / 验证失败"). A theme that only replaces one page is not a valid
runtime theme. The runtime requirement I could confirm is that every provider template
needs a Manifest V3 declaration with `action`, `contractVersion`, `viewModelSchema`
and `digest` all non-empty (`theme_runtime.go`, `RequireDeclaredTemplates`) - the
scaffold satisfies that, so the remaining cause is something else (most likely required
page coverage) and is still unisolated. Worth a follow-up: either the scaffold should
produce an activatable theme or the CLI should fail it too.

## Blocker From The Concurrent Workstream

`node scripts/v3-catalog/generate.mjs --check` currently fails, and it is not from this
workstream:

```
Error: stable identity map contains removed sources; preserve ids through an explicit
move or record deprecation:
- apps/web/app/components/admin/ai/blocks/SFAdminAIIdentityBlock.vue
```

The other workstream deleted four files
(`app/Support/AI/identity.go`, `app/Support/AI/identity_test.go`,
`admin/ai/blocks/SFAdminAIIdentityBlock.vue`, `tests/admin/adminAIIdentity.test.ts`)
and has not yet updated `docs/extensions/v3/catalog-identities.json`, so the stable
identity map still points at a removed source. Their deleted test file also explains the
web suite moving 1209/163 -> 1201/162 (not a regression from this workstream; `bun test`
still reports 0 fail). I left their files and the shared identity ledger untouched.

## Two Honest Limitations

1. **`390x844` could not be produced in this session.** `Emulation.setDeviceMetricsOverride`
   is accepted but does not take effect in this browser carrier:
   `window.innerWidth` stayed 500 and `screen.width` stayed 1920 at `deviceScaleFactor`
   1 regardless of the requested mobile metrics. The QA above was therefore run at
   the narrowest width this carrier allows (500x846), which does exercise the narrow
   layout. The `390x844` pass is still owed and must be run where the viewport can
   actually be controlled. I am recording the measured width rather than labelling a
   500px screenshot as 390.
2. **Task 7's "selected theme provider/digest" evidence cannot be produced until
   Route A lands.** With the Host fallback shell, a `layout: site` plugin page has no
   theme artifact involved: the body belongs to the plugin
   (`sf-theme-template data-extension-id="acme.iconprobe"`) and the shell belongs to
   the Host. There is therefore no theme provider or digest to report for those
   pages. That item, and the part of task 8 that expects the shell to be attributable
   to a theme artifact, depend on Route A (theme-authored shell). Task 8's
   *re-staging* step was not applicable in this increment because no theme source was
   modified; the active artifact was nevertheless verified healthy by identity
   (`sforum.default-theme` 1.0.0, packageDigest `94aec837...`, nodeRevision 3, served
   from `/_sforum/assets/themes/...`).

## Next

- **M1 remainder**: theme-authored shell artifacts with exact package-file and
  digest declarations, publishing exactly one shell provider from the selected
  theme, both built-in themes updated with their identities, and the conflict /
  missing-artifact / digest-drift / disable / rollback / Safe Mode / theme-switch
  tests.
- **M2**: make the real Nuxt catch-all use the shell and close Browser evidence.
- **M4**: re-scoped from "implement" to "verify the existing guard V1 contract,
  correct the stale disclosure, and close allowed/denied coverage". It must also
  resolve the `core.guard.raw_request` intent question.
- **M6**: plugin session state, under the constraints frozen in ADR section 5
  (separate Redis key, never the session payload; own invalidation; privacy
  consumer; Host-declared limits).
- **M8**: dev loop, reusing the `APP_ENV=development` gate pattern, with no trust
  bypass and an explicit `fsnotify` dependency decision.

## Open Questions

- `core.guard.raw_request` intent (deliberate or a hole).
- Whether `HostRouteGuardAuthorizer` is a deliberately retained test double or
  orphaned dead code - it is referenced only from `_test.go` files.
- Whether guard `permissions[]` were meant to be Host-enforced; they are validated
  and disclosed but never enforced.
- Whether a `challenge` outcome is intended for plugin guards (M4 task 5 asks for
  allow/deny/challenge; no type, field, or proto message exists).
# 2026-10-16 M0 Freeze + M1 Shell Increment Handoff

## Changed

- **M0 (contract freeze) completed** for
  `knowledge/plans/2026-08-22-extension-authoring-speed-and-controlled-power.md`.
  ADR: `knowledge/decisions/2026-10-16-extension-authoring-contracts-freeze.md`.
  Inventory:
  `knowledge/sessions/2026-10-16-m0-threat-and-inventory-report.md`.
- **M1 first increment implemented and verified**: `layout: site|bare` on public
  page contributions, with `bare` as the backward-compatible default.

## Decisions

- `layout` default is **`bare`**. Page contributions bind exact artifacts, and no
  current plugin template mounts global chrome, so defaulting to `site` would
  retroactively add a chrome layer to already-approved artifacts.
- `layout: site` resolves to the **Host fallback shell** (`SFHostPublicChrome`)
  until theme-authored shell artifacts exist. That is the contract's defined
  fallback branch, not a stub.
- The plugin page body moved into one reviewed outlet
  (`apps/web/app/components/extensions/public/SFRegistryPageBody.vue`); the route
  shell owns only shell selection. Two shells therefore cannot duplicate the body
  branches, and "exactly one navbar/footer" follows from "exactly one shell".
- **Descriptor points are not content surfaces** (recorded in the ADR). The M4
  milestone "`l2Widget` on every descriptor point" was **cancelled**: every other
  point is a link/action/badge descriptor, so accepting a widget there would
  create a validated, documented, never-rendered capability. Third premise
  correction in this program, after the M2 registry-merge and the M3
  `hero`/`composer_after`/`reply` rejections.

## Two Corrections Found By Independent Verification

1. **The custom-guard runtime is live, not closed.** Docs
   (`docs/extensions/routes.md:283,284,399`,
   `docs/extensions/authoring-guide.md:741`) claim a later wave, while
   `bootstrap/api_assembly_domains.go:618` injects a live evaluator and
   `app/Http/plugin_guard_runtime.go:32` invokes plugin code behind exact-artifact
   trust gates. The docs were written *after* the code went live.
2. **`core.guard.raw_request` grants raw credentials with zero guard code.**
   `plugin_guard_runtime.go:55-57` returns allow on an empty binding, that binding
   is *required* for `core.guard.raw_request`
   (`Support/Routes/request_authority.go:189-191`), and it mints
   `RequestAuthorityRaw`, under which credential headers are forwarded
   (`Support/Extensions/protocol_v2_request_authority.go:129-131`).

An earlier revision of my own ADR and report asserted the **opposite** of (2) - it
trusted the code comment "Credential forwarding remains closed" that sits directly
above the contradicting short-circuit. Behaviour was re-verified directly and the
documents were corrected inline. Intent behind (2) is **not** determinable from the
repository and needs a human decision.

## Verification

- `bun test` in `apps/web`: 1206 pass / 0 fail. `bun run typecheck` clean.
- `go build ./...` clean; pages/controllers/manifest/region focused suites green.
- `node scripts/v3-catalog/generate.mjs --check` green (16 files).
- `node tests/validate-architecture-boundaries.mjs` green, with two baselines
  **lowered** rather than raised: `Support/Pages/registry.go` 1055 -> 1017 and
  `Http/Controllers/Pages/controller.go` 1113 -> 1075, via same-package
  extractions (`validateContribution`/`normalizePublicPath` into
  `registry_contribution_validation.go`; transport DTOs and response helpers into
  `controller_pages_transport.go`).
- Rendered evidence: the probe plugin declares `/probe-bare` (undeclared layout)
  and `/probe-site` (`layout: site`) at the same digest. Browser DOM and SSR HTML
  both show 0 navbar / 0 footer / unchanged `max-w-4xl` class on `/probe-bare`,
  and exactly 1 navbar / 1 footer inside `sf-host-public-chrome` with the plugin
  body rendered in the shell on `/probe-site`. An invalid layout
  (`"fancy"`) is rejected at package preflight.

## M2 Progress (tasks 1-4 + the default flip)

- Tasks 1-4 done. Resolution and the 401/403/404 mapping moved into
  `app/composables/pages/useRegistryPageResolve.ts`; both catch-alls are now thin
  shells differing only in path source and cache key, with a contract test
  asserting neither re-inlines `/pages/resolve-path` or `createError`. The shell
  decision moved into one shared component
  (`app/components/extensions/public/SFRegistryPageShell.vue`), over the single
  reviewed body outlet.
- **Default flipped to `site`** (see the ADR revision and the plan for the
  evidence: 0/5 plugin page templates mount global chrome, 34/34 theme templates
  do). `bare` is the explicit opt-out.
- Runtime evidence, one probe digest, three pages: `/probe-default` (undeclared)
  -> `site` shell with 1 navbar + 1 footer; `/probe-site` -> same; `/probe-bare`
  -> `bare` with 0 + 0 and the unchanged `max-w-4xl` container. Narrow viewport
  (500x846): 1 navbar, 1 footer, no horizontal overflow.

## M2 Exit Gate 3 Verified At Runtime (theme switch)

The third M2 exit gate is "an active-theme change updates the shell without
rebuilding Nuxt". That is now runtime evidence, not a source assertion:

1. Forked the default theme into a scratch package under `extensions/dev/themes/`
   that is structurally identical **except** it declares no `shell` (namespace fully
   renamed so its contribution ids could not collide).
2. Activated it through the normal admin exact-artifact flow. Active theme moved
   `sforum.default-theme` `4691a605...` -> `acme.switchtest` `69d5acc74a1c...`.
3. Plugin pages then reported `data-registry-shell-source="host"` (was `theme`) while
   still rendering **exactly 1 navbar and 1 footer**, and `bare` stayed at 0/0. So the
   shell source follows the active theme per request, the shared geometry contract
   survives the switch, and the dev server was never rebuilt.
4. Restored `sforum-default` via the built-in "restore default theme" action, deleted
   the scratch package, and re-verified: active `4691a605...`, plugin pages back to
   `source=theme` with 1 navbar + 1 footer, home page chrome normal.

Recorded along the way because it will bite the next person: a **forged full-theme
fork** replaces the same 34 core pages, so restoring the default theme reports 34
`冲突` entries against the fork. Activation still succeeds (the conflict is disclosed,
not blocking), but the leftover fork must be removed or it keeps shadowing core pages.

Also observed: a scaffold produced by `sforum make:theme` passes `extension validate`
and `extension test` (0 errors) but the **runtime activation validator rejects it**
("当前无法安全激活主题 / 验证失败"). A theme that only replaces one page is not a valid
runtime theme. The runtime requirement I could confirm is that every provider template
needs a Manifest V3 declaration with `action`, `contractVersion`, `viewModelSchema`
and `digest` all non-empty (`theme_runtime.go`, `RequireDeclaredTemplates`) - the
scaffold satisfies that, so the remaining cause is something else (most likely required
page coverage) and is still unisolated. Worth a follow-up: either the scaffold should
produce an activatable theme or the CLI should fail it too.

## Blocker From The Concurrent Workstream

`node scripts/v3-catalog/generate.mjs --check` currently fails, and it is not from this
workstream:

```
Error: stable identity map contains removed sources; preserve ids through an explicit
move or record deprecation:
- apps/web/app/components/admin/ai/blocks/SFAdminAIIdentityBlock.vue
```

The other workstream deleted four files
(`app/Support/AI/identity.go`, `app/Support/AI/identity_test.go`,
`admin/ai/blocks/SFAdminAIIdentityBlock.vue`, `tests/admin/adminAIIdentity.test.ts`)
and has not yet updated `docs/extensions/v3/catalog-identities.json`, so the stable
identity map still points at a removed source. Their deleted test file also explains the
web suite moving 1209/163 -> 1201/162 (not a regression from this workstream; `bun test`
still reports 0 fail). I left their files and the shared identity ledger untouched.

## Two Honest Limitations

1. **`390x844` could not be produced in this session.** `Emulation.setDeviceMetricsOverride`
   is accepted but does not take effect in this browser carrier:
   `window.innerWidth` stayed 500 and `screen.width` stayed 1920 at `deviceScaleFactor`
   1 regardless of the requested mobile metrics. The QA above was therefore run at
   the narrowest width this carrier allows (500x846), which does exercise the narrow
   layout. The `390x844` pass is still owed and must be run where the viewport can
   actually be controlled. I am recording the measured width rather than labelling a
   500px screenshot as 390.
2. **Task 7's "selected theme provider/digest" evidence cannot be produced until
   Route A lands.** With the Host fallback shell, a `layout: site` plugin page has no
   theme artifact involved: the body belongs to the plugin
   (`sf-theme-template data-extension-id="acme.iconprobe"`) and the shell belongs to
   the Host. There is therefore no theme provider or digest to report for those
   pages. That item, and the part of task 8 that expects the shell to be attributable
   to a theme artifact, depend on Route A (theme-authored shell). Task 8's
   *re-staging* step was not applicable in this increment because no theme source was
   modified; the active artifact was nevertheless verified healthy by identity
   (`sforum.default-theme` 1.0.0, packageDigest `94aec837...`, nodeRevision 3, served
   from `/_sforum/assets/themes/...`).

## Next

- **M1 remainder**: theme-authored shell artifacts with exact package-file and
  digest declarations, publishing exactly one shell provider from the selected
  theme, both built-in themes updated with their identities, and the conflict /
  missing-artifact / digest-drift / disable / rollback / Safe Mode / theme-switch
  tests.
- **M2**: make the real Nuxt catch-all use the shell and close Browser evidence.
- **M4**: re-scoped from "implement" to "verify the existing guard V1 contract,
  correct the stale disclosure, and close allowed/denied coverage". It must also
  resolve the `core.guard.raw_request` intent question.
- **M6**: plugin session state, under the constraints frozen in ADR section 5
  (separate Redis key, never the session payload; own invalidation; privacy
  consumer; Host-declared limits).
- **M8**: dev loop, reusing the `APP_ENV=development` gate pattern, with no trust
  bypass and an explicit `fsnotify` dependency decision.

## Open Questions

- `core.guard.raw_request` intent (deliberate or a hole).
- Whether `HostRouteGuardAuthorizer` is a deliberately retained test double or
  orphaned dead code - it is referenced only from `_test.go` files.
- Whether guard `permissions[]` were meant to be Host-enforced; they are validated
  and disclosed but never enforced.
- Whether a `challenge` outcome is intended for plugin guards (M4 task 5 asks for
  allow/deny/challenge; no type, field, or proto message exists).

## Corrections (2026-10-16, second session - do not rely on the originals)

Two statements in this handoff were disproved by the follow-up pass and are kept
visible instead of deleted:

1. **"A theme that only replaces one page is not a valid runtime theme" /
   "most likely required page coverage" - wrong.** Theme activation does not
   require page coverage: `PreflightThemeContributionsReplacing`
   (`Support/Pages/registry.go:664-693`) only rejects add-path signature
   conflicts, and uncovered core pages render through the Host fallback.
   `cmd/sforum/TestActiveThemeDoesNotRequireFullPageCoverage` proves a one-page
   scaffold replacing the 34-page default theme passes. The actual activation
   failure was the scaffold's `components[]` entry: its `propsSchema`
   (`sforum.page.home@1`, a Host core schema) has no matching `packageFiles[]`
   schema declaration, which the component registry compiles and rejects
   (`exact package schema declaration is missing`; production evidence in
   `extension_events` id 5865). Fixed in the generator, in manifest validation,
   in the fixtures, and in the localized failure copy -
   `decisions/2026-10-16-theme-activation-exactness-and-open-questions.md`.
2. **"`data-theme-region` is preserved" - wrong.** The theme-authored shell is
   rendered through the legacy L1 string path
   (`LoadTemplate` -> `SanitizeTemplateHTML`, bluemonday theme policy), which
   strips `data-theme-region` / `data-theme-owned` / `data-page` /
   `data-theme-shell` from `shellHtml`. Geometry classes and the wrapper's
   `data-extension-id` survive, so the built-in theme's CSS is unaffected; a
   theme keying CSS on those markers would see plugin pages differ from its own
   pages.

Also measured in that pass: on the live plugin pages the theme shell's
`.sf-footer` is `display:none` (`hybrid-forum.css:503`), so a plugin page shows
**no visible footer**, while the theme home page shows one visible footer plus a
hidden one. The M2 Browser acceptance "exactly one visible navbar and footer" is
therefore **not met on plugin pages today**, and it is a theme presentation
issue rather than a shell-ownership defect: the same rule hides the same element
on the theme's own templates.
