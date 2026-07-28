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
- Architecture debt M0-M12 is complete.
- The legacy `Models/Extensions` facade delegates to Catalog, Lifecycle,
  Theme, and Settings collaborators. The runtime `Manager` delegates to
  RuntimeSupervisor, InstanceAdmission, RuntimeInvoker, and
  RuntimeEventsProviders. Both packages retain one mutable owner per state
  family and stay under their ratcheted file/receiver caps.
- M6 full gate and browser QA passed. M7 focused tests, typecheck, build,
  architecture validation, and V3 catalog validation passed.
- Compatibility facades remain only for exact allowlisted consumers and
  tighten as those consumers migrate.
- Stable contracts now live in `Support/ExtensionRuntime`,
  `ExtensionProtocol`, `ExtensionDatabase`, and `ExtensionComposition`.
  Product Models cannot import the legacy runtime package. The legacy package
  retains named Manager, ProtocolStarter, Protocol V2 Host, lifecycle, SDK/CLI,
  and APILTS V1 compatibility consumers under an exact architecture allowlist.
  Decision: `../decisions/2026-07-28-extension-stable-package-boundaries.md`.

Authoritative sources:

- Architecture: `../decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
- Parent task book:
  `../plans/archive/2026-07/2026-07-28-architecture-boundary-debt-repayment.md`
- Progress/residual ledger:
  `../plans/archive/2026-07/2026-07-28-architecture-boundary-debt-repayment.md`
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
- Built-in source, staged immutable version, and active immutable version are
  distinct states. A theme source edit reaches runtime only after built-ins are
  rebuilt, the API restarts and stages the new digest, and an authorized admin
  activates that exact version. Browser QA must inspect the resolved
  provider/digest; checking repository templates alone cannot prove activation.
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
- Admin list/detail responses derive `artifactState` from the active immutable
  package path. A missing package remains visible for operator diagnosis and
  retained lifecycle history, but enable, theme activation, settings
  read/write/reset, settings actions, and runtime restart fail closed. Admin
  plugin/theme/overview pages label the record as artifact-missing and exclude
  it from the settings catalog.
- Super admins can explicitly batch-uninstall uploaded, deletable, disabled
  extensions whose exact package paths remain missing. The Host validates the
  whole batch and records catalog tombstones in one transaction without
  invoking unavailable extension code. Operators choose whether to retain or
  delete Host-owned settings; lifecycle history and plugin-owned business data
  remain in both modes. Re-uploading the same extension ID clears its tombstone
  and restores its catalog identity.
- A missing artifact's per-row uninstall action uses that same tombstone cleanup
  with only the selected extension ID. Ordinary lifecycle uninstall rejects a
  missing package before V1/V2 dispatch, so immutable runtime-publication
  history cannot produce a misleading success followed by catalog reappearance.
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
- Plugin restart is a dedicated Host workflow (`POST
  /admin/extensions/{id}/restart`), not an alias for enable. It preflights the
  exact target and any capability/trust confirmation before downtime, fully
  disables the current runtime, and then re-enables it with deterministic
  phase idempotency keys. A legacy active artifact with a staged Lifecycle V2
  target is bridged through disable, exact staged CAS promotion, and V2 enable;
  a failed bridge remains disabled and can resume the same exact target. Trust
  status and challenge calls use `target=staged` during this recovery so the
  token cannot bind the older active digest; the admin UI keeps this
  disabled-plus-staged state restartable.
  Protected built-ins and declarative artifacts return a normal trust preview
  with `trustRequired=false` even when the uploaded-artifact trust migration
  gate is off.
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
- Role and user permission catalogs include only extension permissions whose
  latest Identity Registry declaration is active. Disabled or uninstalled
  declaration tombstones remain durable but are not assignable.

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
- `/_sforum` is a Host-reserved resource namespace. Public package bytes use
  content-addressed `/_sforum/assets`; authenticated prebuilt admin assets use
  `/_sforum/private-assets`. Route Registry contributions cannot claim either.
- Route Registry supports add, alias, redirect, rewrite, before/after/filter,
  wrap/replace, global middleware, uploads, opaque streams, SSE, and WebSocket
  on declared public/admin/API methods and paths. Final redirect output accepts
  only absolute-path references without query, fragment, CR/LF, or backslash;
  this remains enforced even for restored or otherwise prebuilt plans.
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
- GitHub social login V1 is planned as the protected built-in plugin
  `sforum.auth-github`. Built-in discovery stages it but does not automatically
  trust, enable, configure, or publicly activate it. Core retains all account,
  callback, link, risk/session, and browser-session authority. See
  `../plans/2026-07-27-github-social-login-builtin-plugin.md`. M0 is
  implemented; **T1A** closed Host callback authorization ordering, legacy
  complete bypass, live exact-artifact recheck, PKCE + trusted absolute
  callback URL, and redirectHint validation. **T1B** closed stable
  `IDENTITY_SUBJECT_HMAC_SECRET` config + production fail-closed validation,
  bootstrap digest injection (no process-random), concurrent TTL-aligned
  hashed callback/ticket stores, and registration ticket binding/timestamps.
  **T1C** closed atomic CAS Host activation (exact live package digest bind),
  effective public catalog (Safe Mode / artifact drift / disable remove
  availability), actor-bound activation audit, and truthful probe pending
  (`ok=false`, never `probe_pending` as success). **T1D** closed session-bound
  recent-auth, unlink ownership/revision/TX last-method, password
  upsert/setup, authoritative external registration, and canonical
  CurrentUser session claims. **T1E** closed modular OpenAPI, Core Route
  Catalog reserved callback (closed to Route Registry replacement), controller
  HTTP allowed/denied + lifecycle/two-provider tests, and M0 ADR Host ownership
  correction. Full M1R Host foundation is complete. **T2/M2A** landed the
  protected built-in package `extensions/builtin/plugins/sforum-auth-github`
  (Manifest V3, identity.runtime@1 auth provider, settings/schemas, fake
  GitHub + protocol tests, build-builtin-plugins entry). **T3/M2B** proved
  SyncBuiltins exact-artifact staging without Host public activation, release
  container packaging for `sforum-auth-github`, and Protocol V2 headless E2E
  (plugin runtime → Host login/registration ticket/link). Default remains not
  Host-activated for public login. **T4/M3** admin Login Methods UI complete
  (`identity.provider.manage` may manage auth-plugin settings; page at
  `/admin/settings/login-methods`). **T5/M4A** public login/callback/
  registration UI complete: plugin declares identity provider `label`/`icon`;
  Host public catalog injects localized presentation; Core shells stay vendor-
  agnostic. **T6/M4B** account-security UI implemented. **T7/M5** added the
  lifecycle/security matrix, Identity Extension Surface Matrix, bilingual
  operator/author docs, and start/callback rate limits. Independent review
  rejected program closure. **T8A done:** registration commit fencing
  (activation + live artifact recheck, in-TX policy, `user.registered` +
  transaction-owned audit). **T8B done:** versioned `provider.probe` identity
  operation + GitHub Probe wiring; admin directory merges package catalog
  discovery with live Registry (pre-enable discovered, disabled/drifted
  inspectable; trust/enable authority unchanged); Core admin UI uses Host
  label/icon only. **T8C done:** production Host/plugin refuse fake-GitHub
  endpoint overrides; Redis external-auth rate limit TTL is atomic; auth
  start fails closed on partial wiring; migration 057 no longer touches
  `password_hash`. **T8D evidence prepared:** retained lifecycle/API/browser
  evidence, hard 429 assertions, and migration 058 which downgrades only a
  historical GitHub built-in marked enabled without durable Identity Registry
  evidence. It preserves the immutable package while resetting executable
  status and requires normal actor-bound enable. Independent re-review remains
  mandatory; see
  `../reports/2026-07-27-github-social-login-t8d-requirements-matrix.md`.
  **R1 remediation (2026-07-27):** the Host repeats generic exact-contribution
  and activation validation after auth-provider completion and at the admitted
  browser-session effect boundary. This protects all executable auth-provider
  contributions from Safe Mode, activation, and artifact changes without
  adding vendor branches or changing protected built-in ownership.
  **R2 remediation (2026-07-27):** the Host's external-registration mutation
  reads the operator registration policy from the same PostgreSQL transaction
  as user/link/audit creation. This is Core-owned policy enforcement and does
  not expose any option or database authority to provider plugins.
  **R3 remediation (2026-07-27):** external-registration operator audit no
  longer stores the owning artifact digest. The host preserves extension
  artifact history in its dedicated immutable records; redacting an account
  registration audit must not mutate that separate lifecycle evidence.
  **R4 remediation (2026-07-27):** executable auth-provider disable uses the
  normal Lifecycle V2 service path. The GitHub reference contributes only a
  generic no-side-effect lifecycle stream; Host lifecycle ownership drains the
  exact runtime and retires its live and durable Identity Registry publication.
  **R5 remediation (2026-07-27):** legacy startup repair checks lifecycle
  operation and audit evidence as well as the current durable root. This keeps
  immutable artifacts and never auto-enables code; ambiguous partial history is
  preserved for explicit operator recovery rather than silently downgraded.
- Public navigation completion is planned through the existing V3
  Navigation/Region authority. Plugins declare exact-artifact contributions;
  Core owns operator placement/defaults/backup and themes own presentation.
  Direct plugin writes to operator navigation tables or raw DOM injection stay
  closed. See
  `../plans/2026-07-27-configurable-public-navigation-platform.md`.
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
- Theme packages declare a validated `navigationLocations` capability using
  only the four stable v1 location IDs. The runtime projects capability from
  the exact active immutable snapshot; unsupported locations retain operator
  configuration, while Core emergency fallback reports all v1 locations as
  supported. Default and Nocturne declare all four locations.
- Template validation and runtime island binding are separate gates. Every
  Host island used by a theme template must be present in both
  `allowedHostIslands` and `productionThemeIslandBindings`; paired public
  surfaces such as topic create/edit require completeness tests across all
  builtin themes.
- CSS safety matching operates on full declaration names. Dangerous legacy
  `behavior:` remains rejected, while standard suffix properties such as
  `overscroll-behavior:` must not be rejected by substring matching.
- The selected theme owns public presentation. Core output is emergency-only
  when the runtime cannot safely resolve/render the selected artifact.
- Theme-defined system error pages are **completed** for 403, 404, 429, and
  500/502/503/504. They are virtual Page Registry surfaces selected only by
  the Nuxt error boundary; they have no ordinary public path match. The active
  theme may provide L0/L1 presentation through exact-artifact activation, while
  Host owns status, safe copy/actions, retry policy, cache/robots, and Core
  emergency fallback.
- Plugins cannot replace `system.*` surfaces, and public L2 widgets are
  rejected on system error templates. This closed surface is intentional for
  availability and information-disclosure reasons; reviewed Host islands are
  the only dynamic elements allowed inside theme templates.

Relevant plans:

- `../plans/2026-07-22-theme-consistent-public-resource-404.md`
- `../plans/2026-07-22-theme-defined-system-error-pages.md`

## Settings And Admin UI

### External Auth R7 Recovery

- The plugin runtime coordinator defers application of an older desired full
  set while a durable lifecycle operation remains open. A lifecycle operation
  is therefore the only producer of its exact cross-registry transition.
- Built-in synchronization stages a changed artifact for an enabled executable
  plugin; it never advances `active_version_id` or emits a runtime publication.
  Normal enable/upgrade lifecycle remains the sole activation authority.
- Two narrowly scoped legacy GitHub recovery migrations preserve immutable
  history while restoring a pre-lifecycle audited active artifact and appending
  one matching runtime full-set revision. They do not apply when successful
  lifecycle evidence exists, do not auto-enable code, and do not relax the
  Identity Registry exact-artifact fence.
- R1-R7 remediation runtime evidence was completed against isolated PostgreSQL:
  real lifecycle disable retired the exact provider publication and made both
  catalog and login start fail closed; real enable, probe, and activation then
  restored the exact package digest to the public catalog. Artifact drift and
  Safe Mode also publish no provider. This is **整改完成，等待独立复审**; it is
  not a self-declared program closure. See
  `../reports/2026-07-27-external-auth-r1-r7-requirements-evidence-matrix.md`.

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
  `site.public_surface_revision`. Topic HTML no longer consumes that revision
  because `/t/**` disables whole-page caching; each SSR request resolves current
  Page Registry and contribution state without requiring theme reactivation.
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

### Notifications Extension Surface

- Manifest V3 `notificationTypes` is inert exact-artifact data. A declaration
  owns only ids below its extension namespace and binds a digest-checked schema
  package file; plugins cannot declare required notices.
- Host API v2 `notifications.emit@1` is the sole plugin-to-Core emission path.
  Existing broker authentication and Host Command scope resolution bind exact
  artifact, trust grant, runtime epoch, and instance before notification policy
  or persistence runs.
- The Go SDK constructs actorless typed requests. Host broker validation rejects
  raw actor/session authority; command validation rejects foreign namespaces,
  schema/version drift, undeclared targets, inactive users, oversized/bulk
  recipient sets, rate overflow, and expired deadlines.
- Raw Core notification, preference, and delivery tables remain closed. Safe
  mode/disable/uninstall close new admission while historical notification rows
  retain Host-owned fallback rendering.
- `notification.channel.web_push` is an independently selected provider slot.
  Core owns subscription identity, policy, projection, River retries, and the
  redacted ledger; the exact plugin artifact owns VAPID/Web Push transport.
- The protected built-in `sforum.web-push` is never auto-enabled, configured,
  or selected by discovery. Secret settings remain redacted and the Host-owned
  service worker cannot import provider code.

## Verification

- Go: `cd apps/api && go test ./...`
- Web: `cd apps/web && bun run typecheck`
- Extension/V3 gates are wired through `./scripts/test.sh`; use the narrower
  commands named in the active remediation plan while iterating.
- After catalog changes, regenerate/check with the documented generator rather
  than editing generated files by hand.
- After changing a built-in public theme: run
  `./scripts/build-builtin-plugins.sh`, restart the API, activate the staged
  digest through `/control-panel/extensions/themes`, confirm the Page Registry
  bindings for every affected page, then perform browser geometry and
  interaction checks on the active artifact.

## Next Steps

1. Execute `../plans/2026-07-22-v3-production-rewire-honesty-remediation.md`
   M0-M8 with production-path evidence.
2. Execute public navigation M0 before extending `forum.nav.items`; prove and
   document its production bridge to Navigation Registry instead of adding
   another contribution stack.
3. Keep APILTS compatibility shims until their removal gate, date, and live
   zero-use evidence all pass.
4. Keep system error pages plugin-closed and L0/L1-only when adding future
   status families or browser-facing producers.
5. Keep new product integrations on stable provider/registry contracts and
   regenerate the affected Extension Surface Matrix.
