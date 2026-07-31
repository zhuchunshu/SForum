# 2026-07-31 Editor Image Upload Modal And Paste

## Changed

- Replaced the shared editor toolbar image picker's direct native-file-dialog
  behavior with a focused upload modal. The modal drop zone supports clicking
  to choose multiple images and dragging files into the same surface.
- Added clipboard image upload for the shared Tiptap editor. Raster image files
  paste at the current selection and use the established attachment upload
  policy, placeholder mapping, attachment identity, and Toast feedback.
- Fixed topic/comment publication when the accepted editor document contains
  images but no text. Forum meaningful-content validation now recognizes
  normalized image nodes while preserving empty-document rejection and the
  existing attachment identity/reference checks.
- Aligned comment quick reply, advanced reply, and edit submission guards with
  the native editor document. A payload containing an image node no longer
  returns silently when its text and Markdown serializers are empty.
- Added server-provided image dimensions and bounded `compact`, `standard`, and
  `wide` display modes. Topic/comment content now caps inline images
  independently, and image clicks open an isolated PhotoSwipe gallery with an
  authorized original-media fallback.
- Edit dirty-state signatures now include native editor JSON, so changing only
  an image display mode enables the topic or comment save action.

## Decisions

- The editor intercepts only image file data on paste. Text, links, and other
  non-image clipboard content retain normal Tiptap paste behavior.
- SVG remains rejected for all image-upload entry points, including paste.
- Image nodes remain visual content: `plain_text` and excerpts stay empty for
  image-only posts instead of fabricating searchable text from filenames.

## Verification

- `cd apps/web && bun run typecheck`
- `cd apps/web && bun test tests/framework/editorImageUpload.test.ts tests/framework/editorCanvas.test.ts`
- `cd apps/api && go test ./...`
- `cd apps/web && bun test tests/framework/editorImageUpload.test.ts tests/framework/editorL2Load.test.ts`
- `cd apps/web && bun test tests/forum/commentImageOnlySubmission.test.ts`
- `cd apps/web && bun test tests/framework/editorImageUpload.test.ts tests/framework/editorL2Load.test.ts tests/forum/defaultThemeTopicPage.test.ts`
- `cd apps/web && bun run typecheck`
- `node tests/validate-architecture-boundaries.mjs`
- `ruby scripts/validate-openapi-refs.rb`
- `git diff --check`

## Next

- Manual verification: edit an existing topic and comment, change only the
  image display mode, and confirm save becomes available. Then publish a topic
  and comment with large/tall images, check desktop/mobile inline caps, and
  open both galleries to verify the original route loads only after opening.

## Open Questions

- None.
