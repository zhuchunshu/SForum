# Brand assets

Every extension that ships with the product — built-in plugins, built-in themes,
optional packages, and independent plugin repositories — carries its own brand
identity: a square icon and a 1280×640 social card. This page is the contract.
The generator that produces them lives in [`scripts/brand/`](../../scripts/brand/README.md).

Brand assets are not runtime artifacts: they are not declared in `packageFiles`,
they never load in a browser through the asset registry, and no route serves
them. They exist so a plugin looks like a product on its repository page, in the
docs, and in a future extension catalog.

## Where the files live

```text
extensions/builtin/plugins/<package>/
  sforum.extension.json
  backend/  frontend/  manifest/  schemas/
  brand/
    icon.svg
    icon.png
    social-preview.png
```

Independent plugin repositories use the same layout inside their package
directory, for example `plugins/sforum.fetch-url/brand/`. The main repository
keeps its own card at `docs/assets/brand/social-preview.png`.

## The contract

| File | Size | Format | Budget | Notes |
| --- | --- | --- | --- | --- |
| `brand/icon.svg` | `viewBox="0 0 512 512"` | Self-contained SVG | 40 KiB | Editable source. No `<script>`, `<image>`, `<foreignObject>`, no external `href`. |
| `brand/icon.png` | 512×512 | PNG, RGBA | 120 KiB | Master icon. Transparent outside the rounded square. |
| `brand/social-preview.png` | 1280×640 | PNG | 600 KiB | Chinese primary card. GitHub allows 1 MB; the contract keeps headroom. |
| `brand/social-preview.en-US.png` | 1280×640 | PNG | 600 KiB | Optional English card. Additive: the check ignores extra files. |

Invariants:

- File names are fixed. Do not prefix them with the extension id — the package
  directory already carries the identity, and a fixed name keeps the check and
  every README reference stable.
- `brand/` stays out of `packageFiles`. The kind list has no brand kind on
  purpose: brand bytes are inert, and `ValidatePackageFiles` only verifies
  declared files, so extra files are legal (a package `README.md` already is).
- The package tree digest still covers `brand/` (`DigestTree` hashes every file
  except `.git`, `node_modules`, `vendor`). Adding or changing brand assets is
  therefore a new artifact for the same version — see
  [Digest consequences](#digest-consequences).
- `extensions/dev/` (gitignored scaffolds) and `extensions/fixtures/` (test
  inputs) are exempt. They are not shipped packages.

## Visual template

One template, one accent. The identity is the SForum mark's own language: the
blue gradient `#2671ed → #1f6bec`, rounded stroke caps, deep navy ground. Do not
introduce per-plugin hues; the glyph carries the meaning, the palette carries
the family.

### Icon (512×512)

| Element | Value |
| --- | --- |
| Ground | Rounded square, `rx=112`, vertical gradient `#101b2d → #0b1220` |
| Highlight | Radial brand glow, center of the tile, `#2671ed` at 22% → 0 |
| Edge | 1px hairline `rgba(255,255,255,0.08)` inset by 0.5px |
| Glyph | 256×256 box at `(128,128)`, Tabler outline scaled ×10.667 (native 2-unit stroke becomes ≈21px) |
| Glyph stroke | `url(#brand)` gradient, round caps and joins, `fill="none"` |

### Social card (1280×640)

| Element | Value |
| --- | --- |
| Ground | `#0b1220` |
| Corner wash | Linear gradient `#2671ed` 0 → 14%, running to the bottom-right |
| Inner border | Rect at `(24.5,24.5)` 1231×591, `rx=20`, 1px `rgba(255,255,255,0.07)` |
| Divider | Vertical 1px `rgba(255,255,255,0.06)` at `x=792`, `y=150…500` |
| Eyebrow | `(88,176)`, 14px, `#6b7d94`: `SFORUM` with 3px tracking, then `· 内置插件` / `· 扩展插件` / `· 主题` / `· 框架` |
| Title | `(88,282)`, 64px/700, `#f4f8fd` — the Chinese name |
| Manifest name | `(88,326)`, 20px, `#8fa0b8` — the English `name` from the manifest |
| Tagline | `(88,396)`, 26px, `#b9c7d8` — one line, at most 24 Chinese characters |
| Meta | `(88,446)`, mono 16px: `id · v<version>` (omitted when there is no version) |
| Art tile | 320×320 at `(856,160)`, `rx=72`, same tile fill and hairline, 260px halo behind it |
| Art | 200×200 glyph at `(916,220)`; the framework card uses the SForum mark at 230px instead |
| Footer | Logo mark 28px at `(88,556)`, `SForum` 19px/600 at `(128,577)`, repository slug mono 14px right-aligned at `x=1192` |

Fonts: `Noto Sans CJK SC` (fallbacks `Noto Sans CJK`, `Noto Sans`, `DejaVu
Sans`), monospace `DejaVu Sans Mono`. Both are present on the development
machine; the generator renders on the host, so the shipped PNG is the source of
truth rather than a font the browser must have.

Glyphs come from `@iconify-json/tabler` (already a dependency of `apps/web`).
Never hand-draw a glyph, and never use an emoji.

## Generate

```bash
node scripts/brand/generate.mjs --help          # flags at a glance

# Preview into a scratch directory plus contact sheets; nothing in the repo changes
node scripts/brand/generate.mjs --draft /tmp/sforum-brand --sheet /tmp/sforum-brand

# Regenerate one package (matched by package-directory suffix)
node scripts/brand/generate.mjs --only sforum-smtp

# Write every package and the repository card
node scripts/brand/generate.mjs
```

`--draft` writes `<dir>/<extension id>/{icon.png,social-preview.png}` and the
repository card to `<dir>/repo/social-preview.png`; `--sheet` additionally emits
`sheet-cards.png` and `sheet-icons.png` for review.

`spec.json` holds the only copy of the Chinese title, the tagline, the glyph and
the repository slug; `id`, `name` and `version` are read from each package's
`sforum.extension.json`. Output is deterministic: two runs over the same inputs
produce byte-identical files.

Generation needs `sharp`, resolved from `apps/web/node_modules/sharp`, then
`NODE_PATH`, then the global Bun install. Generation is a maintainer task and is
**not** part of CI — the pipeline only validates committed PNGs, so build images
need no native image dependency.

## Validate

```bash
cd apps/api
go run ./cmd/sforum extension brand check ../../extensions/builtin/plugins/sforum-smtp
go run ./cmd/sforum extension brand check --root ../../extensions/builtin
```

The check verifies presence, byte budgets, PNG magic and exact IHDR dimensions,
and that the SVG parses, uses the right `viewBox`, and references nothing
outside itself. Stable error codes:

| Code | Meaning |
| --- | --- |
| `brand.missing_icon` | `brand/icon.svg` is absent |
| `brand.missing_icon_png` | `brand/icon.png` is absent |
| `brand.missing_social` | `brand/social-preview.png` is absent |
| `brand.too_large` | A file exceeds its byte budget |
| `brand.icon_dimensions` | `icon.png` is not 512×512 |
| `brand.social_dimensions` | `social-preview.png` is not 1280×640 |
| `brand.not_png` | The file is not a valid PNG |
| `brand.svg_invalid` | Not XML, wrong root, or wrong `viewBox` |
| `brand.svg_external_ref` | The SVG embeds or links an external resource |

Inside this repository the same rules run as a Go test over
`extensions/builtin/**` and `extensions/optional/**`, so the api gate fails
instead of a release. Independent repositories call the CLI from their own
`scripts/prepare-package.sh`.

## Reference the assets

A package `README.md` shows the card right below the title (and below the badge
block in an independent repository), so the repository page opens on the
identity instead of a wall of prose:

```md
# <manifest name>

![<Chinese name> 社交卡](./brand/social-preview.png)

<one-line pitch>
```

`icon.png` is what a catalog, the docs, or a future admin surface shows at small
sizes; the README does not need to repeat it, because the card already carries
the mark at full size.

Markdown image targets are validated as local links by
`tests/validate-docs.mjs`, so the files must be committed before the README
references them.

## Repository social preview

GitHub does not read a file for the repository social preview; it has no public
API for it either. Upload `brand/social-preview.png` (or, for the main
repository, `docs/assets/brand/social-preview.png`) once per repository under
**Settings → Social preview**. GitHub renders 1280×640 PNG/JPG/GIF under 1 MB
best; the contract already matches that.

## Digest consequences

Because `DigestTree` covers the whole package tree, adding `brand/` to a package
that already shipped produces a new artifact digest for the same version. Built-in
auto-update deliberately skips same-version digest drift, so the admin sees a
staged candidate that needs one explicit upgrade — or the change ships with a
patch version bump, which auto-update promotes normally. Do not bump versions
only to silence the drift; treat it as a normal artifact change.

## Adding an extension

1. Add an entry to `scripts/brand/spec.json` (kind, Chinese title, tagline,
   glyph, repository slug).
2. Run the generator with `--draft` and `--sheet`, and look at the contact
   sheets: hierarchy, no overflow, legible at 128px.
3. Generate for real, then `sforum extension brand check <package>`.
4. Add the card and icon to the package `README.md`.
5. Upload the card as the repository social preview if the package lives in its
   own repository.
