# Built-in Shortcodes - Design And Task Book

Status: **active** - M0-M9 plus M10A complete; M10B in final review and blocked on exact immutable activation

Date: 2026-08-22
Last updated: 2026-08-29 - publication/editor visual parity is staged in plugin-owned v1.3.3; active artifact promotion remains blocked

Goal: ship a protected built-in `sforum-shortcodes` plugin that provides safe,
structured forum shortcodes without exposing protected content through public
source, stored HTML, excerpts, search, caches, SEO, events, or plugin failure
fallbacks.

## Milestone Status

| Milestone | Status | Evidence |
| --- | --- | --- |
| M0 | **complete** (2026-08-22) | `../decisions/2026-08-22-structured-shortcodes-public-source-secrecy.md`, `../reports/2026-08-22-builtin-shortcodes-m0-threat-inventory.md` |
| M1 | **complete** (2026-08-22) | public/edit-source DTO and API split, authorization/source-secrecy tests, `../sessions/archive/2026-08/2026-08-22-builtin-shortcodes-m1-handoff.md` |
| M2 | **complete** (2026-08-23) | Host nodes/validation, Goldmark/Tiptap import and paste, shared fixture, deterministic fallback/hash, `../sessions/archive/2026-08/2026-08-23-builtin-shortcodes-m2-handoff.md` |
| M3 | **complete** (2026-08-23) | exact active-declaration Protocol V2 dispatch, runtime leases, Host validation/sanitization/budgets/traces, bounded batch, real subprocess lifecycle matrix, `../sessions/archive/2026-08/2026-08-23-builtin-shortcodes-m3-handoff.md` |
| M4 | **complete** (2026-08-23) | seven sealed Query Registry projections, exact content-runtime delegation, PostgreSQL visibility/actor matrix and no-N+1 evidence, `../sessions/archive/2026-08/2026-08-23-builtin-shortcodes-m4-handoff.md` |
| M5 | **complete after independent review** (2026-08-23) | protected built-in `sforum-shortcodes 1.0.1`; DOM/link and boundary fixes, exact Content publication for legacy lifecycle, real-subprocess Host-chain tests, full uncached Go pass, staging/release baseline, SyncBuiltins/admin/Safe Mode/catalog/SSR evidence; `../sessions/archive/2026-08/2026-08-23-builtin-shortcodes-m5-handoff.md` |
| M6 | **complete** (2026-08-23) | production per-node dispatcher, `topic`/`comment` Protocol V2 renderers, recursion/budget/batch/cache controls, full gates, exact `1.1.0` artifact and live SSR evidence; `../sessions/archive/2026-08/2026-08-23-builtin-shortcodes-m6-handoff.md` |
| M7 | **complete** (2026-08-28) | authenticated visibility-filtered selector API, exact L2 toolbar command, five-reference dialog/NodeViews, canonical paste and alias conversion, topic/comment publish-edit-reload Browser evidence; `../sessions/2026-08-28-builtin-shortcodes-m7-handoff.md` |
| M8 | **complete (2026-08-28)** | protected-content substrate, request-only authorization/composition, cache isolation, locale-safe fallback, moderation mention replay fencing |
| M9 | **complete (2026-08-28)** | Host-authoritative login/reply/only-author policies, M4 projection wiring, exact protected Protocol V2 handlers, strict schemas, editor create/edit/delete/unwrap/preview workflow, artifact `sforum-shortcodes@1.3.0`, release baseline, failure/cache/secret-marker and browser verification |
| M10A | **complete (2026-08-28)** | legacy conversion fixture `contracts/fixtures/shortcode-legacy-conversion-v1.json` (29 text + 15 structured cases at handoff; 35 text cases after M10B conformance additions), deterministic converted/literal/invalid/unsupported/over-limit report, Go/TypeScript conformance parity tests, bilingual user/operator docs, authoring-guide Reference 6; no runtime byte changed. Evidence: `../reports/2026-08-28-builtin-shortcodes-m10a-conversion-report.md`, `../sessions/2026-08-28-builtin-shortcodes-m10a-handoff.md`; four parser conformance findings recorded for M10B |
| M10B | **active / blocked** (2026-08-29) | Go/TypeScript parser findings reconciled; focused security/lifecycle, SSR/mobile, catalog and extension checks pass. Plugin-owned publication/editor visual parity is staged in candidate `sforum-shortcodes@1.3.3`, while the active runtime remains `1.3.1`; normal super_admin trust/activation is required before release completion. Full Web and production build pass. Remaining full Go environment residuals are recorded in the M10B report. |

Execute this plan one milestone at a time. Do not start a later milestone until
the current milestone has met its exit criteria and updated durable repository
memory. Each milestone is intentionally sized so it can be handed to a fresh
agent conversation.

## Required Reading

Before every milestone, read:

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/forum.md`
4. `knowledge/modules/extensions.md`
5. `knowledge/modules/frontend.md`
6. `knowledge/decisions/2026-07-06-tiptap-editor-content-storage.md`
7. `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
8. the current hot handoff for this workstream, once one exists
9. this task book

Read `knowledge/modules/options.md` when implementing `friend-links`. Read
identity and permission notes when implementing actor-sensitive shortcodes.

## Confirmed Product Decisions

- Deliver shortcodes through one protected built-in plugin with the proposed
  identity `sforum-shortcodes`.
- Core owns the stable content, parsing, authorization-context, execution,
  public-source redaction, caching, search, and fallback contracts.
- The built-in plugin owns shortcode declarations, parameter schemas,
  renderers, localized labels, editor commands, and editor insertion UI.
- Canonical new content uses Host-accepted structured Tiptap nodes. Bracket
  syntax is an authoring, paste, API compatibility, export, and legacy-import
  format, not executable stored HTML.
- Canonical V1 syntax uses `friend-links`; `friend_links` is not an alias and
  remains literal input.
- `sourceFormat=markdown` API submissions parse admitted standalone shortcode
  blocks. A matching document normalizes to canonical `editor-document`
  storage; ordinary Markdown without admitted shortcodes remains Markdown.
- Reference shortcodes are standalone block atoms, never inline prose nodes.
- Client-rendered HTML, client authorization decisions, and client-provided
  object snapshots are never authoritative.
- `topic-tag` is a legacy alias for the current `category` shortcode. New UI,
  storage, and documentation use `category`.
- Protected shortcode content fails closed. Plugin disable, runtime failure,
  Safe Mode, timeout, invalid output, or cache failure must never reveal it.
- Reference shortcodes fail to a safe, non-interactive label or known Core URL;
  they do not disappear silently or expose a resource that is no longer public.
- A failed reference may use a known Core URL only after a current Host public-
  visibility recheck. Unknown, non-public, invalid, and unconfirmed targets use
  a non-interactive label.
- Public search, excerpts, SEO, outbound events, and webhook payloads never
  include protected shortcode bodies.
- Stored public plain/search/excerpt/SEO placeholders use the site default
  locale. Changing that locale requires content re-render and search reindex.
- `password` is not part of this task book. It requires a separate credential,
  rate-limit, access-session, recovery, audit, and secret-lifecycle design.
- Paid content, invitation codes, arbitrary BBCode, remote embeds, charts,
  buttons, file downloads, alerts, and custom user-defined shortcode handlers
  are outside V1.

## V1 Product Scope

### Reference Shortcodes

| Name | Canonical syntax | Product behavior |
| --- | --- | --- |
| `user` | `[user user_id="42"][/user]` | Render a public user summary card; username lookup may be accepted as authoring input but canonical storage uses stable user identity |
| `topic` | `[topic topic_id="123"][/topic]` | Render a public topic preview; reject hidden, deleted, pending, or otherwise non-public targets |
| `comment` | `[comment comment_id="456"][/comment]` | Render a public comment preview only when both comment and owning topic are public |
| `category` | `[category category_id="7"][/category]` | Render a public category card; legacy `[topic-tag tag_id="7"]` converts to this identity |
| `friend-links` | `[friend-links][/friend-links]` | Render enabled operator-owned friend links in configured order |

### Protected Shortcodes

| Name | Canonical syntax | Product behavior |
| --- | --- | --- |
| `login` | `[login]content[/login]` | Reveal content only to a current active authenticated actor |
| `reply` | `[reply]content[/reply]` | Reveal to the topic author or an active actor with at least one currently public accepted comment on the topic |
| `only-author` | `[only-author]content[/only-author]` | Comment-only; reveal to the comment author or owning topic author |

Public-page staff sessions do not implicitly bypass these conditions. Existing
protected admin/revision interfaces remain the explicit moderation path for raw
source inspection.

## Non-Goals

- Do not add a generic user-configurable BBCode engine.
- Do not execute package code during static install or upload.
- Do not let the plugin read raw Core tables or receive raw sessions.
- Do not make hidden content visible through client-side CSS, hydration, data
  attributes, serialized props, page payloads, or follow-up public requests.
- Do not parse shortcode-like text inside inline code, fenced code blocks, or
  escaped literals.
- Do not render one plugin RPC per shortcode occurrence without a bounded batch
  or plan; avoid N+1 behavior on comment pages.
- Do not store password values, authorization results, or rendered object cards
  in shortcode node attributes.
- Do not implement the old `topic-tag` meaning as both category and tag.

## Current Baseline And Gaps

| Area | Current baseline | Gap this plan must close |
| --- | --- | --- |
| Content Registry | Manifest V3 accepts `kind=shortcode`; immutable catalog, executor, budgets, traces, and sanitizer tests exist | Production Forum adapter is identity-only and does not dispatch exact Protocol handlers |
| Editor Registry | Exact-artifact L2 node/mark/command/toolbar catalog and digest-verified browser loader exist | No Core shortcode node, insertion UI, paste rule, or shortcode toolbar contribution exists |
| EditorDocument | Core accepts Tiptap JSON, normalizes it, renders safe HTML/Markdown/plain text, and preserves disabled-node fallbacks | Plugin catalog projection currently treats contributed nodes as atoms and does not describe nested shortcode containers or typed arguments |
| Forum storage | `posts` stores raw source plus derived HTML/plain text; revisions preserve accepted source | Public DTO currently exposes `rawContent`, which would disclose protected bodies |
| Public topic read | Cached actor-independent topic detail | Topic read does not currently load an optional viewer for content composition |
| Public comments | Optional viewer already participates in soft-delete projection | It does not compose shortcode content for that viewer |
| Search/SEO | Indexes normalized `plain_text`; SSR consumes stored sanitized HTML | Protected bodies require a separate public projection and must never enter these surfaces |
| Friend links | Core public/admin SiteChrome service and APIs already exist | Plugin needs a minimal Host query projection rather than a duplicate model |

## Architecture Contract

### Canonical Nodes

M0 freezes the exact names and schemas in
`../decisions/2026-08-22-structured-shortcodes-public-source-secrecy.md`. The
Host-owned shape is two
generic nodes so plugins do not redefine storage for every shortcode:

```json
{
  "type": "sforumShortcodeRef",
  "attrs": {
    "id": "sforum-shortcodes.user",
    "contractVersion": "sforum-shortcodes.user@1",
    "arguments": { "userId": 42 }
  }
}
```

```json
{
  "type": "sforumShortcodeBlock",
  "attrs": {
    "id": "sforum-shortcodes.login",
    "contractVersion": "sforum-shortcodes.login@1",
    "arguments": {}
  },
  "content": [
    { "type": "paragraph", "content": [{ "type": "text", "text": "content" }] }
  ]
}
```

`sforumShortcodeRef` is an atom. `sforumShortcodeBlock` contains normal admitted
block content. Core validates identity, contract version, bounded JSON
arguments, nesting, node count, and declared schema before storage. The plugin
never receives arbitrary raw HTML.

Frozen V1 limits:

- maximum 32 shortcode nodes per document;
- maximum nesting depth 4;
- maximum 16 arguments per node;
- maximum argument key length 64 bytes;
- maximum scalar string value 512 Unicode code points;
- no arrays or nested objects in V1 arguments;
- maximum render recursion depth 4;
- deterministic cycle detection by resource type and ID;
- bounded batch size and total output bytes inherited from Content Registry.

### Syntax Rules

- Names are ASCII lowercase with `-`; `friend_links` stays literal and
  `topic-tag` is the only V1 legacy alias.
- Closing tags use `[/name]`.
- Reference shortcodes accept an empty body only.
- Block shortcodes require a closing tag and preserve admitted nested rich
  content.
- Arguments use `key="value"` in exported canonical text. Import may accept an
  unquoted integer for legacy compatibility, then normalize it.
- Unknown, malformed, mismatched, over-limit, or disallowed shortcodes remain
  literal text during authoring/import; unsafe API submissions fail validation
  when they claim to be structured shortcode nodes.
- Code marks and code blocks are opaque to shortcode parsing.
- Escaped `\[name]` remains literal text and exports without being activated.
- Markdown and Tiptap conversions use one shared language-neutral conformance
  corpus so Go and TypeScript cannot drift silently.

M0 must briefly survey maintained libraries and framework-native extension
points. Prefer a Goldmark AST extension plus Tiptap Markdown/custom-extension
APIs if they cover the grammar safely. Record a dependency decision before
adding a new parser library.

### Public And Editable Content Separation

Public topic/comment responses must not expose canonical raw source. Introduce
separate contracts rather than conditionally hiding a required field inside the
same DTO:

- public content projection: sanitized/personalized HTML, public plain text,
  public excerpt, render metadata, and no raw source;
- editable source projection: canonical native JSON/Markdown plus current
  revision token, available only after authoritative edit-own/edit-any checks;
- protected revision/admin projection: unchanged permission-gated raw source.

Create dedicated topic/comment edit-source reads or an equivalently explicit
authorization boundary. Do not make public cache entries actor-sensitive merely
to keep the old DTO convenient.

### Write And Read Flow

```text
toolbar / paste / API / legacy import
  -> parse and normalize to Host shortcode nodes
  -> validate declared shortcode + argument schema + placement
  -> store canonical native source and accepted revision
  -> derive public placeholder HTML and public plain/search text
  -> cache actor-independent public base
  -> request loads optional viewer
  -> scan and batch a bounded shortcode render plan
  -> Host rechecks resource visibility and actor conditions
  -> exact-artifact plugin render through Protocol V2
  -> Host validates typed segments and sanitizes HTML
  -> return personalized HTML without raw source
```

Protected child content is available only inside the server-side accepted
document and permission-gated edit/revision paths. It is excluded before public
base HTML/plain/search/event derivation, not removed later with CSS.

### Ownership Boundary

Core owns:

- generic shortcode nodes and acceptance rules;
- bracket grammar contract and shared conformance fixtures;
- public/editable DTO split;
- actor-aware forum content composition entry points;
- Content Registry production dispatch, leases, limits, traces, sanitizer, and
  fallback enforcement;
- minimal batch Host queries and authoritative resource/actor checks;
- public/search/SEO/event projections and cache isolation;
- lifecycle-independent fail-closed protected placeholders.

The protected built-in plugin owns:

- eight canonical shortcode declarations and `topic-tag` import alias metadata;
- argument schemas and localized catalog labels/help;
- safe typed render output;
- exact-artifact backend and editor command modules;
- toolbar picker/dialog behavior;
- plugin settings only when a real operator choice exists.

Do not duplicate Forum, Identity, Profile, Category, or SiteChrome business
logic inside the plugin. Add minimal Host Query/permission projections with
allowed and denied tests instead.

### Fallback Matrix

| Kind | Disabled/runtime unavailable | Unauthorized/non-public target | Invalid declaration/output |
| --- | --- | --- | --- |
| `user` | stable `[user]` unavailable label | generic unavailable label | generic unavailable label |
| `topic` | stable topic-reference placeholder | not-found-style placeholder without target metadata | generic unavailable label |
| `comment` | stable comment-reference placeholder | not-found-style placeholder without target metadata | generic unavailable label |
| `category` | stable category-reference placeholder | generic unavailable label | generic unavailable label |
| `friend-links` | no public output; trace reason only | not applicable | no output plus trace |
| `login` | protected-content unavailable notice; body remains hidden | login-required notice | body remains hidden |
| `reply` | protected-content unavailable notice; body remains hidden | reply-required notice | body remains hidden |
| `only-author` | protected-content unavailable notice; body remains hidden | private-comment notice | body remains hidden |

Fallback text must be localized, SSR-visible, accessible, and free of hidden
content, private identifiers, stack details, and plugin runtime details.

### Cache, Search, SEO, And Events

- Cache the actor-independent public base and immutable parse/render plan, not a
  viewer's unlocked HTML.
- Personalized content responses are private/no-store unless a later proven
  actor-varying cache contract is introduced.
- Batch identical public reference resolutions per response.
- `reply` eligibility reads current authoritative comment state. Deleted,
  hidden, rejected, and pending comments do not unlock content.
- Public `plain_text`, excerpts, and search text use localized placeholders for
  protected blocks and never index their children.
- Reference cards may contribute public names/titles to request-time SSR only
  after resource visibility succeeds. V1 stored plain/search/excerpt/SEO text
  uses stable default-locale labels and does not index resolved resource names,
  titles, or identifiers.
- Topic/comment webhooks and events do not gain raw or protected body fields.
- Protected descendants do not fan out mentions, link previews, or outbound
  projections. Host moderation, sanitizer, attachment, and author trust/link
  checks still inspect the accepted child content without logging or echoing it.
- Sitemap, social metadata, and Page Registry ViewModels consume the same
  public projection as the ordinary public API.

## Milestone And Delegation Map

| Milestone | Scope | Difficulty | Recommended owner |
| --- | --- | --- | --- |
| M0 | Freeze contracts, threats, parser/library choice, and ADR | High | New Codex conversation |
| M1 | Public source secrecy and edit-source API boundary | Very high | New Codex conversation |
| M2 | Host shortcode nodes, grammar, validation, conversions, conformance corpus | Very high | New Codex conversation |
| M3 | Production Content Registry Protocol V2 dispatch | Very high | New Codex conversation |
| M4 | Minimal Host query and authorization projections | High | New Codex conversation |
| M5 | Simple built-in renderers: `user`, `category`, `friend-links` | Low to medium after M4 | OpenCode + DeepSeek V4 Flash, then review |
| M6 | Recursive references: `topic`, `comment` | High | New Codex conversation |
| M7 | Reference shortcode editor toolbar/dialog/paste UX | Medium after contracts freeze | OpenCode + DeepSeek V4 Flash, then review |
| M8 | Actor-sensitive protected-content projection and cache/search isolation | Very high | New Codex conversation |
| M9 | `login`, `reply`, `only-author` handlers and policy tests | High | New Codex conversation |
| M10 | Compatibility fixtures/docs, full lifecycle and release gate | Mixed | DeepSeek for docs/fixtures; Codex for final integration |

The recommended owner is deliberate. Do not hand M0-M4, M6, M8, M9, or the
M10 final gate to a flash model: errors there can expose protected content or
create a false production wiring claim.

## Milestones

### M0 - Contract Freeze And Threat Model

Deliver:

- a decision record for structured shortcode ownership and public-source
  secrecy;
- exact node names, attribute schemas, limits, syntax/escaping/nesting rules,
  placement rules, fallback behavior, and versioning/migration policy;
- a data-flow threat model covering public API, Nuxt payload, SSR HTML,
  hydration props, edit source, revisions, search, excerpts, caches, SEO,
  events, webhooks, logs, traces, and plugin requests;
- a maintained-library/framework-native survey with license, maintenance,
  documentation, ecosystem fit, and complexity comparison;
- an implementation inventory identifying every current producer and consumer
  of `RenderedContent.RawContent`, `HTMLContent`, and `PlainText`;
- an explicit decision on whether Markdown-source API clients gain shortcode
  parsing in V1 or only editor-document plus import/paste compatibility.

Exit criteria:

- no open decision can materially change storage, public API, parser grammar,
  authorization, or fallback semantics;
- the ADR and this task book agree;
- `knowledge/modules/forum.md` and `knowledge/modules/extensions.md` record the
  approved direction;
- a hot handoff points M1 at exact files and contracts.

Completion evidence (2026-08-22):

- Accepted ADR:
  `../decisions/2026-08-22-structured-shortcodes-public-source-secrecy.md`.
- Threat model, maintained-library survey, complete content field inventory,
  and exact M1 entry points:
  `../reports/2026-08-22-builtin-shortcodes-m0-threat-inventory.md`.
- All storage, public API, parser grammar, authorization, and fallback choices
  required by M0 are closed. M1 has not started.

### M1 - Public Source Secrecy

Deliver:

- separate public rendered-content and authorized editable-source DTOs;
- dedicated actor-authorized topic/comment edit-source reads;
- frontend edit flows migrated away from public `rawContent`;
- OpenAPI and frontend types updated together;
- public topic read accepts an optional viewer without weakening guest access;
- public topic/comment/list/Page Registry surfaces demonstrably omit raw source;
- existing admin and revision permissions remain authoritative;
- compatibility handling for existing clients is explicit and time-bounded if
  immediate field removal is impossible.

Required tests:

- anonymous and unrelated authenticated users cannot obtain raw source;
- author/edit-any users can obtain the current editable source;
- denied actors receive a non-enumerating error;
- deleted/hidden resources do not leak source;
- SSR/Nuxt page payload does not contain a seeded secret marker;
- OpenAPI refs, focused Go/Web tests, typecheck, and architecture boundaries.

Exit criteria: a protected marker placed in canonical source is absent from all
public JSON and rendered page payloads before shortcode execution is built.

Completion evidence (2026-08-22): public topic/comment PostgreSQL projections,
JSON/cache DTOs, Page Registry models, and Nuxt-serializable topic payloads are
raw-free; canonical source is available only through uncached, actor-authorized
topic/comment edit-source endpoints. Both routes have reviewed stable V3 route
identities and contextual edit-own/edit-any guard metadata. Full Go and Web
tests, Nuxt typecheck, OpenAPI refs, architecture boundaries, catalog parity,
and diff checks passed. No M2 code was started.

### M2 - Structured Nodes And Grammar

Deliver:

- Host-owned reference and block shortcode node specs;
- exact argument normalization and schema validation;
- nesting/count/depth/cycle-input budgets;
- canonical Markdown/shortcode export and import/paste grammar;
- shared Go/TypeScript conformance fixtures covering valid, malformed,
  escaped, nested, code-block, Unicode, over-limit, and legacy-alias cases;
- readable disabled-node editor fallback without revealing protected children;
- revision hash/no-op behavior based on canonical normalized source.

Required tests:

- unsupported node/attribute/contract/argument/placement denied;
- unknown textual shortcode stays literal;
- shortcode syntax inside code remains literal;
- malformed nesting cannot drop or reorder ordinary content;
- protected fallback never serializes child text into public HTML/plain text;
- Go and TypeScript fixture results match.

Exit criteria: structured documents round-trip deterministically without any
plugin process running.

Completion evidence (2026-08-23): Core EditorDocument admits the two frozen
Host nodes with strict identity/version/attribute/argument/placement/resource/
count/depth/cycle-input validation. Goldmark and Tiptap import the frozen
standalone grammar through native AST/Markdown extension surfaces, and Forum
Markdown writes switch to normalized `editor-document` only when a shortcode
is admitted. `contracts/fixtures/shortcode-text-v1.json` is consumed by both Go
and TypeScript tests. Disabled reference/protected fallbacks run without a
plugin process; protected children remain only in authorized canonical source
and are absent from HTML/plain/excerpt/search and mention fanout. Canonical
native JSON drives stable content hashes and revision no-op identity. Full Go
and Web suites, Nuxt typecheck, OpenAPI refs, architecture boundaries, and diff
checks passed; the complete repository `./scripts/test.sh` gate also passed.
M3 was not started.

### M3 - Production Content Registry Dispatch

Deliver:

- production construction of Content Registry execution bindings from exact
  active declarations;
- Protocol V2 exact handler/renderer dispatch under an acquired runtime lease;
- Host-owned permission recheck, schema validation, typed render segments,
  sanitizer, timeout, output/depth/concurrency budgets, quarantine, traces, and
  deterministic fallback;
- bounded batch execution suitable for one topic plus a comment page;
- lifecycle replay/update/remove wiring for enable, disable, upgrade, rollback,
  crash, Safe Mode, and API restart;
- no false success when declarations exist but runtime invocation is absent.

Required tests:

- real Protocol V2 integration, not injected test callbacks only;
- timeout, crash, stale lease, invalid schema, oversized output, unsafe HTML,
  disable-during-request, rollback, Safe Mode, and recovery;
- targeted race tests for registry publication and concurrent rendering;
- inspector traces identify fallback reasons without recording protected body.

Exit criteria: replace the current production identity adapter with proven
exact-artifact execution, while ordinary content remains unchanged when the
graph is empty.

Completion evidence (2026-08-23): production boot publishes the sealed
`sforum.core.content.post-body@1` target before lifecycle/Safe Mode replay and
constructs Forum bindings only from exact active `render_filter`/`sanitizer`
declarations. Protocol V2 negotiates optional `content.runtime@1`; the SDK owns
actorless typed wire DTOs while the Host revalidates the exact declaration,
artifact/version/runtime identity under an acquired lease. Forum rendering
keeps Host HTML on stale/runtime/schema/timeout/output failures and leaves Host
plain/search/excerpt/hash authoritative. The shared executor enforces bounded
batch size, concurrency, stable order, aggregate output, schema, sanitizer,
timeout, quarantine, and source-free traces. Real plugin subprocess tests cover
normal dispatch, XSS, invalid schema/identity, oversized output, timeout,
crash, disable during request, stale lease, Safe Mode, API restart/recovery,
upgrade, and rollback; focused race tests and the complete repository gate
passed. No concrete shortcode, Host Query projection, or M4 behavior was added.

### M4 - Minimal Host Query And Authorization Projections

Deliver minimal, versioned, batch-capable projections for:

- public user summaries;
- public topic summaries;
- public comments with owning-topic visibility proof;
- public categories;
- enabled ordered friend links;
- topic author/comment author identity checks;
- current actor reply eligibility.

The Host, not the plugin, decides visibility. Return presentation-safe DTOs and
opaque actor/resource decisions; do not expose emails, raw content, sessions,
IP addresses, moderation notes, private avatars, or database access.

Required tests cover public, hidden, deleted, pending, rejected, blocked,
anonymous, author, unrelated user, and privileged-admin cases. Query batches
must be bounded and must not perform one database query per shortcode.

Exit criteria: the plugin can render every V1 shortcode without raw Core DB or
raw authority access.

Completion evidence (2026-08-23): Core Query Registry now publishes seven
sealed V1 projections for public users/topics/comments/categories, ordered
enabled friend links, author decisions, and reply eligibility. ID projections
accept one canonical sorted unique batch of 1-32 positive IDs and compile to a
single PostgreSQL `ANY` predicate; friend links use the same hard result bound.
Host SQL owns public category/topic/comment status and owning-topic checks, and
actor-sensitive SQL receives only the actor bound to an opaque, exact-runtime,
one-use delegation. Actor ID and fingerprints remain in the boot-scoped Host
ledger and are absent from the JWT, plugin payload, RequestContext actor,
authority grants, traces, and result rows. Only exact extension ID
`sforum-shortcodes` receives the exact seven-query content delegation bundle;
other content plugins receive none. The SDK exposes only frozen IDs/version
helpers, `NewShortcodeProjectionIDsFilter`, and the existing
`Host.DelegatedQueryRequest` path. No raw DB capability, migration, concrete
shortcode, renderer, UI, or M5 package was added.

Focused package and race suites, full `go test ./...`, a real PostgreSQL matrix
for public/hidden/deleted/pending/rejected/blocked plus anonymous/author/
unrelated/admin decisions, architecture and release/catalog/OpenAPI gates, and
the complete `./scripts/test.sh` gate passed. The matrix also proves one SQL
statement per bounded batch rather than one query per shortcode occurrence.

### M5 - Simple Built-in Reference Renderers

Create the protected `extensions/builtin/plugins/sforum-shortcodes` package
using established built-in structure. Implement only:

- `user`;
- `category` plus `topic-tag` authoring/import alias;
- `friend-links`.

Deliver Manifest V3 content declarations, schemas, backend handlers, safe typed
render segments, localized fallbacks, plugin tests, package files, generated
digests, protected built-in publication, and catalog inspection.

Constraints:

- do not introduce new Host APIs or change frozen contracts in this milestone;
- stop and report a missing M4 contract instead of reading Core DB or calling
  loopback HTTP;
- use existing public domain DTOs and sanitization;
- render no raw plugin HTML outside typed segments;
- disabled/Safe Mode fallback follows the matrix above.

Exit criteria: source package `extension digest --write`, `extension validate`,
and `extension test` pass; SyncBuiltins stages the exact digest; normal admin
activation publishes all three declarations; API/catalog and browser SSR prove
the expected provider/digest and safe output.

Completion evidence (2026-08-23, independently reviewed): the protected
built-in package `extensions/builtin/plugins/sforum-shortcodes` (id
`sforum-shortcodes`, v1.0.1)
publishes exactly `sforum-shortcodes.user@1`, `sforum-shortcodes.category@1`,
and `sforum-shortcodes.friend-links@1` as Manifest V3 `content` declarations
with strict draft-07 schemas and Protocol V2 typed render handlers. `topic-tag`
stays an import-grammar alias only; no separate declaration or handler.
Handlers consume the frozen `sforumShortcodeRef` call value, perform exactly one
`Host.DelegatedQueryRequest` per reference call (one list query for
friend-links), and emit escaped bounded HTML over typed segments; empty friend
links produce zero public output. The review fixed malformed raw-string
`href` output and its self-confirming tests, projection/URL/Unicode/error
boundaries, a stray package-tree executable, and a process/catalog split in the
legacy lifecycle path. The latter now publishes/quarantines exact Content
Registry declarations with rollback compensation.

Backend unit tests, real-subprocess Host-chain integration tests (Manager +
Protocol V2 + HostAPI Query Registry outlet + real PostgreSQL projections +
Content Registry executor + compiled schemas + sanitizer), source
`extension digest --write`/`validate`/`test`, the staging build script,
8-plugin release baseline, architecture gate, and fresh uncached full API Go
suite all pass. Runtime evidence covers staged-before-admin promotion, normal
admin restart to immutable package
`4e5d51db4761813c1d7fd97d58a9cd3f226abda0b6bc9c24dc34c33a6f179da6`,
disable removal and enable republication of all three declarations, cold Safe
Mode Core-only catalog, normal cold-start restoration, and selected-theme
Browser DOM with raw/private source secrecy. The per-node production dispatcher
is M6, so topic SSR intentionally keeps Host fallback labels; real card
rendering is proven through the Host production chain integration.

### M6 - Topic And Comment References

Implement `topic` and `comment` using M4 projections and M3 execution.

Deliver:

- recursive resource-key stack and deterministic cycle fallback;
- render-depth and total-reference budgets;
- deleted/hidden/pending/rejected/non-public fallback without metadata leak;
- comment owning-topic visibility recheck;
- batch resolution for repeated IDs;
- cache tags/invalidation for referenced resource changes;
- SSR-visible link/card output with JavaScript-disabled usability;
- nested reference tests including self-topic, topic A -> B -> A, comment
  self-reference, deleted target, and disable during render.

Exit criteria: no recursion, N+1, visibility, stale-cache, or source-leak
finding remains in focused review and runtime evidence.

Completion evidence (2026-08-23): Forum public reads and Page Registry now use
the configured `PublicReadService`, which converts already-authorized internal
topic/comment source into per-node render plans and gives the plugin only the
strict shortcode node value. The production dispatcher follows an exact
resource-key stack, fails self-topic, topic A -> B -> A, and comment self-
references closed, and enforces depth 4, 128 total references, 32 calls per
batch, bounded output, per-call timeout, and one total execution deadline.
Canonical duplicate IDs are resolved once and fanned out; PostgreSQL source
loading separately batches topic and comment IDs. The plugin v1.1.0 adds only
`sforum-shortcodes.topic@1` and `sforum-shortcodes.comment@1`, consumes the
frozen M4 `public_topics.batch` and `public_comments.batch` projections, and
rejects comments whose owning topic is not public. Host cache keys include
exact artifact/catalog/locale/tag generations; topic/comment mutations advance
resource tags, and topic/category visibility mutations advance the shared
visibility tag.

Focused, race, PostgreSQL, real-subprocess, crash/timeout/disable/Safe Mode,
fresh full Go, full Web (922 passing), typecheck, OpenAPI, architecture,
eight-plugin release, source digest/validate/test, and `./scripts/test.sh`
gates pass. SyncBuiltins staged v1.1.0 before a normal authenticated admin
restart promoted immutable package
`209aed4e2f6d014db92ad69de2bfb46d0b385540486c8d9d4a8a1805798697b9`.
The live catalog contains exactly Core plus the five expected shortcode
declarations, all five using that same version/digest. Real topic 126 and
comment 359 SSR render safe user/category/friend-links/topic/comment output;
duplicate topic 125 references are batched, while self/cycle, hidden targets,
and a comment owned by hidden topic 137 use non-leaking fallback. Disable and
Safe Mode remove plugin declarations and restore Host fallback; a normal cold
restart restores the exact five declarations and rendered cards. Public JSON,
raw SSR HTML, Nuxt payload, DOM, plain text, and trace scans contain no source,
content hash, shortcode node, actor/session/permission, or private metadata.
The safe content and links are present in raw SSR HTML, so opening them does not
depend on hydration or JavaScript. Selected-theme desktop 1280x720 and mobile
390x844 checks retain `data-provider="sforum.default-theme"`,
`data-template="1"`, and zero horizontal overflow.

### M7 - Reference Editor Experience

Deliver exact-artifact editor commands/toolbars for the five reference
shortcodes:

- one compact shortcode menu using project icons;
- searchable user/topic/comment/category selection through existing or M4-safe
  lookup APIs;
- `friend-links` inserts without arguments;
- loading, empty, denied, error, disabled, and success states;
- paste/input conversion for canonical bracket syntax and `topic-tag` alias;
- stable NodeViews for editing and accessible fallback when the plugin is
  unavailable;
- Markdown export and re-edit round-trip;
- desktop and `390x844` mobile QA with no overflow or clipped controls.

Do not add source JSON/Markdown inspection modes, raw ID-only mystery inputs,
or a second editor toolbar system. Use existing `SFEditor`, trusted L2 loader,
toolbar catalog, icons, dialogs, Toast rules, and theme tokens.

Exit criteria: a non-expert user can insert, edit, remove, preview, publish, and
re-edit all five reference shortcodes without typing syntax.

Completed 2026-08-28. Core exposes only the authenticated, visibility-filtered
`GET /api/v1/composer/references` selector contract for ordinary composers;
M4 Query Registry projections and delegation tokens remain server-internal.
The protected `sforum-shortcodes 1.2.0` exact artifact contributes one trusted
L2 command and one existing-toolbar catalog item. `SFEditor` owns the Host
dialog bridge and editor instance, while the L2 module only requests the
admitted five-resource menu. Searchable user/topic/comment/category options,
argument-free friend-links insertion, stable accessible NodeViews, edit/
replace/delete, preview/write return, focus restoration, Escape, inline state
feedback, project Toast/i18n/icons, canonical paste, and `topic-tag` to
category conversion are covered without adding a source mode or second
toolbar. Real topic 145 and comment 368 published all five references, then
re-entered, updated, reloaded, and restored all five nodes. Exact desktop and
390x844 Browser checks found no horizontal overflow or clipped dialog controls.
The active immutable package digest is
`70d43c0e81a515c22feb6477f3582d51b7d94fc3b1593587dfd4d17bdd3d00f8`;
the exact L2 digest is
`a3fe4c2594e7a5090fd90896b33d7c9fefe9c4461e49e2e2ba3f98c3864c15e5`.

### M8 - Protected Content Foundation

Deliver the actor-sensitive read pipeline before registering protected tags:

- accepted protected block nodes produce only a public placeholder at write
  time;
- child text is excluded from public HTML/plain/excerpt/search/SEO/events/logs;
- topic and comment reads compose personalized output after actor-independent
  store/cache reads;
- personalized responses cannot enter shared server, CDN, Nuxt, or browser
  caches for another actor;
- plugin invocation receives only the minimal actor/resource decision and the
  exact accepted protected fragment needed for a successful render;
- edit source remains authorized; revision/admin inspection remains unchanged;
- moderation/publication checks still inspect full source where policy requires
  it without publishing the result;
- mentions and outbound-link policy behavior inside protected blocks is
  explicitly decided and tested.

Required leak tests seed a unique secret and search for it in anonymous and
unauthorized JSON, HTML, Nuxt payload, excerpts, search results, SEO, cache
entries, traces, logs, events, and webhooks.

Exit criteria: the protected-content substrate is proven fail-closed without
depending on any one shortcode handler.

Completion evidence (2026-08-28): Host EditorDocument writes and re-renders
protected blocks with the configured default-locale closed placeholder while
retaining accepted canonical source only for edit/revision authority. Public
topic/comment reads load actor-independent store/cache data first, then build a
request-only plan and run optional Host authorization. Authorized tests pass
only an exact accepted child fragment to a test dispatcher; no viewer,
authorization decision, fragment, or output is placed in shared cache. Private
execution suppresses Content Registry cache identity/tags and Protocol V2 Host
delegations. Direct topic/comment/reply responses carrying protected composition
set `Cache-Control: private, no-store`; Page Registry already uses the same
private policy. Moderation approval mention replay now projects editor
documents through the public side-effect view, so protected descendants cannot
notify mentioned users. Production dispatcher refresh explicitly skips protected
declarations, leaving `login`, `reply`, and `only-author` unregistered for M9.
Focused Go and race tests prove denied, authorized, locale, cache,
fragment-boundary, and concurrent-viewer behavior. Test-only protected bindings
cover allow, disable, crash, timeout, invalid output, and Safe Mode without any
production protected registration. Unique-marker tests cover API JSON, SSR DOM,
Nuxt hydration, SEO/OG/JSON-LD, Redis/process cache, PG and memory search index/
rebuild, trace/error output, mention/notification, observe/webhook projection,
and Host outbound-link/moderation policy. All topic SSR is `private, no-store`
because Nitro cannot know whether the resolved topic has protected slots before
the API read. Real PostgreSQL/Redis integration, browser evidence, full Go/Web,
typecheck, OpenAPI, architecture, compatibility farm, and `./scripts/test.sh`
pass. The current development API's separately selected optional Meilisearch
plugin still returns 500 on live search and is recorded as a runtime residual;
the Host PostgreSQL search integration and marker index tests are green.

### M9 - Protected Shortcode Policies And UI

Implement:

- `login`: active authenticated actor only;
- `reply`: topic author or active actor with a current public accepted comment;
- `only-author`: valid only in comments; comment author or topic author.

Deliver server-authoritative allowed/denied tests, toolbar dialogs, localized
notices, accessibility, re-edit behavior, and plugin failure fallbacks.

Explicit denied paths:

- anonymous, inactive/banned user, unrelated active user;
- pending/rejected/hidden/deleted reply does not unlock `reply`;
- a reply on another topic does not unlock it;
- `only-author` in a topic is rejected;
- staff on the public page receives no implicit bypass;
- plugin timeout/crash/disable/Safe Mode never reveals child content.

Exit criteria: Browser/API evidence covers every allowed and denied actor for
topic and comment surfaces at desktop and mobile widths.

### M10 - Compatibility, Lifecycle, Documentation, And Release Gate

Deliver:

- old-SForum fixtures for the eight included tags and `topic-tag` alias;
- deterministic conversion report: converted, literal/unsupported, invalid,
  and over-limit entries;
- export syntax documentation and extension-author reference;
- operator documentation for enabling/disabling, fallback behavior, search
  exclusion, and recommended defaults;
- plugin lifecycle matrix for install/enable/disable/upgrade/rollback/uninstall,
  API restart, crash, Safe Mode, and exact built-in restaging;
- migration/revision restore evidence;
- inspector and observability documentation without secret payloads;
- updated module notes, hot handoff, plan status, and index when complete.

Final verification:

1. `node tests/validate-architecture-boundaries.mjs`
2. focused Go tests for Forum, EditorDocument, ContentRegistry, HostAPI, plugin
   lifecycle, and integration packages
3. `cd apps/api && go test ./...`
4. focused Web editor/content tests
5. `cd apps/web && bun test`
6. `cd apps/web && bun run typecheck`
7. `ruby scripts/validate-openapi-refs.rb`
8. source plugin `extension digest --write`, `extension validate`, and
   `extension test`
9. `./scripts/test.sh`
10. Browser QA on topic/comment create, edit, public SSR, anonymous/login/
    reply/author denial paths, desktop and `390x844` mobile
11. runtime catalog/provider/digest proof after SyncBuiltins and normal admin
    activation
12. secret-marker scan across public API, HTML, Nuxt payload, search, SEO,
    caches, traces, logs, events, and webhooks

The program is complete only when the full gate passes or every environmental
failure is recorded with equivalent focused evidence. Source-only tests do not
replace exact immutable built-in runtime evidence.

M10A completion evidence (2026-08-28, fixtures and documentation only):

- `contracts/fixtures/shortcode-legacy-conversion-v1.json`: the frozen
  declaration/alias/unsupported tables (8 declarations, one `topic-tag`
  alias, `password` deferred), 29 text cases classified
  converted (13) / literal (4) / invalid (10) / unsupported (1) /
  over-limit (1), and 15 structured rejections tagged
  `authority: both|host`.
- Go (`shortcode_legacy_conformance_test.go`) and TypeScript
  (`editorShortcodeLegacyConformance.test.ts`) consume that same fixture and
  agree on classification, canonical export, node structure, and fallback
  codes; the frozen declarations are parity-checked against Host code.
- Report: `../reports/2026-08-28-builtin-shortcodes-m10a-conversion-report.md`
  (includes four existing Go/TS parser conformance findings that must be
  reconciled before M10B: the `2^53..int64-max` ID boundary, blank/whitespace
  protected bodies, lazy-continuation closing tags, and list canonical
  newlines).
- Docs: `docs/{zh-CN,en-US}/usage/shortcodes.md` plus usage README links and
  `docs/extensions/authoring-guide.md` Reference 6.
- No runtime, Host API, Query/Content Registry, permission, storage, or
  protected-content change was made; no commit was created.

M10B review evidence (2026-08-28):

- Reconciled the four recorded parser findings without changing the frozen ADR:
  exact positive int64 text above `2^53` remains decimal text in TypeScript,
  blank protected bodies stay literal, list/blockquote lazy-continuation
  closings stay literal, and list-to-heading canonical output matches Goldmark
  byte-for-byte. The four shapes are present in the shared conversion fixture;
  Go and TypeScript conformance suites pass.
- Focused Go Forum, EditorDocument, ContentRegistry, HostAPI, Search,
  Notifications, lifecycle, subprocess, cache, and secret-marker tests pass;
  focused Web shortcode/SSR tests pass. Architecture, OpenAPI, V3 catalog,
  source digest/validate/test, and release baseline checks pass.
- Real local API/Nuxt evidence confirms `/t/145` is `private, no-store`, public
  JSON/SSR/Nuxt payload contain no `rawContent` or protected marker, and the
  390x844 page has no horizontal overflow (`scrollWidth=379`, viewport 390).
- Blocking runtime evidence: after SyncBuiltins staging, the live content and
  editor catalogs still report active `sforum-shortcodes@1.2.0` at digest
  `70d43c0e...`; staged `1.3.0` is present but not promoted. No direct database
  promotion was used. A normal super_admin trust/activation flow with a real
  actor session is still required.
- Full `go test ./...` reached all packages but retained unrelated environment
  failures (identity request timeouts, extension integration lease conflicts,
  one ContentRegistry timing case). Full Web `bun test` retained one
  `listen(0)` EADDRINUSE failure in `tests/extensions/pluginRouteProxy.test.ts`;
  the focused rerun reproduced it. Typecheck/build and V3 catalog pass when run
  with escalated temporary-directory permissions. The final `scripts/test.sh`
  run and normal admin activation remain pending the runtime promotion step.

## Handoff Prompts

Use one prompt per fresh conversation. Replace no architectural decisions in
the prompt; the task book is the authority. Each agent must stop after its
assigned milestone.

### M0 - Codex

```text
在 /Users/inkedus/Code/SForum 执行 knowledge/plans/2026-08-22-builtin-shortcodes.md 的 M0，且只做 M0。先完整阅读 AGENTS.md、knowledge/index.md、任务书 Required Reading 和相关 hot handoff。检查真实代码与接口，不猜测。完成架构契约冻结、威胁模型、依赖/框架原生方案调研、RawContent/HTMLContent/PlainText 生产消费清单和 ADR；有会改变存储、公开 API、权限或 fallback 的歧义必须先问我。不要实现 M1 或产品代码。按任务书运行相称验证，更新 forum/extensions 模块说明、hot handoff 和任务书里程碑状态，并明确 M1 的准确入口文件与未决风险。保护现有脏工作树。
```

### M1 - Codex

```text
在 /Users/inkedus/Code/SForum 继续执行 builtin shortcodes 任务书的 M1，且只做 M1。先核验 M0 ADR、hot handoff 和当前 diff，若 M0 未满足退出条件则停止并报告。实现公共正文 DTO 与授权编辑源 DTO 的严格分离、topic/comment edit-source 权限接口、前端编辑流迁移、OpenAPI 和 public SSR/Nuxt payload 去 rawContent；保持现有 revision/admin 权限。重点测试匿名、无关登录用户、作者、edit-any、隐藏/删除资源和 secret marker 不泄漏。运行任务书 M1 要求的测试、OpenAPI 引用、typecheck 和架构门禁，更新知识库与 handoff。不要开始 M2。
```

### M2 - Codex

```text
在 /Users/inkedus/Code/SForum 执行 builtin shortcodes 任务书 M2，且只做 M2。先验证 M0/M1 完成证据。按已接受 ADR 实现 Host-owned structured shortcode ref/block nodes、严格参数/放置/嵌套限制、canonical import/export/paste grammar 与 Go/TypeScript 共用 conformance corpus。复用 Goldmark/Tiptap 原生扩展点，不自行改变冻结语法；客户端 HTML 不可信。覆盖 malformed、escaped、nested、code block、Unicode、limits、topic-tag alias、disabled protected fallback 和 deterministic round-trip。运行 focused Go/Web tests、typecheck、架构门禁，更新模块、handoff 和任务书状态。不要接生产插件执行或开始 M3。
```

### M3 - Codex

```text
在 /Users/inkedus/Code/SForum 执行 builtin shortcodes 任务书 M3，且只做 M3。目标是把当前 identity-only ForumPostFilter 替换为真实 production Content Registry Protocol V2 exact-artifact dispatch。必须使用 active declaration、runtime lease、Host permission recheck、schema validation、typed render segments、sanitizer、timeouts/budgets/quarantine/traces 和 deterministic fallback；支持有界批处理。用真实 Protocol V2 integration 覆盖 timeout/crash/stale lease/invalid schema/XSS/output limit/disable during request/rollback/Safe Mode/restart，并运行相称 race tests。空 registry 对普通正文仍是 identity。不要实现任何具体 shortcode。更新知识库、hot handoff 和任务书状态后停止。
```

### M4 - Codex

```text
在 /Users/inkedus/Code/SForum 执行 builtin shortcodes 任务书 M4，且只做 M4。先验证 M3 真实生产接线完成。设计并实现最小、版本化、批量的 Host Query/authorization projections：public user/topic/comment/category/friend-links、topic/comment author checks、reply eligibility。Host 必须拥有 visibility 判定；不得向插件暴露 raw DB、raw content、session、email、IP、moderation data 或 private avatar。覆盖 public/hidden/deleted/pending/rejected/anonymous/author/unrelated/admin 的 allowed/denied tests 和无 N+1 证据。不要创建 sforum-shortcodes 插件或开始 M5。更新契约文档、模块、handoff 和任务书状态。
```

### M5 - OpenCode + DeepSeek V4 Flash

```text
在 /Users/inkedus/Code/SForum 实现 knowledge/plans/2026-08-22-builtin-shortcodes.md 的 M5，且只做 M5。开始前完整阅读 AGENTS.md、任务书、M4 hot handoff，并确认 M0-M4 已完成。严格复用已冻结的 Content Registry、EditorDocument 和 Host Query 接口；禁止新增 Host API、读取 Core 数据库、loopback HTTP、客户端鉴权或自定义未声明接口。按现有 protected builtin 模式创建 extensions/builtin/plugins/sforum-shortcodes，只实现 user、category（含 topic-tag 输入兼容别名）和 friend-links 的 manifest declarations、schemas、Protocol V2 handlers、typed safe render segments、i18n fallback 和测试。遇到缺少接口立即停止报告，不要绕过。运行 extension digest --write、validate、test、focused Go integration、架构门禁；按正常 SyncBuiltins/管理员激活流程提供真实 catalog/provider/digest 证据。更新 hot handoff，只标记 M5，不开始 M6。保护脏工作树。
```

### M6 - Codex

```text
在 /Users/inkedus/Code/SForum 执行 builtin shortcodes 任务书 M6，且只做 M6。审查 M5 diff 和真实 runtime evidence 后，实现 topic/comment reference shortcodes。必须复用 M4 visibility projections，加入 resource-key cycle detection、render depth/total reference budgets、batch resolution、cache tags/invalidation、owning-topic visibility recheck 和 SSR usable fallback。覆盖 self-reference、A-B-A、comment self-reference、hidden/deleted/pending targets、repeated IDs、disable during render、Safe Mode 和 no-N+1。不得把 raw content 或私有 metadata 交给插件。运行 focused/full proportional tests，更新 handoff 和任务书状态，不开始 M7。
```

### M7 - OpenCode + DeepSeek V4 Flash

```text
在 /Users/inkedus/Code/SForum 实现 builtin shortcodes 任务书 M7，且只做 M7。先阅读 AGENTS.md、任务书、M6 hot handoff，并确认 M0-M6 完成。严格使用现有 SFEditor、trusted Editor Registry L2 loader、toolbar catalog、项目图标/主题 token 和 M4 lookup 接口；不要新增第二套编辑器或 source JSON/Markdown 模式。实现一个紧凑 shortcode 菜单以及 user/topic/comment/category 搜索选择、friend-links 无参数插入、编辑/删除/预览/重新编辑、canonical paste/input conversion 和 topic-tag alias。补齐 loading/empty/denied/error/disabled/success、键盘/ARIA、i18n、单测和 desktop + 390x844 Browser QA。不得改变冻结 node/grammar/Host API；缺口就停止报告。刷新 exact plugin assets/digests，运行 bun test、typecheck、extension validate/test、架构门禁，更新 handoff，只标记 M7。
```

### M8 - Codex

```text
在 /Users/inkedus/Code/SForum 执行 builtin shortcodes 任务书 M8，且只做 M8。先复核 M1-M7。实现通用 protected shortcode substrate：写入时 public placeholder，child body 从 public HTML/plain/excerpt/search/SEO/events/logs 中排除；topic/comment 按 optional viewer 在 actor-independent store/cache 之后个性化组合；个性化响应不得跨 actor 缓存。插件仅获得最小 actor/resource decision 和成功渲染所需 accepted fragment。严格按 M0 ADR 实现并测试 protected body 内 mentions/outbound links/moderation 语义，不得重新解释。用唯一 secret marker 扫描匿名/unauthorized JSON、HTML、Nuxt payload、excerpt、search、SEO、cache、trace、log、event、webhook。不要注册 login/reply/only-author。更新知识库和 handoff 后停止。
```

### M9 - Codex

```text
在 /Users/inkedus/Code/SForum 执行 builtin shortcodes 任务书 M9，且只做 M9。基于已完成 M8，实现 login、reply、only-author 的插件声明、服务端策略、typed render、toolbar dialogs、localized accessible notices 和 fail-closed fallback。严格按任务书 actor 语义：active login；reply 仅 topic author 或在同主题有当前 public accepted comment 的 active actor；only-author 仅评论且仅 comment author/topic author；staff public view 无隐式 bypass。覆盖所有 allowed/denied、pending/rejected/hidden/deleted reply、other-topic reply、topic placement rejection、timeout/crash/disable/Safe Mode secret non-leak。完成 desktop/mobile Browser 与 API 证据，更新 handoff，不开始 M10。
```

### M10A - OpenCode + DeepSeek V4 Flash

```text
在 /Users/inkedus/Code/SForum 执行 builtin shortcodes 任务书 M10 中仅“兼容 fixtures 与文档”部分，不做最终完成声明。补齐旧 SForum 八种已纳入 shortcode 和 topic-tag alias 的转换 fixtures、converted/literal-invalid/unsupported/over-limit 报告测试、用户语法文档、扩展作者文档、运营启停/fallback/search exclusion/recommended defaults 文档及中英文文案。严格复用现有实现和冻结契约，不改运行时架构、权限、存储或 API。运行文档/fixture focused tests、OpenAPI refs（如涉及）、架构门禁，更新 handoff 并列出留给 M10B 的 runtime/full-gate 项。
```

### M10B - Codex Final Gate

```text
在 /Users/inkedus/Code/SForum 执行 builtin shortcodes 任务书 M10B 最终集成与发布门禁。先审查 M0-M10A 的代码、知识库、测试和 immutable runtime 证据；不要相信里程碑标签而跳过验证。完成 install/enable/disable/upgrade/rollback/uninstall/restart/crash/Safe Mode、revision restore、SyncBuiltins、normal admin activation、catalog/provider/digest、desktop/mobile Browser 和 secret-marker 全面验证。运行任务书 Final verification 的全部门禁；修复范围内问题，记录环境限制但不得用 source-only 测试替代 runtime 证据。全部完成后更新 forum/extensions/frontend 模块、最终 hot handoff、knowledge/index.md、任务书 Status 和 knowledge/plans/README.md，并按规则归档已完成计划。
```

## Definition Of Done

- All eight V1 shortcodes work through the active protected built-in exact
  artifact in topic and applicable comment surfaces.
- Structured source round-trips through create, edit, revision, restore,
  export, paste, and supported legacy conversion.
- Public APIs and SSR never expose raw source.
- Protected bodies never appear for a denied actor through any documented or
  inspected side channel.
- Reference resources never bypass current visibility policy.
- Plugin disable, crash, timeout, upgrade, rollback, uninstall, Safe Mode, and
  API restart have tested deterministic fallbacks.
- Comment pages do not introduce unbounded RPC, query, recursion, or output
  growth.
- Editor UX is complete, accessible, localized, theme-consistent, and verified
  at desktop and mobile widths.
- OpenAPI, SDK/frontend types, extension author docs, operator docs, module
  memory, handoff, and plan status agree with runtime reality.
- The final repository gate and immutable built-in runtime evidence pass.
