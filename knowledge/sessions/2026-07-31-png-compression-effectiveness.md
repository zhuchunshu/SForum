# 2026-07-31 PNG Compression Effectiveness

## Changed

- Improved Host attachment PNG display variants without changing immutable
  originals or the existing `display` URL and fallback contract.
- Recommended strength now selects the standard library's highest lossless
  PNG Deflate level.
- Low-color PNGs with at most 256 exact RGBA colors are encoded as indexed PNG,
  preserving transparency and pixel values while substantially reducing size.
- Added a focused test covering a transparent low-color image and decoded pixel
  equality; the Attachments, Jobs, and Attachments controller tests pass.

## Decisions

- Keep PNG processing lossless as established by the display-variant decision.
- Do not introduce a native `pngquant`/`oxipng` runtime dependency; high-color
  PNGs continue using the bounded standard-library encoder.

## Next

- Observe production compression statistics after an explicit historical
  backfill to confirm the low-color PNG cohort's saved bytes.

## Open Questions

- Lossy PNG quantization or WebP/AVIF output would require a separate format or
  Media Pipeline decision and is intentionally not part of this fix.
