# 2026-07-12 Security Audit Handoff

## Context

- User asked for a full-program bug/security scan (not local-diff review).
- Three parallel audits: identity/auth, forum/attachments, extensions/admin.
- User asked to **record findings + task plan** so a **new chat can finish fixes on `main` with good git commits**.
- **No code fixes were applied in the audit session** — only knowledge/docs.

## Authoritative plan

**`knowledge/plans/2026-07-12-security-audit-fix-batch.md`**

Execute that plan top-to-bottom. Do not re-audit the whole repo unless a finding is obsolete.

## Findings summary (priority)

### Critical

1. **`user.manage` → `user.permission_override` via `permission_compat.go`**  
   Operator (and any `user.manage` holder) can escalate via permission overrides.  
   Contradicts operator seed comment and Phase1 intent.

### High

2. Non–super-admin can **demote non-initial super_admin** (`ReplaceUserRoles`).
3. Password-reset invalid token → **500** (`mapIdentityError` missing case).
4. Plugin proxy: **spoofable `X-SForum-Actor-ID`** if not host-set; forward `Authorization`.
5. Plugin **RouteTarget BaseURL unrestricted** → SSRF from API host.
6. Plugin **`os.Environ()`** inherits host secrets.
7. Enabling uploaded backend plugin = **host RCE** under `extension.plugin.manage` (design; document / dual-control later).
8. Attachment **client MIME + wildcards** + CDN URL bypasses content-handler XSS defenses.
9. **Content edit skips `publicationDecision`** (moderation bypass).
10. **`DeleteComment` does not decrement counters**.

### Medium (in plan commits 9–12)

- Category/topic count bugs on pending delete / move.
- CSRF `CookieSecure` only checks `production`, session uses `shouldUseSecureCookie`.
- Disable plugin permission weaker than enable.
- Zip bomb: trust `UncompressedSize64` + unbounded `ReadAll`.
- Password-reset flood / non-atomic confirm.
- Register conflict before HV; login lockout account DoS; extension secrets plaintext; partial settings wipe; guest-read not on attachments; dead forum policy flags.

### Good baseline (do not regress)

- Session Reset/Regenerate, HttpOnly, token version on password reset.
- Argon2id + dummy hash timing; CSRF double-submit + Origin.
- bluemonday + frontend DOMPurify; path traversal hardening on storage/packages.
- Parameterized SQL; DB browser read-only + sanitized identifiers.
- Production secret panic on placeholder defaults.

## Key file map

| Area | Paths |
|------|--------|
| Permission expand | `apps/api/app/Models/Identity/permission_compat.go`, `policy.go`, `service.go` ReplaceUser* |
| Password reset | `password_reset_service.go`, `Identity/controller.go` mapIdentityError |
| CSRF / session secure | `app/Http/server.go`, `bootstrap/app.go` shouldUseSecureCookie |
| Plugin proxy | `Support/Extensions/route_gateway.go`, `protocol.go`, `Providers/extensions.go` |
| Attachments MIME | `Models/Attachments/service.go`, `Options/attachment_options.go`, content disposition in Attachments controller |
| Forum policy / counts | `Models/Forum/service.go`, `postgres_store.go`, moderation `workbench_store.go` for reference |
| Lifecycle disable | `Models/Extensions/lifecycle_operation.go`, `readArchive` in `service.go` |

## Git / process constraints from user

- Work **directly on `main`**.
- **Use good commits**: one logical fix per commit (plan lists messages).
- Prefer finishing P0–P2 in one session; plan marks optional deferrals.
- Do not kill user web on port 3000.
- Set China proxy before `go get` / `bun` network commands (see Agents.md).

## Next (for implementer)

1. Open plan file; start Commit 1.
2. After all commits: `go test ./...` under `apps/api`.
3. Update plan checkboxes + this handoff “Done” section + `knowledge/index.md` Latest Handoff.
4. Optional: short decision note if product choices made (e.g. loopback-only plugin targets).

## Done (fill when complete)

- [ ] Commits 1–9 (must)
- [ ] Commits 10–12 (should)
- [ ] Tests green
- [ ] knowledge/index.md updated

## One-line starter for new chat

```
按 knowledge/plans/2026-07-12-security-audit-fix-batch.md 在 main 上按 commit 顺序修完安全审计 P0–P2（先读 knowledge/sessions/2026-07-12-security-audit-handoff.md）。直接改 main，每项单独 git commit，测相关 go test，做完更新 plan 勾选与 knowledge/index.md。不要杀 3000 端口的 web dev。
```
