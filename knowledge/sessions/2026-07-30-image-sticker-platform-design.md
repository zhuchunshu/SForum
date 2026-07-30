# 2026-07-30 Image Sticker Platform Design

## Changed

- Added the active custom image sticker platform task book.
- Recorded the generated plugin catalog, immutable Core asset, content node,
  lifecycle, admin, permission, and rendered-size decisions.
- Distinguished custom image stickers from the current Unicode `sforumEmoji`
  behavior.
- Added three standalone interactive editor directions under
  `tmp/demos/sforum-editor-sticker-directions-20260730/`: Forum Canvas, Focus
  Manuscript, and Creator Workbench.
- Integrated the operator-supplied D01 pack with separate 32px picker and 128px
  content assets, picker search, recent use, cursor insertion, and preview sync.
- DOM verification passed for all three directions. Visual Browser QA could
  not run because the in-app browser blocks local `file://` pages.
- The operator selected Forum Canvas (direction 01) for refinement. Accent
  focus outlines and hover fills were removed from its text/select/editor
  inputs while command-button focus and selected states remain visible.

## Decisions

- Plugin authors maintain conventional sticker directories; the extension CLI
  generates one exact catalog and the root Manifest contains one fixed-size
  reference instead of one entry per image.
- Accepted content snapshots the exact asset digest, so later replacement,
  hide, archive, disable, upgrade, rollback, or uninstall cannot rewrite or
  break historical content.
- Core caps rendered stickers at `128x128` on desktop/tablet and `96x96` on
  mobile. Client dimensions/styles are never authoritative.
- PNG, WebP, and GIF are admitted within 2 MiB and 512x512; SVG and remote URLs
  are excluded from V1.

## Next

- Continue refining Forum Canvas from operator feedback.
- Freeze toolbar composition, picker geometry, responsive behavior, keyboard
  states, and plugin L2 command coexistence after that selection.
- After editor design approval, begin M0 contract/threat-model work from
  `../plans/2026-07-30-image-sticker-platform.md`.

## Open Questions

- Desktop picker form after direction selection: popover or persistent side
  panel. The demos consistently use a mobile bottom drawer.
- Whether favorites belong beside search and recent use in V1.
- Inline versus sticker-only paragraph behavior within the accepted size cap.
- Whether the redesigned editor retains write/preview/Markdown/native modes.
