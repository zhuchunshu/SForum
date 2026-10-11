# Decision: AI purposes are a declared registry

Date: 2026-10-12
Status: accepted (implemented in the same change)

Related: `decisions/2026-10-02-ai-provider-gateway.md`,
`decisions/2026-10-12-ai-plugin-tool-exposure-policy.md`,
`plans/2026-10-02-ai-assist-platform.md` (M4 content purposes),
`plans/2026-10-11-ai-plugin-host-api-m1.md`

## Context

Every AI completion already belongs to a purpose, but the purpose was never a
declared thing. Its facts were scattered across the call sites:

- the reply tool allowlist was a generator-local constant
  (`aireply.ReplyToolAllowlist`),
- the cost class was hardcoded in the completer (`CostClassEconomy`),
- the failure posture only existed as `settings.purposeFailurePosture[purpose]`
  with an implicit `fail_open` default,
- "which purposes exist on this site" had no answer at all — the console could
  not enumerate them and the gateway accepted any string as a purpose.

M2 registered plugin tools into one Host registry and made exposure an operator
allowlist, and its decision record explicitly deferred the structural question
here: the allowlist was `reply.*`, not a purpose map.

## Decision

### 1. A purpose is a declaration, and the registry is the only enumerable answer

`app/Support/AI/purpose.go` defines `PurposeDeclaration`:

| Field | Why it is a declaration and not a call-site detail |
| --- | --- |
| `ID` | The name that appears in execution records and usage attribution. |
| `Label` / `Description` | The console renders a human list; a nameless purpose must be impossible. |
| `CostClass` | The purpose owns the tier. The caller never picks a model or a profile. |
| `FailurePosture` | How this purpose degrades when AI is unavailable. |
| `ToolAllowlist` | The declared default set of model-visible tools. |
| `ResourceType` | What kind of resource the purpose serves, for cost reverse-lookup. |
| `PromptVersion` | Participates in the cache key and execution records. |
| `BlastRadius` | `single` / `batch` / `site` — how far a failure reaches, for operator risk reading. |

`PurposeRegistry` rejects duplicate ids and any declaration that fails
`ValidatePurposeDeclaration` (charset, cost class, posture, blast radius, tool
names, and "the allowlist cannot exceed one request budget"). The registry is
built at assembly time and is read-only afterwards: adding or removing a purpose
at request time would let the cost class and tool boundary of one conversation
change mid-flight.

`CorePurposeForumReply` (`forum.reply`) is the first entry and the only one
today. `DefaultPurposeRegistry()` is a process-wide singleton so that "what
does the reply purpose declare" cannot have two answers inside one process.

### 2. Purpose resolution has one implementation, and the precedence is fixed

`PurposeResolver.Resolve(purpose)` merges the declaration with the site
override. Precedence is **site per-purpose override > declared default**, and it
cannot be reversed: the declaration is the capability boundary written by Core
(or, later, a plugin), and the operator can only choose within a declared
purpose — never invent one.

The resolver is shared by the gateway (rejection), the service layer (reply
parameters) and the console (catalog), so the console's "effective value" and
the runtime's actual value cannot diverge.

### 3. An undeclared purpose is denied, and the site switch keeps its single answer

`Gateway.Execute` validates the purpose **after** the site switch and **before**
the gates: a disabled site still answers only `gateway_disabled` (one reason for
"why was there no outbound call"), and an undeclared purpose is denied with
`purpose_unknown` plus a denied execution trace, with **zero** outbound calls.

`GatewayConfig.Purposes` is optional so an older deployment or a unit-test
harness that never injected a registry keeps working; every assembly path
injects it (`Providers.NewAIProvider`), and `ai.DefaultPurposeRegistry()` is the
fallback the service layer uses when it is not given one.

### 4. Site configuration mirrors the registry: `purposes`

`settings.purposes` is a map of `PurposeSettings` per purpose id:

```json
"purposes": {
  "forum.reply": { "tools": { "allowed": ["forum-search"] }, "failurePosture": "fail_closed" }
}
```

- An absent purpose or absent field means "keep the declared default".
- `tools` present (including `{"allowed": []}`) means the operator took over the
  allowlist; an empty list is a valid configuration ("this purpose may call no
  tools at all").
- Shape is validated on save; **existence is deliberately not**. The registry
  changes with Core versions and (later) plugin-contributed purposes, so pinning
  a name at save time would make "write the policy now, add the purpose later"
  impossible. Unregistered entries are kept and ignored at resolve time.

### 5. The reply special case is replaced, not extended

`reply.tools` (introduced with the M2 exposure policy) is now the *legacy*
field:

- `Settings.Normalized()` migrates it into `purposes["forum.reply"].tools` when
  the purpose is not already configured, so an upgrade keeps the operator's
  intent.
- The legacy field is **kept and still validated** rather than deleted, so a
  rollback to the previous version does not lose configuration, and a directly
  written legacy document still round-trips.
- A `purposes` entry always wins over the legacy field.

`purposes` entries that override nothing are dropped on normalize: an entry that
says nothing is indistinguishable from no entry, and keeping it would make the
console claim "customized" for a purpose nobody configured.

### 6. The reply prompt lives in the config only because the interface is reply-shaped

`reply.systemPrompt` stays where it is (one `LText`-editable prompt owned by the
reply console tab). Purposes do not carry prompts yet because no second purpose
exists, and a prompt-per-purpose interface with one consumer would be guessing.
When the first non-reply purpose lands (content purposes, M4), the prompt moves
into the purpose declaration and the config becomes
`purposes[id].systemPrompt`; `ReplyPromptSafetyAppendix` stays non-configurable
for every purpose.

## Consequences

- `GET /admin/ai/settings` now returns a `purposes` catalog (id, labels, cost
  class, resource type, prompt version, blast radius, declared vs effective
  tools, override flags) so the console can render purposes instead of reply
  special cases.
- `PATCH /admin/ai/settings` accepts `purposes`; the DTO round-trips it.
- The reply generator no longer owns a tool constant, a cost class, or a prompt
  version: it resolves all three through one `ReplyParameterSource` call, so
  "cost class from A, tools from B" is not expressible.
- Reading settings still cannot fail a reply: any resolve error falls back to
  the declared defaults (declared cost class + Core tool list + declared prompt
  version), never to "no allowlist" or "all tools".

## Known limitation (next step)

The console UI for per-purpose overrides is **not** in this change. It needs a
settings block plus rendered QA (desktop and `390x844`), and the operator's
running API is the pre-change binary, so the work has to follow an API restart.
The payload already carries the catalog; the remaining work is a tab/block that
edits `form.purposes`.
