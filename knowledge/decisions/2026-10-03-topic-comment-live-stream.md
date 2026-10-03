# Decision: Topic Comment Live Stream (Revision Signal + REST Reconciliation)

## Status

Accepted

## Date

2026-10-03

## Context

The topic page renders one continuous flat comment stream
(`view=flat`, `created_at ASC, id ASC`), so a reader only sees comments that
existed when their page was rendered. Until now the only ways to see a new
comment were a manual reload, the composer's own post-publish refresh, or
navigating to another page. The product ask is: new comments appear without a
page reload.

Facts that shaped the design (all verified in code):

- `GET /topics/{topicID}/comments` already returns `total` (public path uses the
  denormalized `topics.comment_count`), `hasMore` and a flat keyset
  `nextCursor`, so **incremental catch-up needs no new list endpoint**.
- `topics.comment_count` only increments for `status='active'` comments, and the
  moderation approval path updates `topics` directly (`Moderation.workbench_store`),
  not through the forum store. Any app-level "revision" counter would therefore
  miss paths such as approval, admin delete, and `seed:forum` bulk writes.
- The Redis generation counter `forum:gen:comments:{topicID}` uses a non-atomic
  Get→Set bump and a 24h TTL; it exists to expire list caches, not to represent a
  monotonic public revision.
- Notification V2 already ships the pattern this feature needs: a durable
  revision in PostgreSQL, one reconnecting `LISTEN` connection per API process
  whose `NOTIFY` is only a wake hint, an SSE endpoint with heartbeat + reconcile
  tick + bounded connection lifetime, and a web client with `EventSource`,
  `BroadcastChannel` coordination, Web Locks single-connection election,
  visibility suspension and a polling fallback.
- The Nitro catch-all `/api/v1/[...path]` only streams `notifications/stream`
  through a raw `node:http` pipe; every other API path goes through `sendProxy`.

## Decision

1. **Durable revision lives on `topics.comment_revision`**, maintained by
   statement-level `AFTER INSERT/UPDATE/DELETE` triggers on `comments` with
   transition tables (`migration 202610030002`; the PL/pgSQL body is wrapped in
   `-- +goose StatementBegin/End` because goose splits statements on semicolons).
   The migration is additive and keeps old binaries operational, but it is not
   declared `-- +sforum OnlineSafe`: like its sibling index migration from the
   same day, the release keeps the standard maintenance path until the whole
   pending set is reviewed for the online contract. The trigger bumps the revision
   and emits `pg_notify('sforum_forum_comment_revision', topic_id)` only for
   publicly visible changes: an `active` insert, an `active`→other or
   other→`active` transition, a `reply_count` change, or a content edit
   (`updated_at` change that is not the insert-time position update). Pending or
   rejected comments never signal. Statement-level triggering keeps bulk seed and
   bulk moderation at one topic update and one notify per statement instead of
   one per row.
2. **The signal carries no content.** `CommentRevisionState` is
   `{revision, commentCount, lastCommentId, lastCommentCreatedAt}`. Body text,
   author identity, viewer-specific tombstone projections, `replyTo` excerpts and
   extension actions stay owned by `ListComments`; pushing a second projection
   would drift from the list contract and would have to re-implement per-viewer
   visibility.
3. **`GET /topics/{topicID}/comments/revision` is the authoritative, uncached
   read.** It does one `topics` primary-key read plus one reverse tail read on
   `comments_topic_created_idx`, deliberately bypassing the 20s/45s comment list
   cache because a stale revision would keep telling clients "nothing changed".
   Visibility follows `ListComments`: hidden/deleted topics and non-public
   categories are a uniform 404; `forum.guest.read=login_required` is enforced
   through the existing `requireGuestRead` helper.
4. **`GET /topics/{topicID}/comments/stream` is the SSE wake channel**, modeled
   on `/notifications/stream`: `event: revision`, `id: <revision>`, `Last-Event-ID`
   and `?revision=` resume, `: heartbeat` every 10s, a 15s reconcile tick that
   re-reads the durable revision, a 60s connection lifetime, and
   `Cache-Control: no-store, no-transform` (which also keeps Caddy's `encode`
   from compressing the stream). Read failures and hidden-topic transitions end
   the stream; the client reconnects.
5. **`CommentRevisionHub` is per-process and budgeted**: one `LISTEN` connection,
   fan-out bucketed by topic, connection limits of 1024 total / 256 per topic / 8
   per client IP; exceeding them returns 429 `forum.comment_stream_limited` so the
   client degrades to polling instead of retry-storming. The hub is created in
   bootstrap with the API context, so shutdown releases the dedicated connection
   through context cancellation (same ownership model as `Notifications.RevisionHub`).
6. **REST reconciliation wins.** Every wake/tick re-reads the revision; the
   client only decides "append", "show the notice pill", or "nothing changed"
   after comparing against its own rendered list. LISTEN loss, notify coalescing,
   or a dead stream can delay an update but can never produce a wrong one.
7. **The SSE path is host-reserved.** Plugins may not declare
   `/api/v1/topics/:topicID/comments/stream` in a Manifest V3 route list
   (`validRoutePath`). The JSON revision endpoint stays claimable like other
   forum read routes because it has no transport coupling.
8. **Nitro must stream the new path.** `apps/web/server/routes/api/v1/[...path].ts`
   routes streaming paths through the raw pipe helper; otherwise `sendProxy`
   buffers and clients see nothing. The helper is generalized from the
   notification-specific implementation.
9. **The web client reuses the revision-stream runtime** extracted from
   `notificationRealtime.ts` (EventSource + reconnect backoff + BroadcastChannel
   + Web Locks single connection + 30s visibility-aware fallback poll), and adds
   a topic-scoped composable with a pure decision function so the
   append/notice/ignore matrix is unit-testable without a browser.

## Implementation Constraints (found by browser QA)

- **Reader position must be measured against the comment stream, not the
  window.** On desktop the topic center column (`.sforum-topic-page__main`) is its
  own scroll container, so `window`/`document` scroll metrics never change while
  the reader scrolls the thread. The composable compares
  `.sforum-topic-comments__stream`'s `getBoundingClientRect().bottom` with the
  viewport and listens for `scroll` in the capture phase; the notice pill is
  `position: sticky` so it stays visible to a reader who has scrolled up.
- **The SSE resume cursor and the reconciled revision are different values.** The
  stream runtime advances the subscription's revision as soon as a frame arrives;
  if the decision matrix reads that same ref it always concludes
  `remote <= local` and silently drops the change. `useTopicCommentLive` keeps
  `streamRevision` (transport cursor, used in the stream URL) and
  `appliedRevision` (advanced only after a reconcile decides).

## Rejected Alternatives

- **Push full comment payloads over SSE.** Duplicates the list projection
  (viewer-dependent tombstones, reply references, floor indices, extension
  actions, edit marks) and re-implements authorization; drift is guaranteed.
- **Redis generation counter as the public revision.** Non-atomic increment,
  24h TTL, no cross-process wake source, and it is a cache-expiry device.
- **Per-row triggers.** Correct but multiplies `topics` row updates and
  `pg_notify` calls by row count for `seed:forum` and bulk moderation.
- **Polling only (no SSE).** Simplest to operate, but adds constant request load
  for every open topic page and caps freshness at the poll interval.
- **WebSocket.** The platform already has a proven SSE path with the required
  proxy behaviour; WebSocket would add handshake/upgrade handling for a
  server-to-client signal that never needs client frames.
- **Auto-appending regardless of the reader's scroll position.** Moves content
  under a reader who is mid-thread; the notice pill is used whenever the reader
  is not near the bottom.

## Consequences

- New comments appear without a reload, and floors stay monotonic because the
  stream is append-only.
- Comment writes now touch `topics` once per statement (the same row the write
  path already updates), adding a per-topic serialization point for concurrent
  comment writes.
- A reader who is not on the last page never gets an automatic jump; the page
  offers a pill that navigates to the last page.
- Deletions/edits can change the stream without changing the count; the client
  refetches the visible page and only surfaces a notice when something actually
  changed in it.
- Connection budgets are operator-invisible; a busy site degrades to polling
  rather than failing.

## Follow-up

- Re-measure the comment write path (`INSERT` + position update + topic update)
  after the trigger lands, and the flat list read path with the tail read.
- Consider an operator option to disable the stream (default on) if a deployment
  wants poll-only behaviour.
- The tree view (`view=tree`) has no live stream; only the flat timeline does.
