# Decision: reply outcomes are a ledger, not a second execution trace

Date: 2026-10-23

## Status

Accepted and implemented (2026-10-23).

Related: `decisions/2026-10-23-empty-completion-is-a-failure.md`,
`decisions/2026-10-19-ai-observability-and-capabilities.md`

## Context

`ai_executions` records **model calls**: purpose, profile, model, tokens, gate
reason, latency, cache hit. It deliberately contains no prompt or response body.

A reply is a different unit of work. One reply may contain several model calls
(the tool loop) or none at all (gateway denied, context unreadable). Its
important fact is binary and social: **did a comment appear?** Before this
change that fact lived only in `slog`:

- empty answer → `ErrEmptyReply` → `river.JobCancel` (no retry, no trace row);
- publish refused (permission, locked topic, content rule) → error → three
  attempts → discarded job, `ai_executions` still all `succeeded`;
- duplicate trigger → cache replay → a second identical reply.

The admin console therefore could not answer the first question every operator
asks: "为什么没回".

## Decision

1. **A new ledger table `ai_reply_outcomes`** (migration `202610250001`), one row
   per generation attempt, written at the single exit of
   `aireply.Generator.Generate` via `defer`. Rows carry topic / parent and
   trigger comment / bot user / run id / comment id / outcome / reason / detail /
   model calls / tool calls / tokens. `detail` is a stable phrase
   (`finish_reason=length reasoning=true`), never prompt or response text.
2. **Stable outcome codes**: `posted`, `empty_reply`, `tool_loop_stuck`,
   `unavailable`, `publish_failed`, `failed`. They are a CHECK constraint, so a
   typo cannot invent a silent new state.
3. **It is also the idempotency key.** `HasPostedReply(topic, trigger)` returns
   true only for `posted` with a real comment id. `Generate` short-circuits
   before any model call in that case. A rejected or empty attempt does not
   count, so an operator who fixes the configuration still gets the retry.
4. **Writing is best-effort, never a precondition.** A store failure logs a Warn
   and the reply still publishes (same discipline as citations and tool calls).
   A missing ledger degrades to "why did it not reply" being log-only again.
5. **Reading goes through `Models/AI`** with the same `ai.manage` check as
   execution records: an administrator who can see the call must be able to see
   its outcome. `GET /admin/ai/reply-outcomes?limit=` returns
   `{ items: [...] }`, shaped like the execution list so one UI pattern serves
   both.
6. **No public surface.** The ledger is operator diagnostics and sits next to
   user content (topic/comment ids). It gets no `sforum_core_v1` view and no
   HostQuery; the reason is recorded in `knowledge/modules/ai.md`, as the
   extension data-surface policy requires for a deliberately absent read
   surface.

## Consequences

- The console now has both halves: execution records (what the model cost) and
  reply outcomes (what the community saw). A row of green calls next to a red
  `empty_reply` is the correct, readable picture of this failure.
- One extra insert per reply attempt; no read amplification on the hot path.
- Retention is not automated in v1 (a 30-day note lives in the module doc);
  the table grows with reply attempts only.
- The generator now depends on a store it did not have before. It stays
  optional (`WithOutcomes`), so tests and the eval stack keep working without a
  database.
