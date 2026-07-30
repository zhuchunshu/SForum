# 2026-07-31 Editor Image Upload Modal And Paste

## Changed

- Replaced the shared editor toolbar image picker's direct native-file-dialog
  behavior with a focused upload modal. The modal drop zone supports clicking
  to choose multiple images and dragging files into the same surface.
- Added clipboard image upload for the shared Tiptap editor. Raster image files
  paste at the current selection and use the established attachment upload
  policy, placeholder mapping, attachment identity, and Toast feedback.

## Decisions

- The editor intercepts only image file data on paste. Text, links, and other
  non-image clipboard content retain normal Tiptap paste behavior.
- SVG remains rejected for all image-upload entry points, including paste.

## Verification

- `cd apps/web && bun run typecheck`
- `cd apps/web && bun test tests/framework/editorImageUpload.test.ts tests/framework/editorCanvas.test.ts`
- `git diff --check`

## Next

- Manual verification: use the toolbar image command to open the modal, click
  its drop zone and drag images into it, then paste a screenshot at the editor
  cursor and confirm the uploaded images preserve their insertion positions.

## Open Questions

- None.
