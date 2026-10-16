# 2026-10-16 M4 Custom Guard Binding Verification Handoff

Scope: main task book **M4** — "custom authorization guard V1 production
binding". The milestone was re-scoped earlier from "implement" to "verify the
existing production binding, correct the stale disclosure, and resolve the open
intent questions". Nothing was implemented in this pass; M4's deliverable is
verified fact + corrected documentation + one decision request.

## Changed

- `docs/extensions/routes.md` — guard table replaced. `core.guard.raw_request`
  and custom guards are documented as **live**, with the exact gate list, the
  fail-closed failure modes, and a new subsection
  "Custom guards are live, and gated exactly like other plugin code".
- `docs/zh-CN/extensions/routes.md` — same correction in Chinese.
- `docs/en-US/extensions/routes.md` — summary line corrected.
- `docs/extensions/authoring-guide.md` — the `sforum-custom-content` fixture
  table no longer claims "custom/raw guard runtime is a later wave"; it now says
  the fixture stays on the closed guard set by choice.
- `knowledge/decisions/2026-10-16-theme-activation-exactness-and-open-questions.md`
  — carries open question 2 (R3).

No production code was changed for M4.

## Verified facts (traced, not inferred)

Production call chain for a route step whose guard is a plugin guard:

1. Assembly injects the evaluator:
   `apps/api/bootstrap/api_assembly_domains.go:618` —
   `PluginGuards: httpserver.NewRuntimePluginRouteGuardEvaluator(lifecycleStack.RuntimeManager, extensionGuardPolicy)`.
2. Authorizer dispatch:
   `apps/api/app/Support/Routes/core_guard_authorizer.go:219-235` —
   `validPluginGuardBinding(step)` then `a.PluginGuards.EvaluatePluginGuard(...)`;
   an unavailable evaluator or invalid binding returns
   `ErrCoreGuardEvaluatorUnavailable` (deny, no handler run).
3. Guard kind mapping: `apps/api/app/Support/Routes/request_authority.go:189-200`
   — `pluginGuard.Kind == "raw_request"` ⇒ `RequestGuardRawRequest`;
   `"custom"` ⇒ `RequestGuardCustom`.
4. Runtime gate: `apps/api/app/Http/plugin_guard_runtime.go:32-52` — requires an
   exact-artifact lookup (extension id, plugin type, `enabled`, exact version,
   exact package digest), `!SafeMode`, `CurrentArtifactTrusted` when trust is
   required **or** when raw authority is requested, then acquires a
   `RuntimeCallGuard` admission lease and invokes the real plugin code over
   Protocol V2.
5. Declaration acceptance: `app/Support/ExtensionManifest/v3_validate.go:105-107`
   accepts `kind: custom` and `kind: raw_request` with a `packageFiles[]`
   executable match.

Contradicted claims in the previous documentation:

- "`core.guard.raw_request` … Closed in the current host release
  (`ErrRouteGuardUnavailable`)" — false, see (2)/(3)/(4).
- "the executable guard runtime is a later wave; closed today" — false, see (1)
  and the integration tests below.

## D3 - allowed/denied coverage already exists (no new tests written)

Deliberately not creating synonymous tests:

| Side | Test |
| --- | --- |
| allowed (custom, sanitized headers) | `app/Http/plugin_guard_runtime_test.go:17` `TestRuntimePluginRouteGuardEvaluatorInvokesExactSanitizedGuard` |
| allowed + real subprocess | `app/Http/route_guard_production_chain_integration_test.go:44` `TestRouteCustomAndRawGuardsAcrossFiberManagerAndRealProtocolV2Process` |
| denied (typed failure evidence, trust drift, canceled context, raw failure) | `plugin_guard_runtime_test.go:53,117,155,175` |
| denied (real plugin guard returns 403) | `IntegrationTests/commerce_workflow_reference_plugin_integration_test.go:207-215` (guard denies on `deny=1`, on missing permission, and on contract mismatch) |
| raw authority negative | `route_guard_production_chain_integration_test.go:148` `TestRouteHostRouteGuardAuthorizerCannotMintRawOnFiber` |

## Open Questions (unchanged from M0, still need a human)

1. **R3 - `core.guard.raw_request` intent.** `plugin_guard_runtime.go:53-56`
   returns allow when the plugin guard binding is empty, and an empty binding is
   exactly what `core.guard.raw_request` requires
   (`Support/Routes/request_authority.go:189-191`). The same route step then
   resolves to `RequestAuthorityRaw`, under which
   `Support/Extensions/protocol_v2_request_authority.go:129-131` stops filtering
   credential headers, so `cookie` / `authorization` / `x-api-key` /
   `x-auth-token` reach the plugin handler. The adjacent comment ("Credential
   forwarding remains closed until raw-session authority is frozen") asserts the
   opposite of the traced behaviour. No decision record, ADR, or test states the
   intent. **Recorded as open question 2 in the decision note; needs a human.**
2. `challenge` outcome for plugin guards - no type, field, status, or proto
   message exists.
3. Guard `permissions[]` are validated and disclosed but never Host-enforced.
4. `HostRouteGuardAuthorizer` (legacy) is referenced only from tests; in
   production the Fiber stack uses the runtime evaluator, and
   `TestRouteHostRouteGuardAuthorizerCannotMintRawOnFiber` pins that the legacy
   helper cannot mint raw authority. Whether the legacy type is intentionally
   retained or orphaned is still unstated.

## Blocker status

M4's stated predecessor, the "V3 route residual inventory", was already
reconciled in M0 task 3
(`sessions/2026-10-16-m0-threat-and-inventory-report.md` section 4: V3 owns
distribution/release gating, this program owns the authoring/runtime contract).
**M4 is therefore not blocked by it**; the plan status table has been corrected.
M5 and M7 (both keyed on M4) are unblocked by M4's closure except for the
questions above, which are contract questions rather than implementation
blockers.
