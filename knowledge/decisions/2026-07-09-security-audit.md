# 2026-07-09 安全审计与修复待办清单

## 背景

- **审计范围**：SForum 全栈（约 4 万行 Go/Fiber API + Nuxt 前端 + Docker 部署 + OpenAPI 契约）。
- **审计方法**：分认证授权、数据访问、文件上传/存储、前端 XSS、配置部署五大领域，并行深入审计；所有问题均基于实际读取的代码定位（非猜测），并标注 `文件:行号`。
- **状态约定**：`[ ]` 待办，`[x]` 已修复。修复后请勾选并在对应项追加 PR/commit 链接与验证结果。

---

## 🔴 严重（Critical）— 建议立即修复

### [x] C1. 扩展 ZIP 通过 `manifest.Version` 路径穿越 → 任意文件写入 / RCE
- **位置**：`apps/api/app/Support/ExtensionManifest/manifest.go:251`（`Validate` 仅 `TrimSpace`，未约束 `Version`）；`apps/api/app/Models/Extensions/service.go:436`（`versionDir` 拼接）、`:444`、`:447`、`:853`；放大因素 `service.go:831-835`（ZIP 条目 mode 原样保留 → 可执行位）。
- **问题**：`manifest.ID` 有正则 `^[a-z0-9][a-z0-9._-]{1,80}$` 约束，但 `Version` 没有。`filepath.Join(extensionRoot, id, version)` 会把 `"../../../../tmp/evil"` 标准化，使解压文件落到 `extensionRoot` 之外。内部 `HasPrefix(target, root+sep)` 防御因 root 本身被穿越而失效。
- **修复方向**：在 `Validate` 中将 `Version` 约束为严格模式（如 `^[a-zA-Z0-9][a-zA-Z0-9._+\-~]{0,63}$`，禁止 `/`、`\`、纯 `.`/`..`、绝对路径）；或对 `path.Clean(version)` 后拒绝 `../` 前缀。同时把常规文件写盘 mode 掩码为 `0644`，仅对验证后的 backend entry 恢复可执行位。

### [x] C2. 密钥硬编码弱默认值，生产无强制校验
- **位置**：`apps/api/config/config.go:129`（`SessionHashSecret` 默认 `sforum-dev-session-hash-secret`）、`:131`（`ALTCHA_SECRET` 默认 `sforum-dev-altcha-secret`）、`:143`（`MEILI_MASTER_KEY` 默认 `sforum-dev-meili-key`）；缺校验处 `apps/api/cmd/api/main.go`。
- **问题**：这些 secret 是会话伪造/HMAC/搜索权限的核心。启动链路无 `AppEnv=="production"` 时是否被覆盖的校验；`.env.production.example:48` 是占位符 `change-me`，`.env.example` 甚至没有 `SESSION_HASH_SECRET`。运维忘记配置 → 生产静默回退到公开默认值。
- **修复方向**：生产启动路径（或 `config.Load()` 内当 `AppEnv=="production"`）对上述 secret 做非空 + 非占位词校验，缺失即拒绝启动；敏感项改为无默认值、强制显式提供。

---

## 🟠 高危（High）

### [x] H1. `user.manage` 权限可自我提权到 super_admin
- **位置**：`apps/api/app/Models/Identity/service.go:377-396`（`ReplaceUserRoles`）、`:398-423`（`ReplaceUserPermissionOverrides`）；`validateRoles` 在 `service.go:506-524`。
- **问题**：两方法只检查 `actor.Can(PermissionUserManage)`，未校验 `actor.ID == targetUserID`，也未禁止把 super_admin 角色授予非 super_admin actor。拥有单一 `user.manage` 权限的用户可给自己加 super_admin 角色 → 完全提权。仅对"初始 super_admin"有保护（`service.go:388-390`）。
- **修复方向**：禁止将 super_admin 角色授予非 super_admin actor；禁止 actor 操作自身角色/权限（或限制为"只能减不能增"）；授予 super_admin 要求 actor 本身已是 super_admin。

### [x] H2. FTP/SFTP 凭证明文存储 + SFTP 主机密钥默认不校验（MITM）
- **位置**：`apps/api/app/Models/Options/postgres_store.go:55-67`（原样写入 `web_options.value`）；`app/Models/Options/service.go:1069-1070`（仅长度限制，未加密）、`:349-351`（`InternalValues` 返回原始值）；`app/Support/Storage/sftp.go:192-196`（`HostKeyFingerprint` 为空时返回 `ssh.InsecureIgnoreHostKey()`）。
- **问题**：阿里云 AccessKeySecret、腾讯云 SecretKey、FTP/SFTP 密码、SSH 私钥均明文存于 DB，`secret:true` 只控制 API 响应掩码。数据库泄露 = 云存储 + SSH 全面沦陷。SFTP 默认不校验主机密钥 → MITM 可截获凭证。
- **修复方向**：密钥选项用 AEAD（AES-256-GCM）静态加密，密钥来自 `APP_OPTION_ENC_KEY`，仅在 `InternalValues` 解密；SFTP 强制要求主机密钥指纹，缺失时拒绝连接而非回退 `InsecureIgnoreHostKey`。

### [x] H3. 评论/主题正文 `v-html` 单点失败（无前端二次净化）
- **位置**：`apps/web/app/components/SFComment.vue:106`；`extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue:661`；注释见 `SFComment.vue:50`。
- **问题**：评论/主题正文 HTML 直接 `v-html` 渲染，完全依赖后端 bluemonday 净化，前端无 DOMPurify 二次净化。后端策略遗漏、历史数据未重洗、或存在旁路写入即为存储型 XSS；且认证走 cookie，XSS 可直接劫持管理员会话。
- **修复方向**：前端 `v-html` 前加 DOMPurify 二次净化（白名单与服务端对齐）；服务端改为读取时净化（read-time sanitize），覆盖历史数据。

### [x] H4. 无 CORS / 无任何安全响应头
- **位置**：`apps/api/app/Http/server.go:38-78`（无 cors 中间件、无安全头）；`deploy/caddy/Caddyfile`（仅 4 行 `reverse_proxy + encode`）。
- **问题**：缺失 `X-Content-Type-Options`、`X-Frame-Options`、`Strict-Transport-Security`、`Content-Security-Policy`、`Referrer-Policy`。点击劫持、MIME 嗅探、降级攻击无浏览器层防护。
- **修复方向**：Caddyfile 加 `header { HSTS; X-Content-Type-Options nosniff; X-Frame-Options DENY; Referrer-Policy strict-origin-when-cross-origin }`，CSP 按 Nuxt 实际需要配置；若未来 API 需跨域，用 fiber cors 显式白名单域名（绝不配合 `AllowOrigins:"*"` + credentials）。

### [x] H5. Cookie 认证但无 CSRF 防护
- **位置**：`apps/web/app/composables/useApiClient.ts:61`（`credentials:'include'`）；全仓无 CSRF token / `Origin` 校验。
- **问题**：会话通过 cookie 维持，所有写操作（发帖/删帖/改设置/登出）仅凭 cookie 鉴权。`SameSite=Lax` 能挡大部分 CSRF，但 POST 跨站仍有残余风险，配合跨域 `credentials:'include'` 风险更大。
- **修复方向**：后端对所有 mutation 校验 `Origin` 头；引入双提交 CSRF token（meta + 自定义请求头）；确认 Nitro 代理 `apps/web/server/routes/api/v1/[...path].ts` 透传 `Origin`。
- **已修复（2026-07-09）**：采用 Fiber v3 `middleware/csrf`，double-submit cookie（`csrf_`）+ `X-Csrf-Token` header，覆盖全部 unsafe 方法，状态存共享 Redis；新增 `CSRF_TRUSTED_ORIGINS` 配置解决代理后 Host/Origin 不匹配。详见 `decisions/2026-07-09-security-fixes.md` CSRF 章节。

### [x] H6. 容器以 root 运行
- **位置**：`apps/api/Dockerfile:31-57`、`apps/web/Dockerfile:20-30`（无 `USER`/`adduser`）。
- **问题**：所有镜像以 uid 0 运行，且挂载了可写卷（attachments/extensions/themes）。任何 RCE/路径穿越直接获得容器 root。
- **修复方向**：Dockerfile 加 `addgroup -S sforum && adduser -S -G sforum sforum` + `USER sforum`，并 `chown` 工作目录与挂载点。

---

## 🟡 中危（Medium）

### [x] M1. 附件 Content-Type 信任客户端 + `inline` 处理 → 存储型 XSS
- **位置**：`apps/api/app/Http/Controllers/Attachments/controller.go:120-121`（返回客户端 Content-Type + `Content-Disposition: inline`）；`app/Models/Attachments/service.go:388-395`（信任上传者 Content-Type）、`:629-634`（MIME 白名单支持 `*/*`、`image/*` 通配）、`:470`（默认 `Visibility: public`）。
- **问题**：管理员把 `text/html`、`image/svg+xml` 或 `*/*` 加入白名单后，`/attachments/:publicId/content` 即成为存储型 XSS 分发器（`image/*` 会接受 `image/svg+xml`）。匿名可读。
- **修复方向**：非安全类型（`text/html`、`image/svg+xml`、`application/xhtml+xml` 等）强制 `Content-Disposition: attachment` 或全局 attachment；存储的 Content-Type 仅来自 `http.DetectContentType`，忽略客户端头；硬性拒绝 `text/html`/`image/svg+xml`/通配符 MIME 配置。
- **已修复（2026-07-09）**：`Options/attachment_options.go` 新增 `attachmentActiveContentDenylist`，在 `normalizeAttachmentMIMETypes` 入库配置层硬封禁 `text/html`/`image/svg+xml`/`application/javascript` 等主动内容类型（混合列表含任一即整体拒绝，通配符配置保留但 content 层兜底）；`Attachments/controller.go` content handler 统一加 `X-Content-Type-Options: nosniff`，对主动内容类型强制 `Content-Disposition: attachment`（纵深防线）。测试覆盖 denylist/通配符/disposition 决策/文件名净化。

### [ ] M2. 登录/注册/密码重置无端点级限流
- **位置**：`apps/api/app/Http/Controllers/Identity/controller.go:156-179`（login）、`:219-254`（password reset）；全局限流 `app/Http/server.go:58-72`（默认 30 次/分钟 = 43200 次/天）。
- **问题**：无独立失败计数/锁定/按 IP+账号节流；人机验证默认 `disabled`（`config.go:130`）。暴力破解几乎无阻力。
- **修复方向**：对 `/auth/login`、`/auth/register`、`/auth/password-reset/*` 加按 IP（尽量按账号）失败计数 + 指数退避/临时锁定；login 默认要求人机验证或降级触发。
- **待办（2026-07-09）**：降级处理。完整 per-endpoint 失败计数需新建 Redis 计数器基础设施并注入 controller 构造链，属独立功能工程。当前缓解：全局 `LIMITER_WRITE_MAX` 限制所有写请求速率；运营者可收紧该值或启用 login 人机验证（`human_verification.scenarios.login_risk`）。建议作为独立里程碑实现。

### [ ] M3. ALTCHA 令牌在注册前消费（冲突时浪费配额）
- **位置**：`apps/api/app/Http/Controllers/Identity/controller.go:120-129`（verify 后才 Register）；`service.go:81`（`MarkUsed` 立即消费）。
- **问题**：`ValidateRegister`（事务外预检）与 `Register`（事务内）间存在 TOCTOU 窗口；并发冲突时 token 已消费，合法用户的人机验证配额被浪费，可被恶意消耗。
- **修复方向**：两阶段 verify——先 `Verify` 验证真实性不标记，注册成功后再 `MarkUsed`；或冲突时退还 token。
- **待办（2026-07-09）**：降级处理。需重构 HumanVerify 接口为两阶段（`Verify` 不消费 + `MarkUsed` 显式消费），影响所有调用方（login/password-reset/register），属接口重设计。当前影响为用户体验（配额浪费），非安全漏洞。建议与 M2 一并作为认证强化里程碑实现。

### [x] M4. `/auth/registration-status` 公开暴露系统初始化状态
- **位置**：`apps/api/app/Http/Controllers/Identity/controller.go:148-154`；`app/Models/Identity/service.go:67-74`；路由 `routes.go:9`（无认证）。
- **问题**：无认证返回 `nextUserIsInitialSuperAdmin: bool`，无用户时返回 `true`，暴露"系统处于首注册窗口、下个注册者成 super_admin"，是首用户劫持的信息面。
- **修复方向**：未认证请求不返回 `nextUserIsInitialSuperAdmin`（无用户时统一返回 `false` 或 404）；或对该端点严格限流。

### [x] M5. 数据库 `sslmode=disable` 硬编码
- **位置**：`apps/api/config/config.go:103`（默认 `...?sslmode=disable`）；`compose.yaml:51,82`、`compose.dev.yaml:26,42,87`、`compose.prod.yaml:15`；`.env.production.example:18`。
- **问题**：所有示例/默认关闭 DB TLS，跨主机/云 PG 部署明文传输凭据与数据；`pool.go` 无显式 TLS 配置。
- **修复方向**：生产模板改为 `sslmode=require`（或 `verify-full`）并在文档说明；`pool.go` 支持显式 TLS 配置。

### [x] M6. 管理员搜索/DB 浏览器 LIKE 未转义 + 深分页无 page 上限（DoS）
- **位置**：`apps/api/app/Models/Identity/postgres_store.go:181,193`（`LIKE '%' || lower($1) || '%'`）；`app/Models/Identity/service.go:429`（`normalizePage` 无 `maxPage`，对比 `Forum.normalizePage` `service.go:1048,1055` 有 `maxTopicPage=200`）；`app/Models/Database/service.go:237-246`（`normalizeRowsInput` 只 clamp `PerPage`）、`postgres_store.go:114`；`app/Models/Attachments/postgres_store.go:63,66`；`app/Models/Database/postgres_store.go:333`（`ILIKE` 未转义）。
- **问题**：参数化（非注入），但 `%`/`_` 未转义 + `page` 无上限 → 深 OFFSET 全表扫描 DoS。admin 会话被滥用时是廉价 DoS 向量。
- **修复方向**：转义 `%`/`_` 并加 `ESCAPE '\'`，或改前缀匹配；给 `identity.normalizePage`、`Database.normalizeRowsInput` 加 `maxPage` 上限（对齐 Forum 的 200）。

### [x] M7. 个人资料 `websiteUrl` 经 `:href` 绑定 → `javascript:` XSS
- **位置**：`extensions/builtin/themes/sforum-default/layer/app/pages/u/[username].vue:98`（`<a :href="profile.profile.websiteUrl">`）；`contracts/openapi/schemas/profile.yaml:81-83`（仅 `maxLength:200`，无 scheme 校验）。
- **问题**：Vue `v-bind:href` 不过滤 `javascript:` 协议；后端无 scheme 强制校验。用户设 `javascript:alert(document.cookie)` → 访客点击触发 XSS（可窃取 cookie 会话）。
- **修复方向**：前端渲染前校验 scheme 白名单（仅 `http`/`https`/`mailto`，可复用 `apps/web/app/utils/sfEditor.ts:196` 的 `normalizeUserUrl`）；后端 `UpdateProfileRequest.websiteUrl` 加 `format: uri` + scheme 校验。

### [x] M8. 密码重置后旧 session 不失效
- **位置**：`apps/api/app/Models/Identity/password_reset_service.go:122-147`（`ConfirmPasswordReset`）；`postgres_store.go:57-74`（`ConsumePasswordResetToken`）。
- **问题**：令牌生成（crypto/rand 32 字节）、sha256 哈希存储、原子一次性消费、30 分钟过期均正确。但重置密码后不注销该用户其他活跃 session（`AuthSession.Manager` 无 by-userID 失效能力）。攻击者已登录则 session 仍有效。
- **修复方向**：重置成功后使该 userID 所有 session 失效（需 session store 支持 by-user 失效，或维护 userID→sessionID 索引）。

### [x] M9. session `Secure` 依赖 `APP_ENV` 字符串
- **位置**：`apps/api/bootstrap/app.go:115-125`（`CookieSecure: strings.EqualFold(cfg.AppEnv, "production")`）。
- **问题**：`HttpOnly=true`、`SameSite=Lax` 正确；但 `Secure` 仅当 `APP_ENV=="production"` 为 true。误填 `APP_ENV=prod`/`staging` 或漏配 → Secure 退化，cookie 走 HTTP。
- **修复方向**：改为可配置开关，或检测 `APP_URL` 是否 `https://` 决定 Secure，不依赖环境字符串。

### [x] M10. 前端依赖用 `latest`
- **位置**：`apps/web/package.json:18,20,21,29,30`（`@nuxt/ui`、`@nuxtjs/i18n`、`@nuxtjs/seo`、`nuxt`、`vue` 均为 `latest`）。
- **问题**：虽 `bun.lock` 存在可复现，但 `package.json` 用 `latest`，lock 刷新时静默拉主版本，存在破坏性升级/供应链投毒风险。Go 侧 `go.mod` 版本明确，无此问题。
- **修复方向**：将 `latest` 改为具体版本范围（`^x.y.z`）。

---

## 🟢 低危（Low）

### [x] L1. 登录用户名枚举时序差异
- **位置**：`apps/api/app/Models/Identity/service.go:250-271`（用户不存在直接返回，存在时跑 argon2，耗时数十毫秒）。响应码/消息已统一（`controller.go:554-555`），仅时序侧信道。
- **修复方向**：用户不存在时执行一次 dummy argon2 对齐时序。

### [ ] L2. 图片 EXIF/GPS 元数据未清洗
- **位置**：`apps/api/app/Models/Attachments/service.go:397-408`（仅 `image.DecodeConfig` 读尺寸，字节原样存储；4MB LimitReader 已防解码 DoS）。
- **修复方向**：存储前去除 EXIF/重编码（隐私）。
- **待办（2026-07-09）**：降级处理。avatar 上传路径（含人脸照片，隐私最敏感）已通过 `prepareAvatarUpload` 的图片重编码去除 EXIF；通用附件的完整 EXIF 清洗需重构上传字节流（影响 sha256/size/多格式 encoder），收益（低危隐私）与成本不匹配。建议后续在附件存储层引入统一的图片处理管线时一并实现。

### [x] L3. `Forum.paramInt` 解析失败静默返回 0
- **位置**：`apps/api/app/Http/Controllers/Forum/controller.go:458`（`paramInt`）、`:453`（`queryInt`）。当前靠 `GetTopic` 的 `topicID<=0 → 404` 兜底安全，但与 `Identity`（`controller.go:559` `paramInt64` 返回 error → 422）不一致，纵深防御风险。
- **修复方向**：统一 `paramInt` 返回 `(int64, error)`，解析失败映射 400/422。

### [x] L4. 附件 list / DB 浏览器 filter LIKE 未转义（参数化，非注入，仅性能）
- **位置**：`apps/api/app/Models/Attachments/postgres_store.go:63`（双通配，最慢）、`:66`（前缀）；`app/Models/Database/postgres_store.go:333`。
- **修复方向**：转义 `%`/`_` + `ESCAPE`。

### [x] L5. FTP/SFTP `Exists` 吞掉传输错误
- **位置**：`apps/api/app/Support/Storage/ftp.go:92-98`、`sftp.go:113-119`（`Stat` 失败统一返回 `false,nil`，无法区分"不存在"与"存储故障"）。
- **修复方向**：返回实际错误，由调用方决定。

### [x] L6. super_admin 的 deny override 永不生效
- **位置**：`apps/api/app/Models/Identity/policy.go:19-30`（`Actor.Can` 对 super_admin 直接 `return true`，DB 层 deny 扣减逻辑被绕过）。
- **修复方向**：属设计（super_admin 全能），无需改代码；但应文档化，避免运维误以为可限制 super_admin。

---

## ✅ 已确认安全（审计核对过、无问题）

记录以供未来审计参考，避免重复排查：

- **SQL 注入**：用户字符串值全走参数化占位符；唯一动态标识符（DB 浏览器）用 `pgx.Identifier.Sanitize()`；`Forum.ListTopics` 的 ILIKE 已是死代码（搜索改走 Meilisearch，`escapeFilterValue` 已转义）。
- **Zip Slip（条目名）**：`safeArchivePath`/`normalizeObjectKey` 正确拒绝 `..`/绝对路径。
- **Zip Bomb**：解压前按 `UncompressedSize64` 累计校验 50MB 上限 + 控制器层 60MB 限制。
- **本地存储根逃逸**：`filepath.Join` + 前缀校验，`renderObjectKey` 拒绝 `../`。
- **publicId**：128 位 `crypto/rand`，不可枚举。
- **首用户 super_admin 竞态**：`WithBootstrapTx` 用 `pg_advisory_xact_lock` + 同事务，正确缓解（注意：仅 bootstrap 路径受保护）。
- **密码哈希**：argon2id，memory=64MB、time=1、threads=4、key=32B、salt=16B crypto/rand、`subtle.ConstantTimeCompare`（注：time=1 偏低，OWASP 建议≥3）。
- **密码重置令牌**：crypto/rand 32 字节、sha256 哈希存储、原子一次性消费（`UPDATE ... WHERE consumed_at IS NULL RETURNING`）、30 分钟过期。
- **主题/评论 CRUD 的 IDOR**：逐一核对，均比对 `actor.ID == 作者` 或管理权限（`canEditTopic`/`canDeleteTopic`/`canEditComment`/`canDeleteComment`）。
- **admin 端点鉴权**：`/admin/forum/*`、`/admin/moderation/*`、`/admin/mail/test`、`/permissions*`、`/roles*`、`/users*` 全部有 `actor(c)` + service 层权限检查，无遗漏。
- **事务边界**：CreateTopic/UpdateTopic/DeleteTopic/CreateComment 单事务 + defer Rollback + Commit，计数用 `GREATEST(count-1,0)` 防负。
- **前端**：无 localStorage/sessionStorage 存 token、无 `eval`/`new Function`/`document.write`、无开放重定向、无 SSRF（Nitro 代理目标来自环境变量）、密钥选项已掩码、无错误回显堆栈、JSON-LD 经 unhead 转义。
- **git**：`.env` 未入库（`.gitignore:9-13` 正确），`git log --all -- .env` 无记录，无密钥文件提交。
- **端口暴露**：dev/prod compose 仅 `127.0.0.1` 绑定，生产只暴露 web，内部服务在容器内网。

---

## 修复优先级

1. **立即**：C1（路径穿越 RCE）、C2（密钥默认值）、H1（自我提权）— 可被实际利用的高危。
2. **本周**：H2（凭证加密 + SFTP）、H3（XSS 纵深防御）、H4（安全头）、H5（CSRF）、H6（容器降权）。
3. **排期**：M1-M10 中危项。
4. **顺手**：L1-L6 低危项（重构/加固时一并处理）。

---

## Open Questions

- 密钥静态加密（H2/C2）的 `APP_OPTION_ENC_KEY` 与各 secret 是否应统一到外部 secret manager / vault，而非环境变量？
- CSRF 防护（H5）采用 Origin 校验 + 双提交 token，还是迁移到 token-based（Bearer）API 鉴权？
- session by-user 失效（M8）是否需要引入 userID→sessionID 索引，或改用可撤销的 session 版本号？
