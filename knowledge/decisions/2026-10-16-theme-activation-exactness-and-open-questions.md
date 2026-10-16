# Decision: Theme Activation Exactness, And Two Open Questions

## Status

Accepted for the scaffold/validation alignment below. **Two sub-questions are
`Proposed` and need a human decision** (sections 3 and 4); neither is decided
here.

## Context

`make:theme` produced a package that passed `extension validate` and
`extension test` (0 errors) but was **always rejected at runtime activation**
with the generic client message `extension.build_failed`
("主题验证失败，请检查 theme.json / assets 或兼容层路径"). Authors were blocked at
their first step, and the message pointed at the wrong cause.

Root cause (traced, not guessed). The activation path runs three preflights
(`apps/api/app/Models/Extensions/service_lifecycle.go:800-822`):

1. `pageRegistry.PreflightThemePackage`
2. `componentRegistry.ValidateThemeTransition`
3. `validateThemeAssetTransition`

The scaffold died at (2). `transitionTheme`
(`Support/Extensions/component_registry_themes.go:120-125`) installs the target
registration whenever `len(target.Manifest.Components) > 0` and rebuilds the
registry state through `buildComponentRegistryState` →
`compileComponentContribution` (`Support/Extensions/component_registry_schema.go:22`)
→ `compileComponentSchema` → `compileExactProviderSchema`
(`Support/Extensions/provider_slot_schemas.go:43-90`), which requires a
`packageFiles[]` entry of `kind: schema` whose `id`/`version` equal the schema
reference. The scaffold declared
`components[0].propsSchema = "sforum.page.home@1"` (a **Host core page schema**)
and declared no such package file, so activation failed with
`exact package schema declaration is missing`.

Recorded production evidence (dev instance `extension_events`):

```text
id 5865 | acme.switchtest | enable_failed | 2026-10-16 04:45:21+08
extensions: build failed: component registry preflight failed: component registry
declaration is invalid: contribution acme.switchtest.component.home props schema:
exact package schema declaration is missing
```

Contradicting hypothesis that was disproved: "a one-page theme is rejected for
failing to cover the 34 core pages". `PreflightThemeContributionsReplacing`
(`Support/Pages/registry.go:664-693`) only rejects **add-path signature**
conflicts and ignores both the incoming and the outgoing theme; and
`TestActiveThemeDoesNotRequireFullPageCoverage` (`cmd/sforum`) now proves a
1-page scaffold replacing the built-in default theme passes the page preflight.
Uncovered core pages keep rendering through the Host fallback. The earlier
hypothesis in `sessions/2026-10-16-m0-freeze-and-m1-shell-handoff.md:113-119` is
corrected here.

## Decision

1. **`make:theme` no longer declares `components[]`.** A theme component with
   `ssrTemplate` must repeat the template's `viewModelSchema` verbatim
   (`componentSSRTemplateConsistent`, `v3_validate_ui.go:675`), and for a theme
   template bound to a core page that reference is Host-owned
   (`sforum.page.home@1`). A theme package cannot declare a Host core schema
   without vendoring it. `Support/Pages/theme_runtime.go` never reads
   `Manifest.Components` for themes at all, and the only known-good runtime theme
   (`extensions/builtin/themes/sforum-default`) declares none. Theme L1
   presentation is `theme.json.pages[].template` + `templates[]` +
   `packageFiles[]`.
2. **Manifest validation now enforces the same exactness the component registry
   compiles** (`v3_validate_ui.go`, components loop): every component
   `propsSchema`/`resultSchema` must have a matching `packageFiles[]`
   `kind: schema` entry, with a message naming the component and the reference.
   Because `extension validate`, `extension test`, install, and activation all
   load the package through `extensionmanifest.LoadPackage`, authors now see the
   failure at CLI time. Providers, queries, notification types and identity
   operations already had this rule; components were the omission.
3. Fixtures that encoded the unactivatable shape were corrected, not weakened:
   `extensions/fixtures/plugins/sforum-region-demo` gained its exact props
   schema, `testdata/v3/l2-plugin.json` gained both component schema
   declarations, `testdata/v3/minimal-theme.json` dropped its `components[]`, and
   `TestManifestV3ThemePresentationContract` now exercises a theme component with
   a package-local L2 schema.
4. The localized `extension.build_failed` copy now names the real check
   categories and points at the extension event log for the specific cause.

## Consequences

- A theme scaffold passes CLI validation **and** the runtime activation
  preflights; the contract is pinned by
  `apps/api/cmd/sforum/theme_scaffold_activation_test.go`.
- New author-facing documentation:
  `docs/extensions/runtime-themes.md` → "What activation checks, and what it
  does not".
- The `extension.build_failed` response still carries only an error code; the
  specific cause is in `GET /admin/extensions/{id}/events`. Surfacing causes in
  the response body would be a contract/wire change and is not part of this
  decision.

## Open question 1 - Host core schema references in component declarations

**Status: Proposed, needs a human decision.**

Three author-facing artifacts treat a component `propsSchema` that points at a
Host core page schema as legitimate:

- `apps/api/app/Support/ExtensionManifest/testdata/v3/minimal-theme.json`
  (before this change)
- `TestManifestV3ThemePresentationContract` (before this change)
- the `make:theme` scaffold (before this change)

The runtime rejects that shape by construction
(`compileExactProviderSchema`, `provider_slot_schemas.go:59`). The repository
therefore has two incompatible readings:

- **Reading A (implemented here):** exact package declarations are mandatory for
  every component schema reference, so the author-facing artifacts were the bug.
  Fail-closed, no new escape hatch, but a theme can never declare an
  SSR-template component on a core page schema.
- **Reading B:** Host core view-model schemas are Host-provided (as they already
  are for `templates[].viewModelSchema`, which needs no package declaration), so
  components referencing them should compile without a package declaration.
  This is a deliberate relaxation of a validation boundary and would make a
  theme's `components[]` meaningful again; it needs its own review for the
  resulting nil-validator path.

Evidence to decide: `Support/Pages/theme_runtime.go` never consumes a theme's
`components[]`; `Support/Extensions/component_registry_themes.go` and
`component_registry_schema.go` are the only consumers; the built-in theme has
none. If Reading B is chosen, `compilComponentSchema` must resolve core schema
references through a Host-owned registry rather than skipping validation.

## Open question 2 - `core.guard.raw_request` with zero guard code

**Status: RESOLVED 2026-10-16 - decided as a vulnerability and tightened.**
The operator's decision (R1) is recorded in
`knowledge/decisions/2026-10-16-extension-authoring-contracts-freeze.md`
(Revision 1, R1). The evidence chain that raised the question stays here: see
`knowledge/sessions/2026-10-16-m4-guard-binding-verification-handoff.md`
(section "R3"). Outcome: the guard id is refused at manifest validation and all
runtime mappings fail closed; raw request authority requires a package-declared
`kind: raw_request` guard.
