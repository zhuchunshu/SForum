# 2026-07-12 Security Audit Follow-up Remediation

Status: **Ready for implementation**  
Date: 2026-07-12  
Source: second full-program static audit after the first P0-P2 security batch  
Previous batch: `knowledge/plans/2026-07-12-security-audit-fix-batch.md`

## Objective

Close the remaining trust-boundary, SSRF, access-control, secret-storage, and
configuration-correctness findings discovered after the first security batch.
Keep each fix independently reviewable and avoid unrelated refactors.

This plan is authoritative for the findings listed below. The previous security
plan is complete and must not be reopened unless a regression is demonstrated.

## Current baseline

- The prior identity, attachment MIME, plugin proxy, moderation, counter, CSRF,
  zip inflation, and password-reset fixes are present.
- No new unauthenticated SQL injection, path traversal, session fixation, or
  directly exploitable stored XSS was found in this audit.
- `cd apps/api && go test ./...` currently has one known failure:
  `TestAPIErrorCodesHaveLocalizedMessages` reports missing localization for
  `user.not_found`. All other Go packages passed during the audit.
- Dependency CVE scanning and active production penetration testing were not
  performed. Add them as a separate release-security gate rather than treating
  this static audit as exhaustive.

## Execution rules

1. Implement in the order below. P0 and P1 establish security boundaries that
   later work must preserve.
2. One logical fix per commit. Do not mix knowledge-only changes with unrelated
   product work.
3. Every unsafe route change must test both allowed and denied paths.
4. Any database encryption change needs a backwards-compatible read path and a
   documented migration/rotation strategy.
5. Do not kill the user-managed web dev server on port 3000.
6. Before network-dependent Go/Bun commands, export the proxy from `AGENTS.md`.
7. After each task, update its checkbox and record the commit hash in the
   completion table at the end of this document.

## Priority summary

| ID | Priority | Finding | Primary impact |
|----|----------|---------|----------------|
| P0.1 | Critical | Uploaded backend plugin can execute as host under `extension.plugin.manage` | Host compromise |
| P0.2 | High | Outbound webhook accepts internal/special targets and redirects | SSRF / metadata access |
| P1.1 | High | Forum login-required mode does not cover public attachment URLs | Content access bypass |
| P1.2 | Medium | Extension and webhook secrets are plaintext at rest | Credential disclosure |
| P1.3 | Medium | Login lockout is account-only and remotely triggerable | Targeted account DoS |
| P2.1 | Medium | Partial extension settings update deletes omitted values | Configuration loss |
| P2.2 | Medium | Several forum settings are stored but not enforced | Policy bypass / false UI |
| P2.3 | Medium | PAT authentication is not consistently consumed by controllers | API contract failure |
| P3.1 | Low | Webhook PATCH clears omitted description | Data loss |
| P3.2 | Low | `user.not_found` lacks API localization | Test/API message regression |

---

## Wave P0 - Host and network trust boundaries

### [ ] P0.1 Restrict execution of uploaded backend plugins

**Threat model**

`tech_admin` receives `extension.plugin.manage`. That permission can install and
enable an uploaded plugin, after which the backend entry is started with
`exec.CommandContext`. A plugin process can access the host filesystem, network,
and process namespace even though its inherited environment is minimized.
Capability confirmation does not sandbox a malicious executable.

**Required decision**

- Recommended: only an active `super_admin` may install, upgrade, enable, or
  verify a non-builtin plugin containing a backend entry.
- `extension.plugin.manage` may continue to configure, disable, and re-enable
  protected builtin plugins if product requirements need delegated operations.
- If delegated third-party enablement is retained, it must run in a separately
  isolated service/container before being considered safe. Documentation alone
  is not an adequate boundary.

**Implementation scope**

- Add a source/backend-aware policy helper in `Models/Extensions`; do not place
  the decision only in the controller.
- Apply it consistently to install, same-id upgrade, verify, enable, migration,
  and uninstall flows that can introduce or execute backend code.
- Keep theme-only packages and frontend-only plugins on their existing narrower
  permissions unless their build pipeline also executes package-controlled code.
- Update permission catalog/admin confirmation copy so `extension.plugin.manage`
  does not misleadingly imply that arbitrary backend execution is ordinary
  configuration.
- Record the final trust decision in `knowledge/decisions/`.

**Tests / acceptance**

- Non-super-admin `tech_admin` cannot install or enable a non-builtin backend
  plugin.
- The same actor can perform explicitly retained builtin operations.
- Super admin can install and enable a valid backend plugin.
- Frontend-only plugin/theme behavior remains unchanged.
- Denied attempts return a stable 403 reason and append an audit event without
  logging archive contents or secrets.

**Suggested commit:** `fix(extensions): restrict untrusted backend execution to super admins`

### [ ] P0.2 Make outbound webhook delivery SSRF-safe

**Threat model**

Endpoint validation currently accepts any `http`/`https` host. Delivery uses a
default `http.Client`, which follows redirects. This permits requests to
loopback, RFC1918, link-local, IPv6 local ranges, and cloud metadata endpoints.
Validation only at save time would still be vulnerable to DNS rebinding.

**Implementation scope**

- Prefer a mature SSRF-safe transport/resolver if one fits Go's `net/http`
  stack; otherwise keep the custom code small and independently tested.
- Validate scheme, userinfo, port, and every resolved IP at configuration time.
- Reject loopback, private, link-local, multicast, unspecified, documentation,
  and other non-public special-use ranges by default.
- Re-resolve and validate at connection time through a controlled `DialContext`
  to prevent DNS rebinding/TOCTOU.
- Add `CheckRedirect` that validates every redirect destination; cap redirect
  count and never forward the webhook signature to a different origin.
- Production should default to HTTPS. If HTTP is retained for development,
  make the exception explicit and environment/config driven.
- Return a generic validation error to the client; keep safe diagnostic detail
  in admin-visible delivery state.

**Tests / acceptance**

- Reject `127.0.0.1`, `::1`, RFC1918, `169.254.169.254`, link-local IPv6,
  userinfo URLs, and public-to-private redirects.
- Reject a hostname that resolves to any forbidden address.
- Connection-time validation blocks a DNS-rebinding test double.
- A public HTTPS endpoint still receives a signed request.
- Cross-origin redirects do not receive `X-SForum-Signature`.

**Suggested commit:** `fix(webhooks): block private targets and unsafe redirects`

---

## Wave P1 - Access control, credentials, and authentication abuse

### [ ] P1.1 Enforce forum read policy for referenced attachments

**Problem**

`forum.guest.read=login_required` protects forum JSON endpoints but not an
active attachment whose visibility is `public`. A user can share a copied post
attachment URL with an anonymous visitor. Avatars, logos, favicons, and SEO
images still need to remain genuinely public.

**Implementation scope**

- Define attachment purpose/reference semantics instead of applying the forum
  setting to every public image.
- Recommended: forum post attachments authorize through their active topic or
  comment reference; public site assets and avatars use an explicit public
  purpose.
- For remote storage/CDN providers, use short-lived signed URLs or proxy reads
  when authorization is required. A permanent public object URL cannot enforce
  session policy.
- Decide behavior for deleted/hidden/pending topics and private future category
  visibility; fail closed when reference resolution is unavailable.
- Update OpenAPI response/error documentation where the content route changes.

**Tests / acceptance**

- Anonymous access to a referenced forum attachment returns 401 in
  login-required mode and succeeds for an authenticated active user.
- Public mode remains backward compatible.
- Avatar/brand/SEO assets remain anonymously readable.
- Attachments referenced only by hidden, pending, or deleted content do not leak.
- Local and remote-provider URL decoration follow the same policy.

**Suggested commit:** `fix(attachments): enforce forum read policy on post media`

### [ ] P1.2 Encrypt extension and webhook secrets at rest

**Problem**

`extension_settings.value` and `webhook_endpoints.secret` are plaintext, while
core Options already has an AES-GCM `OptionCipher`. Database snapshots or
read-only database disclosure therefore expose provider credentials and webhook
signing keys.

**Implementation scope**

- Reuse the existing crypto package; do not create a second encryption format.
- Encrypt only manifest settings declared as `secret` and webhook secrets.
- Preserve masked API responses and `SecretSet`/`HasSecret` semantics.
- Support reading legacy plaintext during a migration window, then rewrite it
  transactionally to the versioned ciphertext format.
- Define startup behavior for missing/wrong keys. Production must fail closed;
  development may keep the existing documented transparent mode only if it is
  clearly marked non-secure.
- Document key rotation and backup restore requirements.

**Tests / acceptance**

- New secrets are not present as plaintext in database rows.
- Legacy plaintext reads successfully and is migrated without clearing values.
- Wrong-key/corrupt ciphertext never reaches a plugin or webhook request.
- Updating a non-secret setting does not rewrite or erase an existing secret.
- API/list/audit/log output never contains plaintext.

**Suggested commits:**

1. `feat(extensions): encrypt secret settings at rest`
2. `feat(webhooks): encrypt signing secrets at rest`

### [ ] P1.3 Replace account-only login locking with layered throttling

**Problem**

Failures are keyed only by normalized login. An attacker who knows a username
or email can repeatedly lock that account. The global write limiter slows the
attack but does not prevent targeted lockout.

**Implementation scope**

- Use separate Redis counters for source IP, account, and account+IP.
- Recommended behavior: low threshold slows one account+IP pair; higher IP
  threshold limits spraying; account-only state should trigger additional
  verification/backoff rather than an unconditional hard lock at the same low
  threshold.
- Keep login response messages non-enumerating.
- Add bounded TTLs and hashed account identifiers in Redis keys.
- Define Redis failure behavior explicitly; authentication should not become
  globally unavailable solely because a non-authoritative throttle store fails.

**Tests / acceptance**

- One source cannot brute-force or spray accounts cheaply.
- Failures from one IP do not hard-lock the victim for other trusted sources at
  the first threshold.
- Correct credentials plus required verification can recover according to the
  documented policy.
- Redis keys contain no raw email address.

**Suggested commit:** `fix(identity): make login throttling resistant to account lockout abuse`

---

## Wave P2 - Configuration and API correctness

### [ ] P2.1 Preserve omitted extension settings

**Problem**

`UpdateSettings` sanitizes only submitted keys, then `ReplaceSettings` deletes
all rows. A partial request silently removes every omitted non-secret setting.

**Implementation scope**

- Recommended contract: treat `PUT` as a complete form submission but resolve
  every manifest key to submitted value, current stored value, or default before
  replacement; omitted secrets always retain the current value.
- Alternative: change to PATCH merge semantics and update OpenAPI/frontend.
- Validate the complete candidate set before beginning the transaction.
- If plugin restart fails, restore previous settings or make persistence and
  restart state explicit and retryable.

**Tests / acceptance**

- Updating one field preserves all omitted values and secrets.
- Explicit reset remains the only path that removes stored overrides.
- Invalid input changes nothing.
- Restart failure has deterministic rollback/recovery behavior.

**Suggested commit:** `fix(extensions): preserve omitted settings during updates`

### [ ] P2.2 Implement or remove dead forum policy controls

**Affected controls**

- `allowAuthorCloseReplies`
- `autoLockIdleDays`
- `duplicateTitlePolicy`
- `showTopicEditMark`
- `showCommentEditMark`
- `softDeleteVisibility`
- `mentionsEnabled`

**Required behavior**

- `mentionsEnabled=false`: do not populate mentioned usernames and do not send
  mention notifications; ordinary `@text` remains content.
- `allowAuthorCloseReplies`: author lock/unlock permission must combine topic
  ownership, the setting, and a stable author permission; moderators retain
  `topic.lock` authority.
- `duplicateTitlePolicy`: `block` must be server-authoritative; `warn` needs a
  clear response/UI warning flow or should not be offered yet.
- `autoLockIdleDays`: add a registered durable schedule, or remove/hide the
  setting until implemented.
- Edit-mark and soft-delete visibility must be enforced in API presentation,
  not only in one theme. Do not leak deleted comment content in list/reply APIs.

**Tests / acceptance**

- Each visible admin control has a behavior test proving both setting states.
- No frontend-only security or privacy policy.
- Restore recommended defaults is covered.
- Deferred controls are removed from admin/public contracts instead of staying
  as no-op switches.

**Suggested commits:** split by behavior (`mentions`, `topic lifecycle`,
`duplicate titles`, `presentation/deleted visibility`, `auto-lock schedule`).

### [ ] P2.3 Make PAT authentication consistent across API controllers

**Problem**

The Bearer middleware stores authenticated PAT identity/scopes in context, but
most controllers call `AuthSession.CurrentUserID` directly. Valid scoped PATs
therefore work on Identity routes but return 401 on many forum/admin routes.

**Implementation scope**

- Add or reuse one HTTP-layer actor resolver that prefers a valid PAT, otherwise
  uses the cookie session, loads the current active actor, and intersects current
  permissions with token scopes.
- Migrate controllers to the shared resolver; avoid controller-specific copies.
- Keep optional/public actor resolution distinct from required actor resolution.
- Disabled/deleted users and revoked/expired tokens must fail immediately.
- Document which endpoints intentionally disallow PAT, if any.

**Tests / acceptance**

- A PAT works on an allowed non-Identity endpoint.
- Missing scope returns 403 even when the user currently has the permission.
- Permission revoked from the user after token creation takes effect immediately.
- Cookie-only behavior and CSRF protection remain unchanged.
- Invalid `Bearer sft_...` cannot use the CSRF exemption to reach a mutation.

**Suggested commit:** `fix(http): resolve scoped PAT actors consistently across controllers`

---

## Wave P3 - Small correctness fixes and release gates

### [ ] P3.1 Preserve omitted webhook description

- Decode PATCH presence explicitly (pointer field or raw body key check).
- Omitted description preserves the current value; explicit empty string clears
  it.
- Add controller and store/service regression tests.

**Suggested commit:** `fix(webhooks): preserve omitted fields during endpoint updates`

### [ ] P3.2 Restore API localization test baseline

- Add zh-CN and en-US messages for `user.not_found`.
- Verify the message does not expose sensitive identity information on anonymous
  routes; current use is an authenticated admin operation.
- Run the focused localization test and full Go suite.

**Suggested commit:** `fix(localization): add the missing user not found message`

### [ ] P3.3 Add dependency and active-security release checks

This was outside the static audit and should be scheduled before production:

- Run Go vulnerability scanning against the resolved module graph.
- Run Bun/npm advisory scanning against the lockfile.
- Review container base-image CVEs and verify the final runtime user.
- Exercise webhook SSRF cases against a controlled local environment.
- Exercise attachment authorization through local and remote storage providers.
- Record tool versions, advisory database date, accepted risks, and false
  positives in a dated security report.

Do not add a dependency solely for a one-off scan unless it will become a
maintained CI/release gate.

---

## Cross-cutting verification matrix

| Area | Required verification |
|------|-----------------------|
| Extensions | Models/Extensions, Support/Extensions, controller permissions, lifecycle tests |
| Webhooks | Models/Webhooks, Jobs/Webhooks, controllers, DNS/redirect test doubles |
| Attachments | Models/Attachments, controllers, local and signed remote URL behavior |
| Identity | Models/Identity, Support/Auth, login controller tests |
| Forum | Models/Forum, controllers, notification fanout, admin settings UI validation |
| PAT | Bearer middleware plus forum, attachment, extension, and jobs controllers |
| Contracts | OpenAPI refs and request/response schemas after endpoint changes |

Final commands:

```sh
cd apps/api && go test ./...
ruby scripts/validate-openapi-refs.rb
cd apps/web && bun run typecheck
./scripts/test.sh
```

Run focused packages after each commit. Run the full gate after the known
`user.not_found` localization baseline has been fixed.

## Completion table

| Task | Status | Commit | Notes |
|------|--------|--------|-------|
| P0.1 plugin execution boundary | Pending | | Decision record required |
| P0.2 webhook SSRF | Pending | | |
| P1.1 attachment read policy | Pending | | Contract change likely |
| P1.2 secrets at rest | Pending | | Migration and key rotation docs |
| P1.3 login throttling | Pending | | Product/security behavior change |
| P2.1 extension setting preservation | Pending | | |
| P2.2 forum policy controls | Pending | | Split into focused commits |
| P2.3 PAT consistency | Pending | | Shared resolver preferred |
| P3.1 webhook PATCH | Pending | | |
| P3.2 localization baseline | Pending | | Current Go test blocker |
| P3.3 release security scans | Pending | | Dated report required |

## Paste-ready implementation prompt

```text
按 knowledge/plans/2026-07-12-security-audit-followup-remediation.md 从 P0 开始实施。先读 knowledge/index.md、相关 module notes 和 knowledge/sessions/2026-07-12-security-audit-followup-plan.md。每个逻辑修复单独 commit，补允许/拒绝测试，逐项更新计划完成表；不要重做已经完成的 2026-07-12-security-audit-fix-batch，也不要杀 3000 端口的 web dev。
```
