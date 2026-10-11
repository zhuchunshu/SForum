# 2026-10-12 Plugin Runtime Full Recovery And Test Isolation

## Changed

- **Root cause chain for the site-wide 503**: integration tests ran against the
  development database (`commerceTestDatabaseURL` preferred `DATABASE_URL` and
  defaulted to the dev DSN), overwriting the real `sforum.gamification` plugin
  row and deleting its exact database grant. Boot convergence then failed with
  `extension database exact grant not found`, entering Host recovery-only mode.
- **Test isolation**: new `scripts/test-api-isolated.sh` (disposable labeled
  PostgreSQL container, replaces `DATABASE_URL`/`SFORUM_TEST_DATABASE_URL`/
  `SFORUM_COMPAT_DATABASE_URL`, migrates, `--compat` runs the compat farm);
  `scripts/test.sh` now routes Go tests through it. New package-level guard
  `app/Support/Extensions/IntegrationTests/database_isolation_test.go`
  (TestMain rejects inherited dev URLs; per-fixture disposable databases;
  URL validation regression test). `commerceTestDatabaseURL` and the shortcode
  builtin harness no longer fall back to the dev DSN. Documented in bilingual
  `docs/*/development/testing.md` under a new "Database test isolation" section.
- **Agent rules**: new "Agent Database And Runtime Isolation" section in
  `AGENTS.md` (dev state is read-only by default, isolated runner mandatory,
  recovery needs backup + exact CAS re-read, verification requires ready+ack+grant).
- **Runtime recovery** (all through formal lifecycle with actor-bound grants and
  the Identity Registry boundary bound): stale publication member removed via
  `extension recover-stale-builtin` (rev 16); test-fixture grant revoked via
  `RevokeOwnSchema`; plugin re-enabled (revs 17/19); after schema fixes, staged
  artifact promoted through `Service.Restart` (revs up to 29). Durable identity
  tip is `active` at the live artifact; latest ack `applied` with 6 members.
- **Gamification contract fixes** (source package + staging rebuilt):
  - Response schemas (`profile.achievements`, `admin.badges`, `admin.levels`,
    `admin.grant-log`, `admin.action.request`) and the OpenAPI fragment now
    declare `badges`/`grants`/`levels`/`users`/`level` as JSON strings,
    matching the TypedDocument primitive wire format.
  - `profile.achievements` GET no longer declares the copy-pasted
    `admin.action.request@1` request schema (empty GET body failed validation);
    the fragment operation metadata was synced.
  - The OpenAPI fragment is now declared in the manifest (`openapi[]` +
    `packageFiles` kind=openapi) and its operationIds were namespaced under
    `sforum.gamification.`; `extension test` 10/10 green; catalog integration
    test green.
- **Parallel M1.5 compile repair** (the worktree did not compile before this
  session's `go build ./...`): added `ExtensionManifest/v3_ai_tools.go`
  (`ManifestAITool`, constants, `IsOwnedID`), `Events/catalog_namespace.go`
  (`OwnsEventName`, `ResolveDefinition`), `Capabilities` catalog entries
  (`ai.invoke`, `events.emit`), `ErrGatewayUnavailable`, manifest `aiTools`
  field + JSON-schema `$defs/aiTools`, fixture list update, and the
  `RecoverStaleBuiltin` interface method. AI-tool dispatch now reads
  `extension.Manifest.AITools` at the Starter call site (no legacy-cap file
  growth).

## Decisions

- Recovery always went through production code paths (formal `Enable`/`Restart`,
  registry-issued grants, durable identity reconcile) — no direct SQL mutation
  of lifecycle tables. Identity Registry boundary binding was the missing piece
  in the first enable attempt (stale durable tip blocked boot).
- The 422 on `GET /gamification/users/:userId/achievements` **remains open**:
  the response schema binding is correct (probe verified binding and typical
  payload match), so the plugin runtime returns something outside the declared
  200 contract (likely a non-200 status with a `{"reason":...}` body, which
  fails catalog lookup and surfaces as 422). This route has never passed
  rendered QA (M6 pending); it needs plugin-side debugging with admin access
  or a temporary trace ring.

## Verification

- `/api/v1/ready` = 200 `ready:true`; `/api/v1/health` = 200 `status:ok`.
- Topics/categories/navigation (direct and via the Nuxt proxy) = 200.
- `plugin_runtime_publication_acks` latest revision `applied`, member_count 6;
  `sforum.gamification` enabled at version-row 2256; matching active grant
  (id 107) and `active` durable identity tip.
- `go build ./...` OK; `git diff --check` OK.
- Isolated-cluster runs: plugin integration tests + URL guard green
  (`TestPluginIntegrationDatabaseURLGuard`, gamification suite).
- `go test ./app/Support/ExtensionManifest/ ./app/Support/Events/
  ./app/Support/Capabilities/` green; `Support/AI` has one pre-existing
  parallel-session failure (`result_cache_test.go` — their unfinished
  ResultCache work, documented as known-not-blocking in their handoff).
- Architecture gate: two residual violations remain, both from the parallel
  session's in-flight M1.5 files (`app/Support/Extensions` 147 files vs cap
  145; `ExtensionManifest/manifest.go` 921 lines vs cap 920 from the shared
  `aiTools` field). They must be resolved when that session lands its work
  (move `events_publish.go`/`protocol_v2_ai_tool.go` or raise with a decision).

## Next

- **Achievements 422 final diagnosis** (structural fixes all landed, one
  observation step remains): the route catalog binding is verified correct
  (probe: `ValidateRouteSchema` passes for `{"userId":int,"points":int,
  "level":string,"badges":string}`), so production 422 means the plugin's real
  response differs — most likely a gRPC-level error (host turns it into an
  unverifiable shape → 422) or an unexpected payload field. A temporary
  `log.Printf` is already planted at `routes.go` `routeProfileAchievements`
  (built into digest d40ace90…); capture the plugin stderr via the API
  process logs (DiagnosticBuffer) or run `GET /gamification/users/1/achievements`
  with the plugin log visible to see `gamification achievements …` lines.
  The error path now returns contract-bound terminal responses
  (`{"reason","message"}` under `admin.action.response@1`) instead of gRPC
  errors — verify that path fires.
- Then finish M6 rendered QA (desktop + 390x844 on `/control-panel/gamification`
  and `/badges/{username}`), remove the temporary `log.Printf`, and re-run
  `extension digest --write` + staging rebuild + admin enable cycle.
- Merge/land the parallel AI M1.5 session and clear the two architecture
  residuals it owns (it was actively editing `activation.go`,
  `database_grant_authority.go`, `protocol_database_lease.go`,
  `lifecycle_coordinator_host.go` during this session's final phase).
- Keep integration tests on `./scripts/test-api-isolated.sh`; never inherit
  dev `DATABASE_URL`.
- Database backups taken during this recovery (delete when no longer needed):
  `/tmp/sforum-before-agent-fix-20261011-233613.dump`,
  `/tmp/sforum-before-identity-reconcile-20261012-000709.dump`.
