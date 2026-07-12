# 2026-07-13 Session Handoff — Runtime Page Registry Codex Remediation

## Changed

Security and correctness fixes for Runtime Page Registry + simple themes after
Codex review rejected the earlier “P0–P5 complete” claim.

### Implemented for real

1. **L2 closed** — `SFExtensionWidget` no longer dynamic-imports remote/package JS;
   disabled placeholder only.
2. **L1 template sanitizer** — bluemonday allowlist + host-island placeholders;
   size/depth/attr limits; XSS suite in `template_test.go`.
3. **Theme assets** — active theme only, digest query, no SVG, nosniff/CSP,
   symlink reject (`theme_package.go` + controller).
4. **No silent ApproveReplace** — activate registers candidates only;
   super_admin approve binds version+digest+actor; audit actions added.
5. **Atomic theme activate** — preflight → DB → registry; rollback on failure.
6. **Bootstrap restore** — `RestoreActiveThemeRegistry` after SyncBuiltins.
7. **Plugin page lifecycle** — enable registers / disable clears; fixture
   `page-registry-demo` proves add/replace/approve/disable/upgrade digest.
8. **All catalog pages** wrapped in `SFPageOutlet`; constrained pages force core.
9. **Dynamic add routes** — `GET /pages/resolve-path` +
   `pages/[...sfRegistryPage].vue` (real paths; `/x/*` compatibility remains).
10. **Server loader skeleton** — `Support/Pages/loader.go` (relative route only,
    loopback, timeout/size/JSON/sensitive keys); browser no longer fetches plugin
    data from templates.
11. **Web Release** — ordinary themes excluded from composition; default theme
    dropped `frontend.admin` / ThemeSettingsPage contribution.

### Temporarily closed / not done

- **L2 full security model** (widget manifest trust, integrity, CSP grant UI,
  upgrade invalidation for widgets) — **disabled**, not half-implemented.
- Browser Playwright e2e against live servers was **not** run in this session
  (user port 3000 left alone); unit/integration tests cover the security and
  lifecycle paths. Operators should still run browser checks for theme switch
  without Nuxt rebuild when convenient.
- Loader is not yet wired to live `RouteTarget` injection inside every resolve
  path in production bootstrap (transport + validation land; full gateway inject
  can be completed when plugin route targets are available on the pages
  controller). Treat loader **contract + unit tests** as done; end-to-end plugin
  data SSR as follow-up if gateway wiring is incomplete in a given deploy.

### Not claimed as browser e2e

Static/string tests (`pageOutlet.test.ts`) only assert wiring. Real browser e2e
must be separate Playwright/manual runs.

## Commits (this remediation)

1. `fix(pages): close L2 widgets and harden L1 templates and theme assets`
2. `fix(pages): require super_admin approval and atomic theme registry restore`
3. `feat(pages): wire outlets, dynamic add routes, loaders, and web-release decouple`
4. (docs/localization/fixture go.mod tidy — final commit)

## Decisions

- Prefer **disable L2** over shipping unprotected dynamic import.
- Prefer **bluemonday** over regex HTML sandbox.
- Theme settings = host schema from `manifest/settings.json`, not theme admin SFC.

## Next

- Wire pages controller loader to extension runtime `RouteTarget` for full SSR
  data path in production requests.
- Optional: mature SVG-in-skin decision (currently forbidden).
- Browser e2e suite: theme switch, restart restore, plugin path, approve/disable.

## Open Questions

- Whether any operator still depends on default theme `ThemeSettingsPage.vue`
  Vue UI (now removed from composition; schema host page is the path).
