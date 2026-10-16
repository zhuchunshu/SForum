# Runbook: Diagnose A Rejected Theme Activation

- Date: 2026-10-16
- Applies to: `POST /api/v1/admin/extensions/:id/activate` failing with
  `extension.build_failed` (theme activation)
- Companion decision: `knowledge/decisions/2026-10-16-theme-activation-preflight-truth.md`

Theme activation runs three preflights in order
(`app/Models/Extensions/service_lifecycle.go`): the **page** preflight, the
**component registry** transition, and the **asset registry** transition. Only the
first is package-local; the other two are state-dependent, which is why
`extension validate` / `extension test` can be green while activation is rejected.

## Step 1 — read which layer failed

The API wraps each failure with the layer name before it becomes
`extension.build_failed`, and records a durable failure entry through
`recordEnableFailure`. The error text therefore contains one of:

| Text fragment | Layer | Family |
| --- | --- | --- |
| (no layer prefix; a package error such as a template or CSS message) | page preflight | package-side |
| `component registry preflight failed` | component registry transition | stale state |
| `asset registry preflight failed` | asset registry transition | stale state |

Do not stop at the localized `extension.build_failed` copy — it is generic. The
copy now names both families and points here, but the layer name is in the error
text and the detail is in the extension event log.

## Step 2 — package-side failures (page preflight)

Reproducible without state:

```bash
cd <repository>/apps/api
go run ./cmd/sforum extension digest --write <package>
go run ./cmd/sforum extension validate <package>
go run ./cmd/sforum extension test <package>
```

Covers: manifest exactness (three-way template identity, `components[].propsSchema`
exact declarations, digest agreement), unloadable template or `theme.json.shell`,
invalid skin CSS, and path-signature conflicts (an `action: add` path already owned
by another extension). **Page coverage is not checked** — a one-page theme may
replace the 34-page default theme; the uncovered core pages fall back to their core
implementation.

## Step 3 — component registry: a theme runtime outside the transition fence

Symptom: the error names `component registry preflight failed` and the underlying
error is `ErrComponentRegistryConflict: active theme component runtime <id> is
outside the transition fence`.

Meaning: a theme runtime is registered that is neither the outgoing theme nor the
incoming one. Observed in
`TestComponentRegistryThemeTransitionRejectsThirdThemeOutsideTheFence`
(`app/Support/Extensions/component_registry_themes_test.go`): registering a third
theme and switching between two others is rejected, and **clearing that runtime
makes the same switch pass** — the package is fine.

Remedy:

1. List installed themes in the admin extension list and find the one that is
   neither the current active theme nor the one you are activating.
2. Uninstall or disable that theme through the admin flow. That republishes the
   lifecycle state; a direct restart is only enough when the drift was
   process-local (the component registry is an in-process view restored from
   durable lifecycle state at boot).

## Step 4 — asset registry: a theme publication matching neither theme

Symptom: the error names `asset registry preflight failed` and the underlying error
is `errAssetPublicationConflict`.

Meaning: at most one theme publication may exist, and it must match the outgoing
theme or the incoming one. A publication left behind by an incomplete
uninstall/disable matches neither, so **every** later switch fails. Observed in
`TestThemeAssetTransitionRejectsStaleDriftAndMultipleThemes/stale_third_theme_publication_blocks_every_switch`
(`app/Models/Extensions/theme_asset_publication_test.go`); clearing the stale
publication makes the same switch pass.

Remedy: the same as step 3 — clear the stale theme through the admin lifecycle
flow. This one is **durable** (it is restored from the lifecycle publication at
boot), so a restart alone does not clear it.

## Step 5 — verify the switch actually landed

```bash
curl -s http://127.0.0.1:8080/api/v1/site/active-theme/skin
curl -s "http://127.0.0.1:8080/api/v1/pages/resolve-path?path=/probe-site"
node tests/registry-page-shell-runtime-evidence.mjs
```

The two identity endpoints must agree on the active theme provider and digest, and
the runtime evidence script must still report exactly one visible navbar owner and
one footer for `site` pages and zero for `bare`.

## Limits

- Steps 3 and 4 were verified as **unit-level observations** of the rejection paths
  (the two tests above), not by seeding stale state in the dev instance and
  watching a live activation fail. No dev instance reproduction was performed.
- The remedy steps assume the admin lifecycle flow cleans up its own publications.
  If a stale publication survives an uninstall, that is a lifecycle defect to record
  rather than a runbook step to work around.
