# Extension Manifest Split (Core Identity + Includes)

## Status

Accepted and implemented authoring foundation; expanded by Manifest V3.

The single entrypoint plus deterministic include merge remains authoritative.
`2026-07-13-trusted-plugin-theme-platform-v3.md` expands the shard catalog and
contract schemas in P2; removed Web Release/Nuxt Layer fields in historical
examples are not restored.

## Context

Complex plugins already push a lot of developer-facing registration into a
single root `sforum.extension.json`. The protected `sforum.smtp` package is the
first real example: identity, backend entry, providers, admin entry, frontend
admin component maps, contributions, and multi-locale settings copy all live in
one file. As more verticals add routes, events, jobs, and contributions, a
monolithic manifest becomes hard to review, scaffold, and maintain.

SForum already modularized OpenAPI the same way: an entrypoint index plus
focused partial files, merged and validated as one contract. Extension packages
need an analogous authoring model without weakening install-time review or the
single runtime `Manifest` type.

Identity localization (`langs`) is part of the same problem. Root-level
`langs` mixes packaging identity with translation maintenance, and it does not
scale when an extension supports many locales. Frontend admin locales already
live under `frontend/admin/locales/`; identity langs should follow a
directory-per-locale pattern without merging those three i18n layers into one
bag of strings.

## Decision

### 1. Single entry, multi-file authoring

- `sforum.extension.json` remains the **only package entrypoint**. ZIP upload,
  builtin sync, verify, enable, and CLI validation all start from this file.
- Complex packages may declare an optional `includes` object that points at
  partial manifest files under the package root.
- Host code loads the entry file, resolves includes, **merges into one
  `Manifest`**, then runs existing `Normalize` / `Validate`. Downstream storage,
  admin review UI, OpenAPI responses, and runtime consumers keep a single
  merged model. They do not need to understand on-disk layout.

### 2. What stays in the root file

Root `sforum.extension.json` should hold:

| Category | Fields |
|----------|--------|
| Identity defaults | `id`, `name`, `description`, `url`, `author`, `version`, `type`, `sforumVersion` |
| High-risk runtime boundary | `backend`, `providers`, `permissions`, `migrations` (path list) |
| Include index | `includes` (optional) |

Optional short declarations may remain inline when small (for example a
minimal theme `frontend.layer`, or a short `admin.entry`). Complex plugins
should prefer includes for bulky capability blocks.

### 3. What moves out via `includes`

First-class include keys:

| Key | Typical path | Content |
|-----|--------------|---------|
| `langs` | `manifest/langs/` | Identity locale overrides (see below) |
| `settings` | `manifest/settings.json` | Settings schema + presentation copy |
| `contributions` | `manifest/contributions.json` | Ordered contribution items |
| `admin` | `manifest/admin.json` | `admin.entry` + `admin.pages` |
| `frontend` | `manifest/frontend.json` | Layer / admin frontend maps |
| `events` | `manifest/events.json` | Event declarations |
| `routes` | `manifest/routes.json` | Controlled plugin routes |
| `jobs` | `manifest/jobs.json` | Job name declarations |

Simple plugins and most themes may omit `includes` entirely and keep today's
single-file layout. That remains fully supported.

### 4. `langs`: directory-per-locale is the preferred complex form

Identity localization is **not** settings copy and **not** Vue UI copy.

Three i18n layers stay separate:

1. **Identity** — root defaults + `includes.langs` → `Manifest.Langs` /
   `LocalizedDisplay` (list, install review, package name/description).
2. **Declarative settings / contribution labels** — `LocalizedText` inside
   settings/contributions partials (host-rendered forms and contribution
   labels).
3. **Frontend component UI** — `frontend.admin.locales` / Vue i18n JSON under
   `frontend/admin/locales/` (trusted admin components).

Do **not** create one shared `i18n/zh-CN.json` that mixes all three.

#### `includes.langs` shapes (all supported)

**A. Directory (recommended for multi-locale plugins)**

```json
"includes": {
  "langs": "manifest/langs"
}
```

Layout:

```text
manifest/langs/
  zh-CN.json
  en-US.json
```

Each `*.json` file:

- Filename (without `.json`) is the locale key (`zh-CN`, `en-US`, or short
  `zh` / `en`).
- File body is a single `ManifestLocale` object (`name`, `description`, `url`,
  `author`), **not** wrapped as `{ "zh-CN": { ... } }`.
- Fields remain optional; present fields override root identity defaults.
- Directory entries are read in locale-key sort order for stable digests/logs.
- Only `*.json` files are allowed; other files in the directory fail validation.
- An empty directory (zero JSON files) fails validation.

**B. Explicit file list**

```json
"includes": {
  "langs": [
    "manifest/langs/zh-CN.json",
    "manifest/langs/en-US.json"
  ]
}
```

Locale still comes from each filename. Useful when a directory holds drafts
that must not ship.

**C. Single map file (small packages)**

```json
"includes": {
  "langs": "manifest/langs.json"
}
```

Body is today's root `langs` map:

```json
{
  "zh-CN": { "name": "...", "description": "..." }
}
```

Resolution rule: if the include path is a directory → A; if it is a file → C;
if the value is a string array → B.

#### Locale fallback

Keep existing `lookupManifestLocale` behavior after merge:

1. exact locale (`zh-CN`)
2. language prefix (`zh`)
3. root identity defaults (`name` / `description` / …)

New packages should prefer BCP 47-style files (`zh-CN`, `en-US`) to align with
site i18n and frontend admin locales. Short keys (`zh`) remain valid for
broad coverage.

Root identity defaults (`name`, `description`, `author`, `url`) **always stay
in the entry file**. `langs` only carries overrides. Packages with no
translations remain valid.

### 5. Merge and conflict rules

- **No dual source:** if the root file already defines a non-empty block for a
  key (for example root `langs` or root `settings`) and `includes` also
  provides that key, validation fails with `extension.manifest_invalid`.
- **Includes only fill missing blocks.** Root wins by requiring authors to
  choose one source, not by silent override.
- Include paths must pass existing `SafeArchivePath` rules (no `..`, no
  absolute paths, stay inside package root).
- Missing include path, invalid JSON, illegal locale filename, duplicate
  locale after merge, or duplicate setting keys across setting shards → fail
  fast.
- After merge, existing validators (`validateManifestLangs`, settings,
  contributions, theme capability bans, etc.) run unchanged on the combined
  `Manifest`.

Future optional enhancement (not required for v1 of this work): `settings` /
`contributions` may accept a directory of shards with ordered filename merge
and unique key checks. Identity `langs` directory support is in scope for v1.

### 6. Authoring ergonomics and tooling

- Install-time and API surfaces continue to expose the **merged** manifest (or
  today's summaries). Operators do not need to browse include files.
- `make:plugin` should offer a complex scaffold that generates
  `manifest/langs/{zh-CN,en-US}.json` plus empty/example partials and an
  `includes` index. Simple scaffold stays single-file.
- CLI should grow `extension validate` (or equivalent) that loads via the
  package loader and prints merged locale keys / include resolution for
  debugging.
- Publishing does **not** require inlining includes into one file. Source
  layout is the install layout; the host always merges at load time.

### 7. Explicit non-goals (this decision)

- Generating manifests from executable code (JS/Go) at package build time as
  the primary authoring model — packages must stay declaratively reviewable.
- Multiple peer entry files without `sforum.extension.json`.
- Replacing manifest declarations with runtime-only registration for
  permissions, routes, or providers.
- Forcing simple themes into multi-file layout.
- Merging identity langs, settings `LocalizedText`, and Vue locales into one
  tree.
- Splitting settings schema from settings presentation copy in the first
  implementation wave (may be a later decision if settings files grow too
  large).

## Consequences

- Complex extension packages become maintainable without changing the runtime
  security model: enable-time review still sees one validated capability set.
- `sforum.smtp` and future provider plugins can move bulk copy and contributions
  out of the entry file; identity translations live as one file per locale.
- Implementation must introduce a single `LoadPackage(root)` (name flexible)
  path used by upload extract, builtin sync, verify, CLI, and tests. Call sites
  that only `json.Unmarshal` the root file would silently drop includes.
- Authors must learn `includes` and the three i18n layers; docs and scaffolds
  must make the simple path obvious.
- Digest/hash of “installed package files” remains whole-tree based; merged
  manifest content is derived, not a second source of truth on disk.

## Implementation plan reference

Phased tasks live in:

`docs/superpowers/plans/2026-07-12-extension-manifest-split.md`
