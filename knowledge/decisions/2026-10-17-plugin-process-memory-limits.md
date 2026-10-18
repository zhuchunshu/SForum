# 插件子进程 GOMEMLIMIT + GOGC 内存上限

日期：2026-10-17
状态：已接受

## 背景

Protocol V2 插件是每插件一个独立 Go 子进程的隔离模型（安全边界，不可合并）。
2026-08-20 治理已做 SDK 依赖解耦与 `GODEBUG=disablethp=1`（SMTP PSS 27.3MiB → 19.3MiB），
之后单进程常驻水位仍有「GC 滞后」空间：Go 默认 GOGC=100 下小堆进程不主动归还内存。

## 决定

在 `buildPluginProcessEnv`（`app/Support/Extensions/plugin_command_transport.go`，
自 protocol.go 抽出以满足 legacy cap 棘轮）向所有插件子进程注入固定默认：

- `GOGC=50`——GC 更积极，小堆低流量进程的 CPU 代价可忽略。
- `GOMEMLIMIT=48MiB`——软上限。实测插件常驻 RSS 25–30MB（PSS ~15–19MiB），
  20MiB 会贴着 live heap 造成 GC 连续抖动，48MiB 留足峰值余量且仍能压住空闲水位。

可选覆盖：宿主进程环境存在 `SFORUM_PLUGIN_GOMEMLIMIT`（非空）时替换默认值；
空值视为未设置回落默认。该键不进入 `pluginEnvAllowlist` 透传循环，仅在构建时特判，
避免把覆盖语义泄漏成通用 env 通道。**已部署实例零配置生效**：不设任何新 env 即为默认值，
由测试 `TestBuildPluginProcessEnvDefaultsMemoryLimitWithoutOverride` 固定。

## 不改变的东西

- Manifest V3 / Protocol V2 线协议与契约文件零变更。
- env 白名单安全边界不变：宿主密钥仍不透传，production 仍剥离 fake-GitHub 端点覆盖
  （原测试矩阵原样保留并通过）。
- 宿主自身的 `GOMEMLIMIT` 不透传给插件（只认 `SFORUM_PLUGIN_GOMEMLIMIT` 覆盖键）。

## 权衡

- 用 GC CPU 换内存：GOGC=50 约使 GC 频率翻倍，但插件进程低流量小堆，
  绝对增量毫秒级/分钟。极端大负载插件（如 AI 抓取处理大响应）单次调用可能慢 10–20%，
  可用覆盖键放宽。
- GOMEMLIMIT 是软上限，超限触发连续 GC 而非 OOM kill；48MiB 默认远高于实测
  live heap，不构成常规路径风险。

## 实测证据

`knowledge/reports/2026-10-17-plugin-memory-limits-recheck.md`：
19 个插件子进程总 PSS 288MiB、平均 ~15.2MiB/进程（对比 08-20 治理后 SMTP PSS 19.3MiB，
约再降 20%）；`go test ./app/Support/Extensions/...`（真实子进程集成测试）全绿，
插件调用无超时回归。

## 后续

空闲插件进程挂起/懒启动为独立工作流，需先写 ADR（涉及 runtime lease、
required hook provider 豁免、首调用延迟披露），本决定不预置其实现。

## 修订（2026-10-19）

默认值从 48 MiB 上调到 96 MiB。原因不是插件常驻内存变化，而是
`Support/AI` 的输入天花板放宽到 12 MiB：单个大上下文请求在插件进程里要过
JSON + proto（+ 可能的 base64）三份副本，48 MiB 会贴着软上限触发连续 GC。
`GOGC=50`、覆盖键 `SFORUM_PLUGIN_GOMEMLIMIT`、env 白名单的语义全部不变。
见 `2026-10-19-ai-context-and-budget-ceilings.md`。
