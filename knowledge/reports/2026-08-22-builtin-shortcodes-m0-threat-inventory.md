# Built-in Shortcodes M0 Threat Model And Content Inventory

Date: 2026-08-22  
Scope: M0 evidence only; no runtime implementation

## Outcome

M0 freezes the V1 contract in
`../decisions/2026-08-22-structured-shortcodes-public-source-secrecy.md`.
The immediate release blocker is independent of shortcode execution: public
Forum DTOs and actor-independent Redis entries currently include canonical
`rawContent`. M1 must close that boundary before M2 admits a protected node.

The approved product choices are:

- canonical syntax is `[friend-links]`, not `[friend_links]`;
- `sourceFormat=markdown` API submissions parse admitted bracket shortcodes and
  normalize a matching document to canonical `editor-document` storage;
- all reference nodes are standalone block atoms;
- stored public plain text/search/excerpt/SEO placeholders use the site default
  locale and are rebuilt after a default-locale change;
- a failed reference may use a Core URL only after a current Host public-
  visibility recheck; otherwise it is non-interactive.

## Assets and trust boundaries

Protected assets:

1. canonical topic/comment source and accepted revision source;
2. every descendant of `login`, `reply`, and `only-author`;
3. private or non-public reference identity and metadata;
4. actor/session and authorization decisions;
5. raw plugin errors, traces, caches, and render plans that could reconstruct
   the protected assets.

Trust zones:

| Zone | Trust and allowed content |
| --- | --- |
| Browser/API client | untrusted input; no authoritative HTML, object snapshot, or authorization |
| Forum write service | authoritative validation, policy, normalization, and persistence |
| PostgreSQL/revisions | canonical source allowed; access is service/policy controlled |
| Public base/cache/search | actor-independent and permanently safe for an unrelated viewer |
| Request composition | optional viewer; may hold an authorized fragment only for the request |
| Exact built-in runtime | trusted executable, but receives only the declared minimal projection |
| Sync authoring filters | trusted write-time processors; currently receive complete `ContentInput` |
| Observe/webhook/log/trace | projection-only; never receives source or protected descendants |
| Admin/revision API | explicit protected authority; not a public-page bypass |

Threat actors include anonymous users, unrelated authenticated users, an
author trying to forge nodes or resource snapshots, a stale authorized session,
a malicious document intended to exhaust parsing/rendering, a faulty exact
plugin, and a future regression that serializes the wrong content model.

## Threat matrix

| Threat | Current exposure | Frozen control | Delivery |
| --- | --- | --- | --- |
| Public raw-source disclosure | `RenderedContent.rawContent` is required in public topic/comment JSON | separate public and edit-source DTOs; no compatibility leak | M1 |
| Raw-source hash oracle | public `contentHash` hashes source format plus canonical raw input | omit from public DTO unless redefined over public projection | M1 |
| Nuxt hydration leak | public API values enter SSR async data/useState payloads | public API has no source; secret-marker scan rendered payload | M1/M8 |
| SSR/HTML leak | stored HTML currently derives from the whole accepted document | remove protected children during public derivation, never with CSS | M2/M8 |
| Plain/search/excerpt/SEO leak | all derive from `posts.plain_text` | default-locale public placeholders; children never contribute | M2/M8 |
| Cross-actor cache leak | topic/comment cache serializes full DTO; future personalized HTML could be shared | cache public base/plan only; personalized response `private, no-store` | M1/M8 |
| Reference metadata oracle | target ID/title/link could survive a hidden/deleted result | Host visibility recheck; no target metadata when not public | M4-M6 |
| Fallback disclosure | generic fallback does not prove child removal | Host-owned closed protected fallback; generic `preserve_source` forbidden | M2/M3/M8 |
| Fallback localization outage | plugin disable/uninstall/Safe Mode could remove the copy needed to fail closed | stable reason codes and base translations are Core-owned; plugin runtime is not required | M2/M8 |
| Plugin over-disclosure | generic renderer requests can carry whole documents | declared args plus minimal Host projection; authorized child fragment only | M3/M4/M8 |
| XSS/unsafe output | trusted runtime can return attacker-shaped segments | typed segments, output schema, sanitizer, protocol allowlist, CSP | M3/M5-M9 |
| N+1/DoS | one RPC/query per occurrence, nesting, recursion, or large args | fixed node/arg/depth limits, batch plan, resource-key cycles, total budgets | M2-M6 |
| Version confusion/downgrade | stale declaration/runtime may interpret source differently | exact contract versions, leases, explicit migrations, closed stale fallback | M2/M3 |
| Event/webhook leak | observe envelope is durable/outbound | current observe payloads remain body-free; never add protected fields | all |
| Mention side channel | current create/approval parses persisted raw source | protected descendants do not fan out mentions in V1 | M8 |
| Link-policy bypass | hiding a URL inside a protected body could skip author safety policy | Host moderation/trust/link checks still inspect protected children | M8 |
| Log/trace leak | raw errors or arguments could be retained | bounded IDs/outcome/duration only; no args/source/output/actor/raw errors | M3-M10 |
| Locale drift | stored placeholder language can lag site default | default-locale change queues re-render and search reindex | M8/M10 |
| Disabled/Safe Mode leak | source-preserving fallback might render body | protected fallback remains Host-owned without plugin/runtime | M2/M8/M10 |

The most important invariant is stronger than "unauthorized HTML is hidden": a
unique secret marker inside a protected descendant must be absent from public
JSON, SSR HTML, hydration data, public plain text, excerpt, search rows/results,
SEO and structured data, shared cache values, observe events, webhooks,
notifications, plugin requests made before authorization, traces, and logs.

## Current content data flow

```text
ContentInput / sync authoring filters / revision restore
  -> Forum renderer or EditorDocument Accept
  -> RenderedContent {raw, html, plain, excerpt, metadata}
  -> optional Forum Content Registry bridge (HTML replacement only)
  -> posts + source-only revisions
  -> Postgres public/admin/revision readers
  -> actor-independent Redis topic/comment cache
  -> public controller JSON + Page Registry + Nuxt SSR/hydration

posts.plain_text
  -> excerpts/lists/profile/moderation/notification previews
  -> search adapter -> site index / selected search plugin
  -> topic SEO and structured data through excerpt
```

### Producers

| Value | Production producers |
| --- | --- |
| `RawContent` | API request decoding into `ContentInput` in `app/Models/Forum/types.go` and Forum controllers; frontend submissions in `apps/web/app/utils/forum/forumTaxonomy.ts`; synchronous `topic/comment.before_create/update` patches in `service_ops.go` and `comment_create_guard.go`; canonical normalization in `renderer.go` and `EditorDocument.Accept`; revision restore in `revisions_mutation.go` / `revisions_service.go`; developer seed inputs in `cmd/sforum/seed_run.go` and `seed_bulk.go`; static Page ViewModel body rendering in `PageViewModels/source.go` |
| `HTMLContent` | Goldmark/bluemonday and EditorDocument output in `renderer.go`; EditorDocument rendering in `app/Support/EditorDocument`; optional HTML-only replacement through `content_pipeline.go`, `content_pipeline_bridge.go`, and the current Forum Content Registry adapter; seed rendering in `cmd/sforum/seed_bulk.go`; historical preview re-render in `revisions_service.go` |
| `PlainText` | HTML-to-text or EditorDocument acceptance in `renderer.go` / `app/Support/EditorDocument/pipeline.go`; historical preview re-render in `revisions_service.go`; seed rendering in `cmd/sforum/seed_bulk.go`. Content Registry may produce plain text internally, but `content_pipeline_bridge.go` intentionally discards it, leaving Host plain text authoritative |
| `Excerpt` | `ExcerptFromPlain` in `renderer.go`, service read helpers, Forum list scanners, revision preview, moderation/profile readers, notification target preview, and search projection |

### Persistence and reconstruction

| Path | Fields and role |
| --- | --- |
| `app/Models/Forum/postgres_store_ops.go` | inserts/updates `posts.raw_content`, `html_content`, and `plain_text`; topic detail, slug detail, admin detail, and comment readers scan the triple into `RenderedContent` |
| `app/Models/Forum/revision_writes.go` | locks and compares the complete current triple for CAS/revision mutation |
| `app/Models/Forum/revisions.go` | reads the current triple and stores canonical `raw_content` in `post_revisions` |
| `app/Models/Forum/revisions_read.go` | returns permission-gated revision source and derives revision/admin excerpts from `posts.plain_text` |
| `app/Models/Forum/revisions_mutation.go`, `revisions_service.go` | rebuild accepted content from historical raw source for preview/restore |
| `app/Models/Forum/revisions_redaction.go` | irreversibly clears redacted revision raw source and related source metadata |
| migrations `202607060003...`, `202607120008...` | define the posts triple and slim source-only revisions |
| migration `202607140016_stable_core_views.sql` | exposes `html_content` and `plain_text`, but deliberately not `raw_content`, through stable Core views |
| Host Query files `Support/HostAPI/query_registry_core_schema.go`, `v2_query_postgres.go` | project stable-view HTML/plain fields to explicitly trusted query consumers; no raw field |

### RawContent consumers

All production consumers found in the repository are:

- `renderer.go` and EditorDocument acceptance: parse, normalize, hash, and
  derive the other representations;
- `service.go` and `service_ops.go`: length/content validation, new-user
  outbound-link policy, publication/moderation classification, mention scan,
  and write records;
- `Models/Moderation/settings.go` plus `Providers/forum.go`: publication
  policy evaluation over submitted source;
- `content_attachments.go`: parse canonical editor JSON and validate Host
  attachment identity;
- `postgres_store_ops.go`, `revision_writes.go`, `revisions.go`: posts and
  accepted-revision persistence, compare, and restore authority;
- `revisions_read.go`, `revisions_mutation.go`, `revisions_service.go`: admin/
  revision detail, preview, and restore;
- `Models/Notifications/fanout.go`: moderation-approval mention replay reloads
  `posts.raw_content` and reparses it;
- `cmd/sforum/seed_bulk.go` and `seed_run.go`: developer seed generation;
- public `TopicDetail` and `Comment` serialization through `types.go`, the
  Forum controller, and OpenAPI `RenderedContent`;
- frontend `forumTaxonomy.ts`, `SFTopicEditPage.vue`, `SFTopicEditor.vue`, and
  `useTopicCommentComposerDrawer.ts`: restore public response source into topic
  and comment editors;
- admin frontend `utils/admin/adminForumContent.ts`,
  `pages/admin/forum/content.vue`, and `SFAdminForumRevisionDiff.vue`: protected
  detail/revision inspection.

`PageViewModels` does not directly consume Forum `RawContent` for a public
topic page. It consumes the already-loaded topic model and maps HTML/excerpt.
The current leak still reaches browser hydration through the underlying public
topic/comment JSON and Nuxt async state.

### HTMLContent consumers

- `postgres_store_ops.go`, `revision_writes.go`, and `revisions.go` persist,
  lock, and read the current sanitized HTML.
- `content_pipeline.go` and `content_pipeline_bridge.go` pass HTML through the
  optional post-render filter; the production bridge currently changes only
  HTML.
- public topic/comment controllers serialize HTML inside `RenderedContent`.
- `PageViewModels/mapping.go` maps topic/comment HTML to
  `ThemeCompiler.SafeHTML`; `PageViewModels/source.go` supplies it to Page
  Registry topic surfaces.
- `SFTopicShowPage.vue` and `SFComment.vue` render it after the client safety
  wrapper; `plugins/highlight.client.ts` enhances code blocks without owning
  content authority.
- `SFTopicEditPage.vue`, `SFTopicEditor.vue`, admin content pages, and revision
  diff components use it for previews.
- `cmd/sforum/seed_bulk.go` persists rendered fixtures.
- `CachedStore` indirectly serializes all topic and eligible comment DTO
  fields, including HTML, into Redis.

### PlainText and excerpt consumers

- `postgres_store_ops.go` persists plain text and scans it into details;
  `list_topics.go`, `postgres_store_topic_queries.go`, and `search_live.go`
  read bounded prefixes to derive topic excerpts.
- `service.go` applies configured excerpt limits to detail/list results.
- `Models/Profile/postgres_store.go` uses bounded topic/comment plain-text
  prefixes for public activity excerpts.
- `Models/Moderation/workbench_store.go` uses bounded prefixes for protected
  review-list excerpts.
- `Models/Forum/notification_targets.go` returns public-safe target/context
  excerpts; `bootstrap/notification_adapter.go` maps them into notification
  preview DTOs.
- `bootstrap/search_adapter.go` maps topic `PlainText` and `Excerpt` into
  `Search.TopicSearchDoc`; `Support/Search/site_engine.go` persists/indexes
  both; `Support/Extensions/search_protocol.go` sends both to a selected search
  provider; `plugin_search_adapter.go` and `Support/Search/service.go` clear
  full plain text before returning public result lists.
- `PageViewModels/source.go`/`mapping.go` map topic/search excerpts to theme
  views.
- `SFTopicShowPage.vue` passes the topic excerpt to `useSForumSeo`, structured
  data, and visible topic surfaces; `forumTaxonomy.ts` uses excerpts for reply
  references.
- `CachedStore` indirectly stores topic detail/list and comment excerpts.
- stable Core forum views and Host Query expose current HTML/plain text to
  explicitly granted plugin queries; M4 shortcodes must not reuse those broad
  views and instead add minimal public projections.

## Public and protected boundary inventory

| Surface | Current state | Required boundary |
| --- | --- | --- |
| Public topic ID/slug | complete `RenderedContent`, including raw/hash | public DTO only; optional viewer composes HTML after base read |
| Public comments/replies | complete `RenderedContent` per row | public DTO only; viewer-aware composition after shared cache |
| Create/update responses | same public detail/comment shapes | public content shape; client uses dedicated read for later editing |
| Edit source | borrowed from public read | dedicated topic/comment read with mutation-equivalent policy |
| Admin/revisions | permission-gated raw source and preview | retain existing authority and separate types |
| Page Registry/theme | SafeHTML plus excerpt, sourced from public model | same public projection; never raw or protected children |
| Nuxt SSR/hydration | public request objects can include raw | API type structurally cannot include raw/hash oracle |
| Search | stored plain/excerpt sent to site/provider index | only default-locale public placeholders; no protected child/resource title |
| SEO/social/schema | topic excerpt | same public default-locale projection |
| Redis | full topic/comment DTO JSON | actor-independent public base and plan only |
| Observe/webhook | current post-commit payloads contain IDs/metadata, no body | keep body-free |
| Sync authoring filters | complete `ContentInput` at write time | remain trusted write-time extension surface in V1 |
| Notifications | source is parsed for mentions; public excerpt for preview | no protected-descendant mention; preview stays public projection |
| Plugin renderer | production Forum adapter is currently identity-only | exact declaration, minimal Host projections, authorized fragment only |
| Logs/traces | Forum logs IDs/errors; content trace omits source/output/actor/raw error | keep bounded metadata only; secret-marker scan |

## Framework and library survey

Survey date: 2026-08-22.

| Choice | License / maintenance | Fit | Cost and decision |
| --- | --- | --- | --- |
| Goldmark v1.8.2 | MIT; already pinned; active upstream and documented extension APIs | custom block parser, AST node, transformer, and node renderer can recognize standalone tags while native code nodes remain opaque | selected for Go; no new dependency |
| Tiptap 3.27.1 plus `@tiptap/markdown` | MIT; already pinned; active upstream and broad Vue/ProseMirror ecosystem | `Node.create`, block atoms/containers, Markdown tokenizer + `parseMarkdown`/`renderMarkdown`, commands, NodeViews, input and paste rules | selected for editor/import/export; no new dependency |
| Hugo shortcode engine | Apache-2.0; mature, documented, actively maintained | proven nested shortcodes and template integration | rejected: parser is coupled to Hugo page/template context and a different grammar; importing Hugo adds substantial unrelated surface |
| Standalone Go shortcode packages | no surveyed package had comparable maintenance, documentation, adoption, and grammar fit | might reduce initial parser code | rejected for V1; another parser would still need Goldmark-aware code/code-block handling and shared TS parity |
| Hand parsing around rendered HTML | no dependency | superficially simple | rejected: loses source positions/Markdown context and can activate code or escaped text incorrectly |

Repository interface evidence:

- Goldmark `parser.BlockParser`, parser options/AST nodes, and
  `renderer.NodeRenderer` are available in v1.8.2.
- Tiptap's installed types expose `MarkdownTokenizer`, `parseMarkdown`,
  `renderMarkdown`, `createAtomBlockMarkdownSpec`, and
  `createBlockMarkdownSpec`.
- `apps/web/app/utils/sfEditor.ts` already defines a custom Tiptap node with
  Markdown parse/render behavior, so M2 can follow a proven local pattern.
- `EditorDocument.NodeSpec` currently has only `Atom`, `FallbackHTML`, and an
  attribute-name allowlist. `EditorRegistry.DocumentSchema` projects every
  contributed node as an attr-less atom. M2 must extend this Host model rather
  than pretending it already supports a protected container.
- the generic Content Registry `preserve_source` fallback emits a safe label,
  but it does not implement the protected-child exclusion proof required by
  the ADR.

Sources:

- <https://github.com/yuin/goldmark>
- <https://pkg.go.dev/github.com/yuin/goldmark>
- <https://tiptap.dev/docs/editor/markdown>
- <https://tiptap.dev/docs/editor/extensions/custom-extensions>
- <https://gohugo.io/content-management/shortcodes/>
- <https://github.com/gohugoio/hugo>

## Exact M1 entry files

M1 must begin with the public/edit-source split and must not add shortcode
nodes or execution.

Primary backend and contract entry points:

- `apps/api/app/Models/Forum/types.go`: replace public use of
  `RenderedContent` with explicit public and editable-source models.
- `apps/api/app/Models/Forum/service.go` and `service_ops.go`: optional viewer
  public topic read plus mutation-equivalent edit-source authorization.
- `apps/api/app/Models/Forum/postgres_store_ops.go`: add narrowly shaped source
  reads; do not route them through public `CachedStore` entries.
- `apps/api/app/Http/Controllers/Forum/routes.go` and `controller.go`: register
  and serve topic/comment edit-source endpoints; public serializers omit raw
  and raw-derived hash.
- `contracts/openapi/schemas/forum.yaml` and `paths/forum.yaml`: split schemas,
  add edit-source reads/security/errors, and update every public response.

Primary frontend entry points:

- `apps/web/app/utils/forum/forumTaxonomy.ts`: split
  `ForumRenderedContent` from editable source and move
  `forumEditorInitialContent` to the latter.
- `apps/web/app/composables/forum/useForumApi.ts`: add edit-source calls.
- `apps/web/app/components/forum/SFTopicEditPage.vue`: stop loading source from
  `getTopic`.
- `apps/web/app/composables/forum/useTopicCommentComposerDrawer.ts`: fetch
  authorized comment source before edit.

Required consumers/tests to update or prove unchanged:

- `apps/api/app/Http/Controllers/Forum/controller_test.go`
- `apps/api/app/Models/Forum/service_test.go`
- `apps/api/app/Models/PageViewModels/source.go` and `mapping.go`
- `apps/web/app/components/forum/SFTopicShowPage.vue`
- `apps/web/tests/forum/forumTaxonomy.test.ts`
- `apps/web/tests/forum/topicEditPage.test.ts`
- `apps/web/tests/forum/defaultThemeTopicPage.test.ts`

M1 acceptance evidence must cover anonymous, unrelated login, author,
edit-any, hidden, deleted, public topic by ID/slug, comment tree/flat/replies,
create/update responses, Page Registry, and a unique secret marker in JSON and
Nuxt SSR/hydration. Existing admin/revision allowed and denied tests must stay
green.

## Residual implementation risks

- Public source secrecy is not implemented until M1; this report is not a
  mitigation by itself.
- M2 must represent parent/content/typed-argument constraints in
  EditorDocument and keep Go/TS parsers in conformance.
- M3 must replace the identity-only production Forum adapter with real exact-
  artifact Protocol V2 dispatch; executor unit tests are not production wiring
  proof.
- M4 must add narrow batch Host projections. Stable Core views expose broader
  HTML/plain fields and are not an acceptable shortcut.
- M8 must implement protected-descendant side-effect rules, locale rebuild,
  shared-cache separation, and full secret-marker scans before M9 registers a
  protected handler.
