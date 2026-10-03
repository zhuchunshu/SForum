# 2026-08-23 Built-in Shortcodes M2 Handoff

## Status

M0-M2 are complete. M3 has not started and requires explicit user confirmation.

## Changed

- Added Host-owned `sforumShortcodeRef` and `sforumShortcodeBlock` nodes with
  exact identity/version/attribute, typed argument, placement, resource,
  count, depth, and cycle-input validation.
- Added Goldmark AST-backed Markdown import and Tiptap Markdown tokenizers for
  initial content and text paste. Only admitted standalone syntax converts the
  complete Markdown document to normalized `editor-document`; invalid syntax
  remains literal.
- Added canonical shortcode export and normalized native JSON hashing so
  round-trips and revision no-op identity are deterministic.
- Added fail-closed reference/protected fallback. Protected children remain in
  authorized source but are excluded from HTML, plain text, excerpts, search,
  client fallback HTML, and mention fanout.
- Added `contracts/fixtures/shortcode-text-v1.json`, consumed by Go and
  TypeScript tests for valid, malformed, escaped, nested, code/raw HTML,
  Unicode, limits, alias, placement, unknown, and resource-scope cases.
- Completed the M1 edit-source route inventory with stable V3 identities and
  contextual edit-own/edit-any guard metadata; generated route catalogs and
  both JS/Go exact-count parity checks now cover all 345 routes.
- Kept Content Registry execution identity-only. No plugin runtime, renderer,
  Host Query, toolbar, dialog, or concrete shortcode was added.

## Verification

- `cd apps/api && go test ./...` passed.
- `cd apps/web && bun test` passed: 922 tests across 134 files.
- `cd apps/web && bun run typecheck` passed.
- `ruby scripts/validate-openapi-refs.rb` passed.
- `node tests/validate-architecture-boundaries.mjs` passed.
- `./scripts/test.sh` passed with both required compatibility-farm cells.
- `git diff --check` passed.

## Next

Wait for explicit user confirmation, then execute M3 only. Replace the current
identity-only `ForumPostFilter` with real exact-artifact Protocol V2 execution.

Primary M3 entries:

- `apps/api/app/Support/ContentRegistry/forum_post_filter.go`
- `apps/api/app/Support/ContentRegistry/execution.go`
- `apps/api/app/Support/ContentRegistry/execution_types.go`
- `apps/api/app/Support/ContentRegistry/execution_validation.go`
- `apps/api/app/Support/HostAPI/content_registry_admission.go`
- `apps/api/app/Support/Extensions/manager_identity_transport.go` (runtime
  lease/invocation pattern only)
- `apps/api/app/Models/Forum/content_pipeline.go`
- `apps/api/app/Models/Forum/content_pipeline_bridge.go`
- `apps/api/bootstrap/api_assembly_domains.go`

M3 must provide active-declaration execution, runtime leases, Host permission
rechecks, typed segments, sanitizer, budgets, quarantine/traces, bounded batch
dispatch, lifecycle replay, and deterministic fallback. M4 Host queries remain
out of scope.

## Open Questions

- None for M2. Frozen M0 contract changes still require user confirmation and
  a replacement decision record.
