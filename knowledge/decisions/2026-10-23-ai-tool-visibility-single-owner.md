# Decision: AI tool visibility has one editing surface

Date: 2026-10-23

## Status

Accepted and implemented (2026-10-23).

Related: `decisions/2026-10-19-ai-tool-catalog-console.md`,
`decisions/2026-10-12-ai-purpose-registry.md`,
`decisions/2026-10-12-ai-plugin-tool-exposure-policy.md`

## Context

`purposes[id].tools.allowed` is the single stored fact for "which tools may this
purpose offer the model". Runtime resolution is unchanged: a present `tools`
object (including `allowed: []`) wins over the purpose declaration.

Two console surfaces edited that same field:

- `?tab=purposes` (`SFAdminAIPurposesBlock`) rendered a per-purpose checkbox
  list, a free-text tool-name input, and explicit "follow the declaration /
  custom" buttons.
- `?tab=tools` (`SFAdminAIToolsTab`) rendered the cross-purpose tool matrix.

The duplication produced three concrete inconsistencies:

1. The tools-tab intro claimed the tab could only edit console English copy,
   while its purpose columns already wrote `purposes[id].tools.allowed`.
2. `purposeOverridden` used `Boolean(override.tools)`, while
   `purposeToolsAreOverridden` required `allowed` to be an array. The backend
   treats `{"tools":{}}` as an explicit empty allowlist, so the same document
   could show "customised" in one place and "follow the declaration" in another.
3. The purposes tab's "custom" action could pin an override equal to the
   declaration, producing a permanent "customised" state; the tools tab's
   `setPurposeTool` auto-converged the same edit back to no override.

Two capabilities lived only in the purposes tab: naming a tool before it is
registered (the backend deliberately does not validate existence on save), and
a per-purpose reset to the declared list.

## Decision

### 1. The tools tab is the only editing surface

`/control-panel/ai?tab=tools` is the sole place that writes
`purposes[id].tools.allowed`. The purposes tab keeps cost tier, failure
posture, blast radius, prompt version and a read-only summary: effective tool
count, declared/customised badge, and a deep link to `?tab=tools`.

### 2. The summary is read-only, not a second editor

The purposes tab may show the effective count and the tool names in a `title`
attribute. It must not render checkboxes, text inputs, or follow/custom
buttons for tools. "Follow the declaration" remains representable as the
absence of `purposes[id].tools`; the UI no longer needs a separate mode switch.

### 3. Per-purpose reset lives in the matrix header

When a purpose has a tool override, its column header shows a reset action.
`resetPurposeTools` removes `tools` and preserves `failurePosture`; the caller
deletes `purposes[id]` when the result is empty. Resetting tools never changes
the failure posture, and editing the failure posture never drops the tool
override.

### 4. Manual naming moves to a dedicated references block

`SFAdminAIToolReferencesBlock` owns two related jobs:

- list names that are referenced but not registered, with a per-purpose
  remove action (existing `referencedButUnregistered` behaviour);
- name a tool that is not registered yet, so an operator can prepare a
  allowlist before installing the plugin.

Manual naming accepts only unregistered names. Registered tools stay in the
matrix; the same edit has one obvious entry point. Additions respect
`MaxToolsPerRequest`; removals never do.

### 5. Frontend and backend agree on `{"tools":{}}`

`purposeToolsAreOverridden` reports true when the `tools` object is present,
matching `PurposeSettings.ToolAllowlist()`. `selectedPurposeTools` treats a
missing `allowed` as an empty set, and `prunePurposeOverrides` normalises the
shape to `{"tools":{"allowed":[]}}` so a save and reload produce the same
document.

### 6. No backend, API, or migration changes

`PurposeSettings`, `reply.tools` compatibility, `PurposeResolver` precedence,
and the `GET/PUT /admin/ai/settings` payload stay as they are. Existing
settings documents keep working; only the editing surface moves.

### 7. The references block is a separate component

`SFAdminAIToolsTab` was already above the 500-line review threshold. The new
interaction lives in `SFAdminAIToolReferencesBlock`, which has its own V3
component identity and its own focused test file.

## Consequences

- Operators see one place to change tool visibility; the purposes tab still
  answers "how many tools does this purpose expose" without duplicating the
  editor.
- A purpose's tool list and failure posture remain independent decisions, and
  the reset action makes "follow the declaration" reachable in one click.
- "Configure the allowlist before installing the plugin" remains possible.
- The tools intro now describes what the tab actually does.
- The purposes tab's restore button still resets both posture and tools; its
  tooltip says so.
- `{"tools":{}}` can no longer produce contradictory UI states.

## Known limitations

- The purposes tab shows a count, not the full name list; the names are in the
  `title` attribute and in the tools tab.
- Manual naming cannot create a purpose that the registry does not declare;
  unknown purpose ids remain an API-only shape.
- No per-tool usage statistics view yet (unchanged from the 2026-10-19
  decision).
- The tools tab still scrolls its matrix inside its own container on narrow
  screens; the new controls follow the same compact/aria contract.

## Verification

- Focused admin tests: `adminAIPurposes`, `adminAIToolsTab`,
  `adminAIToolsUtils`, `adminAIToolReferences` — 92 pass / 0 fail.
- Full web suite 1566 pass / 0 fail; Nuxt typecheck exit 0; architecture
  boundary gate and V3 catalog validator pass.
- Rendered QA at `1633x810` and `390x844` for both tabs: no page-level
  horizontal overflow, the deep link switches both URL and content, reset
  converges in the draft without touching the stored document, a manual name
  persists across save and reload, and remove cleans it up. The QA document
  was restored to its pre-test state.
