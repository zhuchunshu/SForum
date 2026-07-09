# CSRF 防护实施计划（独立安全里程碑）

## 范围与决策（已与你确认）
- 保护**全部** unsafe 方法端点（含 login/register/password-reset，防登录型 CSRF）。
- token 状态存**共享 Redis fiber.Storage**（`deps.Storage`，与 session 同源）。
- 信任源用**新增 `CSRF_TRUSTED_ORIGINS` 环境变量**（逗号分隔，支持 `https://*.example.com` 通配符），默认从 `APP_URL` 派生。
- 采用 Fiber v3 自带 `middleware/csrf`（已在 go.mod v3.0.0-rc.3 中），double-submit cookie + Origin/Referer 校验，header `X-Csrf-Token`。
- 复用现有 `useApiClient` 单一入口注入 token；修掉 `SFNavbar.vue` 绕过 `request` 的 logout 调用。

## 关键架构事实（来自代码核验）
- API 在 Nuxt 代理后，收到的 Host 是内部地址（`api:8080`），Origin 是公开站点。**必须**把公开 origin 加入 `TrustedOrigins`，否则每个 unsafe 请求 403。
- CSRF 中间件在 GET/HEAD/OPTIONS 时种下可读的 `csrf_` cookie（非 HTTPOnly，默认 SameSite=Lax）；unsafe 方法要求 `X-Csrf-Token` header == cookie 值，且 token 在 storage 中有效。
- **注意**：非 HTTPS 且无 Origin 时，中间件跳过 Origin/Referer 校验（csrf.go:138-141），仍做 token 校验。double-submit token 是核心防线，Origin 是纵深防御。
- session cookie 是 HTTPOnly（SPA 读不到），所以必须用独立的、可读的 `csrf_` cookie。
- 中间件默认 `ErrorHandler` 返回 `fiber.ErrForbidden`(403)，会被现有 `errorHandler` 接住。我自定义 ErrorHandler 返回统一 envelope reason（如 `csrf.invalid`），与项目 `permission.denied`/`rate_limit.exceeded` 风格一致。
- **配置字段名不能叫 `CSRFSecret`**——`config_test.go:144` 有守卫测试明确禁止该名字（防幽灵字段）。新字段命名为 `CSRFTrustedOrigins []string`。

---

## 修复 1：后端配置（新增信任源字段）

**文件**：`apps/api/config/config.go`
- Config struct 新增字段 `CSRFTrustedOrigins []string`（config.go:13-78 struct 内）。
- `Load()` 新增读取：新增 env 列表解析辅助 `envStringSlice(key, defaultRaw)`（逗号分隔 + trim + 去空），读 `CSRF_TRUSTED_ORIGINS`。当该环境变量未设置时，默认从 `APP_URL` 派生（用 `url.Parse` 取 `scheme://host`，APP_URL 为空则默认空列表）。
- 在 config_test.go 增 `envStringSlice` 单测 + `CSRF_TRUSTED_ORIGINS` 读取测试。

**文件**：`apps/api/config/config_test.go`
- 更新 `TestConfigDoesNotExposePhantomSessionOrCSRFFields`（config_test.go:144）：该测试断言不存在 `CSRFSecret` 字段——保持不变（我们用 `CSRFTrustedOrigins`，不叫 `CSRFSecret`），所以这个守卫测试**继续通过、无需改动**。但我会加一个测试断言 `CSRFTrustedOrigins` 字段存在且能从环境读取。

## 修复 2：后端 CSRF 中间件注册

**文件**：`apps/api/app/Http/server.go`
- import 新增 `github.com/gofiber/fiber/v3/middleware/csrf`。
- 在 `NewApp` 中（locale 之后、`registerRoutes` 之前，约 server.go:73）注册 CSRF。**但需在 `/api/v1` group 上注册**以便 GET 也能种 cookie——更准确的做法：在 `registerRoutes` 内 `api := app.Group("/api/v1")` 之后、provider 循环之前 `api.Use(csrf.New(...))`。这样 GET 路由（含 `/auth/session`）会种 token，health 路由（也是 GET，且不受 CSRF 影响）不受干扰。
- CSRF 配置：
  - `Storage: deps.Storage`（共享 Redis）。
  - `CookieName: "csrf_"`（默认）、`CookieSameSite: "Lax"`、`CookieSecure: strings.EqualFold(cfg.AppEnv, "production")`（与 session 一致，生产走 HTTPS）、`CookieHTTPOnly: false`（SPA 必须能读）、`CookiePath: "/"`。
  - `TrustedOrigins: cfg.CSRFTrustedOrigins`。
  - `Extractor: extractors.FromHeader("X-Csrf-Token")`（默认）。
  - 自定义 `ErrorHandler`：把 CSRF 错误映射成统一 envelope，reason 用 `csrf.invalid`（token 缺失/不匹配）/`csrf.origin_invalid`（Origin 不匹配），status 403，走 `apphttp.NewErrorWithFields` 与项目其他错误一致。
- **重要**：CSRF 中间件需访问 `deps.Storage` 与 `cfg`——`NewApp` 已有这两个参数，直接用。locale 中间件已是同样模式（`localeMiddleware(cfg, deps.Options)`）。
- 中文注释说明：为何 group 级注册、为何 TrustedOrigins 必须配代理后的公开 origin、生产 CookieSecure。

## 修复 3：前端 token 注入（单一入口）

**文件**：`apps/web/app/composables/useApiClient.ts`
- `apiHeaders()`（line 49）改造：对非 GET/HEAD 请求，读取 `csrf_` cookie 并附加 `X-Csrf-Token` header。读取方式：
  - client 端：`useCookie('csrf_').value`（Nuxt 自动 cookie，同源可读）。
  - server 端（SSR）：从已透传的 `useRequestHeaders(['cookie'])`（line 51 已有）中解析 `csrf_`，避免 SSR 阶段丢失 token。新增小 helper `csrfTokenFromContext()` 处理两端的取值。
- 由于 `apiHeaders` 被 `request`、`useWebOptions.fetchEnvelope`、`database.vue` 三处共用，一处改造即覆盖。
- **保证首次 GET 种 token**：SPA 首屏会调 `/auth/session`（GET，app.vue 启动），后端种下 `csrf_` cookie，后续 unsafe 请求自动带上。无需专门的 token 端点。

**文件**：`extensions/builtin/themes/sforum-default/layer/app/components/SFNavbar.vue`（line 76）
- 当前 logout 用裸 `$fetch` 绕过 `request`，不带任何 header。改为调用 `useApiClient` 的 `request('/auth/logout', { method: 'POST' })`，让 CSRF header 自动注入。这是审计标记的"两个绕过点"之一。

## 修复 4：部署配置同步

**文件**：`compose.yaml`
- api/worker environment 新增 `CSRF_TRUSTED_ORIGINS: ${CSRF_TRUSTED_ORIGINS:-}`（默认空，由 APP_URL 派生）。

**文件**：`.env.production.example`
- 新增 `CSRF_TRUSTED_ORIGINS=https://forum.example.com`（与 APP_URL 一致，注释说明多域名/通配符写法，以及代理后必须配置的原因）。

**文件**：`docs/development-and-deployment.md`
- Important production variables 列表新增 `CSRF_TRUSTED_ORIGINS`，注明"API 在反向代理后时必须列出公开站点 origin，否则所有写请求被 CSRF 拒绝"。更新现有 CSRF 备注（之前写的是"待落地"）。

**文件**：`docs/security-review-2026-07-09.md`
- 更新 P1 CSRF 条目为已修复，备注实施方式与 CSRF_TRUSTED_ORIGINS 配置要求。

## 修复 5：OpenAPI 契约标注

CSRF 不改变请求/响应 schema，但 unsafe cookie-auth 端点现在要求 `X-Csrf-Token` header + `csrf_` cookie。在 OpenAPI 安全层面标注：
- 在 `contracts/openapi/openapi.yaml`（或 index）的 security scheme 处补充说明 unsafe 端点需 CSRF token，或在每个 path 的 unsafe operation 添加 `parameters` 中的 `X-Csrf-Token`（可选 header）。倾向于在文档/描述层说明而非每个 operation 重复加参数，避免契约膨胀。先看现有 security scheme 结构再定具体写法。

---

## 修复 6：测试（允许/拒绝路径全覆盖）

**后端中间件测试**（`apps/api/app/Http/server_test.go`，仿 `TestLimiterBlocksExcessiveWriteRequests` server_test.go:102）：
- 新增 CSRF 中间件测试组，用 `testConfig()` + `routeProviderFunc` 注册 `GET /safe` 与 `POST /write` 探测路由：
  1. `GET /safe` 无 token → 200，且响应种下 `csrf_` cookie（捕获 `resp.Cookies()` 按 name 选）。
  2. `POST /write` 无 token + 无 csrf cookie → 403（`csrf.invalid`）。
  3. `POST /write` 有 csrf cookie 但缺 header → 403。
  4. `POST /write` cookie + 匹配 `X-Csrf-Token` header → 200。
  5. `POST /write` cookie + 错误 header → 403。
  6. Origin 校验：`POST /write` 带正确 token + 匹配 TrustedOrigin → 200；带未授权 Origin → 403（`csrf.origin_invalid`）。
- cookie 处理沿用现有 `req.AddCookie` 模式（按 name 选 cookie，非 `Cookies()[0]`，因可能多 cookie）。

**后端 controller 回归测试**：
- `Forum/controller_test.go`、`Identity/controller_test.go` 现有测试若用 `apphttp.NewApp` 构建应用，会自动带 CSRF 中间件。需检查这些测试的 unsafe 请求是否会因缺 token 而失败——若失败，提供测试辅助在 unsafe 请求前先 GET 种 token 再带上（封装成 helper，避免每个测试手写）。**这是关键工作量点**：评估现有 controller 测试改造成本，可能需要一个 `withCSRFToken(t, app, req)` 测试 helper。
- Identity controller_test.go（本批新建的 password-reset 测试）同样需适配。

**config 测试**：`envStringSlice` 解析、`CSRF_TRUSTED_ORIGINS` 读取、默认从 APP_URL 派生。

**前端**：typecheck 通过即可（CSRF 注入是运行时行为，无单测框架，靠手动 dev server 验证）。

## 收尾
- 知识库：新增/更新 `knowledge/decisions/2026-07-09-security-fixes.md` 记录 CSRF 决策（Fiber 自带中间件、double-submit、CSRF_TRUSTED_ORIGINS、覆盖 login/register 防登录型 CSRF、Redis storage）；更新 `knowledge/index.md` 状态。
- 运行 `./scripts/test.sh` 全套 + `ruby scripts/validate-openapi-refs.rb`。
- 提交前 `git status` 确认范围。

## 风险与注意
- **现有 controller 测试可能大面积受影响**：它们用 `apphttp.NewApp`，CSRF 中间件启用后 unsafe 请求需带 token。这是最大不确定项，需先跑一次看失败范围，再用 helper 统一适配。如果改造量过大，备选方案是让 CSRF 中间件在 `AppEnv=="test"` 时通过 `Next` 跳过——但**不推荐**（会削弱 CSRF 的测试覆盖）；优先用 helper 适配。
- **SSR token 透传**：SSR 阶段后端种的 csrf_ cookie 需透传回 API（首屏 GET 在 SSR 执行）。需验证 `useRequestHeaders(['cookie'])` 是否包含上游种的 csrf_，可能需要在 Nitro 侧处理 set-cookie 透传。这是前端验证重点。
- **首次访问无 token**：用户首次访问、cookie 尚未种下时，不能发起 unsafe 请求。正常流程（先加载页面→GET 种 token→再交互）天然满足，但需确认无页面在首屏就 POST。
- **代理后 c.Scheme()**：若代理未透传 X-Forwarded-Proto，API 可能误判为 HTTP，导致 HTTPS 的 Referer 兜底失效（但 token 校验仍生效）。建议在文档提示反向代理透传该头。
- 不擅自 kill 用户在 3000 端口的 dev server；前端验证由用户人工确认。