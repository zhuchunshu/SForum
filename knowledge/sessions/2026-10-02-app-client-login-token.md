# 2026-10-02 Session Handoff

## Changed

- `POST /auth/login` 新增可选 `issueApiToken`：一次密码登录同时签发 PAT，
  支撑原生 App / 机器客户端。`apps/api/app/Http/Controllers/Identity/api_tokens.go`
  新增 `issueAPITokenRequest`、`loginIssuedAPIToken`、`parseAPITokenExpiry`、
  `validateLoginAPITokenRequest`、`issueLoginAPIToken`；`createAPIToken` 改为复用
  同一个过期时间解析；`auth_handlers.go` 在会话签发前做形状校验、在 recent-auth
  标记后签发令牌并按需切换响应形状。
- 新增本地化原因键 `api_token.name_required`、`api_token.scopes_required`
  （zh-CN / en-US）。
- 契约：`contracts/openapi/paths/identity.yaml` 的 `authLogin` 改用
  `ApiResponseLogin` 并补充流程说明；`schemas/identity.yaml` 新增
  `ApiResponseLogin`、`LoginIssuedToken`，`LoginRequest` 新增 `issueApiToken`
  与 `stepUpEvidence`。
- 文档：`docs/zh-CN/development/api.md`、`docs/en-US/development/api.md` 新增
  「原生 App / 机器客户端：登录即换令牌」章节。
- 测试：`apps/api/app/Http/Controllers/Identity/login_api_token_test.go`
  （内存 PAT store + Bearer 复用断言 + 形状校验 + 越权 scope 拒绝 + 未接线 503）。

## Decisions

- 复用 `POST /auth/login` 而不是新增 `POST /auth/tokens/exchange`：凭证校验
  （锁定、风控、人机验证、会话策略、登录审计）必须只有一个权威。
- `scopes` 保持显式，不提供「继承全部权限」默认值：权限目录加插件权限键已接近
  `MaxScopes=64`，隐式继承既是特权放大也可能直接失败。
- 明文令牌只在登录响应出现一次；令牌管理端点继续只接受 cookie 会话。
- 未新增客户端 bootstrap 聚合端点：公共站点配置已由 `GET /web-options` 单一投影
  覆盖，再加一层聚合会产生第二个所有者与漂移风险。

## Next

- App 登出服务端自吊销：需给 `routes.DispatchRequest` 增加 bearer 令牌身份，并在
  `core.guard.identity.self_credentials` 中为 `revoke_apitoken` 增加「仅可吊销自身
  令牌」的规则。
- App 版本门禁与弃用信号：目前全仓无 `Deprecation`/`Sunset` 响应头，也没有最低
  支持版本；需要先在 option 注册表加运营项与后台界面，再暴露客户端读取端点。
- 原生推送通道（FCM/APNs）应做成通知通道插件；Host 仍需补「设备注册」契约。
- 第三方登录原生回跳（自定义 scheme / Universal Link + PKCE）未动。

## Open Questions

- 是否允许 PAT 自吊销（放宽 guard 姿态）还是维持「登出只清本地令牌」。
- App 冷启动是否需要 `/web-options` + `/auth/registration-status` 之外的东西；
  目前判断不需要聚合端点。
