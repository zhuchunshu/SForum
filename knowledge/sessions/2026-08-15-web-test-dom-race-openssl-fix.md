# 2026-08-15 CI Quality Gate Web Test Race + OpenSSL Fix Handoff

## Changed

- **Web Bun 测试 DOM 初始化竞态修复**：`@vue/runtime-dom` 在模块求值时用
  `typeof document !== "undefined" ? document : null` 固定内部 `doc` 引用。
  `bun test` 默认在单一进程内共享模块注册表，若某个不挂载 DOM 的测试文件
  （如 `editorL2Load.test.ts` → `sfEditor.ts` → `@tiptap/vue-3` → `vue`）
  先于任何挂载测试导入 Vue，`doc` 会被冻结为 `null`，后续所有 `mount()` 抛
  `TypeError: null is not an object (evaluating 'doc.createElement')`。加载顺序
  在 CI 与本地不同，故本地单文件/批次可过而 CI 失败。
  - 新增 `apps/web/tests/helpers/dom.ts`：集中 `installTestDom(options?)`，
    在 globalThis 上注册 happy-dom 的 `window/document/navigator` 等。
  - 新增 `apps/web/tests/helpers/setup-dom.ts` 作为 Bun 测试 preload。
  - 新增 `apps/web/bunfig.toml`：`[test] preload = ["./tests/helpers/setup-dom.ts"]`，
    保证 document 在任何 vue/@vue/test-utils/@tiptap/vue-3 导入之前注册。
  - 重构批次内 4 个挂载测试（authProvidersPublicUi / authRouteRendering /
    adminLoginMethods / accountSecurityM4b）与 `tests/helpers/vueSfc.ts` 复用
    集中 helper，移除各自模块顶层的 `Object.assign(globalThis, …)`。
  - 新增回归测试 `tests/framework/editorDomLoadOrder.test.ts`：静态导入
    `sfEditor`（先拉入 Vue）后再挂载组件；无 preload 时必然复现 `doc.createElement`
    空引用，有 preload 时通过。
  - `scripts/test.sh` 的 7 文件批次加入该回归测试（现为 8 文件）。

- **Web 运行镜像 OpenSSL 漏洞修复（CVE-2026-45447）**：`oven/bun:1.3.14-alpine`
  （Alpine 3.22.4）自带的 `libcrypto3/libssl3 3.5.6-r0` 存在 HIGH 漏洞，修复版
  为 `3.5.7-r0`。在 `apps/web/Dockerfile` 的 `prod` 阶段新增
  `RUN apk add --no-cache --upgrade 'libcrypto3>=3.5.7-r0' 'libssl3>=3.5.7-r0'`。
  未引入 Node、保持非 root `sforum` 用户与现有 Bun 启动方式。

## Decisions

- 采用 Bun test `[test].preload`（而非拆分 DOM/非 DOM 进程）作为确定性隔离：
  preload 在每个进程、每个测试文件导入前统一安装 DOM，且只影响 `bun test`，
  不影响 `vue/server-renderer` 的 SSR 语义。未添加 `.trivyignore` 或降级扫描。

## Verification

- 8 文件批次连续多次通过（38 pass / 0 fail）。
- `cd apps/web && bun test`：884 pass / 0 fail（128 files）。
- `cd apps/web && bun run typecheck` 与 `bun run build` 通过。
- 架构边界校验通过；`git diff --check` 干净。
- prod-stage 复现镜像 `apk list` 显示 libcrypto3/libssl3 3.5.7-r0；Trivy
  `HIGH,CRITICAL` + `--ignore-unfixed --exit-code 1` 扫描 0 漏洞。

## Next

- 待 push 后由 CI 的 container/web Trivy 扫描与 Quality gate 复核确认。
