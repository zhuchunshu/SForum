# 2026-10-17 Session Handoff — dev 循环「保存即生效」激活链

## Changed

- 追加（同日第九轮，移除资产重发布竞态源）：实测推翻 round 6 结论——lifecycle 的
  AssetRegistry 与 frontendService.publicAssets 是同一实例（bindAssetRegistryConsumers），
  升级本身经 fenced ReplaceAllIfRevision 发布公开资产；round 6 加的
  RepublishPublicAssetsForDev（无 fence ReplaceAll）是第二写者，移动 revision 导致
  下一次升级 "asset registry revision changed during replacement" + 补偿失败。
  round 6 的「资产缺失」实为当时升级因 contractVersion 失败、republish 掩盖了失败。
  已整体移除（接口/字段/方法/bootstrap 参数/测试）。实机：CSS-only touch → 13 秒
  staged→upgraded，浏览器经 runtime 端点拿到含 clean-e2e 标记的最新内容（active
  12528）。另：带外 disable + digest --write + 重新启用，修复了调试期反复失败升级
  留下的身份发布归属错位（startup restore fail-closed 打死 API）；身份发布 rev 127
  已重新归属 active 摘要。

- 追加（同日第八轮，自动晋升路线撤除）：操作者确认「自动提版」语义不对（契约版本
  是作者承诺），全部移除：构建管线不再 bump contractVersion、promoteOne/Upgrade 的
  视图改写全部清除、UpgradeInput.Extension 已删。保留本轮排障沉淀的永久资产：
  ErrInvalidManifest 规则编号标注（admin/v3/ui/contribution 四处）、preflight 失败
  台账带完整上下文（static preflight + candidate 身份）、lifecycle 升级失败自动
  recovery retry。实机验证：touch CSS → 16 秒自动 staged→upgraded（08:13:49）。
  版本确认规则（来自操作者 NOTES）：**改内容不提版即自动生效**；改接口声明（props/
  schema/引用）才需手动 bump——这是正确的开发者介入点，不应自动化。

- 追加（同日第六轮，前端制品补齐）：回答「前端为什么还要走生命周期切换制品」
  发现三处缺口并全部修复：① dev 构建管线不含前端构建——`BuildFrontend`
  （frontend/admin 有 package.json 时 bun run build）接入 BuildAndDigestDevPackage；
  ② L2 组件 contractVersion 手动 bump 摩擦——UpgradeInput 新增非序列化
  `Extension` 内存载体，dev 通道在副本上把 components[].contractVersion 改写为
  dev-<digest8>（制品/真实版本不变；Upgrade 仅在 ID/version/stagedDigest 精确
  匹配时采信）；③ 公开资产注册表只在 Host 启动重建——DevAutoRuntime 升级成功后
  调 FrontendService.RepublishPublicAssetsForDev 完整图重发布。另修升级健壮性：
  dev upgrade 带确定性 Idempotency-Key（dev-auto/<id>/<digest>）、
  requires-explicit-retry 时自动 decision=retry recovery 一轮后重试、promote
  失败写 extension_events 台账、非 V2 包如实报错。实机四轮迭代验证：touch CSS →
  ~6s 自动 staged + upgraded，asset-inspector 显示新 packageDigest 46756d89…，
  运行时资产端点返回含 dev-live-round4 的最新内容。测试 +3。

- 追加（同日第五轮，终态 B 方案）：操作者两度明确「去掉登录/授权全过程，0 命令」。
  落地：① watcher 入 host——`bootstrap/dev_plugin_watch.go`（fsnotify、600ms 去抖、
  排除产物目录）+ `Models/Extensions/dev_auto_runtime.go`（sync→grant→enable/upgrade，
  合成无凭据 `dev-auto` 系统账号作 actor；V2 authority 要求真实 UserID 故不用
  ID=0）+ `Support/ExtensionPackage/dev_build.go`（host 内 go build + digest 写回
  含 SRI 重算）；仅 development 非 Safe Mode 装配，生产不注入。② wrapper 删除，
  CLI 登录提示/凭据缓存/PAT 层全部移除（dev_trust.go 回落 env-only 遗留态）。
  ③ 新测试 4 个：首次安装自动启用 / 已启用热更 upgrade（grant 对准 staged 摘要）/
  disabled 不被翻回 / dev-auto 账号幂等复用。④ 实机证据：dev 库自动出现
  dev-auto（id 157）、oauth2-provider 与 marketplace 自动 upgraded、touch 源文件
  6 秒内重建（digest 7cc3d26b…）；marketplace/harvester/fetch-url backend 因宿主
  新增 x/term 已统一 go mod tidy。决策：`decisions/2026-10-17-dev-zero-command.md`。

- 追加（同日第四轮，零凭据化）：操作者拍板「不要 super_admin 门」。评估后**保留
  门、消灭凭据摩擦**：`devTrustClient` 三级解析（env 会话/PAT → agent 缓存
  `$XDG_CACHE_HOME/sforum/dev/agent-credentials.json` 0600，PAT/会话对两种
  可撤销形态 → env 凭据自动登录入缓存 → 终端一次性提示（密码不回显，
  x/crypto/ssh/terminal）自动入缓存）；`dev-plugin-loop.sh` 删除凭据检查块。
  拒绝为 install/enable/upgrade 开 dev 无认证通道（handler 自身强制
  super_admin，改动即「第二条激活路径」）。浏览器导入引导（ego_login_import）
  在 Chrome 运行时 spawn 超时未成串通（如实记录），首次引导暂由操作者在
  终端一次性登录完成缓存。新增测试：缓存往返/损坏 fail closed/解析优先级
  （TestDevAgentCacheRoundTrip、TestNewDevTrustClientCredentialResolution）；
  x/term v0.47.0 依赖随 x/crypto 引入。

- 追加（同日第三轮，agent 契约固化）：`AGENTS.md` 插件开发约束改写为「必须走
  dev live-reload 链」——明确 agent 三条禁令（不许为暂存新摘要重启 dev API、
  不许让操作者逐制品人工确认/信任、不许手写 SRI/摘要脚本），外部根为独立插件
  仓库首选形态（不许拷贝/symlink 进主树），凭据只声明一次于 `.env` 且会话内
  不得交互式索取；锚定两份 decision。双语手册
  （`docs/extensions/build-and-load.md` + `docs/zh-CN`）把旧「重启 API →
  后台重新启用 → 再信任」迭代教法替换为链式流程并在排障表逐行更新，
  `docs/en-US` stub 指向 canonical 页无需另改（docs 门禁随即全绿）。
- 追加（同日第二轮，会话摩擦评估）：操作者问「dev 免确认会怎样」，三方案评估后
  采纳 A——`extension dev-trust create --ensure`（幂等，409 视为成功；
  httptest 四用例测试 `TestExtensionDevTrustCreateEnsure` 钉住 confirm 仍必带）+
  `scripts/dev-plugin-loop.sh <package-root>` 一键 wrapper（源 .env → API health
  检查 → ensure 会话 → exec `extension dev --auto-upgrade`）。方案 B（refresh
  自动建会话，会绕过 Create 的源根 attestation）与 C（v1 兼容开关，实测仍要
  super_admin 且丢逐摘要审计）在决策记录里拒绝并留了理由。
- `apps/api/cmd/sforum/dev_extension.go`：`devExternalRoots()` 让 `extension dev` /
  `dev-trust create` 接受 `EXTERNAL_EXTENSION_ROOTS` 下的包根（之前 CLI 比服务端更严，
  外部插件仓库直接被拒）；新增 `--auto-upgrade`（隐含 `--auto-trust`）；watch loop 的
  `autoTrust` 字段泛化为 `devTrustActivator` 接口（Refresh + Upgrade）。
- `apps/api/cmd/sforum/dev_trust.go`：`devTrustRefresher.Upgrade` 走既有
  `POST /admin/extensions/{id}/upgrade`。
- `apps/api/cmd/sforum/manifest_digest.go`：`digest --write` 重算已声明
  `assets[].integrity`（`refreshAssetIntegrity`；未声明的保持原样，缺文件报真实路径）。
- `apps/api/app/Models/Extensions/dev_trust_session.go`：`WithSourceSyncer` +
  `GrantDevArtifact` grant 前先 `SyncExternalSources`（字段 `sourceSyncer`，仅开发装配
  注入；失败返回真实原因）。
- `apps/api/bootstrap/api_assembly_domains.go`：dev 装配注入
  `.WithSourceSyncer(extensionService)`。
- `contracts/openapi.yaml` + `components/parameters.yaml`：路径参数
  `{extensionID}` → `{id}`（对齐 Go 路由目录 `:id`，修复 HEAD 上已存在的
  docs 端点校验红）；`paths/extensions.yaml` dev-trust-refresh 描述补充暂存行为。
- 文档：`docs/zh-CN|en-US/development/cli.md`（外部根、`--auto-upgrade`、重扫描、
  SRI；端点一律写花括号形式 `{id}`）。

## Decisions

- `knowledge/decisions/2026-10-17-dev-live-activation-chain.md`。

## Evidence

- `go test ./cmd/sforum/ -count=1`、`go test ./app/Models/Extensions/ -count=1` 全绿；
  新测试：外部根接受/拒绝、digest SRI 重算三件套、Grant 前重扫/失真提取、
  auto-upgrade 链顺序（refresh→upgrade→activated）、失真与仅 auto-trust 分支。
- `validate-openapi-refs`（2856 refs）、`validate-docs.mjs`（all checks passed，
  含修复 HEAD 既有 2 条红）、`go build ./...` 通过。
- 实机（air 重启后的 dev API）：外部仓库
  `sforum-oauth2-provider/plugins/sforum-oauth2-provider` 经
  `extension dev <root> --once --skip-install` 完整走通 build→digest→validate→test
  （0 errors / 7 checks，digest `e2f23ab7…`）。
- 架构门禁本轮零新增违规；工作树上另有 4 个非本会话文件的既有 diff 违规
  （`service.go`/`service_package.go`/`ExtensionManifest/manifest.go`/
  `adminExtensions.ts`，属并行未提交改动，见下）。

## Next

- 操作者：把 dev 专用 super_admin 凭据写入仓库根 `.env`
  （`SFORUM_ADMIN_EMAIL`/`SFORUM_ADMIN_PASSWORD`，gitignored 仅本机），之后
  `scripts/dev-plugin-loop.sh /home/inkedus/Code/github/sforum-oauth2-provider/plugins/sforum-oauth2-provider`
  一条命令完成全部循环。首次激活仍按管理端流程启用一次（upgrade 前置是 enabled）。
- `tests/admin-extension-activation.sh` / dev-loop 场景脚本可补一条
  `EXTERNAL_EXTENSION_ROOTS` 外部根用例（骨架已可跑）。
- prepare-package.sh 的 python SRI 步骤现在冗余（digest --write 已重算），可删可留。

## Open Questions

- 工作树里并存另一组未提交改动（web settings/login-methods、types.go、
  json_schema.go 等），architecture 门禁在其上有 4 条既有违规——与本会话无关，
  提交时需分属各自工作流。
