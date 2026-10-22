# 2026-10-23 sforum-latest 严格更新入口

## Changed

- 新增根目录 `sforum-latest.sh`：每次运行解析 GitHub 最新 Release，下载并校验
  该 Release 的 `sforum-bootstrap.sh`，备份旧文件后再交给 `install`/`upgrade`。
- 稳定版标签从 `releases/latest/download/...` 的 302 `Location` 解析，避免消耗
  GitHub API 配额；预发布走 `/releases` API，`--version` 直接使用显式标签。
- 旧版 `gh`（没有 `attestation` 子命令）自动降级为 SHA-256 校验，并把
  `SFORUM_SKIP_ATTESTATION=true` 传给 bootstrap；支持 `attestation` 的 `gh`
  仍做来源校验。
- README、`docs/zh-CN|en-US/deployment.md`、
  `docs/zh-CN|en-US/deployment/one-click.md` 与 `knowledge/modules/backend.md`
  已同步脚本定位、用法、锁与备份路径。

## Decisions

- `sforum-latest.sh` 不作为 Release 资产，也不加入 bootstrap 的
  `managed_files`：旧 Release 的 `validate_bundle` 会因缺少该文件失败，纳入
  发布链需要先解决跨版本兼容。
- 包装器只负责取回并校验，实际安装/更新仍由 Release bootstrap 与其工具包执行；
  `--confirm-maintenance` 仍要求直接调用 `upgrade.sh`。

## Verification

- `bash -n sforum-latest.sh`。
- `./sforum-latest.sh --check`：稳定版、预发布、固定标签、位置参数四条路径通过；
  SHA-256 与 `SHA256SUMS` 条目一致。
- 在临时安装目录把 `exec` 换成桩实测 upgrade 路径：旧 bootstrap 备份、原子替换、
  `--yes --bootstrap` 透传均符合预期，运行后没有 `.sforum-latest.*` 临时目录或
  锁残留。

## Next

- 若要随 Release 分发，需要同时修改 `scripts/ci/build-deploy-asset.sh`、
  `scripts/ci/finalize-release-assets.sh`、`sforum-bootstrap.sh` 的
  `managed_files` 与 `deploy/scripts/bootstrap_test.sh`，并处理旧 Release 兼容。
- 脚本推送到 `main` 后，文档里的 raw 获取地址才会生效。

## Follow-up（2026-10-23）

- 增加 `sforum-latest.sh.sha256` 与 `tests/validate-sforum-latest.sh`，文档入口改成
  一条 curl（下载包装器 + SHA-256 校验 + 执行），并在 release-tooling gate 中防止
  哈希与文档入口漂移。
- 包装器改为同步完整 `sforum-deploy.tar.gz`：校验 tar.gz 与 `VERSION`，从包内
  bootstrap 提取权威受管清单，把 `deploy.sh`、`upgrade.sh`、compose 文件、
  `deploy/scripts`、Caddy 示例与 `VERSION` 一并原子替换；`--check` 报告缺失或过期
  的文件。Git 检出目录里实际同步需要 `--allow-git`，避免覆盖源码。
