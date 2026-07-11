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

**Status:** complete.

### Goal

Introduce one load path that:

1. Reads `sforum.extension.json`
2. Resolves `includes` (if present)
3. Merges into `Manifest`
4. `Normalize` + `Validate`

Packages without `includes` behave exactly as today.

### Tasks

- [x] `LoadPackage` / `LoadPackageFS` / `LoadRootBytes` in
  `apps/api/app/Support/ExtensionManifest/load.go`
- [x] Path safety via `SafeArchivePath`
- [x] Merge keys: langs, settings, contributions, admin, frontend, events,
  routes, jobs, migrations, permissions, hooks, providers, adminPages
- [x] Dual-source detection
- [x] langs directory / list / single-file map
- [x] settings/contributions directory shards (Phase 4 early)
- [x] Wire builtin sync + ZIP `readArchive` + snapshot load
- [x] Unit tests

### Exit criteria

- [x] ExtensionManifest / ExtensionPackage / Extensions tests pass

---

## Phase 2 — Migrate `sforum.smtp` as reference package

**Status:** complete.

### Tasks

- [x] Split SMTP into thin root + `manifest/langs/`, settings, contributions,
  admin, frontend
- [x] Update `smtp_manifest_test.go` and web ownership tests
- [x] Knowledge module notes updated

---

## Phase 3 — Developer experience

**Status:** complete.

### Tasks

- [x] `make:plugin --complex` multi-file scaffold
- [x] `extension validate [path]` (+ `--json`)
- [x] Module note documents CLI + complex scaffold

---

## Phase 4 — Shard directories and polish

**Status:** complete (core pieces).

- [x] Settings shard directory (`manifest/settings/*.json`) with unique key
  merge — implemented in `load.go`; used by complex scaffold
- [x] Contributions shard directory — same list loader as settings
- [ ] Settings presentation copy extracted to per-locale files (deferred;
  not needed yet)
- [x] Merged manifest JSON via `extension validate --json`
- [x] SMTP is the multi-file reference; other packages stay single-file

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
