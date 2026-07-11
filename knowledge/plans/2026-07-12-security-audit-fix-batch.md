# 2026-07-12 Security Audit Fix Batch

> **Goal:** 在一次新对话中，按优先级修完 2026-07-12 全量安全审计的 P0–P2 项。  
> **工作区：** 直接在 `main` 上改。  
> **Git：** 每个逻辑修复单独 commit（见下方 commit 计划）；不要把无关重构塞进同批。  
> **禁止：** 不要 kill 用户的 web dev（端口 3000）；网络装包前设本地代理。

## Source audit

- Session that produced findings: 全量安全扫描（身份 / 论坛附件 / 扩展管理），非 local-diff review。
- Prior related: `knowledge/decisions/2026-07-09-security-audit.md`, `2026-07-09-security-fixes.md`.
- Handoff: `knowledge/sessions/2026-07-12-security-audit-handoff.md`.

## Working rules

1. 读本计划 + handoff 后直接实现，不要重新全量扫描 unless 修的时候发现结论过时。
2. 每修一项：改代码 → 补/改测试 → `cd apps/api && go test` 相关包 → **git commit**。
3. Commit message 用完整句子说明 why；中英文均可，与仓库近期风格一致即可。
4. 不 force-push；不改无关文件；需要迁移时用 Goose SQL 并说明 upgrade 路径。
5. 全部完成后：更新本 plan 勾选状态 + session handoff + `knowledge/index.md` Latest Handoff。
6. 收尾跑：`cd apps/api && go test ./...`（至少）；有合同变更再跑 `ruby scripts/validate-openapi-refs.rb`。

---

## Commit plan (execute in order)

### Commit 1 — Critical: stop `user.manage` expanding to permission overrides

**Severity:** Critical (privilege escalation)

**Problem:**  
`legacyPermissionChildren[PermissionUserManage]` 包含 `PermissionUserPermissionOverride`，使仅有 `user.manage` 的角色（含内置 **operator**）可通过 `ReplaceUserPermissionOverrides` 给他人任意 allow，与种子注释/产品意图矛盾。

**Files:**
- `apps/api/app/Models/Identity/permission_compat.go` — 从 `PermissionUserManage` 子列表移除 `PermissionUserPermissionOverride`（可保留 `PermissionUserView`）。
- `apps/api/app/Models/Identity/permission_compat.go` 或 `policy_test.go` / 新测试 — 断言：仅有 `user.manage` 时 `Can(user.permission_override)==false`；`ReplaceUserPermissionOverrides` 返回 `ErrPermissionDenied`。
- Migration（若 Phase1 已把 `user.permission_override` 物化给持有 `user.manage` 的角色/用户）：新增 Goose SQL，**删除**「仅因 user.manage 继承而写入、且非显式单独授予」的 role_permissions / user_permission_overrides 中的 `user.permission_override` 行。  
  - 参考：`apps/api/database/migrations/202607120001_fine_grained_permissions_phase1.sql`  
  - 安全策略：只移除通过兼容/迁移自动扩散的行；不要误删 super_admin 全权限语义（super_admin 是代码层全能，不依赖 catalog 行）。
  - 若难以精确区分「显式授予」vs「迁移扩散」，优先：从所有 **非 super_admin 角色** 的 role_permissions 去掉 `user.permission_override`，除非该角色 key 在 seeds 中明确应有该权限（当前只有应单独授权，builtin operator/moderator/tech_admin 都不应有）。

**Tests:**
- unit: expand map
- service: actor with only `user.manage` cannot ReplaceUserPermissionOverrides
- optional: actor with explicit `user.permission_override` still can

**Suggested commit message:**
```
fix(identity): stop user.manage from granting permission overrides

Legacy parent expansion let operators escalate via user.permission_override.
Remove the child link, clean migrated grants, and add regression tests.
```

---

### Commit 2 — High: super_admin membership changes only by super_admin

**Severity:** High

**Problem:**  
`ReplaceUserRoles` 阻止：改自己、摘初始 super_admin、**授予** super_admin。  
未阻止：非 super_admin 对**非初始** super_admin **移除** super_admin。

**Files:**
- `apps/api/app/Models/Identity/service.go` — 若 target 当前含 `RoleSuperAdmin` 且新集合不含（或任何改变 super_admin 成员身份），要求 `actor.IsSuperAdmin()`；可复用/扩展 `ErrSuperAdminGrantRestricted` 或新错误。
- `service_test.go` — operator 不能 demote 非初始 super_admin；super_admin 可以。

**Suggested commit message:**
```
fix(identity): require super_admin to alter super_admin role membership

Non-initial super_admins could be demoted by user.manage holders.
```

---

### Commit 3 — High: map password-reset token errors to 4xx

**Severity:** High (API correctness)

**Problem:**  
`ConfirmPasswordReset` → `ErrPasswordResetTokenNotFound` 未在 `mapIdentityError` 中映射 → 500 internal_error。

**Files:**
- `apps/api/app/Http/Controllers/Identity/controller.go` — map 到 422 或 401 + 稳定 reason（如 `auth.password_reset_invalid`）。
- 现有/新增 controller 测试。

**Suggested commit message:**
```
fix(identity): return 4xx for invalid password-reset tokens

Unmapped ErrPasswordResetTokenNotFound previously became 500.
```

---

### Commit 4 — High: strip spoofable plugin proxy headers

**Severity:** High

**Problem:**  
`RouteGateway.Proxy` 只 `Del("Cookie")`，未先删除客户端带来的 `X-SForum-Actor-ID` / Extension-ID / Locale；匿名 public 路由时客户端可伪造 Actor-ID。`Authorization` 等也会转发。

**Files:**
- `apps/api/app/Support/Extensions/route_gateway.go` — CopyTo 后强制 Del：`X-SForum-Actor-ID`, `X-SForum-Extension-ID`, `X-SForum-Locale`, `Authorization`, 以及合理的其它鉴权头；再只设置宿主权威值。
- `route_gateway_test.go` — 客户端伪造头不得出现在上游请求（除非宿主写入）。

**Suggested commit message:**
```
fix(extensions): strip client identity headers before plugin proxy

Always delete X-SForum-* and Authorization, then set host-authored values only.
```

---

### Commit 5 — High: constrain plugin RouteTarget BaseURL (SSRF)

**Severity:** High

**Problem:**  
插件 `RouteTarget().BaseURL` 任意 URL，API 代发请求 → 内网 SSRF。

**Files:**
- `apps/api/app/Support/Extensions/protocol.go`（或独立 validate 函数）— 解析 URL 后：
  - scheme 仅 `http`/`https`
  - 禁止 userinfo
  - host 解析后必须是 loopback（127.0.0.1 / ::1）或配置白名单（若暂无配置，至少 loopback-only）
  - 拒绝 link-local / metadata 常见地址
- 测试：合法 loopback 通过；`169.254.169.254`、内网 IP、`file:` 拒绝。

**Suggested commit message:**
```
fix(extensions): allow only loopback plugin route targets

Unvalidated BaseURL let plugins SSRF through the API host.
```

---

### Commit 6 — High: minimal env for plugin subprocess

**Severity:** High (secret blast radius)

**Problem:**  
`cmd.Env = os.Environ()` 把 DATABASE_URL、SESSION_HASH_SECRET 等全部传给插件。

**Files:**
- `apps/api/app/Support/Extensions/protocol.go` — 最小白名单（PATH, HOME/TMPDIR, LANG, 及 go-plugin handshake 需要的变量）+ 已有 `SFORUM_SETTING_*`。
- 测试：构造 env 时不包含 `DATABASE_URL`（可用可测 helper）。

**Note:** 若某 builtin 插件依赖额外 env，查 `extensions/builtin` 后显式加入，不要回退到全量 Environ。

**Suggested commit message:**
```
fix(extensions): start plugins with a minimal environment

Stop inheriting host secrets such as DATABASE_URL and session keys.
```

---

### Commit 7 — High: server-authoritative attachment MIME + active-content deny under wildcards

**Severity:** High (stored XSS via CDN)

**Problem:**  
`inspectUpload` 信任非空客户端 Content-Type；`mimeAllowed` 支持 `*/*` / `image/*` 时放行 `image/svg+xml`、`text/html`。API content 有 nosniff+强制下载，但 `decorateURL` 优先 CDN URL 绕过。

**Files:**
- `apps/api/app/Models/Attachments/service.go` — 始终 `DetectContentType`（或 sniff 优先）；allowlist 用探测结果；通配符下仍拒绝 `options.IsAttachmentActiveContentType`。
- `apps/api/app/Models/Options/attachment_options.go` — 配置层拒绝 `*/*` 或规范化时展开为安全默认集合；`mimeAllowed` 与 denylist 交叉。
- 测试：客户端声称 `image/png` 但 body 为 HTML/SVG → 拒绝；`image/*` 不接受 svg。

**Optional follow-up (same commit if small):** Content-Disposition 文件名 strip `\r\n`（controller）。

**Suggested commit message:**
```
fix(attachments): detect MIME server-side and block active types

Client MIME and wildcards could accept SVG/HTML served via public CDN URLs.
```

---

### Commit 8 — High: re-run publication policy on content edits

**Severity:** High (moderation bypass)

**Problem:**  
`CreateTopic`/`CreateComment` 调用 `publicationDecision`；`UpdateTopic`/`UpdateComment` 不调用，编辑可绕过预审。

**Files:**
- `apps/api/app/Models/Forum/service.go` — 内容变更时重跑 policy；触发则 status→pending（对齐 create 的计数/索引语义）或返回明确错误。
- 测试：先过审再编辑外链 → pending 或拒绝。

**Suggested commit message:**
```
fix(forum): re-evaluate publication policy on content edits

Edits previously skipped moderation gates used on create.
```

---

### Commit 9 — High/Medium: comment delete and topic count integrity

**Severity:** High (comment) / Medium (topic/category)

**Problem:**
- `DeleteComment` 不减 `topics.comment_count` / parent `reply_count` / category（对比 moderation workbench）。
- `DeleteTopic` 对从未 +1 的 pending 仍 -1；move category 无 status/`GREATEST`。

**Files:**
- `apps/api/app/Models/Forum/postgres_store.go` — DeleteComment 事务内递减（仅原 status active）；DeleteTopic/move 仅对 count-eligible status 调整，用 `GREATEST(...,0)`。
- 测试覆盖 delete comment 与 pending delete。

**Suggested commit message:**
```
fix(forum): keep topic and category counters correct on delete and move

Normal comment deletes never decremented counts; pending topic delete undercounted categories.
```

---

### Commit 10 — Medium: CSRF CookieSecure align with session

**Files:**
- `apps/api/app/Http/server.go` — CSRF `CookieSecure` 使用与 session 相同逻辑。  
  注意：`shouldUseSecureCookie` 在 `bootstrap`；可把 helper 抽到 `config` 包或在 `server` 内复制等价逻辑（prefer 抽到 config 避免循环依赖）。
- 现有 CSRF 测试仍绿。

**Suggested commit message:**
```
fix(http): set CSRF Secure cookie whenever session cookies are secure

Staging HTTPS previously left csrf_ readable over plain HTTP.
```

---

### Commit 11 — Medium: plugin disable permission parity + zip inflate cap

**Files:**
- `apps/api/app/Models/Extensions/lifecycle_operation.go` — DisableOperation 与 Enable 一样要求 plugin manage（+ release 若 web release）。
- `apps/api/app/Models/Extensions/service.go` — `readZipFile` 用 `io.LimitReader` + 累计真实字节，不信任 `UncompressedSize64`  alone。
- 测试。

**Suggested commit message:**
```
fix(extensions): align disable permissions and cap zip inflation

Release-only actors could disable trusted plugins; zip bombs trusted declared sizes.
```

---

### Commit 12 — Medium: password-reset rate limit + atomic confirm (if time)

**Optional same session if capacity remains:**

1. Redis 限流：按 email hash + IP（如 3/hour/email）。
2. Confirm：token consume + password update + token version + revoke sessions 同一事务。
3. 注册：HV 在冲突检查之前（或冲突在 HV 开启时返回通用错误）— 产品取舍，优先 HV-first for conflict only if tests exist.

**Suggested commit message(s):** split if both done.

---

## Explicitly out of scope for “one session finish” (document only)

除非用户新对话明确要求，否则本批**不做**完整实现：

| Item | Reason |
|------|--------|
| 非 builtin 插件 enable 强制 super_admin | 产品策略；可在 tech_admin 文档/注释标明 RCE 面 |
| 扩展 secret AES 落库 | 较大迁移；记为 follow-up |
| 扩展 settings PUT merge 语义 | 行为变更；可单独 PR |
| 主题 builder 最小 env | 对齐 commit 6 的 follow-up |
| 游客只读覆盖附件 | 产品：login_required 是否含媒体 |
| SoftDeleteVisibility / DuplicateTitlePolicy 落地 | 死配置清理或实现，另开任务 |
| 登录 lockout 改 IP 维度 | 产品权衡 |
| 缩短默认 session 30d/180d | 产品 |

若时间不够：保证 **Commit 1–9** 完成；10–12 可次优。

---

## Verification checklist

- [ ] `cd apps/api && go test ./app/Models/Identity/...`
- [ ] `cd apps/api && go test ./app/Support/Extensions/...`
- [ ] `cd apps/api && go test ./app/Models/Attachments/...`
- [ ] `cd apps/api && go test ./app/Models/Forum/...`
- [ ] `cd apps/api && go test ./app/Http/...`（CSRF / identity controller）
- [ ] `cd apps/api && go test ./...` before final handoff
- [ ] 每个 commit 独立、message 清楚、working tree 在收尾前 clean（或仅剩 knowledge 更新 commit）

---

## Paste-ready prompt for a new chat

见 handoff 文末「一句话启动」；也复制如下：

```
按 knowledge/plans/2026-07-12-security-audit-fix-batch.md 在 main 上按 commit 顺序修完安全审计 P0–P2（先读 knowledge/sessions/2026-07-12-security-audit-handoff.md）。直接改 main，每项单独 git commit，测相关 go test，做完更新 plan 勾选与 knowledge/index.md。不要杀 3000 端口的 web dev。
```
