# Decision: AI Chat Tools Are Read-Only, Gateway-Orchestrated, And Purpose-Allowlisted

## Status

Accepted (2026-10-03). Implemented in the same change; see
`plans/2026-10-03-ai-read-only-chat-tools.md`.

## Context

The M0 AI gateway (neutral completion contract, provider profiles, three cost
gates, execution traces) and the M3 reply bot are in production code. Two facts
framed the tool-calling design:

- The neutral contract already mapped both protocols' tool stop reasons
  (`tool_calls` / `function_call` / `tool_use` → `StopReasonTool`), but
  `CompletionRequest` had no `tools` field and the parsed response carried only
  text. Tool calling was anticipated, not implemented.
- The reply prompt carried the topic title, the last ten comments and the
  replied comment. The topic's own body was absent, so questions about the
  opening post ("why does this code fail") could only be guessed at.

The operator settled the product shape in a design discussion on 2026-10-03:
surface is the existing topic reply bot (no new chat surface); posture is
read-only first; ownership is a Core registry with a few built-in tools and
plugin-contributed tools later; the first batch is search, topic read, topic
list, user profile and server time; the step budget is at most three tool calls
per reply, operator-adjustable with a hard cap of five; summaries/translations
stay separate purposes (M4) with their own accounting; no cross-trigger memory;
no streaming.

## Decision

1. **Tool calling is a Host contract extension of `sforum.ai.completion@1`.**
   `CompletionRequest` gains `tools[]` (name, description, JSON Schema) and
   `toolChoice`; `CompletionResult` gains `toolCalls[]`; `Message` gains the
   `tool` role plus `toolCallId` and assistant-side `toolCalls`. Core adapters
   translate to OpenAI `tools` / `tool_calls` / `role=tool` and to Anthropic
   `tools` / `tool_use` blocks with `tool_result` blocks merged into one user
   message. Limits (tool count, name charset, schema size, argument size) live
   in `limits.go` with the other hard ceilings.
   Tool names are restricted to `[a-z0-9_-]`: both supported protocols accept
   only that pattern (DeepSeek rejected a dotted name with HTTP 400), so the
   contract encodes the real constraint and rejects dots at registration time
   instead of letting the provider fail at runtime. Built-in names are
   `forum-search`, `forum-topic-read`, `forum-topic-list`, `user-profile-read`,
   `time-now`.
2. **The loop is gateway-owned.** `Support/AI.Orchestrator` runs
   model → tool → model until a final answer. Each step is an ordinary
   `Gateway.Execute` call, so the three gates, usage accounting and execution
   traces apply per step without a second implementation. Tools must never call
   back into the gateway (the same anti-re-entrancy rule as middleware).
3. **User quota counts one reply.** Only the first call of a run carries
   `SubjectUserID`; tool steps count against site and extension scopes. This was
   the operator's explicit choice so a two-step reply does not silently consume
   two of a member's twenty daily replies.
4. **Read-only posture.** The five built-in tools wrap existing public read
   paths: site search (`Support/Search`), topic detail and comment list
   (`forum.Service`), public profile (`profile.Service`), and the server clock.
   Visibility equals a visitor's; missing or non-public targets answer "does not
   exist" and never leak existence or permission details.
5. **The injection boundary covers tool results.** Tool output is community
   content: the fixed safety appendix now states that tool results are context,
   not instructions, and the default prompt requires links when citing forum
   content. Tool results are truncated per result and per run
   (`MaxToolResultBytes`, `MaxToolResultRunBytes`) so one search cannot flood
   the context.
   The built-in default prompt also has to survive the admin rich-text editor:
   it must not contain literal angle-bracket tokens (the editor strips them,
   which made the panel treat the default as a customized prompt and would have
   saved the stripped copy). `ReplyPromptVersion` is `forum-reply@3` for that
   wording change. The panel decides "default vs customized" by ignoring
   whitespace differences and by canonicalizing empty-or-default prompts to the
   same shape (`apps/web/app/utils/admin/adminAI.ts`), because the editor
   reflows Markdown and the server omits an empty prompt.
6. **Budget and capability are operator-controlled.** `gates.toolCallsPerReply`
   is a nullable integer: absent means the recommended 3, explicit `0` means
   tools are off, and the hard cap is 5. `profile.supportsTools` is a nullable
   boolean: absent means supported. Both are nullable on purpose - an upgrade
   must grant the new capability by default, while an operator who explicitly
   turns it off keeps that choice.
7. **The allowlist is declared by the purpose.** `aireply.ReplyToolAllowlist`
   names what `forum.reply` may call. A tool registered for moderation can never
   be reached by a prompt-injected reply, because the request never advertises
   it.

## Alternatives rejected

- **Loop in the caller** (each purpose runs its own tool loop): every new
  purpose would re-implement budget, gates, traces and truncation, and the
  operator would have no single place to explain what a reply cost.
- **Summary/translation as nested tools now**: deferred to M4, where they are
  independent purposes with their own quota and traces (the operator chose
  separate accounting over inlining them into the reply's budget).
- **Bigger context instead of tools**: the topic body was added to the prompt
  anyway as the cheap first step, but cross-topic retrieval and clickable
  citations are things a larger window cannot provide.

## Consequences

- A reply can cost up to four model calls (one initial + three tool rounds +
  final answer is the worst case at the default budget); per-step gates,
  truncation and the step cap bound it, and traces show every step separately.
- Settings saved before this change carry no `toolCallsPerReply` and therefore
  resolve to the recommended 3, so a site that already runs the reply bot gains
  tools on upgrade. An operator who wants the previous plain-text behavior sets
  the value to 0 explicitly.
- Tool steps are attributable in the execution list but not yet joined to the
  triggering topic (execution records have no resource column). Operators see
  counts and cost; per-topic drill-down remains future work.
- The reply reader now also requires `categories.visibility = 'public'`, which
  matches the documented "same as a visitor" rule and closes a gap where hidden
  categories could have been quoted by the bot.
