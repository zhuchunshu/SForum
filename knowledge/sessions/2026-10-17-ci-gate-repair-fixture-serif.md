# 2026-10-17 CI Gate Repair Handoff

## Changed

推送 `6ab053f42` 后 main CI 双红（contracts gate + Go gate），本地复现后修复并单独提交：

1. **contracts gate**：`tests/validate-page-registry-runtime.js` 仍把 catch-all 断言绑在单个路由文件上；`6ab053f42` 已把解析收敛到 `useRegistryPageResolve` + `SFRegistryPageBody/Shell`。改为按「路由壳 + 专属协作者」拼接校验，并补一条 `useRegistryPageResolve` 引用断言。
2. **contracts gate（隐性）**：本次推送源码新增组件后 `public-page-surfaces.json/md` 未再生成；另外 validator 的手工计数 tripwire（routes 362 / UI 317）落后于实际（363 / 321）。重新运行 `scripts/v3-catalog/generate.mjs` 并把两处计数 bump 到 363/321（沿用 1fcfdbf05 的 exact-count bump 惯例）。
3. **Go gate**：两个新参考插件 fixture（`sforum-page-loader-reference`、`sforum-session-checkout-reference`）manifest 按 exact-artifact 写死 `backend/plugin` SHA-256，但二进制不提交且摘要随工具链变化无法复现；静态契约测试（`sdk/plugin`、`ExtensionOpenAPI`）直接加载源码树 manifest 于是必炸。新增共享助手 `app/Support/ExtensionPackage/referencefixture.StageBackendFixture`：临时副本 → 构建后端 → 摘要回写 → 校验，语义与已有 `stageSessionCheckoutFixture` / `reference-packages-gate.sh` 一致。**教训：助手初版把摘要写回了源码树 manifest（已修复并用 git checkout 还原）——暂存类助手永远只写 `t.TempDir()` 内的副本。**
4. **Go gate（隐性）**：`TestPrivacyEraseUserUsageContract` 依赖“`DATABASE_URL` 为空”这一环境假设，而 `go-gate.sh` 会 export 真实隔离库 URL，导致走到真实连接而非可读的空连接串报错。测试内 `t.Setenv("DATABASE_URL", "")` 固定化。
5. 顺手清理：`6ab053f42` 意外提交了 55MB 的 `apps/api/sforum` ELF 二进制（无任何脚本引用）；已从跟踪移除并加入 `.gitignore`。**历史大小仍在老 revision 里；如在意 repo 体积可后续 rewrite。**

## Decisions

- fixture 二进制继续不入库，测试以「暂存副本+现场构建+摘要对齐」为准——与 `scripts/ci/reference-packages-gate.sh` 的定义互相印证（fixtures 是 CI 夹具，不是可分发作者包）。
- validator 的 exact-count（routes/）继续充当“新增核心面必须人工评审”的 tripwire，新增面时同步 bump 数字。

## Next

- 无遗留；CI 应自绿。用户本地有一组未提交的 `SFExtensionWidget` L2 有界退避重试改动（连同其测试），不在本次修复范围内，待用户自行提交。
