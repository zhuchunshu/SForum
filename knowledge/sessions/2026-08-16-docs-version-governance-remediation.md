# 2026-08-16 Session Handoff

## Changed

**stable/prerelease 通道语义（deploy.sh / upgrade.sh）**
- `upgrade.sh` 新增 `--channel stable|prerelease`，默认 **stable**；`latest`
  只解析 `/releases/latest`（稳定 Release）。`--channel prerelease` 解析
  `releases?per_page=1`（最新发布含预发布）。显式 `vX.Y.Z-alpha.N` 始终可用。
- `deploy.sh` 同样新增 `--channel`（默认 stable），`latest` 走
  `/releases/latest`；无稳定 Release 时 fail-closed 并提示改用
  `--channel prerelease`。
- 两种脚本的帮助文本更新；中英文部署文档与 README 同步（
  `tests/validate-docs.mjs` 校验文档含 `--channel prerelease` 与稳定安装入口）。
- 测试：`deploy/scripts/upgrade-version-selection_test.sh` 重写覆盖
  stable / prerelease / 显式版本 / `--yes` 无人值守 / 无可用 Release /
  非法通道与版本；`deploy/scripts/deploy_test.sh` 新增 channel 用例。

**Release 部署资产**
- 新增 `scripts/ci/build-deploy-asset.sh`：打包固定名称
  `sforum-deploy.tar.gz`（compose*.yaml、deploy.sh、upgrade.sh、
  `.env.production.example`、deploy/scripts/*、Caddy 示例、VERSION、README）
  与独立 `upgrade.sh` 资产。
- `scripts/ci/finalize-release-assets.sh` 期望清单加入
  `sforum-deploy.tar.gz` + `upgrade.sh`（含内容与可执行校验），计入
  `SHA256SUMS`；`release.yml` 新增 `deploy-asset` job（artifacts
  `release-asset-deploy`），attestation 自动覆盖新资产。
- `generate-release-notes.sh` 发布说明加入部署包下载与 SHA256SUMS/
  `gh attestation verify` 校验步骤。
- `scripts/ci/release_assets_test.sh` 覆盖新资产（9 文件 / 8 校验行）。

**文档事实与补齐**
- Go 要求统一为 1.26.6（锚定 `apps/api/go.mod`）：AGENTS.md、
  getting-started（中/英）、development/setup（中/英）、
  knowledge/modules/backend.md。
- README 与中英文 deployment.md 改用稳定滚动安装入口
  `releases/latest/download/sforum-deploy.tar.gz`，不再硬编码 v3.0.1；
  固定部署示例使用 `$SFORUM_VERSION`；新增校验下载、通道语义、
  健康检查/日志/故障恢复、备份恢复命令等运维章节；保留
  v3.0.0-alpha.12/13 历史兼容说明。
- 新增文档（中英平行）：`usage/account-security.md`、
  `development/api.md`（认证/CSRF/PAT/信封，仅依据 OpenAPI 与实现）。
- 重写 `development/cli.md`（中/英）：完整命令表（version、seed:perf、
  users:reset-password、revisions backfill、extension quarantine、
  extension system-tier、dev:cleanup-orphan-plugins 及全部缺失 flags）。
- 扩展 `usage/admin.md`（中/英）：用户/角色/权限、论坛/分类/标签、审核、
  附件与 FS/S3、SMTP/测试邮件、品牌/导航/公告、SEO、Webhook、系统更新与任务。
- 新增 `extensions/builtin/plugins/sforum-smtp/README.md`；扩充过短的
  search-site、storage-s3、web-push README。
- 更新 docs/README.md、中英文文档首页、usage/development 索引。

**文档 CI 门禁**
- 新增 `tests/validate-docs.mjs`：本地链接与标题锚点、中英文件结构平行、
  滚动文档禁真实发行号（历史预发布段落白名单）、Go 版本与 go.mod 一致、
  CLI 总览覆盖真实命令（从 cmd/sforum Use: 字符串派生）、稳定安装入口、
  Release workflow 产出部署资产并纳入 SHA256SUMS。已接入 `scripts/test.sh`。
- 新增 `.github/dependabot.yml`：Actions、Docker、Go（apps/api + 7 个内置
  插件模块）、npm/apps/web（Bun 仓库，Dependabot 按 package.json 跟踪），
  全部手动审查，不自动合并。

## Decisions

- 滚动安装入口使用“最新稳定版”；预发布必须显式 `--channel prerelease` 或
  显式版本；内部始终解析为具体 vX.Y.Z 并运行对应 GHCR 镜像，生产 Compose
  不运行浮动 latest。
- GitHub Actions 继续固定 commit SHA；Go/Bun/npm 依赖继续 lockfile；Docker
  生产镜像继续不可变 Release tag；协议版本/Manifest V3/Host API v2 继续固定。
- 文档中不写“当前版本”，演示固定部署一律用 `$SFORUM_VERSION`。

## Review Fixes (2026-08-16, second pass)

**下载与校验流程**
- README、中英文 deployment.md、generate-release-notes.sh 统一为：
  先下载 `sforum-deploy.tar.gz` → 再下载 `SHA256SUMS` → `grep` 精确条目到
  独立文件 → `sha256sum -c`（macOS `shasum -a 256 -c`）→ 可选
  `gh attestation verify` → 校验通过后才 `tar -xzf`。不再把压缩包管道直接
  喂给 tar，不再以 `--ignore-missing` 作为主要校验。
- generate-release-notes_test.sh 新增回归：检测 `| tar -xz` 管道、
  `--ignore-missing`、缺少 `-o sforum-deploy.tar.gz` 保存步骤，以及
  解压行出现在下载行之前。

**deploy.sh 当前版本 vs 目标版本**
- `resolve_target_version`（deploy/install/update）：显式 `--version` 优先，
  否则按 `--channel` 解析 latest；**不再**读取 `.deployrc`/`.env.production`
  旧版本作为目标。
- `resolve_current_version`（status/logs/restart/stop/backup/restore/
  rollback）：默认用已部署版本，不访问 GitHub；显式 `--version` 仍优先。
- 交互菜单逐项解析（安装或更新 → target；其余 → current）。
- deploy_test.sh 新增 4 个回归：已有 .deployrc + stable/prerelease/显式版本
  的 deploy，以及已有 .deployrc + status 不访问 GitHub。

**upgrade.sh 可信下载入口**
- 文档刷新更新器改为 `releases/latest/download/upgrade.sh`（不再用
  `raw.githubusercontent.com/.../main/upgrade.sh`），同流程精确校验
  `upgrade.sh` 条目 + 可选 provenance；显式版本用
  `releases/download/<TAG>/upgrade.sh`；alpha.13 历史段落保留不可变 tag。

**文档事实修正**
- api.md：密码修改为 `POST /auth/password`；外部登录路径写全
  start/complete（POST）与 callback（GET）；CSRF 豁免列出 Bearer PAT 与
  `/webhooks/inbound/{source}`（gateway skeleton）；Core topics/comments 的
  Idempotency-Key 为可选，缺失不报 400，仅声明必须幂等的插件路由才 400。
- admin.md：删除“运行时站点选项可改后台路径”；分类地址改为
  `/control-panel/forum/categories`；Webhook 入站如实写为 gateway skeleton
  （仅非空 body + 回执），出站为 River 后台自动重试、后台无手动重试按钮。
- 无效示例 `export SFORUM_VERSION=$SFORUM_VERSION` 改为
  `SFORUM_VERSION=<VERSION>` + `export SFORUM_VERSION`。

**validate-docs.mjs 加强**
- 未知 `new*Command` 构造器现在直接失败（不再静默跳过；同时修复了
  key 拼接未剥离 `()` 导致映射恒为空的隐患）。
- 本地链接校验覆盖扩展到 `extensions/**/README.md`；修正了
  sforum-search-site README 的错误相对路径（`../../../` → `../../../../`）。
- 版本检查改为**精确 token 白名单**（仅 `v3.0.0`，理由为 alpha.13 历史
  兼容说明），一行内即使出现 alpha/beta/rc 也不再整行豁免。
- 新增 Dockerfile golang 基础镜像与 go.mod 工具链一致性检查。
- 新增 `tests/validate-docs_test.sh`，第三轮扩展到 9 个用例（1 正 + 8 失败
  路径：扩展 README 断链、安装命令非 fail-closed、未知 CLI 构造器、未解析
  AddCommand 参数、Dockerfile 版本漂移、两个稳定号场景、历史 prerelease
  前缀误豁免），用 `SFORUM_VALIDATE_ROOT` 指向夹具树运行并断言
  具体失败消息；已接入 scripts/test.sh。

**依赖更新治理**
- dependabot.yml：新增根目录 `docker-compose` ecosystem（跟踪 compose*.yaml 中的
  PostgreSQL/Redis/Meilisearch/Caddy/Mailpit 镜像）、`/tools/proto` 与
  `/tests/compat` gomod；fixtures 模块明确排除并注释原因（CI-only 契约
  夹具，非生产/非运行）；weekly + groups + npm PR limit 控制噪声；
  不把镜像改成 latest。

## Review Fixes (2026-08-16, third pass)

**下载校验真正 fail-closed**
- README、中英文部署文档与自动 Release notes 的可执行片段统一放入
  `set -eu` 子 shell；下载、唯一条目检查、checksum 或已安装 `gh` 的
  attestation 任一步失败，都会在 `tar`、`chmod`、脚本执行前终止。
- checksum 选择从行尾 `grep` 改为 `awk '$2 == "<filename>"'`，并要求结果
  恰好一行；历史 alpha.13 raw tag 下载增加已核实的 immutable SHA-256。
- `generate-release-notes_test.sh` 与 `validate-docs.mjs` 增加 fail-closed、
  精确文件名字段、唯一条目回归约束。

**交互菜单版本状态隔离**
- `deploy.sh` 分离 `REQUESTED_RELEASE_VERSION` 与工作态 `RELEASE_VERSION`；每次
  菜单动作都从显式请求或对应 current/target authority 重新解析，避免先查看
  status/logs 后 deploy 静默复用旧版本。
- `deploy_test.sh` 新增同一进程 `status -> deploy` 回归，要求重新访问稳定
  Release API 并拉取解析后的稳定版本。

**文档与依赖治理门禁**
- CLI 校验在识别已知 `new*Command(...)` 后检查 AddCommand 参数块剩余内容；
  `existingCommand`、`buildRootChild()` 等非匹配表达式现在直接失败，并有夹具
  回归。
- 版本扫描改为识别完整 semver token：允许完整 prerelease 示例/历史事实，
  但 bare `v3.0.0` 不再因共享前缀获得全局豁免。
- Compose 固定到已从公开 registry 确认存在的标签：PostgreSQL
  `17.11-alpine`、Redis `7.4.10-alpine`、Meilisearch `v1.53.1`、Mailpit
  `v1.30.7`；根目录使用专门的 `docker-compose` ecosystem（`docker` 的 YAML
  fetcher 只保留 Kubernetes/Helm 清单），Dependabot 继续负责 reviewed upgrades。
- 管理后台文档将“每 5 分钟刷新”修正为真实语义：结果最多缓存 5 分钟，且
  管理员可手动强制检查；未声明后台定时刷新。
- `validate-docs.mjs` 现在提取滚动文档中每个明确的 HTTP method/path，分别
  对照模块化 OpenAPI path item 与生成的 Go Core Route Catalog；失败路径测试
  独立覆盖 OpenAPI method 漂移和 Go 路由注册漂移。Dependabot 的 Compose
  ecosystem 与 governed go.mod 覆盖也纳入门禁；当前夹具套件共 13 个用例
  （1 正 + 12 失败）。

## CI Fix (2026-08-17)

- 修复 Quality gate 的干净检出断链：`extensions/optional/README.md` 过去在
  本地存在但整个 `extensions/optional/` 被 `.gitignore` 排除，导致
  `extensions/README.md` 的 `./optional/` 链接只在开发机通过。现在可选扩展
  目录随仓库跟踪，仅忽略生成的后端二进制与打包产物。
- `validate-docs.mjs` 在 Git worktree 中拒绝指向 gitignored 本地目标的
  Markdown 链接，防止本地文件掩盖 CI 断链；夹具套件新增对应失败路径，
  当前共 14 个用例（1 正 + 13 失败）。完整仓库门禁、Go 构建、884 个 Web
  单测和 Nuxt 生产构建通过。

## CI Fix (2026-08-17): release artifact execute-bit normalization

- **根因**：v3.0.9-alpha.2 的 verify-release-assets job 失败于
  "Standalone upgrade.sh asset is not executable"。`build-deploy-asset.sh`
  正确地把 dist/upgrade.sh 置为 0755，但
  `upload-artifact` → `download-artifact` 跨 job 传输不保留 Unix 文件权限，
  普通文件以 0644 落地，随后 finalizer 的 `[[ -x ]]` 检查失败。本地
  `release_assets_test.sh` 未模拟这条 artifact 边界，因此未被发现。
- **修复**（`scripts/ci/finalize-release-assets.sh`）：保留文件存在/非空/
  首行 shebang 检查；新增 `bash -n` 校验 standalone upgrade.sh 语法；内容
  校验通过后 `chmod 0755`，再做 `-x` 后置检查；最后才生成 SHA256SUMS。
  注释说明 GitHub Artifact 不保留执行位。chmod 只放进 finalizer（不放
  release.yml），finalizer 可独立处理真实 artifact 输入。
- **回归**（`scripts/ci/release_assets_test.sh`）：build-deploy-asset.sh
  后先断言 upgrade.sh 原本可执行 → `chmod 0644` 模拟 Artifact 下载结果 →
  运行 finalize 必须成功 → 断言执行位已恢复 → 断言 SHA256SUMS 有且仅有一
  条 upgrade.sh 条目 → 新增 tar 内权限断言（deploy.sh、upgrade.sh、
  deploy/scripts/*.sh 在压缩包内保持 `-rwxr-xr-x`）。变异验证：临时移除
  finalizer 的 chmod 后测试按预期失败（"not executable"），证明回归确实
  能捕获原始缺陷。
- **验证**：`bash -n` 三个脚本、`release_assets_test.sh`、
  `generate-release-notes_test.sh`、`git diff --check`、
  `release_workflow_test.rb` 全部通过；完整 `./scripts/test.sh` 跑到
  compatibility farm 为止（`DATABASE_URL`/`SFORUM_TEST_DATABASE_URL` 未配
  置，postgres cells 必败，既有文档化限制），此前所有门禁含 `go test
  ./...` 全部通过。actionlint 未安装，release.yml 未改动。
- **剩余风险**：最终确认仍依赖下一次真实 release tag 运行（不得移动
  v3.0.9-alpha.2 标签，需新建 prerelease tag）。

## Next

- 修复合并后新建 prerelease tag 验证 `/releases/latest/download/
  sforum-deploy.tar.gz` 与 `upgrade.sh` 资产可下载、checksum/attestation
  正确，并验证独立 updater 按文档 `chmod 0755` 后可执行（流水线已在
  release.yml 就绪，未实际发布测试）。
- 解决下方阻塞性冲突后再决定是否发布第一个稳定 Release。

## Open Questions

- **阻塞性冲突（未掩盖）**：仓库路线图与模块笔记明确要求“生产重接线
  M3/M5/M6/M7 关闭前，当前发行必须继续标记为 prerelease”，而
  `knowledge/index.md`（2026-08-01）记录“v3.0.1 已准备进入 release gate”。
  我无法在本机核实 GitHub 当前实际 Release 状态（无网络确认）；若 v3.0.1
  已作为 stable 发布，则与路线图诚实性承诺矛盾。需要产品决定：是继续把
  发行标记为 prerelease 直到生产重接线关闭，还是修订路线图。SECURITY.md
  的“仅支持最新稳定版”在首个 stable 发布前意味着无正式受支持版本，也需
  一并确认。
- 本地 `go test ./...` 已通过。完整 `./scripts/test.sh` 仅在 compatibility
  farm 停止：当前环境未配置 `DATABASE_URL` / `SFORUM_TEST_DATABASE_URL`；
  compatibility farm 之后的门禁已单独运行并通过。
- 备份策略、`en-US` 完整度等既有 Open Questions 保持不变。
