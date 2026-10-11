# Decision: Plugin AI Tools Are Exposed By An Explicit Operator Allowlist

## Status

Accepted (2026-10-12). Implemented in the same change.

## Context

M2 registers every declared plugin tool into the Host tool registry, and the
reply purpose exposes the model only what its own allowlist names. That allowlist
was a Go constant (`aireply.ReplyToolAllowlist`, six Core tools), so a plugin tool
could never become visible no matter how the operator felt about it. The M2
decision record flagged the exposure question as something that must be answered
before the M4 purpose registry lands, and explicitly ruled out hand-editing the
Core allowlist as the way to connect it.

## Decision

### 1. The allowlist lives in the AI settings document, not in plugin code

The reply purpose reads `reply.tools.allowed` from the site settings:

```json
"reply": { "tools": { "allowed": ["forum-search", "vendor-plugin-tool"] } }
```

- Absent (`tools` null) → the purpose's declared Core list, i.e. today's
  behaviour. Recommended defaults therefore expose nothing new, and the reply bot
  cannot be widened by a plugin install.
- Present → exactly the listed names, including Core names the operator chose to
  remove. An empty list is a valid configuration meaning "the reply may call no
  tools at all".

### 2. Exposure is operator-gated, and the gate is re-read per reply

Installing a plugin never makes its tool reachable. An operator names it. The
value is read per reply (like the system prompt), so a change takes effect without
a restart, and the source reports "unset" separately from "empty" — collapsing the
two would let one settings save silently disable every tool.

### 3. Shape is validated on save; existence is resolved on use

On save the Host rejects invalid names (charset, duplicates, more than
`MaxToolsPerRequest` entries — a longer list cannot be honoured by one request).
Existence is deliberately *not* checked on save: the registry changes as plugins
are enabled or disabled, so pinning it would make "write the policy now, install
the plugin later" impossible. At request time the registry skips unregistered
names, which is already how the tool descriptor path behaves.

### 4. A settings read failure falls back to the declared Core list

If reading settings fails, the reply uses the purpose's declared allowlist rather
than an empty or a permissive one. Plugin tools must never appear because of an
error, and Core tools must not disappear because of one.

## Incidental fixes recorded here

- **The settings DTO dropped `reply` entirely.** The AI console has been sending
  `reply.systemPrompt` in its settings payload, but `updateSettingsRequest` had no
  such field and `writeSettings` replaces the whole document, so saving any AI
  setting silently erased an operator's reply prompt. The DTO now round-trips
  `Reply` (prompt + tool overrides), and a controller test pins it. This is the
  kind of bug a "field exists in the UI type but not in the transport type" gap
  produces: the UI looked like it worked.
- The reply generator now takes the allowlist source alongside the prompt source,
  so both operator overrides travel the same path and both are read per reply.

## Consequences

- Operators control model-visible tooling without a new permission and without
  editing code: the AI settings document is already the AI policy surface, and it
  is covered by `ai.manage`.
- The console UI for `reply.tools.allowed` is not part of this change. It needs a
  settings block plus rendered QA (desktop and 390x844), and it belongs with the
  M4 purpose registry work so the control can be per purpose rather than
  reply-specific.
- Purpose-specific shape is the known limitation of this step: the field is
  `reply.*`, not a purpose map. M4 replaces it with declared purposes
  (`ai.purpose` registry) and this field becomes the reply purpose's entry in that
  registry rather than a special case.
