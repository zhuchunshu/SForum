# 2026-07-30 Attachment Image Display Variants

## Status

Accepted and implemented for Host JPEG/PNG processing.

## Context

Ordinary attachments retained only their uploaded bytes. The Attachment
Configuration compression tab therefore had no real processor, operators could
not inspect queue health or savings, and forum image URLs always returned the
original object. A safe implementation must preserve originals, reuse existing
storage adapters and authorization, avoid blocking uploads, and recover when a
queue insert or worker process fails.

## Decision

1. The uploaded attachment is immutable. Core may generate one derived variant
   named `display`; downloads, audits, and later processing continue to use the
   original.
2. V1 processes JPEG and PNG only. JPEG compression strength `0..100` maps to
   quality `95..70`; PNG uses progressively stronger lossless encoding. EXIF
   orientation is applied before an optional proportional resize.
3. Recommended defaults are enabled, strength `55`, maximum dimension `2560`,
   minimum source size `256 KB`, and minimum savings `8%`.
4. `attachment_variants` owns derived-object metadata.
   `attachment_compression_tasks` is the durable source of task state and
   deduplicates by attachment, variant name, and policy digest. River is the
   delivery mechanism, not the authority.
5. A one-minute reconciler republishes pending work. A worker claim has a
   15-minute lease; stale work is reclaimable and becomes terminally observable
   after three attempts.
6. The display endpoint executes the original attachment read policy. Missing,
   stale, disabled, unreadable, or unprofitable variants transparently fall
   back to the original bytes.
7. Proxied JPEG/PNG attachment metadata returns the display endpoint as `url`,
   so existing consumers benefit without polling. Explicit site-public assets
   that already use a provider/CDN URL keep that URL.
8. Saving settings affects new tasks only. Historical images require an
   explicit, permission-protected backfill command.
9. This is a Host implementation. It does not claim a general plugin Media
   Pipeline byte-processing ABI; a later provider-neutral transform contract
   requires a separate decision.

## Consequences

- Upload latency and original-file integrity are unchanged.
- A generated variant never becomes an authorization shortcut.
- Storage usage temporarily includes both original and useful display bytes;
  physical attachment cleanup deletes both.
- Policy changes make old variants ineligible immediately, while explicit
  backfill controls regeneration cost.
- WebP, AVIF, thumbnails, animated formats, and plugin-provided transforms are
  intentionally outside V1.
