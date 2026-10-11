# Decision: AI Tool Contribution (M2) — Registration, Exposure, Reserved Slot

## Status

Accepted (2026-10-11). Implemented in the same change as
`plans/2026-10-11-ai-plugin-host-api-m1.md` M2.

## Context

M1 put plugin-initiated AI calls inside the Host gateway (accounting, gates,
trace). M2 lets a plugin hand the Host a tool that the completion loop may call,
so that a plugin capability reaches the model through the same orchestration,
step budget, gates, truncation, and failure posture as a Core tool.

The working tree already carried most of the M2 surface — manifest
`aiTools[]` (types, JSON schema, validation, normalization), the `Support/AI`
declaration contract and plugin registry, the `sforum.ai.tool` transport in
`app/Support/Extensions`, an SDK registry, and a testdata manifest fixture — but
three things were missing: nothing called the registration path, the SDK server
never routed the reserved slot, and the change did not pass the architecture
gate or the catalog-docs gate.

## Decisions

### 1. Registered is not visible (fail-closed exposure)

Every declared tool of every enabled plugin is registered into the same
`ai.ToolRegistry` that holds the Core built-in tools, so "which AI tools exist on
this site" has exactly one enumerable answer. Visibility is a separate decision
owned by the running purpose: the model only sees the names in
`RunInput.ToolNames`, and plugin tools are not added to any Core purpose's
allowlist. `forum.reply` therefore keeps exactly its six Core tools, and this
milestone changes no model-visible behavior at all.

Consequence: a shipped tool contribution cannot silently widen what the reply
bot can call. Widening it requires an explicit operator/purpose decision, which
is deliberately deferred (see Open Questions).

### 2. Registration happens at API assembly

`registerPluginAITools` runs once in `wireAPIDomainServices`, from the same
extension set the runtime was reconciled against, and is skipped entirely in
Safe Mode or recovery-only mode (there the plugin runtimes are not running, and
registering their tools would produce "visible but uncallable" entries).

Consequence, accepted: enabling or disabling a plugin changes the AI tool list
only after an API restart. Synchronizing the tool set with the runtime full-set
publication is a follow-up work item, not part of M2.

### 3. Contributing a tool does not require `ai.invoke`

`ai.invoke` (M1) is the capability to *spend site AI budget*: it is checked on
`sforum.ai.complete@1`, where the caller's purpose and cost class are resolved
against site configuration and the caller's quota is charged. A tool provider
does neither — the completion loop is Host-owned and the purpose that runs it
pays. Requiring `ai.invoke` would therefore misstate what the capability means
and would tell operators that a tool plugin can bill them, which it cannot.

What a tool contribution does require is what every executable contribution
requires: an exact-artifact trust review, `backend.entry`, and protocol version
2. No new capability key was added for M2.

### 4. The reserved slot is the transport; the declared timeout is the only freedom

Tool dispatch reuses the versioned `ProviderCall` family on the reserved slot
`sforum.ai.tool` with operation `invoke` rather than a new proto service: the
existing path already carries exact-declaration binding, contract version,
deadline, and error envelope, and a second RPC would duplicate those boundaries
and drift from them. Request/response schemas are
`sforum.ai.tool.invoke.request@1` / `sforum.ai.tool.invoke.response@1`.

The Host dispatches only to a declaration frozen in that runtime's manifest, and
the per-call deadline must equal the declared `timeoutMs`. `timeoutMs` is
required in the manifest and is never defaulted: the number the operator reads
is the number the tool can consume.

### 5. Plugin tools never shadow Core tools

`PluginToolRegistry.Apply` skips a name that already exists in the Host
registry. A plugin cannot replace `forum-search` with its own implementation
while the operator still sees a tool with that name. Duplicate names *inside*
the plugin set are rejected outright, because the model cannot be asked to
disambiguate a name whose owner is undefined.

### 6. The SDK must route the reserved slot explicitly

`Server.ProviderCall` now dispatches `sforum.ai.tool` to the AI tool registry
and returns `Unimplemented` when that registry is absent, before the generic
provider namespace is consulted. Without that branch, a plugin that only
declared `providers[]` could answer a Host AI tool call, and the tool the model
invoked would no longer be the one the manifest declared. A test registers a
generic provider that claims the same slot and asserts it is never called.

### 7. Transport lives in a focused collaborator, not on the God objects

`Manager` (receiver-method cap 72), `RuntimeInvoker` (cap 28), and
`RuntimeEventsProviders` (cap 11) are all at their non-growth baselines, so the
new dispatch path is a standalone collaborator in
`app/Support/Extensions/protocol_v2_ai_tool.go` (`AIToolRuntimeInvoker`, reading
the process `Starter` the way `ExtensionEventPublisher` reads the event
providers). `ProtocolStarter.InvokeVersionedAITool` moved out of `protocol.go`
into that file, which keeps `protocol.go` at its 1011-line cap.

On the bootstrap side, the concrete translation adapter lives in
`app/Providers/extensions.go` (`NewAIToolInvoker`), an already allowlisted
legacy-importer file, and is used by both the bootstrap assembly and the
subprocess integration test so there is exactly one field-by-field translation
to keep honest. The new `bootstrap/ai_plugin_tools.go` therefore introduces no
new production import of the legacy `Support/Extensions` package, and no new
entry was added to the legacy-import allowlist.

## Architecture Baseline Adjustments

- `apps/api/app/Support/ExtensionManifest/manifest.go`: **920 → 907** (lowered).
  The AI tool limits moved next to the declarations they bound in
  `v3_validate_platform.go`, and `IsOwnedID` plus `manifestHasPermission` moved
  to the validator that owns their only callers.
- `apps/api/app/Support/Extensions/protocol_v2_client.go`: **1107 → 1111**.
  The frozen-artifact client stores one parallel declaration slice per
  manifest family; declaring `aiTools[]` costs one config field, one client
  field, and two plumbing lines. Raising was preferred to compressing unrelated
  code, because the alternative (a second lookup path for AI tool declarations)
  would have created exactly the drift this client exists to prevent.
  **Reduction condition:** consolidate the per-family declaration slices into a
  single manifest snapshot inside the client.
- `apps/api/app/Support/Extensions` flat directory: **146 → 147**.
  `protocol_v2_ai_tool.go` follows the one-file-per-transported-capability
  convention already used by `plugin_command_transport.go`,
  `manager_query_transport.go`, `admin_surface_transport.go`, and the SEO
  transport. **Reduction condition:** move the provider-call transport family
  into a `Support/ExtensionProtocol`-style subpackage the next time that family
  is touched, and lower this cap accordingly. A second consecutive raise for the
  same family is not acceptable.

## Verification

- `go build ./...`, the focused Go suites for `Support/AI`,
  `Support/ExtensionManifest`, `Support/Extensions`, `Support/HostAPI`,
  `Support/Events`, `bootstrap`, `sdk/...`, `Models/AIReply`, `Models/AITools`,
  `node tests/validate-architecture-boundaries.mjs`, and the committed
  catalog-docs check are green.
- New tests: manifest declaration accept/deny matrix, runtime declaration
  validation, registry shadowing and duplicate rules, declaration binding and
  failure shape of `PluginTool.Invoke`, assembly registration behavior,
  adapter translation and fail-closed behavior, SDK reserved-slot routing and
  denial reasons, and cross-package contract/limit drift pins.
- End-to-end (real subprocess, no database):
  `extensions/fixtures/plugins/sforum-ai-tool-reference/` declares one AI tool
  and answers it through the reserved slot;
  `app/Support/Extensions/IntegrationTests/ai_tool_reference_plugin_integration_test.go`
  builds and starts it and drives the model-facing tool interface, asserting
  both Host-side refusals (declaration not frozen in the runtime manifest,
  deadline drift against the declared timeout) and the difference between an
  explainable tool failure and an internal plugin error.

## Open Questions

- How a plugin tool becomes callable by a purpose: an operator opt-in list in
  the AI console, a purpose-declared allowlist, or manifest-declared purposes.
  This must be decided before the M4 purpose registry lands, and it must not be
  answered by adding plugin names to a Core purpose allowlist by hand.
- Whether operators should see plugin tools in the AI console (and be able to
  disable a single tool) is part of the same question.
