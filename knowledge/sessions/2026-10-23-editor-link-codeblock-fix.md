# 2026-10-23 Editor Link And Code-Block Paste Fix

## Changed

- `apps/web/app/utils/editor/link.ts` (new): `SForumLink` (`inclusive: false`)
  and `insertSForumLink` (empty-selection insert of the URL as link text).
- `apps/web/app/utils/editor/clipboard.ts` (new): Host paste normalization —
  literal text inside code blocks (plain-text first, HTML→text fallback) and
  code-only HTML collapsed into one `<pre><code>` with language detection and
  escaped output.
- `apps/web/app/utils/editor/codeBlock.ts` (new): `SForumCodeBlock` extends
  `@tiptap/extension-code-block` and merges a top-level multi-textblock
  selection into one code block.
- `apps/web/app/utils/sfEditor.ts`: uses `SForumLink`, registers the clipboard
  extension before Link/shortcode paste handlers, disables StarterKit's
  `codeBlock` (always) and adds `SForumCodeBlock` only for the `full` preset.
- `apps/web/app/components/SFEditor.vue`: `setLink()` skips code-block contexts;
  an empty selection outside an existing link inserts the URL as link text
  instead of setting a stored mark.
- `apps/web/package.json` + `bun.lock`: direct
  `@tiptap/extension-code-block@3.31.3`.
- Tests: `tests/framework/editorLinkBehavior.test.ts`,
  `tests/framework/editorCodeBlockPaste.test.ts`,
  `tests/helpers/editorClipboard.ts`.

## Root Causes / Decisions

- Link: Tiptap v3's `Link.inclusive()` returns `options.autolink`, so an
  autolinked/pasted link remained inclusive; typing at its end extended the
  mark. ProseMirror drops non-inclusive marks at the boundary, so fixing the
  mark alone stops continuation. The toolbar previously used `setMark` on an
  empty selection, which wrote stored marks and linked all following text;
  that path now inserts the URL as link text and leaves the cursor outside.
- Code block: plain-text paste into an active code block already worked.
  Splitting was reproduced for (a) HTML-only clipboards (ProseMirror parses
  HTML and splits the block), (b) line-wrapped `<pre>`/`<div>` code HTML
  (one block per `<pre>`), and (c) `toggleCodeBlock` over multiple paragraphs
  (native `toggleNode` per textblock). The fix covers all three.
- Merge scope is deliberately limited to a top-level `doc` range of textblocks.
  List/table/blockquote multi-block selections still use native behavior; this
  is a known limitation, not a regression.
- Do not promise Markdown source mode; storage remains Markdown + native JSON as
  before. No API/OpenAPI/Go/DB changes.

## Verification

- `bun test`: 1565 pass / 0 fail (189 files).
- `bun run typecheck`: exit 0.
- `node tests/validate-architecture-boundaries.mjs`: pass (1922 production
  files).
- Browser QA on the real Nuxt dev server (`/components` page, 1633x810):
  - Paste `https://example.com`, then type ` 普通内容`: only the URL is wrapped
    in `<a>`; following text is outside.
  - Toolbar link with an empty selection (stubbed `window.prompt`): inserted
    `https://example.com/` as the only anchor; following ` 普通内容` stayed
    outside.
  - Type ` ```shell ` then paste HTML-only
    `<pre><code>line1<br>line2<br><br>line3</code></pre>`: one
    `<pre><code class="language-shell">` with `line1\nline2\n\nline3`.
  - Paste three `<pre>` siblings into a paragraph: one `language-shell` code
    block with `line1\nline2\nline3`.
  - Paste `line1\n\nline2\n\nline3`, select all, `Control+Alt+c`: one
    `<pre><code>` with `line1\nline2\nline3`.
- Browser QA used a local throwaway account `editorqa` (id 216) created on the
  dev database; remove it if unwanted. Dev API/web servers were started only
  for QA and stopped afterwards.

## Next

- Optional: cover nested multi-block merges (blockquote/list) if product wants
  the same one-block behavior there.
- Optional: collapse wrapper-based code HTML (`figure > pre`, line-number
  tables) if real clipboard samples show it; current rule intentionally only
  handles top-level code-only `pre` containers.

## Open Questions

- None blocking. The reported real-world clipboard shape was not captured; the
  three reproduced paths cover the observed failure modes.
