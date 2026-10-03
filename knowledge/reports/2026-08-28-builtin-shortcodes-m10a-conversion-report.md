# Built-in Shortcodes M10A Legacy Conversion Report

Date: 2026-08-28  
Scope: M10A only — compatibility fixtures, conformance verification, and
documentation. No runtime, Host API, registry, permission, storage, or
protected-content change was made.

## Outcome

M10A adds one language-neutral legacy-conversion fixture and focused Go and
TypeScript conformance tests, and produces the user, extension-author, and
operator documentation for the built-in shortcodes. The deterministic
conversion report below is enforced by the tests, not by prose.

- Fixture (new): `contracts/fixtures/shortcode-legacy-conversion-v1.json`
  (29 text cases + 15 structured cases + the frozen declaration/alias/
  unsupported tables).
- Fixture (existing M2): `contracts/fixtures/shortcode-text-v1.json`
  (16 cases), still consumed unchanged by the M2 Go/TypeScript tests.
- Go tests:
  `apps/api/app/Support/EditorDocument/shortcode_legacy_conformance_test.go`
  (declaration parity, alias/unsupported parity, text classification,
  structured Host rejection, completeness, fallback code/label parity).
- TypeScript tests:
  `apps/web/tests/framework/editorShortcodeLegacyConformance.test.ts`
  (same fixture: classification equality, canonical output, client-visible
  fallback codes, structured client-inspection coverage).

## Legacy syntax evidence

The old SForum (PHP era) implemented these shortcodes through
`ShortCodeR`/Thunder-shortcode patterns and editor insert helpers. Historic
source consulted (git history, not runtime):

- `app/Plugins/Core/src/Lib/ShortCode.php` (annotation handlers for
  `login`, `reply`, `password`, `user` `user_id`, `comment` `comment_id`,
  `topic-tag` `tag_id`, and friends via `friend_links`);
- `app/Plugins/Core/src/Lib/ShortCodeR/Defaults.php`
  (`topic` via `topic_id`; `button`/`file`/`chart` positional styles);
- `app/Plugins/Core/src/ShortCode/Single.php` (`friend_links` widget tag,
  underscore name, no declared arguments);
- `app/Plugins/Topic/src/Lib/ShortCode/ShortCode.php`
  (`only-author`, comment-only);
- editor inserts: `[login]content[/login]`,
  `[reply]content[/reply]`, `[only-author]content[/only-author]`,
  `[password ...]content[/password]` (TinyMCE `sf-hidden` plugin),
  `[friend_links]` (widget tag without closing tag).

The final legacy spellings map to V1 canon as follows:

| Legacy syntax | V1 classification | Canonical V1 |
| --- | --- | --- |
| `[login]body[/login]` | converted | `sforum-shortcodes.login@1` block |
| `[reply]body[/reply]` | converted | `sforum-shortcodes.reply@1` block |
| `[only-author]body[/only-author]` (comment only) | converted | `sforum-shortcodes.only-author@1` block |
| `[user user_id="42"][/user]` / `user_id=42` | converted | `sforum-shortcodes.user@1` ref, `userId` 42 |
| `[topic topic_id="123"][/topic]` / `topic_id=123` | converted | `sforum-shortcodes.topic@1` ref, `topicId` 123 |
| `[comment comment_id="456"][/comment]` | converted | `sforum-shortcodes.comment@1` ref, `commentId` 456 |
| `[category category_id="7"][/category]` | converted | `sforum-shortcodes.category@1` ref, `categoryId` 7 |
| `[topic-tag tag_id="7"][/topic-tag]` | converted (alias) | `sforum-shortcodes.category@1` ref, `categoryId` 7 |
| `[friend-links][/friend-links]` | converted | `sforum-shortcodes.friend-links@1` ref, no args |
| `[friend_links]` / `[friend_links][/friend_links]` | literal | legacy underscore name, not an alias |
| `[password ...]body[/password]` | unsupported (deferred) | no V1 declaration; input stays literal |
| `[media]`, `[chart]`, `[alert-*]`, `[carousel]`, `[button]`, `[file]`, `[InvitationCode]`, `[topic-comment]`, `[buy]`, `[code]` | literal | outside the V1 eight |

## Deterministic conversion report

Structure-level results of the new fixture, by classification (assertions use
`ParseShortcodeMarkdown` + `Accept` on Go and `inspectSForumShortcodeMarkdown`
+ the Tiptap editor on TypeScript; both consume the same fixture file):

| Classification | Fixture count (new) | M2 corpus count | Meaning |
| --- | --- | --- | --- |
| converted | 13 | 6 | Activates, normalizes to exactly one canonical node set; canonical export asserted by tests |
| literal | 4 | 5 | Unknown/out-of-scope/legacy-name text remains byte-identical input |
| invalid | 10 | 4 | Recognized name or alias but argument/ID/closure/nesting/placement/body rules violated; text stays literal, structured nodes are rejected |
| unsupported | 1 | 0 | `password` recognized as legacy but deferred; no conversion, no credential behavior |
| over-limit | 1 | 1 | Depth-5 nesting (new) and 33 nodes (M2) stay literal and fail budgets |
| structured rejections (invalid/over-limit) | 15 | — | native JSON nodes: 8 asserted by both Host and client inspection, 7 by Host authority only |

Per new-fixture case:

- converted: `legacy-login-block`, `legacy-reply-block`,
  `legacy-only-author-comment`, `legacy-user-quoted`,
  `legacy-user-unquoted`, `legacy-topic-unquoted`,
  `legacy-comment-quoted`, `legacy-category-canonical`,
  `legacy-topic-tag-alias`, `legacy-friend-links-canonical`,
  `legacy-protected-nested-unicode-ref`,
  `legacy-protected-body-code-fence`, `legacy-ref-inside-protected`.
- literal: `legacy-friend-links-underscore-literal`,
  `legacy-friend-links-closeless-literal`,
  `legacy-out-of-scope-media-literal`,
  `legacy-escaped-opening-literal`.
- invalid: `legacy-user-wrong-argument-key`,
  `legacy-user-zero-id`, `legacy-user-negative-id`,
  `legacy-user-non-numeric-id`, `legacy-user-overflow-id`,
  `legacy-user-missing-closing`, `legacy-duplicate-argument`,
  `legacy-mismatched-nesting`, `legacy-only-author-topic`,
  `legacy-empty-protected-body`.
- unsupported: `legacy-password-deferred`.
- over-limit: `legacy-depth-five-nesting`.

Structured cases (`authority: both` = Host validation plus the client
`inspectSForumShortcodeDocument` both reject; `authority: host` = the client
inspection is UX-only and the Host is the authority):

- both: `structured-unknown-identity`, `structured-password-identity`,
  `structured-only-author-in-topic`, `structured-empty-protected-body`,
  `structured-reference-in-paragraph`,
  `structured-reference-in-list-item`, `structured-over-limit-nodes`,
  `structured-over-limit-depth`.
- host: `structured-wrong-contract-version`,
  `structured-unknown-argument`, `structured-missing-argument`,
  `structured-float-argument`, `structured-reference-with-body`,
  `structured-protected-as-reference`, `structured-extra-attribute`.

Coverage required by M10A (ordinary references, protected blocks, nesting,
escaping, code blocks, Unicode, duplicate arguments, wrong IDs, `topic-tag`
alias, protected child nodes): the M2 corpus pins
escaped openings, fenced/indented/inline/raw-HTML opacity, duplicate arguments,
mismatched nesting, inline placement, 33-node budget, alias normalization,
Unicode nested content, and `only-author` scope. The new corpus adds the
legacy spellings, wrong-ID families, empty-body, missing-closing, depth
budget, code-inside-protected-body, escaped legacy opening, password
deferral, and structured rejections.

## Go / TypeScript parity verification

For every fixture case the two conformance suites assert the same outcome:

- classification: Go `ParseShortcodeMarkdown` activation (plus `Accept`) and
  TypeScript `inspectSForumShortcodeMarkdown(...).activated && .withinBudgets`
  agree on converted vs not-converted for all 29 text cases;
- canonical export: Go `Accepted.Markdown` and
  Tiptap `editor.getMarkdown()` are byte-identical for all 13 converted cases;
- node structure (type/ID/arguments/depth) is identical;
- fallback codes: the frozen declaration table
  (`shortcode.user.unavailable`, `shortcode.topic.unavailable`,
  `shortcode.comment.unavailable`, `shortcode.category.unavailable`,
  `shortcode.friend_links.omitted`,
  `shortcode.protected.unavailable`) is asserted on the Host side
  (`shortcodeFallback`) and on the client side (`data-fallback` in rendered
  HTML for top-level nodes; friend-links renders nothing);
- structured rejections: Go `validateAndNormalize` returns
  `ErrInvalidShortcode` for all 15 cases and the public `Accept` pipeline
  fails closed; TypeScript client inspection rejects the 8 `both` cases.

## Conformance findings (not fixed by M10A)

While building the corpus, probing found existing M2-era parser divergences
between the Go and TypeScript implementations. They are outside the allowed
M10A scope (runtime parser changes) and are recorded for the M10B gate. None
of them appears in the shared corpus, so the corpus stays green.

1. **ID boundary `[2^53, 2^63-1]`.** Go parses and normalizes e.g.
   `[user user_id=9007199254740992][/user]` (positive int64 per the frozen
   ADR); TypeScript holds `Number.isSafeInteger` and keeps the input literal.
   Go: activated. TS: literal. The frozen contract requires positive int64,
   so the TypeScript parser deviates.
2. **Blank or whitespace-only protected body.** `[login]\n\n[/login]` and
   `[login]\n   \n[/login]`: Go keeps the input literal; TypeScript activates
   a block whose body is an empty paragraph, which the Host would then reject
   on save (empty protected body invalid).
3. **Closing tag absorbed as lazy continuation.** `[login]\n> quote\n[/login]`
   and `[login]\n- a\n- b\n[/login]` (no blank line before the closing tag):
   Go's Goldmark AST marks the closing line as part of the blockquote/list
   and keeps the input literal; the TypeScript line heuristic activates.
4. **Canonical list rendering.** A bullet list followed by a heading inside a
   protected body re-renders `- item\n...\n\n\n## heading` in Go but
   `- item\n...\n\n## heading` in TypeScript.

Recommended M10B action: reconcile the TypeScript parser with the frozen
grammar (or, if a product decision changes the grammar, revise the ADR and
both parsers together), then add these four shapes to the shared corpus with
an explicit expectation. Before M10B starts, the teams should also decide
whether legacy `[friend_links]` (underscore, sometimes without a closing tag)
should ever convert, or remain permanently literal as documented now.

## Documentation delivered

- `docs/zh-CN/usage/shortcodes.md` and `docs/en-US/usage/shortcodes.md`:
  the eight syntaxes with examples and limits, `topic-tag` compatibility,
  `password` deferral, protected-content matrix for anonymous/unauthorized/
  disabled/Safe Mode, public-body/search/SEO/excerpt exclusion, and the
  edit/preview/re-edit workflow, plus the operator lifecycle and exclusion
  rules.
- `docs/zh-CN/usage/README.md` and `docs/en-US/usage/README.md`: chapter links.
- `docs/extensions/authoring-guide.md` Reference 6: Host node/grammar
  constraints, Manifest/Content Registry/typed-render boundaries, the
  closed data surface (no Core DB, raw content, actor/session, or Query
  delegation), fallback/schema/version/alias rules, and how to add a
  fixture and conformance case.

## M10B startup items (runtime / full gate)

1. Reconcile the four parser divergences above with a decision, then extend
   the shared corpus and both implementations.
2. Full lifecycle matrix: install/enable/disable/upgrade/rollback/uninstall,
   API restart, crash, Safe Mode, SyncBuiltins restaging, normal admin
   activation, catalog/provider/digest proof after exact-artifact
   activation.
3. Migration/restore evidence for historical content and revisions.
4. Runtime fallback matrix per actor (anonymous, unrelated, author, staff,
   reply-eligible) on real topic/comment surfaces, desktop and `390x844`.
5. Secret-marker scan across public API, SSR HTML, Nuxt payload, search
   indexes/results, SEO/OG/JSON-LD, shared caches, traces/logs, observe
   events, and webhooks with the built-in plugin enabled in production mode.
6. Recommended-default and reset verification for any operator-facing
   setting (the shortcode plugin itself ships no settings, so this is about
   the plugin enable state and any future options).
7. Confirm `sforum-shortcodes` activation state remains at the staged
   exact artifact and that no declaration/version was changed by M10A
   (M10A changed no runtime bytes).

## Boundary statement

M10A performed no runtime change: no Host API, Query Registry, Content
Registry, permission, storage format, public API, protected-content security
model, plugin runtime handler, per-node dispatcher, cache, or search logic
was modified. Fixtures and tests that appear to exercise behavior only
consume existing parsers/validation; parser fixes are explicitly deferred to
M10B.
