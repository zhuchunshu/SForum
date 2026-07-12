# Runtime Page Registry & Simple Themes — Implementation Plan

Status: **implemented (P0–P5)**  
Date: 2026-07-13  


## Implementation status

- [x] P0 inventory + ADR anchors
- [x] P1 Page catalog + SFPageOutlet + admin Pages
- [x] P2 L0 skin activate without rebuild
- [x] P3 L1 templates + add/replace + approval
- [x] P4 L2 prebuilt widget loader (`SFExtensionWidget`)
- [x] P5 decouple Web Release; retire public theme Layer activation path

See session handoff `knowledge/sessions/2026-07-13-runtime-page-registry-p0-p5.md`.

Audience: humans and AI sessions starting a greenfield implementation track

**Decision (accepted):**  
`knowledge/decisions/2026-07-13-runtime-page-registry-themes.md`

**Product goal:**  
WordPress/Typecho-like theme simplicity + Flarum-like author-prebuilt assets +
SForum security contracts. Operators do **not** rebuild the site to activate
normal themes. Themes and plugins can **add and replace view pages**.

---

## How To Use This Plan In A New Session

1. Read the ADR above and this plan.
2. Read `knowledge/modules/extensions.md` + `knowledge/modules/frontend.md`.
3. Pick the **next open phase** (`P0` … `P5`); do not skip safety gates.
4. Follow **Commit Discipline** on every phase.
5. After each phase (or logical sub-task), update this plan’s checklist and add
   a short handoff under `knowledge/sessions/`.

---

## Commit Discipline (mandatory)

Use small, reversible commits. Prefer **one concern per commit**. Do not squash
away intermediate safety commits during this program unless the user asks.

### Rules

1. **Commit early, commit often** — after each green sub-task, not only at
   phase end.
2. **Never mix** “new registry code” with “delete old theme pipeline” in the
   same commit.
3. **Never mix** “feature” with “mass reformat / unrelated refactor”.
4. Each commit message: **why**, not only what (complete sentences).
5. Before destructive steps (deleting Layer activation, removing supervisor
   theme switch), create an explicit commit titled like  
   `chore(themes): mark layer activation deprecated`  
   so `git revert` / `git reset` has a clear boundary.
6. Prefer **additive** commits first; **deletions** only in P5 after exit
   criteria.
7. If a phase fails tests, fix forward with small commits; if the approach is
   wrong, `git revert` the phase’s commits rather than rewriting history on
   `main` unless the user requests rewrite.
8. Do **not** force-push shared branches. Do **not** amend published commits
   without user approval.
9. Network install commands (`go get`, `bun add`, …) must set the project proxy
   per `Agents.md` before running.

### Suggested commit prefixes

| Prefix | Use |
| --- | --- |
| `docs(themes):` | ADR, plan, module, authoring notes |
| `feat(pages):` | Page catalog, registry, outlet, APIs |
| `feat(themes):` | L0 skin, L1 templates, activation path |
| `feat(extensions):` | Plugin page contributions, approvals |
| `fix(…):` | Bugfixes within this program |
| `test(…):` | Tests only |
| `chore(themes):` | Deprecation markers, feature flags, cleanup |
| `refactor(…):` | Internal moves with **no** behavior change |

### Rollback cheat sheet

| Situation | Action |
| --- | --- |
| Last commit bad, not pushed | `git reset --soft HEAD~1` (keep changes) or `--hard` only if user agrees and tree is clean of other work |
| Last N commits bad, pushed | `git revert` range (preferred) |
| Feature flag off | Disable `pages.runtime_*` / theme runtime flags in options; no code rollback needed |
| Theme broken in prod | Admin: restore core pages / activate default theme (P2+ UX) |
| Need old Layer path during dual-stack | Keep Layer activation until P5; flag selects path |

### Dual-stack safety flag (implement in P0/P1)

Introduce runtime options (names can be adjusted during impl, document final
keys in options module):

- `pages.registry_enabled` — catalog + admin inspect only at first
- `themes.runtime_l0_enabled` — L0 CSS/assets without rebuild
- `themes.runtime_l1_enabled` — template replace path
- `themes.layer_activation_enabled` — legacy Nuxt Layer path (default **on**
  until P5)

Default for production until exit criteria: **legacy on**, runtime features
opt-in or staged.

---

## Architecture Snapshot

```text
                    ┌─────────────────────────┐
  Request URL  ───► │  Page Registry (host)    │
                    │  match page id + provider│
                    └───────────┬─────────────┘
                                │
              ┌─────────────────┼─────────────────┐
              ▼                 ▼                 ▼
        Core ViewModel   Plugin data API    Fallback core
              │                 │                 │
              └────────────┬────┘                 │
                           ▼                      │
                  Template runtime (L1)           │
                  + host SF islands               │
                  + optional L2 widgets           │
                           │                      │
                           └────────── failure ───┘
```

**Still separate systems (do not merge into Page Registry):**

- Contributions (nav items, badges, topic actions, …)
- Events / filters
- Provider slots
- Extension route proxy (plugin APIs)
- Trusted admin Web Release (admin Vue only)

---

## Phase Overview

| Phase | Name | Operator rebuild? | Exit criteria (summary) |
| --- | --- | --- | --- |
| **P0** | Docs freeze + inventory | N/A | ADR+plan linked; inventory of pages & Layer touchpoints |
| **P1** | Page catalog + outlet shell | No | Every core public page has stable id; outlet wired; admin list |
| **P2** | L0 Skin themes | No | Upload/activate skin without Nuxt rebuild; CSS live |
| **P3** | L1 templates + replace/add | No | Replace `forum.home` (and 1–2 more); approval + fallback |
| **P4** | L2 prebuilt widgets | No* | Plugin page with prebuilt component loads safely |
| **P5** | Decouple Web Release + remove Layer theme path | No | Theme activate ≠ full rebuild; Layer path deprecated/removed |

\*Author builds widget once; site does not.

---

## P0 — Inventory, Flags, Doc Anchors

**Goal:** Make the next session able to code without rediscovering scope.

### Tasks

- [x] Link ADR + this plan from `knowledge/index.md` and modules
      (`extensions`, `frontend`).
- [x] Inventory core public routes/pages → draft **Page ID catalog** table
      (live: `docs/extensions/page-catalog.md`; **Go catalog** is P1 SOT).
- [x] Inventory theme activation / Web Release / `current.json` /
      `dev-theme-runtime.mjs` / `extension.theme_activate` job touchpoints.
- [x] List reserved path prefixes for registry.
- [x] Decide flag key names and defaults; document in options note if keys land.
      (Keys documented in page-catalog + options module; **not** registered in
      Options code until P1+.)
- [x] Explicitly mark “no new Layer-only theme features” in extensions module.

### Suggested commits

1. `docs(themes): accept runtime page registry ADR and implementation plan`
2. `docs(themes): inventory core pages and layer activation touchpoints`
   (can be same PR/session as research notes in knowledge)

### Tests / verification

- Docs only; no product behavior change required.
- Optional: markdown link sanity.

### Rollback

- Revert doc commits; no runtime impact.

---

## P1 — Page Catalog + `SFPageOutlet`

**Goal:** Stable page ids and a single resolution hook **without** changing
theme packaging yet.

### Tasks

- [ ] Define host **Page Catalog** (id, default path pattern, access class,
      ViewModel contract version, core component/page mapping).
- [ ] Implement registry service (Go and/or shared JSON consumed by web):
      list pages, resolve provider (always `core` initially).
- [ ] Admin read-only UI or API: “Pages” list (id, path, provider=core).
- [ ] Introduce `<SFPageOutlet page="forum.home" />` (name flexible) and migrate
      **one** page end-to-end as pattern (prefer home).
- [ ] Gradually wrap remaining public pages (can span multiple commits).
- [ ] OpenAPI for any new admin/public registry endpoints.
- [ ] Feature flag `pages.registry_enabled`.

### Suggested commits

1. `feat(pages): add core page catalog definitions`
2. `feat(pages): registry resolve always-core provider`
3. `feat(pages): admin list page providers`
4. `feat(pages): SFPageOutlet and migrate forum.home`
5. `feat(pages): migrate remaining public pages to outlets` (split if large)
6. `test(pages): catalog and resolve coverage`
7. `docs(pages): page id catalog for authors`

### Tests / verification

- Go unit tests for catalog uniqueness and reserved ids.
- Web typecheck for outlet props.
- Manual: home still renders identical with flag on/off if dual path exists.

### Rollback

- Flag off + revert commits; pages still work via previous components if
  outlet is thin wrapper.

### Do not do in P1

- No Liquid engine yet.
- No theme package format change.
- No deletion of Layer builder.

---

## P2 — L0 Skin (zero rebuild)

**Goal:** A theme can change CSS tokens/assets/locales without Nuxt rebuild.

### Tasks

- [ ] Theme manifest fields for L0: `assets/theme.css`, token file, locale
      files (align with existing extension settings where useful).
- [ ] Activation path: DB pointer + cache bust + public URL for theme assets
      **without** queueing `extension.theme_activate` build when only L0.
- [ ] Host injects active theme stylesheet (and CSS variables) on public
      layouts.
- [ ] Validate CSS (css-tree / allowlist); reject scripts in skin packages.
- [ ] Admin: activate L0 theme; progress UI does not claim “building Nuxt”
      for pure L0.
- [ ] Scaffold CLI option or docs for minimal L0 theme.
- [ ] Flag `themes.runtime_l0_enabled`.

### Suggested commits

1. `feat(themes): l0 manifest and asset serving`
2. `feat(themes): activate skin without web rebuild`
3. `feat(themes): inject active theme stylesheet on public layout`
4. `feat(themes): css validation for uploaded skins`
5. `test(themes): l0 activate and fallback to default`
6. `docs(themes): authoring guide for l0 skins`

### Tests / verification

- Upload fixture skin → activate → CSS visible → restore default.
- Confirm **no** new `extension_theme_releases` build row for pure L0 (or
  equivalent assertion).
- Security: reject CSS with `expression` / external `@import` policy as
  designed.

### Rollback

- Flag off; Layer path still works.
- Revert L0 commits; default theme CSS unchanged.

---

## P3 — L1 Templates, Add & Replace Pages

**Goal:** WP/Typecho-like template override; plugins/themes add or replace
views.

### Tasks

- [ ] Choose template approach (document in a short decision addendum if
      needed):
  - Prefer: allowlisted HTML + host tags + limited conditionals, **or**
  - JSON layout + host islands (simpler sandbox).
  - Use mature libs; avoid bespoke full language.
- [ ] Template load + render in SSR path via Page Registry provider.
- [ ] Host component registry mapping `sf-*` → Vue islands (reuse SF*).
- [ ] Manifest: page contributions `add` / `replace` + `target` + `contract`.
- [ ] Plugin optional data loader (existing extension route proxy + schema
      validation).
- [ ] Approval UX: super_admin grants replace; bind id/version/digest.
- [ ] Conflict UX: multiple providers → admin selects winner.
- [ ] Fallback on render error; audit log; one-click restore core.
- [ ] Catch-all route for registered `add` paths; reserved prefix checks.
- [ ] Migrate default home (and optionally topic or login chrome) to default
      templates composed of host islands.
- [ ] Flag `themes.runtime_l1_enabled`.

### Suggested commits

1. `feat(pages): template runtime skeleton and sandbox rules`
2. `feat(pages): host component island registry`
3. `feat(extensions): page contribution manifest validation`
4. `feat(pages): resolve replace provider with admin selection`
5. `feat(pages): approval grants for core page replace`
6. `feat(pages): catch-all dynamic public pages`
7. `feat(themes): default templates for forum.home`
8. `feat(pages): render fallback and restore-core action`
9. `test(pages): replace home template and conflict selection`
10. `docs(themes): l1 authoring and page contracts`

### Tests / verification

- Theme replaces `forum.home` → HTML differs → restore core.
- Plugin adds `/docs/:slug` (or fixture path) with permission class.
- Plugin cannot register `/admin/*` or `/api/*`.
- Login replace must still include host login form component (negative test).
- Failure injection → core fallback.

### Rollback

- Flag off → all providers force core templates.
- Revert P3 commits; P1 outlets still render core Vue pages.

### Do not do in P3

- Module Federation.
- Runtime compile of arbitrary Vue SFC from uploads.
- Core API route replacement.

---

## P4 — L2 Prebuilt Widgets

**Goal:** Advanced plugins ship `frontend/dist/*.js|css` loaded at runtime.

### Tasks

- [ ] Manifest: widget components map + integrity hash.
- [ ] Host loader: `<sf-extension-widget>` / dynamic import with CSP.
- [ ] Admin trust rules for public L2 (stricter than pure templates; define
      whether super_admin grant required).
- [ ] SSR: shell from L1 template; enhance client-side.
- [ ] Disable extension → unregister widgets, clear caches.
- [ ] Reference fixture plugin with a trivial prebuilt widget.
- [ ] Docs: build with Vite library mode → Web Component recommendation.

### Suggested commits

1. `feat(extensions): l2 widget manifest and integrity checks`
2. `feat(pages): runtime loader for prebuilt widgets`
3. `feat(extensions): fixture plugin with sample widget`
4. `test(extensions): widget load and disable cleanup`
5. `docs(extensions): authoring prebuilt frontend widgets`

### Tests / verification

- Widget appears on plugin page; disable removes it.
- Bad hash rejected.
- No full site rebuild on enable (assert).

### Rollback

- Do not load L2 if flag off; pages degrade to template-only.

---

## P5 — Decouple Web Release; Retire Layer Theme Path

**Goal:** Public theme activation never rebuilds Nuxt. Remove dual-stack debt.

### Preconditions

- P2 + P3 exit criteria met in staging/dev.
- Default theme visual parity acceptable on L0+L1.
- At least one non-default theme (e.g. Signal Garden or new fixture) on new
  format.
- Trusted **admin** Web Release still works **independently**.

### Tasks

- [ ] Theme activate API: only runtime registry/skin/template path.
- [ ] Stop composing public theme Layer into Web Release inputs.
- [ ] Keep Web Release for trusted admin frontends only (update ADR
      `trusted-admin-plugin-runtime` notes if needed).
- [ ] Deprecate `extension.theme_activate` build job / `theme-releases`
      uploaded server switch for themes.
- [ ] Simplify `dev-theme-runtime.mjs` / production `runtime.mjs` theme
      watching (or remove theme branch).
- [ ] Migration guide for existing Layer themes; optional compatibility
      shim period with `themes.layer_activation_enabled=false` default.
- [ ] Delete dead code after shim period (separate commits).
- [ ] Update CLI scaffold: default new themes to L0+L1 package layout.
- [ ] Final docs + knowledge module rewrite of “theme = Nuxt Layer”.

### Suggested commits

1. `chore(themes): default disable layer activation flag`
2. `feat(themes): activate only via runtime page registry path`
3. `refactor(web-release): stop including public theme layers`
4. `chore(themes): deprecate theme nitro release pipeline`
5. `docs(themes): migration from nuxt layer themes`
6. `chore(themes): remove layer activation job and supervisor branch`
   (**only after** deprecation window / user approval)
7. `docs(knowledge): extensions and frontend modules post-cutover`

### Tests / verification

- Full `./scripts/test.sh` (or agreed subset + targeted e2e).
- Activate runtime theme: no build worker Nuxt compile.
- Enable trusted admin plugin: Web Release still builds if required.
- Restore default theme instant.

### Rollback

- Re-enable `themes.layer_activation_enabled` **if** code not yet deleted.
- After deletion commits: `git revert` the removal commits (keep deprecation
  commit as historical marker).

---

## Page ID Catalog (draft — finalize in P1)

**Authoritative inventory (P0):** `docs/extensions/page-catalog.md`

Summary (locale-stripped paths; also exist under `/en/...`):

| Page ID | Default path (approx) | Notes |
| --- | --- | --- |
| `forum.home` | `/` | Latest feed — **first outlet target** |
| `forum.category.index` | `/categories` | |
| `forum.category.show` | `/c/:categorySlug` | |
| `forum.tag.index` | `/tags` | `forum.tags.public_pages` |
| `forum.tag.show` | `/tags/:tagSlug` | |
| `forum.topic.show` | `/t/...` | `seo.topic_url_mode` |
| `forum.topic.create` | `/topics/new` | login |
| `forum.profile.show` | `/u/:username` | `features.public_profiles` |
| `forum.my.home` | `/my` | login |
| `forum.my.content_review` | `/my/content-review` | login |
| `forum.settings.profile` | `/settings/profile` | login |
| `forum.settings.security` | `/settings/security` | login |
| `forum.notifications` | `/notifications` | host page, login |
| `moderation.review` | `/moderation` | host + moderation middleware |
| `auth.login` | `/login` | host form required if replaced |
| `auth.register` | `/register` | |
| `auth.forgot_password` | `/forgot-password` | |
| `auth.reset_password` | `/reset-password` | |
| `site.terms` | `/terms` | |
| `site.privacy` | `/privacy` | |
| `site.guidelines` | `/guidelines` | |
| `system.not_found` | n/a | |

Exact paths must match live Nuxt routes and i18n prefix strategy when catalog
is implemented.

### Reserved path prefixes (draft)

See full table in `docs/extensions/page-catalog.md`. Minimum set:

- `/control-panel` (default admin prefix) and configured admin prefix
- `/admin` (legacy constant)
- `/api`
- `/_nuxt`, `/__nuxt`, `/__sforum`
- OAuth / session callback paths as implemented
- Attachment authenticated download paths as implemented

---

## Manifest Sketch (non-normative until P2/P3)

```json
{
  "pages": [
    {
      "id": "acme.custom-home",
      "action": "replace",
      "target": "forum.home",
      "template": "templates/home.html",
      "contract": "sforum.page.home@1"
    },
    {
      "id": "acme.docs",
      "action": "add",
      "path": "/docs/:slug",
      "template": "templates/docs.html",
      "access": "public",
      "data": {
        "source": "plugin",
        "route": "/docs/data"
      }
    }
  ],
  "skin": {
    "css": ["assets/theme.css"],
    "tokens": "assets/tokens.css"
  }
}
```

Final schema lives with extension manifest validation (JSON Schema preferred).

---

## Explicit Non-Goals

- Replacing core REST mutations via page registry
- Untrusted runtime Vue SFC compilation
- Multi-active themes
- Module Federation in v1
- Stopping contribution points / filters / providers (they stay)

---

## Parallel Work Warnings

Do **not** block unrelated tracks (storage E7, security fixes) on this plan.
If touching shared files (`nuxt.config`, extension activate API, web release
composer), keep commits tiny and coordinate via handoff notes.

Avoid expanding Layer theme capabilities except critical bugs.

---

## Definition Of Done (whole program)

1. Ordinary theme: HTML/CSS/templates only; activate without site rebuild.
2. Theme or plugin can replace `forum.home` and add a public page safely.
3. Core API and security routes cannot be overridden.
4. Approvals, conflicts, fallback, restore, audit exist.
5. Web Release is not required for public theme L0/L1 activation.
6. Knowledge base and authoring docs match reality.
7. Git history is granular enough to revert any phase without archaeology.

---

## Next Session Starter Prompt (copy-paste)

```text
Implement P0/P1 of knowledge/plans/2026-07-13-runtime-page-registry-themes.md
per ADR knowledge/decisions/2026-07-13-runtime-page-registry-themes.md.
Follow Commit Discipline: small commits, one concern each, no Layer deletion.
Start with inventory + page catalog if not done; then SFPageOutlet for forum.home.
```
