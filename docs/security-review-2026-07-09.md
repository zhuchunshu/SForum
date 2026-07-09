# 2026-07-09 代码与安全审阅报告

## 范围

- 审阅对象：SForum 当前工作区的 Go Fiber API、Nuxt 前端代理、内置默认主题页面、部署配置与相关知识库文档。
- 审阅方式：静态代码审阅，未修改业务代码，未执行完整测试套件。
- 重点方向：鉴权边界、公开数据可见性、配置安全、附件响应、默认部署体验。

## 结论摘要

本次审阅发现 6 个条目，其中 **5 个经核验属实并已全部修复**、**1 个经核验不成立**：

- 2 个 P1（已修复）：CSRF 防护缺失、评论接口绕过主题/分类可见性。
- 3 个 P2（已修复）：密码重置人机验证启用后失效、生产 Redis/session 配置错配、附件主动内容风险。
- 1 个 P3（经核验不成立）：原报告称资料更新接口可直接写入任意头像附件 ID；实际 `Profile/postgres_store.go` 的 `validateAvatarAttachment` 已在同一事务内校验附件存在、owner==actor、status==active、`image/*`，越权引用他人/不存在/非图片/非 active 附件均返回 `ErrProfileInvalid`。降级为可选的纵深防御优化。

> 修复记录（2026-07-09）：5 个属实问题分两批修复——评论可见性/密码重置 HV/生产配置/附件主动内容（第一批），CSRF 防护（独立里程碑）。详见各章节「已修复」说明与 `knowledge/decisions/2026-07-09-security-fixes.md`。
>
> 更正记录（2026-07-09）：P3 原始判断基于 service 层归一化只校验 `avatarAttachmentId > 0`，遗漏了 store 层事务内的归属/状态/类型校验。详见下方 P3 章节订正。

## Findings

### P1: Cookie 鉴权写接口缺少 CSRF 防护

**影响**

浏览器会自动携带 `sforum_session` Cookie。当前写接口仅依赖 SameSite=Lax Cookie，缺少 CSRF token 或严格 Origin/Referer 校验。只要攻击者能诱导已登录用户发起跨站 unsafe 请求，就可能触发登录态下的管理或成员操作。

**证据**

- `apps/api/bootstrap/app.go`
  - session 使用 `extractors.FromCookie("sforum_session")`。
  - Cookie 配置为 `HTTPOnly` 与 `SameSite=Lax`，但没有配套 CSRF token。
- `apps/api/app/Http/server.go`
  - 全局中间件包含 request id、recover、compress、写接口 limiter、locale。
  - 未看到 CSRF 或 Origin/Referer 校验。
- `apps/web/server/routes/api/v1/[...path].ts`
  - Nuxt server route 直接 `proxyRequest(event, target.toString())`，没有在代理层阻断 unsafe 跨站请求。
- `apps/api/app/Http/Controllers/Identity/routes.go`
  - 存在 `/auth/login`、`/auth/logout`、角色、权限、用户角色等 POST/PUT/PATCH/DELETE 路由。
- `apps/api/app/Http/Controllers/Forum/routes.go`
  - 存在发帖、评论、隐藏、恢复、锁定、置顶等 unsafe 路由。
- `knowledge/decisions/2026-07-05-browser-session-jwt-strategy.md`
  - 项目决策文档已明确记录：cookie-authenticated browser flow 的 unsafe 写请求在 production-ready 前仍需要 CSRF protection。

**建议**

- 为所有 cookie-authenticated unsafe 方法添加 CSRF 防护。
- 推荐使用同步器 token 或 double-submit token；同时可叠加严格 Origin/Referer 校验。
- API 侧必须作为权威防线，前端隐藏按钮或路由守卫不能替代。
- 增加允许和拒绝路径测试，覆盖普通成员与管理员 unsafe 操作。

**已修复（2026-07-09）**：采用 Fiber v3 自带 `middleware/csrf`，double-submit cookie（`csrf_`）+ `X-Csrf-Token` header，状态存共享 Redis storage，覆盖全部 unsafe 方法（含 login/register/password-reset，防登录型 CSRF）。注册在 `/api/v1` group，GET 请求种下可读 cookie 供 SPA 读取回传。新增 `CSRF_TRUSTED_ORIGINS` 配置（逗号分隔，默认从 `APP_URL` 派生），解决代理后 Host 与公开 Origin 不匹配的问题。前端 `useApiClient` 自动注入 header，修复 `SFNavbar` logout 绕过点。7 个中间件测试覆盖允许/拒绝/token/Origin/envelope 路径。生产部署必须在 `CSRF_TRUSTED_ORIGINS` 列出公开站点，否则所有写请求被拒。

### P1: 评论和回复接口可绕过主题/分类可见性

**影响**

主题详情接口会限制主题状态和分类可见性，但评论列表与回复列表只检查评论自身状态。攻击者如果知道隐藏/删除主题 ID 或评论 ID，可能仍可读取 active 评论内容，包括后端渲染后的 HTML 与纯文本摘要。

**证据**

- `apps/api/app/Models/Forum/postgres_store.go`
  - `GetTopic` / `GetTopicBySlug` 查询包含：
    - `topics.status IN ('active', 'locked')`
    - `categories.visibility = 'public'`
  - `listCommentsFlat` 仅过滤：
    - `comments.topic_id = $1`
    - `comments.status = 'active'`
  - `listCommentsTree` 根评论与子孙评论查询同样只过滤 topic id 与 comment status。
  - `ListCommentReplies` 仅过滤：
    - `comments.parent_comment_id = $1`
    - `comments.status = 'active'`
- `apps/api/app/Http/Controllers/Forum/routes.go`
  - `GET /topics/:topicID/comments`
  - `GET /comments/:commentID/replies`
  - 这两个读取路由是公开 GET。
- `apps/api/app/Models/Forum/service.go`
  - `ListComments` 只校验 view 与分页语义后透传 store。
  - `ListCommentReplies` 只校验 comment id 大于 0 后透传 store。

**建议**

- 评论列表查询应 join `topics` 与 `categories`，复用主题详情的公开可见性规则。
- 回复查询应从 parent comment 追溯 topic，再校验 topic/category 是否公开可见。
- 对隐藏主题、删除主题、隐藏分类分别补充回归测试，期望返回 not found 或权限错误，而不是空绕过。

### P2: 启用密码重置人机验证后流程会失效

**影响**

后台配置允许启用 `password_reset` 人机验证场景，但当前密码重置请求没有正确读取嵌套的人机验证字段，默认主题页面也没有提交验证 token。运营者一旦启用该安全选项，普通用户可能无法请求密码重置邮件。

**证据**

- `apps/api/app/Models/Options/service.go`
  - `humanVerificationScenarios` 包含 `password_reset`。
  - `HumanVerificationConfig` 会把该场景映射到 runtime config。
- `apps/api/app/Http/Controllers/Identity/controller.go`
  - `passwordResetRequest` 绑定 `passwordResetRequestPayload`。
  - 该 payload 中人机验证字段是 `humanVerification` 嵌套对象。
  - 随后调用 `verifyHumanVerification`。
  - `verifyHumanVerification` 再次把 body 绑定到 `humanVerificationRequest`，期待顶层 `provider` / `token`，没有读取 `req.HumanVerification`。
- `extensions/builtin/themes/sforum-default/layer/app/pages/forgot-password.vue`
  - 请求体只发送 `{ email }`，没有 ALTCHA widget/token。

**建议**

- `passwordResetRequest` 应从第一次绑定得到的 `req.HumanVerification` 调用 verifier，避免二次绑定同一 body。
- 默认主题忘记密码页应接入与注册页一致的人机验证控件。
- 增加测试：
  - 未启用 password_reset verification 时可正常请求。
  - 启用后缺 token 应拒绝。
  - 启用后有效 token 应通过。

### P2: 生产配置样例与实际 Redis/session 配置不一致

**影响**

照着生产样例部署时，API 会尝试用密码连接 Redis，但 Compose 中 Redis 没启用认证，可能导致 session storage、业务缓存、人机验证 Redis client 初始化或运行失败。另有 session/CSRF 环境变量命名与代码不一致，容易让生产部署误以为安全配置已生效。

**证据**

- `.env.production.example`
  - `REDIS_PASSWORD=change-me`
  - `SESSION_SECRET=change-me`
  - `CSRF_SECRET=change-me`
- `compose.yaml`
  - Redis command 是 `redis-server --appendonly yes`，没有 `--requirepass`。
- `apps/api/bootstrap/app.go`
  - API 将 `cfg.RedisPassword` 传给 Fiber Redis storage 与共享 Redis client。
- `apps/api/app/Support/Redis/session.go`
  - Redis storage config 会使用传入的 `Password`。
- `apps/api/config/config.go`
  - 实际读取的是 `SESSION_HASH_SECRET`，不是 `SESSION_SECRET`。
  - 当前没有看到 `CSRF_SECRET` 被读取或 CSRF 功能接入。
- `docs/development-and-deployment.md`
  - 生产清单列出了 `SESSION_SECRET` 与 `CSRF_SECRET`，与当前实现不一致。

**建议**

- 二选一修复 Redis：
  - Compose Redis 使用 `--requirepass ${REDIS_PASSWORD}` 并同步 healthcheck。
  - 或生产样例默认 `REDIS_PASSWORD=`，明确说明外部 Redis 有密码时才填写。
- 将文档和样例中的 `SESSION_SECRET` 改为当前代码实际使用的 `SESSION_HASH_SECRET`。
- 在 CSRF 功能落地前，不要让生产样例暴露一个不会生效的 `CSRF_SECRET`；或者同步实现 CSRF 配置。

### P2/P3: 附件允许列表放宽后可能引入同源主动内容风险

**影响**

附件默认允许列表相对安全，但后台 MIME 校验允许任意合法 MIME 或通配符。如果运营者允许 `image/svg+xml`、`text/html` 等主动内容类型，公开附件会以同源 `inline` 响应返回，可能形成 XSS 或同源脚本执行面。

**证据**

- `apps/api/app/Models/Options/service.go`
  - 默认允许 MIME 包括图片、PDF、text/plain、zip。
- `apps/api/app/Models/Options/attachment_options.go`
  - `normalizeAttachmentMIMETypes` 接受合法 `type/subtype` 与 `*` 段。
  - 没有阻止 `text/html`、`image/svg+xml` 或 `*/*`。
- `apps/api/app/Models/Attachments/service.go`
  - 上传校验使用配置中的 allowed MIME types。
- `apps/api/app/Http/Controllers/Attachments/controller.go`
  - 内容响应设置 `Content-Type` 为存储值。
  - `Content-Disposition` 使用 `inline`。
  - 未设置 `X-Content-Type-Options: nosniff`、sandbox CSP 或危险类型强制下载。

**建议**

- 对主动内容类型采取硬边界：
  - 阻止 `text/html`、`image/svg+xml` 等作为普通公开附件。
  - 或对这些类型强制 `Content-Disposition: attachment`。
- 添加 `X-Content-Type-Options: nosniff`。
- 若未来需要预览 SVG/HTML，应放到隔离域名或 sandbox 响应策略下。
- 后台配置 UI 应提示允许主动内容的风险。

### P3: 资料更新接口头像附件 ID（经核验：原判断不成立，降级为可选优化）

**影响**

经二次核验，原始判断「可直接写入任意头像附件 ID」**不成立**。

普通资料更新接口接受 `avatarAttachmentId`，service 层归一化确实只校验其为正数，但 store 层在写入头像引用前已在同一事务内做完整校验：

- 附件必须存在（`SELECT ... FROM attachments WHERE id = $1`）。
- `owner_user_id` 必须等于当前用户。
- `status` 必须为 `active`。
- `content_type` 必须以 `image/` 开头。

任一不满足返回 `ErrProfileInvalid`，越权引用他人附件、不存在附件、非 active 或非图片附件均被拦截。因此不存在完整性缺陷，仅作为可选纵深防御优化项保留。

**证据**

- `apps/api/app/Http/Controllers/Profile/controller.go`
  - `updateProfileRequest` 接受 `avatarAttachmentId`，透传 service。
- `apps/api/app/Models/Profile/service.go`
  - `normalizeUpdateProfileInput` 仅校验 `avatarAttachmentId > 0`。
  - `UpdateMyProfile` 在 `avatarAttachmentId != nil` 时调用 `s.store.SetAvatarAttachment(ctx, actor.ID, attachmentID, actor.ID)`，传入 `actor.ID` 作为归属校验基准。
- `apps/api/app/Models/Profile/postgres_store.go`
  - `SetAvatarAttachment` 在事务内调用 `validateAvatarAttachment(ctx, tx, userID, attachmentID)`。
  - `validateAvatarAttachment` 查询 `SELECT id, public_id, owner_user_id, content_type, status FROM attachments WHERE id = $1`，校验 `attachment.OwnerUserID != userID || attachment.Status != "active" || !strings.HasPrefix(... "image/")` → `ErrProfileInvalid`。

**建议（可选纵深防御，非缺陷修复）**

- 可在 service 层提前校验归属/状态，减少一次事务往返（当前已由 store 层保证正确性）。
- 或收敛到专门的头像上传/设置流程，让资料更新接口不接受裸 `avatarAttachmentId`。
- `validateAvatarAttachment` 已有完备逻辑，可补充对应的单元测试固定其拒绝行为。

## 修复优先级建议

1. 先处理 P1 评论可见性绕过，因为这是直接数据泄漏风险，修复边界相对清晰。
2. 同步规划 CSRF 防护，这是 cookie session 架构进入生产前的基础安全能力。
3. 修复密码重置人机验证，避免安全开关一打开就造成用户锁死。
4. 纠正生产配置样例，降低部署踩坑概率。
5. 收紧附件响应策略，避免后续运营配置扩大攻击面。
6. P3 头像附件写入经核验不构成缺陷，store 层已有归属/状态/类型校验；如需可选优化见 P3 章节，优先级最低。

## 验证建议

- 后端：
  - 增加 forum store/service 测试，覆盖隐藏主题、删除主题、隐藏分类的 comments/replies。
  - 增加 identity controller/service 测试，覆盖 password_reset human verification。
  - 增加 attachment content response 测试，覆盖危险 MIME 的 disposition/header。
  - 增加 profile update 测试，覆盖头像附件归属与状态。（P3 经核验 store 层已有校验，此测试为补充固定，非缺陷修复。）
- 集成：
  - 在 CSRF 落地后覆盖带 token、缺 token、错误 token、错误 Origin 的 unsafe 请求。
  - 更新 OpenAPI 契约，标注 unsafe cookie-auth endpoints 的 CSRF 要求。
- 配置：
  - 用 `.env.production.example` 跑一次 `docker compose config` 和 API 启动冒烟测试。

## 备注

- 本报告初稿只整理审阅发现，未实施修复。后续修复进展见各章节订正说明。
- P1 评论可见性、P2 密码重置人机验证、P2 生产配置、P2 附件主动内容风险、P1 CSRF 防护 已于 2026-07-09 全部修复。
- P3 经核验不成立，store 层事务内已有归属/状态/类型校验。
- 本报告核验阶段为只读静态审阅，未修改业务代码。
