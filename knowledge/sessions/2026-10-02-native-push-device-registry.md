# 2026-10-02 Session Handoff

## Changed

- 原生推送设备注册（Core 侧契约）：
  - 迁移 `apps/api/database/migrations/202610020004_notification_push_devices.sql`
    （`push_devices`：user_id/device_id/platform/token_hash/token_ciphertext/
    app_version/locale/device_name/status/last_seen_at，唯一约束
    `(platform, token_hash)`）。**注意版本号**：原为 `202610020001`，与并发的 AI
    工作流重号且已被 goose 应用，改为 `202610020004` 并已在本机开发库应用。
  - `app/Models/Notifications/devices.go`：`PushDevice` 公开视图、领域校验
    `NormalizePushDeviceInput`（平台白名单 ios/android/desktop、deviceId 8-128、
    令牌 16-4096 可打印 ASCII、appVersion/locale/deviceName 长度）、注册幂等 upsert、
    改绑、同 deviceId 旧令牌自动撤销、按 deviceId 撤销、令牌哈希 + Core 密钥密文。
  - `app/Http/Controllers/Notifications/devices.go` + 路由
    `GET/POST /api/v1/push/devices`、`DELETE /api/v1/push/devices/:deviceId`；
    控制器先做领域校验再进 store；审计事件
    `notification.push_device.register` / `.revoke`；新增 5 条本地化 reason。
  - bootstrap 注入 `optionCipher`（`WithPushDeviceCipher`），生产环境令牌不以明文落库。
- 契约：`contracts/openapi/paths/notifications.yaml` 三个 operation + 三个 schema，
  `contracts/openapi.yaml` 索引两条；OpenAPI 引用 493 条全部解析。
- 文档：双语 API 手册新增「原生推送设备注册」。
- 目录生成物：`scripts/v3-catalog/generate.mjs` 新增
  `/api/v1/push/devices` 的 `login` 守卫分类与 reviewed 映射；
  `catalog-identities.json` 新增三条路由身份；已重新生成全部 15 个目录文件，
  `--check` 通过。
- **代并发工作流登记的身份**（需要他们知晓）：为了让生成器可运行（否则我的新路由
  不会进入 Core Route Catalog，CSRF/Bearer 中间件不生效），按生成器约定补了三条
  UI 身份：`core.component.admin.sf_admin_ai_observability_tab`、
  `core.component.admin.sf_admin_ai_providers_tab`、
  `core.component.page.admin.ai`。如果他们采用别的 id，需要以他们的为准收敛。

## Decisions

- Core 只拥有「设备令牌 ↔ 用户 ↔ App 实例」的归属与生命周期；FCM/APNs 传输属于通知
  通道 provider 插件，Core 不内置厂商 SDK。
- 令牌按 `(platform, token_hash)` 唯一；重复注册改绑并把同 deviceId 的旧令牌置为撤销，
  覆盖换账号、令牌轮换、重装三种场景。
- 本轮**不**引入 `native_push` 通道键：没有 provider 与投递语义的通道是死配置面，
  通道键 + provider 槽 + 参考 provider 插件应一起落地（下一刀）。
- 决策记录：`decisions/2026-10-02-native-push-device-registry.md`。

## Verification

- `go build ./...`；`go test ./app/Models/Notifications/`（含真库集成测试：
  改绑、令牌轮换、跨用户撤销、密文列不含明文，均在回滚事务内执行）、
  `go test ./app/Http/Controllers/Notifications/`（所有者隔离、越权 404、未认证不触达
  store、422 reason 映射）、`go test ./app/Support/Localization/` 全部通过。
- 迁移已在本机开发库应用（goose version 202610020004）。
- 目录 `--check` 通过；架构边界与文档门禁通过；OpenAPI 引用 493 条解析通过。

## Next

- 下一刀（原生推送通道）：Core 声明 `native_push` 通道键 + provider 槽位选择 +
  参考 provider 插件（FCM/APNs），并把通道投递存储从硬编码 `web_push` 放宽为
  通道白名单；投递目标从 `push_devices` 解析（解密令牌）。
- 设备注册尚未接任何投递路径，因此 App 目前只能注册/列出/撤销。
- 渲染态 Browser QA 仍未做（本会话无 BrowserSkill CLI、桌面工具不可用）。
