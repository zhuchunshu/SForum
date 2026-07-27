# GitHub 登录方式（运营指南）

[← 使用说明](./README.md) · [扩展与主题](./extensions.md)

SForum V1 将 GitHub 登录作为**受保护内置插件** `sforum.auth-github` 提供。
**发现内置包 ≠ 已对外开启登录**。密码登录始终可用；Safe Mode 下第三方登录关闭。

## 前置条件

1. 站点已完成首次注册（首个 `super_admin` 必须走核心注册，不能用 GitHub）。
2. 已配置可信公开地址 `APP_URL`（生产须 HTTPS）。
3. 生产已设置强随机 `IDENTITY_SUBJECT_HMAC_SECRET`（身份绑定备份密钥，轮换需迁移方案）。
4. 你具备扩展管理与 `identity.provider.manage`（或 `super_admin`）。

## 在 GitHub 创建 OAuth App

1. GitHub → Settings → Developer settings → OAuth Apps → New OAuth App。
2. **Authorization callback URL** 必须与 Host 展示的绝对回调一致，形如：  
   `{APP_URL}/api/v1/auth/providers/sforum.auth-github.auth/callback`  
   在「登录方式」页可一键复制。
3. 记录 **Client ID**，生成 **Client Secret**（仅粘贴到 SForum SecretStore，勿写入主题或浏览器）。

官方参考（核验日 2026-07-27）：

- [Authorizing OAuth Apps](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps)

## 在 SForum 启用（推荐顺序）

| 步骤 | 在哪里 | 说明 |
| --- | --- | --- |
| 1. 确认已发现 | 扩展列表 | 内置包经 `SyncBuiltins` 仅**暂存**，默认未信任/未启用/未激活 |
| 2. 精确制品信任 | 扩展详情 | 超级管理员对**当前 digest** 做可执行信任确认 |
| 3. 启用插件 | 扩展详情 | 启用 ≠ 公开登录按钮 |
| 4. 填写 Client ID / Secret | 登录方式或插件设置 | Secret 走 SecretStore，浏览器永不回显 |
| 5. 探测 | 登录方式 | 仅证明配置存在与接口可达；**不能**无 code 证明 Secret 正确 |
| 6. 打开操作 | 登录方式 | 分别打开 **登录 / 注册 / 绑定**，默认全关 |
| 7. 验证 | 访客隐身窗 | 仅当 Host 激活对应操作时，登录/注册页才出现按钮 |

后台入口：`/control-panel` 下 **设置 → 登录方式**（`/admin/settings/login-methods`）。

## 访客能做什么

- **登录**：GitHub 身份已绑定本站账号时，签发普通浏览器会话。
- **显式注册**：未绑定且站点允许注册时，经 GitHub 后到 `/register?ticket=…` 填用户名/邮箱（无密码字段）。
- **绑定 / 解绑**：账号安全页；解绑最后一种登录方式会被拒绝，需先设本地密码。
- **外部账号设密码**：账号安全 → 本地密码；需最近认证。

## 生命周期注意

| 事件 | 对访客的影响 |
| --- | --- |
| 重启 API | Redis 回调状态仍有效（TTL 内）；HMAC 密钥必须稳定 |
| 禁用插件 / 吊销信任 / 卸载 | 按钮消失；已有绑定保留为惰性记录 |
| 上传/暂存新 digest | **不会**自动继承激活；需对新 digest 再确认并重新打开操作 |
| Safe Mode / ForceDrain | 第三方入口关闭；密码登录与 Host 恢复仍可用 |
| 流程中途制品变更 | 进行中的 callback fail closed，需重新开始 |

恢复推荐默认：关闭 login/registration/link，**保留** Client Secret（UI 会说明）。

## 排障

| 现象 | 排查 |
| --- | --- |
| 管理可见、访客无按钮 | 是否 Host 激活了对应操作？插件是否启用且配置齐全？是否 Safe Mode？ |
| 回调后提示过期/重放 | 10 分钟 TTL；勿刷新回调 URL；检查多实例是否共享 Redis |
| `auth.provider_unavailable` | 制品漂移、未配置、运行时未就绪、探测失败 |
| `auth.external_identity_unlinked` | 该 GitHub 账号尚未绑定；引导注册或先密码登录再绑定 |
| 生产启动失败 | 检查 `IDENTITY_SUBJECT_HMAC_SECRET` / `APP_URL` |
| 限流 `rate_limit.exceeded` | start/callback 按 IP 限流；稍后重试 |

**安全**：日志、审计、API、浏览器历史、诊断中都不应出现 code、token、Client Secret、PKCE verifier、raw state、raw subject、subject digest 或上游错误正文。若发现泄漏，立即轮换 GitHub Client Secret 并报告。

## 相关文档

- [扩展与主题](./extensions.md)
- [首次注册](./first-login.md)
- 插件作者：[sforum-auth-github README](../../../extensions/builtin/plugins/sforum-auth-github/README.md)
- 决策：`knowledge/decisions/2026-07-27-github-social-login-builtin-v1.md`
