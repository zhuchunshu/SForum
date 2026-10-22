# Decision: Host-Approved Protected Default Theme At First Install

## Status

Accepted and implemented 2026-10-23.

## Context

A fresh install rendered the **Core fallback** on every public page
(`data-provider="core"`, `data-template="0"`) until an administrator opened
Extensions → Themes and clicked "重新激活" on the shipped default theme.

Traced cause (verified, not guessed):

1. `SaveBuiltin` inserts a built-in theme as `installed`, not `enabled`
   (`builtinInitialStatus`, `app/Models/Extensions/postgres_store.go`), so the
   theme only becomes active through `ActivateTheme`.
2. `SyncBuiltins` therefore calls `EnsureDefaultThemeActive` →
   `PostgresStore.ActivateTheme`, which publishes a `startup_repair` revision
   with `approveCoreReplacements = false` and `actorUserID = 0`.
3. Core page replacement approval was read only from
   `page_provider_bindings.approved_by` (FK to `users`). A brand-new database
   has **zero users**, so approval was structurally impossible.
4. The apply path then took the candidate-only branch
   (`SwitchThemeContributions`): theme contributions were registered but no
   bindings were written, so `Registry.Resolve` returned `core`.
5. Only the admin activation path (`ActivateThemeExact` +
   `approveCoreReplacements=true`) wrote the 34 bindings and made the theme
   render.

The repository already encoded this behaviour as an expectation:
`assertThemeE2ECoreRendered` in
`apps/api/bootstrap/theme_runtime_restart_e2e_test.go` asserted that the first
Nitro render is the Core fallback.

The approval gate itself is correct and stays: a theme's `theme.json` can
replace Core pages, which is a real privilege. The defect was applying a
*third-party* trust gate to the Host's own protected package, which the Host
already trusts enough to compile as the emergency fallback
(`RegisterDefaultThemeFallback`) without any approval.

## Decision

1. Approval provenance becomes explicit. `page_provider_bindings` and
   `theme_runtime_publications` carry `approval_source ∈ {admin, host}`
   (`theme_runtime_publications` also carries `source_approval_source`).
   `admin` requires a real `approved_by`/`actor_user_id`; `host` requires it to
   be absent.
2. **Only** the protected built-in default theme
   (`id = sforum.default-theme`, `source = builtin`, `is_system`, not
   `is_deletable`) may be host-approved. The check exists in the store
   (`lockProtectedBuiltinDefault`), in publication validation
   (`validThemeRuntimePublication`: host approval with another theme id is
   rejected) and again in the Page Registry adapter.
3. Host approval happens only on the two paths that have no human actor:
   - `PostgresStore.ActivateTheme` (startup repair / first install / theme-loss
     recovery), when no human approval already exists for that exact artifact;
   - `EnsureInitialThemeRuntimePublication` (legacy import: active theme exists
     but the publication ledger is empty).
   `ActivateThemeExact` — the human preview/activation path — never passes the
   switch, so a human activation without `approveCoreReplacements` still
   produces an unapproved revision.
4. A staged **new digest** of the same built-in theme is still a new artifact
   and still requires one explicit admin activation. `promoteStagedBuiltinThemeIfHealthy`
   keeps the unapproved path on purpose: that is the digest checkpoint recorded
   in `AGENTS.md`, and automating it would let a source edit silently change the
   live UI.
5. Uploaded and third-party themes are untouched: no host approval is reachable
   for them, and their bindings are always `admin`.
6. `EnsureDefaultThemeActive` writes one `theme_activated` extension event when
   it host-approves, so the automatic decision is visible in the extension
   event log.

## Consequences

- A fresh install renders `data-provider="sforum.default-theme"` and
  `data-template="1"` on first boot with no operator action; the e2e assertion
  was flipped accordingly (`assertThemeE2EDefaultThemeRendered`).
- Runtime evidence (isolated API, empty database, `storage/builtin-dev` as the
  builtin root): one publication `(approved=t, approval_source=host,
  actor_user_id=NULL, reason=startup_repair)`, 34 bindings all `host`,
  `/pages/resolve?id=forum.home` → `provider=sforum.default-theme`,
  `templatePath=templates/home.html`, rendered three-column shell with the
  navbar island. Restarting the API kept the same single publication and the
  same 34 host bindings.
- `theme_runtime_publications` is append-only, so historical rows cannot be
  backfilled with their true provenance; the migration adds the columns with
  `DEFAULT 'admin'`. For rows with `core_replacements_approved = false` that
  value carries no meaning (the CHECK and the Go validator ignore it). This is
  documented in the migration itself.
- The legacy anonymous `approved ⇒ actor` CHECK had to be replaced; its name is
  PostgreSQL-generated, so the migration locates it by definition instead of
  guessing `theme_runtime_publications_check2`. The Down migration recreates the
  original shape and refuses when host-approved history exists.
- The isolated theme-publication integration fixture now applies
  `202607270063` (missing-artifact removals) as well: it was missing, so every
  test in that fixture failed on `SaveInstalled`/`SaveBuiltin` before this
  change. Those tests now actually run.

## Known adjacent issue (not fixed here)

`page_provider_bindings.approved_by` is `ON DELETE SET NULL`. Deleting the
approving administrator silently turns an approved binding into an unapproved
one for the next `lockThemeApproval` read. That is an independent approval
durability defect; fixing it needs an explicit approval ledger rather than an
FK-dependent boolean.
