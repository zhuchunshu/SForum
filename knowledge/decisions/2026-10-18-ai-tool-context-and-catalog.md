# Decision: AI Tool Context Whitelist and Registered Tool Catalog

## Status

Accepted (2026-10-18). Implemented in this change.

## Context

A plugin-contributed AI tool could only receive `{name, arguments}`. It had no
way to know which language the site speaks, which topic the conversation is
happening in, or which member triggered the call. The first real web-search
plugin made that concrete: without a locale it cannot ask the provider for
Chinese results, and without a member id it cannot keep a per-member quota.

Two facts constrained the design:

- `Support/AI.ToolContext` already carries `ViewerUserID`, `TopicID`,
  `CommentID` and `Locale`, and the reply generator already fills the first
  three. The value existed on the Host side; only the plugin dispatch dropped
  it.
- `Identity` has **no** per-user locale column. The only locale the Host can
  honestly supply is the site default (`options.site.default_locale`). A tool
  that receives `zh-CN` is being told "this site is Chinese", not "this member
  writes Chinese".

Separately, the AI console's purpose allowlist was a free-text box. It accepted
any string and the server silently skipped unregistered names, so the only way
to learn a plugin tool's exact name was to read the plugin's manifest. The tool
registry already knew every registered name, but nothing exposed it.

## Decisions

### 1. Context is a manifest whitelist, defaulting to `locale`

`aiTools[].context` lists `locale`, `topic`, `viewer`. Omitted means
`["locale"]`.

Why a whitelist rather than always passing everything: `viewerUserId` is an
identity disclosure to a third-party process. SForum already treats identity
and raw authority as powers that must be declared separately, and a plugin tool
that only searches the web has no business receiving member ids. Making the
default `["locale"]` keeps the common case zero-config while forcing an
explicit decision for the two facts that are not free.

Why `locale` and not "the member's language": there is no per-user locale to
read. Passing the site default and documenting it is honest; inventing a
per-user value would be a fabricated fact that plugins would build behaviour on.

Validation is exact-match (`Locale`, `ip`, duplicates and empties are rejected)
and is not normalised: a manifest is a declaration the author wrote, and a
mis-cased value should fail loudly rather than be silently rewritten into a
different meaning.

### 2. The request document grows additive fields; the schema stays `@1`

The tool invoke document (`sforum.ai.tool.invoke.request@1`) gains optional
`locale`, `topicId`, `viewerUserId`. The version is **not** bumped.

This follows the precedent already recorded for `sforum.ai.completion@1` when
image parts were added: additive optional fields stay backward compatible, and
a v1 plugin that ignores them keeps working byte-for-byte. A version bump would
have forced every existing plugin to re-declare, for a change none of them can
observe.

Fields are written only when the declaration listed them, so a tool that
declares nothing receives exactly the old shape. `TopicID`/`ViewerUserID` are
omitted when zero rather than written as `0`, because `0` already means "no
topic" in `ToolContext` and sending it would be noise.

### 3. The catalog is a top-level payload field, not per-purpose

`GET /admin/ai/settings` returns `availableTools[]` next to `purposes[]`.

The two answer different questions. `availableTools` is a global fact ("what is
registered on this site"); a purpose's allowlist is a per-purpose configuration
("what this purpose may use"). Nesting the catalog under each purpose would
imply they could differ, and would multiply the same list by the number of
purposes.

Each entry carries `origin` (`kind: core | plugin`, plus `extensionId` and
`version`) so the console can label a plugin tool without parsing its name.
`origin` is display metadata only: dispatch is unchanged and still keyed by
tool name.

### 4. Disabled plugins are not registered

`registerPluginAITools` iterated every extension returned by `ExtensionStore.List`
and never checked `Status`, despite its own doc comment saying "已启用插件".
That produced entries that could be named in an allowlist and would then fail on
every call, because a disabled plugin has no running runtime.

The fix is a `Status != StatusEnabled` skip. The catalog made the bug visible
(it would have listed phantom tools), but the bug predates it.

### 5. The console offers registered tools instead of only accepting typed names

The purposes tab now lists `availableTools` as checkboxes, labelled
`name · extensionId` for plugin tools, and keeps the free-text input as the
escape hatch. A selected name that is in neither the declaration nor the
catalog is badged as unknown, because the server will skip it.

The catalog is optional in the payload: an older API without `availableTools`
renders the previous interface rather than breaking the page.

## Alternatives Considered

- **Pass everything unconditionally.** Simplest, rejected: it discloses member
  ids to every tool, including ones that only need a locale.
- **Bump `sforum.ai.tool.invoke.request@2`.** Rejected: additive fields are
  already the established rule, and a bump would break every existing plugin
  for a change they cannot observe.
- **Put the catalog on the purpose rows.** Rejected: it would turn a global
  fact into a per-purpose one and duplicate it N times.
- **Derive locale per member from the request.** Not possible: no per-user
  locale exists in the schema, and inferring one from the message text would be
  a guess.

## Consequences

- `PluginTool.Invoke` now reads `ToolContext` and filters it by declaration; the
  `Origin()` method makes plugin tools self-describing to the catalog.
- `Models/AI.Service` gains `WithToolCatalog` (assembly-time injection, because
  the tool registry is built after the AI service) and `AvailableTools`.
- The reply generator resolves the site locale through an optional
  `LocaleSource` interface, so test and legacy assemblies without one keep
  working with an empty locale.
- New tools that want per-member behaviour must add `viewer` to `context` and
  will show that disclosure in their manifest review.
- Tool registration still happens during API assembly: enabling or disabling a
  plugin takes effect on the catalog after an API restart. That remains an
  open item, not something this change fixed.
