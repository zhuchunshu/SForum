# Decision: Rich Content Rendering — GFM Extensions And Syntax Highlighting

## Status

Accepted

## Context

The post body rendering pipeline (Tiptap editor → Markdown → Go goldmark →
bluemonday sanitize → frontend `v-html` into `.sf-prose`) had two gaps that
made rendered content look broken:

- **Front/back renderer mismatch.** The Tiptap editor was configured with
  `gfm: true`, but the Go renderer called `goldmark.Convert(...)` with **no
  extensions** (`renderer.go`). As a result GFM-only constructs were silently
  dropped on publish:
  - Tables became literal `| a | b |` text.
  - `~~strikethrough~~`, `- [ ]` task lists, and bare-URL autolinks were lost.
  - The editor preview did not match the published output.
- **No syntax highlighting.** Code blocks rendered as a single flat color.

The baseline `.sf-prose` typography (@tailwindcss/typography + teal tokens)
was acceptable; it was undermined by the issues above.

A demo (`tmp/prose-style-demo.html`) presented four candidate prose styles.
The operator selected **"A · Classic Teal"** (refined current style) and
approved fixing the backend GFM rendering and adding code highlighting in the
same change.

## Decision

Apply three coordinated changes; do **not** batch-rerender existing posts.

### 1. Backend: enable GFM extensions and relax the sanitizer

- `apps/api/.../renderer.go`: replace `goldmark.Convert(...)` with
  `goldmark.New(goldmark.WithExtensions(extension.GFM)).Convert(...)`.
  GFM bundles Table, Strikethrough, Linkify, and TaskList, matching the
  editor's `gfm: true`.
- `sanitizeHTML` (same file) adds two **narrow, regex-gated** allowances on
  top of `bluemonday.UGCPolicy()`:
  - `class="language-<lang>"` on `<code>` only (regex `^language-[a-z0-9+#.-]+$`)
    so the highlighter can identify the language. UGCPolicy strips this by
    default (verified empirically).
  - `<input type="checkbox">` with only `checked`/`disabled` attributes, so
    GFM task lists render their checkboxes. UGCPolicy strips `<input>` by
    default; non-checkbox inputs and event-handler attributes are still
    stripped (verified — the relaxation is convergent).
- Tables (`<table>...`), `<del>`/`<s>`, and autolink `<a>` are already kept by
  stock UGCPolicy — no sanitizer change needed for those.

### 2. Bump `RenderVersion` to v2

- `apps/api/.../types.go`: `goldmark-bluemonday-v1` → `goldmark-bluemonday-v2`.
- Existing posts keep their stored v1 HTML until each is next edited, at which
  point the existing create/update path re-renders to v2. **No migration or
  batch-rerender job is added** (none existed before); this is the documented
  trade-off to avoid a data-rewriting migration.

### 3. Frontend: highlight.js via a Vue directive + Classic Teal prose

- New client plugin `apps/web/app/plugins/highlight.client.ts` registers a
  global `v-highlight` directive that, on `mounted`/`updated`, scans
  `pre code` blocks inside the bound element and highlights them. It uses
  `highlight.js/lib/core` with ~14 common languages registered by alias to
  control bundle size; unregistered languages fall back to auto-detection.
- `apps/web/app/assets/css/highlight-theme.css` defines teal-harmonized token
  colors for light/dark, registered in `nuxt.config.ts`.
- `v-highlight` is attached to the three `v-html` content containers: the
  topic body (`pages/t/[...path].vue`), comment bodies (`SFComment.vue`), and
  the editor preview (`SFEditor.vue`).
- `.sf-prose` in `main.css` is refined to Classic Teal (teal list markers and
  blockquote rail, dark `pre` background that coexists with highlight.js
  token spans, bordered zebra-striped tables with horizontal scroll on narrow
  screens, task-list checkbox styling).

## Consequences

- Published content now matches the editor preview for tables, strikethrough,
  task lists, and autolinks; code blocks are syntax-highlighted.
- The sanitizer relaxation is intentionally minimal and regex-scoped; backend
  tests assert both GFM output preservation and that the relaxation stays
  convergent (non-checkbox inputs and event-handler attributes are dropped).
- Old posts are **not** upgraded automatically. Operators who want old posts
  to gain GFM features must edit them (or a future batch-rerender job, out of
  scope here, must be written).
- Adding a new highlight.js language requires registering it in
  `highlight.client.ts`.

## Sources

- goldmark extensions: https://github.com/yuin/goldmark#extensions
- bluemonday UGCPolicy: https://github.com/microcosm-cc/bluemonday#usage
- highlight.js: https://highlightjs.org/
