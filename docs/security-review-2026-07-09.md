# 2026-07-09 代码与安全审阅报告

## 范围

- 审阅对象：SForum 当前工作区的 Go Fiber API、Nuxt 前端代理、内置默认主题页面、部署配置与相关知识库文档。
- 审阅方式：静态代码审阅，未修改业务代码，未执行完整测试套件。
- 重点方向：鉴权边界、公开数据可见性、配置安全、附件响应、默认部署体验。

## 结论摘要

本次审阅发现 6 个值得优先处理的问题：

- 2 个 P1：CSRF 防护缺失、评论接口绕过主题/分类可见性。
- 3 个 P2：密码重置人机验证启用后失效、生产 Redis/session 配置错配、附件主动内容风险。
- 1 个 P3：资料更新接口可直接写入任意头像附件 ID。

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

### P3: 资料更新接口可直接写入任意头像附件 ID

**影响**

普通资料更新接口接受 `avatarAttachmentId` 并只校验其为正数，然后写入用户资料。手工 API 请求可能把头像指向不存在的附件、他人的附件、非 active 附件或非头像用途附件。当前影响偏低，但在头像展示和附件权限继续扩展后会变成更明显的完整性问题。

**证据**

- `apps/api/app/Http/Controllers/Profile/controller.go`
  - `updateProfileRequest` 接受 `avatarAttachmentId`。
  - `updateMyProfile` 直接传入 service。
- `apps/api/app/Models/Profile/service.go`
  - `normalizeUpdateProfileInput` 只校验 `avatarAttachmentId > 0`。
  - `UpdateMyProfile` 合并后直接 `UpsertProfile`。
- `apps/api/app/Models/Profile/store.go`
  - profile store interface 当前没有头像附件归属/状态校验入口。

**建议**

- 资料更新接口不要直接接受任意 `avatarAttachmentId`，或在写入前校验：
  - 附件存在。
  - 附件状态 active。
  - 附件 owner 是当前用户。
  - 附件用途/类型符合头像规则。
- 更推荐只允许专门的头像上传/设置流程修改头像附件引用。
- 增加越权引用他人附件、引用不存在附件、引用 disabled/deleted 附件的拒绝测试。

## 修复优先级建议

1. 先处理 P1 评论可见性绕过，因为这是直接数据泄漏风险，修复边界相对清晰。
2. 同步规划 CSRF 防护，这是 cookie session 架构进入生产前的基础安全能力。
3. 修复密码重置人机验证，避免安全开关一打开就造成用户锁死。
4. 纠正生产配置样例，降低部署踩坑概率。
5. 收紧附件响应策略，避免后续运营配置扩大攻击面。
6. 收敛头像附件写入路径，保持资料数据完整性。

## 验证建议

- 后端：
  - 增加 forum store/service 测试，覆盖隐藏主题、删除主题、隐藏分类的 comments/replies。
  - 增加 identity controller/service 测试，覆盖 password_reset human verification。
  - 增加 attachment content response 测试，覆盖危险 MIME 的 disposition/header。
  - 增加 profile update 测试，覆盖头像附件归属与状态。
- 集成：
  - 在 CSRF 落地后覆盖带 token、缺 token、错误 token、错误 Origin 的 unsafe 请求。
  - 更新 OpenAPI 契约，标注 unsafe cookie-auth endpoints 的 CSRF 要求。
- 配置：
  - 用 `.env.production.example` 跑一次 `docker compose config` 和 API 启动冒烟测试。

## 备注

- 本报告只整理审阅发现，没有实施修复。
- 当前工作区存在未提交改动；修复前建议先确认这些改动是否属于正在进行的头像/附件配置工作，避免覆盖他人修改。
