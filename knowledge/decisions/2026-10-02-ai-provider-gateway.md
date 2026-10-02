# Decision: AI Capabilities Ship As Host Contracts With Plugin Providers

## Status

Accepted

## Context

SForum needs AI-assisted moderation, mention replies, and further content
automation. The Host already carries the substrate this requires: the
synchronous publication decision seam (`Moderation.Settings.Evaluate` behind
`Forum.Service.publicationDecision`), the pre-publish pending queue and review
workbench, Manifest V3 `kind: "filter"` decision hooks, the versioned provider
slot registry (`mail.provider`, `attachment.storage.provider`,
`search.provider`, `sforum.identity`, `sforum.seo`), the namespaced Secret
Store, policy-checked outbound HTTP with SSRF guards, and durable plugin jobs
with retry and concurrency contracts.

Two properties make an AI slot unlike every existing slot.

First, calls are metered and billed. A mail, storage, or search plugin that
misbehaves consumes server resources; an AI plugin spends the operator's money,
and a single bug - for example an AI call on every topic view - can produce tens
of thousands of billable calls before anyone notices.

Second, the call site can sit on the moderation publication boundary. A
provider plugin is therefore able to influence what is allowed to be published,
which no other slot can do.

The operator additionally required that the capability be plugin-shaped and
that a further plugin be able to extend an existing AI capability, not merely
add another vendor.

## Decision

- Core owns the gateway contract **and the default execution path**: provider
  slot, neutral completion contract, profile registry, credential references,
  cost gates, cache, accounting, audit, and trace. Protocol translation
  (`openai-chat`, `anthropic-messages`) and the guarded outbound call are Core
  responsibilities, because the operator requirement to ship DeepSeek and
  OpenAI built in with an editable base URL and API key cannot hold if vendor
  behavior lives in plugin code - changing a base URL would mean changing a
  plugin. The `ai.provider` slot remains the extension point for non-standard
  protocols (self-hosted inference, Bedrock, gRPC) rather than the default path.
- Providers are keyed by **protocol**, not vendor. `openai-chat` and
  `anthropic-messages` are two Core-owned translation adapters over the
  neutral contract; DeepSeek, OpenAI, and compatible gateways are profiles of
  the former, Claude a profile of the latter.
- A provider is a **profile**: `{protocol, baseUrl, apiKeyRef, model,
  defaults, capabilities}`. Built-in DeepSeek and OpenAI profiles ship with
  editable `base_url` and `model`; operators may add their own.
- Credentials are Secret Store references (`sforum.secret://`). Provider keys
  never appear in ordinary settings documents.
- Three extension point classes are separated and frozen before the first
  product purpose ships: provider (vertical), purpose (horizontal), middleware
  (decoration). Middleware re-entrancy is rejected, ordering is deterministic
  (priority then extension ID), and each middleware declares its own failure
  posture.
- Cost control is three gates with distinct blast radii: a rate limit (site or
  extension), a quota (per extension, per user), and a budget (site). Usage is
  accounted per provider x purpose x caller.
- Configuration is three-layered. Policy items are fully configurable with
  recommended defaults and one-click restore; tuning items are configurable up
  to Host-enforced hard ceilings, mirroring `ContentRegistry.ExecutionLimits`;
  boundary items (accounting, trace, secret handling, ordering rules, contract
  versions) are not configurable by plugins or by Admin.
- Every AI execution snapshots the configuration version in effect, so an
  operator change never retroactively reinterprets an earlier decision.
- The AI never becomes the actor of record. `Moderation.SubmitDecision` keeps
  requiring a real `ReviewerUserID`; AI contributes advice, confidence,
  reasons, and a prefilled review note only.
- No LLM call runs on a synchronous user request path. Synchronous work stays
  limited to locally computable checks.
- AI output is re-moderated like user content, and generated content never
  triggers further generation.
- Bots are ordinary user rows (`users.kind`) that occupy a user slot, count as
  online users, and take presence from provider health.
- Embeddings and vector retrieval are excluded, as is multimodal input.
  Similarity-dependent features (duplicate detection, semantic search,
  similar-topic recommendation, account clustering, and review-workbench
  historical-decision retrieval) are deferred together with them.

## Consequences

- Operators get one place to see and cap AI spend, and plugin authors never
  handle credentials or vendor protocols.
- The default execution path goes through the Host outbound guard: public
  addresses only, no URL userinfo, bounded redirects, bounded timeout, and a
  byte cap on the response body. A future provider plugin for a non-standard
  protocol will need its own declared outbound authority rather than inheriting
  this one.
- A plugin that exhausts its quota fails alone; it cannot take down site-wide
  moderation or notification delivery.
- Deferring embeddings removes a whole capability class from the roadmap, not a
  single feature. If an operator later wants similarity features, the cheapest
  reopening is a remote embedding profile on the existing gateway rather than a
  new local model service; that requires its own decision.
- Snapshotting configuration versions adds storage and a schema obligation on
  the first AI migration.
- Because AI advice is not authority, an unattended site still has human
  bottlenecks by design. That is accepted: the failure mode of automated
  judgment is worse than the failure mode of a queue.
- M1 ships no product feature. Freezing the extension contract before the first
  real purpose is deliberate, to avoid hard-coding a purpose into Core and
  rewriting it later.

## Follow-Up

- `knowledge/plans/2026-10-02-ai-assist-platform.md`
