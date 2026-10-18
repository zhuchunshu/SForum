# 2026-10-18 Session Handoff — 站点时区保存失败修复

## Changed

- 新增 `apps/api/app/Support/Timezone`：blank import `time/tzdata` +
  `Load(name)`；宿主侧时区解析统一入口。
- `app/Models/Options/site_datetime_options.go`、
  `app/Support/ExtensionManifest/v3_validate.go`、
  `app/Support/Jobs/plugin_schedule_periodic.go` 三处 `time.LoadLocation`
  改为 `timezone.Load`（`pluginPeriodicSchedule` 参数改名 `zone` 避免遮蔽包名）。
- 新增 `tests/validate-runtime-tzdata.mjs`，接入 `scripts/test.sh`。

## Root Cause

运行镜像 `apps/api/Dockerfile` 最终阶段是 `alpine:3.22`，只装
`ca-certificates`；无 `/usr/share/zoneinfo`，二进制 `CGO_ENABLED=0` 且无
`time/tzdata`。`normalizeSiteTimezone` 的 `time.LoadLocation` 因此拒绝所有
非 UTC 时区 → `UpdateMany` 返回 `ErrInvalidOption` → 422 `options.invalid`
→ 后台只看到通用「保存失败」。`UTC` 被 Go 特判所以正常。

证据：

- 发布镜像 `ghcr.io/zhuchunshu/sforum-api:v3.0.9` 内静态探测：
  `Asia/Shanghai => unknown time zone`，`UTC => UTC`；镜像无 zoneinfo；
  二进制内无内嵌时区数据。
- 本机 dev 实测保存 `Asia/Shanghai` 成功（host 有 zoneinfo），说明只在容器
  运行环境复现。
- 修复后：`CGO_ENABLED=0` 构建的探针在 `alpine:3.22` 内解析
  `Asia/Shanghai` / `America/New_York` / `Europe/London` / `Australia/Sydney`
  全部成功。

## Decisions

- 内嵌 tzdata，不在镜像安装：`decisions/2026-10-18-embed-tzdata.md`。

## Next

- 重建并发布 `api` 镜像（worker 同镜像），在部署实例后台保存一次
  `Asia/Shanghai` 做端到端确认（200、落库、刷新保持、公开页 +08:00）。
- 若失败实例其实是本机 dev：本机已验证可保存，需要抓一次失败请求的 422
  响应体再定向；容器根因独立成立，仍应发布本次修复。
- 可选后续：`options.invalid` 目前不带字段名，管理端无法定位具体选项；
  若要提升可诊断性，可给选项校验加字段级错误（需要改契约与前端映射）。

## Open Questions

- 无。
