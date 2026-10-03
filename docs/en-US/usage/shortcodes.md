# Shortcodes (Built-in Shortcodes)

[← Usage](./README.md)

Shortcodes are structured reference and protected-content blocks that can be
embedded in topic/comment bodies. They are provided by the protected built-in
plugin **`sforum-shortcodes`**. The server converts them into normalized
document nodes and composes safe output for the current viewer; no shortcode
spelling is executable stored HTML.

## Supported syntaxes (8)

| Shortcode | Canonical syntax | Purpose | Constraints |
| --- | --- | --- | --- |
| User reference `user` | `[user user_id="42"][/user]` | Renders a public user summary card | `user_id` must be a positive integer; references have no body |
| Topic reference `topic` | `[topic topic_id="123"][/topic]` | Renders a public topic preview | `topic_id` positive integer; hidden/deleted/pending and other non-public targets render a not-found style without target metadata |
| Comment reference `comment` | `[comment comment_id="456"][/comment]` | Renders a public comment preview only when the comment and its owning topic are public | `comment_id` positive integer; hidden comments or non-public owning topics do not render |
| Category reference `category` | `[category category_id="7"][/category]` | Renders a public category card | `category_id` positive integer |
| Friend links `friend-links` | `[friend-links][/friend-links]` | Renders enabled operator-owned friend links in configured order | No arguments; no public output when nothing is enabled |
| Login-visible `login` | `[login]content[/login]` | Visible to the current active authenticated actor | Body required; must occupy a complete block |
| Reply-visible `reply` | `[reply]content[/reply]` | Visible to the topic author or an active actor with at least one currently public accepted comment on the topic | Body required |
| Author-only `only-author` | `[only-author]content[/only-author]` | Comments only; visible to the comment author or the owning topic author | Does not convert in topics; body required |

### Syntax rules

- A shortcode tag must occupy its own logical line (indentation up to three
  spaces); surrounding prose on the same line keeps the text literal.
- Reference nodes (`user`/`topic`/`comment`/`category`/`friend-links`) require
  an explicit matching closing tag `[/name]` and accept no body.
- Protected blocks (`login`/`reply`/`only-author`) also require a matching
  closing tag; the body may contain ordinary Markdown block content
  (paragraphs, headings, lists, quotes, code, images, other shortcodes),
  nested at most 4 levels, with at most 32 shortcode nodes per document.
- Canonical arguments use `key="value"`; import and paste also accept an
  unquoted positive integer (`user_id=42`) and normalize it to the quoted
  form; leading zeroes are removed.
- `user_id`/`topic_id`/`comment_id`/`category_id`/`tag_id` must map to the
  declared argument names. Unknown keys, `0`, negative values, non-numeric
  values, and values beyond signed int64 do **not** convert and stay literal.
- Escaped `\[user ...]`, inline code, indented/fenced code blocks, raw HTML
  blocks, and shortcode-looking text inside tables are never parsed.
- Unknown or out-of-scope legacy tags (`media`, `chart`, `alert-*`,
  `carousel`, `button`, `file`, `InvitationCode`, …) stay literal.

### Editor assistance

- The toolbar shortcode action opens a searchable picker for `user`/`topic`/
  `comment`/`category`; `friend-links` inserts directly; selected body content
  can be wrapped as `login`/`reply`/`only-author` (comment scope).
- Pasting or typing the canonical syntax converts into the corresponding
  nodes; protected blocks can change type, be deleted, or be unwrapped. No
  source mode is required or exposed.

## Legacy compatibility

### `topic-tag` is an import alias for `category`

The old SForum board reference was written as
`[topic-tag tag_id="7"][/topic-tag]`. Pasting or importing it now converts only
into the current **category** reference:

```text
[topic-tag tag_id="7"][/topic-tag]   →   [category category_id="7"][/category]
```

- It never converts to a legacy tag-reference node, and never survives
  canonical storage, export, or search text as `topic-tag`.
- New content should use `[category category_id="7"][/category]` directly.

### The legacy `friend_links` spelling does not convert

Old SForum used `[friend_links]` (underscore, sometimes without a closing
tag). The underscore spelling is **not** an alias: it stays literal and never
renders friend links. Use the canonical `[friend-links][/friend-links]`.

### `password` is not supported in this stage (deferred)

The legacy credential-gated `[password password="..."]…[/password]` content is
**outside V1 scope** and has no conversion or credential behavior at all. The
syntax stays literal text: it is not parsed, no input form appears, and no
password is validated. It requires a separate design for credentials,
rate limiting, access sessions, recovery, audit, and secret lifecycle.

## Protected content (`login` / `reply` / `only-author`)

| Scenario | What is shown |
| --- | --- |
| Anonymous visitor | "Log in to view this content" (`login`) / "Reply to view this content" (`reply`) / author-only notice (`only-author`); the body never appears |
| Authenticated but not eligible | The matching notice; the body never appears |
| Eligible (`login`: active session; `reply`: topic author or same-topic current public accepted comment; `only-author`: comment or topic author) | Server-composed protected body |
| Staff on the public page | **No implicit bypass**; the public policy applies. The protected admin/revision views are the explicit management path |
| Plugin disabled / Safe Mode / runtime unavailable, timeout, crash, invalid output | "Protected content unavailable" notice; the body stays hidden |
| Author re-editing | Canonical nodes restored through the authorized edit-source reads (below); public JSON never carries raw source |

Protected descendants are excluded from public HTML, plain text, excerpts,
search text, and SEO at write time and composed per viewer at request time —
never by CSS hiding, never by client-side decisions, and never in Nuxt page
data, raw SSR HTML, or shared caches.

### Public bodies, search, SEO, and excerpts

- Protected bodies never enter: public topic/comment APIs, SSR HTML, Nuxt
  payloads, plain text, excerpts, site/third-party search indexes,
  SEO/OG/JSON-LD, or sitemaps.
- Stored public placeholders use the **site default locale**; changing that
  default requires a bounded content re-render and search reindex.
- Reference cards contribute only stable default-locale labels to stored
  search text; resolved usernames, titles, or IDs are not indexed.
- Personalized output is never shared across actors: responses carrying
  protected composition use `private, no-store`; shared caches hold only the
  actor-independent public base.
- Topic/comment observe events and webhooks add no body or protected fields;
  protected descendants trigger no mention notifications, link previews, or
  outbound link projections.

## Editing, previewing, and re-editing

1. **Edit** — edit protected block bodies directly in the editor; pick
   reference targets through the dialog or paste the canonical syntax.
2. **Preview** — in write mode protected blocks render as bordered nodes whose
   inner content stays editable; the client serializes them only as a
   "Protected content unavailable" placeholder and never puts the protected
   body into preview HTML.
3. **Publish** — the server re-validates, normalizes, and composes output for
   the current viewer.
4. **Re-edit** — the editor restores the current canonical source through the
   authorized `/topics/{id}/edit-source` and `/comments/{id}/edit-source`
   reads. Public endpoints never return raw source; do not recover source from
   public JSON/Nuxt payloads.

## Operator notes

### Enabling, disabling, fallback

- `sforum-shortcodes` is a protected built-in shipped with the release.
  `SyncBuiltins` stages its exact artifact at API startup; enabling it is a
  normal admin confirmation under Extensions → Plugins.
- **Enabled**: the eight declarations enter the Content Registry and
  shortcodes start converting/rendering.
- **Disabled / Safe Mode / artifact missing or runtime unavailable**: the
  declarations are removed, shortcodes do not convert, and the text falls back
  to the Host-owned placeholders described above. Ordinary content is
  unaffected and protected bodies stay hidden.
- The plugin currently ships **no operator settings**: no switches, no
  parameters, no restore-defaults button. "Recommended configuration" =
  confirming the enabled exact artifact; the way to get back to
  "no conversion" is disabling the plugin or Safe Mode.

### Failure fallback matrix (public page)

| Shortcode | Runtime unavailable/disabled | Target non-public/unauthorized | Invalid output |
| --- | --- | --- | --- |
| `user` | user-reference unavailable | generic unavailable label | generic unavailable label |
| `topic` | topic-reference unavailable | not-found style, no target metadata | generic unavailable label |
| `comment` | comment-reference unavailable | not-found style, no target metadata | generic unavailable label |
| `category` | category-reference unavailable | generic unavailable label | generic unavailable label |
| `friend-links` | no public output (trace reason only) | not applicable | no output plus trace reason |
| `login` | protected unavailable; body hidden | login-required; body hidden | body hidden |
| `reply` | protected unavailable; body hidden | reply-required; body hidden | body hidden |
| `only-author` | protected unavailable; body hidden | private-comment; body hidden | body hidden |

Notice copy uses the request locale; stored public placeholders use the site
default locale.

### Search, cache, logging, and event exclusions

- Search: protected descendants and resolved reference titles/IDs never enter
  the index.
- Cache: shared caches hold only the actor-independent public base;
  personalized output is `private, no-store`.
- Logs/traces: bounded identities, versions, outcome/fallback reason codes,
  durations, and aggregate counts only — never arguments, child text,
  serialized documents, rendered output, actor identity, raw errors, or cache
  values.
- Events/webhooks: no body or protected fields; protected descendants trigger
  no mentions, link previews, or outbound link projections.

Extension authors and integrators should read
[Extension reference — Content Registry and shortcode boundaries](../../extensions/authoring-guide.md#reference-6--shortcode-declarations).
