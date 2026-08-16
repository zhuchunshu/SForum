# API 使用

[← 开发指南](./README.md)

面向集成方：如何调用 SForum 的 JSON API，包括认证、CSRF、Token 与统一响应
信封。完整契约以 [OpenAPI](../../../contracts/openapi.yaml) 为准，本页只是
入门入口，禁止臆造端点。

## 基础

- Base URL：`/api/v1`（浏览器端由 Nuxt 同域反代；服务端/脚本可直连 API
  loopback 端口）。
- 响应信封统一为：`code`（整数，等于 HTTP 状态码）、`message`（后端按请求
  语言本地化）、`data`；稳定的机器可读错误原因位于 `data.reason`，字段级
  错误位于 `data.fields`。

```json
{
  "code": 422,
  "message": "注册失败：请按标出的提示修改后再提交。",
  "data": {
    "reason": "auth.register_invalid",
    "fields": { "username": ["请填写用户名。"] }
  }
}
```

## 认证方式

### 1. 浏览器会话（Cookie）

`POST /auth/register` 与 `POST /auth/login` 成功后由服务端颁发会话 Cookie，
浏览器后续请求自动携带。`GET /auth/session` 返回当前用户。

### 2. Personal Access Token（PAT）

适用于脚本与外部服务：

1. 在 `/settings/tokens`（页面）或 `POST /auth/tokens`（Cookie 会话）创建
   Token。Token 格式为 `sft_<publicId>.<secret>`；secret 只在创建时返回一次。
2. 调用时携带 `Authorization: Bearer sft_<publicId>.<secret>`。
3. Scope 只能是用户已持有的权限键；`super_admin` 的绕过能力会被剥离，PAT
   不能当作无限会话。
4. Token 管理类接口（列出/吊销/轮换）拒绝 Bearer 认证，返回
   `api_token.cookie_required`；创建、删除等不安全操作使用 Cookie 会话时
   仍需 CSRF（详见下文），使用有效 PAT 时跳过 CSRF。

## CSRF

所有 `/api/v1` 下的不安全方法（POST/PUT/PATCH/DELETE）受 double-submit CSRF
保护：

1. 先发一个安全请求（GET/HEAD/OPTIONS）获取可读的 `csrf_` Cookie。
2. 不安全请求携带 `X-Csrf-Token: <csrf_ 的值>` 请求头。
3. 缺失或不匹配返回 `403` + `data.reason = "csrf.invalid"`；`Origin`/`Referer`
   不受信任返回 `403` + `data.reason = "csrf.origin_invalid"`。
4. 受信任来源由 `CSRF_TRUSTED_ORIGINS` 配置（默认等于 `APP_URL` 的来源）。

CSRF 豁免：

- 携带有效 `Authorization: Bearer sft_…` 的请求（非浏览器客户端）跳过 CSRF；
- 入站 Webhook 路径 `POST /webhooks/inbound/{source}` 跳过 CSRF（当前为
  gateway skeleton：只校验非空 body 并回执，插件 verify/parse hooks 尚未
  接入）。

## 幂等

`POST /topics` 与 `POST /topics/{topicID}/comments` 接受**可选**的
`Idempotency-Key` 请求头，用于重试安全；缺失时不视为错误。仅当插件路由在
OpenAPI 中**声明必须幂等**时，缺失/非法键才返回 `400`；其余语义（`409`
请求冲突、`503` 存储失败等）见 OpenAPI 顶部说明。

## 常用入口（节选，以 OpenAPI 为准）

| 用途 | 端点 |
| --- | --- |
| 注册 / 登录 / 登出 | `POST /auth/register`、`POST /auth/login`、`POST /auth/logout` |
| 当前用户 / 语言 / 外观 | `GET /auth/session`、`PUT /auth/locale`、`PUT /auth/appearance` |
| 会话管理 | `GET /auth/sessions`、`DELETE /auth/sessions/{sessionId}`、`POST /auth/sessions/revoke-others` |
| PAT | `POST /auth/tokens`、`GET /auth/tokens`、`DELETE /auth/tokens/{tokenID}`、`POST /auth/tokens/{tokenID}/rotate` |
| 密码 | `POST /auth/password-reset/request`、`POST /auth/password-reset/confirm`、`POST /auth/password` |
| 邮箱验证 | `POST /auth/email-verification/request`、`POST /auth/email-verification/confirm` |
| 外部登录 | `GET /auth/providers`、`POST /auth/providers/{providerId}/{operation}/start`、`POST /auth/providers/{providerId}/{operation}/complete`、`GET /auth/providers/{providerId}/callback`、`GET /auth/external-identities` |
| 入站 Webhook | `POST /webhooks/inbound/{source}`（gateway skeleton，跳过 CSRF） |
| 健康 / 就绪 | `GET /api/v1/health`、`GET /api/v1/ready` |

管理端点（`/admin/…`）要求 `admin.access` 等后台权限，权限以后端策略为准；
可用的权限键与用户/角色管理见[管理后台](../usage/admin.md)。

## 参考

- 完整 OpenAPI：`contracts/openapi.yaml`（入口索引；路径按模块拆分在
  `contracts/openapi/paths/`，模式在 `contracts/openapi/schemas/`）。
- 认证与安全模型：[使用说明-账户与安全](../usage/account-security.md)。
