# Custom Image Stickers Use A Generated Catalog And Immutable Core Assets

Date: 2026-07-30

## Context

SForum needs operator-managed custom image sticker packs and plugin-contributed
packs. A sticker may appear in accepted topic/comment content and immutable
revision history, so ordinary plugin lifecycle or mutable image URLs must not
change or break previously accepted content.

Listing every image inside the root Manifest V3 `stickerPacks` array would make
large packs difficult to author and review. Moving the same array through
`includes` would relocate the size without removing the manual work. Runtime
plugin registration would unnecessarily require executable code for static
media.

The existing editor also has a Host-owned `sforumEmoji` node for Unicode
characters. Image stickers have different storage, rendering, retention, and
admin semantics and must not reuse that node.

## Decision

- Core owns sticker pack/item state, operator overrides, media admission,
  immutable asset identity, content canonicalization, rendering, retention,
  permissions, and public/admin APIs.
- Image content uses a separate Host-owned `sforumSticker` atom node.
- Accepted native content snapshots both stable sticker identity and the exact
  verified asset digest. Replacing an item later affects new insertions only.
- Plugin authors place images under a conventional `stickers/<pack>/`
  directory and maintain only small pack-level metadata.
- The existing extension digest workflow generates one normalized
  `stickers/.catalog.json` with exact item paths, digests, media facts, IDs, and
  localization.
- Manifest V3 adds one constant-size logical `stickerCatalog` reference to the
  generated catalog's exact `packageFiles` entry. Individual image entries do
  not grow the root Manifest.
- Host validation recomputes catalog and image identity from the immutable
  package snapshot. Activation imports admitted bytes into Core-owned
  content-addressed storage; public rendering never depends on a live plugin
  process or mutable plugin path.
- Static installation may validate and preview the catalog but does not execute
  plugin code or publish contributions.
- Hiding removes discovery only. Archive prevents new insertion. Existing
  accepted content and accepted revisions remain renderable across operator
  changes, plugin disable/upgrade/rollback/uninstall, restart, and Safe Mode.
- Operator uploads reuse the attachment storage provider through a focused
  sticker service/purpose rather than becoming user-owned post attachments.
- V1 admits PNG, WebP, and GIF up to 2 MiB and 512x512 pixels. SVG and remote
  URL ingestion are rejected.
- Core ignores client-provided URL, digest, dimensions, class, style, and raw
  HTML authority. The server emits only verified intrinsic width/height and a
  narrow sanitized sticker element.
- Shared content styling caps stickers at `128x128` CSS pixels on desktop/tablet
  and `96x96` CSS pixels on mobile while preserving aspect ratio.
- V1 retains any asset used by accepted content or accepted revision history.
  Revision-aware physical garbage collection is deferred.

## Consequences

- Plugin authoring is directory-oriented and the root Manifest stays small.
- Exact-artifact inspection remains possible through the generated catalog and
  complete package digest.
- Plugin packages do not need executable runtime, frontend L2 modules, raw
  database access, or remote media authority to contribute stickers.
- Core storage may retain obsolete media for a long time, but historical
  fidelity and Safe Mode rendering take priority in V1.
- Admin suppression must be stored separately from plugin-owned declarations
  so upgrades cannot erase operator choices.
- Content writes need a server-side sticker resolver before accepted document
  rendering, plus tests that distinguish new insertion from preservation of an
  archived historical reference.
- The new editor may redesign its picker and toolbar independently as long as
  it consumes the stable catalog/insert/node contracts and preserves the size
  and security rules above.

## Rejected Alternatives

- A handwritten root `stickerPacks` array: too large and repetitive.
- `includes.stickers` with the same per-item declarations: moves rather than
  removes authoring complexity.
- Runtime API registration: turns static media into executable behavior.
- Nested pack ZIP files: introduces a second archive extraction and resource
  exhaustion boundary without enough V1 benefit.
- Remote image URLs: cannot guarantee immutable, private, or durable rendering.
- Generic image nodes: allow URL/size semantics that do not preserve catalog
  identity or plugin lifecycle behavior.

## Follow-Up

The next design session must define the new editor and sticker picker UX. See
`../plans/2026-07-30-image-sticker-platform.md`; implementation must not begin
before milestone D1 freezes that product design.

