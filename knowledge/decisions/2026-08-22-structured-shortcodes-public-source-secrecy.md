# Decision: Structured Shortcodes And Public Source Secrecy

Date: 2026-08-22  
Status: Accepted for Built-in Shortcodes V1 (M0)

## Context

SForum will provide eight shortcodes through the protected built-in plugin
`sforum-shortcodes`. Five shortcodes render public references and three protect
nested content according to the current viewer. The current Forum model cannot
safely host protected bodies because one `RenderedContent` value is serialized
to public topic and comment responses and includes canonical `rawContent`.

The Host must own storage, parsing, validation, authorization, public
projection, execution limits, and fallback. The plugin may declare and render
the product shortcodes, but it must not become an alternate content store or an
authorization authority.

This decision freezes the contracts that M1-M10 must implement. It does not
change runtime behavior by itself.

## Decision

### Contract identities

The Host owns exactly two Tiptap node types:

| Node | ProseMirror role | Content |
| --- | --- | --- |
| `sforumShortcodeRef` | block atom | none |
| `sforumShortcodeBlock` | block container | admitted block content |

Both nodes have exactly three attributes:

```json
{
  "id": "sforum-shortcodes.topic",
  "contractVersion": "sforum-shortcodes.topic@1",
  "arguments": { "topicId": 123 }
}
```

- `id` is a declared lowercase ASCII identity.
- `contractVersion` must exactly match the active or historically admitted
  declaration for `id`.
- `arguments` is an object and is never omitted. Undeclared attributes fail
  structured-document validation.
- Client HTML, object snapshots, authorization results, rendered cards, and
  secrets are never node attributes.

V1 freezes these declarations:

| Syntax name | Node | ID and contract version | Canonical arguments |
| --- | --- | --- | --- |
| `user` | ref | `sforum-shortcodes.user@1` | `userId`: positive int64 |
| `topic` | ref | `sforum-shortcodes.topic@1` | `topicId`: positive int64 |
| `comment` | ref | `sforum-shortcodes.comment@1` | `commentId`: positive int64 |
| `category` | ref | `sforum-shortcodes.category@1` | `categoryId`: positive int64 |
| `friend-links` | ref | `sforum-shortcodes.friend-links@1` | none |
| `login` | block | `sforum-shortcodes.login@1` | none |
| `reply` | block | `sforum-shortcodes.reply@1` | none |
| `only-author` | block | `sforum-shortcodes.only-author@1` | none |

The canonical name is `friend-links`. `friend_links` is not a V1 alias and
remains literal input. `topic-tag` is the only V1 legacy alias; it imports as
`category`, maps `tag_id` to `categoryId`, and never survives canonical
storage or export.

Username search is editor UI assistance, not a stored or bracket argument.
The Host resolves it to a stable positive `userId` before node admission.

### Argument limits and validation

The Host applies all limits before persistence and repeats relevant limits
before execution:

- at most 32 shortcode nodes per document;
- at most four nested shortcode levels, counting the outer shortcode as level
  one;
- at most 16 argument entries per node;
- canonical argument keys match `[a-z][A-Za-z0-9]*` and are at most 64 ASCII
  bytes;
- bracket argument keys match `[a-z][a-z0-9_]*` and must map to an exact
  declared canonical key;
- V1 generic scalar values are non-null strings, booleans, or signed int64
  integers; floats, arrays, and nested objects are rejected;
- strings are at most 512 Unicode code points;
- duplicate, unknown, missing, or type-mismatched arguments are invalid;
- render recursion is at most four levels and reference cycles are detected by
  `(resource type, stable ID)`;
- Content Registry batch, segment, deadline, and total-output limits remain
  additional hard ceilings.

The eight built-in schemas are stricter than the generic scalar envelope: the
four ID references each require one positive int64 and the other four
declarations require an empty object.

### Placement and nesting

Both Host nodes are block nodes. They may occupy a complete document block or
a complete block inside an admitted `sforumShortcodeBlock`. They cannot occur
inside a paragraph, heading, inline mark, table cell, list item text, code
block, or another inline context.

Reference nodes are atoms and have no body. Protected block nodes require a
body containing ordinary admitted block content; that body may include nested
shortcode blocks or references within the four-level limit. An empty protected
body is invalid. `only-author` is admitted only in comment documents. The
other seven declarations are admitted in topic and comment documents.

### Bracket grammar V1

The language contract is `sforum.shortcode-text@1`.

- Names use lowercase ASCII letters, digits after the first character, and
  `-`. Only the frozen declared names and the `topic-tag` alias activate.
- Opening and closing tags are case-sensitive. Closing tags are `[/name]`.
- A shortcode tag must occupy a standalone logical line after indentation is
  removed. Surrounding prose on the same line keeps the text literal.
- Reference syntax has an explicit matching closing tag and an empty body.
- Protected syntax has an explicit matching closing tag and preserves its
  admitted Markdown/block body.
- Canonical arguments use `key="value"`. Integer arguments export as quoted
  base-10 values without a sign or leading zeroes. Import may accept an
  unquoted positive decimal integer and normalizes it.
- Canonical export orders arguments by the declaration schema.
- `\[name]` is literal. Export escapes a literal declared/alias opening tag so
  a later import cannot activate it accidentally.
- Inline code, indented code, fenced code, code-mark content, and raw HTML are
  opaque to shortcode parsing.
- Unknown, malformed, mismatched, repeated-argument, over-limit, and
  placement-invalid textual syntax remains literal authoring/import text.
- A structured API node that falsely claims a shortcode identity, version,
  schema, placement, or limit fails validation; it is not converted to text.

The canonical forms are therefore:

```text
[user user_id="42"][/user]
[topic topic_id="123"][/topic]
[comment comment_id="456"][/comment]
[category category_id="7"][/category]
[friend-links][/friend-links]
[login]
protected blocks
[/login]
```

`sourceFormat=markdown` API submissions participate in this parser. When an
admitted shortcode is found, the Host parses the complete Markdown document,
normalizes it to the Host nodes, accepts it through EditorDocument, and stores
canonical `sourceFormat=editor-document` JSON. Ordinary Markdown with no
admitted shortcode remains on the existing Markdown path. HTML submissions do
not parse bracket syntax. Paste, explicit import, legacy conversion, Go, and
TypeScript must use one language-neutral conformance corpus.

### Storage and versioning

Canonical shortcode documents use the existing Host-owned
`sforum.editor-document@1` envelope. Adding these explicitly versioned nodes
does not by itself change that storage version. A storage-version bump is
required only when an old Host cannot safely identify, preserve, validate, and
fail closed on the document shape.

Each declaration version changes when its argument schema, placement,
authorization meaning, visible success semantics, or fallback secrecy changes.
Copy-only localization changes do not change the declaration version.
Grammar changes require a new `sforum.shortcode-text` version and paired
fixtures.

Accepted old source and revisions remain interpretable. Upgrades may introduce
a new declaration version, but new writes cannot silently rewrite historical
source on read. Any bulk migration must be explicit, resumable, reversible from
accepted revisions, and must rebuild derived HTML/plain/search data. Rollback
must retain a Host fallback for every historically admitted protected node.

### Public, editable, and administrative projections

One DTO must no longer serve all three trust levels.

1. The public rendered projection contains sanitized or personalized HTML,
   public-safe plain text and excerpt, and non-sensitive render metadata. It
   contains no canonical source, editor metadata needed only for re-editing,
   protected child, or hash derived from canonical source.
2. The editable source projection contains the canonical source format/value,
   editor metadata, content hash, attachment references needed for editing,
   and current revision token. It is returned only by dedicated topic/comment
   edit-source reads after the same server-authoritative edit-own/edit-any,
   resource-state, and edit-window policy used by mutation.
3. Existing revision and admin projections retain raw-source access behind
   their existing permission checks. Deleted or otherwise non-editable source
   is not recovered through the author endpoint; protected admin/revision
   authority remains the recovery path.

M1 removes `rawContent` immediately from public topic, slug, comment list,
reply, create, update, Page Registry, and Nuxt payload contracts. There is no
compatibility window that continues the leak. Public `contentHash` is also
removed unless it is redefined as a hash of the public projection; the current
raw-source hash is an oracle and is not public metadata.

The dedicated read paths are:

- `GET /topics/{topicId}/edit-source`
- `GET /comments/{commentId}/edit-source`

Frontend edit flows must use those reads. Public actor-independent cache
entries never contain editable source.

### Public projection and locale

Protected descendants are removed while deriving the public base, before any
public HTML, plain text, excerpt, search, SEO, event, cache, or log value is
created. CSS hiding and client-side authorization are forbidden.

Stored `posts.plain_text`, excerpts, search documents, and SEO summaries use a
localized non-secret placeholder in the site's default locale at write time.
A default-locale change therefore requires a bounded content re-render and
search reindex operation. Request-time HTML notices use the request locale.
The stored/public `plainText` remains actor-independent even when HTML is
personalized for an authorized viewer.

Reference nodes contribute only stable default-locale labels to stored search
text in V1. Resource titles, usernames, or target IDs are not indexed through
shortcode rendering. A later visibility-aware indexing design requires its own
contract and invalidation proof.

### Authorization

Host policy is authoritative:

- `login`: reveal only to a current active authenticated actor;
- `reply`: reveal only to the topic author or a current active actor with at
  least one currently public accepted comment on that same topic;
- `only-author`: comment-only; reveal only to the comment author or owning
  topic author;
- a staff session on a public page receives no implicit bypass;
- reference resources render only after current Host visibility checks;
- comment references additionally require the owning topic to be public.

The plugin receives the minimum Host decision and public projection needed to
render. It receives no raw session, email, IP, moderation state, private avatar,
Core table access, or unfiltered resource body. A protected child fragment is
provided only after the Host authorizes the viewer.

### Side effects from protected descendants

Host safety and moderation validation still inspect accepted protected child
content. Size rules, unsafe-link policy, sanitizer admission, attachment
authority, and author trust restrictions cannot be bypassed by nesting content
in a protected block. Errors and diagnostics must not echo that content.

Protected descendants do not generate mention notifications, link previews,
outbound link projections, observe-event fields, or webhook fields in V1.
This avoids notifying an actor who cannot independently read the block and
keeps side effects from becoming a content-existence oracle.

### Caching

- Cache only an actor-independent public base and immutable bounded render
  plan.
- Never put canonical source in public topic/comment Redis entries.
- Personalized HTML is `private, no-store`; it is composed after the shared
  cache boundary.
- Identical public reference lookups are batched per response. Plugin result
  caches carry Host cache tags and cannot outlive visibility changes.
- No viewer identity, authorization result, or unlocked fragment is stored in
  a cross-actor cache.

### Fallback

All fallback copy is localized, SSR-visible, accessible, bounded, and contains
no child body, private target metadata, IDs, raw runtime error, or artifact
detail.

Core owns lifecycle-independent translations for these stable fallback reason
codes; the plugin owns catalog/help and successful-render copy, but cannot be a
dependency of disabled, uninstall, Safe Mode, stale-artifact, or crash output:

| Reason code | Use |
| --- | --- |
| `shortcode.reference.unavailable` | invalid declaration/output or generic reference failure |
| `shortcode.user.unavailable` | user renderer unavailable |
| `shortcode.topic.unavailable` | public topic renderer unavailable |
| `shortcode.topic.not_found` | topic is non-public or cannot be confirmed |
| `shortcode.comment.unavailable` | public comment renderer unavailable |
| `shortcode.comment.not_found` | comment/owning topic is non-public or cannot be confirmed |
| `shortcode.category.unavailable` | category renderer unavailable/non-public |
| `shortcode.protected.unavailable` | protected runtime unavailable; child hidden |
| `shortcode.login.required` | viewer fails `login` policy |
| `shortcode.reply.required` | viewer fails `reply` policy |
| `shortcode.only_author.private` | viewer fails `only-author` policy |
| `shortcode.friend_links.omitted` | trace-only reason; no public output |

Public response metadata may expose only these stable codes, never plugin
runtime details. Core translations are part of the Host contract and remain
available without the exact plugin artifact.

| Declaration | Runtime unavailable/disabled | Unauthorized or non-public | Invalid declaration/output |
| --- | --- | --- | --- |
| `user` | localized user-reference unavailable | generic unavailable | generic unavailable |
| `topic` | localized topic-reference unavailable | not-found style, no metadata | generic unavailable |
| `comment` | localized comment-reference unavailable | not-found style, no metadata | generic unavailable |
| `category` | localized category-reference unavailable | generic unavailable | generic unavailable |
| `friend-links` | no public output; trace reason only | not applicable | no output; trace reason |
| `login` | protected-content unavailable; body hidden | login-required; body hidden | body hidden |
| `reply` | protected-content unavailable; body hidden | reply-required; body hidden | body hidden |
| `only-author` | protected-content unavailable; body hidden | private-comment; body hidden | body hidden |

For a reference failure, a fallback may link to a stable Core URL only when
the Host has just rechecked that target as currently public. If visibility is
false, invalid, stale, timed out, or cannot be established, fallback is a
non-interactive label. The link uses the current Host-produced public path
(profile, configured canonical topic path, owning-topic comment anchor, or
category path); it is never reconstructed from untrusted node arguments.
Protected fallback is always non-interactive.

The generic Content Registry `preserve_source` label is not an acceptable
protected-body fallback. Protected declarations require a closed Host-owned
fallback even if the plugin or registry is absent.

### Events, plugin calls, logs, and traces

- Public observe events and webhooks do not gain body fields.
- The four existing synchronous authoring filters
  `topic.before_create`, `topic.before_update`, `comment.before_create`, and
  `comment.before_update` remain trusted write-time processors and may receive
  `ContentInput`. They are not public projections; any future redaction is a
  separately versioned extension-contract change.
- Renderer requests receive declared arguments, a minimal Host projection,
  locale/scope, and an authorized accepted fragment where required, never the
  complete Forum `RenderedContent` or raw document.
- Logs and traces record bounded identities, versions, outcome/fallback reason,
  duration, and aggregate counts only. They never record arguments, child
  text, serialized documents, rendered output, actor identity, raw errors, or
  cache values.
- Error responses use stable codes and never echo source fragments.

## Implementation choice

V1 adds no shortcode parser dependency. Go uses the already-present Goldmark
AST/parser/renderer extension interfaces; the editor uses the already-present
Tiptap node, Markdown tokenizer/parse/render, input-rule, and paste-rule APIs.
Both implementations are subordinate to the shared conformance corpus and
server-side validation. See the M0 report for the library comparison.

## Consequences

- M1 is a deliberate breaking correction to the public Forum JSON contract;
  repository clients and OpenAPI change in the same milestone.
- M2 must extend `EditorDocument.NodeSpec`; the current `Atom` plus attr
  allowlist model cannot express the frozen block container, typed arguments,
  placement, and protected fallback semantics.
- M3 cannot claim production execution while the Forum bridge remains an
  identity-only HTML filter.
- M4 must expose bounded Host query/authorization projections before a built-in
  renderer is created.
- Default locale changes require re-render/reindex support once shortcodes are
  stored.
- Secret-marker tests must scan JSON, SSR, hydration, cache, search, SEO,
  events, webhooks, traces, logs, and plugin requests.

## Rejected alternatives

- Raw bracket text as executable storage: loses typed validation and safe
  editor round trips.
- Per-shortcode Tiptap storage node types: lets plugin product declarations
  redefine the Host storage schema.
- Hugo's shortcode engine: mature but coupled to Hugo page/context semantics
  and a different grammar; it would add a large framework dependency.
- A new standalone shortcode parser: no surveyed Go package matched the
  maintenance, documentation, license clarity, and ecosystem fit of extending
  Goldmark already in the repository.
- Public `rawContent` with conditional field hiding: cache and serializer paths
  can regress; separate DTOs make the trust boundary structural.
- Client-side reveal or CSS hiding: protected bodies remain recoverable from
  HTML, hydration, and page payloads.
- Generic `preserve_source` for protected failures: its label is not a proof
  that protected descendants were excluded from derived output.

## References

- `../plans/2026-08-22-builtin-shortcodes.md`
- `../reports/2026-08-22-builtin-shortcodes-m0-threat-inventory.md`
- `2026-07-06-tiptap-editor-content-storage.md`
- `2026-07-13-trusted-plugin-theme-platform-v3.md`
- Goldmark: <https://github.com/yuin/goldmark>
- Tiptap Markdown: <https://tiptap.dev/docs/editor/markdown>
- Tiptap custom nodes: <https://tiptap.dev/docs/editor/extensions/custom-extensions/create-new/node>
- Hugo shortcodes: <https://gohugo.io/content-management/shortcodes/>
