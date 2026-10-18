# 浏览器访问受登录守卫保护的 stream 路由时跳登录页 - Task Book

Status: **completed (2026-10-18)**
Date: 2026-10-18
影响仓库：SForum（Core，`apps/api`）；插件侧零改动
需求来源：`sforum-oauth2-provider` 的授权确认页 `GET /api/oauth/consent`（`mode: stream`、
`guard: core.guard.login`）在未登录时返回 JSON 401，浏览器里就是一行裸 JSON，没有返回入口。

---

## 1. 一句话

当**浏览器要网页**（`GET`/`HEAD` + `Accept: text/html`）访问一条**产出 HTML 的 stream 路由**、
而 `core.guard.login` 判定未登录时，宿主回 `302 /login?redirect=<原路径+查询>`，而不是 JSON 401。
其他所有情形（API 客户端、SSE、非 stream 模式、非安全方法）保持现状。

---

## 2. 现状（代码证据）

| 事实 | 位置 |
| --- | --- |
| 守卫失败被统一映射成 401 JSON | `apps/api/app/Http/route_dispatcher.go:685` `mapRouteDispatchError` → `fiber.NewError(401, "auth.required")` |
| 错误 envelope 只出 JSON，不做内容协商 | `apps/api/app/Http/errors.go` `errorHandler` → `ErrorResponse(c, status, reason)` |
| stream 路由的守卫在 `Open()` 内失败并**原样**返回该错误 | `apps/api/app/Support/Routes/stream_dispatcher.go:157` → `finishStreamOpenError(..., guardFailure=true)` → 返回未包装的 `err` |
| 失败时尚未写任何响应字节 | `apps/api/app/Http/route_dispatcher.go:403` `serveRouteStream`：`start, err := dispatch.Open(...)`，err 分支直接 `mapRouteDispatchError` |
| Open 失败后仍能读到路由 mode | `RouteStreamDispatch.Step()`（`apps/api/app/Support/Routes/stream_dispatcher.go:120`） |
| 触发本需求的插件端点 | `sforum-oauth2-provider` 的 manifest：`path=/api/oauth/consent`、`mode=stream`、`guard=core.guard.login` |
| 站内登录页已支持回跳 API 端点 | `apps/web/app/utils/identity/authReturn.ts`：`normalizeAuthReturnPath`（拒 `//` 与跨源）+ `requiresDocumentNavigation`（`/api/`、`/plugin/` 前缀走整页导航） |

实测（dev API，`127.0.0.1:3000`）：

- `GET /api/oauth/consent?request=sfoq_x`（无 cookie）→
  `401 {"code":401,"message":"请先登录。","data":{"reason":"auth.required"}}`
- `GET /api/oauth/authorize?...`（插件自己 302）→ `302 /login?redirect=...`

---

## 3. 为什么在 Core 做

1. **插件拦不住**：守卫在插件 handler 之前执行，插件代码一行都没跑，没有机会改回应方式。
2. **会扩散**：stream 是宿主唯一能吐 HTML 的模式，任何想提供浏览器页面的插件都会撞上同一堵墙。
3. **保持 fail-closed**：插件继续声明 `core.guard.login`，安全默认值不变；Core 只是把"拒绝"
   的表达方式换成浏览器能理解的那一种。

---

## 4. 设计

### 4.1 触发条件（四条同时满足，缺一不可）

1. `errors.Is(err, ErrRouteLoginRequired)`
2. `dispatch.Step().Mode == extensionmanifest.RouteModeStream`
3. `c.Method()` 是 `GET` 或 `HEAD`
4. `c.Get(fiber.HeaderAccept)` 包含 `text/html`

### 4.2 目标地址

```text
/login?redirect=<url.QueryEscape(path + "?" + rawQuery)>
```

- 只用 `c.Path()` + `c.Request().URI().QueryString()`；**禁止**用 `Referer`、`Host`、
  `X-Forwarded-*` 构造目标。
- 目标必须是站内绝对路径：`strings.HasPrefix(target, "/")` 且不以 `//` 开头，且不含
  `\`、`\r`、`\n`。任一条不满足 → 退化为 `Location: /login`（不带 `redirect` 参数）。
- 响应加 `Cache-Control: no-store`（URL 里可能带一次性令牌）。

### 4.3 不改变的部分

- 非 stream 模式（http / sse / multipart / websocket）行为完全不变。
- 非浏览器请求（curl、SDK、EventSource）仍是 401 JSON envelope，`reason` 仍是 `auth.required`。
- 守卫失败的 trace 照常记录（`finishWithoutTrace` 已在 `Open()` 内执行），跳转不额外记事件。

### 4.4 备选（不采用）

在 `errorHandler` 里做内容协商：那里拿不到路由 mode，会把普通 API 的 401 一起改掉。不采用。

---

## 5. 实现

文件：`apps/api/app/Http/route_dispatcher.go`，函数 `serveRouteStream`。

```go
start, err := dispatch.Open(c.Context())
if err != nil {
	if location, ok := streamLoginRedirect(c, dispatch, err); ok {
		c.Set(fiber.HeaderCacheControl, "no-store")
		return c.Redirect().Status(fiber.StatusFound).To(location)
	}
	return mapRouteDispatchError(err)
}
```

新增两个辅助函数（同文件，紧跟 `serveRouteStream`）：

```go
// streamLoginRedirect 把「浏览器访问受登录守卫保护的 HTML 流」翻译成跳转。
// 只有 stream 模式 + 浏览器文档导航 + 登录守卫失败才成立；其余情形交回 401 JSON。
func streamLoginRedirect(c fiber.Ctx, dispatch *routes.RouteStreamDispatch, err error) (string, bool) {
	if dispatch == nil || !errors.Is(err, ErrRouteLoginRequired) {
		return "", false
	}
	if dispatch.Step().Mode != extensionmanifest.RouteModeStream {
		return "", false
	}
	if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
		return "", false
	}
	if !strings.Contains(c.Get(fiber.HeaderAccept), "text/html") {
		return "", false
	}
	target := c.Path()
	if query := string(c.Request().URI().QueryString()); query != "" {
		target += "?" + query
	}
	if !safeLoginReturnPath(target) {
		return "/login", true
	}
	return "/login?redirect=" + url.QueryEscape(target), true
}

// safeLoginReturnPath 保证回跳目标是站内绝对路径，挡住 open redirect 与 header 注入。
func safeLoginReturnPath(target string) bool {
	return strings.HasPrefix(target, "/") &&
		!strings.HasPrefix(target, "//") &&
		!strings.ContainsAny(target, "\\\r\n")
}
```

依赖说明：`errors`、`strings`、`net/url`、`fiber`、`extensionmanifest`、`routes` 均已在
`route_dispatcher.go` 的 import 中，无需新增。

---

## 6. 安全

| 风险 | 处理 |
| --- | --- |
| open redirect | 只接受 `/` 开头且非 `//` 的路径；`/login` 页自身还有 `normalizeAuthReturnPath` 二次校验 |
| header 注入 | 整个目标经 `url.QueryEscape`；拒绝 `\r\n`、`\` |
| 令牌泄露进日志 | 目标只来自当前请求，不新增日志；不要 log 完整 URL |
| 端点存在性探测 | 只有 stream 路由会 302，非 stream 仍 401 —— 与现状一致（路由存在性本来就通过 404/401 可探测） |
| CSRF | GET/HEAD 不触发 CSRF 中间件，无副作用 |

---

## 7. 测试计划

新增 `apps/api/app/Http/route_dispatcher_stream_login_redirect_test.go`，复用
`route_dispatcher_stream_test.go` 的 invoker 脚手架（`streamHTTPTestInvoker`）。

| # | 用例 | 期望 |
| --- | --- | --- |
| 1 | GET + `Accept: text/html` + 未登录 + stream 路由 | 302，`Location` 精确等于 `/login?redirect=%2Fapi%2Foauth%2Fconsent%3Frequest%3Dsfoq_test` |
| 2 | 同上但 `Accept: application/json` | 401，envelope reason `auth.required` |
| 3 | 同上但不带 `Accept` | 401 |
| 4 | `Accept: text/event-stream` + `mode: sse` + 未登录 | 401（SSE 不跳转） |
| 5 | `POST` + `Accept: text/html` | 401 |
| 6 | 请求路径 `//evil.example/x` + `Accept: text/html` | 302，`Location` 恰好是 `/login`（无 `redirect` 参数） |
| 7 | 已登录 + `Accept: text/html` | 行为不变（正常进入 handler） |
| 8 | 非守卫错误（如 runtime transport 失败） | 行为不变（原有映射） |

另在 `route_dispatcher_stream_preflight_test.go` 补一条回归：302 分支不写任何 body。

---

## 8. 验收（本地 dev）

dev API 由 `air` 常驻在 `~/Code/github/SForum/apps/api`，改完自动重编译。

```bash
# 1. 浏览器视角：应 302 到 /login?redirect=...
curl -s -i -H 'Accept: text/html' \
  'http://127.0.0.1:3000/api/oauth/consent?request=sfoq_deadbeef' | head -5

# 2. API 视角：应仍是 401 JSON
curl -s -i 'http://127.0.0.1:3000/api/oauth/consent?request=sfoq_deadbeef' | head -5

# 3. 回归：authorize 自己的 302 不受影响
curl -s -o /dev/null -w '%{http_code} %{redirect_url}\n' \
  'http://127.0.0.1:3000/api/oauth/authorize?client_id=sfo_client_fceaa85a41f3b9eafeb9ff9d&redirect_uri=http%3A%2F%2F127.0.0.1%3A3000%2F&response_type=code&scope=profile&state=abc'

# 4. 单测
cd apps/api && go vet ./app/Http/... && go test ./app/Http/... ./app/Support/Routes/...
```

浏览器闭环（推荐）：未登录打开 `/api/oauth/authorize?...` → 跳到 `/login?redirect=...` →
登录后回到 authorize → 确认页正常显示 → 允许 → 回跳带 `code`。

---

## 9. 不做的事

- 不改 `errorHandler`，不改 `mapRouteDispatchError` 的其它分支。
- 不动 `core.guard.login` 语义，不动任何插件 manifest。
- 不为 http 模式路由加同类跳转（http 路由本来就能自己 302）。
- 不做"登录后自动回到原请求"的服务端会话逻辑（Nuxt 登录页已经处理）。
- 不引入配置开关；确需开关先提出来讨论。

---

## 10. 回滚

单文件改动 + 一个新增测试文件，`git revert` 即可，无数据与契约影响。

---

## 11. 实施记录（2026-10-18）

**改动文件**

- `apps/api/app/Http/route_dispatcher.go`（706 → 743 行）：`serveRouteStream` 的
  `dispatch.Open` 错误分支加 `streamLoginRedirect` 短路 + `Cache-Control: no-store`；
  新增 `streamLoginRedirect` / `safeLoginReturnPath` 两个函数。无新 import、无配置开关。
- `apps/api/app/Http/route_dispatcher_stream_login_redirect_test.go`（新增）：10 个用例，
  复用 `streamHTTPTestInvoker` / `streamHTTPTestSession` / `routeRegistryPlanResolver` 脚手架。
- 未改：插件仓库、Nuxt、`errorHandler`、`mapRouteDispatchError`、其他测试文件。

**测试结果**（`go vet ./app/Http/...` 通过；`go test ./app/Http/... ./app/Support/Routes/...` 全绿）

| # | 用例 | 结果 |
| --- | --- | --- |
| 1 | `TestStreamLoginRedirectSendsBrowserToLogin`（302 + 精确 `Location` + `no-store` + 空 body） | PASS |
| 2 | `TestStreamLoginRedirectKeepsJSONEnvelopeForAPIClients` | PASS |
| 3 | `TestStreamLoginRedirectRequiresHTMLAccept` | PASS |
| 4 | `TestStreamLoginRedirectIgnoresSSEMode`（`text/event-stream` 与 `text/html` 两个子用例） | PASS |
| 5 | `TestStreamLoginRedirectIgnoresUnsafeMethods` | PASS |
| 6 | `TestStreamLoginRedirectFallsBackToPlainLogin`（7 条 `safeLoginReturnPath` 表 + 合成 `//` ctx） | PASS |
| 7 | `TestStreamLoginRedirectPassesAuthenticatedBrowsersThrough` | PASS |
| 8 | `TestStreamLoginRedirectIgnoresNonGuardFailures` | PASS |
| 9 | `TestStreamLoginRedirectSupportsHEAD` | PASS |

§7 里「在 `route_dispatcher_stream_preflight_test.go` 补 body 空回归」并入用例 1（硬约束不允许改该文件）。

**§8 实机验收**（dev API 实际监听 `HTTP_PORT=8080`，3000 是 Nuxt dev server，因此命令改打 8080）

| 命令 | before | after |
| --- | --- | --- |
| `curl -H 'Accept: text/html' .../api/oauth/consent?request=sfoq_deadbeef` | `401` JSON `{"code":401,...,"reason":"auth.required"}` | `302`，`Location: /login?redirect=%2Fapi%2Foauth%2Fconsent%3Frequest%3Dsfoq_deadbeef`，`Cache-Control: no-store`，`Content-Length: 0` |
| 同 URL 不带 `Accept` | `401` JSON | `401` JSON（不变） |
| `curl .../api/oauth/authorize?...` | `302 /login?redirect=...`（插件自己发） | `302 /login?redirect=...`（不变） |
| `go test ./app/Http/... ./app/Support/Routes/...` | 全绿 | 全绿 |

补充实机矩阵（`/api/oauth/consent`）：`Accept: text/html` → 302；`application/json` → 401；
`text/event-stream` → 401；`*/*`（curl 默认）→ 401；`text/html;q=0.9` → 302；
`HEAD` + `text/html` → 404（该路由只声明 `GET`，隐式 HEAD 仅对 http 模式生效）。

**取证注意**：首次在 10:30 复现时该端点返回 **404**，原因是 `sforum.oauth2-provider`
处于 dev-auto 升级失败循环（`extension_database_runtime_leases` 泄漏租约导致
`reserve runtime lease capacity: extension database runtime lease conflict`），插件运行时未起、路由未发布。
air 在改动后重编译时该冲突被清掉，路由随之恢复发布。属 dev 环境问题，与本改动无关。

### 11.1 已登录会话的实机复核（2026-10-18，补做）

用真实浏览器会话（`sforum_session` cookie，host-scoped、跨端口生效）对同一端点复核：

| 场景 | 结果 |
| --- | --- |
| 已登录 + `Accept: text/html` → `/api/oauth/consent?request=sfoq_…` | `200`，`Content-Type: text/html; charset=utf-8`，46 KB 授权确认页（**不跳转**） |
| 同一 URL 去掉 cookie + `Accept: text/html` | `302`，`Location: /login?redirect=%2Fapi%2Foauth%2Fconsent%3Frequest%3Dsfoq_…`，`Cache-Control: no-store` |
| 真实浏览器（ego，带会话）打开 `/api/oauth/authorize?...` | 落在 `http://127.0.0.1:8080/api/oauth/consent?request=…`，标题「授权确认 | SForum」 |
| 授权确认 POST `/api/oauth/consent/decision`（`allow`） | `200 {"status":"redirect","location":"http://127.0.0.1:3000/?code=sfoc_…&state=abc"}` |

即：守卫通过时 stream 路由照常吐 HTML，未登录时同一 URL 才跳转，插件自己的确认→回跳链路未受影响。

**dev 拓扑限制（不是本改动的缺陷）**：跳转目标是站内绝对路径 `/login`，浏览器会按当前 origin 解析。
dev 下 API 在 `:8080`、登录页在 Nuxt `:3000`，`8080/login` → 404、`3000/login` → 200，所以浏览器
跟随 302 后在 dev 里会看到 API 的 404。生产由 `deploy/caddy/Caddyfile` 在同一 origin 同时服务
API 与前端，`/login` 才能正确命中 Nuxt 登录页。任务书 §4.2 明确禁止用 `Host`/`Referer` 构造目标，
因此这里不做 origin 改写。

**发现（不在本任务范围）**

1. `HEAD` 对 stream 路由只有显式声明时才可达（`Support/Routes/prepare.go:424-429`），本改动的 HEAD 分支属防御性代码。
2. 以 `//` 开头的请求路径在 `Support/Routes/path.go:110-121` 的规划阶段就被拒绝，永远到不了
   `Open()`，因此 `Location: /login` 退化分支当前只能单元验证（用例 6）。
3. `ErrRoutePermissionDenied` / `ErrRouteGuestRequired` 在 stream 路由上对浏览器仍返回 JSON —— 同类问题，按 §9 不做。
4. dev watcher 重建循环与插件 lifecycle 升级失败仍在发生（`knowledge/index.md` 2026-10-18 手记称已修），值得另开任务核查。
5. dev 下 `/api/oauth/*` 只能在 `:8080` 直接访问（Nuxt 只代理 `/api/v1`），因此「未登录点开授权页 → 浏览器落到登录页」
   在 dev 里无法单 origin 走通；生产单 origin 下没有这个问题。

