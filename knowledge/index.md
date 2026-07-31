# Knowledge Index

Project memory entry point for humans and AI sessions. Read this file, the one
hot handoff for the current workstream, and the relevant module note. Do not
load archived sessions or completed plans as current context.

## Active Workstreams

### Custom Image Sticker Platform

- Status: **active design**; Core/plugin/storage/rendering architecture is
  approved, including the generated plugin catalog and immutable historical
  assets. The Forum Canvas base editor is implemented and Browser-verified;
  the sticker catalog, node, picker, and admin product are not implemented.
- Plan: `plans/2026-07-30-image-sticker-platform.md`
- Handoff: `sessions/2026-07-30-image-sticker-platform-design.md`
- Decision: `decisions/2026-07-30-image-sticker-catalog.md`
- Modules: `modules/forum.md`, `modules/extensions.md`,
  `modules/attachments.md`

### Configurable Public Navigation Platform

- Status: **active, M0-M6 complete**; all four public locations share the
  canonical actor-sensitive navigation authority, sidebar taxonomy expands
  through `core.dynamic.categories`, stored placements now match the admin
  document (footer navigation defaults empty), and exact active-theme capability
  is projected from immutable runtime state. M7 is the final lifecycle and
  release gate.
- Plan: `plans/2026-07-27-configurable-public-navigation-platform.md`
- Handoff:
  `sessions/2026-07-28-public-navigation-platform-handoff.md`
- Decision:
  `decisions/2026-07-27-operator-owned-public-navigation.md`
- Modules: `modules/options.md`, `modules/frontend.md`,
  `modules/extensions.md`

### V3 production rewire honesty remediation

- Status: **ready**; eight production-call-chain findings remain. Support-only
  evidence is not production closure.
- Plan: `plans/2026-07-22-v3-production-rewire-honesty-remediation.md`
- Prior partial evidence: `sessions/2026-07-22-p11-p12-p13-production-rewire-handoff.md`
- Module: `modules/extensions.md`

### Built-in GitHub social login

- Status: **整改完成，等待独立复审** (2026-07-27). R1-R7 now have focused,
  isolated runtime, and Browser evidence; the generic Host/plugin architecture
  and protected built-in reference remain subject to independent acceptance.
- Plan: `plans/2026-07-27-github-social-login-builtin-plugin.md`
- Remediation:
  `plans/2026-07-27-external-auth-core-plugin-review-remediation.md`
- Handoff:
  `sessions/2026-07-28-github-plugin-restart-handoff.md`
- Prior remediation handoff:
  `sessions/2026-07-27-external-auth-review-remediation-handoff.md`
- Evidence matrix:
  `reports/2026-07-27-external-auth-r1-r7-requirements-evidence-matrix.md`
- Decision: `decisions/2026-07-27-github-social-login-builtin-v1.md`
- Module: `modules/identity.md`

### V3 P13 residual

- Status: P0-P12 complete; Manifest V3 + Protocol V2 are now the only accepted
  extension contracts. Remaining P13 work is the honesty remediation above and
  the independent request-time theme-loader APILTS residual.
- Do not remove `sforum.theme.l1.request-time-loader` before its RemoveAfter
  date plus zero-shim evidence.
- Plan: `plans/2026-07-13-trusted-plugin-theme-platform-v3.md`
- Progress ledger: `plans/2026-07-13-trusted-plugin-theme-platform-v3-progress.md`
- Handoff: `sessions/2026-07-29-manifest-v3-protocol-v2-only.md`
- Module: `modules/extensions.md`

### Content-addressed asset namespace

- Status: **implemented**; browser-facing theme and extension resources use
  Host-reserved `/_sforum` digest paths outside the JSON API namespace.
- Handoff: `sessions/2026-07-23-content-addressed-assets-handoff.md`.
- Decision: `decisions/2026-07-23-content-addressed-asset-namespace.md`.
- Residual: remove legacy `/api/v1/...assets...` compatibility paths only with
  APILTS/deprecation evidence.

## Current Project State

- **Web:** Nuxt 4, Vue 3, Nuxt UI 4, Bun, SSR-first, `zh-CN` default and
  `en-US` secondary locale.
- **API:** Go Fiber v3, PostgreSQL, Redis, River, Goose, and sqlc.
- **Forum:** taxonomy, topics/comments, moderation lifecycle, configurable
  policy, million-scale read-path work, and content revisions V1 are shipped.
- **Identity:** Redis sessions, RBAC, permission overrides, first-user
  `super_admin`, and account-session management are shipped.
- **Search:** protected PostgreSQL site search is the default; Meilisearch is an
  optional external plugin; Host-ledger reconciliation repairs missing, stale,
  and obsolete documents automatically.
- **Extensions:** Manifest V3, exact-artifact trust, lifecycle, Host API v2,
  registries, Page Registry themes, theme-defined virtual system error pages,
  and buildless settings UI are present; extension-owned permission
  localization is present; Bootstrap ABI v1 is independent from Protocol V2,
  and failed initial plugin convergence leaves the API in Host recovery-only
  mode; production-rewire honesty findings remain open.
- **Notifications:** V2 is complete: transactional reply/mention/moderation
  fanout, layered policy and own-user preferences, exact-artifact plugin
  emission, durable-revision SSE with REST/reconnect fallback, generic channel
  delivery, and the protected Web Push reference provider are shipped.
- **Dev:** Compose owns PostgreSQL, Redis, and Mailpit. The user owns the web
  dev server on port 3000; do not kill it.

## Latest Handoff

- Editor image upload: the shared Tiptap editor now opens a click-or-drop
  upload modal from its toolbar, supports clipboard image uploads and
  exact-position drag uploads through the existing attachment policy, persists
  transactional attachment identity, and permits image-only topic/comment
  publication without fabricating plain text. API, focused frontend, and
  architecture tests pass; the advanced local static URL prefix remains
  optional and empty by default:
  `sessions/2026-07-31-editor-image-upload-modal-and-paste.md`
- Extension fixture audit: all 18 tracked fixture packages now satisfy the
  Manifest V3 / Protocol V2 baseline; three stale static manifests were fixed,
  a full static inventory gate was added, and the deleted V1 Host API reference
  was removed from current authoring guidance:
  `sessions/2026-07-31-extension-fixture-audit.md`
- Navbar notification preview: the public bell now opens all/reply/mention
  previews with at most the latest three rows per tab, a full-history hint,
  recipient-authorized excerpts, desktop popover and mobile bottom-sheet
  behavior. Typecheck, focused tests, architecture validation, and user
  interaction verification pass:
  `sessions/2026-07-31-navbar-notification-preview.md`
- Selection quote reply: eligible topic/comment text selections now show an
  absolute, content-anchored `引用并回复` action and feed a safe Markdown quote
  into the existing topic/comment composer and Notification V2 path. Focused
  tests, typecheck, and desktop Chrome interaction pass; the connected Chrome
  viewport override did not apply for the requested mobile-size replay:
  `sessions/2026-07-31-selection-quote-reply.md`
- Edit save validation feedback: topic and comment edit buttons now explain
  no-op saves and missing cross-author reasons while preserving native disable
  only for in-flight submissions. Focused tests, typecheck, and architecture
  validation pass; rendered topic-route Browser QA remains blocked by repeated
  in-app navigation timeouts:
  `sessions/2026-07-31-edit-save-validation-feedback.md`
- Mobile topic editor visibility: topic create/edit now align the editor body
  with their responsive canvas height and preserve page-specific bottom space
  after shared home padding, so the complete status row scrolls above the fixed
  action dock. Focused tests and Chrome QA at `402x905` plus `1280x720` pass:
  `sessions/2026-07-31-mobile-topic-editor-visibility.md`
- Admin user sorting: `/control-panel/users` now requests stable server-side
  ordering by joined/updated time, username, display name, email, or status,
  with selectable direction and page-1 reset. Automated gates pass; rendered
  desktop/mobile Browser QA remains pending after Chrome tab-claim timeouts:
  `sessions/2026-07-31-admin-user-sorting.md`
- Forum code highlighting: topic/comment rich content now uses the selected
  paper-line code block with language labels, sticky line numbers, copy Toasts,
  expanded grammars, and light/dark styling. `/t/59` now treats undeclared code
  as `纯文本 / TXT` and has no nested rounded border:
  `sessions/2026-07-31-forum-code-highlighting.md`
- Admin user-group tabs: `/control-panel/roles` now separates group management
  and extension permission reviews into query-synced fixed tabs, lazily loads
  review data, and preserves the exact approved-permission merge into dirty
  role drafts. Typecheck and focused source tests pass; rendered manual QA is
  pending:
  `sessions/2026-07-31-admin-role-tabs.md`
- Attachment upload policies: existing RBAC remains the upload-eligibility
  authority, while audited role/user policies resolve per-file limits under
  site and HTTP transport caps. The admin page exposes role and user controls,
  and oversize ordinary uploads return a specific 413 response:
  `sessions/2026-07-31-attachment-upload-policies.md`
- Flat comment separator rendering: removed the row bottom border that looked
  like an intermittent horizontal scrollbar while moving across comments;
  tree-mode branch separators and code-block scrolling remain unchanged:
  `sessions/2026-07-31-comment-separator-rendering.md`
- Responsive public sidebar: desktop and mobile now share
  `public.sidebar.primary`, one serializable drawer owner, and the same
  page-sidebar DOM; Navbar falls back to generic navigation only on pages
  without a desktop left rail, while V1 mobile data remains compatible:
  `sessions/2026-07-31-responsive-public-sidebar.md`
- Mobile comment actions now keep reply/permalink inline, move secondary
  actions into a floor-adjacent menu at narrow widths, and preserve the full
  desktop action strip; active-theme Browser QA passed at `402x905` and
  `1280x720`:
  `sessions/2026-07-31-mobile-comment-actions.md`
- Default-theme topic readability now uses a semantic 12/14/14/16px scale;
  body text, publication metadata, comments, composer guidance, and the right
  rail are larger and clearer. The mobile discussion heading now preserves the
  desktop horizontal geometry with a compact 44px row and 24px lead-in. Source
  tests and theme validation pass; the new immutable candidate awaits confirmed
  local reactivation and 402px Browser evidence:
  `sessions/2026-07-31-default-topic-readability.md`
- Release gate repair: `v3.0.0-alpha.3` stopped before publication because the
  main CI lacked a direct Vue SFC compiler test dependency and the release
  waiter misparsed GitHub's empty in-progress conclusion. Fresh Web installs
  now also run `nuxt prepare` before Bun resolves Nuxt aliases; the next
  immutable prerelease is `v3.0.0-alpha.4`:
  `sessions/2026-07-31-release-gate-repair.md`
- Release pipeline artifacts: the maintainer helper returns immediately by
  default; GitHub reuses the exact main CI, publishes four multi-platform
  images, six cross-platform CLI archives, two Linux backend bundles,
  checksums, and provenance after scan and smoke:
  `sessions/2026-07-30-release-pipeline-artifacts.md`
- Attachment image optimization: ordinary proxied JPEG/PNG uploads now produce
  durable `display` variants in the background, use them through the existing
  attachment URL with transparent original fallback, and expose policy,
  statistics, and explicit backfill controls in Attachment Configuration:
  `sessions/2026-07-30-attachment-image-compression.md`
- Personal appearance settings: logged-in users now have a dedicated
  `/settings/appearance` sidebar page with immediate unsaved preview,
  save-only persistence, and explicit restoration to live site inheritance;
  normal and system-error documents share the same user-first appearance
  resolution, including authenticated hard-refresh errors:
  `sessions/2026-07-30-personal-appearance-settings.md`
- Unlinked external login choice continuation: an unbound provider subject now
  enters `/auth/continue` and may either prove an existing local account then
  auto-bind, or use the existing registration flow then auto-bind. Both paths
  share the Host ticket/link authority, with browser binding, exact-artifact
  revalidation, and no email-based account matching:
  `sessions/2026-07-30-external-login-registration-continuation.md`
- Shared topic comment composer: quick reply remains inline; advanced reply,
  comment reply, and comment edit use one bottom drawer with pointer/keyboard
  height adjustment. Its toolbar/canvas/status layout now expands only the
  canvas and preserves the status row when compact heights require scrolling.
  The legacy advanced-reply page redirects into that drawer, and desktop/mobile
  Browser QA plus hydration checks pass:
  `sessions/2026-07-30-shared-comment-composer-drawer.md`
- Custom image sticker platform design: the Forum Canvas base is now live in
  `SFEditor` with quiet focus, responsive toolbar scrolling, and preserved
  write/preview plus trusted L2 contracts. Markdown source and native JSON
  inspection are no longer exposed, while their persistence payloads remain.
  The Unicode emoji picker is gone, while the real sticker command remains
  gated on the immutable catalog and `sforumSticker` node; size caps remain
  `128x128` desktop and `96x96` mobile:
  `sessions/2026-07-30-image-sticker-platform-design.md`
- Uploaded avatar media route repair: AvatarView now exposes stable
  `/media/avatars/{publicId}` URLs, the Nuxt Host proxies them to the existing
  attachment authority, and uploaded avatars bypass IPX so they render instead
  of falling back to initials:
  `sessions/2026-07-30-avatar-media-route-repair.md`
- API startup deadlock and theme replay repair: notification LISTEN lifecycle
  now closes before the shared PostgreSQL pool, so bootstrap errors remain
  visible; theme binding replay tolerates a deleted historical approver, and
  the live API is healthy and ready on port 8081:
  `sessions/2026-07-30-api-startup-theme-replay-repair.md`
- Login methods settings page: account login methods now live on independent
  `/settings/login-methods` with a dedicated sidebar entry, Page Registry
  surface, Host island, built-in theme templates, and activated default-theme
  runtime evidence:
  `sessions/2026-07-30-login-methods-settings-page.md`
- Local password settings page: local password setup/change now lives on
  independent `/settings/password` with a dedicated sidebar entry, Page Registry
  surface, Host island, built-in theme templates, and recent-auth-aware UI:
  `sessions/2026-07-30-local-password-settings-page.md`
- Personal access tokens settings: PAT management now lives on independent
  `/settings/tokens` with a dedicated sidebar entry, Page Registry surface,
  Host island, built-in theme templates, Create/Manage tabs, and preset +
  checkbox scope picker:
  `sessions/2026-07-30-personal-access-tokens-settings.md`
- Sidebar about link: the public left sidebar's bottom "About {siteName}"
  entry now has runtime `site.about_url` plus `site.about_open_in_new_tab`
  settings exposed from Site Settings; empty URL keeps the old inert text:
  `sessions/2026-07-30-sidebar-about-link.md`
- Forum cooldown recovery feedback: topic/comment cooldowns stay independently
  configured, while 429 responses now publish standard and structured recovery
  timing and all create surfaces show a server-authoritative countdown without
  locking drafts:
  `sessions/2026-07-30-forum-cooldown-recovery-feedback.md`
- SSR-stable chrome controls: anonymous SSR now renders guest actions
  immediately, while language, appearance, and authenticated-user ClientOnly
  surfaces keep visible, inert, geometry-stable fallbacks until hydration;
  focused tests, raw HTML checks, and desktop/mobile Browser QA pass:
  `sessions/2026-07-30-ssr-stable-chrome-controls.md`
- Shared public page headings: category, tag, home/search, notification,
  account-settings, and moderation list shells now use one Core header
  component with centralized page/section typography tokens; focused tests
  pass and operator visual verification remains:
  `sessions/2026-07-30-shared-public-page-headings.md`
- Daytime background admin follow: Core Personalization now offers 12 localized
  light palettes shared by public and admin surfaces; preset choices preview
  immediately in admin memory, persist only after save, and every background
  mapping remains outside `.dark`. Focused automation and operator interaction
  verification pass:
  `sessions/2026-07-30-daytime-background-admin-follow.md`
- Moderation theme presentation repair: complete and operator-verified; the
  plugin-closed workbench now renders through a constrained active-theme L1
  shell, Host fallback navbar geometry is scoped, and moderation/profile center
  columns use the correct foreground surface token:
  `sessions/2026-07-30-moderation-theme-presentation-repair.md`
- Built-in theme activation repair: the default and Nocturne themes now bind all
  27 Page Registry templates to exact Manifest V3 declarations; source and
  staged-artifact validation pass, and the repaired default digest is staged for
  operator activation:
  `sessions/2026-07-30-builtin-theme-v3-template-declarations.md`
- Localized transactional mail: Core now produces paired HTML/text password
  reset, registration welcome, reply, mention, and moderation templates; mail
  language is snapshotted before queueing, and the new welcome-email Mail
  Settings control defaults off. Authenticated Browser QA is blocked by the
  current login-page `$setup.t is not a function` error:
  `sessions/2026-07-30-localized-transactional-mail.md`
- Profile settings runtime identity: `/settings/profile` now persists the
  private `users.locale` preference through `PUT /auth/locale` and displays its
  username path from normalized public `site.domain`; the domain defaults from
  the trusted `site.url` host, strips protocols/trailing slashes on save, and
  focused domain tests pass while operator UI verification remains:
  `sessions/2026-07-30-account-default-language.md`
- S3 storage provider Admin UX: multi-instance plugin roots can no longer be
  selected as writers, plugin configuration deep-links to the attachment
  instance editor, and new built-in storage providers require explicit enable:
  `sessions/2026-07-30-multi-instance-s3-storage-handoff.md`
- Attachment storage localization: storage-instance UI messages are now in the
  correct attachment namespace, and probe data messages follow request locale
  while retaining stable reasons and raw stored diagnostics; rendered Browser
  QA needs an authenticated administrator session:
  `sessions/2026-07-30-attachment-storage-i18n-fix.md`
- Registration settings state fix: a legacy disabled registration switch with
  no mode now renders as `closed` in Site Settings, while fresh defaults remain
  open; focused unit coverage passes and administrator-session Browser QA is
  still pending:
  `sessions/2026-07-30-registration-settings-state-fix.md`
- Extension black-box test split: all 28 public-contract and joined-runtime
  tests now live in a focused `IntegrationTests` package, the one necessary
  root subprocess helper is ratcheted, and ordinary plus race suites pass;
  the full gate remains blocked by concurrent out-of-scope
  `Models/Extensions` architecture growth:
  `sessions/2026-07-30-extension-black-box-tests-split.md`
- Authentication shared shell: login, registration, and password recovery now
  share runtime public branding and appearance tokens; both built-in themes
  mount the operator-owned footer on all four authentication templates.
  Password-recovery ALTCHA is now enabled by default once the operator enables
  and configures the provider. Manual desktop/mobile verification and
  active-artifact activation remain pending:
  `sessions/2026-07-30-password-recovery-frontend-handoff.md`
- Multi-instance S3-compatible attachment storage: Core now owns named
  instance identity, SecretStore references, exact historical routing, probe
  and one-click writer selection; protected `sforum.storage-s3` owns AWS
  S3/MinIO/R2 behavior, while FTP/SFTP protected built-ins were removed:
  `sessions/2026-07-30-multi-instance-s3-storage-handoff.md`,
  `decisions/2026-07-30-multi-instance-s3-storage.md`
- Comment user preview: comment avatars and author names now open the compact
  A profile card before canonical profile navigation; focused tests, build,
  and component-level desktop/mobile Browser QA pass, while real topic-theme
  QA remains pending API recovery:
  `sessions/2026-07-30-comment-user-preview-handoff.md`
- Plugin bootstrap startup recovery: restored the historical Bootstrap ABI v1
  cookie independently from Protocol V2, added cross-built compatibility and
  built-in version gates, kept the API alive in recovery-only mode on initial
  convergence failure, and added exact protected-artifact quarantine; manual
  operator verification remains:
  `sessions/2026-07-30-plugin-bootstrap-startup-recovery.md`
- Manifest V3 / Protocol V2 only: package installation and executable runtime
  now reject every older or missing contract; V1 runtime, SDK, fixtures,
  built-ins, rollback artifacts, and documentation were removed before public
  release:
  `sessions/2026-07-29-manifest-v3-protocol-v2-only.md`,
  `decisions/2026-07-29-manifest-v3-protocol-v2-only.md`
- Runtime site URL OAuth callbacks: external-auth display and execution now use
  the admin `site.url` first and inherit environment `APP_URL` only when empty;
  request Host remains untrusted, production stays HTTPS-only, and start-time
  callback transactions remain stable:
  `sessions/2026-07-29-runtime-site-url-oauth-callback.md`,
  `decisions/2026-07-29-runtime-site-url-oauth-callback.md`
- Lifecycle Identity stale recovery: exact failed-disable evidence can now
  repair the pre-fix `enabled + tombstone` state at startup, while new durable
  Identity changes commit atomically with the aggregate registry phase;
  disable, enable, restart, settings restart, and cold-start verification pass:
  `sessions/2026-07-29-lifecycle-identity-stale-recovery.md`
- CI quality gate repair: architecture growth was split into focused owners,
  stale fixtures and performance budgets now match current dependencies, and
  the PostgreSQL 17 Actions service is migrated on the required-test port while
  compatibility credentials remain isolated from opt-in suites:
  `sessions/2026-07-29-ci-quality-gate-fix.md`
- Lifecycle V2 settings restart: enabled plugins such as
  `sforum.auth-github` now preflight before persistence and restart the exact
  active artifact through Host lifecycle orchestration; stable recovery errors
  replace the prior generic 500, while operator verification remains:
  `sessions/2026-07-29-lifecycle-v2-settings-restart-fix.md`
- Stale extension permission suggestions: approval remains exact-artifact
  fail-closed, while authorized rejection can now close pending records after
  disable, uninstall, missing-artifact cleanup, or replacement:
  `sessions/2026-07-29-stale-extension-role-suggestion-rejection.md`
- Default site brand assets: the supplied V3 mark now ships at
  `/brand/sforum-logo.svg`; empty runtime Logo/favicon options resolve to it
  without storing the fallback, while operator URLs remain authoritative:
  `sessions/2026-07-29-default-site-brand-assets.md`
- Admin attachment navigation: Attachment Configuration and Attachment
  Management are independent permission-aware routes; Attachment Configuration
  contains Basic Configuration and Compression Configuration tabs; Basic
  Configuration now includes persistent field guidance and units, while the old
  URL redirects compatibly and the governance extension placement follows
  Attachment Management; the manager now provides server-backed button
  pagination with filter reset and stale-page recovery; operator verification
  remains:
  `sessions/2026-07-29-admin-attachment-submenus-handoff.md`
- Personalization brand assets: compact click/drag uploads now auto-fill URL
  plus attachment ID and show previews; brand SVG is safely rasterized to PNG
  without weakening the ordinary attachment denylist:
  `sessions/2026-07-29-personalization-brand-asset-upload.md`,
  `decisions/2026-07-29-brand-svg-rasterization.md`
- Category directory layout: removed the duplicate desktop top offset, moved
  group focus into the main toolbar, and reduced both side rails; immutable
  theme refresh and desktop/mobile manual verification remain:
  `sessions/2026-07-29-category-directory-layout-handoff.md`
- Forum content relative time: topic/comment publish and accepted-edit times
  now use a one-month relative window with site-timezone `Y-m-d H:i:s`
  fallback; automated and manual verification remain for the user:
  `sessions/2026-07-29-forum-content-relative-time-handoff.md`
- Canonical public search route: `/search?q=...` now owns a complete
  `forum.search` Page Registry/theme surface; old homepage search URLs redirect,
  and operator runtime verification remains:
  `sessions/2026-07-29-public-search-route-handoff.md`
- Announcement authoring: labeled bilingual create form, field-oriented
  SForum editor preset, time-window controls, local validation, and
  server-sanitized Markdown presentation; operator Browser QA remains:
  `sessions/2026-07-29-announcement-editor-handoff.md`
- Missing-artifact uninstall routing: per-row cleanup now targets only the
  selected missing extension, ordinary lifecycle DELETE fails closed, and
  immutable publication history no longer causes refresh-time reappearance:
  `sessions/2026-07-29-extension-missing-artifact-uninstall-routing.md`
- GitHub Actions and security remediation: reusable CI/security checks, four
  multi-platform GHCR images, version-pinned deployment, gRPC remediation, and
  Fiber `v3.4.0` across all 18 source Go modules with session failure cleanup;
  the Buf tool graph now uses patched CEL, text, and compression dependencies;
  redirect targets, database pool integers, and stored Argon2 verification now
  reject unsafe input before crossing their runtime boundaries; the unpatched
  imaging dependency is gone and JPEG/PNG transforms now explicitly reject
  TIFF; uploaded extension ZIP paths now have a CodeQL-visible strict source
  guard plus independent snapshot containment checks; external-auth runtime
  evidence now logs only credential-neutral primitive projections; release
  tags now inject one shared Core/Web/API/worker/migrator/CLI build identity,
  shown once beside the SForum admin brand, while the protected overview shows
  build and runtime diagnostics; the bilingual `scripts/release.sh` validates
  and pushes the annotated release tag without building locally; repository
  licensing is now explicitly MIT under Inkedus while separately licensed
  third-party material retains its own terms:
  `sessions/2026-07-29-github-actions-release-pipeline-handoff.md`,
  `decisions/2026-07-29-mit-project-license.md`
- Notification email opt-in defaults: admin per-type email control, Core
  transaction-scoped user preference enforcement, hard site gate, safe default
  migration, personal managed state, and desktop/mobile Browser QA:
  `sessions/2026-07-28-notification-email-opt-in-handoff.md`
- Notification stream lifecycle hotfix: downstream disconnect now destroys the
  Nuxt upstream SSE socket, API subscriptions have one-minute leases, and the
  client uses bounded backoff plus HMR cleanup instead of native retry storms:
  `sessions/2026-07-28-notification-stream-lifecycle-hotfix-handoff.md`
- Notification detail type navigation: list/detail now share the same type
  rail, loaded-scope counts, current-type highlight, and filter-state return;
  focused verification passed while rendered Browser QA remains to repeat
  after the Chrome control timeout:
  `sessions/2026-07-28-notification-detail-type-nav-handoff.md`
- Notification list source identity: reply/mention rows now use the actor's
  configured avatar, system rows use Tabler icons, hidden targets scrub actor
  summaries, and desktop/mobile Browser QA is complete:
  `sessions/2026-07-28-notification-source-avatar-handoff.md`
- Topic detail stale-comment repair: `/t/**` whole-page SWR removed, explicit
  anonymous/session cache-control policies, API generation caches retained,
  and direct anchor/runtime proof with all current comments:
  `sessions/2026-07-28-topic-page-cache-correctness-handoff.md`
- Unified Admin Mail and Notification settings: one permission-aware fixed-tab
  surface, compact shared settings geometry, legacy notification-route redirect,
  and desktop/mobile Browser QA:
  `sessions/2026-07-28-admin-mail-notification-settings-parity-handoff.md`
- Notification detail preview and realtime badge recovery: independent
  authenticated list/detail routes, recipient-authoritative preview, selected
  theme proof, single-owner public chrome, immediate auth subscription, and
  SSE error/reconnect/visible-page REST reconciliation:
  `sessions/2026-07-28-notification-detail-preview-handoff.md`
- Architecture boundary debt M0-M12 completed: fixed tabs, domain placement,
  backend splits, focused collaborators, stable extension contracts, import
  ratchets, ADR, and resumed final gate evidence:
  `sessions/2026-07-28-architecture-debt-m12-handoff.md`
- Architecture boundary debt M10: runtime `Manager` is now a 72-method
  compatibility facade over four focused collaborators with single owners for
  admission and event/provider state; M11 stable packages are next:
  `sessions/2026-07-28-architecture-debt-m10-handoff.md`
- GitHub built-in restart repair: dedicated Host restart orchestration,
  correct built-in trust preview semantics, exact staged-target recovery,
  legacy-to-Lifecycle-V2 bridge, and successful browser/database evidence
  without 409 or exact-fence conflict:
  `sessions/2026-07-28-github-plugin-restart-handoff.md`
- Architecture boundary guardrails: future feature work is now constrained by
  domain placement, fixed Core Tab components, Go package responsibility, and
  baseline-enforced non-growth for large files/flat packages/God objects:
  `sessions/2026-07-28-architecture-boundary-guardrails-handoff.md`
- Runtime public URL override: admin `site.url` may be cleared to inherit
  environment `APP_URL`; public consumers receive the resolved value while
  OAuth/CSRF/cookie security stays environment-owned:
  `sessions/2026-07-27-app-url-runtime-fallback-handoff.md`
- Extension artifact-presence reconciliation: DB-retained plugin/theme records
  expose `artifactState`; missing packages fail closed and super admins can
  explicitly batch-uninstall eligible records through an atomic, data-aware
  confirmation flow:
  `sessions/2026-07-27-extension-artifact-presence-handoff.md`
- Search automatic reconciliation: provider-neutral Host ledger, startup + 15m
  bounded repair schedule, and real-runtime cleanup of 92 historical ghosts:
  `sessions/2026-07-27-search-auto-reconciliation-handoff.md`
- Configurable public navigation M0-M6: revisioned four-location editor,
  recovery/backup, canonical topbar/sidebar/mobile/footer SSR authority,
  dynamic taxonomy, exact theme capabilities, and selected-theme
  desktop/mobile Browser QA are complete; M7 final gate is next:
  `sessions/2026-07-28-public-navigation-platform-handoff.md`
- Notification Platform V2 completed M0-M7 with exact-artifact Web Push,
  real PostgreSQL multi-node/fixture evidence, hidden-target scrubbing, and a
  green full repository gate; use this handoff only for independent review:
  `sessions/2026-07-27-notification-platform-plan-handoff.md`
- Built-in GitHub social login: R1-R7 remediation packet completed with
  isolated HTTP/Browser artifacts; independent review remains required:
  `sessions/2026-07-27-external-auth-review-remediation-handoff.md`
- Topic create/edit selected-theme and Core-fallback shell parity, exact
  artifact activation, CSS validator boundary fix, and editor Host Island
  allowlist:
  `sessions/2026-07-27-topic-composer-shell-parity-handoff.md`
- Topic side card **贡献者**: author+editors avatar group (max 5), public
  contribution timeline modal, `TopicDetail.contributors` +
  `GET /topics/{id}/contribution-timeline` (no body/reason; staff exposed):
  `sessions/2026-07-27-topic-contributors-handoff.md`
- Editor-document edit load fix: topic/comment editors restore native Tiptap
  JSON via `forumEditorInitialContent` → `SFEditor.initialContent`, never seed
  Markdown v-model with `rawContent`; save uses `forumContentFromEditorPayload`:
  `sessions/2026-07-27-editor-document-edit-load-fix.md`
- Comment stream visual refresh: deep-link highlight now light-sweep animation;
  header meta regrouped (time · floor right-aligned, OP badge, one-line reply
  quote): `sessions/2026-07-26-comment-stream-visual-refresh-handoff.md`
- `/settings/profile` + `/settings/security` three-column chrome extracted into
  shared `SFSettingsShell` (slots: main / rail / head-actions; supersedes the
  07-24 "no SettingsShell yet" decision):
  `sessions/2026-07-26-settings-shell-refactor-handoff.md`
- Profile Replies tab deep links now include `/page/N#comment-{id}` via API
  `commentPage` (SSR cannot see URL hash alone):
  `sessions/2026-07-24-profile-reply-comment-page-links.md`
- Comment cross-page anchor SSR resolve (topic page) + path-style `/page/N`:
  `sessions/2026-07-24-comment-cross-page-anchor-handoff.md`
- `/moderation` host chrome sidebar tokens aligned with default-theme hybrid:
  `sessions/2026-07-24-moderation-sidebar-token-parity-handoff.md`

## Recently Completed

- Tri-state color mode reliability: shared Automatic/Light/Dark authority and
  public/admin menus, resolved-only extension bridge, cache-neutral SSR, and
  safe local canonical origin completed. See
  `reports/2026-07-27-tristate-color-mode-reliability-final.md`.
- Settings chrome: shared `sforum-settings` CSS shell, `SFSettingsAccountNav`,
  profile preview rail, security summary rail; fullwidth theme templates;
  chrome now componentized as `SFSettingsShell` (2026-07-26).
- `/moderation` workbench sidebars: reuse `SFHomeNavigation`, notifications rail
  section language, shared mobile drawer keys; queue overview large count;
  decision rail restyle; typecheck + workbench unit tests pass.
- Default-theme `/categories` grouped directory: implemented against confirmed
  Demo 01 with real category-group API data, SSR internal API reads, focused
  tests, typecheck, production build, and browser screenshot QA.
- Default-theme public profile B1: `/u/{username}` now uses the shared
  three-column shell with real public activity grouped by date, real public
  stats/recent topics, self-only edit UI, mobile drawers, i18n copy, OpenAPI
  updates, and focused Go/Bun/browser validation.
- Default-theme notifications and topic composer now use the shared responsive
  three-column shell with real API flows, mobile drawers, persistent errors,
  appearance-aware feedback, and focused Browser QA.
- Theme-defined system error pages: completed 2026-07-23. The selected public
  theme can own L0/L1 presentation for 403, 404, 429, and 500/502/503/504
  virtual surfaces while Host preserves status, safe copy/actions, no-store,
  noindex/nofollow, and Core emergency fallback. See
  `plans/2026-07-22-theme-defined-system-error-pages.md`.
- Theme-consistent public resource 404: M0-M6 completed 2026-07-23. Ordinary
  semantic 404s retain the healthy selected theme and real private HTTP 404;
  Core is emergency-only. See
  `plans/2026-07-22-theme-consistent-public-resource-404.md`.

## Other Open Product Tracks

- Iteration A engagement: `plans/2026-07-12-iteration-a-engagement-loop.md`
- Admin settings blueprint: `plans/2026-07-12-admin-settings-richness.md`
- Extension surface density: `plans/2026-07-12-extension-surface-density.md`
- Strategic direction: `plans/2026-07-12-development-directions.md`

## Navigation

| Path | Role |
| --- | --- |
| `modules/` | Living feature state; prefer over dated prose |
| `sessions/` | One concise handoff per active workstream |
| `sessions/archive/YYYY-MM/` | Cold historical checkpoints; do not load by default |
| `plans/` | Active, ready, blocked, and blueprint task books |
| `plans/archive/YYYY-MM/` | Completed, cancelled, and superseded task books |
| `decisions/` | Architecture/product ADRs; keep and mark replacements in-file |
| `reports/` | Point-in-time performance and security evidence |
| `glossary.md` | Shared framework terms |

Historical references that require verification before use:

- `knowledge/archive/architecture-maturity-audit.md` -- 2026-07-12 pre-V3 scorecard
- `knowledge/archive/legacy-sforum-feature-gap.md` -- partially stale old-product comparison
- `knowledge/archive/research.md` -- early stack/library research; accepted ADRs supersede it

## Session Rules

1. Read root `AGENTS.md`, this index, one relevant module note, and only the
   active plan/handoff required for the task.
2. Do not read `sessions/archive/` or `plans/archive/` unless recovering
   historical evidence.
3. Update the living module note instead of appending completed work here.
4. Keep one hot handoff per active workstream and archive it when the track
   closes or a newer handoff supersedes it.
5. For V3 percentages and LTS residuals, trust the active progress ledger, not
   archived session prose.

## Open Questions

- Production backup destination and retention policy for operator deployments.
- Whether `en-US` copy must be complete for the first public release.
- Category-scoped ACL timing relative to global RBAC.
- Timing of the next architecture maturity re-audit after V3 LTS cleanup.
