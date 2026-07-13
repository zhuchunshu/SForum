# 2026-07-13 Runtime Page Registry And Simple Themes

## Status

Accepted — implemented through P5 with security remediation and **round-2 lifecycle
close** (2026-07-13). See:
- Plan: `knowledge/plans/2026-07-13-runtime-page-registry-themes.md`
- Security remediation: `knowledge/sessions/2026-07-13-runtime-page-registry-codex-remediation.md`
- Round-2: `knowledge/sessions/2026-07-13-runtime-page-registry-round2-remediation.md`

The implemented Page Registry remains a V3 migration input, but its constrained
target is superseded by
`2026-07-13-trusted-plugin-theme-platform-v3.md`: V3 uses Go `html/template`,
complete theme-owned public presentation, trusted digest-bound L2, plugin
template overrides, and trusted Route Registry replacement of declared API/admin
routes. Only safe-mode/pre-plugin health/CLI recovery stay non-overridable.

### Round-2 addenda (normative)

1. **Route matching** uses canonical signatures (param names do not distinguish routes)
   and a specificity-ordered table (static > param > catch-all); never Go map iteration.
2. **Access** is an enum fail-closed: empty→public; unknown fails install/enable/activate.
3. **Loader data** is host SSR only via loopback gateway; clients must not call plugin
   data routes; redirects and Cookie/Authorization forwarding are forbidden.
4. **Web Release** applies extension enable/disable only through full lifecycle
   (`RegisterPluginPackage` / `ClearExtension`), not raw store Enable/Disable.
5. **Provider bindings** persist and re-check `contract_version` on approve and resolve.
6. **Theme switch** uses atomic replace-extension so shared add paths do not block activation.

## Context

SForum today treats installable themes as **Nuxt Layer** packages. Activating an
uploaded theme queues a full web build (River job), writes
`theme-releases/current.json`, and switches Nitro via a supervisor. That works,
but:

- Theme authors must know Nuxt, Vite, SSR, and the host dependency set.
- Operators pay a full-site rebuild for visual changes.
- Theme activation is coupled to **Web Release** (trusted admin plugin frontends).
- Plugin contributions can inject UI fragments, but **core route override is
  forbidden**, so themes/plugins cannot cleanly replace whole public pages
  without shipping another Layer.

Mature forum/CMS products take a different tradeoff:

| System | Lesson |
| --- | --- |
| WordPress | Template hierarchy + selective override; plugins can replace final template |
| Typecho | Tiny theme packages, missing-file fallback |
| Discuz | Fixed extension points, not rebuild-the-app |
| Flarum | Extensions register routes; authors prebuild frontend assets |

SForum already has strong **plugin** contracts (events, filters, providers,
contributions, Host API). What is missing is a **view-layer** contract that is
as simple as WP/Typecho for ordinary themes, while keeping core API and
security authority intact.

## Decision

### Product definition

> Themes and plugins may **add or replace view pages** through a unified
> **Page Registry**. Ordinary pages use **runtime templates** composed of
> **host-owned components**. Complex interaction uses **author-prebuilt**
> assets. Core business and APIs are extended only via providers, filters,
> events, and plugin-owned routes — never by hijacking core mutations or
> security endpoints. **Operators never rebuild SForum** to activate a normal
> theme.

### Three capability layers

| Layer | Author ships | Operator builds site? | Purpose |
| --- | --- | --- | --- |
| **L0 Skin** | CSS tokens, fonts, images, locales | No | Branding / appearance |
| **L1 Template** | Page/layout templates + partials + host component tags | No | Add/replace public page chrome |
| **L2 Interactive** | Prebuilt ESM / Web Components (+ optional CSS) | No (author builds once) | Charts, custom editors, heavy UIs |

Ordinary themes use L0 + L1 only.

### Page Registry

- Every core public page has a stable logical id, e.g. `forum.home`,
  `forum.topic.show`, `auth.login`, `system.not_found`.
- Themes and plugins declare **page contributions**:
  - `action: add` — new path + template (or L2 widget host page)
  - `action: replace` — `target` = stable page id (not raw path grab)
- Resolution order for a request:

  ```text
  URL → match logical page → selected provider (admin/theme default)
       → ViewModel (core or plugin data contract)
       → render template + registered islands
       → on failure → core fallback page
  ```

- Core pages become thin outlets (conceptually `<SFPageOutlet page="..." />`).
- Unknown public paths may hit a catch-all that consults the registry.
- Reserved prefixes never available for custom public routes: `/admin`,
  `/api`, auth/OAuth callbacks, attachment auth endpoints, and other
  security-sensitive paths listed in the plan.

### What may be added or replaced

| Object | Add | Replace |
| --- | --- | --- |
| Public GET view pages | Yes | Yes (via page id) |
| Login/register **chrome** | Yes | Yes, but must embed host form components |
| Ordinary admin feature pages | Yes (existing extension admin pages) | Whitelist only |
| Plugin-owned API under `/api/v1/extensions/{id}/*` | Yes | Own routes only |
| Core API mutations | No | **No** |
| Session / OAuth / attachment auth | No | **No** |
| Permission middleware | No | **No** |
| Provider behavior (mail, storage, search, …) | Via provider slots | Via provider slots |

This **narrows** the older rule “plugins cannot override core routes” to:

> Plugins and themes cannot override **core API or security routes**.  
> They **may** add or replace **view pages** through the Page Registry.

Related accepted decisions remain in force for non-view surfaces:

- Provider slots, events, filters, contributions
- Trusted admin Web Release for privileged Vue admin components
- API policy checks always authoritative

### Template model (not “rewrite everything in Liquid”)

- Host remains **Nuxt** for SSR, admin, i18n, SEO plumbing, and SF components.
- Themes **do not** ship Nuxt Layers as the long-term format.
- L1 templates compose **registered host components** (Vue islands), e.g.
  `sf-topic-list`, `sf-login-form`, `sf-comment-stream`. Themes do not implement
  auth, sessions, or write paths.
- Template syntax may be HTML-with-allowlist + limited Liquid-style control
  flow, or a JSON layout descriptor — exact engine is an implementation choice
  in the plan, not a product hard requirement. Prefer mature libraries
  (e.g. LiquidJS, parse5, css-tree, JSON Schema) over custom parsers.
- Existing SF Vue components stay the implementation of host islands; we do
  **not** reimplement business UI twice for v1.

### Conflict, approval, and safety

- Core page always remains the fallback provider.
- Replacing a core page requires **super_admin** approval bound to extension
  id, version, and package digest.
- Multiple extensions claiming the same `target`: admin **explicitly** chooses
  the active provider (no silent `order: 9999` wins).
- Theme activation may auto-propose that theme’s declared page replaces; the
  activation confirm UI must list impact.
- Extension update invalidates high-risk replace grants until reconfirmed.
- Template failure, missing assets, or plugin data timeout → automatic core
  fallback.
- One-click restore core pages; all replace/restore/conflict choices audit-logged.

### Theme package target shape

```text
my-theme/
├── sforum.extension.json
├── theme.json              # tokens, page declarations, replaces
├── templates/
├── partials/
├── assets/                 # css, images
└── locales/
```

Long-term themes **should not require**: package.json, Bun, Nuxt, Vue SFC
source, Nitro, or a site-wide Web Release to activate L0/L1.

### Web Release scope

- **Theme activation must stop requiring a full Nuxt rebuild** for L0/L1.
- Trusted **admin** plugin (and theme admin settings UI) frontends may keep a
  separate Web Release pipeline; it must not be triggered by ordinary public
  theme skin/template activation.
- Until L1 lands, existing Nuxt Layer activation remains a **compatibility
  path** (see migration), not the product north star.

### Migration stance

1. Add Page Registry and L0/L1 **alongside** current Layer themes.
2. Freeze new investment in Layer-only theme capabilities (no new Layer-only
   features unless required for security/regression).
3. Port default public chrome to host components + default templates.
4. Convert reference custom themes (e.g. Signal Garden) to the new format.
5. Remove theme build jobs, theme `current.json` Nitro switch, and theme
   supervisor coupling only after L0+L1 parity and a documented deprecation
   window.

## Consequences

### Positive

- Theme authoring approaches WordPress/Typecho simplicity.
- Operators get instant (or near-instant) theme switches without rebuilding.
- Plugins gain first-class full-page add/replace without forking Nuxt.
- Security and API authority stay centralized.

### Costs / risks

- Large migration from Layer-based themes and Web Release theme path.
- Need careful SSR + island hydration design.
- Admin UX for page providers, approvals, and conflicts must be built.
- Temporary dual-stack (Layer + runtime themes) increases maintenance until
  cutover.

### Supersedes / revises

| Prior decision | Change |
| --- | --- |
| `2026-07-06-core-framework-plugin-first-architecture` | “No core route override” → no **API/security** route override; view pages allowed via registry |
| `2026-07-06-plugin-event-extension-points` | Same narrowing for routes |
| `2026-07-06-extension-platform-v2` | Theme activation no longer “only via full build pipeline” as end state |
| `2026-07-06-plugin-enable-theme-activate-default-theme` | Public UI ownership stays host+theme; **format** changes from Layer-only |
| `2026-07-07-incremental-theme-fallback` | Fallback idea kept; mechanism becomes template/provider fallback, not Layer file merge only |
| `2026-07-10-trusted-admin-plugin-runtime` | Keep for admin; **decouple** from public theme activation |

Appearance presets (`appearance.theme` / 配色预设) remain distinct from
installable themes.

## Non-goals (v1 of this program)

- Module Federation as the default plugin frontend loader
- Arbitrary remote code execution of untrusted Vue SFC at runtime
- Letting themes own write APIs or permission checks
- Multi-active themes (still exactly one active theme)
- Replacing the Go plugin subprocess model

## References

- Plan / task book: `knowledge/plans/2026-07-13-runtime-page-registry-themes.md`
- Modules: `knowledge/modules/extensions.md`, `knowledge/modules/frontend.md`
- Inspiration notes in session discussion (WP template hierarchy, Typecho
  themes, Flarum routes, Discuz extension points)
