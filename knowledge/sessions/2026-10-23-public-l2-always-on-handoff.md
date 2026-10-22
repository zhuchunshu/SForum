# 2026-10-23 Session Handoff — Public L2 开关移除

## Changed

- 删除 `SFORUM_V3_PUBLIC_L2`：`config.Config.V3PublicL2`、env 读取、
  `FrontendService.publicL2`、`WithPublicL2`、装配点，以及 `compose.yaml`、
  `compose.prod.yaml`、两份 env 模板、`deploy/scripts/configure-production.sh`。
- 三处 gate（`publicRuntimeGates`、`publicPagePolicyGates`、`PublicContentStyles`）
  只保留 `safeMode` 与 `v3TrustChallenges`。公开 L2 恒开。
- 测试：删 `TestLoadDefaultsPublicL2ByEnvironment`；清掉 4 处 `WithPublicL2(true)`；
  P9 E2E 子进程 env 去掉该变量；新增
  `TestPublicFrontendStaysClosedWithoutTrustChallengeGate`（信任门仍是独立关）。
- `tests/validate-v3-production-trust.mjs` 反转为"禁止再引入该变量"（API config、
  前端装配、两份 Compose、两份 env 模板、生产 env 生成脚本）。
- 文档：中英 `deployment/environment.md`、`docs/extensions/v3/governance.md`
  gate 表、`docs/extensions/authoring-guide.md` 的 public L2 段。
- 决策记录 `2026-10-23-public-l2-always-on.md`；`2026-07-13-trusted-plugin-theme-platform-v3.md`
  加 superseded 指针；`knowledge/modules/frontend.md` 更新。

## Decisions

- 不做兼容读取：旧部署仍设该变量会被静默忽略。
- 不引入 per-package 授权，不改 builtin 免 grant，不改 5s lease 轮询。
- 运行期唯一止损是 Safe Mode；版本级回滚用 revert。

## Verification

- `go build ./...`、`go test ./config/ ./app/Models/Extensions/ ./app/Http/Controllers/Extensions/` 全绿。
- `node tests/validate-v3-production-trust.mjs`、`node tests/validate-docs.mjs` 通过。
- 其余：`./scripts/test.sh` 与浏览器侧运行时取证（见下）。

## Next

- 跑全量 `./scripts/test.sh`（含 contracts-gate、web typecheck/build）。
- dev 实例重启后确认：`/api/v1/extensions/runtime/content-styles` → 200 且含
  `sforum-shortcodes.asset.content-style`；`SFORUM_SAFE_MODE=1` 时 404。
- 这是行为变更，发布说明需明确：旧 `SFORUM_V3_PUBLIC_L2=false` 不再生效。

## Open Questions

- 内置包免 grant + L2 恒开的组合是否需要在 builtin 升级时加显式确认：留待下一轮评估。
