# AI 上下文与预算天花板：从「防呆边界」回到「模型能力」

日期：2026-10-19
状态：已接受

## 背景

`Support/AI/limits.go` 里的常量最初按「小上下文、低成本小站」定标：
输出 32k、单次输入 400KB、单条消息 100KB、工具结果 8KB、工具步数 5、
月度预算 10 万元。上线后发现两个问题：

1. **它们被当成产品天花板用了**。1M token 上下文的模型（Claude 长上下文、
   Gemini 2.5 Pro、GPT-5 长输出）在站内根本用不到：400KB 输入预算约等于
   十几万汉字，连一半上下文都装不满；32k 输出让长文生成直接不可达。
2. **放宽这件事在前端不可见**。控制台把 `maxTokens`、闸门、预算的上下限
   硬编码在自己的组件里，后端常量改了前端不知道。两处数字漂移的直接后果
   是「界面允许、服务端 422」。

同时定位到一个独立的正确性缺陷：`PUT /admin/ai/settings` 失败时只返回一个
`ai.settings_invalid`，`data.fields` 永远是空的；而 `<input type="number">`
被清空后会把空串写进载荷，Go 的 `int` 解码失败同样折叠成同一个 reason。
运营者拿到的是一个没有任何指向的 422。

## 决定

### 1. 天花板按模型能力重定标（`Support/AI/limits.go`）

| 常量 | 旧值 | 新值 |
|---|---|---|
| `MaxCompletionTokens` | 32,000 | 128,000 |
| `DefaultMaxCompletionTokens` | 2,048 | 4,096 |
| `MaxMessages` | 64 | 256 |
| `MaxMessageBytes` / `MaxSystemBytes` | 100,000 | 1 MiB |
| `MaxTotalInputBytes` | 400,000 | 12 MiB |
| `MaxToolResultBytes` | 8,000 | 64,000 |
| `MaxToolResultRunBytes` | 16,000 | 256,000 |
| `MaxToolCallsPerReply` | 5 | 12 |
| `MaxRateLimitPerMinute` | 6,000 | 60,000 |
| `MaxExtensionDailyQuota` | 1,000,000 | 100,000,000 |
| `MaxUserDailyQuota` | 10,000 | 1,000,000 |
| `MaxMonthlyBudgetMicros` | 100,000 元 | 10,000,000 元 |

**默认值全部不变**：`RecommendedSettings()` 仍是 60 次/分、500 次/日、20 元/月，
`DefaultMaxCompletionTokens` 仍保守。放宽的是「运营者最多能设多大」，不是
「开箱允许多大」。这符合仓库的默认值约定：安全默认 + 可调上限。

### 2. 放宽不削弱安全边界

- 这些常量限制的是**单次调用规模**。防滥用靠预算闸门、速率闸门、配额闸门
  （`gate.go` 的 `EvaluateGates`），它们的默认值与判定逻辑零改动。
- 内存压力由两处兜底：插件 gRPC 单消息上限与插件进程 `GOMEMLIMIT`（见下）。
- `Validate` 的拒绝口径不变，只是从「返回第一个错误」变成「收集全部 Issue」。

### 3. 上限成为下发给前端的契约

`GET /admin/ai/settings` 新增 `limits`（`supportai.CurrentLimits()`），控制台据此
渲染 `min`/`max` 与提示文案。`apps/web/app/utils/admin/adminAI.ts` 保留一份同值
兜底，仅用于旧载荷；`TestSDKAICompletionLimitsMatchHost` 与前端测试钉住一致性。
**以后调天花板只需要改后端常量。**

### 4. 字段级 422

- `Settings.Issues()` / `Profile.Issues()` 逐项报告 `{Field, Code}`，`Validate()`
  退化为「有任意 issue 即失败」的包装，对外错误类型不变。
- 控制台把 `Issues()` 映射成 `422 ai.settings_invalid` + `data.fields`；解码失败
  映射成新 reason `ai.settings_malformed`（字段类型不对），只暴露字段路径，不回显
  原值。两者都经 `Localization` 输出各语言文案（`ai.field.*`）。

### 5. 连带调整三处

1. **插件 SDK 镜像常量**：`sdk/plugin/v2/ai.go` 的 `HostAICompleteMaxMessages` /
   `HostAICompleteMaxTokens` 同步为 256 / 128_000。SDK 刻意不 import `Support/AI`，
   这层镜像没有编译期约束，新增测试钉住。
2. **gRPC 单消息上限**：`sdk/plugin/v2/transport.go` 的
   `DefaultMaxMessageBytes` 4 MiB → 16 MiB。`DefaultProtocolV2MaxMessageBytes`
   派生自它，握手广播与 host/client 两端同时跟随。约束是
   `DefaultMaxMessageBytes >= MaxTotalInputBytes`：Host 允许的输入不能被传输层
   先拒掉。
3. **插件进程内存**：`pluginProcessGOMEMLIMITDefault` 48 MiB → 96 MiB。
   单个 12 MiB 请求在插件进程里要过 JSON + proto 三份副本，48 MiB 会贴着软上限
   触发连续 GC。

## 权衡

- 更大的单消息上限意味着单个插件进程的峰值内存更高（软上限，超限触发连续 GC
  而不是 OOM kill）。96 MiB 仍是「数倍于实测常驻」的保守值。
- 放宽天花板后，一次配置失误可能产生更大账单。这是运营者显式选择的结果，
  且预算闸门仍然生效；不为了防呆把模型能力按住不用。
- `Issues()` 收集全部问题而不是首个：错误信息变多，但 422 才能定位。

## 后续

- 大 payload 的插件路径需要一次实测（沿用
  `decisions/2026-10-17-plugin-process-memory-limits.md` 的复测方式），
  若 96 MiB 出现连续 GC 抖动，按覆盖键 `SFORUM_PLUGIN_GOMEMLIMIT` 调整。
- 图片上限（4 张 / 单张 2 MiB）不在本次范围：它与输入总预算耦合，
  需要单独决策「图片是否占用文本预算」。

## 证据

- `app/Support/AI/limits_test.go`：镜像常量、传输容量、上限自洽、1M token 前提。
- `app/Support/AI/settings_issues_test.go`：字段路径与原因码。
- `app/Http/Controllers/AI/controller_test.go`：`ai.settings_malformed` 与往返可保存。
- `apps/web/tests/admin/adminAISaveValidation.test.ts`：空串/小数收敛、URL 与字节
  长度校验、保存前拦截。
