# Decision: The AI console owns one model-tool catalog

Date: 2026-10-19

## Status

Accepted and implemented (2026-10-20). `./scripts/test.sh` passed and the
desktop plus `390x844` rendered QA is recorded in
`sessions/2026-10-19-ai-tools-tab-handoff.md`.

Related: `decisions/2026-10-12-ai-purpose-registry.md`,
`decisions/2026-10-12-ai-plugin-tool-exposure-policy.md`,
`decisions/2026-10-18-ai-tool-context-and-catalog.md`,
`decisions/2026-10-19-ai-tool-call-visibility-and-prompt-tool-section.md`

## Context

The AI console could already name tools per purpose
(`purposes[id].tools.allowed`) and `GET /admin/ai/settings` already returned
`availableTools[]`, but that catalog carried only `name`, `description` and
`origin`. An operator could not see a plugin tool's declared timeout, what
arguments it takes, which purposes actually expose it, or whether a purpose
could call tools at all.

Two gaps had no answer anywhere:

- Plugin tools had no way to describe themselves in English for operators
  without changing the model-visible `description`. Translating that text would
  change the system prompt, the result cache key and the eval baseline.
- A whitelist can outlive the tool it names (plugin disabled, uninstalled or
  renamed). The server skips unregistered names silently, so the console showed
  a checked box for a tool the model could never see, with no way to clean it up.

Two statements in the module note and `host-api-v2.md` were also stale: both said
tool registration only happens during API assembly, so enabling or disabling a
plugin needed an API restart. That stopped being true once `aiToolRefresher`
began rebuilding the registry after every successfully applied runtime set;
`Available()` reads the live table.

`ai_tool_calls` (migration `202610190005`) now records every tool call of a
reply, with a public projection and an admin run replay. A tool-call data plane
exists; a per-tool statistics view does not.

## Decision

### 1. One tools tab is the single operator surface for registered tools

`/control-panel/ai?tab=tools` is the sixth fixed tab. It lists every registered
tool from `availableTools[]` with:

- the model-visible description and the console English copy, labelled by
  source (operator override or declared default);
- origin: `Core`, or `@extensionId` plus version;
- the declared plugin timeout;
- a bounded parameter summary (top-level names and required entries);
- which purposes currently expose it, and a per-purpose exposure toggle;
- a cleanup entry for names that are referenced but no longer registered.

The tab cannot create tools. Tool declarations stay in code and manifests; the
tab edits console copy and purpose allowlists only.

Plugin tools whose extension declares a settings page also carry a jump link to
that page. The Host exposes `settingsPath` only when the manifest really has a
settings document (or a `/settings` admin page); the console never guesses the
route from the extension id, because a guessed link turns "where do I configure
this plugin" into a 404.

### 2. Three description sources, one fixed precedence

| Source | Where it lives | Who sees it |
| --- | --- | --- |
| Operator override | `settings.toolDisplayDescriptions[tool][locale]` | Console |
| Declaration | manifest `aiTools[].descriptionI18n`, or Core `DisplayDescriptions()` | Console |
| Model description | manifest `description` / `Definition().Description` | Model |

Precedence is **operator > declaration > none**, and it cannot be reversed: an
operator fills a gap, they do not rewrite the model's instructions from a text
box.

The model-visible description is deliberately **not** localized.
`RenderToolSection` renders `Definition().Description` into the system prompt
byte-for-byte. A second, translated model text would turn a console edit into a
prompt change, a cache-key change and an eval-baseline change.

### 3. `descriptionI18n` is a map, not `LocalizedText`

`LocalizedText` carries a custom `MarshalJSON`, and Go's `omitempty` does not
apply to struct fields, so a zero value would serialize as `""`. Every old
manifest's canonical form would gain a field and
`protocolRuntimeManifestDigest` would drift for packages nobody touched. A
`map[string]string` with `omitempty` serializes away when absent, so old
manifests keep a byte-identical canonical form and digest.

The field is optional. `manifestVersion` stays `3`, authors do not bump
`aiTools[].contractVersion`, and `validateAITools` gains exactly one rule
(`v3 aitools rule #14`): every value is non-empty and at most 1024 bytes, keys
must already be canonical locale tags, and at most 8 locales are allowed. A
package that adopts the field gets a new manifest digest — that is the normal
exact-artifact rule, not a compatibility break.

### 4. Operator overrides are console-only, validated and normalized

`settings.toolDisplayDescriptions` is a map of tool name → locale → text. It is
part of the settings document, so it is validated like the rest: valid tool
name, locale-key shape (a 2–3 letter language plus an optional region,
case-insensitive), at most 8 locales per tool, at most 64 tools, at most 1024
bytes per value. Normalization trims and drops empty entries; it does not
rewrite invalid ones — rejecting with a field path is better than silently
discarding what an operator wrote.

Existence is deliberately not checked on save: the registry changes with the
plugin lifecycle, so pinning a name would make "write the copy now, install the
plugin later" impossible. The console filters against the live catalog.

### 5. Exposure edits auto-converge

`setPurposeTool` materializes the effective set, adds or removes the name, then
compares the result with the purpose's declared set. When they match, the whole
`purposes[id].tools` field is deleted. Without this, "check then uncheck" would
leave an override equal to the declaration and the console would claim
"customized" forever. `removeToolFromAllPurposes` applies the same convergence
per purpose and leaves overrides for purposes outside the current catalog
untouched.

### 6. No global tool switch

Per-purpose overrides are already the single source of truth. A global switch
would create a second answer to "can this tool be called", and the console would
have to explain which one wins.

### 7. Unregistered references are visible and removable

`referencedButUnregistered` lists every name in an effective allowlist that the
registry does not know, together with the purposes that reference it. The server
keeps skipping them; the console must show them, because a silent skip looks
exactly like a working configuration.

### 8. Three diagnostics, not one boolean

- `gates.toolCallsPerReply` resolves to `0`: the tool budget is off.
- The purpose's cost class has no bound profile.
- The bound profile declares `supportsTools: false`.

Each has a different fix, so they are reported separately instead of collapsing
into "tools are unavailable".

### 9. Saving settings invalidates the result cache

`ai_settings.revision` increments on every save (`revision = revision + 1`), and
`CacheKey` includes the revision. Saving console copy therefore invalidates
cached completions even though no model input changed. This is accepted: the
alternatives are a second revision column or field-level cache keys, and both
add a permanent correctness surface to save a few cache entries.

### 10. The data plane exists; the statistics view does not

`ai_tool_calls` stores each call after a successful publication (`seq` is the
trace slice index, `UNIQUE (comment_id, seq)` makes River retries idempotent).
The public projection rides `GET /comments/:id/citations` as `toolCalls`, and
admin replay reads it through the run timeline. A per-tool call/failure
statistics view is follow-up work; the system must not be described as having no
tool-call data plane.

## Consequences

- Operators can answer "who provides this tool, how long may it take, what does
  it need, and who can call it" without reading code or querying the database.
- Plugin authors get a console translation channel that cannot accidentally
  change model behaviour, and old plugins keep their digest.
- Core ships `en-US` console copy for the nine built-in tools
  (`Models/AITools/descriptions.go`). The Chinese column is the model-visible
  description itself, so no second Chinese copy is declared.
- `availableTools[]` grows three optional fields (`descriptionI18n`,
  `timeoutMs`, `parameters`); `origin` is unchanged, and consumers that ignore
  unknown fields are unaffected.
- The tab shows the declared plugin timeout read-only. There is still no
  site-level tool budget gate (operator decision), so the declaration remains
  the only upper bound an operator can see.

## Known limitation

- Display-copy saves invalidate the result cache (Decision 9).
- No per-tool usage statistics UI yet (Decision 10).
- The tab reports diagnostics; it does not fix them. Binding a cost class or
  enabling a provider stays in the providers tab.
- A package's manifest digest changes when it starts declaring
  `descriptionI18n`. There is no way to add console copy without a new artifact,
  and that is the intended exact-artifact behaviour.
