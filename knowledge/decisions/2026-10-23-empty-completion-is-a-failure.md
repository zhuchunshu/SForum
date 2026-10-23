# Decision: an empty completion is a failure, and it is never cached

Date: 2026-10-23

## Status

Accepted and implemented (2026-10-23).

Related: `decisions/2026-10-19-ai-observability-and-capabilities.md`,
`decisions/2026-10-11-ai-private-provider-endpoints.md`

## Context

The admin console showed a column of `succeeded` rows for `forum.reply` with
`0 ms` and `0 / 0` tokens, `forum-reply@10`, while the community received no
reply at all. Reading the gateway explained the whole row:

- The only success path that records **no** latency and **no** usage is the
  result-cache hit (`Support/AI/gateway.go`). Every other success copies the
  provider's usage and measures the call with a monotonic clock.
- The wire layer accepts a well-formed 200 whose `content` is empty and whose
  `usage` is missing (`wire_openai.go`). A reasoning-only answer, a proxy that
  answers instantly, or a `max_tokens` above what the model declares all land
  there.
- `Execute` then ran `account()` and `store()` unconditionally on success, so a
  degenerate answer was written into the Redis result cache for ten minutes.
  Every later request with the same key replayed it in 0 ms and "succeeded"
  again — one bad answer became ten minutes of deterministic failure.
- The reply generator saw empty text, returned `ErrEmptyReply`, and the job
  layer cancelled the job (`river.JobCancel`) with a log line. Nothing about
  "the reply was never produced" reached `ai_executions`, so the console kept
  showing green.

The old contract said the opposite on purpose: `TestGatewayKeepsReasoningOutOfResultText`
asserted `succeeded` + empty text, and the connectivity diagnose mapped the same
shape to `ok=true, reason=empty_reply`.

## Decision

1. **An empty completion fails at the gateway.** `validateCompletion` rejects
   any result with no text **and** no tool calls, with the stable reason
   `empty_completion`. The provider's usage is still recorded (it was billed),
   but the row is `failed` and carries `finish_reason`, `reasoning=<bool>` and
   `usage` in `error_summary`.
2. **Empty results are never cached.** Two guards: the validation runs before
   `store()`, and `store()` refuses empty text without tool calls on its own.
3. **A cache hit must explain itself.** The hit row keeps `cacheHit=true` and
   records `0 ms / 0 tokens / 0 spend`, because that is what *this* invocation
   consumed: the money already sits on the row that produced the cached answer.
   The console shows a `cache` badge and a filter, plus a one-line explanation.
   (Copying the cached usage onto the hit row was rejected: cost dashboards sum
   these columns, and it would double count real money.)
4. **A tool-call-only response stays a success.** It is a normal step of the
   tool loop, not an empty answer.
5. **Diagnose keeps its two-layer meaning.** The connectivity probe treats
   `empty_completion` as a successful round trip (`ok=true`) with
   `reason=empty_reply` and the actionable hint, because "the round trip worked,
   the model said nothing" is exactly what the operator needs to see.
6. **The gateway returns the partial result with that error.** `StopReason`,
   `Reasoning` and `Usage` are the only explanation of the emptiness; dropping
   them turned every empty answer into "something failed". Text is still not
   returned as an answer, and callers must not publish it.

## Consequences

- `succeeded` now means "the model produced something usable". Empty answers
  turn red and are countable, which is what makes "为什么没回" answerable.
- Every deployment with Redis changes behaviour on the first empty answer: it
  now costs one failed call instead of ten minutes of poisoned replays.
- A provider that legitimately answers with nothing (refusals, instantly-empty
  proxies) surfaces as a failure. That is the intended trade: a silent success
  was being read as a working reply bot.
- The reply pipeline still needs its own fact ("did a comment appear"). That is
  the separate `ai_reply_outcomes` table — see
  `decisions/2026-10-23-reply-outcome-ledger.md`.
