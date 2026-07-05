# Decision: Tiptap Editor And Triple Content Storage

## Status

Accepted

## Context

SForum needs a forum editor that can grow into custom toolbar controls,
custom emoji, attachments, mentions, and future structured nodes without
locking the product into a fixed third-party UI.

The user also confirmed that persisted forum content should keep three
representations:

- sanitized HTML for fast SSR/display.
- Markdown source for editing, moderation, exports, and readable history.
- native Tiptap JSON for lossless structured content and future custom nodes.

## Decision

Use Tiptap as the first rich editor foundation in `apps/web`.

Persist forum post/topic body content with three explicit fields when the
forum schema is implemented:

- `content_html_sanitized`: server-generated and sanitized HTML used for
  rendering. Client-generated HTML must never be trusted as the canonical
  stored HTML.
- `content_markdown`: Markdown generated from the accepted Tiptap document,
  used for editable source, audits, and migration/export workflows.
- `content_native_json`: the accepted Tiptap JSON document, used as the
  canonical structure for custom emoji, attachments, mentions, and future
  custom nodes.

The API must own final content acceptance:

- Parse the submitted native JSON/Markdown through an allowlisted Tiptap/forum
  schema.
- Reject or strip unsupported nodes, marks, attributes, protocols, and raw HTML.
- Rebuild Markdown and HTML on the server from accepted content.
- Sanitize rendered HTML with `bluemonday` before writing it to
  `content_html_sanitized`.
- Treat client preview HTML as untrusted draft data only.

## Consequences

- Frontend editing remains flexible enough for custom toolbar and custom emoji
  nodes.
- PostgreSQL can serve SSR/read models quickly from sanitized HTML while still
  retaining editable and structured source data.
- Search indexing should prefer normalized text extracted from accepted content,
  not arbitrary client HTML.
- Forum migrations and post revisions must preserve all three content fields,
  or regenerate derived HTML/Markdown from native JSON when rules change.
- Backend tests are required for allowed and denied content paths, especially
  script tags, event-handler attributes, unsafe URL protocols, images, links,
  and custom node attributes.

## Sources

- Tiptap Nuxt install: https://tiptap.dev/docs/editor/getting-started/install/nuxt
- Tiptap Markdown extension: https://tiptap.dev/docs/editor/markdown
- Tiptap custom extensions: https://tiptap.dev/docs/editor/extensions/custom-extensions
- bluemonday sanitizer: https://github.com/microcosm-cc/bluemonday
