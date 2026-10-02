# AI Assist Platform - Design And Task Book

Status: **ready** - Host AI contracts and the plugin/provider split are
approved; M0 contract work may start

Date: 2026-10-02
Last updated: 2026-10-02

Goal: let an operator attach AI to moderation, reply, summarization,
translation, and reporting workflows through Host-owned provider, purpose, and
middleware contracts, so that every AI capability is an extension contribution
and one more plugin can extend an existing capability without Core changes.

## Required Reading

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/extensions.md`
4. `knowledge/modules/moderation.md`
5. `knowledge/modules/notifications.md`
6. `knowledge/decisions/2026-10-02-ai-provider-gateway.md`
7. `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
8. this task book

## Scope

In scope:

- Host AI gateway: provider slot, neutral completion contract, profile
  registry, Secret Store credential references, the three cost gates, cache,
  idempotency, audit, and execution trace.
- Three extension points: provider (vertical), purpose (horizontal), and
  middleware (decoration).
- AI-assisted moderation: asynchronous review advice attached to the existing
  pre-publish pending queue and review workbench.
- Bot identity: `users.kind`, administrator-driven conversion, and
  health-driven presence.
- Content purposes shipped as separate plugins: topic summary, cross-language
  translation, community digest, report risk ranking, notification digest.
- Extension authoring assistance on top of the `sforum` CLI scaffold.

Out of scope by explicit operator decision on 2026-10-02:

- **Embeddings and vector retrieval.** No local embedding model, no pgvector,
  no similarity-based feature. Duplicate-post detection, semantic search,
  similar-topic recommendation, account clustering, and review-workbench
  historical-decision retrieval are all deferred together. Reopening this is a
  separate decision record, not a follow-up task inside this plan.
- **Multimodal input.** No image moderation, no generated alt text, no vision
  requests.

## Confirmed Product Decisions

- Providers are addressed by **protocol**, not vendor: `openai-chat` covers
  ChatGPT, DeepSeek, Moonshot, Qwen, Ollama, vLLM, and compatible gateways;
  `anthropic-messages` covers Claude.
- DeepSeek and OpenAI ship as **built-in provider profiles** with editable
  `base_url`, `api_key_ref`, and `model`. Operators may add profiles.
- Moderation failure posture and high-confidence automatic action are **Admin
  switches**, not hard-coded behavior.
- Bots **occupy a user slot and count as online users**. Presence follows AI
  provider health, so a failed provider makes the bot appear offline.
- Administrators can convert an existing user into a bot.
- Quota, budget, rate limits, and most other parameters are **Admin
  configurable**, subject to the configuration layering below.
- LLM calls never run on the synchronous write path.
- AI advises; a human `ReviewerUserID` still performs every moderation
  decision.

## Existing Baseline To Reuse

| Area | Existing owner | Required treatment |
| --- | --- | --- |
| Publication decision seam | `Models/Moderation` `Settings.Evaluate` behind `Forum.Service.publicationDecision` | Attach AI review advice after the decision, never replace the synchronous seam |
| Pending queue and workbench | `Models/Moderation/workbench_store.go`, `workbench_types.go` | Extend pending items and review context with AI advice fields; keep `SubmitDecision` authoritative |
| Provider slots | `Support/Extensions` `VersionedProviderSlotRegistry`, `ProviderSlotSelectionAPI` | Add `ai.provider` through the same fenced lifecycle and selection API |
| Plugin decision hooks | Manifest V3 `kind: "filter"` events as declared by `sforum-content-policy` | Declare `ai.before_completion` / `ai.after_completion` as the same hook shape |
| Secrets | `Support/SecretStore` with `sforum.secret://` references | Store provider credentials as references; never echo values into settings |
| Outbound HTTP | `Support/HostAPI` policy-checked HTTP plus `Support/OutboundHTTP` SSRF guards | All provider traffic goes through the guarded path |
| Durable async work | `Support/Jobs` `PluginJobContract` with concurrency, retry, and idempotency | AI work uses a dedicated queue and concurrency budget |
| Bounded tuning | `Support/ContentRegistry` `ExecutionLimits` hard ceilings | Copy the "configurable but still bounded" pattern for timeouts, retries, and rate limits |
| Operator defaults | `Models/Moderation` `RecommendedSettings()` + `ResetSettings` | Every AI setting gets a recommended default and one-click restore |

## Extension Points

Three classes are deliberately separated. Mixing them produces a contract that
cannot be explained after the fact.

| Class | Provider of the contribution | Contract |
| --- | --- | --- |
| Vertical - new vendor/protocol | provider plugin | `ai.provider` slot; implements `ai.completion@1` |
| Horizontal - new AI capability | purpose plugin | purpose registry entry: input/output schema, prompt reference, trigger, cost class |
| Decoration - pre/post processing | any plugin | `ai.before_completion` / `ai.after_completion` filter hooks |

Rules that keep the three separable:

- A middleware must never call the gateway. Re-entrancy is rejected.
- Middleware ordering is deterministic: priority first, then extension ID
  lexicographic order. Registration order never decides behavior.
- Each middleware declares its own failure posture; the default is fail-open
  (pass the original value through).
- Cache keys include purpose, caller extension, prompt version, and model, so
  two plugins can never read each other's results.
- Purposes declare a cost class; the gateway, not the plugin, decides which
  provider profile serves that class.

## Contracts (draft)

```text
ai.completion@1 request
  { profileRef?, purpose, costClass, system, messages[], maxTokens,
    temperature?, responseFormat?, metadata{ caller, resource, locale } }

ai.completion@1 result
  { text, stopReason, usage{ inputTokens, outputTokens, cachedTokens },
    providerArtifact{ extensionId, version, digest, protocol, model },
    promptVersion, latencyMs, cacheHit }
```

Provider profile shape:

```text
{ id, protocol: openai-chat | anthropic-messages, baseUrl, apiKeyRef,
  model, defaults{ maxTokens, temperature, timeoutMs }, capabilities{} }
```

Protocol translation is a Core-owned adapter concern with explicit mappings
for system placement, tool call shape, image encoding, stream delta shape,
usage field names, stop reason, and authentication headers.

## Configuration Layering

| Layer | Contents | Constraint |
| --- | --- | --- |
| Policy - fully configurable | purpose toggles, profile assignment per cost class, failure posture, automatic-action switch and confidence threshold, quotas and budget, trigger scope, bot identity, redaction toggles | Every item needs a recommended default and one-click restore |
| Bounded - configurable with ceilings | provider timeout, retry count, rate limit, cache TTL, automatic-action confidence | Host rejects values above the hard ceiling; mirrors `ExecutionLimits` |
| Boundary - not configurable | usage accounting, decision trace, secret encryption and audit, middleware ordering rules, contract versions | Neither plugins nor Admin can change these |

Configuration placement: gateway parameters live in Core settings and are
managed from one Admin AI panel; purpose parameters live in each purpose
plugin's `settings.json`. A purpose plugin declares that it needs AI; the
gateway decides which profile and budget serve it.

Every AI execution snapshots the configuration version in effect. Later edits
must never reinterpret an earlier decision.

## Cost Gates

Three gates with different blast radii; exceeding one must not silently break
the others.

| Gate | Limits | Scope | On exceed |
| --- | --- | --- | --- |
| Rate limit | calls per second/minute | site or single extension | short queue, reject when full |
| Quota | calls and tokens per day/month | single extension, single user | reject only that caller |
| Budget | money per month | whole site | global degradation to rules mode plus Admin notice |

Accounting is three-dimensional: provider x purpose x calling extension.
Exceeding a plugin quota must be visible in Admin ("plugin X exhausted its
daily quota, 1,284 calls rejected"), never a silent no-op.

Defaults follow Beginner-Friendly Defaults: a small per-extension daily quota
that an operator must raise deliberately, a conservative site budget that is
on by default, and a per-user daily limit for user-triggered purposes.

## Bot Identity

- `users.kind` (`human` | `bot`, default `human`); migration required - the
  current table only has `status`.
- Bots occupy a user slot and count as online users.
- Presence is health-driven with hysteresis: N consecutive provider failures
  mark the bot offline, M consecutive successes restore it.
- A user mentioning an offline bot receives an immediate "assistant
  unavailable" response instead of waiting for a reply that will not arrive.
- Bots keep ordinary reachability: mentionable, reportable, mutable by
  moderators, blockable.
- Bots do not enter the new-user trust ladder, hold no login credential by
  default, cannot be granted admin permissions by default, and default to
  muted notifications.
- Bot-generated comments carry a persisted `ai.generated` marker and a depth
  counter. AI output never triggers AI.
- Bot output passes moderation like user content. A bot must not become a
  laundering path around the review queue.
- Mention-triggered replies require an explicit trigger; not every mention
  invokes the bot.

## Milestones

### M0 - Gateway kernel

Deliverable: `ai.provider` slot, `ai.completion@1`, both protocol adapters,
profile registry, Secret Store wiring, usage accounting, the three gates,
cache and idempotency, execution trace, and a Core Admin panel that defaults
to disabled.

The default execution path is Core-owned protocol translation plus the Host
outbound guard, not a plugin call: the requirement to ship DeepSeek and OpenAI
built in with editable base URL and API key cannot hold if vendor behavior
lives in plugin code. `ai.provider` stays reserved for non-standard protocols.

Acceptance evidence: provider selection resolves through the versioned slot
registry; a profile edit changes the effective endpoint without a restart;
missing credentials fail closed with an operator-visible reason; usage rows
carry provider, purpose, and caller; trace records prompt version and
middleware chain; a disabled gateway performs zero outbound calls.

### M1 - Extension points frozen

Deliverable: `ai.purpose` registry, `ai.before_completion` /
`ai.after_completion` hooks, per-extension quota accounting, deterministic
middleware ordering, re-entrancy rejection, and an Admin trace viewer.

Acceptance evidence: a fixture purpose plugin and a fixture middleware plugin
run end to end without Core edits; re-entrant middleware is rejected; ordering
is stable across restarts and registration permutations; a middleware failure
with fail-open posture leaves the host value intact.

This milestone intentionally ships no product feature. Its purpose is to
freeze the extension contract before any product purpose exists, so the first
real purpose cannot hard-code itself into Core.

### M2 - Moderation assistant

Deliverable: asynchronous AI advice on pending topics and comments, rendered
first in the review workbench with a suggested action, confidence, reasons,
and a prefilled review note; one-click accept or reject; failure posture and
automatic-action switches; per-decision audit including the configuration
version.

Acceptance evidence: a pending item shows advice without blocking publication;
accepting advice produces an ordinary `SubmitDecision` with a real reviewer;
provider failure degrades to rules mode without stalling the queue; every
decision can be replayed with the prompt version, model, and configuration
that produced the advice.

### M3 - Reply bot

Deliverable: bot identity end to end, explicit mention trigger, loop
prevention, quotas, generation labelling, and re-moderation of generated
output.

Acceptance evidence: an offline provider shows the bot offline and answers
mentions with an availability notice; generated content never triggers another
generation; a bot comment passes the same moderation path as a user comment;
per-user quota exhaustion is visible and reversible.

### M4 - Content purposes

Deliverable: topic summary, cross-language translation, community digest,
report risk ranking, and notification digest, each as an independent purpose
plugin.

Acceptance evidence: each purpose can be disabled without affecting the
others; summary caches invalidate on edit; a purpose plugin uninstall leaves
historical AI results degraded but renderable.

### M5 - Extension authoring assist

Deliverable: `sforum` CLI scaffold assistance that turns a described
requirement into a Manifest V3 skeleton and contract declarations.

Acceptance evidence: generated output must pass `extension validate`,
`extension digest --write`, and `extension test` before it can be enabled;
enabling still requires the exact-artifact, actor-bound `super_admin`
confirmation; no generated package executes code at install time.

## Constraints

- No LLM call on a synchronous user request path.
- `SubmitDecision` keeps requiring a real reviewer. AI never becomes the
  actor of record.
- AI work uses a dedicated queue and concurrency budget so a content storm
  cannot starve mail and notification delivery.
- Prompt templates are versioned; changing one never retroactively
  reinterprets stored decisions.
- Outbound content redaction (email, phone, IP) is configurable and audited.
- Every AI capability must be removable by disabling a plugin, without
  breaking already published content.

## Open Questions

- Whether the first release exposes per-category or per-user-group scoping for
  AI purposes, or stays global with a reserved `scope` field in the contract.
- Default numeric values for the per-extension daily quota and site monthly
  budget.
- Whether administrators may edit Core-owned moderation prompts directly, or
  only through constrained append-only additions.
