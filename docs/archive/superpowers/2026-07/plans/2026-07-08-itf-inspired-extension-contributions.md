# Itf-Inspired Extension Contributions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. If the worker does not have these skills, treat each task as an independent checklist and stop for review after every task.

**Goal:** Borrow the useful part of old SForum's `Itf` mechanism: named, ordered, easy-to-consume extension contributions, while keeping new SForum's manifest validation, permission model, subprocess plugin boundary, and host-owned extension points.

**Architecture:** Do not recreate a global `Itf()` array or arbitrary hook system. Add a typed contribution registry owned by the extension platform: core declares contribution points, manifests declare ordered contribution items, the backend validates and resolves effective contributions, and each consumer decides how to interpret its own typed payload.

**Tech Stack:** Go Fiber v3, existing extension manifest/runtime services, PostgreSQL-backed extension metadata, modular OpenAPI, Nuxt 4/Vue 3/Nuxt UI, existing admin module registry, existing default Nuxt Layer theme.

---

## Legacy Itf Lessons

Old SForum's `Itf` is worth studying because it made extension authoring feel lightweight:

- A plugin could call `Itf()->add('some-point', 10, $data)`.
- Consumers could call `Itf()->get('some-point')` and decide how to merge or render items.
- Numeric IDs gave a simple ordering mechanism.
- The same shape served UI slots, editor configuration, menus, provider lists, shortcodes, middleware chains, and theme replacement.

The useful idea is **not** the PHP implementation. The useful idea is:

- explicit point name
- contribution id
- ordered entries
- consumer-owned interpretation
- simple plugin authoring

## What Not To Copy

- Do not add a global `Itf()` helper.
- Do not let plugins invent arbitrary point names at runtime.
- Do not accept closures, raw HTML, raw component paths, handler class names, or in-process executable callbacks as contribution data.
- Do not let contributions override core routes, bypass policy checks, or read raw session cookies.
- Do not use string key sorting such as `menu_10`; use an integer `order` and stable tie-breakers.
- Do not merge provider slots, events, filters, routes, and UI contributions into one untyped bucket.
- Do not allow uploaded themes to replace arbitrary Nuxt namespaces outside the existing theme activation pipeline.

## Borrowed Model For New SForum

Translate old `Itf(class, id, data)` into:

```json
{
  "contributions": [
    {
      "point": "forum.topic.actions",
      "id": "demo.bookmark",
      "order": 200,
      "label": {
        "zh-CN": "收藏",
        "en-US": "Bookmark"
      },
      "icon": "i-lucide-bookmark",
      "payload": {
        "type": "extensionRoute",
        "method": "POST",
        "path": "/topic-actions/bookmark"
      }
    }
  ]
}
```

Core owns the contribution-point catalog. A plugin may only contribute to known points, using a payload shape accepted by that point.

## First Slice

Implement the smallest useful slice:

- manifest schema and validation for declarative contributions
- backend contribution-point catalog
- backend effective contribution registry
- admin API and UI for inspecting available points and active contributions
- one concrete runtime consumer: `forum.topic.actions`

`forum.topic.actions` intentionally mirrors a valuable old Itf pattern: topic dropdown/action menu items. The new version only permits safe descriptors that call declared extension routes under `/api/v1/extensions/{extensionId}/*`; it does not render arbitrary plugin HTML.

## Non-Goals

- No arbitrary UI component injection.
- No extension-owned frontend bundle loading.
- No shortcode renderer.
- No middleware-chain replacement for core topic/comment creation.
- No payment-provider implementation.
- No marketplace trust/signature system.
- No migration of existing events or provider slots into this registry.
- No plugin route override.

## Must Read First

- `AGENTS.md`
- `knowledge/index.md`
- `knowledge/modules/extensions.md`
- `docs/extension-platform-v2.md`
- `knowledge/decisions/2026-07-06-core-framework-plugin-first-architecture.md`
- `knowledge/decisions/2026-07-06-extension-platform-v2.md`
- `knowledge/decisions/2026-07-06-plugin-event-extension-points.md`
- `docs/superpowers/plans/2026-07-06-extension-plugin-runtime.md`
- `docs/superpowers/plans/2026-07-07-extension-admin-manifest-v2.md`
- `apps/api/app/Support/ExtensionManifest/manifest.go`
- `apps/api/app/Models/Extensions/service.go`
- `apps/api/app/Http/Controllers/Extensions/routes.go`
- `apps/web/app/config/adminModules.ts`
- `apps/web/app/utils/adminExtensions.ts`
- `extensions/builtin/themes/sforum-default/layer/app/pages/t/[topicID]/[topicSlug].vue`

## Contribution Point Rules

- Every point has a stable id, owner module, kind, payload schema, default sorting rule, and visibility rule.
- Effective runtime contributions come only from enabled plugins and the active theme unless a point explicitly allows inactive management contributions.
- Disabled plugin contributions remain inspectable in admin but must not affect public or admin runtime behavior.
- Contributions are ordered by `order ASC`, then `extensionId ASC`, then `id ASC`.
- A single extension cannot register the same `point + id` twice.
- Cross-extension duplicate `id` values are allowed because the effective key is `extensionId + point + id`.
- Payload validation belongs to the owning module. Generic manifest validation should reject unsafe shapes; module validation should reject semantically invalid payloads.
- User-facing labels must support localization. At minimum, accept `zh-CN` and `en-US` maps and fall back to the extension name only for admin inspection.

## Permission And Security Model

- Contribution inspection requires `extension.manage`.
- Public contribution consumption must never grant authority by itself.
- Unsafe contribution actions must call extension routes, and route policy checks remain authoritative.
- A contribution may hide or show a button, but the backend route decides whether the action is allowed.
- Manifest validation rejects external URLs, core API paths, path traversal, unknown HTTP methods, unknown icons outside approved icon prefixes, and payload fields not accepted by the point schema.
- Event/filter/provider behavior remains separate from contribution descriptors.

## Suggested Data Shape

Backend types should stay close to this model unless existing code reveals a better local convention:

```go
type ManifestContribution struct {
	Point   string            `json:"point"`
	ID      string            `json:"id"`
	Order   int               `json:"order,omitempty"`
	Label   map[string]string `json:"label,omitempty"`
	Icon    string            `json:"icon,omitempty"`
	Payload json.RawMessage   `json:"payload,omitempty"`
}

type ContributionPointDefinition struct {
	ID          string
	Owner       string
	Kind        string
	Description string
}

type EffectiveContribution struct {
	ExtensionID string
	Point       string
	ID          string
	Order       int
	Label       map[string]string
	Icon        string
	Payload     json.RawMessage
}
```

For `forum.topic.actions`, use a narrower payload after validation:

```go
type TopicActionContributionPayload struct {
	Type    string `json:"type"`
	Method  string `json:"method"`
	Path    string `json:"path"`
	Confirm bool   `json:"confirm,omitempty"`
}
```

Allowed values:

- `type`: `extensionRoute`
- `method`: `POST`, `PUT`, `PATCH`, or `DELETE`
- `path`: relative extension route beginning with `/`, never `/api`, never `http://`, never `https://`, never containing `..`

## Implementation Tasks

### Task 0: Preflight And Scope Lock

**Files:**
- Read: files listed in "Must Read First"
- Inspect: `git status --short`

- [ ] Confirm there are unrelated dirty files before editing. Do not revert unrelated work.
- [ ] Confirm this work is a new typed contribution registry, not a port of old `Itf`.
- [ ] Confirm `events`, `filters`, `provider slots`, `routes`, `settings`, and `admin.pages` remain first-class manifest fields.
- [ ] Confirm the first runtime consumer is only `forum.topic.actions`.
- [ ] Suggested commit after Task 0 has no code changes: none.

### Task 1: Record The Architecture Decision

**Files:**
- Create: `knowledge/decisions/2026-07-08-itf-inspired-extension-contributions.md`
- Modify: `knowledge/modules/extensions.md`
- Optional modify: `docs/extension-platform-v2.md`

- [ ] Add a decision record explaining why SForum borrows the ordered contribution registry idea but rejects arbitrary in-process hooks.
- [ ] Document the separation between events/filter/provider slots and declarative contributions.
- [ ] Add the first accepted contribution point: `forum.topic.actions`.
- [ ] Update `knowledge/modules/extensions.md` with the contribution registry direction and security boundaries.
- [ ] If `docs/extension-platform-v2.md` has a roadmap section that mentions extension points, add a short subsection for typed contributions.
- [ ] Suggested commit: `docs: record typed extension contribution direction`

### Task 2: Manifest Schema And Validation

**Files:**
- Modify: `apps/api/app/Support/ExtensionManifest/manifest.go`
- Test: `apps/api/app/Support/ExtensionManifest/manifest_test.go`
- Modify: `contracts/openapi/schemas/extensions.yaml`

- [ ] Add failing tests for valid and invalid `contributions`.
- [ ] Valid test case: a plugin declares one `forum.topic.actions` contribution with localized label, approved icon, integer order, and safe relative extension route payload.
- [ ] Invalid test cases:
  - unknown contribution point
  - duplicate `point + id` in one manifest
  - missing `point`
  - missing `id`
  - external `payload.path`
  - payload path containing `..`
  - payload path targeting `/api/v1/...`
  - unknown HTTP method
  - unknown payload type
  - icon outside approved `i-lucide-` or `i-tabler-` prefixes
- [ ] Add `Manifest.Contributions []ManifestContribution`.
- [ ] Add normalization for contribution ids, point ids, icons, labels, order, and raw payload.
- [ ] Add validation against the core contribution-point catalog.
- [ ] Update OpenAPI manifest schemas to document `contributions`.
- [ ] Run `cd apps/api && go test ./app/Support/ExtensionManifest -count=1`.
- [ ] Run `ruby scripts/validate-openapi-refs.rb`.
- [ ] Suggested commit: `feat: validate extension contributions in manifests`

### Task 3: Backend Contribution Catalog And Registry

**Files:**
- Create: `apps/api/app/Support/Extensions/contributions.go`
- Test: `apps/api/app/Support/Extensions/contributions_test.go`
- Modify: `apps/api/app/Providers/extensions.go`
- Modify: `apps/api/bootstrap/app.go`

- [ ] Add `ContributionPointDefinition`, `EffectiveContribution`, and `ContributionRegistry`.
- [ ] Register built-in point definitions from code, not from plugin manifests.
- [ ] Include `forum.topic.actions` as the first built-in point.
- [ ] Add registry methods:
  - `Definitions() []ContributionPointDefinition`
  - `All() []EffectiveContribution`
  - `ByPoint(point string) []EffectiveContribution`
  - `ByExtension(extensionID string) []EffectiveContribution`
- [ ] Build effective contributions from installed extension manifests and runtime state.
- [ ] Include contributions from enabled plugins only.
- [ ] Include contributions from the active theme only if a future theme point declares support; the first slice can keep theme contributions inactive.
- [ ] Add tests for sorting, duplicate handling, disabled plugin exclusion, and stable tie-breaking.
- [ ] Wire the registry through the existing extension provider/bootstrap path.
- [ ] Run `cd apps/api && go test ./app/Support/Extensions ./app/Providers -count=1`.
- [ ] Suggested commit: `feat: add extension contribution registry`

### Task 4: Admin API For Contribution Inspection

**Files:**
- Modify: `apps/api/app/Http/Controllers/Extensions/routes.go`
- Modify: `apps/api/app/Http/Controllers/Extensions/controller.go`
- Test: `apps/api/app/Http/Controllers/Extensions/controller_test.go`
- Modify: `contracts/openapi/paths/extensions.yaml`
- Modify: `contracts/openapi/schemas/extensions.yaml`

- [ ] Add `GET /api/v1/admin/extensions/contribution-points`.
- [ ] Add `GET /api/v1/admin/extensions/contributions`.
- [ ] Require `extension.manage` for both routes.
- [ ] Return contribution point definitions with owner, kind, description, and supported payload type.
- [ ] Return active/effective contributions with extension id, contribution id, point, order, label, icon, and sanitized payload summary.
- [ ] Add tests for allowed and denied access.
- [ ] Add OpenAPI path items, response schemas, and permission notes.
- [ ] Run `cd apps/api && go test ./app/Http/Controllers/Extensions -count=1`.
- [ ] Run `ruby scripts/validate-openapi-refs.rb`.
- [ ] Suggested commit: `feat: expose extension contribution inspection api`

### Task 5: Admin UI For Extension Points

**Files:**
- Modify: `apps/web/app/config/adminModules.ts`
- Modify: `apps/web/app/utils/adminExtensions.ts`
- Test: `apps/web/tests/adminExtensions.test.ts`
- Create: `apps/web/app/pages/admin/extensions/contributions.vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`

- [ ] Add frontend types for contribution points and effective contributions.
- [ ] Add API helpers for `contribution-points` and `contributions`.
- [ ] Add a new Extensions submenu page named "Extension Points" / "扩展点".
- [ ] Show known contribution points, active contributions, source extension, order, and payload summary.
- [ ] Keep the page read-only in this first slice.
- [ ] Add empty, loading, and error states.
- [ ] Add tests for contribution sorting and display helpers.
- [ ] Run `cd apps/web && bun test`.
- [ ] Run `cd apps/web && bun run typecheck`.
- [ ] Suggested commit: `feat: show extension contribution points in admin`

### Task 6: First Runtime Consumer - Topic Action Contributions

**Files:**
- Modify: `apps/api/app/Models/Forum/types.go`
- Modify: `apps/api/app/Models/Forum/service.go`
- Modify: `apps/api/app/Http/Controllers/Forum/controller.go`
- Test: `apps/api/app/Models/Forum/service_test.go`
- Test: `apps/api/app/Http/Controllers/Forum/controller_test.go`
- Modify: `contracts/openapi/paths/forum.yaml`
- Modify: `contracts/openapi/schemas/forum.yaml`
- Modify: `apps/web/app/composables/useForumApi.ts`
- Modify: `apps/web/app/utils/forumTaxonomy.ts`
- Test: `apps/web/tests/forumTaxonomy.test.ts`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/t/[topicID]/[topicSlug].vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`

- [ ] Extend topic detail response with `extensionActions`.
- [ ] Resolve only `forum.topic.actions` contributions from enabled plugins.
- [ ] Convert each action into a public-safe descriptor: localized label, icon, HTTP method, extension route URL, and optional confirmation flag.
- [ ] Do not include raw payload fields that the topic page does not need.
- [ ] Keep backend policy authoritative: executing the action still calls the extension route proxy, which enforces route access.
- [ ] Add tests showing disabled plugin actions are omitted.
- [ ] Add tests showing invalid payloads cannot appear in topic detail.
- [ ] Update OpenAPI topic detail schema.
- [ ] Render actions in the topic detail action menu with icon buttons or menu rows, using existing icon integration.
- [ ] Add frontend helper tests for generated extension action URLs.
- [ ] Run `cd apps/api && go test ./app/Models/Forum ./app/Http/Controllers/Forum -count=1`.
- [ ] Run `cd apps/web && bun test`.
- [ ] Run `cd apps/web && bun run typecheck`.
- [ ] Run `ruby scripts/validate-openapi-refs.rb`.
- [ ] Suggested commit: `feat: render safe topic action contributions`

### Task 7: Developer Ergonomics And Scaffold Examples

**Files:**
- Modify: `apps/api/cmd/sforum/generator.go`
- Test: `apps/api/cmd/sforum/generator_test.go`
- Modify: `docs/extension-platform-v2.md`
- Optional create: `docs/extensions/contributions.md`

- [ ] Add a commented manifest example for `contributions` to generated plugin scaffolds.
- [ ] Keep generated examples disabled or clearly demo-only so new plugins do not accidentally expose runtime actions.
- [ ] Add documentation explaining contribution point ids, payload restrictions, ordering, and security expectations.
- [ ] Document that plugin authors should prefer provider slots, events, routes, or settings when those are the correct contract.
- [ ] Run `cd apps/api && go test ./cmd/sforum -count=1`.
- [ ] Suggested commit: `docs: document extension contribution authoring`

### Task 8: Verification And Handoff

**Files:**
- Modify: `knowledge/index.md` if navigation/status changes
- Create: `knowledge/sessions/2026-07-08-itf-inspired-extension-contributions.md`

- [ ] Run `cd apps/api && go test ./...`.
- [ ] Run `cd apps/web && bun test`.
- [ ] Run `cd apps/web && bun run typecheck`.
- [ ] Run `ruby scripts/validate-openapi-refs.rb`.
- [ ] Run `./scripts/test.sh` if this lands as a full feature slice.
- [ ] Add a session handoff with changed files, decisions, next steps, and open questions.
- [ ] Suggested commit: `chore: hand off extension contribution registry work`

## Acceptance Criteria

- A plugin can declare a valid `forum.topic.actions` contribution in `sforum.extension.json`.
- Invalid contribution declarations are rejected before activation.
- Admin users with `extension.manage` can inspect known contribution points and active contributions.
- Disabled plugins do not affect topic detail actions.
- Topic detail can expose safe extension action descriptors without arbitrary HTML or frontend asset loading.
- Executing a topic action still goes through the extension route proxy and its policy checks.
- OpenAPI documents all new manifest, admin, and forum response shapes.
- Tests cover allowed and denied admin access, manifest validation, ordering, disabled-plugin exclusion, and topic detail action resolution.

## Later Slices

- Add `forum.topic.sidebar.blocks` only after SForum has a safe host-rendered block descriptor model.
- Add editor toolbar or sanitizer contribution points only after the Tiptap/editor API is deliberately modeled.
- Add payment provider UI contributions after core owns provider-neutral payment intents and transaction lifecycle.
- Add extension-owned frontend assets only behind CSP, asset validation, version compatibility, and clear rollback behavior.
