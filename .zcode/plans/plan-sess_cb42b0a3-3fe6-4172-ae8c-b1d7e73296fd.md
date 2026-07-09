# 全栈安全审计剩余 23 条修复计划

## 核实结论
三个 Explore agent 逐条核实了实际代码：23 条全部仍存在（L6 是 by-design 文档项）。工作区的未提交改动是 Contributions 功能，与 C1 无关，可同分支改（不同区域）。决策已确认：C1 直接改、H2 用 AES-GCM、M8 用 token 版本号。

## 分批策略（按子系统聚合）
23 条按子系统分 6 批，每批内部高内聚。顺序按风险与依赖：先 Critical/High，再 Medium，最后 Low。每批改完即测。

---

## 批次 A：配置与密钥（C2 + M5 + M9）
高内聚：都改 config/bootstrap 的 secret/部署安全。

- **C2 生产密钥强制校验**：`config/config.go` 新增 `validateProductionSecrets(cfg)`，当 `AppEnv=="production"` 时校验 `SessionHashSecret`/`AltchaSecret`/`MeiliMasterKey` 非空且非占位词（`change-me`/`sforum-dev-*`），失败则 `Load()` panic 拒绝启动。在 `cmd/api/main.go` 调用。`config_test.go` 覆盖：生产+默认值→panic、生产+有效值→通过、非生产+默认值→通过。
- **M5 DB sslmode**：`config.go` 默认 DATABASE_URL 改 `sslmode=require`（开发本地无 TLS 时显式覆盖为 disable）；`.env.production.example`、`compose.prod.yaml`、docs 改 `sslmode=require`。`compose.yaml`/`compose.dev.yaml` 保留 disable（开发环境）。config_test 更新默认值断言。
- **M9 session Secure 不依赖字符串**：`bootstrap/app.go:124` 改为检测 `APP_URL` 是否 `https://` scheme 决定 CookieSecure（生产兜底仍保留 `AppEnv=="production"` 时强制 true）。bootstrap_test 覆盖。

## 批次 B：身份与权限（H1 + M4 + M8 + L1）
高内聚：都改 Identity service/controller/session。

- **H1 防自我提权**：`Identity/service.go` `ReplaceUserRoles`/`ReplaceUserPermissionOverrides` 加：(1) `actor.ID == targetUserID` 拒绝（自改）；(2) 授予/操作 super_admin 角色要求 actor 本身是 super_admin。新增 sentinel error。service_test 覆盖：member 提权自己→拒、member 授他人 super_admin→拒、super_admin 操作→通过。
- **M4 registration-status 信息泄漏**：`Identity/service.go` `RegistrationStatus` 对未认证请求不返回 `NextUserIsInitialSuperAdmin`（无用户时返回 false 或不包含该字段）。controller/routes 已无认证，service 层处理。service_test 覆盖。
- **M8 token 版本号**：session 数据加 `token_version`，users 表加 `current_token_version`（migration）；`AuthSession.Manager.CurrentUserID` 校验 session token_version == user current_token_version，不一致视为无会话；密码重置成功时 `store.IncrementUserTokenVersion`。涉及 migration + store + manager + password_reset_service。manager_test 覆盖。
- **L1 登录时序对齐**：`Identity/service.go` Login 用户不存在时执行一次 dummy argon2（预生成固定 dummy hash）对齐时序。service_test 覆盖（用短超时断言两条路径耗时接近，或断言走了 dummy 验证）。

## 批次 C：扩展安全（C1）
独立子系统，与 Contributions 改动同文件不同区域。

- **C1 Version 路径穿越**：`ExtensionManifest/manifest.go` `Validate` 新增 `manifestVersionPattern = ^[a-zA-Z0-9][a-zA-Z0-9._+\-~]{0,63}$`（禁止 `/`、`\`、纯 `.`/`..`），校验 Version。`manifest_test.go` 覆盖危险值拒绝。`Models/Extensions/service.go` `extractArchiveFiles` 常规文件 mode 掩码为 `0644`（仅 backend entry 恢复可执行位）。service_test 覆盖 mode。

## 批次 D：存储与凭证（H2）
独立子系统（Options 加密 + Storage SFTP）。

- **H2a AES-GCM 静态加密**：新增 `Support/Crypto` 包（AES-256-GCM 加解密 helper），密钥来自 `APP_OPTION_ENC_KEY`（config 新字段）。`Options/postgres_store.go` 写入时加密 secret 选项、`InternalValues` 解密。提供一次性迁移（启动时检测明文 secret 值并自动加密重写）。bootstrap 注入密钥。crypto_test + options store_test 覆盖。C2 一并校验生产 `APP_OPTION_ENC_KEY` 非空。
- **H2b SFTP 强制主机密钥**：`Storage/sftp.go` 空指纹时拒绝连接（返回错误）而非 `InsecureIgnoreHostKey`。sftp_test 覆盖。

## 批次 E：内容与响应安全（H3 + H4 + H6 + M7 + L2）
跨前后端，但都属"内容/响应防护"主题。

- **H3 前端 DOMPurify**：`apps/web` 新增 `dompurify` 依赖；`SFComment.vue`、`t/[...path].vue` 的 `v-html` 前用 DOMPurify 净化（白名单与服务端 bluemonday 对齐）。新增 `utils/sfSanitize.ts` 统一入口。typecheck 通过。
- **H4 安全响应头**：`deploy/caddy/Caddyfile` 加 `header { Strict-Transport-Security; X-Content-Type-Options nosniff; X-Frame-Options DENY; Referrer-Policy strict-origin-when-cross-origin }`；CSP 按 Nuxt 实际需要配置。API 侧评估是否也加全局中间件（目前附件有 nosniff，其余无）。
- **H6 容器降权**：`apps/api/Dockerfile`（api/worker/migrate 三 stage）、`apps/web/Dockerfile`（prod stage）加 `addgroup`/`adduser` + `chown` 工作目录与挂载点 + `USER sforum`。
- **M7 websiteUrl 前端兜底**：`u/[username].vue` 的 `:href` 前用 `normalizeUserUrl`/scheme 白名单校验（服务端已有 `isValidWebsiteURL`，前端补纵深防御）。
- **L2 EXIF 清洗**：`Attachments/service.go` `inspectUpload` 对图片附件存储前重编码去除 EXIF（或用 imaging 库）。注意：avatar 路径已 OK，只改通用附件路径。attachments service_test 覆盖。

## 批次 F：健壮性与性能（M2 + M3 + M6 + M10 + L3 + L4 + L5 + L6）
杂项加固。

- **M2 端点级限流**：`Identity/controller.go` login/register/password-reset 加按 IP 失败计数 + 临时锁定（复用 Redis storage 或 limiter，加专门 middleware/helper）。配置化阈值。
- **M3 ALTCHA 两阶段**：`Identity/controller.go` register 改为 `Verify`（不 MarkUsed）→ Register 成功后 `MarkUsed`；或冲突时退还 token。需 HumanVerify 暴露不消费的验证方法。
- **M6 LIKE 转义 + page cap**：Identity/Database/Attachments postgres_store 的 LIKE 转义 `%`/`_` + `ESCAPE '\'`；Identity/Attachments/Database `normalizePage` 加 `maxPage`（对齐 Forum 的 200）。
- **M10 前端依赖固定**：`apps/web/package.json` 的 `latest` 改为具体版本（从 bun.lock 读取当前锁定的版本）。
- **L3 paramInt 一致**：`Forum/controller.go` `paramInt`/`queryInt` 返回 error→422（对齐 Identity）。
- **L4**：与 M6 同批（LIKE 转义）。
- **L5 FTP/SFTP Exists 错误**：`Storage/ftp.go`/`sftp.go` Exists 返回真实错误，调用方决定。
- **L6**：文档化 super_admin deny 不生效（policy.go 注释 + 知识库），不改代码。

---

## 收尾（每批通用）
- 每批改完跑 `go test ./受影响包/` + 受影响前端 typecheck。
- 全部完成后跑 `./scripts/test.sh` + `ruby scripts/validate-openapi-refs.rb`。
- 勾选 `knowledge/decisions/2026-07-09-security-audit.md` 对应条目并加修复说明。
- 知识库：每批追加决策到 `2026-07-09-security-fixes.md`，更新 index。
- M4/M6/M7 若涉及契约字段语义变更，同步 OpenAPI。

## 风险与注意
- **C1 与 Contributions 同文件**：改 manifest.go Validate 的 Version 区域（:162-169、:290），Contributions 改的是 :276 validateContributions 和新增类型。提交时你需自行合并（不同区域，冲突概率低）。
- **H2a 迁移**：首次启动需把存量明文 secret 加密重写，需保证幂等（已加密的不重复处理）。
- **M8 token 版本**：需 DB migration 加列，且所有 session 创建路径写入版本号——涉及面较广，仔细核对 session 创建点。
- **L2 EXIF 重编码**：通用附件图片重编码会改变文件字节与可能的大小/尺寸校验逻辑，需确认不破坏现有 attachment 流程（avatar 已用 imaging 库，可复用）。
- **M10 锁定版本**：从 bun.lock 读取精确版本，避免引入破坏性升级。
- 不擅自 kill 3000 端口 dev server；前端改动由你人工验证。
- 全程不提交不推送，改动留工作区。