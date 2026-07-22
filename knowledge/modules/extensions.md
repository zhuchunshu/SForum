# Extensions Module

## Purpose

Owns installable plugins and themes, their exact-artifact trust, lifecycle,
runtime registries, provider slots, settings, admin surfaces, and recovery.
Core stays the host framework; deployment/vendor behavior belongs in plugins.

Plugins are multi-enable executable or declarative packages. Exactly one theme
is active. Public theme activation swaps Page Registry/L0/L1 runtime state and
does not rebuild Nuxt.

## Current Program State

- Manifest V3, trust/recovery, lifecycle ledger, Host API v2, registry
  families, Page Registry themes, buildless settings UI, catalogs, and P0-P12
  phase gates are present.
- P13 implementable parity work is closed, but the platform is **not 100%**:
  compatibility deletions remain APILTS-gated, and the production-rewire
  acceptance review reopened eight call-chain findings.
- Do not remove `sforum.protocol.v1` or
  `sforum.theme.l1.request-time-loader` before RemoveAfter around 2026-11-28,
  live zero-shim evidence, and the deletion checklist pass.
- Fail-closed `SFPageOutlet` remains a Host emergency surface by design.
- Forum content revisions V1 keeps history Core-owned: Core exposes
  authorized topic/comment revision and admin content read routes, but no raw
  revision query or mutation provider is open to plugins. Safe observe payloads
  still must not include raw source, reason text, IPs, or attachment provider
  internals. `topic.updated` / `comment.updated` expose revision metadata only;
  plugins have no raw-history query, restore, redaction, or CAS override surface.

Authoritative sources:

- Architecture: `../decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
- Parent task book: `../plans/2026-07-13-trusted-plugin-theme-platform-v3.md`
- Progress/residual ledger:
  `../plans/2026-07-13-trusted-plugin-theme-platform-v3-progress.md`
- Production remediation:
  `../plans/2026-07-22-v3-production-rewire-honesty-remediation.md`
- Generated traceability: `../../docs/extensions/v3/catalogs/traceability.md`
- LTS handoff:
  `../sessions/2026-07-21-trusted-plugin-theme-platform-v3-p13-lts-residual-handoff.md`

## Open Production Findings

Do not close these from Support-only tests. Require the production bootstrap
binding plus durable/restart/multi-node evidence defined by the remediation
plan.

| Finding | Plan milestone |
| --- | --- |
| Legacy `enc::` values are not migrated to SecretStore | M1 |
| Production marketplace signing key is not wired in deploy env | M2 |
| Runtime rollout records after activation and uses fictional `api-local` canary | M3 |
| SystemTier computes `LoadOrder` but startup discards it | M4 |
| Marketplace/Privacy lack real production consumers and safe lifecycle calls | M5 |
| CompatFarm can soft-pass RPC errors, has a narrow matrix, and runs twice | M6 |
| Commerce uses Dispatcher only for `add` | M7 |
| Full gates/catalog/web residual remain | M8 |

Prior partial evidence, not closure:
`../sessions/2026-07-22-p11-p12-p13-production-rewire-handoff.md`.

## Package Sources And Storage

- Protected built-ins are discovered only under `extensions/builtin/` and are
  boot-synchronized through `SyncBuiltins`.
- Uploaded packages are immutable snapshots under `EXTENSION_ROOT`; they are
  separate from public attachments.
- `EXTERNAL_EXTENSION_ROOTS` accepts comma-separated collection roots whose
  children are `plugins/*` and `themes/*`. API and standalone worker scan them
  after protected built-ins.
- External discovery reuses the manifest loader and canonical snapshotter. New
  packages are inert `source=uploaded` installs. Changed packages become staged
  versions; discovery never promotes, trusts, enables, or selects them.
- Invalid roots/packages and duplicate IDs produce structured diagnostics.
  Store/database uncertainty is fatal so boot cannot reconcile against unknown
  lifecycle state.
- Removing a source directory does not uninstall its stored snapshot.
- Containers must mount every external collection read-only into API and
  standalone worker processes.

Decision: `../decisions/2026-07-22-external-extension-source-roots.md`.

## Trust And Lifecycle Boundaries

- Static install validates, previews, and stores an inert package. It never
  executes package code, migrations, frontend imports, or external effects.
- First executable enable requires a one-use, actor-bound `super_admin`
  challenge over the exact digest and complete canonical impact document.
- Grants bind package, backend, admin frontend, migrations, declarations,
  authorities, Host/Frontend contracts, dependencies, and runtime epoch.
- Uploading a new digest stages a candidate; it does not replace active bytes,
  inherit trust, stop the active process, or change provider selection.
- Lifecycle operations use a durable PostgreSQL operation/step ledger with CAS,
  checkpoints, attempts, actor/audit snapshots, and restart recovery.
- Disable/upgrade closes admission before draining routes, services, jobs, and
  schedules. Recovery supports retry, safe skip where declared, rollback, and
  out-of-band CLI disable.
- Safe mode, pre-plugin boot health, immutable snapshot rollback, and CLI
  recovery are Host-owned and non-overridable.
- Raw request/session authority and raw core database access are independent
  high-risk powers, never implied by ordinary extension trust.

## Permissions

- `extension.view`: inspect packages, runtime state, events, and catalogs.
- `extension.plugin.manage`: plugin install, trust, enable/disable, settings,
  upgrade, and uninstall.
- `extension.theme.manage`: theme install, trust, settings, activation, and
  rollback.
- `extension.manage` remains a parent compatibility alias; authoritative
  handlers use the fine-grained permissions.
- Executable trust confirmation remains `super_admin` only even when lifecycle
  administration is delegated.
- Plugin-declared permission keys and recommended role mappings never grant
  themselves. Host role/permission administration remains authoritative.
- Plugin permission `label` and `description` accept `LocalizedText`. The
  exact artifact owns those translations; the Host persists and resolves them
  for permission catalog APIs without adding plugin keys to Core i18n files.

## Manifest And Registries

- Root manifest: `sforum.extension.json`; complex V3 packages may shard through
  `includes` while simple packages keep a single file.
- Validation covers identity/version/compatibility, exact package-file hashes,
  dependencies/conflicts/provides, declarations, entry paths, migrations,
  capabilities, admin pages, theme assets/templates, and unsafe archive paths.
- Manifest locale fallback is exact locale, language prefix, then root display
  fields. Admin micro-frontend locales are separate from manifest identity and
  settings/contribution labels.
- Versioned registries cover routes, hooks, services, providers, jobs,
  schedules, commands, admin surfaces, queries, identity/permission/profile,
  media, navigation/regions, content, cache, assets, and packages.
- Route Registry supports add, alias, redirect, rewrite, before/after/filter,
  wrap/replace, global middleware, uploads, opaque streams, SSE, and WebSocket
  on declared public/admin/API methods and paths.
- Core-owned handlers keep authoritative policy checks. A trusted replacement
  handler or custom guard owns only the authorization contract it explicitly
  declares and must remain inspectable/auditable.
- Registry conflicts, selected providers, grants, active artifacts, replacement
  handlers, and rollback snapshots must be visible to operators.

Generated author catalogs live under `docs/extensions/catalogs/`; V3 runtime
catalogs and governance live under `docs/extensions/v3/`.

## Host API And Runtime

- Protocol v2 uses HashiCorp go-plugin gRPC/AutoMTLS and exact
  Manifest-selected contracts. Protocol v1 is an isolated compatibility path,
  not a silent downgrade.
- Each process receives a runtime-scoped Host broker bound to token, artifact,
  grant, epoch, instance, authority, deadline, locale, and trace. Plugin-supplied
  actor identity is rejected unless Host-attested delegation exists.
- The supported Go authoring surface is `apps/api/sdk/plugin`; CLI validation is
  `sforum extension test [path]`.
- Runtime resilience includes per-extension concurrency, deadlines, circuit
  state, crash/restart reaping, admission fencing, and inspectable degraded
  state.
- Versioned River jobs execute only against the exact live artifact/grant/job
  contract and payload schema. Upgrade policy chooses execute, drain, declared
  migration, or cancellation.
- Unsafe bounded plugin HTTP routes may use Host-owned idempotency replay;
  streaming modes cannot request replay.

## Provider And Product Boundaries

- Core defines stable provider slots, settings/reset/probe UX, permissions,
  no-op or protected defaults, and typed contracts.
- Vendor-specific mail, storage, search, payment, analytics, notification, and
  external-integration behavior belongs in plugins.
- Site PostgreSQL search is the protected default; Meilisearch is an optional
  external plugin.
- SMTP is a protected built-in plugin. Attachment storage supports local and
  plugin provider selection through the current storage contract.
- Core may own shared payment/entitlement semantics, but gateways and vendor
  webhook behavior remain plugins.

Every core module maintains an Extension Surface Matrix. A closed surface needs
an explicit security, integrity, or ownership reason; ordinary integrations
should use dedicated registries instead of raw database or whole-route power.

## Themes And Page Registry

- Themes are buildless runtime packages with `theme.json`, templates, and
  assets. They cannot declare backend runtime, migrations, jobs, or provider
  capabilities.
- L0 owns CSS/tokens/fonts/images/locales; L1 owns reviewed server-rendered
  templates; trusted author-prebuilt L2 is digest-authorized and contract-bound.
- Page Registry matching is deterministic: static before parameter before
  catch-all. Access declarations fail closed.
- Activation verifies exact artifacts, prewarms runtime state, replaces page
  contributions/skin atomically, and preserves rollback.
- The selected theme owns public presentation. Core output is emergency-only
  when the runtime cannot safely resolve/render the selected artifact.
- Theme-defined 403/404/429/server-error work is currently blocked after M0;
  the focused public-resource 404 plan is the precursor. Plugins and public L2
  remain closed for these system pages.

Current plans:

- `../plans/2026-07-22-theme-consistent-public-resource-404.md`
- `../plans/2026-07-22-theme-defined-system-error-pages.md`

## Settings And Admin UI

- Plugins/themes use one versioned settings document. Recommended defaults and
  one-click reset are Host-rendered.
- Schema UI supports tabs, groups, columns, callouts, common field types,
  secrets, validation, and allowlisted settings actions without an extension
  frontend build.
- Complex settings may use an author-prebuilt immutable admin micro-frontend
  only after exact-artifact trust. Import, API, mount, CSS, cleanup, or
  quarantine failure falls back to Schema UI.
- Provider probes run in restricted short-lived processes without a Host API
  token or runtime registrations.
- Public contributions gated by settings bump
  `site.public_surface_revision`; anonymous topic SWR varies by that revision so
  operators do not reactivate a theme after toggling a badge/sidebar item.
- Admin surfaces include overview, plugin/theme lists and details, settings,
  event log, extension points, Page Registry, lifecycle progress/recovery, and
  provider inspection. The App Store remains a local framework shell until a
  real marketplace consumer is production-wired.
- List/detail runtime RSS is best-effort and attributes only owned backend
  plugin child processes of the current API process.

## Important Paths

| Path | Responsibility |
| --- | --- |
| `apps/api/app/Models/Extensions` | Package, trust, lifecycle, registries, settings |
| `apps/api/app/Support/ExtensionRuntime` | Plugin process/runtime coordination |
| `apps/api/app/Support/HostAPI` | Host broker and compatibility API |
| `apps/api/sdk/plugin` | Public Go plugin SDK |
| `apps/web/app/pages/admin/extensions` | Admin extension surfaces |
| `apps/web/app/components/extensions` | Shared extension admin UI |
| `extensions/builtin` | Protected packages scanned at boot |
| `extensions/optional` | Ship-with-repo operator-installed packages |
| `extensions/fixtures` | Contract and CI fixtures |
| `docs/extensions` | Authoring, contracts, catalogs, V3 evidence |

## Verification

- Go: `cd apps/api && go test ./...`
- Web: `cd apps/web && bun run typecheck`
- Extension/V3 gates are wired through `./scripts/test.sh`; use the narrower
  commands named in the active remediation plan while iterating.
- After catalog changes, regenerate/check with the documented generator rather
  than editing generated files by hand.

## Next Steps

1. Execute `../plans/2026-07-22-v3-production-rewire-honesty-remediation.md`
   M0-M8 with production-path evidence.
2. Keep APILTS compatibility shims until their removal gate, date, and live
   zero-use evidence all pass.
3. Close current-HEAD regression M7 before implementing the focused themed 404
   plan or broader system-error pages.
4. Keep new product integrations on stable provider/registry contracts and
   regenerate the affected Extension Surface Matrix.
