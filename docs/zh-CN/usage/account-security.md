# 账户与安全

[← 使用说明](./README.md)

面向站长与普通用户：邮箱验证、密码、登录设备、访问令牌、外部登录绑定、
个人外观与通知设置。

## 邮箱验证

- 若站点启用了邮箱验证强制（Site Settings → 账户安全），未验证用户会被
  引导到等待验证页 `/email-verification`。
- 验证邮件为一次性哈希链接；链接已使用后立即失效。可在等待页主动发送验证
  邮件（需要先完成 ALTCHA 人机验证，页面不会自动发送）。
- 完成验证后即可正常发布主题/评论/上传附件；未验证用户在未强制模式下发布
  权限由站点策略决定。
- 站长可在后台查看用户验证状态，并手动“验证/重置”用户邮箱状态（重置会使
  旧链接全部失效）。
- API：`POST /auth/email-verification/request`、`POST /auth/email-verification/confirm`。

## 密码找回与修改

- **找回**：登录页 → 忘记密码（`/forgot-password`），输入邮箱后发送重置
  邮件；重置链接进入 `/reset-password` 设置新密码。找回流程不会泄露账号
  是否存在（非枚举）。
- **发送频率**：请求/重发受服务器侧冷却与频率限制（Site Settings → 账户
  安全），倒计时显示在页面上。
- **修改**：登录后进入 [设置 → 密码](./README.md#章节)（`/settings/password`）
  设置或修改本地密码；修改需要最近一次认证（recent-auth）确认。
- 未设置过本地密码的账号（例如只用外部登录注册）可在该页面补设。
- API：`POST /auth/password-reset/request`、`POST /auth/password-reset/confirm`、`POST /auth/password`。

## 登录设备 / 会话撤销

设置 → 账户安全（`/settings/security`）会列出当前账号的登录会话：

- 查看当前会话与历史会话（浏览器/设备信息）；
- 单点撤销某个设备会话；
- “撤销其他会话”可一次性登出当前设备之外的所有会话；
- 修改密码等敏感操作后，建议立即撤销其他会话。

API：`GET /auth/sessions`、`DELETE /auth/sessions/{sessionId}`、`POST /auth/sessions/revoke-others`。

## Personal Access Token（PAT）

设置 → 访问令牌（`/settings/tokens`）用于创建脚本/服务调用 API 的令牌：

- 令牌格式 `sft_<publicId>.<secret>`，secret 只在创建时显示一次，请立即保存；
- Scope 只能选择你当前已持有的权限键；
- 可随时吊销或轮换令牌；管理令牌本身需要 Cookie 会话（API 拒绝用 Bearer
  管理令牌，返回 `api_token.cookie_required`）。

API 调用方式见[开发指南-API 使用](../development/api.md)。

## 外部登录绑定

- 站点可启用外部登录（如内置 GitHub 登录）；在登录页选择对应方式即可。
- 未绑定的外部账号会进入 `/auth/continue`：可证明已有本地账号后自动绑定，
  或走注册流程后自动绑定；系统不会用邮箱自动匹配账号。
- 设置 → 登录方式（`/settings/login-methods`）可查看/解绑已绑定的外部身份。
- API：`GET /auth/providers`、`GET /auth/external-identities` 等，见
  OpenAPI `contracts/openapi.yaml`。

## 个人外观与通知设置

- **外观**：设置 → 外观（`/settings/appearance`）选择浅色/深色/跟随系统，
  以及个人配色背景；支持即时预览与“恢复站点默认”。
- **通知**：设置 → 通知（`/settings/notifications`）管理站内信、邮件与
  浏览器通知（Web Push）的接收偏好；站点管理员可以在后台限制可选项。
- **资料**：设置 → 资料（`/settings/profile`）维护用户名、简介、头像与
  界面语言偏好。

## 相关

- 管理侧策略（强制验证、冷却、密码策略、通知策略）：[管理后台](./admin.md)。
- API 契约：[开发指南-API 使用](../development/api.md)。
