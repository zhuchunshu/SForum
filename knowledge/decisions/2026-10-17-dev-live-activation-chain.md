# 决策：dev 循环的「保存即生效」激活链（外部仓库 + 自动暂存/升级 + SRI 重算）

日期：2026-10-17。关联：
`2026-10-16-extension-authoring-contracts-freeze.md`（ADR 6.2，M8 dev loop）、
`2026-10-17-dev-trust-session.md`（会话级 dev 自动信任）。

## 背景

外部插件仓库（如 `sforum-oauth2-provider`，与主程序不同目录）的迭代循环此前是：
手工 `prepare-package.sh` → **重启 API**（`SyncExternalSources` 只在 bootstrap 跑一次）
→ 管理端手工升级。四个缺口：

1. `extension dev` / `dev-trust create` 的 CLI 根白名单只认仓库内
   `extensions/dev|optional|fixtures`，比服务端 `DevTrustService.devSourceRootAllowed`
   （早已接受 `EXTERNAL_EXTENSION_ROOTS`）更严——文档还承诺了外部根可用；
2. 运行期没有重扫描入口：改完包必须重启 API 才能暂存新摘要；
3. `--auto-trust` 只写信任授予，staged → active 的 upgrade 仍要手工；
4. `extension digest --write` 不重算 `assets[].integrity`，资产一改包即 invalid，
   作者只能手写脚本重算 SRI。

## 决定

1. **CLI 对齐服务端**：`resolveDevPackageRoot` 追加 `EXTERNAL_EXTENSION_ROOTS`
   （逗号分隔、绝对路径、与 `config.Load` 同一约定）。受保护 `extensions/builtin`
   仍然拒绝。**不引入 symlink 逃逸面**：快照管线本就拒绝 symlink，判定是纯词法
   `filepath.Rel`，与服务端同一语义。
2. **暂存并入信任刷新**：`DevTrustService.WithSourceSyncer`（仅开发装配注入，
   bootstrap 传 `extensionService`）使 `GrantDevArtifact` 在读取制品身份**之前**
   跑一次 `SyncExternalSources`。同步失败返回真实原因，不折叠成
   `trust_challenge_required`（否则作者无从排查）。生产装配不注入，行为为零变化；
   未配置外部根时同步为空操作，行为与现状一致。
3. **自动升级复用既有路由**：`--auto-upgrade`（隐含 `--auto-trust`）让 dev 循环在
   信任续期成功后调用既有 `POST /api/v1/admin/extensions/:id/upgrade`。不新增任何
   绕过信任的路径：grant 由 dev 会话写入（逐摘要审计），upgrade 消费它；disabled
   等失败输出 `auto_upgrade_failed` 且循环不中断。
4. **SRI 重算内建**：`digest --write` 对**已声明** integrity 的资产重算
   `sha256-<base64(sha256(文件))>`。从不给未声明的资产凭空加字段，包基线不变。

## 不做的事

- 不做信任绕过：每个摘要仍有独立 grant + 审计行；Safe Mode 吊销全会话不变。
- 不让运行时执行包内脚本/增加解释执行后端：Go 编译无法消除，只能被常驻 watcher
  隐藏（保存后 2–10 秒增量构建是唯一延迟）。
- 不新增 HTTP 路由：复用 `dev-trust-refresh`（文档本来描述的就是
  "上传新包 → grant → enable"）与 `upgrade`，route catalog / OpenAPI / 审核矩阵
  零新增面。

## 附加决定（同日）：会话建立摩擦的三方案评估

问题：「dev 模式免确认会怎样？」——先把两件常被混为一谈的事拆开：
**凭据**（super_admin HTTP 会话，install/enable/upgrade 路由本身的要求）与
**dev-trust 显式确认**（`confirm: true`，每开发会话一次）。

- **C（`SFORUM_V3_TRUST_CHALLENGES=false`，已有 v1 兼容开关）拒绝**：实测
  `lifecycle.go:86` 与 `service_lifecycle.go` —— 挑战关闭后 install/enable 仍走
  `requireSuperAdminForUntrustedBackend`，凭据环节一步未省；审计从逐摘要 exact
  grant 退化为幂等兼容 grant。等于只丢语义、不减摩擦。
- **B（`dev-trust-refresh` 无会话时自动建会话）拒绝**：`Create` 承担的
  源根白名单 + manifest ID attestation 会整个被绕过（`GrantDevArtifact` 只看
  会话存在），违反 ADR「任何未满足条件必须 fail closed 到人工路径」。
- **A 采纳（保持显式确认 + 摩擦压缩）**：`extension dev-trust create --ensure`
  幂等（409 `extension.dev_trust_session_exists` 视为成功，confirm 仍逐请求携带、
  由服务端裁决）+ `scripts/dev-plugin-loop.sh <package-root>` 一键 wrapper
  （源 .env 凭据 → ensure 会话 → exec 带 `--auto-upgrade` 的 watch 循环）。
  会话语义、TTL、Safe Mode 吊销、逐摘要审计一行不改；日常只有一条命令，
  凭据放入 gitignored 的 `.env`（dev 机专用）后全程免交互。

## 附加决定之三（同日）：dev 零凭据化 —— 缓存 agent 凭据而非移除 super_admin 门

操作者最后的拍板方向是「dev 下不要 super_admin 门」。评估后保留门、取消凭据摩擦：

- **拒绝对 install/enable/upgrade 路由开 dev 无认证通道**：这些 handler 在自身
  逻辑里强制 super_admin（`requireExtensionsMutationAuthority`/`requireCoreGuardPermission(evaluation, "*")`），
  改角色位即「第二条激活路径」——生产 fail-closed、逆转 Safe/恢复语义的 ADR 底线；
  且本机进程可调无认证 mutation 会让「可执行插件代码进 Host」从 dev 凭据门
  降级成文件系统写权限门。
- **接受的部分（零凭据、零交互）**：`devTrustClient` 三级解析——
  `SFORUM_SESSION_COOKIE`/`SFORUM_DEV_AGENT_PAT` → agent 缓存
  （`$XDG_CACHE_HOME/sforum/dev/agent-credentials.json` 0600，两种可撤销形态：
  PAT `{kind:pat}`（CSRF 豁免、管理端可撤销）或会话对 `{kind:session}`）→
  env 凭据登录自动入缓存 → 终端一次性提示输入（密码不回显，
  `x/crypto/ssh/terminal`）自动入缓存。缓存永远不含密码；生产不读。
  `dev-plugin-loop.sh` 删除凭据检查块，解析交给 CLI 错误信息。
- **浏览器导入引导未完成（如实记录）**：`ego_login_import` 对系统 Chrome 的
  headless spawn 在「Chrome 正在运行 + closeSource」下仍超时失败（三次尝试，
  `closedSource:false`）。当前引导路径＝操作者终端一次性提示缓存；浏览器导入
  是未来增强（工具侧修复 spawn 后零输入也可达成）。
- 用户体验终态：`./scripts/dev-plugin-loop.sh <包根>` 首次提示一次登录，
  此后每次运行（含所有 agent 会话）零确认、零输入、零提示——super_admin 门
  只在审计行里继续活着。

## 已知边界（诚实记录）

- L2 组件内容变化仍须人工提升 `components[].contractVersion`，否则 upgrade 被
  组件注册表拒绝——该失败原样暴露给作者（auto_upgrade_failed 带错误文本）。
- dev-trust create 等 CLI 命令需要操作者自己的 super_admin 凭据（会话建立本身就
  是那次人工确认），agent 不代持凭据。
- 顺带修复：OpenAPI 路径参数此前写 `{extensionID}` 而运行时/Go 路由目录是 `:id`，
  文档端点校验（`tests/validate-docs.mjs`）一引用就红（HEAD 上已存在的 2 条失败）。
  统一为 `{id}`（index 27 条路径 + `ExtensionID` 参数组件 `name`），refs 校验 2856 条通过。
