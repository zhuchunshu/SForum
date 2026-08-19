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
- Manifest V3 admin pages may use `view: component` with a package-prebuilt
  `.mjs` entry and optional CSS. The Host retains the admin route, chrome,
  heading, authorization, exact-artifact trust, and failure quarantine while
  the plugin mounts only the page body. Settings and page components share one
  aggregate `adminFrontendDigest`; component IDs are package-unique and assets
  use component-specific immutable private URLs. Production still does not
  compile Vue SFCs or load Nuxt Layers.
- `AttachmentStorageProviderCatalog` directly owns storage provider candidate
  and availability reads; aggregate `Service` and `CatalogService` forwarding
  methods were removed. Settings projection lives with settings lifecycle, and
  Manifest setting declarations live with the settings document contract; the
  corresponding architecture baselines were lowered.
- Custom image sticker contributions are in approved design, not
  implementation. Plugin authors will maintain conventional sticker
  directories; the extension CLI will generate one exact catalog and Manifest
  V3 will reference that catalog once rather than listing every image. See
  `../plans/2026-07-30-image-sticker-platform.md` and
  `../decisions/2026-07-30-image-sticker-catalog.md`.
- Package installation accepts only Manifest V3. Executable packages accept
  only Protocol V2 with a valid Host API V2 declaration; V1 loaders, runtime
  adapters, SDK entry points, built-in artifacts, fixtures, and rollback paths
  have been removed before the first public release.
- All 18 tracked extension fixtures were re-audited against that baseline.
  Static packages now pass the same SDK `LoadAndTest` path used by the CLI as a
  single inventory gate; Page Registry templates, plugin page schemas, and
  prebuilt admin assets carry current exact `packageFiles` declarations. The
  nine executable reference packages pass their real Protocol V2 integration
  tests.
- The formal SEO ZIP lifecycle chain creates and migrates a disposable
  PostgreSQL database before publishing runtime revisions. Executable fixture
  package paths are temporary, so this test must never fall back to the shared
  development database and leave a boot revision pointing at deleted files.
- Executable process bootstrap is deliberately separate from application
  protocol negotiation: Host and SDK share the fixed HashiCorp go-plugin
  Bootstrap ABI v1 cookie, while only the post-launch gRPC application contract
  is Protocol V2. Cross-built compatibility coverage prevents those version
  axes from drifting together.
- Architecture debt M0-M12 is complete.
- The 28 public-contract and joined-runtime black-box test files now live in
  `Support/Extensions/IntegrationTests`; the legacy package root retains only
  one allowlisted external test helper needed by database-lease subprocess
  tests. Focused recursive test commands must use
  `go test ./app/Support/Extensions/...`.
- The legacy `Models/Extensions` facade delegates to Catalog, Lifecycle,
  Theme, and Settings collaborators. The runtime `Manager` delegates to
  RuntimeSupervisor, InstanceAdmission, RuntimeInvoker, and
  RuntimeEventsProviders. Both packages retain one mutable owner per state
  family and stay under their ratcheted file/receiver caps.
- M6 full gate and browser QA passed. M7 focused tests, typecheck, build,
  architecture validation, and V3 catalog validation passed.
- The 2026-07-31 release-blocker repair migrates historical `enc::` secret
  settings into SecretStore before any SettingsLifecycle read/write, requires
  the documented Marketplace Ed25519 public key in production Compose, passes
  SystemTier priority into the real plugin full-set start plan, and rejects
  CompatFarm RPC errors. The local full repository gate is green.
- RuntimeRollout no longer fabricates a healthy `api-local` acknowledgement.
  Until real node-bound health proof is wired before active publication, the
  ordinary extension lifecycle deliberately does not bind this post-terminal
  rollout hook. Marketplace install remains staged-only and has no supported
  product consumer; Marketplace/Privacy and the real rollout gate remain open
  work rather than advertised release features.
- The generic `account-settings` navigation contract now carries through
  Manifest V3 lifecycle publication with exact runtime filtering. Core also
  owns the `IdentityDelegation` and `ConsentBridge` capability boundaries for
  optional identity providers; these packages expose no raw Core database or
  session authority and are not OAuth-specific.
- Public roadmaps now label M3/M5/M6/M7 as prerelease residuals. P0-P12 phase
  checklist completion is not presented as stable production completion until
  those rows and the joined M8 gate close.
- Compatibility facades remain only for exact allowlisted consumers and
  tighten as those consumers migrate.
- Enabled Lifecycle V2 plugins now use Host-owned exact disable/enable
  orchestration after a settings mutation. The Host preflights both phases
  before changing the SettingsLifecycle document; delegated provider-settings
  permissions remain scoped to the owning auth/mail plugin. Preflight failure
  leaves settings untouched, while a post-persistence restart failure returns
  an explicit recovery error instead of a generic 500.
- Lifecycle registry compensation reverses source and target fences before
  reconciling durable Identity state. Backend-only plugins with zero page
  contributions are absent from Page Registry and ThemeRuntime; legacy startup
  registration clears stale state instead of creating a blank artifact that
  can never satisfy the Lifecycle V2 exact fence.
- The settings restart coordinator now lives with the existing settings
  lifecycle owner instead of adding another file to the legacy flat package;
  role-suggestion decisions likewise live in a focused Identity Registry file.
  Architecture ratchets remain non-growth constraints rather than being raised
  to accommodate these changes.
- The lifecycle page reconciliation loop now lives with exact page runtime
  staging, Protocol V2 provider invocation lives with provider-slot execution,
  and Page Registry operator binding management has its own focused file. The
  corresponding large-file ratchets were lowered after these extractions.
- Stable contracts now live in `Support/ExtensionRuntime`,
  `ExtensionProtocol`, `ExtensionDatabase`, and `ExtensionComposition`.
  Product Models cannot import the legacy runtime package. The legacy package
  retains named Manager, ProtocolStarter, Protocol V2 Host, lifecycle, and
  SDK/CLI consumers under an exact architecture allowlist.
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
- Manifest/protocol decision:
  `../decisions/2026-07-29-manifest-v3-protocol-v2-only.md`

## Production Remediation Status

Do not close residuals from Support-only tests. Require the production
bootstrap binding plus durable/restart/multi-node evidence defined by the
remediation plan.

| Finding | Status | Plan milestone |
| --- | --- | --- |
| Legacy `enc::` values can be dropped | **closed** by production binding + PostgreSQL first-write regression | M1 |
| Production Marketplace key missing from deploy inputs | **implemented** with strict Compose/env/docs wiring; candidate-image smoke pending | M2 |
| Runtime rollout used fictional `api-local` after activation | fictional Ack removed and unsafe post-terminal binding disabled; real gate still **open** | M3 |
| SystemTier order discarded before startup | **closed**; real full-set starter consumes priority order | M4 |
| Marketplace/Privacy lack real product consumers | actor and rollback lookup hardened; supported HTTP/CLI consumers still **open** | M5 |
| CompatFarm could soft-pass RPC errors and runs a narrow/repeated matrix | RPC soft-pass **closed**; matrix and single-run residual **open** | M6 |
| Commerce uses Dispatcher only for `add` | **open** | M7 |
| Full gates/catalog/web residual | local gates and admin overview QA **closed**; release-image and Page Registry live smoke pending | M8 |

Prior partial evidence, not closure:
`../sessions/2026-07-22-p11-p12-p13-production-rewire-handoff.md`.

## Package Sources And Storage

- Protected built-ins are discovered only under `extensions/builtin/` and are
  boot-synchronized through `SyncBuiltins`.
- Optional ship-with-repository packages live under the tracked
  `extensions/optional/` tree. The directory README is part of the repository;
  only generated plugin backend binaries and package archives are ignored.
- A protected built-in removed from the release tree is atomically removed
  from the latest plugin runtime desired set, disabled, and hidden behind a
  Host catalog tombstone. Immutable extension/version rows referenced by
  historical publications are retained. If the same built-in ID ships again,
  synchronization clears the tombstone but does not silently re-enable it.
- Built-in source, staged immutable version, and active immutable version are
  distinct states. A theme source edit reaches runtime only after built-ins are
  rebuilt, the API restarts and stages the new digest, and an authorized admin
  activates that exact version. Browser QA must inspect the resolved
  provider/digest; checking repository templates alone cannot prove activation.
- Both protected built-in themes declare every `theme.json` page template as an
  exact Manifest V3 `templates` + `packageFiles` pair. The CLI regression gate
  validates both source packages so a page mapping cannot ship without its
  immutable template identity and digest. The source-to-activation workflow is
  now a repository rule and is documented in `extensions/README.md` and the
  extension authoring/CLI guides.
- Uploaded packages are immutable snapshots under `EXTENSION_ROOT`; they are
  separate from public attachments.
- `EXTERNAL_EXTENSION_ROOTS` accepts comma-separated collection roots whose
  children are `plugins/*` and `themes/*`. The API scans them after protected
  built-ins.
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
  missing package before runtime dispatch, so immutable runtime-publication
  history cannot produce a misleading success followed by catalog reappearance.
- Containers must mount every external collection read-only into the API.

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
  out-of-band CLI disable. Initial API plugin convergence failure enters a
  Host-owned recovery-only HTTP mode: health remains live, readiness reports
  the immutable desired revision/artifacts, and every product route returns
  `503` without changing active or staged database authority. The recovery
  readiness and guard paths do not re-read the failed extension catalog, so a
  malformed or incompatible artifact cannot block the fallback HTTP surface a
  second time.
- Ordinary uploaded packages remain recoverable through `extension disable`.
  A protected built-in/system executable requires `extension quarantine` with
  the exact active version and 64-character digest; the command atomically
  disables that artifact and appends immutable runtime and audit evidence.
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
- The planned sticker contract deliberately does not use a handwritten
  `stickerPacks` array or `includes.stickers`. One generated catalog exact-lists
  directory-discovered media, and one constant-size Manifest reference binds
  that catalog. Static install validates it without executing plugin code;
  lifecycle activation imports bytes into Core-owned immutable storage.
- Validation covers identity/version/compatibility, exact package-file hashes,
  dependencies/conflicts/provides, declarations, entry paths, migrations,
  capabilities, admin pages, theme assets/templates, and unsafe archive paths.
- Uploaded ZIP entry names are rejected at the archive-reader boundary before
  any snapshot operation when they contain `..`, NUL, an absolute/UNC prefix,
  or a Windows drive prefix. Snapshot normalization independently rejects
  traversal segments, symlinks, special files, duplicates, and file/directory
  collisions before using `filepath.Join`.
- Manifest locale fallback is exact locale, language prefix, then root display
  fields. Admin micro-frontend locales are separate from manifest identity and
  settings/contribution labels.
- Versioned registries cover routes, hooks, services, providers, jobs,
  schedules, commands, admin surfaces, queries, identity/permission/profile,
  media, navigation/regions, content, cache, assets, and packages.
- `/_sforum` is a Host-reserved resource namespace. Public package bytes use
  content-addressed `/_sforum/assets`; authenticated prebuilt admin assets use
  `/_sforum/private-assets`. Route Registry contributions cannot claim either.
- Public asset publication preallocates only the fixed 256-declaration output
  limit and fails closed above that limit; untrusted manifest slice lengths are
  never added together to determine an allocation size.
- Route Registry supports add, alias, redirect, rewrite, before/after/filter,
  wrap/replace, global middleware, uploads, opaque streams, SSE, and WebSocket
  on declared public/admin/API methods and paths. Final redirect output accepts
  only absolute-path references without query, fragment, CR/LF, or backslash;
  declaration paths, normalized request paths, and final `Location` values
  explicitly reject a second `/` or `\\` so browser network-path
  normalization remains impossible even for restored or prebuilt plans.
- Nuxt plugin-route proxy integration fixtures capture forwarded method, query,
  and body server-side and return a fixed `text/plain` + `nosniff` response;
  attacker-shaped request material is never reflected by the test HTTP server.
  Production proxying still preserves the trusted plugin's declared response
  bytes and media type rather than applying a Host HTML sanitizer.
- Core-owned handlers keep authoritative policy checks. A trusted replacement
  handler or custom guard owns only the authorization contract it explicitly
  declares and must remain inspectable/auditable.
- Registry conflicts, selected providers, grants, active artifacts, replacement
  handlers, and rollback snapshots must be visible to operators.

Generated author catalogs live under `docs/extensions/catalogs/`; V3 runtime
catalogs and governance live under `docs/extensions/v3/`. Route discovery uses
an explicit map for the fixed Fiber registration-method domain; `All` receives
the stable method identity `all`, ordinary methods have fixed lowercase
identities, and unsupported or wildcard-shaped values fail closed. Stable
catalog IDs do not rely on string replacement as sanitization.

## Host API And Runtime

- Protocol V2 uses HashiCorp go-plugin gRPC/AutoMTLS and exact
  Manifest-selected contracts. No alternate executable transport or silent
  downgrade is supported.
- Bootstrap ABI v1, application Protocol V2, and Host API V2 are distinct
  contracts. Startup diagnostics classify cookie, CPU/exec format, dynamic
  dependency, and executable-permission failures without exposing raw child
  stderr through public readiness responses.
- Each process receives a runtime-scoped Host broker bound to token, artifact,
  grant, epoch, instance, authority, deadline, locale, and trace. Plugin-supplied
  actor identity is rejected unless Host-attested delegation exists.
- The supported Go authoring surface is `apps/api/sdk/plugin`; CLI validation is
  `sforum extension test [path]`.
- Author-facing docs added 2026-08-16: `docs/extensions/routes.md` (declared
  HTTP routes: manifest `routes[]` semantics, core guards, Protocol V2
  `InvokeRoute`/`RouteStream`, ingress probe + Nuxt proxy, limits, testing) and
  `docs/extensions/build-and-load.md` (backend go.mod `replace` wiring, frontend
  prebuild, digest/validate/test/package, dev load via `EXTERNAL_EXTENSION_ROOTS`
  or upload, trust + iteration loop). `sforum-custom-content` is the canonical
  route fixture; authoring-guide gained a Reference 4 pointing at it and
  cross-links from both new pages. Both pages have full Chinese translations
  under `docs/zh-CN/extensions/` (routes.md, build-and-load.md); `docs/en-US/extensions/`
  holds short pointer pages so the zh/en handbook trees stay structurally
  parallel (`validate-docs.mjs` enforces the file-list parity).
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
- SMTP is a protected built-in plugin. Attachment storage keeps Core `local`
  as the safe default and supports legacy single-provider selections plus
  Host-owned named plugin instances. The protected `sforum.storage-s3` plugin
  implements AWS S3, MinIO, Cloudflare R2, and compatible services without
  putting vendor SDK behavior in Core; the former FTP/SFTP built-ins are gone.
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
- Theme runtime publications retain their immutable historical actor ID. When
  another node replays Page Registry bindings after that user has been deleted,
  the mutable `approved_by` foreign key resolves to `NULL`, matching its
  existing `ON DELETE SET NULL` contract without changing publication history.
- Theme packages declare a validated `navigationLocations` capability using
  only the four stable v1 location IDs. The runtime projects capability from
  the exact active immutable snapshot; unsupported locations retain operator
  configuration, while Core emergency fallback reports all v1 locations as
  supported. The built-in default theme declares all four locations. The
  former Nocturne/Night Harbor theme is no longer shipped as a built-in.
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
- Lifecycle registry publication now appends durable Identity revisions in the
  same Serializable PostgreSQL transaction as the aggregate registry phase.
  A later registry-family failure therefore cannot leave an Identity tombstone
  committed while the aggregate phase remains at source. Startup also repairs
  the pre-fix shape only when the exact enabled artifact, latest failed disable,
  uncommitted publication, source state/registry phases, actor, and audit event
  all agree; ambiguous or drifted history remains fail-closed.
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
- Ordinary plugin admin pages may use the same trusted prebuilt component
  contract. They are served only for enabled plugins, outside Safe Mode, to an
  actor with `extension.view` and the page's optional declared permission;
  failures preserve the Host admin shell and render a page-local retry state.
- Public plugin pages inheriting the active theme shell remain a separate
  follow-up and are not part of the admin page-body implementation.
- Provider probes run in restricted short-lived processes without a Host API
  token or runtime registrations.
- Public contributions gated by settings bump
  `site.public_surface_revision`. Topic HTML no longer consumes that revision
  because `/t/**` disables whole-page caching; each SSR request resolves current
  Page Registry and contribution state without requiring theme reactivation.
- Page Registry separates behavior replacement from theme presentation:
  `replaceable` admits trusted plugin replacement after approval, while
  `themeable` admits only selected-theme L1 templates around reviewed Host
  islands. Existing replaceable pages remain themeable; `moderation.review` is
  the first themeable/non-replaceable Core surface.
- Admin surfaces include overview, plugin/theme lists and details, settings,
  event log, extension points, Page Registry, lifecycle progress/recovery, and
  provider inspection. The App Store remains a local framework shell until a
  real marketplace consumer is production-wired.
- List/detail runtime RSS is best-effort and attributes only owned backend
  plugin children of the current API or its PID-namespace-sharing Worker.
  Linux release images use procfs, including the production extension root
  `/var/lib/sforum/extensions`; running backends without a current sample show
  "not sampled yet" instead of the false "no independent process" state. The
  admin overview orders per-plugin rows by RSS and reports transient same-owner
  overlap instead of silently inflating one plugin's value.
- Protected built-in backend binaries use `-ldflags="-s -w"` in the development,
  Docker, and release build paths. This removes linker/debug payloads from the
  shipped executable without changing the plugin protocol; the current seven
  built-ins save roughly 69 MiB in aggregate on the local build host.
- Extension trust digest verification streams SHA-256 through a fixed-size
  buffer. It preserves exact-root, stable-file, size, and regular-file checks
  while avoiding an `io.ReadAll` allocation proportional to a 20-30 MiB plugin
  artifact on every GuardPolicy refresh.

## Important Paths

| Path | Responsibility |
| --- | --- |
| `apps/api/app/Models/Extensions` | Package, trust, lifecycle, registries, settings |
| `apps/api/app/Support/ExtensionRuntime` | Plugin process/runtime coordination |
| `apps/api/app/Support/Extensions/LifecycleRecovery` | Exact-evidence durable publication startup recovery |
| `apps/api/app/Support/PluginBootstrap` | Shared Host/SDK process bootstrap ABI and startup diagnostics |
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

1. Complete the new editor product design before implementing the image sticker
   generated-catalog and lifecycle contract.
2. Execute `../plans/2026-07-22-v3-production-rewire-honesty-remediation.md`
   M0-M8 with production-path evidence.
3. Execute public navigation M0 before extending `forum.nav.items`; prove and
   document its production bridge to Navigation Registry instead of adding
   another contribution stack.
4. Keep APILTS compatibility shims until their removal gate, date, and live
   zero-use evidence all pass.
5. Keep system error pages plugin-closed and L0/L1-only when adding future
   status families or browser-facing producers.
6. Keep sensitive Core workbenches plugin-closed unless behavior replacement is
   an explicit product decision; use `themeable` only for constrained Host-island
   presentation.
7. Keep new product integrations on stable provider/registry contracts and
   regenerate the affected Extension Surface Matrix.
