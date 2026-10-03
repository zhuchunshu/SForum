# AI Read-Only Chat Tools - Design And Task Book

Status: **active** - contract, adapters, gateway loop, five built-in tools, the
reply integration and the admin settings landed together; verification is
partially complete (see Verification), and plugin-contributed tools remain a
separate step.

Date: 2026-10-03
Last updated: 2026-10-03

Goal: let the AI reply bot consult the forum instead of guessing. The bot gets
a small set of **read-only** tools - site search, topic read, topic list, public
profile, server time - executed by a Host-owned loop with a per-reply step
budget, per-step gates, per-step traces, and citations in its answers.

## Required Reading

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/decisions/2026-10-03-ai-read-only-chat-tools.md`
4. `knowledge/decisions/2026-10-02-ai-provider-gateway.md`
5. `knowledge/plans/2026-10-02-ai-assist-platform.md`
6. this task book

## Scope

In scope:

- `sforum.ai.completion@1` tool extension: `tools[]` / `toolChoice` in the
  request, `toolCalls[]` in the result, `role=tool` messages with `toolCallId`;
  OpenAI and Anthropic translation; contract limits.
- Host-owned orchestration loop with a per-reply step budget, per-step gateway
  calls (gates, accounting, traces), tool-result truncation, and a hard stop
  when the budget is spent.
- Five Core built-in read-only tools in `app/Models/AITools`, registered by the
  assembly into one `ToolRegistry`.
- Reply integration: topic body in the prompt, purpose allowlist, tool context
  (viewer, topic, comment), citation requirement and injection boundary in the
  system prompt, prompt version bump.
- Operator controls: `gates.toolCallsPerReply` (nullable; absent = recommended
  3, explicit 0 = tools off, hard cap 5), `profile.supportsTools` (nullable;
  absent = supported), admin panel fields, i18n, and OpenAPI schema updates.

Out of scope (explicit):

- Write actions of any kind (posting, editing, moderation, flags).
- Summary/translation as nested tools: they remain M4 purposes with their own
  quota and traces (operator decision, 2026-10-03).
- Cross-trigger memory: the bot stays stateless per trigger; older content is
  fetched on demand.
- Streaming: the AI path has no streaming code; the tool loop ships non-streamed.
- Plugin-contributed tools: the registry accepts them, but no extension-facing
  manifest surface is declared yet. That is the next contract step.

## Confirmed Product Decisions (operator, 2026-10-03)

| Question | Decision |
| --- | --- |
| Surface | Existing topic reply bot; no new chat surface |
| Posture | Read-only first (zero side effects) |
| Ownership | Core registry + five built-in tools; plugins contribute later |
| First batch | search / topic read / topic list / profile / time |
| Step budget | <= 3 tool calls per reply, operator-adjustable, hard cap 5 |
| Nested calls | Separate purpose accounting (M4 purposes), not inlined |
| Memory | Stateless per trigger |
| Streaming | Not now |
| Injection boundary | Tool results are untrusted data, same as post bodies |
| Visibility | Visitor-visible content only; "not found" without existence leaks |
| User quota | One reply = one user quota unit; steps count on site/extension scopes |

## Implementation Map

| Area | Files |
| --- | --- |
| Contract + limits | `Support/AI/contract.go`, `Support/AI/limits.go` |
| Protocol adapters | `Support/AI/wire_openai.go`, `Support/AI/wire_anthropic.go`, `Support/AI/wire.go` |
| Loop + registry | `Support/AI/tool_loop.go` |
| Profile capability / gate | `Support/AI/profile.go`, `Support/AI/settings.go`, `Support/AI/gateway.go` (cache key) |
| Built-in tools | `Models/AITools/{aitools,search,topic_read,topic_list,profile,clock}.go` |
| Reply integration | `Models/AIReply/{context,generator,completer,postgres_reader}.go`, `Support/AI/reply_prompt.go` |
| Assembly | `Providers/ai_reply_generator.go`, `bootstrap/api_assembly*.go` |
| Admin + contract | `apps/web/app/components/admin/ai/tabs/SFAdminAIProvidersTab.vue`, `composables/admin/useAdminAI.ts`, both locale files, `contracts/openapi/schemas/ai.yaml` |

## Acceptance Criteria

1. A reply whose question needs forum facts calls `forum-search` (and optionally
   `forum-topic-read`), cites links, and never claims "someone on the forum
   said" without a link.
2. The step budget is respected; the final model call carries no tool
   declarations, and the loop stops with a final answer.
3. Every step appears as its own execution record; the user quota records one
   entry per reply, the site/extension scopes one per step.
4. `toolCallsPerReply = 0` (or a profile with `supportsTools: false`) produces
   the previous plain-completion behavior byte for byte.
5. A tool failure (unknown tool, engine unavailable, missing topic) becomes a
   model-facing error result and does not fail the reply; an internal failure
   is logged and reported generically.
6. Non-public topics, hidden categories and missing users are answered as
   "does not exist" and never appear in prompts.
7. Prompt-injected instructions inside a post or a tool result are not obeyed
   (safety appendix covers both).

## Verification

- `go test ./app/Support/AI/... ./app/Models/AITools/... ./app/Models/AIReply/... ./app/Models/AI/...`
  - contract/tool validation, both protocol adapters, the loop (tool then
    answer, budget, gates mid-loop, unknown tool, error surfacing, truncation,
    registry rules), the five tools, and the reply integration.
- `bun run typecheck` + `bun test` in `apps/web` for the admin panel fields.
- `ruby scripts/validate-openapi-refs.rb` after the schema edit.
- `node tests/validate-architecture-boundaries.mjs` for the new packages/files.
- Rendered browser QA (2026-10-03, BrowserSkill, 1440x900 + 390x844): both new
  fields render with correct geometry and no overflow; the panel reports
  "using default" and stays clean on load, and a one-character edit flips it to
  "customized" plus the unsaved-changes hint.
- Pending (next session): one live reply on a real deployment to observe the
  first real tool call.

## Residuals

- Execution records carry no resource column, so a tool step cannot be joined
  back to its topic in the admin list. Add when per-topic drill-down is needed.
- `forum-search` depends on the site engine being available; when the engine is
  unavailable the tool reports it and the model answers from context.
- Plugin-contributed tools need the `ai.tool` manifest surface and an install
  review before any third-party tool can run; until then the registry is
  Core-only by construction.
