# 2026-10-17 Session Handoff

## Changed

- `upgrade.sh`：
  - 交互模式新增更新渠道询问（1=稳定版推荐 / 2=最新发布版含 alpha/beta；回车默认稳定版，仅本次生效，不写入 `.deployrc`）；显式 `--channel`/`--version` 时跳过询问；目标版本确认行对 prerelease 追加 `(prerelease)` 标注。
  - GitHub API 查询与 `docker pull` 均为 3 次指数退避重试（`fetch_with_retry` + pull 重试循环）。
  - `.deploy.lock` 锁自愈：锁目录写入 `pid`；持有进程已死或锁龄 >1 小时自动清理残留锁；活锁仍拒绝。
  - 健康检查超时输出最后未通过的探针（api ready / web health / web api ready）。
  - 新增维护窗口降级路径：目标迁移无法在线安全执行时，交互模式询问确认后走 `maintenance_update`（备份 → 停旧槽 → 全量迁移 → 启动目标槽 → 切流，失败回滚旧槽）；`--yes` 拒绝并给出 `--confirm-maintenance` 一键授权命令。
  - `.deployrc` 版本缺失/非法时从 `api-blue`/`api-green` 容器镜像 label `org.opencontainers.image.version` 反查自愈。
  - `bootstrap_topology` 遇到不在线安全的迁移时直接在本次拓扑转换中执行全量迁移（本来就有维护窗口）。
- `sforum-bootstrap.sh`：交互 install/upgrade 同样询问发布渠道（`--lang` 影响 UI 语言；`--yes`/`--defaults`/显式 `--channel`/显式 `--version` 跳过）。
- `deploy/scripts/upgrade-version-selection_test.sh`：新增渠道询问（含 2=prerelease）、docker pull 重试、`--confirm-maintenance` 允许/拒绝/无人值守三路径、锁自愈（死进程清理 + 活锁拒绝）用例；`default-latest` 输入改为 `'\n\nn\n'`（三行：渠道、版本、确认）。
- `docs/zh-CN/deployment.md` 与 `docs/en-US/deployment.md`：补充交互渠道询问、`--confirm-maintenance`、维护窗口降级语义。

## Decisions

- 渠道选择**不持久化**到 `.deployrc`（用户确认每次交互都问）；prerelease 仍永不隐式选择（`--yes` 默认 stable）。
- 迁移不兼容时交互模式**原地降级停机升级**（用户确认），不再甩给 `./deploy.sh`；`--yes` 保持拒绝以避免无人值守静默停机，错误信息给出可复制的一键命令。

## Next

- 若希望 `deploy.sh` 首次部署也有同样的交互渠道询问，可复用相同的提示模式（本次未改 `deploy.sh`）。
- `tests/validate-docs_test.sh` 存在与本次改动无关的既有失败（fixture 缺 `apps/api/cmd/sforum/dev_extension.go`），需单独修复。

## Open Questions

- 无。
