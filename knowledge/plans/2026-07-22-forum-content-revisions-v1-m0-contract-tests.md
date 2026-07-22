# Forum Content Revisions V1 — M0 Contract Test Matrix

Status: M0 accepted test design; executable implementation starts in M1+.

This matrix names the production boundary and expected failure before schema or
runtime changes. It must stay aligned with
`knowledge/plans/2026-07-22-forum-content-revisions-v1.md` and
`knowledge/decisions/2026-07-22-forum-content-revisions-ledger.md`.

## Fixture Strategy

- Use PostgreSQL integration fixtures for migration, backfill, CAS, restore,
  redaction, attachment rebinding, and hard-delete cascade. SQLite or fake-store
  tests are not sufficient for lock ordering, `SKIP LOCKED`, arrays, triggers,
  or FK behavior.
- Create a focused fixture builder under Forum tests once M1 starts:
  `author`, `staff_editor`, `history_viewer`, `history_only`, `moderator`,
  `super_admin`, and `member`.
- Seed topics/comments across `active`, `locked`, `pending`, `rejected`,
  `hidden`, and `deleted`; deleted resources are read-only.
- Seed category/tag transitions: active category/tag, disabled tag, missing
  historical category slug, and pending/disabled tag snapshots.
- Seed attachment cases: owned active public attachment, other-user attachment,
  disabled attachment, deleted attachment, and historical ID missing from
  `attachment_references`.
- Seed plugin test doubles for synchronous filters:
  `topic.before_update` patch/reject, `comment.before_update` patch/reject, and
  observe event capture. Unauthorized/stale requests must not invoke them.
- Seed legacy `post_revisions` rows that only contain source/editor/render
  fields. Their expected `restorableFields` are content-only and
  `snapshotComplete=false`.
- Use two physical database connections for sequential and concurrent CAS tests
  so one stale request is proven under a real row lock, not just a fake store.

## Service And Store

| Test name | Boundary | Fixture | Expected failure before implementation |
| --- | --- | --- | --- |
| `TestRevisionLedgerCreateTopicWritesRevisionOne` | `Service.CreateTopic` + `PostgresStore` | active category, author | no `posts.current_revision`; no version-1 row |
| `TestRevisionLedgerCreateCommentWritesRevisionOne` | `Service.CreateComment` + `PostgresStore` | active topic, author | no version-1 row |
| `TestRevisionLedgerSelfTopicEditAppendsAcceptedSnapshot` | topic edit pipeline | author, content filter patch | old code stores previous request state before overwrite |
| `TestRevisionLedgerStaffTopicEditRequiresReason` | service validation | staff editor differs from author | no reason field exists |
| `TestRevisionLedgerSelfEditAllowsEmptyReason` | service validation | author edit | no revision metadata exists |
| `TestRevisionLedgerCommentEditAppendsOneSnapshot` | comment edit pipeline | comment author/staff | old code stores previous snapshot and no `comment.updated` |
| `TestRevisionLedgerMultiFieldTopicEditChangedFields` | store transaction | title/body/category/tags/attachments changed | no `changed_fields` |
| `TestRevisionLedgerNoopSkipsEffects` | store + cache/search/audit fakes | normalized identical snapshot | current code touches timestamps and may create revision |
| `TestRevisionLedgerSequentialConflictReturns409Reason` | HTTP/service/store | expected revision behind current | no expected revision contract |
| `TestRevisionLedgerConcurrentConflictUnderLock` | two PostgreSQL tx/connections | same expected revision racing | no locked CAS recheck |
| `TestRevisionLedgerPluginPatchIsSavedAsAcceptedState` | filter + render + store | patch title/content | old revision captures superseded source |
| `TestRevisionLedgerPluginRejectWritesNothing` | filter + store | rejecting filter | should remain green by preserving no side effects |
| `TestRevisionLedgerModerationStatusUnauthoritativeForRestore` | moderation + edit pipeline | hidden/pending/rejected resources | no restore API yet |
| `TestRevisionLedgerEditedMarkerUsesCurrentRevision` | public read models | migrated version 1 and version 2 posts | current code uses `EXISTS(post_revisions)` |
| `TestRevisionLedgerLegacyIncompleteRowsAreContentOnly` | revision detail/restore | legacy rows without metadata | no ledger model |
| `TestRevisionLedgerRestoreCreatesNewVersion` | restore service | old revision target | no restore API yet |
| `TestRevisionLedgerRestoreUnavailableCategoryFailsAtomically` | restore + category validation | missing category slug | no restore API yet |
| `TestRevisionLedgerRestoreUnavailableTagFailsAtomically` | restore + tag validation | disabled/missing tag | no restore API yet |
| `TestRevisionLedgerRestoreUnavailableAttachmentFailsAtomically` | restore + attachment validator | deleted/foreign attachment | no restore API yet |
| `TestRevisionLedgerLifecycleFieldsPreservedOnRestore` | restore transaction | hidden/pending topic/comment | no restore API yet |
| `TestRevisionLedgerHardDeleteCascadesRevisions` | database FK | hard delete post/resource | partial current FK coverage only |
| `TestRevisionLedgerSoftDeleteRetainsHistory` | lifecycle + revision read | soft deleted target | no history read API yet |
| `TestRevisionLedgerUserDeleteAnonymizesActors` | user FK behavior | deleted editor user | new actor FKs absent |
| `TestRevisionLedgerSuperAdminRedactsOldRevision` | redaction service | non-current revision | no redaction API yet |
| `TestRevisionLedgerRejectsCurrentRevisionRedaction` | redaction service | current revision | no redaction API yet |

## Migration And Backfill

| Test name | Boundary | Fixture | Expected failure before implementation |
| --- | --- | --- | --- |
| `TestRevisionLedgerMigrationIsAdditive` | Goose SQL text + migrator | current schema | migration does not exist |
| `TestRevisionLedgerBackfillNumbersLegacyRowsByStableOrder` | backfill command/job | multiple old rows with same timestamp | no backfill |
| `TestRevisionLedgerBackfillInsertsCurrentAfterLegacy` | backfill command/job | edited and unedited posts | no current snapshot |
| `TestRevisionLedgerBackfillIsIdempotent` | rerun backfill | partially completed rows | no backfill |
| `TestRevisionLedgerBackfillResumesAfterInterruption` | batched runner | forced stop after N posts | no backfill |
| `TestRevisionLedgerBackfillUsesSkipLockedBatches` | SQL text + integration | concurrent workers | no backfill SQL |
| `TestRevisionLedgerBackfillReportsProgress` | CLI/job output | mixed complete/pending/error | no progress contract |
| `TestRevisionLedgerNoRestrictiveAuditFK` | migration SQL text | evolved revision schema | no evolved schema |

## HTTP And OpenAPI

| Test name | Boundary | Fixture | Expected failure before implementation |
| --- | --- | --- | --- |
| `TestHTTPUpdateTopicRequiresExpectedRevision` | controller/OpenAPI | missing field | currently accepted |
| `TestHTTPUpdateCommentRequiresExpectedRevision` | controller/OpenAPI | missing field | currently accepted |
| `TestHTTPRevisionConflictMaps409` | controller error mapping | stale token | no `forum.revision_conflict` |
| `TestHTTPRevisionListRequiresTopicHistoryPermission` | topic revision route | member/history viewer | route absent |
| `TestHTTPRevisionListRequiresPostHistoryPermission` | comment revision route | member/history viewer | route absent |
| `TestHTTPHiddenRevisionTargetNonEnumeration` | history routes | hidden target unauthorized | route absent |
| `TestHTTPRevisionListOmitsRawSource` | revision list route | staff viewer | route absent |
| `TestHTTPRevisionDetailIncludesSourceAfterPermission` | revision detail route | history viewer | route absent |
| `TestHTTPRestoreRequiresHistoryAndEditAny` | restore route | history-only/edit-only users | route absent |
| `TestHTTPRedactRequiresSuperAdmin` | redact route | moderator/super_admin | route absent |
| `TestHTTPRevisionPaginationCapsPerPage` | revision list route | `perPage>100` | route absent |
| `TestOpenAPIForumRevisionRefsValidate` | modular OpenAPI | new path/schema refs | refs absent |

## Frontend

| Test name | Boundary | Fixture | Expected failure before implementation |
| --- | --- | --- | --- |
| `forumContentAdminNavPermissions` | admin nav config | edit-any/history users | page absent |
| `forumContentTabsHideWithoutPermission` | admin workbench | topic-only/comment-only actors | page absent |
| `forumContentEditSubmitsExpectedRevision` | editor composable/component | loaded detail token | token absent |
| `forumContentStaffReasonFieldRequired` | edit form | cross-author edit | reason field absent |
| `forumContentConflictPersistentAlert` | edit flow | 409 response | conflict UI absent |
| `forumRevisionTimelineLazyLoadsDetails` | history panel | summaries then detail | history UI absent |
| `forumRevisionDiffDesktopAndMobileWraps` | diff component | long lines/code blocks | diff UI absent |
| `forumRevisionMetadataDiffStructuredRows` | diff component | title/category/tags/attachments changed | diff UI absent |
| `forumRevisionRestoreModalRequiresReason` | restore UI | selected old revision | restore UI absent |
| `forumRevisionRedactionSuperAdminOnly` | redaction UI | super_admin/moderator | redaction UI absent |
| `forumRevisionToastsAndErrorsFollowPolicy` | UX policy | success/error states | UI absent |
| `forumRevisionI18nCompleteness` | zh-CN/en-US keys | admin labels/errors | i18n absent |

## Extension Effects

| Test name | Boundary | Fixture | Expected failure before implementation |
| --- | --- | --- | --- |
| `TestRevisionStaleRequestDoesNotInvokeTopicFilter` | service + hook fake | stale expected revision | no CAS |
| `TestRevisionUnauthorizedRequestDoesNotInvokeTopicFilter` | service + hook fake | no edit permission | should remain green |
| `TestRevisionCommentBeforeUpdatePatchAllowlist` | event catalog/filter | plugin patches content | hook absent |
| `TestRevisionCommentBeforeUpdateRejects` | service + hook fake | rejecting plugin | hook absent |
| `TestRevisionTopicUpdatedPayloadHasRevisionMetadataNoSecrets` | observe event capture | edit/restore | payload lacks revision metadata |
| `TestRevisionCommentUpdatedPayloadHasRevisionMetadataNoSecrets` | observe event capture | edit/restore | event absent |
| `TestRevisionRestoreEmitsOneCanonicalEvent` | restore + event capture | restore success | route absent |
| `TestRevisionFailuresDoNotInvalidateCacheOrSearch` | cache/search fakes | validation, stale, reject | no CAS/no-op boundary |

## M0 Validation Scope

M0 adds this matrix and the ADR only. If any executable test is added before
M1, it must assert current static contracts or be skipped with an explicit M1
activation condition; the repository must not be left with permanently failing
tests at the end of M0.
