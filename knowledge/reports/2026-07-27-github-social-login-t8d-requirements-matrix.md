# GitHub Social Login T8D Requirements Matrix

Date: 2026-07-27
Status: Evidence packet prepared for independent Codex review. This is not a
program-closure declaration.

## Scope And Boundaries

Only the protected built-in `sforum.auth-github` and its Host-owned external
authentication paths were exercised. No additional provider or program was
started. The normal user-owned Nuxt server remained on port 3000.

Runtime artifact identity:

| Item | SHA-256 |
| --- | --- |
| GitHub backend | `fde320d22822bc9f9457d5987b364d8c9203acbd1847241c98fc6f61ffdcbb7f` |
| GitHub package | `5d73651dd4013bc04abeb6f99f9ef0686303ee683c40d9a483f927f6b5c09942` |
| `start.input` | `e050111f11272b97d0340e1756f9d03a1f3f7d0bfecf914a2fb14deee8750676` |
| `start.output` | `5f25e9910d79859a2b412ed3b2925f42d64a27b4649726f7f958f90055fad2a0` |
| `complete.input` | `d64392bc7a693b6573c1442cce23ba4345eb0bca2b30958d1abfbfa1f79d5eb5` |
| `auth.complete.output` | `9baac20e0ea448f7e6b6e4ee9d72e46663a84b3099ed81d1a787ed2dd4f960a6` |
| `probe.input` | `c104cf368d17c24cc0eccddadf5bed01149e7243f144a81e756a743e85307324` |
| `probe.output` | `ec8213a0d6845edea3a465e1d05c55c1e3324e3b100a1982a40298deae62b4d7` |

## Requirement Matrix

| T8D requirement | Evidence | Result | Review notes |
| --- | --- | --- | --- |
| Replace source-text checks for critical admin/public/account flows | Rendered happy-dom interaction regressions in `apps/web/tests/adminLoginMethods.test.ts`, `authRouteRendering.test.ts`, `authProvidersPublicUi.test.ts`, and `accountSecurityM4b.test.ts`; real browser flow below | PASS with qualification | The decisive evidence is the real Browser QA, not the helper DOM alone. The independent reviewer should reject any source-text-only assertion masquerading as interaction coverage. |
| Assert HTTP rate-limit `429` | `TestM5_StartRateLimited` and `TestT8D_StartRateLimitedAsserts429` hard-fail unless status is 429 and reason is `rate_limit.exceeded` | PASS | Also checks client-IP key shape and redaction; callback limit asserts safe 302 reason. |
| Focused Go/PostgreSQL/controller/plugin tests | Commands in Verification section | PASS | PostgreSQL-focused filters used `SFORUM_TEST_DATABASE_URL=postgres://sforum:sforum@127.0.0.1:15432/sforum?sslmode=disable`. |
| OpenAPI, Nuxt tests/typecheck/build, repository gate | Commands in Verification section | PASS, except separately recorded pre-existing full `bun test` failures | `./scripts/test.sh` passed after migration 058. |
| Build, stage, confirm, enable exact artifact through lifecycle APIs | `GOCACHE=/private/tmp/sforum-go-cache ./scripts/build-builtin-plugins.sh`; `/private/tmp/sforum_t8d_http_flows.mjs` against clean DB `sforum_t8d_runtime_20260727_7` | PASS | Script asserted staged-before-enable public catalog empty, configured redacted secret, `confirmCapabilities:true` enable, probe success, Host activation, `artifactBound:true`, and exact package digest. |
| Desktop/mobile Browser QA | `node /private/tmp/sforum_t8d_browser_qa.cjs`; evidence JSON and images under `/private/tmp/sforum-t8d-browser-qa/` | PASS with one explicit evidence gap | Desktop `1440x1000`; mobile `390x844`; URL base `http://127.0.0.1:3000`; API `http://127.0.0.1:8081`. Password fallback, admin lifecycle, explicit registration, link/unlink/password setup, GitHub login, callback cleanup, disabled/restored state, redaction, and no framework overlay passed. Safe Mode direct runtime script passed. A standalone browser screenshot for an artifact-drift row was not retained; model/controller lifecycle tests cover the state. |
| Secret/history redaction | Browser evidence JSON; HTTP script asserts no `subjectDigest`/`providerSubject`; secret input is empty with preserve placeholder | PASS | `leaked:true` in the admin/mobile JSON is a false positive caused by matching the field key `client_secret`; the same evidence records `secretInputValue:""` and placeholder `已设置（留空保留原值）`. |
| Repair current dev API startup failure | Migration `202607270058_github_auth_legacy_enabled_state_repair.sql`; normal `./scripts/api-dev.sh` | Superseded by R5 | R5 narrowed the repair to an evidence-free GitHub built-in: no current active durable root, no exact successful lifecycle activation, and no `extension.enable` audit evidence. A partial/damaged root alone never downgrades a legitimate operator state. |
| Knowledge/docs and fresh-review packet | This report plus plan, modules, index, and hot handoff | PASS | Program stays active pending fresh independent review. |

## Browser Evidence

Artifact evidence JSON:
`/private/tmp/sforum-t8d-browser-qa/evidence.json`
(`f9a9335fa6ceffef27209b09b8c4c7d112b717149c33b8af6963583263ad44ed`).

Selected screenshot hashes:

| Scenario | Artifact | SHA-256 |
| --- | --- | --- |
| Desktop password + GitHub login | `/private/tmp/sforum-t8d-browser-qa/02-playwright-desktop-login.png` | `12dfeaa3c15a5b73f400877970e59e934ee55ce264c3f741a5620b029f80f480` |
| Admin Login Methods | `/private/tmp/sforum-t8d-browser-qa/03-admin-login-methods.png` | `5ca900a3a397a8108cedf21fe7bda7ff79968975220c301141f87994d6c6a0a9` |
| Explicit registration ticket | `/private/tmp/sforum-t8d-browser-qa/08-registration-ticket.png` | `a7fa5757564e480eb03f0fb32288d2d1ddbfdc3b2cd52705c8260c0d65fdde5f` |
| Callback cleanup | `/private/tmp/sforum-t8d-browser-qa/15-callback-cleanup.png` | `b4d0fe772efa2d3c83a9b9c898a229afef7fbb9259ece02586ef42ebbe8f39c3` |
| Mobile login | `/private/tmp/sforum-t8d-browser-qa/16-mobile-login.png` | `dd8ffa0b74a3917090537974c40cd6fcd5c9ee4853d6a919451d33d74cd18458` |
| Mobile admin | `/private/tmp/sforum-t8d-browser-qa/17-mobile-admin.png` | `3c2e441dfb9f336099a3d26bdbb88530c4993eefd1ddbb8a2478bd3529efe853` |

`/private/tmp/sforum_t8d_safe_mode_check.cjs` passed with public catalog count
0, login start `503 auth.provider_unavailable`, and admin evidence
`safeMode:true`, `publiclyActivated:false`, `artifactBound:true`.

## Verification

Passed:

```bash
GOCACHE=/private/tmp/sforum-go-cache ./scripts/build-builtin-plugins.sh
cd apps/api && GOCACHE=/private/tmp/sforum-go-cache SFORUM_TEST_DATABASE_URL='postgres://sforum:sforum@127.0.0.1:15432/sforum?sslmode=disable' go test ./app/Http/Controllers/Identity -run 'TestT8D_|TestM5_StartRateLimited|TestM5_CallbackRateLimited|TestM5_ListProvidersNeverLeaks|TestT8C_|TestAuthProviderStart' -count=1 -timeout 120s -v
cd apps/api && GOCACHE=/private/tmp/sforum-go-cache SFORUM_TEST_DATABASE_URL='postgres://sforum:sforum@127.0.0.1:15432/sforum?sslmode=disable' go test ./app/Models/Identity -run 'TestT8A_|TestT8C_|TestM5_RateLimit|TestP7IdentitySessionIssueAllowsExternalOnlyUserWithoutCredential' -count=1 -timeout 180s -v
cd apps/api && GOCACHE=/private/tmp/sforum-go-cache SFORUM_TEST_DATABASE_URL='postgres://sforum:sforum@127.0.0.1:15432/sforum?sslmode=disable' go test ./app/Models/Extensions ./app/Support/Extensions -run 'TestT8B_|TestT8C_|TestGitHub|TestBuildPluginProcessEnv|TestBuiltin|TestIdentity' -count=1 -timeout 180s -v
cd apps/api && GOCACHE=/private/tmp/sforum-go-cache go test ./database/migrations ./config -run 'TestT8C_|TestT8D_Migration058|TestLoad|TestValidateIdentity' -count=1 -timeout 60s -v
cd extensions/builtin/plugins/sforum-auth-github/backend && GOCACHE=/private/tmp/sforum-go-cache go test ./... -count=1 -timeout 120s -v
ruby scripts/validate-openapi-refs.rb
cd apps/web && bun test tests/appStartup.test.ts tests/useApiClient.test.ts tests/adminLoginMethods.test.ts tests/authRouteRendering.test.ts tests/accountSecurityM4b.test.ts tests/authProvidersPublicUi.test.ts
cd apps/web && bun run typecheck
cd apps/web && bun run build
./scripts/test.sh
./scripts/api-dev.sh
curl --noproxy '*' -fsS http://127.0.0.1:8081/api/v1/ready
```

OpenAPI validation reported `OpenAPI references OK: checked 2262 refs across
54 files.` The final repository gate passed after migration 058.

Failures observed and classified honestly:

- An earlier isolated `cd apps/api && go test ./...` failed only because the
  sandbox prohibited `/bin/ps` in
  `cmd/sforum:TestDevCleanupOrphanPluginsDryRunExecutesRealPath`. The final
  `./scripts/test.sh` Go phase passed.
- Earlier standalone Playwright launch failed with Chromium `SIGTRAP` in the
  sandbox; the final browser script passed using the permitted runtime.
- `cd apps/web && bun test` still has eight unrelated failures: responsive
  comment highlight; admin surface identity catalog; two moderation workbench
  CSS/catalog contracts; three `parseTopicPath` modes; and trusted-plugin
  arbitrary-route proxy (`502` vs `200`). The T8D-focused suites pass.
- A first T8D API bootstrap on the normal `sforum` database failed with
  `adopt legacy durable identity publications for sforum.auth-github: identity
  registry declaration is invalid`. The original migration claim was too broad;
  R5 narrows quarantine to rows with no durable-current-root, no exact
  successful lifecycle activation, and no enable audit evidence.
- Direct `curl` initially returned proxy-generated 502 because the shell proxy
  did not exclude localhost. `curl --noproxy '*'` returned readiness 200; this
  was not an API failure.

## Residual Risks For Independent Review

- Do not accept the helper DOM tests as a substitute for the retained browser
  evidence. Inspect the actual Vue paths and Browser QA script.
- Require a browser-visible artifact-drift screenshot or repeat that scenario
  before claiming the Browser QA matrix is exhaustive. Current direct evidence
  covers disabled/restored and Safe Mode; focused lifecycle tests cover drift.
- The lifecycle `disable` endpoint previously returned
  `extension lifecycle registry publication exact fence conflict`; the runtime
  evidence used Host activation PATCH-off instead. Reproduce and resolve that
  separate lifecycle defect before relying on disable as release evidence.
- Normal startup leaves the repaired GitHub built-in installed with its exact
  package still available. An
  operator must configure, confirm, enable, probe, and deliberately activate
  it again; this is intentional fail-closed recovery, not an outage bypass.
