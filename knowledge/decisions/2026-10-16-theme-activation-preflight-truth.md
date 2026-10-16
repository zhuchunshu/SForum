# Theme Activation Preflight: What Actually Rejects A Scaffold

- Date: 2026-10-16
- Status: accepted (finding + copy correction)
- Related: M2/M9 of
  `knowledge/plans/2026-08-22-extension-authoring-speed-and-controlled-power.md`

## Question

The task book recorded that a `make:theme` scaffold passes `extension validate` and
`extension test` (0 errors) but is **rejected at runtime activation**, with the
client showing only the generic `extension.build_failed` copy. It hypothesised that
the scaffold is rejected because a one-page theme cannot replace the 34-page default
theme ("page coverage"). C1 asked for that comparison as a permanent Go test.

## Correction to this document's own first draft

This decision was first written as if the coverage experiment and the root-cause
finding were new in this pass. They were **not**: an earlier session (same day,
`apps/api/cmd/sforum/theme_scaffold_activation_test.go`, mtime 05:30) had already
(a) refuted the coverage hypothesis with `TestActiveThemeDoesNotRequireFullPageCoverage`,
(b) found the real root cause — the scaffold used to declare `components[]` whose
`propsSchema` pointed at a Host core-page contract, which the CLI accepted while the
runtime component registry rejected it, and (c) applied the fix (the scaffold no
longer declares components), pinning CLI↔runtime parity in
`TestThemeComponentSchemaDeclarationIsEnforcedByCliAndRuntime`.

What this pass adds: the **conflict control** (the page preflight does reject an
`action: add` path owned by another extension, and accepts the same package on an
unowned path), the stale-state analysis below as a secondary path, and the
`extension.build_failed` copy correction. Nothing about the coverage refutation or
the scaffold fix is claimed as new here.

## Finding: the coverage hypothesis is refuted (independently, earlier)

There is **no page-coverage check** anywhere in the theme activation path. What
`PreflightThemePackage` actually validates:

- the package loads (`LoadThemePackage`): manifest shape and `theme.json`;
- every declared skin CSS file resolves and is valid CSS;
- every `theme.json.pages[].template` loads, and a declared `theme.json.shell`
  template loads (this is the shell-loadability preflight);
- page contributions normalize (`prepareContributions`);
- **path-signature conflicts** only: with the new and the outgoing theme ignored,
  an `action: add` path already owned by another extension is rejected
  (`registry.go:664-693`, `ErrConflictProvider`).

Evidence — the earlier session's `TestActiveThemeDoesNotRequireFullPageCoverage`
(`apps/api/cmd/sforum/theme_scaffold_activation_test.go`) plus this pass's conflict
control `TestThemePreflightRejectsSamePathAddOwnedByAnotherExtension`
(`apps/api/app/Support/Pages/theme_scaffold_preflight_contract_test.go`):

- a scaffold-shaped theme (one `action: add` page, one template, one skin CSS, no
  shell) preflights cleanly as a fresh install (`prevThemeID = ""`);
- it also preflights cleanly when replacing the **real** default theme whose 34
  `action: replace` contributions are registered in the registry
  (`prevThemeID = "sforum.default-theme"`);
- the reverse control passes as expected: an `action: add` path already owned by a
  different extension **is** rejected — so that layer really does check conflicts,
  just not coverage.

The scaffold also passes the author command chain unchanged: `make:theme` →
`digest --write` → `validate` → `test` → `0 errors, 0 warnings, 5 checks`.

## Where a scaffold can actually be rejected

Two of the three preflights are **state-dependent**, which is exactly why the CLI
cannot reproduce them and why the generic copy misleads authors:

1. **Component registry transition** (`component_registry_themes.go:39-95`):
   after validating the artifact shape, it walks the theme registrations and
   deletes only those matching the target or the source. Any other theme
   registration fails the transition with
   `ErrComponentRegistryConflict: active theme component runtime <id> is outside the transition fence`.
   A leftover theme runtime from an earlier activation is enough.
2. **Asset registry transition** (`theme_asset_publication.go:36-63,420-454`):
   at most one theme publication may exist, and it must match either the outgoing
   theme or the incoming one (`assetArtifactMatchesExtension` /
   `artifact == target.Artifact`); otherwise `errAssetPublicationConflict`. A
   publication left behind by an incomplete uninstall/disable blocks every later
   activation.
3. **Manifest exactness** (the package-side family, and the *actual* historical
   cause): the three-way template identity and digest agreement, plus the
   `components[].propsSchema` exact-declaration rule. The scaffold's former
   `components[]` declaration is what made the CLI green while activation was
   rejected; the scaffold no longer emits it, and both sides now reject the same
   package (`TestThemeComponentSchemaDeclarationIsEnforcedByCliAndRuntime`).

## Decision

- Do **not** add a page-coverage requirement. It is not the cause, and requiring a
  theme to declare every page the previous theme owned would break the legitimate
  partial-theme case (core pages simply fall back to their core implementation).
- Correct the client copy for `extension.build_failed` in both locales so it names
  **both** families: imprecise package declarations, and stale state blocking the
  switch — and says the error text names the preflight layer while the extension
  event log holds the detail. Before this change the copy mentioned only the
  declaration family, so an author whose package was fine had no direction.
- Record that `extension validate` / `extension test` are **package-local** checks
  and cannot detect the two state-dependent rejections; the runbook for diagnosing
  them is the extension event log plus the two registry inspections.

## The two state-dependent paths are now observed, not just read

Both candidates were upgraded from code reading to focused observations:

- **Component fence** —
  `TestComponentRegistryThemeTransitionRejectsThirdThemeOutsideTheFence`
  (`app/Support/Extensions/component_registry_themes_test.go`): registering a third
  theme runtime and switching between two others is rejected with
  `ErrComponentRegistryConflict` naming the fence **and** the stale runtime id;
  clearing that runtime makes the same switch pass.
- **Asset publication** —
  `TestThemeAssetTransitionRejectsStaleDriftAndMultipleThemes/stale_third_theme_publication_blocks_every_switch`
  (`app/Models/Extensions/theme_asset_publication_test.go`): a single theme
  publication owned by a third (stale) theme makes the transition fail with
  `errAssetPublicationConflict`; clearing it makes the same switch pass. This
  exercises the `!matchesSource && !matchesTarget` branch, which the pre-existing
  "exactly one theme owner" subtest does not reach (it returns earlier, at "more
  than one theme").

Operator-facing consequence: `knowledge/runbooks/2026-10-16-theme-activation-rejection-diagnosis.md`
maps the failing preflight layer to its family, its remedy, and its durability
(the component registry is an in-process view restored from durable lifecycle state
at boot; the asset publication is durable and is not cleared by a restart alone).

## Limits of this finding (not claimed as verified)

- The **historical** rejection's root cause is pinned by the earlier session's fix
  and its parity test, not by a reproduction run in this pass.
- The two state-dependent paths are observed at unit level (the tests above), not by
  seeding stale state in the dev instance and watching a live activation fail.
- The improved `extension.build_failed` copy is now asserted by
  `TestThemeActivationFailureMessageNamesBothCauses`
  (`app/Support/Localization/locales_test.go`), which requires both locales to name
  the declaration family (`packageFiles`, `theme.json`), the stale-state family
  (leftover uninstall/disable), the event log, and the fact that the error names the
  failing preflight layer.
