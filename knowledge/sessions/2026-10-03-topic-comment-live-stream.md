# 2026-10-03 Session Handoff — Topic Comment Live Stream

## Changed

- **Backend (M1/M2)**: `migration 202610030002` adds `topics.comment_revision`
  plus statement-level `comments` triggers (transition tables) that bump the
  revision and `pg_notify('sforum_forum_comment_revision', topic_id)` only for
  publicly visible changes. The PL/pgSQL body is wrapped in
  `-- +goose StatementBegin/End`; without those markers goose splits the file on
  the semicolons inside `$$` and the migration fails at parse time.
- New `forum.CommentLiveService` + `CommentRevisionHub` (one LISTEN connection
  per process, topic-bucketed fan-out, 1024/256/8 connection budgets) and two
  core routes: `GET /topics/:topicID/comments/revision` (uncached facts,
  visibility identical to `ListComments`) and
  `GET /topics/:topicID/comments/stream` (SSE, 10s heartbeat, 15s reconcile,
  60s lifetime, `Last-Event-ID`/`?revision=` resume, 429 when over budget).
- **Frontend (M3)**: `utils/realtime/revisionStream.ts` is the shared revision
  runtime extracted from the notification client (which is now a thin adapter,
  so notification tests still pass). `useTopicCommentLive` reconciles against the
  revision endpoint and applies the pure decision matrix in
  `utils/forum/commentLiveDecision.ts`; `SFCommentStreamNotice.vue` renders the
  pill. `SFTopicShowPage.vue` only wires them (report-dialog state moved to
  `composables/moderation/useModerationReportDialog.ts`, which paid for the line
  budget and lowered the architecture baseline to 1154).
- **Proxy/contracts**: `server/utils/apiStreamPath.ts` marks SSE paths; the Nitro
  catch-all uses it so comment streams go through the raw pipe instead of
  `sendProxy`. OpenAPI gained both paths plus `CommentRevisionState`;
  `/api/v1/topics/:topicID/comments/stream` is a reserved Manifest V3 path.
- Catalogs regenerated (355 routes / 299 UI surfaces) and the two stale pinned
  counts in `tests/validate-v3-p0-catalogs.mjs` were corrected.

## Decisions

- `decisions/2026-10-03-topic-comment-live-stream.md` — revision signal + REST
  reconciliation, no payload push, statement-level triggers, host-reserved SSE
  path, polling fallback.

## Verification

- `go build ./...`, focused Go packages, and the trigger integration test against
  the real dev Postgres (insert/pending/approve/edit/position-update/reply-count/
  soft-delete/hard-delete, batch = exactly one bump, NOTIFY payload = topic id).
- `bun test` — 966 pass / 0 fail; `node tests/validate-architecture-boundaries.mjs`
  and `node tests/validate-v3-p0-catalogs.mjs` pass; OpenAPI refs validated with a
  Python port of `scripts/validate-openapi-refs.rb` (ruby is not installed here).
- Dev database: migration applied with `go run ./cmd/migrate`.
- Full Go suite compared against HEAD `408709ab3` on two fresh databases: the
  failing set is byte-identical (60 tests / 11 packages), i.e. this work adds no
  failure. Those 60 are pre-existing integration/environment failures.
- **Browser QA done** against the running dev API (8080) and web (3000) on topic
  403, driving real writes through the trigger:
  1. reader at the bottom of the last page → the new comment was appended in
     place with no reload (floors kept counting: `#11`, `#14`);
  2. reader scrolled up → sticky pill `有 1 条新评论` appeared at the bottom of the
     scroll viewport and the list did **not** move;
  3. clicking the pill → list caught up and the viewport moved to the newest
     comment (`#19`), pill cleared;
  4. `GET .../comments/stream` returned `text/event-stream` with
     `Cache-Control: no-store, no-transform` and a `: heartbeat` frame at 10s.
  Probe comments were deleted afterwards and `topics.comment_count` restored.
  Two real bugs were found only by this browser pass and are fixed:
  - **Reader position was measured with `window`/`document` scroll**, but the
    desktop center column (`.sforum-topic-page__main`) is its own scroll
    container, so `nearBottom` stayed `true` and every new comment was appended
    even while the reader was scrolled up. It now compares the comment stream's
    `getBoundingClientRect().bottom` against the viewport and listens for scroll
    in the capture phase; the pill is `position: sticky` so it stays visible.
  - **The SSE resume cursor and the reconciled revision were the same ref**, so
    the transport advanced it before the decision ran and every frame was judged
    `remote <= local` (no pill, no append). They are now separate
    (`streamRevision` vs `appliedRevision`).

- **Mobile QA (390x844 / Pixel-7 emulation) done** in the same pass:
  the topic page renders without horizontal overflow and comment actions collapse
  to icons; the notice pill was initially invisible and unclickable because the
  fixed 60px mobile bottom navigation (`z-index: 40`) covers the viewport bottom,
  so the pill is now offset above it (`@media (max-width: 980px) { bottom: calc(60px + 12px) }`)
  — after the fix the pill is visible above the bottom nav and clicking it
  appended and scrolled to the newest comment.
- **Multi-tab behavior observed**: with two tabs on the same topic, the visible
  tab updated promptly; the hidden tab only caught up when it became visible
  (background tabs are throttled; the visibility handler triggers the catch-up
  refresh). That is the intended UX, not a defect — but it means "实时" is
  per-visible-tab.
- Probe comments from both QA passes were deleted and `topics.comment_count`
  reconciled with the real active count.

## Next

- Two different browsers/profiles and an offline/reconnect pass are still
  unverified (no offline toggle was available in the browser tooling).
- Consider an operator option to disable the stream (default on) and re-measure
  the comment write path now that every statement also updates `topics`.

## Open Questions

- Pre-existing red tests, **not** caused by this work and reproducible at HEAD
  `408709ab3` on a fresh database:
  `database/migrator` — `TestSiteNavigationMaterializedDefaultsPreserveExplicitPlacements`,
  `TestSiteNavigationMobileCategoriesMigrationPreservesExistingPlacement`,
  `TestSiteNavigationMobileCategoriesMigrationMaterializesDefault`,
  `TestSiteNavigationCommandsAreAtomicAndRetainSnapshots` (the first failure is
  `CREATE SCHEMA sforum_core_v1 already exists` at version 202607140016, which
  cascades into `column icon_hidden does not exist`). Test isolation, not the
  live-stream change; needs a separate pass.
- The Manifest V3 identity ledger was missing entries for `SFMentionContent.vue`
  and `SFUserMentionPreview.vue` at HEAD (the catalog gate failed before this
  work); both were added here so the gate can run.
