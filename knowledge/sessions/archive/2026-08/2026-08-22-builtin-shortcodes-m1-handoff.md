# 2026-08-22 Built-in Shortcodes M1 Handoff

## Status

M0-M1 are complete. M2 has not started and requires explicit user confirmation.

## Changed

- Split Forum content into `PublicRenderedContent` and protected
  `RenderedContent`/`EditableContentSource`. Public topic/comment SQL, DTOs,
  Redis entries, HTTP JSON, Page Registry models, and Nuxt-serializable payloads
  omit canonical source, editor metadata, attachment IDs, and source hashes.
- Added uncached `GET /topics/:topicID/edit-source` and
  `GET /comments/:commentID/edit-source`. `EditableSourceService` authorizes
  active edit-own/edit-any actors, edit windows, resource state, and a comment's
  owning topic before the store reads source.
- Public topic ID/slug reads now accept an optional actor through
  `PublicReadService`; M1 still returns the actor-independent public base. Both
  Forum HTTP and Page Registry pass the authoritative optional actor.
- Migrated topic and comment edit flows to the dedicated source endpoints and
  use the returned `currentRevision` for update CAS.
- Kept admin/revision projections and their existing authorization on complete
  `RenderedContent`. Immediate public field removal is the accepted breaking
  security correction; there is no compatibility endpoint that leaks source.
- Kept the legacy Forum `Service` receiver cap at 77 and lowered the
  `postgres_store_ops.go` large-file baseline from 1604 to 1585 after moving
  source SQL into its focused owner.

## Contracts

- Public content fields: `id`, `htmlContent`, `plainText`, `excerpt`,
  `renderVersion`.
- Editable source fields: `rawContent`, `sourceFormat`, `editorType`,
  `editorVersion`, `contentHash`, `attachmentIds`, `currentRevision`.
- Anonymous edit-source reads return 401. Unrelated actors and hidden/deleted
  topic/comment resources return non-enumerating 404. Author edit-window expiry
  remains the existing conflict response. Edit-any actors are not author-window
  limited.
- Public cache entries remain actor-independent and source-free. Edit-source
  store calls bypass the public cache decorator.

## Verification

- `cd apps/api && go test ./...` passed.
- `cd apps/web && bun test` passed: 904 tests across 133 files.
- `cd apps/web && bun run typecheck` passed.
- `ruby scripts/validate-openapi-refs.rb` passed: 2,669 refs across 54 files.
- `node tests/validate-architecture-boundaries.mjs` passed: 1,616 production
  files scanned; 172 remain above the 500-line review threshold.
- `git diff --check` passed.

## Next

Wait for explicit user confirmation, then execute M2 only.

Primary M2 backend entries:

- `apps/api/app/Support/EditorDocument/types.go`
- `apps/api/app/Support/EditorDocument/schema.go`
- `apps/api/app/Support/EditorDocument/pipeline.go`
- `apps/api/app/Support/EditorDocument/render.go`
- `apps/api/app/Support/EditorDocument/pipeline_test.go`
- `apps/api/app/Support/Markdown/renderer.go`
- `apps/api/app/Models/Forum/renderer.go`

Primary M2 frontend entries:

- `apps/web/app/utils/sfEditor.ts`
- `apps/web/app/components/SFEditor.vue`
- `apps/web/tests/framework/editorCanvas.test.ts`

M2 must add one shared Go/TypeScript conformance corpus for the frozen valid,
malformed, escaped, nested, code-block, Unicode, over-limit, legacy-alias, and
deterministic round-trip cases. Use Goldmark and Tiptap native extension points;
do not reinterpret storage, public API, permission, or fallback contracts.

## Open Questions

- None. Any newly discovered ambiguity that changes a frozen M0 contract must
  stop for user confirmation rather than being resolved during M2.
