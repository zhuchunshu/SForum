# 2026-07-27 GitHub Social Login Independent Re-review Handoff

## Status

T8D implementation and evidence collection are complete enough for a fresh
independent Codex review. The GitHub social-login program is still **active**:
this handoff does not declare it closed. Do not start another provider or a
new program from this handoff.

The authoritative requirements/evidence packet is:
`../reports/2026-07-27-github-social-login-t8d-requirements-matrix.md`.

## Changed In T8D

- Critical admin/public/account source-text checks were replaced with
  happy-dom interaction regressions; retained real Playwright evidence covers
  the actual `/login`, `/register`, `/settings/security`, and
  `/control-panel/settings/login-methods` flows.
- HTTP rate-limit regressions now fail unless the start endpoint returns 429
  with `rate_limit.exceeded`; callback limit continues to assert safe redirect
  and redaction.
- External login/register now call Host risk evaluation with canonical
  `login` / `register` purposes. Regression coverage also proves an
  external-only user can receive a session without a credential row.
- The GitHub built-in was rebuilt; runtime package digest is
  `5d73651dd4013bc04abeb6f99f9ef0686303ee683c40d9a483f927f6b5c09942`.
  Lifecycle evidence used normal settings, enable, probe, and Host activation
  APIs on `sforum_t8d_runtime_20260727_7`.
- Migration `202607270058_github_auth_legacy_enabled_state_repair.sql` fixes
  a real normal-dev startup blocker. R5 narrowed it to an evidence-free
  `sforum.auth-github` built-in: no current active durable root, no exact
  successful lifecycle activation, and no `extension.enable` audit record.
  A partial/damaged root alone cannot downgrade a legitimate operator state.
  It preserves immutable packages and never auto-adopts or restores executable
  state. The current `sforum` DB
  then started via `./scripts/api-dev.sh`; direct
  `curl --noproxy '*' http://127.0.0.1:8081/api/v1/ready` returned 200.

## Key Evidence

- Browser script: `/private/tmp/sforum_t8d_browser_qa.cjs`
- Browser JSON: `/private/tmp/sforum-t8d-browser-qa/evidence.json`
- Browser images: `/private/tmp/sforum-t8d-browser-qa/`
- Lifecycle HTTP script: `/private/tmp/sforum_t8d_http_flows.mjs`
- Safe Mode script: `/private/tmp/sforum_t8d_safe_mode_check.cjs`
- Final repo gate: `./scripts/test.sh` passed after migration 058.

Desktop Browser QA used `1440x1000`; mobile used `390x844`. It covered password
fallback, settings redaction, disabled/restored public catalog, explicit
registration ticket, external-only password setup, link/unlink, callback query
cleanup, GitHub login, and mobile public/admin rendering. Safe Mode direct API
evidence had public catalog count 0 and login start 503.

## Review Focus And Remaining Risks

- Treat helper DOM tests as regression support only. Verify the browser script
  and rendered components; source-text inspection is not behavioral evidence.
- No standalone artifact-drift browser screenshot was retained. Focused
  lifecycle/controller tests cover artifact drift, but a reviewer should
  require that visual proof before asserting exhaustive Browser QA.
- The extension lifecycle `disable` endpoint previously produced
  `extension lifecycle registry publication exact fence conflict`; runtime
  evidence intentionally used Host activation PATCH-off. Reproduce that issue
  independently and keep it open unless actually fixed.
- Full `bun test` has eight unrelated known failures; the focused T8D suites,
  typecheck, build, OpenAPI validation, and final repository gate passed. The
  matrix records all failure classifications, including the earlier sandbox
  `/bin/ps` restriction and a Chromium sandbox SIGTRAP.

## Next

Perform one independent code/runtime review only. Accept or reject T8D with
findings grounded in code and reproduced evidence. Do not implement another
provider, do not start another program, and do not self-declare program closure
without resolving or explicitly accepting the listed risks.
