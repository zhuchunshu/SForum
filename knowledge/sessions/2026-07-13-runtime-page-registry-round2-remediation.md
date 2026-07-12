# 2026-07-13 Session Handoff — Runtime Page Registry Round-2 Codex Remediation

## Changed

Second-round Codex review fixes for Runtime Page Registry lifecycle, routing,
access, loader SSR wiring, Web Release sync, and contract persistence.

### 1. Deterministic parameterized routes

- New `apps/api/app/Support/Pages/route_matcher.go`:
  - `CanonicalRouteSignature` (param names ignored; static/param/catch-all distinct)
  - compiled route table sorted by specificity (static > param > catch-all)
  - stable match order independent of Go map iteration
- Register rejects semantic path conflicts across extensions
  (`/docs/:slug` vs `/docs/:id`, public vs login equivalents).
- `ResolveAddedPathMatch` returns real route params for templates/loaders.

### 2. Access fail-closed

- Allowed: `public` (empty→public), `login`, `guest`, `moderation`, `permission`.
- Unknown non-empty access fails at register/enable/activate preflight.
- `permission` requires contribution `permission` key; API uses `actor.Can`.
- `guest` returns **409** when already authenticated (not silent allow).
- HTTP controller enforces access before loader.

### 3. PageDataLoader production SSR wiring

- `LoaderGateway` + Pages Controller `WithLoader`.
- Bootstrap injects runtime `RouteTarget` adapter (enabled+running plugins only).
- Strict loopback target validation (URL parse / IP loopback; no `contains`).
- Redirects disabled; Cookie / Authorization / CSRF never forwarded.
- Actor only as `X-SForum-Actor-ID-Hint` (not auth authority).
- Timeout, size, content-type, JSON, sensitive-key, optional schema checks.
- Frontend `SFThemeTemplate` / `SFPageOutlet` / `[...sfRegistryPage]` consume
  SSR `loaderData` only — no client fetch to plugin routes.

### 4. Web Release lifecycle sync

- `WebReleaseCoordinator.ApplyEffects` no longer calls Store.Enable/Disable.
- Uses `ApprovedLifecycleApplier` → `Service.ApplyApprovedLifecycleEffect`:
  - enable: status → runtime start → `RegisterPluginPackage` (compensate on fail)
  - disable: stop runtime → `ClearExtension` → status
- Bootstrap: `WithLifecycle(extensionService)` + `WithLifecycleOwnsStart(true)`.

### 5. contract_version persistence

- Migration `202607130003_page_provider_contract_version.sql`.
- Postgres store R/W `contract_version`.
- Approve requires non-empty matching contract (request == contribution == core).
- Client `templatePath` ignored; path from registered contribution only.
- Resolve re-validates version/digest/contract; mismatch → core + diagnostic log.
- Admin UI shows contract read-only.

### 6. Theme switch atomic replace

- `ReplaceThemeContributions` / `RegisterThemePackageReplacing`:
  final-state preflight; same add paths allowed across old→new theme;
  conflicts with enabled plugins still fail; rollback keeps old snapshot.

### 7. Critical bugfix: Fiber string reuse

- `MemoryStore.UpsertBinding` clones all binding strings (`strings.Clone`).
- Controller clones `pageId` params before Approve/Restore.
- Symptom: map key `forum.home` corrupted to `eorum.home` after subsequent
  Fiber requests reused the underlying buffer — public resolve fell back to core
  while admin/direct Resolve still looked correct until the next map lookup.

### 8. Tests & gates

- HTTP suite: `apps/api/app/Http/Controllers/Pages/controller_test.go`
- Route matcher / theme replace / access / contract tests
- Web Release lifecycle effect tests
- Live optional: `tests/validate-page-registry-runtime.js` (`PAGE_REGISTRY_API`)

## Decisions

- Prefer host SSR loader gateway over client plugin data fetch.
- Prefer lifecycle applier over coordinator touching extension store.
- Prefer `strings.Clone` for any Fiber-sourced string stored past request.

## Loader wiring (production)

1. `bootstrap/app.go` builds `LoaderGateway` with runtime route targets + package roots.
2. `PagesProvider.WithLoader(gw)`.
3. On `GET /pages/resolve` and `/pages/resolve-path`, after access pass:
   `gw.LoadForResolved` / `LoadForContribution`.
4. Response includes `loaderData` / `loaderError`; Nuxt islands receive props only.

## Web Release page sync

1. Publish activation → `ApplyEffects(forward=true)`.
2. Each effect → `ApplyApprovedLifecycleEffect(id, targetStatus)`.
3. Plugin pages appear/disappear immediately; no API restart required.
4. Reverse effects restore previous status with the same lifecycle path.

## New migration

- `apps/api/database/migrations/202607130003_page_provider_contract_version.sql`

## Next

- Operators: run browser live suite with isolated ports:
  `PAGE_REGISTRY_API=http://127.0.0.1:<api> PAGE_REGISTRY_WEB=http://127.0.0.1:<web> node tests/validate-page-registry-runtime.js`
- Optional: full JSON Schema library if simple subset is insufficient.
- L2 remains disabled by design.

## Open Questions

- None blocking for round-2 close.
