# AI Plugin Host API - M1 + M1.5 + M2 Task Book

Status: **M1 + M1.5 + M2 landed** — the Host AI command, the event publish
command, and the `ai.tool` contribution path (manifest declaration, Host
transport, runtime registration, SDK dispatch) are in production code

Date: 2026-10-11
Last updated: 2026-10-11 (M2 landed)

Goal: give plugins one Host-owned entry point for AI calls (M1) and one for
publishing their own namespaced events (M1.5), so plugin-initiated work enters
the same gates, accounting, and audit as Core-owned work instead of growing a
parallel ledger outside them.

## Required Reading

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/ai.md`
4. `knowledge/modules/extensions.md`
5. `knowledge/decisions/2026-10-02-ai-provider-gateway.md`
6. `knowledge/decisions/2026-10-03-ai-read-only-chat-tools.md`
7. `knowledge/plans/2026-10-02-ai-assist-platform.md`
8. `knowledge/plans/2026-10-03-ai-read-only-chat-tools.md`
9. this task book

## Scope

### M1 — Host AI command

In scope:

- One Host command `sforum.ai.complete` registered in the sealed command
  catalog, callable by plugins through Host API v2.
- A new process capability `ai.invoke`, declared in the capability catalog and
  enforced through the command's `RequiredCapability`.
- Neutral request/response documents reusing the existing
  `sforum.ai.completion@1` contract types (`purpose`, `costClass`, `system`,
  `messages[]`, `maxTokens`, `tools[]`, `promptVersion`, metadata).
- Full reuse of the existing gateway: settings resolution, profile selection by
  cost class, three gates, usage accounting, execution trace, protocol
  adapters, and the guarded outbound invoker.

Explicitly out of scope (deferred to later milestones):

- `ai.tool` contribution is **landed** (M2, 2026-10-11): manifest `aiTools[]`
  declarations, the reserved `sforum.ai.tool` ProviderCall transport, Host-side
  registration into the Core tool registry, and the SDK registry + server
  routing. Registered is not visible — no purpose allowlist admits plugin tools
  yet. See `decisions/2026-10-11-ai-tool-contribution-m2.md`.
- `ai.provider` slot wiring (M3), `ai.purpose` registry (M4), completion hooks
  and append-only prompt contribution (M5).
- Plugin-specified models. A purpose declares a cost class; the gateway decides
  which profile serves it. This is a boundary rule, not a default.
- Streaming. The AI path has no streaming code today.
- Caching plugin calls before the unwired `ResultCache` is fixed (see Risks).

### M1.5 — Event publish command

In scope:

- One Host command `sforum.events.emit` that lets a plugin publish an envelope
  under its own namespace.
- Bridging the publish into the existing `appevents` publisher so declared
  listeners receive it.

Out of scope:

- Cross-plugin synchronous request/response. Use `services` for that.
- Emitting Core-owned event names. Namespaces stay separate.

## Context: why this is the first step

`Support/AI` owns the neutral contract, the protocol adapters, the three cost
gates, usage accounting, execution traces, and the SSRF-guarded outbound
invoker. Plugins currently have no way in. `GatewayConfig.Invoker` is already
an interface reserved for this, but production injects Core's own
`HTTPInvoker`, and `NewBuiltinRegistry` is the only caller that ever populates
the tool registry.

A plugin can today open its own outbound connection under `net.outbound` and
call a model vendor directly. That works and is exactly the problem: the call
never touches the gates, never writes an execution record, and never appears in
usage. The operator cannot explain what AI cost. Every later extension point
(tools, purposes, hooks) assumes plugin calls are already accounted for, so
building them first would rebuild a parallel ledger.

## Confirmed Design Decisions

1. **One command, not a proto service.** The command catalog is sealed,
   versioned, idempotent, actor-delegation aware, and audited per execution.
   Adding a new gRPC service would duplicate all of that. Reuse
   `protocolV2CommandDefinition` and register in
   `NewPostgresProtocolV2CommandRuntime`.
2. **Neutral documents, not vendor shapes.** The command input mirrors
   `ai.CompletionRequest` with `ProfileRef` omitted — a plugin must not address
   a profile directly. Output mirrors `ai.CompletionResult`. Both reuse the
   existing `TypedDocument` schema id/version mechanism.
3. **A plugin is always the caller.** `Metadata.CallerExtensionID` is set from
   the verified runtime identity, never from request input. This is what makes
   per-extension quota accounting correct; it must not be forgeable.
4. **`ActorMode: service`.** AI calls are server-initiated background work.
   A plugin cannot supply an actor; the command layer already rejects that.
5. **Subject user is opt-in and bounded.** `SubjectUserID` is accepted only
   when the caller supplies a delegated actor. A plugin must not be able to
   spend an arbitrary member's daily quota.
6. **No re-entrancy.** The command must not be callable from inside a
   completion hook. Deferred to M5, where the hook layer enforces it.
7. **M1.5 is separate and independent.** It touches the event bus, not the AI
   gateway, and can ship before or after M1.

## M1 Contract

```text
command: sforum.ai.complete@1
capability: ai.invoke
actor mode: service

input  sforum.ai.complete.input@1
  purpose           string   required, non-empty
  costClass         string   required, economy|standard|premium
  system            string   optional, <= MaxSystemBytes
  messages[]        array    required, 1..MaxMessages
  maxTokens         int      required, 1..MaxCompletionTokens
  temperature       number   optional, MinTemperature..MaxTemperature
  responseFormat    string   optional, text|json
  tools[]           array    optional, <= MaxToolsPerRequest
  toolChoice        string   optional, auto|none|required
  promptVersion     string   optional, participates in cache key + trace
  subjectUserId     int64    optional, requires delegated actor
  resourceType      string   optional
  resourceId        string   optional
  locale            string   optional

output sforum.ai.complete.result@1
  text              string
  stopReason        string
  toolCalls[]       array
  usage             { inputTokens, outputTokens, cachedTokens }
  provider          { extensionId, extensionVersion, packageDigest, protocol, model }
  promptVersion     string
  configRevision    int64
  latencyMs         int64
  cacheHit          bool
```

Failure shapes follow the existing command error vocabulary. New stable
reasons:

| Reason | Meaning |
| --- | --- |
| `ai.gateway_disabled` | Site switch off; zero outbound calls |
| `ai.purpose_invalid` | Missing or malformed purpose |
| `ai.cost_class_invalid` | Cost class outside the three values |
| `ai.profile_unavailable` | No enabled profile bound to the cost class |
| `ai.credential_missing` | Profile credential unresolved |
| `ai.request_invalid` | Contract validation failed |
| `ai.quota_exceeded` | Extension or user daily quota reached |
| `ai.budget_exceeded` | Site monthly budget reached |
| `ai.rate_limited` | Site rate limit reached |
| `ai.provider_failed` | Request went out and failed |
| `ai.unsafe_target` | Outbound guard rejected the address |

## M1.5 Contract

```text
command: sforum.events.emit@1
capability: events.emit
actor mode: service

input  sforum.events.emit.input@1
  name      string  required, must start with "<callerExtensionId>."
  kind      string  required, observe|filter
  payload   object  required

output sforum.events.emit.result@1
  delivered  int
  failed     int
```

**Hard constraint discovered during review:** `RuntimeEventsProviders.Emit`
looks the event up with `appevents.FindDefinition` and rejects unknown names
with `extension.event_unknown`. Manifest validation already allows namespaced
custom names via `knownOrNamespacedContract`, so declarations pass but
publication does not. M1.5 must therefore teach the event layer to accept a
caller-namespaced name, not only extend the command catalog.

Namespace rule: the name must begin with the caller's own extension id followed
by a dot. A plugin may never publish another plugin's namespace or a Core-owned
name.

## Implementation Steps

### M1

1. **Capability catalog** — add `ai.invoke` to `app/Support/Capabilities/catalog.go`
   with risk `medium` and bilingual labels. Update the OpenAPI capability
   fragment and the plugin docs catalog.
2. **Schema constants** — add input/output schema ids and versions beside the
   other command constants, following the `Command*SchemaID` naming used by
   existing commands.
3. **Command definition** — new file
   `app/Support/HostAPI/v2_command_ai.go` with
   `newProtocolV2AICompleteCommandDefinition(gateway *supportai.Gateway)`.
   Set `ActorMode: service`, `RequiredCapability: capabilities.AIInvoke`, and
   wire `Preview` / `Prepare` / `Execute`.
   Build the `CompletionRequest` in `Execute`, copying
   `Metadata.CallerExtensionID` from the verified runtime identity.
4. **Registration** — add the definition to
   `NewPostgresProtocolV2CommandRuntime` and thread the gateway through
   `PostgresProtocolV2CommandRuntimeConfig`. Missing dependency must fail boot,
   matching the existing "missing dependencies fail boot" rule.
5. **SDK helper** — add a typed helper in `apps/api/sdk/plugin/v2/` so plugin
   authors do not hand-build the document, mirroring the existing
   notification emit helper.
6. **Tests** — allowed and denied paths for capability, caller identity
   forging, cost class validation, quota rejection, gateway-disabled, and
   provider failure. Reuse `newProtocolV2CommandEngine` test harness style.

### M1.5

1. **Capability catalog** — add `events.emit`.
2. **Event layer change** — extend the publish path so a caller-namespaced name
   resolves, with the namespace prefix enforced. Keep Core-owned names
   unaffected.
3. **Command definition** — new file
   `app/Support/HostAPI/v2_command_events.go`, bridging to the publisher.
4. **Registration** — add to the command runtime.
5. **Tests** — allowed publish under own namespace; denied when publishing
   another plugin's namespace, a Core name, or a mismatched kind.

### Both

1. `./scripts/proto.sh generate && ./scripts/proto.sh check` if any proto
   surface changes.
2. Update `contracts/openapi` when the capability or catalog surface changes;
   run `ruby scripts/validate-openapi-refs.rb`.
3. Regenerate V3 catalogs; run the catalog completeness scripts.
4. `node tests/validate-architecture-boundaries.mjs` — new files must not
   cross 1000 lines and must not grow the two legacy non-growth packages.
5. Add a knowledge module note update and a decision record.

## Verification

- `cd apps/api && go build ./... && go test ./app/Support/HostAPI/... ./app/Support/AI/... ./app/Support/ExtensionManifest/... ./app/Support/Extensions/ ./bootstrap/ ./sdk/...`
- Fixture plugin exercising the AI command end to end, with the capability
  granted and withheld — **done**:
  `extensions/fixtures/plugins/sforum-ai-command-reference/` +
  `app/Support/Extensions/IntegrationTests/ai_command_reference_plugin_integration_test.go`
  (granted completion with real accounting and usage counters; quota exhaustion
  and site-disabled denials with zero outbound plus denied execution traces;
  withheld capability rejected at plan stage).
- Event publish command end to end — **done**:
  `extensions/fixtures/plugins/sforum-events-command-reference/` +
  `app/Support/Extensions/IntegrationTests/events_command_reference_plugin_integration_test.go`
  (own-namespace publish committed and delivered to the fixture's declared
  listener; foreign namespace and Core-owned names refused with
  `events.namespace_forbidden`). This run exposed and fixed a plan-validity defect
  that made the command unusable through the real broker.
- Fixture plugin declaring `aiTools[]` end to end through a real subprocess —
  **done**: `extensions/fixtures/plugins/sforum-ai-tool-reference/` +
  `app/Support/Extensions/IntegrationTests/ai_tool_reference_plugin_integration_test.go`
  (builds, starts, dispatches through the reserved slot, and asserts the
  not-frozen-declaration and deadline-drift refusals).
- Execution records show `caller_extension_id` set to the calling plugin and
  `purpose` / `cost_class` populated.
- With the site switch off, a plugin call performs zero outbound calls and
  returns `ai.gateway_disabled`.
- Quota exhaustion rejects only the calling extension.

## Risks And Open Items

- **`ResultCache` is unwired** — **closed** (2026-10-11): `Providers/ai.go` now
  wires `Support/Cache` (Redis) through `ai.NewResultCache`, so identical prompts
  inside the 10-minute window are served from the cache, cost nothing, and report
  `cacheHit: true`. See `decisions/2026-10-11-ai-result-cache-wiring.md`.
- **`RedactionSettings` implementation** — **closed** (2026-10-11): redaction
  now runs inside `Gateway.Execute` before the cache key and before any outbound
  call, covering system text, message parts, and tool-call arguments. See
  `decisions/2026-10-11-ai-outbound-redaction.md`.
- **Plugin jobs cannot reach the `ai` queue.** `RiverJobEnqueuer` hardcodes
  `QueueDefault`. Slow AI work competes with ordinary jobs and ignores
  `JOB_QUEUE_AI_WORKERS`. Out of scope here, but it will surface as soon as
  plugins do real AI work.
- **Open question:** whether a plugin call counts against the site rate limit
  in addition to its extension quota. Current `EvaluateGates` already applies
  both; confirm that is intended for plugin traffic before shipping.
- **Open question:** default `ExtensionDailyQuota` (currently 500) may be
  generous for many plugins sharing one site budget.

## Constraints

- No AI call on a synchronous user request path.
- The gateway never lets a caller choose a profile or model directly.
- Caller identity comes from the verified runtime, never from request input.
- Boundary layer stays non-configurable: accounting, trace, secret handling,
  outbound guards, contract versions.
