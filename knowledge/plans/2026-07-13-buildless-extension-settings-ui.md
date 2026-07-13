# Buildless Extension Settings UI — Implementation Task Book

Status: **completed (P0–P6)**
Date: 2026-07-13
Audience: a new AI/human session implementing the complete track

Decision:
`knowledge/decisions/2026-07-13-buildless-extension-settings-ui.md`

Primary goal:

> Plugins and themes share a rich settings system. Schema and action-based
> settings never rebuild SForum. Complex component settings are author-prebuilt
> and dynamically loaded after explicit digest-bound trust, so the operator
> still does not rebuild the host.

## Implemented contract adjustments

The completed implementation follows the target architecture with these
evidence-based adjustments to the original proposal:

- Explicit Component UI confirmation uses an actor-bound, one-use challenge
  that expires after five minutes. The challenge proves an intentional admin
  action; the persisted grant remains bound to extension id, version, API
  version, component id, and `adminFrontendDigest`.
- JavaScript is imported only from the authenticated, same-origin immutable
  digest endpoint. Optional CSS is fetched with credentials from the matching
  digest endpoint, MIME-checked, and injected as a scoped `<style>` so cookie
  authentication does not depend on browser stylesheet credential behavior.
- The CLI emits one deterministic versioned Settings Document for complex
  settings. Manifest `includes` and legacy array/directory shards remain fully
  supported, but the new template avoids needless fragmentation.
- The protected default theme deliberately remains Schema-only. Builtin-source
  Component UI policy is covered by source-trust tests using the same runtime;
  a separate uploaded Schema theme fixture proves public theme activation and
  admin rendering stay independent.
- The host does not add arbitrary script origins to CSP. The loader constructs
  only its own same-origin digest URL, and the asset endpoint enforces auth,
  allowlisted asset names, exact digest bytes, MIME, `nosniff`, and
  `Cross-Origin-Resource-Policy: same-origin`.

## How to use this task book in a new conversation

1. Read `AGENTS.md`, `knowledge/index.md`, the decision above, this entire task
   book, and the current `knowledge/modules/extensions.md` and
   `knowledge/modules/frontend.md` notes.
2. Inspect the current working tree before editing. Preserve unrelated and
   user-authored changes, especially the active runtime-theme/dev-compose work.
3. Implement phases in order. Do not jump to micro-frontends before Schema UI
   is complete and tested.
4. Use small, reversible commits only when the user explicitly authorizes
   commits. Otherwise leave a clean, reviewed working diff grouped by phase.
5. After every completed phase, update this checklist and add or extend a
   `knowledge/sessions/` handoff.
6. Do not stop after scaffolding while a safe, in-scope implementation step
   remains. The requested endpoint is the complete plan, including tests and
   documentation, unless a concrete blocker is found.

## Current baseline to preserve

- Extension settings live in `extension_settings`; defaults come from the
  merged package manifest.
- Settings GET/PUT/reset already handle localization, recommended values,
  masked secrets, preserve-on-empty, audit, and plugin restart/rollback.
- The admin dynamic settings page already renders a generic host form.
- Trusted Vue components use `frontend.admin`, contribution points, digest
  grants, a static admin registry, Web Release, client-only rendering, and
  browser-session quarantine.
- SMTP is the reference plugin with a custom settings Vue page.
- Public themes use runtime Page Registry + L0/L1 and activate without Nuxt
  rebuild.
- Current Web Release planning excludes themes and includes trusted enabled
  plugin admin frontends.
- `bun run dev` is plain Nuxt. `bun run dev:compose` composes builtin trusted
  admin frontends from source with Vite HMR.

Do not regress these behaviors while introducing the new path.

## Architecture target

```text
                    Extension settings document
                                │
             ┌──────────────────┼──────────────────┐
             ▼                  ▼                  ▼
         Schema UI       Schema + Actions      Component UI
             │                  │                  │
       host renderer       host renderer       prebuilt module
             │                  │                  │
       no author JS        no author UI JS      explicit trust
             │                  │                  │
       no host build        no host build        no host build
```

Runtime ownership:

```text
manifest presentation ──► API normalized page model ──► host renderer
stored values          ──► API validation/secrets    ──► extension_settings
declared action        ──► host action executor      ──► provider/RPC adapter
prebuilt component     ──► digest trust + asset URL  ──► client mount bridge
```

## Cross-cutting rules

1. Reuse existing manifest loading, settings storage, localization, trust
   grants, permission helpers, Toasts, and quarantine before adding new
   infrastructure.
2. API policy is authoritative. UI hiding or disabled buttons are UX only.
3. Old array-shaped settings manifests remain valid.
4. Invalid optional presentation falls back safely; invalid field/storage
   semantics fail package validation.
5. Do not introduce arbitrary action URLs or arbitrary remote component URLs.
6. Themes and plugins use the same renderer. Type-specific restrictions belong
   in validation/lifecycle policy, not duplicate pages.
7. Public theme settings consumption and Page Registry activation remain
   independent of admin UI rendering.
8. Prefer project-native Nuxt UI controls. Before adding a form/layout
   dependency, document why existing components cannot support the contract.
9. Keep files cohesive; split renderer fields/layout/actions into focused
   components rather than growing another giant settings page.
10. Add useful Chinese comments for non-obvious trust, fallback, secret, and
    lifecycle behavior.

---

## P0 — Baseline reconciliation and contract inventory

Goal: remove ambiguity before changing behavior.

### Tasks

- [x] Inspect the merged extension manifest types and loaders, including
      multi-file `includes.settings` behavior and stored canonical manifests.
- [x] Inventory all current setting field types and presentation fields.
- [x] Inventory current settings permission/lifecycle checks for:
      - enabled plugin
      - disabled/installed plugin
      - active theme
      - inactive installed theme
- [x] Inventory the SMTP custom settings component and identify which parts
      become Schema UI, Actions, or remain genuinely component-only.
- [x] Inventory existing provider `Probe` implementations and admin endpoints;
      reuse them rather than inventing parallel probe protocols.
- [x] Inventory trusted frontend grants, package digests, registry loading,
      component error isolation, and Web Release composition hashing.
- [x] Resolve stale knowledge statements about theme `frontend.admin` against
      current code. Document current behavior before implementing the target.
- [x] Add focused contract fixtures for old array settings, new document
      settings, tabs, actions, and component mode.

### Exit criteria

- A short inventory section is added to the session handoff.
- Tests/fixtures identify every existing field type and compatibility shape.
- No runtime behavior changes yet.

### Suggested commit

`test(extensions): capture settings UI compatibility baseline`

---

## P1 — Versioned settings document and backend normalization

Goal: establish one backward-compatible internal settings contract.

### Proposed external shape

```json
{
  "schemaVersion": 1,
  "ui": {
    "mode": "schema",
    "layout": "tabs",
    "tabs": [
      {
        "id": "home",
        "label": { "zh-CN": "首页", "en-US": "Home" },
        "groups": ["homeCopy"]
      }
    ],
    "callouts": []
  },
  "fields": [],
  "actions": []
}
```

### Tasks

- [x] Add focused manifest types for:
      - `SettingsDocument`
      - `SettingsUI`
      - `SettingsTab`
      - `SettingsCallout`
      - `SettingsColumn` or a minimal equivalent
      - `SettingsAction`
- [x] Normalize legacy `[]SettingDefinition` to schemaVersion 1 with
      `mode=schema`, `layout=form`.
- [x] Keep canonical storage deterministic so package digest and stored
      manifests do not change nondeterministically.
- [x] Validate:
      - supported schema version
      - unique tab/action ids
      - group references exist
      - a group is not ambiguously assigned to multiple tabs unless the
        contract explicitly permits it
      - component id exists when `mode=component`
      - component mode requires a schema fallback (`fields` may be empty only
        when explicitly accepted by product policy)
      - themes stay within settings UI capabilities
      - actions use catalogued executor kinds, never URLs
- [x] Reject dual sources when both `ui.mode=component` and a conflicting
      legacy `admin.extension.settings.page` contribution are declared.
- [x] Add a compatibility normalizer for existing settings-page contributions
      during migration. Define one internal source of truth after normalization.
- [x] Resolve localized UI labels/callouts using the same locale fallback rules
      as current setting labels.
- [x] Update `sforum extension validate` and `extension test` output to include
      renderer mode, tabs, actions, and component identity.
- [x] Update scaffolding so new themes default to Schema UI and complex plugin
      scaffolds can opt into settings actions/component mode.

### API model

Extend the settings response rather than exposing raw manifest JSON:

```text
AdminExtensionSettingsPage
├── extensionId/type/status
├── renderer { mode, layout, component?, fallback }
├── tabs/callouts/presentation
├── items
├── values
├── secrets
└── actions
```

The server resolves localization and lifecycle availability before returning
the model.

### Tests

- [x] Legacy array load/validate/canonicalize.
- [x] New document load/validate/canonicalize.
- [x] Includes path and directory shard compatibility.
- [x] Duplicate ids and unknown group references rejected.
- [x] Unsupported schema versions rejected.
- [x] Component reference and conflict tests.
- [x] Theme restriction tests.
- [x] Locale fallback tests.

### Exit criteria

- All existing extension fixtures still validate.
- Old packages receive the same generic form behavior.
- The API can return a localized renderer model without frontend interpretation
  of raw manifest structures.

### Suggested commits

1. `feat(extensions): version extension settings documents`
2. `feat(extensions): validate settings presentation and actions`
3. `test(extensions): cover legacy and versioned settings manifests`

---

## P2 — `SFExtensionSettingsRenderer` and Schema UI

Goal: rich plugin/theme settings without extension frontend code.

### Component structure

```text
components/extensions/settings/
├── SFExtensionSettingsRenderer.vue
├── SFExtensionSettingsTabs.vue
├── SFExtensionSettingsGroup.vue
├── SFExtensionSettingsField.vue
├── SFExtensionSettingsCallout.vue
├── SFExtensionSettingsActions.vue
├── SFExtensionSettingsFooter.vue
└── SFTrustedSettingsComponent.vue
```

Names may follow the repository’s exact component auto-import conventions, but
responsibilities should stay split.

### Tasks

- [x] Extract the existing generic settings form from the dynamic extension
      page into the renderer and focused subcomponents.
- [x] Preserve all current controls, validation messages, loading states,
      recommended defaults, secret preservation, reset, save, Toasts, and
      theme-aware success styling.
- [x] Render `layout=form` identically to the current fallback.
- [x] Implement `layout=tabs` with stable route-independent local state.
- [x] Render group headings, optional columns, and callouts using Nuxt UI/SF
      components.
- [x] Keep error messages beside fields; non-error action success may use
      auto-dismiss Toasts under the repository rules.
- [x] Add visible renderer badges:
      - 声明式
      - 自定义组件
      - 通用兼容
- [x] Distinguish settings data state from renderer state:
      - settings saved immediately
      - component unavailable, fallback active
      - component grant required
      - component changed, reconfirmation required
- [x] On invalid/missing optional UI presentation, log a diagnostic and render
      the linear fallback instead of breaking the page.
- [x] Ensure disabled/inactive extension settings access follows the P3 policy
      without accidentally mounting extension code.

### Default theme migration

- [x] Express the default theme’s home copy, rails, navigation, layout, and
      other current fields through tabs/groups/callouts in the settings
      document.
- [x] Do not restore a dev-only theme Vue page that differs from production.
- [x] Delete or clearly retire stale unused default-theme settings SFC files
      only after confirming no current manifest/runtime reference remains.
- [x] Verify saved theme values still apply immediately through
      `GET /site/active-theme/settings`.
- [x] Add a second runtime theme fixture proving uploaded themes receive the
      same renderer with no Web Release.

### Tests

- [x] Component tests for form and tabs layouts.
- [x] Group ordering, empty groups, callouts, secret fields, save/reset.
- [x] Fallback on malformed optional presentation.
- [x] Theme and plugin page parity.
- [x] Browser/manual validation in Chinese and English, dark and light admin
      appearance, narrow and wide layouts.
- [x] Confirm no Web Release is queued when changing settings values or
      installing/activating a schema-only theme.

### Exit criteria

- Default theme has a polished multi-tab settings page in both development and
  production without `frontend.admin`.
- Existing plugins without UI metadata look no worse than before.
- Schema-only plugin/theme packages never require an admin frontend build.

### Suggested commits

1. `refactor(web): extract extension settings renderer`
2. `feat(web): render tabbed extension settings schemas`
3. `feat(themes): describe default settings UI without admin code`
4. `test(web): cover extension settings renderer modes`

---

## P3 — Schema Actions and configure-before-enable lifecycle

Goal: provider/service plugins can offer operational settings without custom UI.

### Action contract

Actions are descriptors, not browser code:

```json
{
  "id": "probe",
  "kind": "provider_probe",
  "label": { "zh-CN": "测试连接", "en-US": "Test connection" },
  "placement": "footer",
  "useDraftValues": true
}
```

Exact kinds must be chosen after the P0 inventory. Prefer a small host catalog
and adapters over a single arbitrary RPC method name.

### Tasks

- [x] Add host-owned endpoint:
      `POST /api/v1/admin/extensions/{id}/settings/actions/{actionId}`.
- [x] Add OpenAPI path, request/response schemas, errors, lifecycle notes, and
      permission/security description.
- [x] Implement an action catalog/executor that validates:
      - action belongs to this extension/version
      - actor may manage the extension/settings/provider
      - requested draft keys are declared settings
      - secrets use explicit preserve/new-value semantics
      - action kind is supported
      - lifecycle permits execution
      - timeout and response size limits
- [x] Reuse current mail/storage/provider Probe paths through adapters.
- [x] Return host-owned structured results (`success`, `reason`, `message`,
      optional details/suggestions), never plugin-provided HTML.
- [x] Audit action id, extension id, actor, success/failure, and duration;
      never log credentials or draft secret values.
- [x] Render action loading, success details, blocking field errors, and error
      Toast/alert behavior according to project conventions.
- [x] Allow installed/disabled plugins and inactive themes to read and save
      declared settings without executing extension code.
- [x] Ensure normal extension routes/events/jobs/providers are not registered
      merely to edit settings.
- [x] Initially disable code-requiring actions while the plugin is disabled if
      safe ephemeral execution is not already supported.
- [x] If implementing pre-enable probes, create a restricted short-lived
      runtime that exposes only the approved probe method; it must not register
      routes, events, jobs, schedules, or providers.
- [x] Update enable UX to show configuration completeness and available probe
      results without turning optional recommendations into hard requirements.

### SMTP migration

- [x] Move ordinary SMTP fields, help, recommended defaults, and connection
      test into Schema + Actions.
- [x] Compare the remaining SFC behavior. If nothing genuinely component-only
      remains, retire the SMTP settings SFC from the active manifest.
- [x] Keep SMTP as the authoring-guide reference for provider settings and
      probe actions.

### Tests

- [x] Allowed and denied action permission paths.
- [x] Unknown action, unknown fields, oversized input, timeout, plugin failure.
- [x] Secret draft values never appear in logs/responses.
- [x] Settings editable while disabled; plugin runtime stays stopped.
- [x] Enabled plugin settings restart/rollback remains correct.
- [x] Provider probe adapter tests.
- [x] OpenAPI reference validation.

### Exit criteria

- SMTP-level settings UX works without a custom Vue settings page.
- Operators can install → configure → enable, and can probe before enable only
  where the executor safely supports it.
- Schema actions do not expand frontend trust.

### Suggested commits

1. `feat(extensions): execute declared settings actions`
2. `feat(extensions): allow safe configuration before enablement`
3. `feat(mail): migrate smtp settings to schema actions`
4. `test(extensions): cover settings action policy and secrets`
5. `docs(extensions): document schema actions and provider probes`

---

## P4 — Stop unnecessary Web Releases

Goal: current trusted Vue compatibility releases occur only when their actual
admin frontend changes.

### Tasks

- [x] Introduce deterministic `adminFrontendDigest` computed only from the
      frontend admin contract and files that affect the built admin component:
      component sources, styles, locales, component map, relevant
      contributions, lockfile/dependency metadata, and API version.
- [x] Do not include backend binaries, public theme assets/templates, ordinary
      setting values, or unrelated manifest fields in that digest.
- [x] Bind frontend trust and Web Release composition to the dedicated digest
      while retaining package identity/version checks where required.
- [x] Migrate existing grants safely. Do not silently broaden an uploaded
      package’s trust during migration.
- [x] Reuse an active/ready artifact when the full admin composition hash is
      unchanged.
- [x] Ensure plugin enable/disable without an admin component does not queue a
      Web Release.
- [x] Ensure a backend/settings-only upgrade with unchanged trusted admin
      frontend does not rebuild Nuxt.
- [x] Change Web Release typecheck policy from the current always-run behavior
      to an explicit mode:
      - `off`
      - `report` (non-blocking/background where practical)
      - `block`
- [x] Keep CI and `./scripts/test.sh` typecheck mandatory.
- [x] Preserve immutable artifact verification, activation, rollback, and
      coordinator lifecycle semantics.
- [x] Update admin copy so operators can see why a release was reused, skipped,
      or required.

### Tests

- [x] Digest changes for component/locale/lock changes.
- [x] Digest unchanged for backend/settings/public-theme-only changes.
- [x] Composition reuse and concurrent request tests.
- [x] Enable/disable lifecycle tests with and without admin frontend.
- [x] Grant invalidation tests.
- [x] Typecheck policy tests for off/report/block.

### Exit criteria

- Unrelated extension changes no longer cause a host build.
- Existing trusted Vue extensions continue to work through Web Release.
- Release reuse is deterministic and auditable.

### Suggested commits

1. `feat(extensions): fingerprint admin frontends independently`
2. `feat(web-release): reuse unchanged admin compositions`
3. `feat(web-release): separate release and ci typecheck policy`
4. `test(web-release): cover digest and reuse behavior`

---

## P5 — Prebuilt Component UI runtime

Goal: complex settings UI loads without rebuilding Nuxt/Nitro.

### Public author contract

Prefer a framework-neutral mount API over raw Vue SFC loading:

```js
export const apiVersion = 1

export function mount(target, bridge) {
  // render into target
  return () => { /* cleanup */ }
}
```

The exact signature must be documented and contract-tested before enabling
uploaded packages.

### Tasks

- [x] Define Admin Micro-frontend API v1 in `@sforum/admin-sdk` or a focused
      companion package:
      - settings page context
      - read/update draft values
      - save/reset
      - namespaced extension API request
      - Toast
      - locale/translation
      - navigation
      - appearance tokens
      - cleanup lifecycle
- [x] Define manifest fields for a self-contained prebuilt entry and optional
      CSS. Files must remain under a safe package-relative admin dist root.
- [x] Validate extension, MIME type, size limits, path containment, and API
      version. Reject source SFCs as runtime entries.
- [x] Compute the immutable admin frontend digest during install/upgrade.
- [x] Store/serve assets from a digest-addressed same-package location through
      an authenticated admin asset endpoint or an equally safe static mapping.
- [x] Set immutable cache headers and prevent path traversal/content sniffing.
- [x] Do not allow arbitrary remote asset URLs.
- [x] Add explicit trust UX to install/enable/detail:
      - identify extension/author/version
      - explain that code runs with administrator browser authority
      - require intentional confirmation (verification code/re-auth pattern)
      - allow “use Schema UI instead”
- [x] Reuse or evolve existing frontend trust grants so trust binds extension,
      version, API version, component identity, and admin frontend digest.
- [x] Package update with a changed digest automatically falls back to Schema
      UI until reconfirmed.
- [x] Add client-only dynamic module loading from the digest URL.
- [x] Mount inside a dedicated boundary; pass only the versioned bridge.
- [x] Reuse error boundary, failure counting, and browser-session quarantine.
- [x] On missing grant, failed import, contract mismatch, mount failure, or
      quarantine, render Schema fallback and a clear operator notice.
- [x] Ensure component trust is independent from public theme activation.
- [x] Permit builtin component UI through protected-source policy while still
      using the same runtime contract and fallback.
- [x] Add a small fixture micro-frontend that proves install → confirm → load →
      update digest → reconfirm → fallback behavior.

### Security review checklist

- [x] Component code is accurately described as fully trusted, not sandboxed.
- [x] Confirmation tokens cannot be replayed across actor, extension, version,
      or digest.
- [x] Asset endpoint cannot serve arbitrary package files.
- [x] The loader accepts only the host-generated same-origin immutable digest
      path; asset responses enforce authentication, allowlisting, MIME,
      `nosniff`, and same-origin resource policy.
- [x] API requests from the bridge remain namespaced and backend-authorized.
- [x] No Cookie/Authorization value is directly exposed as bridge data.
- [x] Secrets are not passed to the component unless present as administrator
      draft input; stored secrets remain masked/preserved.
- [x] One broken component cannot break navigation or other admin pages.

### Tests

- [x] Manifest/path/digest validation.
- [x] Asset authorization, cache, traversal, MIME, and missing file tests.
- [x] Trust grant, revoke, update invalidation, and confirmation replay tests.
- [x] Browser test for successful mount and cleanup.
- [x] Browser verification covers no grant, successful mount/cleanup, revoke,
      and Schema fallback; import/mount/quarantine failure paths are covered by
      focused runtime contract tests.
- [x] Uploaded plugin fixture, uploaded Schema-theme fixture, and builtin-source
      component policy fixture.
- [x] Confirm no Web Release/Nuxt build row is created for prebuilt components.

### Exit criteria

- A complex plugin settings component can be installed and enabled without a
  host build.
- An administrator can decline trust and still configure the extension through
  Schema UI.
- Upgrading component bytes requires reconfirmation but not rebuilding SForum.

### Suggested commits

1. `feat(admin-sdk): define admin micro-frontend bridge v1`
2. `feat(extensions): validate and serve prebuilt admin components`
3. `feat(extensions): confirm digest-bound component trust`
4. `feat(web): load trusted settings components at runtime`
5. `test(extensions): cover component trust and fallback lifecycle`
6. `docs(extensions): publish prebuilt component authoring guide`

---

## P6 — Migration, deprecation, and complete operator loop

Goal: make the buildless path the normal documented product behavior.

### Tasks

- [x] Update plugin/theme scaffolding:
      - Schema UI by default
      - optional actions
      - optional prebuilt component template
- [x] Update authoring guide with three complete references:
      1. theme tabs/groups, no JS
      2. provider plugin Schema + Probe action
      3. trusted prebuilt complex component
- [x] Update trusted-admin documentation to distinguish legacy Vue Web Release
      from Admin Micro-frontend API v1.
- [x] Update extension list/detail/settings UI with renderer/trust/build status.
- [x] Add one-click restore of Schema fallback without deleting settings,
      secrets, backend state, or extension package.
- [x] Update marketplace/install review metadata so operators can see:
      - no author JS
      - actions only
      - trusted component included
- [x] Mark legacy settings-page contribution + full Web Release path deprecated
      for new packages after parity; do not remove until a documented window
      and builtin/reference migration are complete.
- [x] Remove stale default-theme custom settings SFC/contribution if still
      present and unused.
- [x] Reconcile all knowledge notes and decisions that describe themes entering
      Web Release.
- [x] Add final session handoff with migration compatibility and remaining
      optional cleanup.

### Exit criteria

- New theme: upload → settings tabs → activate → save, with no host build.
- New provider plugin: install → configure → probe → enable, with no host build.
- New complex plugin: install → confirm → dynamically load, with no host build.
- Declining component trust leaves a functional Schema fallback.
- Host build remains for SForum releases and the temporary legacy compatibility
  path only.

### Suggested commits

1. `feat(cli): scaffold buildless extension settings`
2. `docs(extensions): document all settings renderer modes`
3. `chore(web-release): deprecate settings-only host rebuilds`
4. `test(extensions): validate complete buildless operator loops`

---

## OpenAPI and contract workflow

When P1/P3/P5 add or change endpoints:

1. Put extension paths in the existing modular extension path contract.
2. Put reusable settings page/action/component schemas in the extension schema
   contract.
3. Document permission and lifecycle behavior, including disabled extensions.
4. Update frontend API types/utilities that depend on the shape.
5. Run:

```bash
ruby scripts/validate-openapi-refs.rb
```

## Verification matrix

### Backend

```bash
cd apps/api && go test ./...
cd apps/api && go build ./...
```

### Frontend

```bash
cd apps/web && bun test
cd apps/web && bun run typecheck
cd apps/web && bun run build
```

Use the exact available Bun test script/target after inspecting
`apps/web/package.json`; do not invent a missing command.

### Full repository gate

```bash
./scripts/test.sh
```

### Manual/browser scenarios

1. Default theme multi-tab settings in Chinese and English.
2. Uploaded schema-only theme with no frontend package.
3. Disabled plugin settings editable while runtime remains stopped.
4. SMTP/provider probe success and failure.
5. Component trust accepted, declined, revoked, and invalidated on update.
6. Component import/mount failure falls back without breaking admin.
7. Confirm no Web Release for schema/actions/prebuilt component paths.

Network-dependent dependency commands must use the repository proxy from
`AGENTS.md`. Prefer no new dependency unless the implementation cannot remain
clear and maintainable with existing Nuxt UI, Go, and browser APIs.

## Definition of done

The track is complete only when all are true:

- [x] Plugins and themes share the same versioned settings document.
- [x] Legacy settings arrays remain compatible.
- [x] Tabs/groups/callouts render through the host in dev and production.
- [x] Default theme no longer needs custom admin frontend code for rich settings.
- [x] Provider-style actions work through host policy and structured results.
- [x] Settings can be safely configured before enablement.
- [x] Unchanged admin frontend changes do not rebuild the host.
- [x] Prebuilt component UI dynamically loads after explicit digest trust.
- [x] Declined/failed component UI safely falls back to Schema UI.
- [x] Theme activation remains independent of Web Release.
- [x] OpenAPI, SDK, scaffolding, authoring docs, catalogs/fixtures, module notes,
      decisions, and handoff are updated.
- [x] Relevant unit, integration, browser, build, typecheck, OpenAPI, and full
      repository gates pass, or any unrelated pre-existing failure is clearly
      recorded with evidence.

## Rollback strategy

- P1/P2: legacy settings arrays and linear renderer remain the fallback.
- P3: disable action exposure; settings storage remains usable.
- P4: restore package-digest Web Release planning if dedicated digest reuse has
  a correctness issue.
- P5: disable runtime component loading and use Schema UI or legacy Web Release.
- Never delete stored settings or secrets when rolling back presentation code.
- Never couple rollback of admin UI to public theme activation.
