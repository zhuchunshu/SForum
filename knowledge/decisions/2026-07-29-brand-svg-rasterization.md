# Brand SVG Uploads Use Server-Side Rasterization

## Status

Accepted on 2026-07-29.

## Context

Operators need to upload SVG logos and icons from the Personalization Brand
tab. SForum deliberately rejects `image/svg+xml` as active content in ordinary
attachments because a public same-origin SVG can contain scripts, event
handlers, external resources, and other executable behavior.

## Decision

- Accept `.svg` only through `POST /admin/site/brand-assets`.
- Parse with `github.com/srwiley/oksvg` and render with
  `github.com/srwiley/rasterx` (BSD-3-Clause, established Go SVG raster stack).
- Limit source SVG to 2 MB, require a finite positive viewBox and at least one
  drawable path, cap the output edge at 1024 pixels, and reject blank output.
- Store and publish only the resulting transparent PNG. Apply the existing
  `site-brand` Media Registry policy, storage provider, event, metadata, and
  site attachment-reference flow to that PNG.
- Keep the global SVG active-content denylist unchanged.

## Alternatives

- Publishing the original SVG was rejected because storage-provider public URLs
  can bypass Host response headers and expose same-origin executable content.
- Browser-only sanitization was rejected because upload security must remain
  server authoritative.
- A handwritten SVG allowlist sanitizer was rejected because SVG/CSS/URL
  interactions are too broad for a small custom security parser.

## Consequences

- Operators can select or drag SVG brand artwork, but the saved asset is PNG
  rather than a vector original.
- SVG features outside the parser's supported drawing subset, including
  script-only or text-only documents without drawable paths, fail validation.
- General attachment behavior and its existing security tests do not change.
