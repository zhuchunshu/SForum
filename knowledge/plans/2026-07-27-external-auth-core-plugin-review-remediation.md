# External Auth Core And Provider Plugin Review Remediation

Status: **R1-R7 remediation complete; independent re-review requested**. The
2026-07-27 independent review rejected the T8D evidence packet. This task book
records the complete remediation of the external-auth framework, not only the
GitHub adapter and not only the T8D tests.

Date: 2026-07-27

## Goal

Close the independent-review findings across:

- Host-owned external-auth orchestration and security effects;
- the generic multi-provider Identity Registry and activation model;
- executable provider-plugin lifecycle;
- the protected built-in GitHub reference plugin;
- public/admin/account Vue rendering;
- migration, redaction, PostgreSQL, runtime, and Browser evidence.

Do not add another provider. GitHub remains the only executable reference
plugin for this remediation.

## Task Ledger

- [x] R1 - External login effect fence
- [x] R2 - Transactional registration policy
- [x] R3 - Audit and history redaction
- [x] R4 - Exact extension disable lifecycle
- [x] R5 - Migration 058 truthfulness
- [x] R6 - Real Vue and Browser evidence
- [x] R7 - Final integrated verification (isolated API + Nuxt runtime packet)

## Required Reading

Read these files completely before editing:

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/identity.md`
4. `knowledge/modules/extensions.md`
5. `knowledge/plans/2026-07-27-github-social-login-builtin-plugin.md`
6. This task book
7. `knowledge/decisions/2026-07-27-github-social-login-builtin-v1.md`
8. `knowledge/decisions/2026-07-27-github-social-login-m0-contract-freeze.md`
9. `knowledge/sessions/2026-07-27-external-auth-review-remediation-handoff.md`
10. `knowledge/reports/2026-07-27-github-social-login-t8d-requirements-matrix.md`

## Working Rules

- One remediation conversation executes R1 through R7 sequentially. It may
  start the next task only after the current task's focused tests, checkpoint
  report, task-ledger update, and knowledge handoff update are complete.
- Do not stop after an intermediate task merely to request a new conversation.
  Continue through R7 unless a real blocker remains after reasonable
  investigation. Do not mark later tasks complete to work around a failed
  earlier exit criterion.
- Preserve all unrelated dirty-worktree changes. Do not commit, push, revert,
  or kill the user-owned Nuxt process on port 3000.
- Start with a short library/framework survey for any new test or transaction
  mechanism. Prefer pgx/PostgreSQL and existing Nuxt/Vue test tooling.
- Tests must call production entry points. A helper that reimplements the
  controller, Vue component, or lifecycle is not acceptance evidence.
- Every denied path must prove zero user/link/session/runtime side effects.
- Provider-specific OAuth behavior remains inside the protected built-in.
  Core must not branch on `github`, extension ID substrings, or vendor labels.
- Do not weaken exact-artifact, actor-bound trust, Safe Mode, recent-auth,
  password fallback, rate limits, callback consumption, or secret redaction.

## R1 - External Login Effect Fence

Fix the callback-to-session TOCTOU window.

- After the provider `complete` RPC returns, revalidate Host activation,
  operation, exact live contribution, owner extension/version/package digest,
  provider contract, and Safe Mode before resolving the linked account.
- Revalidate at the Host session-effect boundary, after risk evaluation and
  immediately before the pending session is persisted. Define and document the
  linearization point; do not claim impossible cross-store atomicity.
- Put the reusable validation in the identity service. The controller must not
  recreate Registry matching rules.
- Add a deterministic concurrency test with a blocking fake provider/risk
  hook: disable, artifact replacement, trust revoke, or Safe Mode entered
  while the callback is in flight must prevent session persistence.
- Preserve generic unlinked-account behavior and password login fallback.

Exit: no session is issued from a callback whose authorization is invalid at
the documented Host login-effect fence.

## R2 - Transactional Registration Policy

Make the registration-policy claim truthful.

- Read the authoritative registration option through the same `pgx.Tx` used
  for user, default role, external link, and audit writes.
- Use an existing Options store/query where possible. If concurrent policy
  updates must be serialized, lock or revision-fence the authoritative option
  row through a documented PostgreSQL mechanism.
- Do not call an independently pooled `RegistrationEnabled` dependency from
  `ensureExternalRegistrationPolicyTx`.
- Add real PostgreSQL concurrency tests that pause registration after the fast
  check, close registration, then prove the specified ordering and zero
  effects. Also cover bootstrap/no-user behavior and rollback.

Exit: the policy read and account mutation have a defensible, tested database
transaction boundary.

## R3 - Audit And History Redaction

Remove prohibited digest material.

- Stop writing `ownerPackageDigest`, subject digest, raw subject, state,
  verifier, token, code, or secret into external-registration audit metadata.
- Keep only bounded identifiers required for operator audit, such as provider
  ID, owner extension ID, correlation ID, actor/target, and action.
- Add a narrowly scoped forward migration that removes
  `ownerPackageDigest` from existing `auth.external_register.success`
  metadata without changing unrelated audit events or immutable extension
  artifact history.
- Add real PostgreSQL tests for new writes and historical cleanup.
- Recheck API, logs, probe diagnostics, linked-account history, and Browser
  responses for both field names and values.

Exit: code and existing rows satisfy the documented audit-redaction contract.

## R4 - Exact Extension Disable Lifecycle

Reproduce and repair the executable provider disable path.

- Use an isolated database and runtime; do not mutate the user's normal
  development database.
- Build, stage, trust, enable, configure, probe, and activate the exact GitHub
  built-in through normal APIs or the same production services.
- Reproduce the prior `extension lifecycle registry publication exact fence
  conflict` from the real extension disable path.
- Fix the underlying lifecycle/Identity Registry publication transition. Do
  not replace extension disable with provider activation PATCH-off.
- Prove disable drains/stops the exact runtime, removes its live Identity
  publication and public catalog entry, preserves settings/secrets/history,
  blocks start/callback effects, and leaves password login available.
- Add a PostgreSQL/service/controller integration regression for the exact
  GitHub-shaped identity publication and a generic second fixture shape.

Exit: the normal extension disable endpoint succeeds and the provider becomes
inert without an exact-fence conflict or collateral provider removal.

## R5 - Migration 058 Truthfulness

Make legacy repair conditions and documentation agree.

- Decide from repository evidence whether migration 058 should be narrowed,
  replaced by an explicit recovery operation, or removed before merge.
- It must never downgrade a legitimate operator-enabled built-in merely
  because one durable publication row is missing or damaged.
- If the migration remains, encode every claimed evidence predicate in SQL.
  Do not call a state "unaudited" without checking the authoritative lifecycle
  operation/audit records.
- Keep Down fail-closed: never auto-enable executable code.
- Replace the SQL substring-only test with real PostgreSQL scenario tests for
  stale pre-T8D state, legitimate enable, partial/corrupt publication,
  unrelated plugins, wrong source/type, and rerun idempotency.
- Remove or correct all overclaims in the plan, handoff, report, and docs.

Exit: migration behavior is narrow, executable, idempotent, and described
exactly as implemented.

## R6 - Real Vue And Browser Evidence

Replace evidence that can pass while production Vue is broken.

- Survey the existing Nuxt/Vue tooling before adding dependencies.
- Mount/import the actual `login-methods.vue`, `SFLoginFormPage.vue`,
  `SFRegisterFormPage.vue`, `SFSecuritySettingsPage.vue`, and their real child
  components/composables. Do not build parallel DOM harnesses.
- Cover two catalog-driven providers in rendered tests: ordering, independent
  activation, generic label/icon rendering, one provider failing without
  hiding the other, and no vendor branch in Core.
- Turn every Browser QA result into a hard assertion, including provider
  button presence, probe execution, lifecycle badges, stable callback reason,
  console errors, overlays, and secret/history redaction.
- Replace the broad field-name regex that reports false `leaked:true` values
  with separate assertions for forbidden values and forbidden public fields.
- Add a browser-visible artifact-drift scenario and screenshot.
- Keep reproducible Browser/HTTP scripts in the repository. The report must
  list exact commands, database, URLs, viewports, artifact/package/schema
  digests, JSON, screenshots, and hashes.

Exit: breaking an actual critical Vue path or returning a secret makes the
automated evidence fail.

## R7 - Final Integrated Verification

Do not implement new behavior in this task. Verify the complete framework.

- Run focused identity/controller/extensions/Identity Registry/migration tests,
  GitHub backend tests, OpenAPI refs, relevant real Vue tests, Nuxt typecheck
  and build, then `./scripts/test.sh`.
- Start the normal API with `./scripts/api-dev.sh` without touching port 3000;
  verify readiness.
- In an isolated runtime, execute password fallback, admin lifecycle,
  configure/probe/activate, login, explicit registration, link/unlink/password
  setup, callback cleanup/replay, provider disable, artifact drift, Safe Mode,
  and redaction.
- Verify package, executable, schema, lifecycle-script, Browser JSON, and
  screenshot digests agree.
- Produce a replacement requirements matrix. List every failure and distinguish
  code failures from environment failures.
- Update the original GitHub task book and knowledge status to
  `independent re-review requested`; do not self-declare the program closed.

Exit: a fresh reviewer can independently accept or reject the whole
external-auth Core/plugin system from reproducible evidence.

## Required Checkpoint Report

At the end of every R1-R7 task, within the same remediation conversation:

1. Update this task's checkbox/status in this plan.
2. Update `knowledge/modules/identity.md` and/or `extensions.md` only for facts
   that changed.
3. Replace the single hot remediation handoff; update `knowledge/index.md`.
4. Report changed files, security/architecture impact, exact commands/results,
   remaining risks, and the next task ID.
5. Emit a concise checkpoint report before proceeding to the next task. The
   final R7 report must summarize all prior checkpoints and provide the exact
   handoff text for independent review.

## Final Review Acceptance

The remediation is not complete until an independent reviewer confirms:

- the four P1 findings are fixed with production-path tests;
- extension disable works through the real lifecycle;
- migration 058 is truthful and narrow;
- critical tests mount actual Vue;
- Browser facts are hard assertions with artifact-drift evidence;
- generic multi-provider behavior remains intact;
- the GitHub adapter remains vendor-isolated and Host identity authority is
  unchanged.
