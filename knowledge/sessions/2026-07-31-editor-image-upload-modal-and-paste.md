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
- `git diff --check`

## Next

- Manual verification: publish a topic and a comment containing only one
  uploaded image, then confirm both render the image after reload. Repeat once
  with paste or drag upload to cover the shared insertion paths.

## Open Questions

- None.
