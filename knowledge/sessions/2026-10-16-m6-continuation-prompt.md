# 新对话提示词 — M6 收口（S9 委派 → 服务注册 → SDK → 隐私擦除 → M9 参考包 → 关闭 M10 行 5）

把下面 ```text 块里的内容整段贴给新对话。

```text
# 任务：SForum M6 收口——插件命名空间会话状态从「契约已定 + 存储与擦除钩子已落地」做到闭环

你在 /home/inkedus/Code/github/SForum 工作（Go/Fiber v3 + Nuxt4/Vue3 monorepo）。
目标：把 M6 做完，让 M10 行 5 与 M9 最后一项能诚实关闭。每个切片只有「绿的仓库 + 该切片的证据 + 知识库更新」三件齐全才算完成。

## 第 0 步：先读，不要凭本提示词的转述当结论

1. AGENTS.md（工作协议，全部条款有约束力）
2. knowledge/index.md（Latest Handoff + Current Project State；开头两条是本工作流的交接）
3. knowledge/sessions/2026-10-16-extension-authoring-closeout-handoff.md（**Phase 4/5 是上一轮的完整交接**，含新发现的 S9 与两个子代理的发现）
4. knowledge/decisions/2026-10-16-extension-authoring-contracts-freeze.md（**Revision 2 的 S1–S9 是权威契约**，尤其 §2.2 与 S9）
5. knowledge/sessions/2026-10-16-m6-session-state-implementation-guide.md（实施指南 + 必须的测试清单）
6. knowledge/sessions/2026-10-16-m10-joined-release-gate-matrix.md（Revision 2 逐行验收矩阵）
7. knowledge/plans/2026-08-22-extension-authoring-speed-and-controlled-power.md（主任务书 M6/M9/M10 行）
8. knowledge/plans/2026-10-16-plugin-frontend-experience.md（相邻任务书，M9 参考包与它共享）
与代码冲突时以代码为准，并在报告里指出冲突。

## 环境事实（实测，直接用）

- 网络命令前：export https_proxy=http://127.0.0.1:7897 http_proxy=http://127.0.0.1:7897 all_proxy=socks5://127.0.0.1:7897
- 开发服务器：web :3000（**用户自启，不要 kill**）；API :8080（air 管理）。
  重启 API 且不留 diff：cp 备份 → 往 apps/api/app/Support/Pages/catalog.go 追加一行合法 Go 注释 → 轮询
  apps/api/tmp/sforum-api 的 mtime 变化 → 等启动完成 → 删掉该行 → diff -q 确认逐字节一致。
  air 只监听非 _test.go 的 .go 内容写入（touch 无效）。
- 真实 Redis（M6 的 opt-in 测试要它）：容器 sforum-redis 映射到 127.0.0.1:16379。
  环境变量：SFORUM_PLUGIN_SESSION_REDIS_ADDR=127.0.0.1:16379、SFORUM_PLUGIN_SESSION_REDIS_PASSWORD=…、
  SFORUM_PLUGIN_SESSION_REDIS_DB（默认 15）。测试自建唯一 installation 前缀并精确清理；**永远不要用应用 DB**。
  真写满 4096 键的 opt-in 测试：SFORUM_PLUGIN_SESSION_FULL_CAP=1（约 75s）。
- 只读核查数据库：docker exec -e PGPASSWORD=sforum sforum-postgres psql -U sforum -d sforum -tAc "…"
- proto 工具链是自举的：bash scripts/proto.sh {lint|generate|check}（自己 build buf 与插件到 tools/proto/.bin）。
  **已知陷阱**：check 用 git status --porcelain 判定，因此**新增未跟踪**的生成文件永远被报「过期」；
  真实漂移看 `git diff --stat -- apps/api/sdk/plugin/v2/gen` 为空 + 连续两次 generate 的哈希一致。
  若你在本轮新增了 proto 文件，请在报告里如实说明 check 的这一行为，不要假装它绿。
- 窄屏证据必须用真实 Chrome over CDP：node tests/browser-public-390x844.mjs（390x844 dpr3）。
  该脚本已修好水合等待，支持 SFORUM_L2_PATHS（预期公开 L2 真的挂载）与 SFORUM_PUBLIC_PATHS/BARE_PATHS/QA_DIR/EVIDENCE_OUTPUT/SETTLE_MS。
  `ego_*` 侧栏载体**不是**窄屏载体（Emulation.setDeviceMetricsOverride 不生效，innerWidth 恒 500），不要用它冒充 390 宽。
- 管理端会话：优先用已登录的 ego 载体经 CDP Network.getCookies 取 cookie；或走 tests/admin-extension-activation.sh。
  dev 会话 cookie（httpOnly、会过期）：sforum_session=XM3hI52kWRj5TsObFdxoaf0q2jJI4DZ6osd0lrCx2JI

## 已完成基线（不要重做，可依赖）

- **契约已冻结到 Revision 2 的 S1–S9**。S9 是上一轮实现时才发现的缺口，已在 ADR 里定下机制：
  插件→宿主的调用通道按**运行时**绑定（protocolV2HostBinding 长驻），session 无法从它派生；
  会话范围必须像既有 HostCommandDelegation/HostQueryDelegation 那样，用**宿主签名的一次性委派**传递。
- **app/Support/PluginSessionState 包已落地**（types.go / store.go / backend_redis.go / fake_backend_test.go / store_test.go，
  23 个顶层测试 / 101 case），`go test -race` 绿。已实现：S1 键派生、S2 有界不透明值、S3 四路 TTL 上界、S6 上限、
  三张索引（每会话键 / 每会话扩展 / 每扩展会话）、幂等 InvalidateSession/InvalidateExtension。
  计数走「扫描成员 + 逐个确认值键存活 + 顺手摘陈旧成员」，**不是** SCARD（SCARD 会把陈旧成员算进去）。
- 上一轮修掉的 4 件事**不要改回去**：(D1) InvalidateSession 已删掉那次对空成员列表的冗余 SetRemove（幂等由 store 自己保证）；
  (D2) Get/Delete 现在对超长键返回 ErrKeyTooLong（与 Set 一致）；(D3) Backend 接口里的 SetCard 已删除（从无调用方）；
  (D4) EffectiveTTL 里不可达的 TTL 夹取分支已删。
- **擦除钩子已落地并有测试**：登出 + 登录替换（Support/AuthSession/plugin_session_state.go + manager.go 两处调用，5 条测试）；
  撤销 / 设备数驱逐 / 改密（Models/Identity，四个 SQL 点已改成 RETURNING sid，提交后才擦除）；
  扩展禁用 / 卸载（Models/Extensions，提交后 best-effort）。
- **proto 已落地但刻意不可用**：contracts/proto/sforum/host/v2/session.proto（service PluginSessionService 只有 Get/Set/Delete，
  请求只有 context + namespace_key（Set 另加 value/ttl_ms），**没有任何 extension id/user id/session id 字段**——这就是 S7 的安全属性）
  + 两个生成文件。**它刻意没有注册进 app/Support/HostAPI/v2.go，也刻意没有写进 tests/validate-host-api-v2-docs.mjs 与
  docs/extensions/host-api-v2.md**。不要「顺手」补上：声明了却不可用比没有更糟（M4 取消的先例）。
  sdk/plugin/v2 的 TestGeneratedServiceCatalog 已加入该服务。
- M10 行 1/2/3/4/6/8/9/10 已取证；**行 5（会话状态）与行 7（真多节点）开放**。
- tests/browser-public-390x844.mjs 已修：等真实水合（#__nuxt.__vue_app__）→ 对 SFORUM_L2_PATHS 要求 L2 终态
  （mounted|fallback|quarantined，绝不在 idle/loading 判稳定）→ 可显式断言挂载；负控制实测会 FAIL。
- 目录身份已登记（SFSettingsAccountSidebar），18 个 v3 目录已重新生成且 --check 通过。

## 待办（按价值排序；每条含验收与必须的证据）

### T1（最优先）S9：宿主签名的一次性会话委派
- 落地机制：新增一次委派消息 + RequestContext 字段（镜像 HostCommandDelegation/HostQueryDelegation 的形状与注释），
  宿主**只在自己发起、且自己知道 session 的调用点**签发（HTTP 请求内的插件调用）；插件在 Get/Set/Delete 上回带；
  宿主校验签名 + 绑定（actor/runtime/artifact）+ 一次性/过期，然后从**自己的记录**得到 session 指纹。
- 不得改动 S8：Actor.session_id 继续不填充；插件永远看不到也提供不了 session 标识。
- 验收：wire 层两侧——签发校验通过；篡改 token / 过期 / 复用 / 绑定不符（跨 runtime、跨 artifact）全部 fail closed；
  并用 descriptor 层（不是目测）确认新字段没有引入任何 caller 可填的 identity 字段。
- 证据：proto lint/generate 输出、测试名与原文输出、以及 check 的已知行为说明。

### T2 服务适配 + 注册
- 新增 app/Support/HostAPI/plugin_session_state_service.go 与 plugin_session_state_protocol_v2.go（镜像 CacheService 的分层）：
  扩展身份/运行时实例由 ProtocolV2RuntimeIdentityFromContext 派生，session 来自 T1 的委派；
  在 app/Support/HostAPI/v2.go 的 registerProtocolV2 里按 cache 的同款可选参数方式注册。
- Safe Mode 与准入必须 fail closed：读返回未命中、写被拒绝；未准入/未认证一律拒绝。
- 验收：经**真实 gRPC** 的 allowed round trip；Safe Mode 双侧；未准入侧；错误不泄漏底层文本。

### T3 SDK helper + 文档（顺序不能反）
- 插件侧 helper（get/set/delete 三操作，不暴露 Fiber/Nuxt/Redis 类型），随 @sforum/plugin-ui 或 SDK 的既有形态交付。
- **只有当服务真的被 serve 之后**，才把 session.proto 加进 tests/validate-host-api-v2-docs.mjs 的 hostProtoPaths
  并补 docs/extensions/host-api-v2.md 的双语内容。先写文档后实现 = 制造「声明了却不可用」。

### T4 隐私擦除
- Support/Privacy 至今**没有任何消费者**。注册一个 contribution，其 erase hook 遍历被擦除用户的会话并对每个调
  InvalidateSession（记得：值是永不进审计/日志的）。
- 验收：擦除后该用户的插件状态确实消失；且断言审计/日志里没有值。

### T5 M9 参考包（会话状态工作流夹具）
- extensions/fixtures/plugins/ 下一个聚焦夹具（真实协议 V2 后端 + 一个需要会话状态的小工作流），
  走作者命令（digest --write / validate / test）并在干净临时目录可跑；双语文档同步。
- 这一步完成后 M9 的最后一项才算关闭。

### T6 joined 验收测试（把 S5 矩阵逐行落成测试）
allowed 往返、跨插件拒绝、跨会话拒绝、**调用方身份不可表达**（请求形状里没有该字段）、轮换（同 sid 不丢）、
替换（新登录不继承）、登出、撤销/设备数驱逐、过期（含请求 TTL 超过会话剩余寿命时必须收紧而非放宽）、
上限（值>1MiB、键>512B、每会话>64、每扩展>4096，且上限处更新已有键仍允许）、隐私擦除、并发、Safe Mode。
不许用同义测试凑数；也不许弱化断言。

### T7 关闭 M10 行 5
- 逐行更新 knowledge/sessions/2026-10-16-m10-joined-release-gate-matrix.md（行 5 结论 + 证据引用），
  同步主任务书 M10 状态行、knowledge/plans/README.md、knowledge/index.md，node tests/validate-docs.mjs 通过。

### T8 需人工决定 / 外部依赖（记录后继续下一项，不要停下来等）
- M10 行 7：真多节点需要运维提供第二节点；Go 级 barrier/reconcile 证据是真实的，但不是多节点证据。
- M10 行 9 残留：从 registry 拉真实已发布 tag；compose.prod.yaml 的反代/TLS 层。
- 是否允许新增 Protocol V2 字段（T1）本身是一次契约变更，已记为 S9；若你想改机制，请先改 ADR 再动代码。

## 每次改动后必跑的门禁

  cd apps/web && bun run typecheck
  cd apps/web && bun test
  cd apps/api && go build ./... && gofmt -l app cmd config sdk
  cd apps/api && go test ./app/Support/PluginSessionState/ -count=1 -race
  cd apps/api && go test ./app/Support/Pages/... ./app/Http/Controllers/Pages/... ./app/Support/ExtensionManifest/... -count=1
  cd apps/api && go test ./... -count=1
  node tests/validate-architecture-boundaries.mjs
  node scripts/v3-catalog/generate.mjs --check
  node tests/validate-docs.mjs
  node tests/validate-public-page-surface-catalog.mjs
  node tests/validate-v3-production-trust.mjs
  ruby scripts/validate-openapi-refs.rb
  bash scripts/ci/reference-packages-gate.sh
  bash scripts/proto.sh lint        # 改了 proto 之后还要 generate，并按上面的说明解读 check

## 已知噪音（不要追，也不算你的回归）

- `gofmt -l app cmd config sdk` 恒有 4 个既有文件：app/Http/Controllers/Forum/shortcode_handlers.go、
  app/Models/Identity/external_auth_t1c_activation_test.go、app/Support/AI/purpose_resolver.go、app/Support/AI/tool_loop.go（勿动）。
- **并发工作流遗留**（上一轮的另一个会话，21:33 后未再活动；其文件在禁改清单内）：
  · `TestIslandAttributesAllowDeclaredPropsOnly` 失败——`theme_body_island.go` 声明 sf-account-settings-sidebar 的
    active 为 Required，而 `template.go` 同一条目是空属性表，验证器因此接受了缺必填 prop 的岛。这是**真实的表间不一致**，
    不要改测试掩盖，也不要动那两个文件；在报告里留着即可。
  · `gofmt -l` 因此多报 app/Support/Pages/{theme_body_island.go,template_test.go}（纯格式差异，勿动）。
- 负载敏感 flake（单跑即绿）：TestRevisionStreamReconcilesMissedWakeAndHeartbeats、
  TestAIPluginCommandReferenceEntersHostGatesAndAccounting、TestComponentCompositionTimeoutCrashUnauthorizedAndFallback。
- `./scripts/test.sh` 会在 scripts/ci/release-tooling-gate.sh 处中止（既有）。
- 上一轮实测 `go test ./...` 是 exit 0（128 包、零 FAIL），所以**不要**预先断言某些测试必然红；以实际运行为准。

## 硬规则（违反即不算完成）

1. 永远不要抬高 tests/architecture-boundaries-baseline.json；超限就抽取职责并**下调**基线。
   注意 apps/api/app/Models/Extensions/service.go **恰好 1000 行**，往它加任何一行前先抽取或把新代码放到有空间的文件里
   （上一轮把新增的 ServiceOption 移到了 legacy_plugin_runtime_publication.go）。
2. 新增/变更 UI 面必须在 docs/extensions/v3/catalog-identities.json 登记（否则目录生成器抛错）；
   命名与 owners 从既有兄弟条目转写，不要发明语义。
3. 公开页每个渲染路径**恰好一个**可见 navbar 与 footer；bare 为零。
4. 主题模板遵守三方身份契约（theme.json ↔ templates[] ↔ packageFiles[]，摘要一致）。
5. 改内置主题源码 ≠ 运行时生效：必须重建 → 重启暂存 → 走管理端流程激活 → 用 /site/active-theme/skin 与 /pages/resolve-path 验证。
6. **源码级单测不算运行时证据**；需要运行时证据的项必须给 HTTP/浏览器落地证据（命令输出摘要、DOM 片段、截图路径）。
7. 诚实性：假设标为假设；没验证的不要写已验证；做不到如实记录；推翻自己先前结论要写**更正记录**。
   不要用同义测试、伪造归因（如给插件页硬套 data-template="1"）或「500 宽当 390」勾清单。
8. 不要「顺手」注册或文档化尚未真正服务的能力（**声明了却不可用比没有更糟**）。
9. 不修改并发工作流的文件：app/Support/Pages/{extension_bridge,template,theme_body_island,registry_contribution_validation}.go、
   app/Http/Controllers/{AI,Extensions}/*（只读核对除外）。上一轮我只补登记了目录身份（转写），语义与测试一律没动。
10. 不要在漫长一轮里留下半应用状态；每个切片收尾时仓库必须绿。dev API 被拖挂时：先用
    go run ./cmd/sforum extension disable <id> 出带外禁用，再按上面的探针重启，确认 diff -q 逐字节一致。

## 工作风格与工具约定

- 用 rg/glob/grep 检索、read 读文件、edit/write 改文件（不要用 cat/Python 写文件）。
- 能并行就并行；**测试波及面大或互不重叠的切片可开多个 subagent**，每个分片要求：只改指定文件、不弱化断言、
  两侧覆盖、自跑 gofmt + 该包 go test 并回报**原文输出**；分片报告里要它明确列出「没能测到什么」。
  （上一轮的经验：给 subagent 冻结接口能大幅提高产出质量；它们会在冻结代码里挖出真缺陷——那正是要它们做的事，
   但**修复归你**，因为它们不许改冻结文件。）
- 每轮结束必须：跑完全部门禁、确认仓库绿、更新知识库（模块笔记 / 热交接 / index / 计划状态表与正文 / plans/README）。

## 交付要求

报告按「结果 → 关键证据 → 验证方式 → 阻塞与未完成 → 建议下一步」组织，简明可核对；
明确区分**已验证完成 / 部分完成并说明缺什么 / 阻塞并说明被什么阻塞**，不允许「应该可以」这类无证据表述。
```

## 使用说明（给人看的）

- 上面那段整块贴进新对话即可。它是自包含的：机制、文件、命令、噪音、硬规则都在里面。
- 若新对话的预算只够一件事，就只让它做 **T1（S9 委派）**——它是唯一真正的阻塞点，其余全部依赖它。
- 建议顺序：T1 → T2 → T3 → T4/T5 → T6 → T7；T8 只需记录。
- 上一轮的两个 subagent 分工（存储测试 / proto）已被验证有效；T1 的 wire 层可以照同样方式再开一个分片。
