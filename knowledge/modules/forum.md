# Forum Module

## Purpose

Owns category groups/categories, tags, topics, tree comments, shared content,
accepted revisions, lifecycle states, public read models, and forum policy.

## Current Status

- Core taxonomy, topic/comment creation and lifecycle, public/admin UI, runtime
  settings, moderation integration, search projection, and million-scale read
  path M0-M7 are implemented.
- The shared full editor uploads images through the existing attachment API
  from either the toolbar picker or drag-and-drop. A mapped ProseMirror
  placeholder preserves the original selection/drop position while uploads run
  asynchronously, even when the author continues editing elsewhere.
- Uploaded image nodes carry Host-issued attachment identity and emit explicit
  `content.attachmentIds`. Topic/comment create and update validate the node
  URL, public ID, attachment ID, active/public state, and owner-or-existing-
  resource authority before replacing references transactionally.
- Image-only editor documents are valid topic/comment bodies after Host
  normalization. Meaningful-content validation recognizes accepted image nodes
  without inventing searchable plain text; empty documents, empty paragraphs,
  and decorative-only horizontal rules remain invalid. Comment quick reply,
  advanced reply, and edit use the same native-node-aware presence check, so
  empty text serializers cannot silently discard an image-only submission.
- Native image nodes also carry bounded dimensions and a `compact`, `standard`,
  or `wide` display mode. Topic and comment surfaces cap inline images at
  separate reading widths, while each surface opens as its own PhotoSwipe
  gallery; the inline URL stays optimized and the authorized original route is
  requested only after an explicit viewer action.
- Logged-in users with reply permission can select text in topic or comment
  content and open the existing reply drawer through a compact `引用并回复`
  action. Topic selections create top-level drafts; comment selections retain
  the direct parent so Notification V2 reaches that comment author. The action
  is anchored in topic-scroll document coordinates rather than fixed to the
  viewport, and selected text enters the editor as an HTML-escaped Markdown
  blockquote capped at 500 Unicode characters.
- Topic and comment edit save actions keep validation-blocked states visually
  disabled but clickable. A save attempt now reports semantic no-op content,
  missing cross-author reasons, empty bodies, and other required-field errors;
  only active submissions remain natively disabled. Edit reasons still count
  toward the leave-page guard but cannot enable a no-op content revision.
- A custom image sticker platform is in approved design, not implementation.
  Core will own pack/item state, immutable asset revisions, a dedicated
  `sforumSticker` editor-document node, rendering/retention, admin authority,
  and a cacheable effective catalog. Forum Canvas (direction 01) is selected
  for refinement from the standalone editor/sticker demos under
  `../../tmp/demos/sforum-editor-sticker-directions-20260730/`. See
  `../plans/2026-07-30-image-sticker-platform.md` and
  `../decisions/2026-07-30-image-sticker-catalog.md`.
- Topic and comment create cooldowns remain independently configurable. A
  cooldown rejection now returns HTTP `429` with standard `Retry-After` plus
  `retryAfterSeconds` / `retryAt`; topic creation and both comment composers
  show a server-authoritative countdown while preserving editable drafts.
- Top-level quick replies use the compact inline editor and submit without
  opening an overlay. Advanced topic reply, comment reply, and comment edit
  share one responsive `USlideover` composer that opens from the bottom on
  desktop and mobile and exposes a pointer/keyboard height handle. The former
  standalone advanced-reply route is compatibility-only and redirects into
  this drawer without changing create/update authorization, revision CAS,
  moderation, or cross-author audit-reason rules. Inside the drawer, the full
  editor uses a stable toolbar/canvas/status grid: extra drawer height expands
  only the canvas, while compact heights scroll the drawer body instead of
  clipping the editor status row.
- Topic detail exposes public **contributors** (author + body edit/restore
  actors, max 5 + count) and `GET /topics/{id}/contribution-timeline` for a
  header-only publish/edit timeline. Staff actors are fully exposed by default;
  full revision source remains `topic.revision.view_any` only.
- Topic detail and comment rows expose optional `editedAt` from the current
  accepted `post_revisions` entry when the corresponding edit-mark setting is
  enabled. The public detail UI shows relative publish/edit times through one
  month and forces older values to site-timezone `Y-m-d H:i:s`; resource
  `updatedAt` remains lifecycle/counter metadata and is not used as edit time.
- Comment avatars and author names now open a compact public-profile preview
  before navigation. The card reads the existing public profile contract,
  exposes only real topic/comment counts and profile data, and keeps the
  canonical `/u/:username` navigation inside the card.
- Soft-deleted comments now render a localized, bordered deletion notice in the
  shared public comment component instead of an empty content block.
- Comment actions keep the full authorized set on desktop. At `640px` and
  below, reply and permalink stay inline while edit, delete, report, and
  extension actions move into an accessible menu beside the public floor
  number. This is presentation-only; existing API authorization and action
  dispatch remain authoritative.
- Default-theme topic creation now has a production UI shell that reuses the
  existing create-topic API, category/tag policy, content limits, permission
  check (`topic.create`), `SFEditor`, field errors, Toast feedback, successful
  redirect, and unsaved-content guard. The UI adds live publish summary and
  pre-publish checks only; it does not change API semantics.
- Content revisions V1 is **complete**: authenticated topic/comment edit, lazy
  history/detail, diff/preview, restore, super-admin redaction, stale-CAS,
  mobile diff, allowed/denied API checks, development backfill, query evidence,
  audit-retention proof, and operator documentation are closed. The unrelated
  concurrent `ExtensionManifest` gate failure is tracked outside this module.
- PostgreSQL site search is the protected default. Meilisearch is optional and
  must not be described as the required/default forum read path.

Active revision sources:

- Plan: `../plans/archive/2026-07/2026-07-22-forum-content-revisions-v1.md`
- M0 contract matrix:
  `../plans/archive/2026-07/2026-07-22-forum-content-revisions-v1-m0-contract-tests.md`
- Decision: `../decisions/2026-07-22-forum-content-revisions-ledger.md`
- Final handoff: `../sessions/archive/2026-07/2026-07-23-forum-content-revisions-v1-m7-handoff.md`

## Domain Model

- Category groups contain ordered categories. Current category visibility is
  `public` or `hidden`; role/category ACL is deferred.
- Topics own title, slug, category, author, state, counters, activity, tags,
  and a shared `content_id`.
- Comments form arbitrary-depth trees through parent/root/path metadata and own
  a shared `content_id`.
- `posts` stores raw content, sanitized HTML, plain text, format/editor/render
  versions, hash, and current accepted revision.
- `post_revisions` is an immutable accepted-version ledger. Derived HTML,
  excerpt, and plain text are recomputed rather than duplicated as historical
  authority.
- `topic_revision_snapshots` freezes topic metadata needed to reconstruct an
  accepted revision without turning the hot topic row into history storage.
- Topic states are `active`, `locked`, `hidden`, and `deleted`; tags are
  `active`, `pending`, or `disabled`.

## Content Revisions V1

Migration `202607220052` added `posts.current_revision`, numbered nullable
ledger metadata, concurrent `(post_id, revision_no)` indexes, and topic
revision snapshots. New topic/comment creation writes accepted revision 1 in
the same transaction.

Mixed legacy reads expose effective `currentRevision >= 1`; edited state is
effective revision `> 1`, so the initial accepted row does not mark all content
as edited. Backfill command:

```text
sforum revisions backfill --batch=N [--loop]
```

It claims `current_revision=0` batches with `FOR UPDATE SKIP LOCKED`, preserves
legacy row order, inserts the current snapshot, and is idempotent.

M2 added `topic.revision.view_any` and `post.revision.view_any`, permission
migration `202607220053`, moderator template/frontend labels, and read-only
history/admin content surfaces. Topic/comment revision list/detail are
service-gated by the matching history permission: lists return summary headers
only, while detail returns raw source only after authorization and renders a
safe historical preview on demand. Admin topic/comment content list/detail
require `admin.access` plus matching edit-any or history-view permission and do
not add admin mutation routes.

M3 makes PATCH versioned: `expectedRevision` is required in the public API,
the service rejects already-stale requests before synchronous filters, and the
PostgreSQL write transaction locks the resource/post and repeats CAS before any
write. Effective edits append exactly one accepted final snapshot and increment
`posts.current_revision`; semantic no-ops leave timestamps, ledger, audit,
cache, and search untouched. Cross-author edits require a trimmed reason of at
most 500 runes and append generic audit in the same transaction. The public
editors submit their loaded token. `comment.before_update` / `comment.updated`
now complement enriched safe `topic.updated` revision metadata.

V1 boundaries:

- M2 complete: view permissions, revision list/detail read models, admin
  content list/detail, tests, and OpenAPI.
- M3 complete: mandatory `expectedRevision`, two-stage CAS, final accepted edit
  snapshots, no-op detection, reason/audit rules, and comment update hooks/events.
- M4 complete: canonical topic/comment restore append new `restore` revisions,
  re-run filters/rendering/moderation/cache/search, validate current taxonomy and
  historical attachment ownership/availability, and redaction tombstones with
  transaction-bound audit. Legacy incomplete snapshots restore content only.
- M5 complete: admin management UI loads protected content through read models
  and sends canonical PATCH edits with the loaded revision token.
- M6 implementation complete: authorized staff get a summary-only timeline,
  lazy raw-detail reads, sanitized historical preview, source/metadata diff,
  restore confirmations, and super-admin-only irreversible redaction. The
  reviewed `diff` 9.0.0 BSD-3-Clause dependency is direct. Public history and
  force overwrite remain out of scope. The M6 authenticated browser and API
  release matrix is complete; preserve these boundaries for M7.
- M7 rollout evidence: the local development database backfill reached zero
  pending posts; revision-list explain uses its dedicated index; audit cleanup
  is regression-tested not to affect ledger rows; admin list indexes are staged
  in migration `202607231000`. Production keeps the compatibility read until
  every database has a recorded zero-pending backfill proof. See
  `../reports/2026-07-23-forum-content-revisions-v1-m7.md`.
- Use npm package `diff` 9.0.0 (BSD-3-Clause) for the diff UI; do not install
  npm `jsdiff`.
- Collaboration, CRDT, drafts, notifications, retention controls, and public
  revision browsing remain outside V1.
- Revision query/mutation, raw history content, reason text, IPs, and attachment
  provider data remain closed to plugins in V1.

## Content And Rendering Rules

- Backend accepts Markdown, HTML, and `editor-document` source formats. The
  structured format stores Host-accepted native Tiptap JSON in `raw_content`;
  the ambiguous legacy `json` format is rejected by both runtime and schema.
- Native `orderedList.start` values are normalized as integers before storage
  and rendering. Default `1` is omitted; zero and negative integers are
  preserved in both normalized JSON and sanitized `<ol start>` HTML, while
  fractional and string values fall back to the default. The sanitizer allows
  only a signed decimal integer on `ol[start]`.
- Markdown renders with `goldmark` plus GFM extensions; display HTML is
  sanitized with `bluemonday`.
- Sanitized `<pre><code>` content is progressively enhanced in the browser with
  a language label, line numbers, syntax highlighting, and copy feedback. A
  declared supported language selects its grammar; an unknown declared
  language keeps its honest label without guessed tokens, and an undeclared
  block is always localized plain text rather than auto-detected.
- `RenderVersion` is `goldmark-bluemonday-v2`; existing rows keep earlier HTML
  until a later edit or explicit migration.
- Content/excerpt limits are Unicode-rune based. Excerpts derive at read time
  from `plain_text` using `forum.reading.excerpt_rune_limit`.
- Topic/comment writes may provide `content.attachmentIds`. Explicit arrays
  replace references transactionally, omission preserves them on edit, and an
  empty array clears them. Native editor image nodes must name an attachment in
  that same array and use its matching `/media/attachments/{publicId}` or
  compatible historical API URL; forged or partial identity fails closed.
- Hidden/deleted/moderation-only content stays out of public SSR, sitemap, and
  search indexes.
- Planned custom image stickers are distinct from Unicode `sforumEmoji` and
  generic images. Accepted content will snapshot a stable sticker ID plus exact
  asset digest, while Core rendering will cap display at `128x128` CSS pixels
  on desktop/tablet and `96x96` on mobile. This contract is approved but not
  yet implemented.

## Public URLs And Reads

| Resource | URL |
| --- | --- |
| Categories | `/categories`, `/c/:categorySlug` |
| Tags | `/tags`, `/tags/:tagSlug` |
| Topics | `/t/<path>` |

The public `/categories` index is a grouped directory, not a fabricated
dashboard. It keeps empty public groups visible, filters out hidden groups and
categories defensively, sorts categories only inside their owning group, and
derives totals/distribution/activity from the category DTO counters.

The `/tags` index is a public read surface when
`forum.tags.public_pages=enabled`. Its heat overview, directory filters, and
right-rail stats are derived only from active tag `topicCount`, `createdAt`,
name, slug, description, and status fields returned by the public tag API; there
are no fabricated likes, follows, trends, or weekly activity counters.

`seo.topic_url_mode` controls topic paths:

| Mode | Shape | Lookup |
| --- | --- | --- |
| `id_slug` | `/t/123/hello-world` | topic ID |
| `id` | `/t/123` | topic ID |
| `slug` | `/t/hello-world` | globally unique slug |

The catch-all topic route recognizes old shapes and redirects to the canonical
path. Only a 404 advances to the next lookup candidate; network/API failures
are not swallowed.

Comment pagination uses a path segment `/page/N` (N>1 appended, N=1 omitted) so
`forumTopicPath(topic, mode, page)` and `parseTopicPath` share one parser. Old
`?page=N` query links still resolve and are normalized to the path segment on
the client after hydration (never an SSR redirect, to preserve zero-flash
rendering of the resolved page).

Cross-page `#comment-{id}` anchors resolve with zero flash: when the URL has a
target anchor and no explicit page, `SFTopicShowPage` resolves the page
server-side via the comment-page endpoint, then SSR-renders that page so the
target comment is present in first-paint HTML and the browser scrolls natively.
Slug/mode mismatches still 301/replace, but page-segment normalization is
client-only.

Public reads expose only active/locked topics. Locked topics remain readable
but reject new comments. Viewer-aware deleted-comment tombstones never expose
body fields or deleted-parent reply excerpts.

## Pagination, Cache, And Scale

The completed task book is archived at
`../plans/archive/2026-07/2026-07-21-million-scale-read-path.md`; durable results
live in `../reports/` and
`../decisions/2026-07-21-read-replica-and-api-horizontal-scale.md`.

- Public topic and flat-comment lists accept shallow page pagination and opaque
  keyset `after`; cursor wins over page. Responses return `hasMore` and
  `nextCursor` where applicable.
- Topic keysets include pin state as the first ordering dimension.
- Tree comments page root comments and cap descendants per root using
  `forum.comments.tree_descendants_per_root` (default 50). Truncated roots set
  `hasMoreChildren` and load more through the replies endpoint.
- `CachedStore` caches taxonomy, topic detail, topic lists, and eligible public
  comment lists through Redis-backed generation keys.
- Topic-list invalidation uses global/category/tag scoped generations; writes
  bump only affected scopes plus global.
- Topic detail caches support ID and slug lookup with reverse-map invalidation.
  There is deliberately no composite topic-page cache.
- Nuxt `/t/**` disables whole-page caching because SSR embeds live comment and
  permission-aware payloads. Anonymous responses use `public, no-cache` for
  mandatory revalidation; session/edit responses use `private, no-store`.
  Topic/detail/comment scale remains owned by the API's topic-scoped Redis
  caches and generation invalidation.
- Horizontal default remains one PostgreSQL primary plus shared Redis. Read
  replicas are deferred to measured thresholds; no `DATABASE_READ_URL` runtime
  path exists yet.
- Comment page resolve (`GET /topics/:topicID/comments/:commentID/page`) returns
  the flat-view page holding a comment, for cross-page `#comment-{id}` anchor
  deep-linking. The service reuses `GetCommentSummary` and counts active
  comments ordered before `(path_key, id)` (strict alignment with
  `listCommentsFlat`). Only active comments resolve; soft-deleted, cross-topic,
  or hidden-topic targets return 404 without leaking status. The endpoint is
  not cached: it is a primary-key lookup plus an indexed COUNT, and caching
  would introduce drift when comments are deleted.

## Policy And Runtime Settings

Forum options cover default category, tag creation/public pages, tag limits,
pagination, content limits, cooldown/daily caps, edit windows, comment depth,
tree descendant cap, excerpt length, guest read mode, list sort/hot window,
author close/delete rules, edit marks, duplicate-title policy, soft-delete
visibility, and mention limits.

- Recommended defaults are configurable and resettable in the multi-tab forum
  settings UI.
- `forum.guest.read=login_required` makes public taxonomy/topic/search/comment
  endpoints return 401 to anonymous users.
- New-user trust options may tighten cooldowns/daily caps and outbound-link
  policy during the configured trust window.
- Tag modes are `controlled`, `review`, and `open`.
- Public topic/comment page sizes default to 20, accept 1-100, and remain
  server-authoritative when callers omit `perPage`.

## Notification Fanout

- Active top-level comments notify the topic author; nested comments notify
  only the direct parent author. Self and inactive recipients are skipped.
- Topic and comment creates parse mentions from the stored, filtered source via
  goldmark. Inline/fenced code is ignored; case variants and duplicates collapse
  per recipient while reply and mention remain distinct intents.
- Pending topic/comment approval loads stored source and target context inside
  the decision transaction, then writes moderation plus eligible reply/mention
  projections exactly once. Rejection writes only the author's moderation result.
- Notification/outbox failure rolls back the owning content or moderation
  transaction. Topic/comment edits do not emit new mention notices in V2.
- Notification target resolution reuses public Forum visibility: topic must be
  active/locked in a public category; comment must also be active.

## Authorization

- Topic create: login plus `topic.create`.
- Comment create: login plus `post.create`.
- Own/any topic edit: `topic.edit_own` / `topic.edit_any`.
- Own/any topic delete: `topic.delete_own` / `topic.delete_any`.
- Topic lock/unlock: `topic.lock`; pin/unpin: `topic.pin`.
- Own/any comment edit: `post.edit_own` / `post.edit_any`.
- Own/any comment delete: `post.delete_own` / `post.delete_any`.
- Category administration: `category.manage`; tag administration:
  `tag.manage`.
- Forum settings writes are restricted to the permission owning the changed
  setting family.
- API policy checks are authoritative. Frontend visibility is only UX.

## Extension Boundary

- Core owns canonical forum tables, policy, revision authority, core handlers,
  and default read models.
- Plugins use declared hooks/events, contributions, providers, queries,
  navigation/regions, and other versioned registries.
- Trusted route/guard/query replacements are possible only through V3 exact-
  artifact grants and own the declared authorization contract. Ordinary
  integrations must not request raw core database access or whole-route power.
- Search indexing, notification fanout, analytics, and provider integrations
  project from stable Core contracts.
- Regenerate the Forum Extension Surface Matrix when revision events/catalogs
  change.
- Planned plugin sticker packs use a generated exact catalog and Core import;
  no executable registration or remote media URL is required. Disable,
  upgrade, rollback, uninstall, and Safe Mode must leave accepted historical
  sticker content renderable.

## Important Paths

| Path | Responsibility |
| --- | --- |
| `apps/api/app/Models/Forum` | Domain services, stores, policy, cache decorator |
| `apps/api/app/Http/Controllers/Forum` | Public/admin HTTP handlers |
| `apps/api/database/queries/forum.sql` | sqlc forum queries |
| `contracts/openapi/paths/forum.yaml` | Forum HTTP contract |
| `contracts/openapi/schemas/forum.yaml` | Forum schemas |
| `apps/web/app/components/forum` | Reusable forum UI |
| `apps/web/app/pages/t/[...path].vue` | Topic route host |

## Next Steps

1. Keep comment composer behavior consolidated in
   `useTopicCommentComposerDrawer`; do not reintroduce page-local edit/reply
   state or a standalone advanced-reply editor.
2. Complete the new editor product design before starting the custom image
   sticker platform implementation.
3. Preserve the accepted revision ledger, CAS, authorization, and redaction
   boundaries when sticker references enter topic/comment content.
4. Keep OpenAPI, allowed/denied policy tests, module status, and Extension
   Surface Matrix synchronized at each milestone.
5. Treat role-scoped category ACL and reactions/bookmarks as separate future
   tracks rather than sticker scope.
