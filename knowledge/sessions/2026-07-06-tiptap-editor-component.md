# 2026-07-06 Session Handoff

## Changed

- Added direct Tiptap dependencies to `apps/web/package.json` and
  `apps/web/bun.lock`.
- Replaced the old textarea-backed `SFEditor` with a Tiptap-backed editor that
  supports custom toolbar actions, custom emoji nodes, preview, Markdown source,
  native JSON inspection, character/word counts, and a `content-change` payload.
- Moved editor extension setup, URL helper logic, custom emoji node definition,
  and content payload types into `apps/web/app/utils/sfEditor.ts` so the Vue
  component remains focused on state and rendering.
- Expanded the dev-only `/components` composer section to show the editor and
  the three future storage representations: sanitized HTML, Markdown source,
  and native JSON.
- Updated component CSS for the editor toolbar, view modes, preview rendering,
  source/native output, emoji picker, and triple-output panels.

## Decisions

- Use Tiptap as the first forum editor foundation.
- Store accepted forum content as sanitized HTML, Markdown, and native Tiptap
  JSON when the forum schema is implemented.
- Treat client-generated HTML as preview/draft data only; API-side content
  acceptance must regenerate and sanitize stored display HTML.

## Next

- Implement backend content normalization and XSS regression tests before
  shipping topic/post write endpoints.
- Wire real attachment upload into the image toolbar action instead of URL
  prompts.
- Add mention and attachment nodes once forum user/topic endpoints exist.

## Open Questions

- Whether `content_native_json` or `content_markdown` should be the canonical
  edit source for revision diffs in the first forum milestone.
