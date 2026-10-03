# 2026-08-28 Built-in Shortcodes M10A Handoff

## Changed

- Added `contracts/fixtures/shortcode-legacy-conversion-v1.json`: frozen
  declaration table (8 identities), the sole `topic-tag` → `category` alias
  metadata, `password` marked deferred/unsupported, 29 text cases classified
  `converted`/`literal`/`invalid`/`unsupported`/`over-limit`, and 15
  structured (native JSON) rejection cases tagged `authority: both|host`.
- Added Go conformance tests
  (`apps/api/app/Support/EditorDocument/shortcode_legacy_conformance_test.go`):
  declaration/alias/unsupported parity with Host code, text classification,
  structured Host rejection, corpus completeness, and fallback
  code/label parity.
- Added TypeScript conformance tests
  (`apps/web/tests/framework/editorShortcodeLegacyConformance.test.ts`):
  same fixture, classification equality, canonical output, client-visible
  fallback codes, and client inspection only for `authority: both`.
- Wrote user/operator docs `docs/{zh-CN,en-US}/usage/shortcodes.md` (+ usage
  README rows) and `docs/extensions/authoring-guide.md` Reference 6
  (Host node/grammar constraints, Manifest/typed-render boundaries, closed
  data surfaces, fallback/schema/version/alias rules, fixture-how-to).
- Recorded `knowledge/reports/2026-08-28-builtin-shortcodes-m10a-conversion-report.md`
  and updated plan status, `knowledge/plans/README.md`, forum/extensions/
  frontend module notes, and `knowledge/index.md`.

## Decisions

- Legacy spellings convert only through the frozen grammar: `topic-tag`
  imports as `category` (`tag_id` → `categoryId`) and never survives canonical
  output; `friend_links` stays literal; `password` is deferred and must not be
  implemented as conversion or credential behavior; out-of-scope legacy tags
  (`media`, `chart`, `alert-*`, `carousel`, `InvitationCode`, `buy`, `code`,
  `topic-comment`) stay literal.
- The classification report is machine-enforced via the shared fixture, not
  prose; structured cases flag `authority: both|host` because the
  TypeScript client inspection is UX-only while the Host validation is
  authoritative (version/argument/attribute/type/body checks are Host-only).

## Next

- M10B (final gate). Before M10B: reconcile four recorded Go/TypeScript parser
  conformance findings (report §Conformance findings) — ID range
  `[2^53, int64-max]`, blank/whitespace-only protected bodies,
  lazy-continuation closing tags after list/blockquote, and list canonical
  newlines — then add those shapes to the shared corpus.
- M10B must also run: full lifecycle matrix, migration/restore evidence,
  runtime fallback matrix per actor, secret-marker scan, catalog/provider/
  digest proof, browser QA, recommended-default/reset verification, and the
  complete final verification list in the task book.

## Open Questions

- Whether legacy `[friend_links]` (underscore, sometimes closeless) should
  permanently remain literal or gain an import alias after a product
  decision; documented today as literal.
- No M10A runtime byte was changed; the exact `sforum-shortcodes@1.3.0`
  artifact, its digest, and the frozen grammar/limits must be re-confirmed at
  M10B rather than assumed.
