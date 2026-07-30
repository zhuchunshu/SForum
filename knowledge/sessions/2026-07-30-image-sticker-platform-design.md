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
- Landed the Forum Canvas base surface in production `SFEditor`: extracted a
  focused toolbar component, added paragraph/H2/H3 and strikethrough controls,
  retained write/preview/Markdown/JSON compatibility, and applied the quiet
  desktop/mobile document geometry to full, compact, and basic-field presets.
- Removed the Unicode emoji picker from the production toolbar without
  removing the historical `sforumEmoji` content node. Trusted plugin L2
  extension admission and the existing editor-document payload remain intact.
- Kept the production sticker command closed until the immutable catalog and
  `sforumSticker` node exist; a static client pack or generic image insertion
  is not an acceptable substitute.
- Production Browser QA passed at `http://127.0.0.1:3000/topics/new` in the
  active `sforum.default-theme` template at `1920x960` and `390x844`: no root
  focus shadow, no horizontal page overflow, mobile toolbar scrolling, working
  bold plus preview/Markdown/JSON transitions, and no console warnings/errors.

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
- The redesigned editor keeps write, preview, Markdown, and native JSON modes
  so inspection and existing content workflows do not regress.

## Next

- Begin M0 contract and threat-model work from
  `../plans/2026-07-30-image-sticker-platform.md` before adding a production
  sticker button.
- Freeze the desktop picker container, favorites scope, and inline versus
  sticker-only paragraph behavior alongside the real catalog DTO.
- Add the production picker and plugin toolbar contribution UI only after the
  Host catalog, immutable media resolution, and `sforumSticker` node are
  available.

## Open Questions

- Desktop picker form after direction selection: popover or persistent side
  panel. The demos consistently use a mobile bottom drawer.
- Whether favorites belong beside search and recent use in V1.
- Inline versus sticker-only paragraph behavior within the accepted size cap.
