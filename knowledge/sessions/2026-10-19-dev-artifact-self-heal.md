# 2026-10-19 Dev Artifact Self-Heal & Lifecycle Terminal Consistency

## Changed

- `app/Models/Extensions/external_sources.go`: stage 幂等从「DB 里 version+digest
  相同」改为 `externalArtifactReusable`——除版本与摘要外，还要求记录路径等于本次
  `SnapshotExternalSourceOwned` 刚用 `DigestTree` 校验过的 `snapshot.Root`。active
  与 staged 记录都必须满足；空路径一律不可复用。路径漂移的自愈走
  `SaveInstalled`，事件为 `EventVerified`（“missing or relocated … re-published”），
  真正的新内容仍是 `installed`/`upgraded`。目录在同一内容寻址路径被删时，快照
  重建已完成自愈，不写 DB、不写台账。
- `app/Models/Extensions/postgres_store.go` (`ensureExtensionVersion`): `ON CONFLICT
  (extension_id, version, package_digest)` 由 `DO NOTHING` 改为只修 `package_path`
  （`WHERE … IS DISTINCT FROM`），manifest/摘要仍不可改写。同摘要换路径不再永久
  missing。
- `app/Models/Extensions/artifact_retention.go`: 新增 `ErrArtifactUnusable` 与
  `stagedArtifactUsable`（目录存在 + manifest 可解析且 ID/Type 一致 + `DigestTree`
  等于冻结摘要 + 声明的 backend entry 是普通文件）。放在既有「制品判据唯一来源」
  文件里，避免给已达 92 文件上限的 `Models/Extensions` 再加文件。
- `app/Models/Extensions/dev_auto_runtime.go`: `promoteOne` 在
  `ErrLifecycleCoordinatorRetryRequired` 后先过 `stagedArtifactUsable`，
  再 `recoverLatestFailedUpgrade`。恢复扫描现在只认 `PackageDigest` 等于当前
  staged 摘要的操作，且 `AttemptCount >= devAutoMaxFailedUpgradeAttempts (3)` 即
  放弃；放弃只写一条 `dev_auto_skipped` 台账（按 ext+digest 去重），不再自动恢复。
- `app/Models/Extensions/lifecycle_inspection.go` + `contracts/openapi/schemas/
  extensions.yaml` + 前端类型：生命周期汇总新增 `closed`（`completed_at != nil`），
  与 `ListOpenLifecycleOperations` fence、`RecoverLifecycleOperation`、CLI 同一谓词。
  管理端在 `closed` 时把 `state` 标注为「最后阶段」并新增 `lifecycle.states.*`。
- `cmd/sforum/recovery_lifecycle_close.go`: already-closed 报错带上
  `state/terminal/completedAt`，不再只回一句「已关闭」。

## Decisions

- 幂等判据锚定「本次校验过的快照根」而不是额外 `os.Stat`：快照发布已经证明目标
  目录存在且字节等于摘要，路径相等就是磁盘证据，零额外 I/O。
- 不改历史 `state`、不做数据迁移：`uninstall` 收尾于 `uninstalling`、`disable`
  收尾于 `draining` 是阶段语义，终态由 `completed_at`/`terminal_result` 表达。
  operation 703 的「已关闭 vs 未关」是表述歧义，不是账本错误。
- dev auto 的闸门放在「是否自动恢复」处而非首次 Upgrade 之前：保留第一次失败作为
  可诊断证据，只断掉重试风暴。

## Next

- 观察新台账 `dev_auto_skipped` 在作者机器上的可读性；若需要，管理端事件页加
  该 action 的中文标签（当前按原始 action 渲染）。
- 若要根治「相对 `EXTENSION_ROOT` 导致同一行在不同 CWD 下解析不同」，再单独评估
  config 层绝对化（生产已有绝对路径强校验，dev 默认仍相对）。

## Open Questions

- `disable` 成功也收尾在 `draining`：管理端展示已按 `closed` 修正，是否要把
  阶段名收尾改为 `disabled` 需要一次状态机评审（会动 CAS 期望值与契约枚举）。
