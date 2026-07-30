# Custom Image Sticker Platform - Design And Task Book

Status: **active** - platform architecture is approved; the Forum Canvas base
editor is implemented, while the sticker contract and product remain unbuilt

Date: 2026-07-30
Last updated: 2026-07-30 - Forum Canvas production base verified

Goal: let operators upload and manage custom image sticker packs, let plugins
contribute packs without a large handwritten Manifest, and let forum content
render immutable stickers safely without historical posts breaking when packs
are hidden, removed, disabled, upgraded, or uninstalled.

This plan covers custom image stickers only. Unicode emoji, reactions, likes,
and an emoji character picker are separate product areas.

## Required Reading

Before implementation, read:

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/forum.md`
4. `knowledge/modules/extensions.md`
5. `knowledge/modules/attachments.md`
6. `knowledge/decisions/2026-07-30-image-sticker-catalog.md`
7. `knowledge/decisions/2026-07-06-tiptap-editor-content-storage.md`
8. `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
9. the current hot sticker handoff under `knowledge/sessions/`
10. this task book

## Confirmed Product Decisions

- A sticker is an operator- or plugin-supplied image, not Unicode emoji.
- Core owns the canonical pack/item catalog, immutable content identity,
  authorization, rendering, storage retention, and admin overrides.
- Admin-managed packs support create, edit, reorder, hide, and remove.
- Plugin packs are declarative exact-artifact contributions. Operators may
  hide or suppress them, but Core does not edit bytes inside a plugin package.
- Hiding affects discovery only. Existing forum content remains renderable.
- Removing is a soft archive. Existing posts and accepted revisions remain
  renderable; new content cannot insert an archived item.
- Plugin disable, upgrade, rollback, uninstall, and Safe Mode cannot break
  already accepted sticker content.
- Root Manifest files do not contain an item per sticker. A generated catalog
  lists the image files and their exact metadata.
- Client-supplied image URLs, dimensions, inline styles, and arbitrary HTML are
  never authoritative.
- Rendered stickers have Core-owned maximum dimensions in every content
  surface. Desktop/tablet maximum is `128x128` CSS pixels; mobile maximum is
  `96x96` CSS pixels.
- The sticker platform exposes an editor integration contract, but the new
  editor's picker, mobile interaction, search, recent-use behavior, and
  composition geometry must remain aligned with the selected Forum Canvas
  direction when their production contract is frozen.

## Current Editor Design Gate

Interactive comparison demos live at
`tmp/demos/sforum-editor-sticker-directions-20260730/index.html`:

- Forum Canvas: compact, familiar default forum composer.
- Focus Manuscript: reduced chrome and an editorial writing surface.
- Creator Workbench: persistent outline, editor, and publish inspector.

The operator selected Forum Canvas as the base direction. Its input surfaces
use quiet focus treatment without an accent outline or hover fill; command
buttons retain visible keyboard focus and selected states. The production
`SFEditor` now uses this base geometry and retains write, preview, Markdown,
and native JSON modes. The Unicode emoji picker is removed from the toolbar,
but historical `sforumEmoji` documents remain supported.

All three use the supplied D01 image sticker pack, not Unicode emoji. They
share search, recent-use tabs, insertion at the current selection, preview
sync, 32px picker thumbnails, 128px content assets, a desktop/tablet `128x128`
cap, and a mobile `96x96` cap. These are design artifacts, not production
implementation of the sticker product. The editor base surface is accepted;
the real picker remains gated on the Host catalog and `sforumSticker` node.

## Existing Baseline To Reuse

| Area | Existing owner | Required treatment |
| --- | --- | --- |
| Structured content | `Support/EditorDocument` | Add one Host-owned `sforumSticker` atom node; do not overload `sforumEmoji` or generic `image` |
| Editor runtime | Tiptap 3 and `SFEditor` | Preserve a small insert command/catalog boundary; do not freeze the new editor UI in this plan |
| Plugin lifecycle | Manifest V3 lifecycle and registry publication | Publish/remove exact sticker catalog contributions through the same fenced lifecycle |
| Exact artifacts | package digest, `packageFiles`, CLI digest/validate/test | Bind one generated catalog and verify every referenced image before import |
| Storage | attachment storage provider slot and media validation | Reuse the selected storage adapter through a sticker-specific service/purpose |
| Public media | Host-owned stable media routes | Serve immutable digest-addressed sticker bytes without depending on a live plugin process |
| Revisions | accepted `post_revisions` ledger | Preserve exact sticker asset revisions in historical source and preview |
| Admin | forum admin routes and RBAC | Add a focused pack manager, not another oversized settings tab |

The current hardcoded `sforumEmoji` node represents Unicode characters. It may
remain for compatibility, but it is not the storage or rendering contract for
image stickers.

## Domain Model

Use focused domain ownership rather than adding sticker responsibilities to
the legacy Forum or Extensions facades.

Conceptual records:

- **Sticker pack**: stable key, localized name, source type, source identity,
  order, active/hidden/archived state, and operator override state.
- **Sticker item**: stable key, pack key, localized label, current asset
  revision, order, state, and source identity.
- **Sticker asset revision**: content digest, storage provider reference,
  object key, detected MIME, byte size, pixel width/height, animation flag,
  and creation/source artifact identity.
- **Catalog revision**: monotonic revision/digest used for ETag caching and
  frontend invalidation.

Exact SQL names are chosen during M0 after inspecting the then-current schema.
Do not put structured sticker catalogs into `web_options` or extension settings.

### Stable Identity

- Operator packs/items use generated portable stable keys independent of
  database sequence IDs.
- Plugin packs use `extension id + pack id`.
- Plugin items use `extension id + pack id + item id`.
- Asset revisions use their verified content digest.
- Renaming an image file changes its convention-derived item ID unless the
  pack metadata provides an explicit stable ID override.

### Status Semantics

- `active`: discoverable and insertable.
- `hidden`: absent from ordinary picker discovery but still resolvable and
  renderable. This is an operator presentation override, not deletion.
- `archived`: not insertable into new content; retained for existing accepted
  content and revision history.
- A plugin disable/uninstall produces the effective equivalent of hidden or
  archived discovery state without deleting imported immutable asset bytes.

Local records may be edited and archived. Plugin records retain plugin-owned
metadata; operator changes are bounded override/suppression records so an
upgrade cannot silently erase operator choices.

## Plugin Authoring Contract

### Source Layout

Plugin authors maintain files by convention:

```text
stickers/
└── cats/
    ├── pack.json
    ├── wave.webp
    ├── laugh.gif
    └── angry.png
```

`pack.json` contains pack-level localization and optional filename metadata
overrides. It does not need a handwritten entry for every image:

```json
{
  "id": "cats",
  "name": {
    "zh-CN": "猫猫",
    "en-US": "Cats"
  },
  "labels": {
    "wave": {
      "zh-CN": "挥手"
    }
  }
}
```

Defaults:

- directory name becomes the pack ID;
- filename stem becomes the item ID;
- filename stem is the fallback label;
- optional metadata may override labels and stable IDs;
- only regular contained files are admitted; traversal, symlinks, duplicate
  normalized paths, and case-collision ambiguity fail validation.

### Generated Catalog

Extend the existing `extension digest --write` workflow to scan sticker roots
and generate `stickers/.catalog.json`. The generated file contains normalized
pack/item IDs, localized metadata, relative paths, SHA-256 digests, detected
MIME, byte size, pixel dimensions, and animation facts.

The root Manifest contains one constant-size logical `stickerCatalog`
reference to one exact `packageFiles` entry. The example name is the intended
contract; M0 must integrate it with the existing V3 schema without weakening
exact package identity:

```json
{
  "stickerCatalog": "demo.stickers.catalog",
  "packageFiles": [
    {
      "id": "demo.stickers.catalog",
      "kind": "asset",
      "path": "stickers/.catalog.json",
      "digest": "<catalog-sha256>"
    }
  ]
}
```

Individual images do not grow the root Manifest. They are exact-listed inside
the generated catalog and remain bound by the complete package digest. The Host
recomputes their digests and media facts; generated metadata is not trusted
without verification.

Do not use `includes.stickers` merely to move a large handwritten array, a
nested ZIP per pack, a remote image URL, or executable runtime registration.

### Lifecycle

- Static install validates and previews the catalog without executing plugin
  code or publishing stickers.
- Enable verifies the exact active artifact, imports admitted image bytes into
  Core-owned content-addressed storage, and publishes one deterministic catalog
  snapshot.
- Upgrade and rollback reconcile stable IDs while preserving every historical
  asset digest already accepted into content.
- Disable and uninstall remove picker discovery contributions and retain
  operator suppressions plus historical immutable bytes.
- Safe Mode publishes no plugin discovery contribution, but Core's historical
  sticker media route remains available.
- Catalog conflicts, source artifact identity, import errors, and operator
  suppressions must be inspectable.

No plugin process, Host API token, raw database access, remote fetch, or custom
frontend code is required for a sticker contribution.

## Storage And Media Admission

Use a sticker-specific upload/import service over the existing attachment
storage adapter and Media Registry. Do not make sticker images ordinary post
attachments owned by the posting user.

V1 admission defaults:

- formats: PNG, WebP, and GIF;
- no SVG;
- no remote URL ingestion;
- maximum encoded size: 2 MiB per image;
- maximum source dimensions: 512x512 pixels;
- declared MIME, extension, and detected content must agree;
- preserve animation only for admitted GIF/WebP input;
- compute digest from accepted bytes and deduplicate identical assets.

The public media route is Host-owned and immutable by asset digest. It must not
read through a running plugin or a mutable plugin filesystem path.

V1 deliberately retains bytes that have ever appeared in accepted content or
accepted revision history. Physical garbage collection for never-used or fully
redacted assets is deferred until a revision-aware retention proof exists.

## Content Storage And Rendering Contract

Add a distinct Host-owned Tiptap atom node named `sforumSticker`. Do not store
it as a generic image and do not reuse `sforumEmoji`.

Canonical accepted attributes conceptually contain:

```json
{
  "type": "sforumSticker",
  "attrs": {
    "id": "demo.cats.wave",
    "assetDigest": "<immutable-asset-digest>",
    "label": "挥手"
  }
}
```

The browser may request insertion by current item ID, but the server resolves
and writes the canonical asset digest and label snapshot. Client-supplied URL,
digest, dimensions, class, and style are not authoritative.

Storing the asset digest prevents an extension upgrade or operator replacement
from changing old posts in place. New insertions use the current asset revision;
old accepted source keeps the exact old digest.

The server renders a narrow element such as:

```html
<img
  class="sf-sticker"
  data-sticker-id="demo.cats.wave"
  src="/media/stickers/<asset-digest>.webp"
  alt="挥手"
  width="320"
  height="320"
  loading="lazy"
  decoding="async">
```

Only server-detected intrinsic width/height may be emitted. Sanitizer changes
must allow only the exact sticker attributes required by this contract.

Plain text, excerpts, accessibility, Markdown export, and search use a readable
label fallback such as `[贴纸: 挥手]`; they do not index image URLs or digests.

### Existing And Archived References

- New content may insert only an effectively active item.
- Hidden items remain valid for exact historical rendering and edits that
  preserve the accepted reference.
- Archived items cannot be newly introduced.
- Editing an old post may preserve an exact archived `id + assetDigest` pair
  already present in the loaded accepted revision.
- A crafted request cannot introduce a historical or foreign digest by naming
  it directly.
- Restore and historical preview use the same immutable resolution contract.

## Rendered Size Contract

All Core content surfaces must apply the same class-based bounds, including
topic body, comment body, editor write surface, preview, admin content preview,
revision preview, quotes, and any theme Host island rendering accepted HTML.

```css
.sf-prose .sf-sticker,
.sf-editor__content .sf-sticker {
  display: inline-block;
  width: auto;
  height: auto;
  max-width: min(128px, 35vw);
  max-height: 128px;
  object-fit: contain;
  vertical-align: middle;
}

@media (max-width: 640px) {
  .sf-prose .sf-sticker,
  .sf-editor__content .sf-sticker {
    max-width: 96px;
    max-height: 96px;
  }
}
```

Implementation may centralize the selectors in the shared prose/editor token
layer, but may not weaken these behavior bounds. A sticker must never create
horizontal page overflow, cover adjacent controls, or resize an editor toolbar.

The editor design discussion may decide whether a sticker-only paragraph has a
different presentation, but it may not exceed the accepted maximum without a
new product decision.

## API And Authorization

The exact paths and DTO names are frozen in M0 with OpenAPI, but the surface
must include:

- a cacheable public effective catalog read with revision/ETag;
- Host-owned immutable sticker media reads;
- admin pack/item list, create, upload, edit, reorder, hide, restore, and
  archive commands;
- plugin source/artifact and operator override inspection;
- stale-write protection for ordered/admin document mutations.

Add a distinct `forum.stickers.manage` permission because pack administration
is grantable independently from forum runtime settings and attachments. Admin
screens also require `admin.access`. API checks are authoritative.

The picker catalog contains no secrets or raw storage provider data. Public
responses expose only effective active items, localized labels, safe immutable
media URLs, intrinsic dimensions, and catalog revision.

## Admin Product Surface

Use a dedicated Forum -> Sticker Packs management route and focused domain
components. Do not add a large manager to the forum settings route shell.

The eventual page must support:

- pack create/rename/reorder/hide/archive for operator-owned packs;
- multi-image upload, item rename/reorder/move/hide/archive;
- source and effective-state inspection;
- bounded operator hide/suppress actions for plugin-owned packs/items;
- clear loading, empty, disabled, field-error, operation-error, and success
  states;
- success Toasts following active appearance tokens;
- explicit confirmation for destructive archive operations.

The concrete layout is deferred until the new editor discussion establishes
shared picker/catalog interaction patterns.

## Editor Integration Boundary - Design Pending

The platform must eventually provide the new editor with:

- an effective sticker catalog query;
- a stable insert command accepting a current sticker item identity;
- a Host-owned `sforumSticker` node extension;
- loading, unavailable, empty, and denied results;
- immutable preview URLs and intrinsic media facts;
- preservation of canonical native JSON during edit/restore.

The following are intentionally **not decided** here:

- toolbar placement and icon grouping;
- popover, panel, or mobile drawer geometry;
- pack tabs, search, pagination, and virtualized grids;
- recent stickers, favorites, keyboard navigation, and shortcuts;
- inline versus sticker-only paragraph interaction;
- whether the new editor keeps the current multi-mode inspection UI;
- how plugin L2 editor commands coexist visually with Core commands.

Resolve the remaining picker questions during M0 before M4 implementation.

## Milestones

### D0 - Architecture And Memory

Status: **complete**

- Record this task book, decision, module notes, and hot handoff.
- Freeze the generated-catalog and immutable-rendering architecture.
- Do not implement product code.

### D1 - New Editor Product Design

Status: **in progress** - production base complete; sticker picker details remain

- [x] Inspect all current topic/comment/admin editor contexts.
- [x] Implement and verify the Forum Canvas base editor geometry and states.
- [x] Preserve editor-document, Markdown/native inspection, and trusted L2
  extension loading contracts.
- Freeze sticker picker behavior without changing the platform contracts above.
- [x] Add rendered desktop/mobile evidence at `1920x960` and `390x844`.

### M0 - Contract And Threat Model

- Freeze tables, DTOs, OpenAPI paths, node schema, sanitizer attributes,
  permission, media limits, lifecycle matrix, and generated catalog schema.
- Decide exact Manifest V3 schema placement for the constant-size catalog
  reference and update the extension impact/trust presentation.
- Add contract tests before production implementation.

### M1 - Catalog, Storage, And Admin API

- Implement focused sticker domain/store/service packages.
- Add migration, permission seed/catalog text, upload/import validation,
  immutable media route, catalog revision/ETag, and admin commands.
- Test allowed and denied paths, stale writes, hide/archive semantics, unsafe
  media, and storage failures.

### M2 - Accepted Content And Rendering

- Add `sforumSticker` to Tiptap and `EditorDocument` Core schema.
- Add server canonicalization, exact digest preservation, sanitizer policy,
  Markdown/plain/search fallback, and shared size styles.
- Cover create, edit, restore, revision preview, disabled item preservation,
  malicious attributes, overflow, and mobile rendering.

### M3 - Plugin Catalog And Lifecycle

- Extend digest/validate/test CLI behavior and Manifest schema.
- Validate generated catalogs and import exact media without plugin execution.
- Publish through fenced lifecycle; cover enable, upgrade, rollback, disable,
  uninstall, restart, multi-node reconciliation, and Safe Mode.
- Add catalog/registry inspector visibility and Extension Surface Matrix updates.

### M4 - Admin And Editor UI

- Implement the approved D1 editor picker and the focused admin manager.
- Add localization, accessible controls, Toasts, empty/loading/error states,
  and responsive behavior.
- Preserve existing topic/comment create/edit submission contracts.

### M5 - Release Gate

- Run focused Go and Bun tests, OpenAPI reference validation, architecture
  boundary validation, Nuxt typecheck/build, and full repository test gate.
- Run authenticated Browser QA for admin management and topic/comment
  create/edit at desktop and `390x844` mobile.
- Verify no horizontal overflow, no duplicate chrome, exact theme provider,
  immutable historical rendering, plugin lifecycle, and API denied paths.
- Update module notes, plan status, handoff, docs, and generated extension
  catalogs.

## Definition Of Done

- Operators can create, upload, reorder, hide, restore, and archive local image
  sticker packs through the admin UI.
- Plugins contribute directory-based packs with one generated catalog reference
  and no handwritten root Manifest item list.
- Static install never executes plugin code; lifecycle publication remains
  exact-artifact, inspectable, reversible, and Safe Mode aware.
- Topic/comment create, edit, restore, revision preview, and public render keep
  exact immutable sticker bytes.
- Hiding, archive, plugin disable, upgrade, rollback, and uninstall do not break
  accepted or historical content.
- Client-provided URLs, dimensions, styles, SVG, remote media, unsafe paths,
  MIME confusion, and unknown digests fail closed.
- Every rendered content surface enforces `128x128` desktop/tablet and `96x96`
  mobile maximums without horizontal overflow.
- New permission paths have allowed and denied API tests.
- OpenAPI, extension author docs/catalogs, Forum Extension Surface Matrix,
  knowledge modules, and handoff memory are synchronized.

## Deferred

- Unicode emoji picker or replacement of the current `sforumEmoji` behavior.
- Reactions, likes, and per-post reaction counts.
- User-uploaded personal sticker packs.
- Favorites, recents, recommendations, and marketplace discovery.
- Remote/CDN-backed plugin sticker catalogs.
- Operator resizing, rotation, cropping, or arbitrary per-node dimensions.
- Nested sticker pack archives.
- Physical deletion of assets referenced by accepted revision history.
