# 2026-10-12 Recovery Note: AI Plugin Host API (M1/M1.5/M2) tracked edits lost to a worktree revert

## What happened

- 00:28:04 有人/某工具把工作树里**所有 tracked 文件**恢复到 HEAD（`git status` 从
  85 项降到 54 项且只剩 `??`）。本会话在 M1/M1.5/M2 与后续两项 AI 工作里的 tracked
  侧改动全部消失（untracked 新文件仍在）。
- 00:43–00:50 期间有**另一个写入者**在同一批文件上继续编辑（`v3_normalize.go`
  00:48:57、`v3_validate_platform.go` 00:49:36、`schemas/manifest-v3.schema.json`
  00:50:43），并留下 `apps/api/tmp/build-errors.log`、`apps/api/tmp/sforum-api`。
  该写入者当前的 schema 状态是坏的：`$ref: #/$defs/contractVersion` 指向不存在的
  定义，`TestManifestV3AuthoritativeFixtures` 因此编译 schema 失败。
- 因此本会话没有继续覆盖写入，改为记录并等待归属确认。

## Surviving artifacts (untracked, intact)

- AI: `app/Support/AI/redaction.go`、`redaction_test.go`、`result_cache.go`、
  `result_cache_test.go`、`tool_contribution.go`、`tool_contribution_test.go`、
  `tool_plugin.go`（后两者为 M2 会话产物）。
- Host API: `app/Support/HostAPI/v2_command_ai.go`、`v2_command_ai_test.go`、
  `v2_command_events.go`、`v2_command_events_test.go`。
- Extensions: `protocol_v2_ai_tool.go`（含 `AIToolRuntimeInvoker` 与
  `ProtocolStarter.InvokeVersionedAITool`）、`events_publish.go`、
  `protocol_v2_ai_tool_test.go`、`IntegrationTests/ai_tool_reference_plugin_integration_test.go`、
  `IntegrationTests/ai_command_reference_plugin_integration_test.go`、
  `IntegrationTests/reference_plugin_database_test.go`。
- bootstrap: `ai_plugin_tools.go`、`ai_plugin_tools_test.go`。
- SDK: `sdk/plugin/v2/ai.go`、`ai_test.go`、`ai_tools.go`、`ai_tools_test.go`。
- fixtures: `extensions/fixtures/plugins/sforum-ai-tool-reference/`、
  `sforum-ai-command-reference/`、`sforum-contract-events/schemas/`、
  `ExtensionManifest/testdata/v3/ai-tool-plugin.json`、`v3_ai_tools_test.go`。
- knowledge: 决策记录 `2026-10-11-ai-plugin-complete-command.md`、
  `2026-10-11-events-publish-command-and-flat-cap.md`、
  `2026-10-11-ai-tool-contribution-m2.md`、`2026-10-11-ai-outbound-redaction.md`、
  `2026-10-11-ai-result-cache-wiring.md`，handoff
  `2026-10-11-ai-plugin-host-api-m1-handoff.md`、`-m2-handoff.md`，计划
  `plans/2026-10-11-ai-plugin-host-api-m1.md`。

## Tracked edits to re-apply (gap list)

1. `app/Support/HostAPI/v2_command_runtime_postgres.go`：`Events ProtocolV2EventPublisher`
   与 `AI ProtocolV2AIGateway` 配置字段 + 注册 `newProtocolV2EventsEmitCommandDefinition`
   与 `newProtocolV2AICompleteCommandDefinition`。
2. `bootstrap/host_command_runtime.go`：`deferredAIGateway`（+ deferred 事件发布者），
   binder 返回类型同时携带两个后填目标。
3. `bootstrap/api_assembly_domains.go`：`commandDeferred.AI.Set(aiProvider.Gateway())`、
   `registerPluginAITools(aiToolRegistry, providers.NewAIToolInvoker(lifecycleRuntime), items)`、
   `providers.NewAIProvider(..., cache.NewRedisCache(sharedRedisClient))`。
4. `app/Providers/ai.go`：`cacheStore cache.Cache` 参数 + `Cache: supportai.NewResultCache(cacheStore)`。
5. `app/Providers/extensions.go`：`NewAIToolInvoker`（`supportai.PluginToolInvoker` 适配器）。
6. `app/Support/AI/gateway.go`：`request = ApplyRedaction(settings.Redaction, request)`（缓存键之前）
   + 命中路径 `cached.CacheHit = true`。
7. `app/Support/Extensions/protocol_v2_client.go`：`aiTools` 配置/客户端字段（当前已由另一写入者补回，
   需核对）。
8. `sdk/plugin/v2/runtime.go`：`aiToolRegistry` 字段、`WithAIToolRegistry`、
   `ProviderCall` 的保留 slot 分支（无注册表时 Unimplemented）。
9. `app/Support/Events/catalog.go`：调用方命名空间事件的解析放宽（M1.5）。
10. manifest 侧：`aiTools` JSON schema（含正确 `$defs`）、`v3_fixtures_test.go` 的
    want 列表补 `ai-tool-plugin.json`；若采用另一写入者的 `v3_ai_tools.go` 布局则保留其文件、
    只补缺口。
11. `tests/architecture-boundaries-baseline.json`：扁平目录 146→147、
    `protocol_v2_client.go` 1107→1111、`manifest.go` 调到实际值。
12. docs：`docs/extensions/host-api-v2.md`（AI completion / events / ai.tool / host_commands /
    脱敏 / 缓存）、`authoring-guide.md`（场景表 + 清单条目）、`extensions/fixtures/README.md`
    （两个新 fixture）、`docs/extensions/catalogs/{capabilities,manifest-v3}.md` 重刷。
13. knowledge：`index.md`（Latest Handoff + AI 段 + 决策列表）、`modules/ai.md`、
    `modules/extensions.md`、`plans/README.md` 状态行。

## Verification to re-run after recovery

```bash
cd apps/api && go build ./...
cd apps/api && go test ./app/Support/ExtensionManifest/... ./app/Support/AI/... \
  ./app/Support/HostAPI/... ./app/Support/Extensions/ ./bootstrap/ ./app/Providers/ ./sdk/... \
  ./app/Models/AI/... ./app/Models/AIReply/...
cd /home/inkedus/Code/github/SForum && node tests/validate-architecture-boundaries.mjs
./scripts/test-api-isolated.sh -run 'TestAIPluginCommandReferenceEntersHostGatesAndAccounting|TestAIToolReferencePluginDispatchesThroughRealRuntime|TestNotificationReferencePluginEmitsThroughRealBroker' -count=1 ./app/Support/Extensions/IntegrationTests
```

## Recovery status (2026-10-12 01:5x)

已按上面的清单完成重建并重新验证：

- tracked 侧：HostAPI 命令运行时（`Events`/`AI` 配置 + 两个命令注册）、
  `bootstrap/host_command_runtime.go` 的 deferred 目标、两个装配阶段、
  `Providers/ai.go`（缓存参数）、`Providers/extensions.go`（`NewAIToolInvoker`）、
  `Support/AI/gateway.go`（脱敏 + `cacheHit`）、`sdk/plugin/v2/runtime.go`（保留 slot 路由）、
  `Support/Jobs` 与 `Models/Extensions`、`HostAPI/plugin_jobs.go` 的队列声明、
  manifest 的 `jobs[].queue` 与 schema、架构基线（`manifest.go` 920→916、
  `Support/Extensions` 扁平目录 145→147）。
- 修正了并行写入者留下的两处缺陷：`v3_ai_tools.go` 的 `json:"timeoutMS"` 与其
  自身 schema 要求的 `timeoutMs` 不一致（导致子进程 fixture 清单校验失败）、
  `v2_command_runtime_postgres_test.go` 的命令目录计数。
- docs 与 knowledge 已重做（host-api-v2 四节 + authoring-guide 场景表/清单/Job queues）。
- 验证：`go build ./...`、聚焦 Go 套件、架构门禁、目录文档同步、Host API 文档校验、
  V3 目录校验全绿；隔离集群上四个用例通过（插件命令四路径、AI 工具子进程派发、
  notification reference、真实 Postgres 的缓存命中零花费）。

## Left for the other workstream

`app/Support/Extensions/IntegrationTests/gamification_database_grant_enable_integration_test.go`
（untracked，幸存）引用了被回滚掉的 tracked 符号
`extensions.WithDatabaseGrantAuthority`、`extensions.DatabaseGrantAuthority`、
`audit.ActionExtensionDatabaseGrant`，因此该包在它的 tracked 侧恢复前无法编译。
本次验证时临时把该文件移到 /tmp（校验 md5 后原样放回，内容未改）。
