# Extension Manifest Split Implementation Plan

> **For agentic workers:** Implement task-by-task. Prefer small PRs that keep
> single-file manifests working at every step. Decision record:
> `knowledge/decisions/2026-07-12-extension-manifest-split.md`.

**Goal:** Let complex plugins/themes keep a thin `sforum.extension.json` for
identity and high-risk runtime boundaries, while bulky declarations (especially
per-locale identity `langs`, settings, contributions, admin, …) live in
include files under `manifest/`. Host always merges to one `Manifest` before
validation and runtime use.

**Architecture:** Authoring-time multi-file layout; load-time merge; single
validated runtime model. No change to plugin subprocess boundaries, provider
slots, or install-review product loop.

**Tech stack:** Go `app/Support/ExtensionManifest`, extension package install /
builtin sync paths, `cmd/sforum` CLI, modular OpenAPI only if public shapes
change (expected: no public shape change), Nuxt unchanged for v1.

---

## Target authoring layout (complex plugin)

```text
my-plugin/
  sforum.extension.json          # identity defaults + backend/providers + includes
  manifest/
    langs/
      zh-CN.json                 # ManifestLocale body; filename = locale
      en-US.json
    settings.json
    contributions.json
    admin.json                   # optional if small enough to keep inline
    frontend.json                # optional
  backend/
  frontend/admin/
    locales/                     # Vue UI only — not identity langs
      zh-CN.json
      en-US.json
```

### Root entry example

```json
{
  "id": "sforum.smtp",
  "name": "SForum SMTP",
  "description": "Protected built-in SMTP mail delivery provider.",
  "url": "https://github.com/zhuchunshu/sforum",
  "author": { "name": "SForum", "url": "https://github.com/zhuchunshu/sforum" },
  "version": "1.0.0",
  "type": "plugin",
  "sforumVersion": ">=0.1.0",
  "backend": {
    "entry": "backend/plugin",
    "rpc": "hashicorp-go-plugin",
    "protocolVersion": 1
  },
  "providers": [{ "slot": "mail.provider", "label": "SMTP", "timeoutMs": 15000 }],
  "includes": {
    "langs": "manifest/langs",
    "settings": "manifest/settings.json",
    "contributions": "manifest/contributions.json",
    "admin": "manifest/admin.json",
    "frontend": "manifest/frontend.json"
  }
}
```

### `manifest/langs/zh-CN.json`

```json
{
  "name": "SForum SMTP",
  "description": "受保护的内置 SMTP 邮件发送提供方。",
  "author": { "name": "SForum" }
}
```

### `includes.langs` resolution

| Value | Meaning |
|-------|---------|
| `"manifest/langs"` or `"manifest/langs/"` | Directory: each `*.json` → locale key from basename |
| `["manifest/langs/zh-CN.json", "..."]` | Explicit list; locale from basename |
| `"manifest/langs.json"` | Single file: `map[string]ManifestLocale` |

---

## Non-goals (this plan)

- [ ] Settings schema vs presentation i18n split (later if needed)
- [ ] Unified package-wide `i18n/` mixing identity + settings + Vue
- [ ] Code-generated manifests as primary authoring
- [ ] Forcing themes into multi-file layout
- [ ] Changing merged API response shapes for admin list/detail

---

## Phase 0 — Documentation lock-in

**Status:** complete (documentation only; no runtime code).

- [x] Decision: `knowledge/decisions/2026-07-12-extension-manifest-split.md`
- [x] Plan: this file
- [x] Module note section in `knowledge/modules/extensions.md`
- [x] Index pointer in `knowledge/index.md`
- [x] Session handoff for next implementer:
  `knowledge/sessions/2026-07-12-extension-manifest-split-plan.md`

---

## Phase 1 — Package loader (backward compatible)

### Goal

Introduce one load path that:

1. Reads `sforum.extension.json`
2. Resolves `includes` (if present)
3. Merges into `Manifest`
4. `Normalize` + `Validate`

Packages without `includes` behave exactly as today.

### Tasks

- [ ] Add `Includes` field to a root DTO or parse `includes` before full
  struct fill (prefer a thin load DTO so `includes` is not part of the runtime
  `Manifest` stored/returned unless useful for debug).
- [ ] Implement `LoadPackage(root string) (Manifest, error)` (name flexible) in
  `apps/api/app/Support/ExtensionManifest` (or adjacent package if circular
  deps appear).
- [ ] Path safety: reuse `SafeArchivePath`; join under `root`; reject escape.
- [ ] Merge keys (v1): `langs`, `settings`, `contributions`, `admin`,
  `frontend`, `events`, `routes`, `jobs` (implement all keys in the loader even
  if sample migration only uses a subset).
- [ ] Dual-source detection: root block present **and** include for same key →
  error.
- [ ] `langs` directory / list / single-file map resolution as specified.
- [ ] Directory rules: only `*.json`; sort by locale key; empty dir fails;
  illegal locale filename fails; duplicate locale fails.
- [ ] Wire **all** production load sites to `LoadPackage`:
  - builtin extension sync
  - ZIP install / extract validation
  - verify / enable preflight that re-reads installed manifest
  - any CLI that validates packages
  - tests/fixtures helpers
- [ ] Unit tests in `ExtensionManifest`:
  - no includes (golden parity with current Validate)
  - langs directory merge + `LocalizedDisplay` zh-CN / en-US / fallback
  - langs explicit list
  - langs single map file
  - dual-source langs fail
  - dual-source settings fail
  - path traversal in includes fail
  - empty langs dir fail
  - non-json file in langs dir fail
  - settings + contributions include merge smoke test

### Exit criteria

- [ ] `go test ./app/Support/ExtensionManifest/...` (and any touched extension
  packages) pass
- [ ] Existing builtin single-file manifests still load and sync
- [ ] No OpenAPI change required unless loader errors need a new reason code
  (prefer reuse `extension.manifest_invalid`)

---

## Phase 2 — Migrate `sforum.smtp` as reference package

### Goal

Prove the complex layout on a real built-in plugin.

### Tasks

- [ ] Split
  `extensions/builtin/plugins/sforum-smtp/sforum.extension.json` into:
  - thin root entry + `includes`
  - `manifest/langs/zh-CN.json` (and `en-US.json` if useful)
  - `manifest/settings.json`
  - `manifest/contributions.json`
  - `manifest/admin.json` and/or `manifest/frontend.json` as needed
- [ ] Keep runtime behavior identical: settings keys, defaults, contribution
  point ids, component ids, provider slot.
- [ ] Update tests that read the SMTP manifest path
  (`smtp_manifest_test.go`, web ownership tests, fixtures).
- [ ] Manual / scripted check: builtin sync still registers `sforum.smtp`;
  settings page contribution still resolves.

### Exit criteria

- [ ] SMTP package entry file is short (identity + boundary + includes)
- [ ] Focused backend + relevant validate scripts pass
- [ ] Knowledge module notes updated with “SMTP uses includes” example

---

## Phase 3 — Developer experience

### Goal

Make the multi-file layout easy to create and debug.

### Tasks

- [ ] `make:plugin` optional complex scaffold (flag or prompt):
  - `manifest/langs/zh-CN.json`, `en-US.json`
  - stub `settings.json` / `contributions.json`
  - root `includes` pointing at them
- [ ] `make:theme` remains single-file by default; optional langs dir only if
  themes need identity translations
- [ ] CLI `extension validate <path>` (or fold into existing command):
  - load via `LoadPackage`
  - print type/id/version
  - print resolved include keys
  - print merged locale keys
  - exit non-zero on invalid
- [ ] Short author doc section (module note or `docs/`): simple vs complex
  package; three i18n layers; langs directory rules

### Exit criteria

- [ ] Scaffold produces a package that `extension validate` accepts
- [ ] Docs describe directory-per-locale langs without mixing Vue locales

---

## Phase 4 — Optional follow-ups (out of first implementation PR)

- [ ] Settings shard directory (`manifest/settings/*.json`) with unique key
  merge
- [ ] Contributions shard directory
- [ ] Settings presentation copy extracted to per-locale files (only if
  settings JSON remains painful after Phase 2)
- [ ] Pack command that prints merged manifest JSON for marketplace review
- [ ] Migrate other builtins/dev samples only when they become complex

---

## Conflict matrix (implementer cheat sheet)

| Situation | Result |
|-----------|--------|
| No `includes` | Current behavior |
| `includes.langs` only | Merge into `Manifest.Langs` |
| Root `langs` + `includes.langs` | Invalid |
| Root `settings` + `includes.settings` | Invalid |
| `includes.langs` → directory with `zh-CN.json` | `Langs["zh-CN"]` |
| `includes.langs` → file map | Unmarshal map |
| `includes.langs` → string array of files | Merge each file by basename |
| Langs dir empty | Invalid |
| Langs dir has `readme.md` | Invalid |
| Include path `../x` | Invalid |
| Missing include file | Invalid |
| Duplicate locale files after normalize | Invalid |
| Theme declares plugin-only includes content | Still invalid after merge via existing theme rules |

---

## Suggested PR breakdown

1. **Loader + tests only** (Phase 1) — no package moves
2. **SMTP migration** (Phase 2)
3. **CLI + scaffold + docs polish** (Phase 3)

Do not combine loader rewrite with large scaffold refactors in one PR.

---

## Verification checklist (before calling the feature done)

- [ ] Single-file default theme still validates and syncs
- [ ] Single-file signal-garden (or other dev theme) still validates
- [ ] SMTP multi-file package validates; settings keys unchanged
- [ ] `LocalizedDisplay` for SMTP zh-CN still returns Chinese name/description
- [ ] Upload of a multi-file ZIP with includes works end-to-end in admin
- [ ] Malicious include path rejected
- [ ] Dual-source rejected
- [ ] `go test` for ExtensionManifest + Extensions model paths
- [ ] Relevant `tests/validate-*.js` if they parse manifests on disk

---

## Open questions for implementers (defaults if unanswered)

1. **Should `includes` appear in API-stored raw manifest JSON?**  
   Default: store/return merged capability view as today; do not require clients
   to resolve includes. Optionally keep original root bytes on disk only.

2. **Digest calculation**  
   Default: continue hashing package files as installed on disk (includes
   remain separate files). Do not hash only merged JSON.

3. **Root empty arrays vs omitted keys for dual-source**  
   Default: treat omitted / null / empty slice-or-map as “not defined” so an
   empty `"settings": []` does not block `includes.settings`. Document this in
   code comments.
