# 扩展版本垃圾回收（Version GC，运维参考）

> **语言切换：** [English (canonical)](../../extensions/version-gc.md) · 本文为中文翻译。
> 字段名、端点与错误码以英文原文和代码为准；中文版若与代码不一致，以代码为准。

版本 GC 回收扩展的**历史版本行与其不可变磁盘目录**（`EXTENSION_ROOT` 下的
`storage/extensions/<id>/<version>/<digest>/`）。它是显式、可审计、
fail-closed 的：删除从不是自动的、定时的，也永远不会落到 active 或 staged
制品上。

决策记录：`knowledge/decisions/2026-10-17-version-gc-publication-restrict-cooperation.md`。

## 版本是什么

SForum 扩展采用「不可变制品 + 台账指针」模型：每个版本是一个只读目录，
`extension_versions` 行记录精确路径，`extensions.active_version_id` /
`extensions.staged_version_id` 是 CAS 指针。GC 把行和目录**一起**删，且只按
"先盘后账"这个方向删。

## CLI

```bash
# 默认 dry-run：打印候选与保留/阻塞原因，并签发一次性确认 token
# （与本次精确计划绑定，15 分钟有效）。
sforum extension gc --json

# 执行（token 一次性；必须重复相同的 --retain/--retain-days/--extension-id，
# 因为 token 同样绑定这些选项）。
sforum extension gc --confirm-token <dry-run 输出的 token>

# 策略旋钮。
sforum extension gc --retain 3 --retain-days 30 --extension-id <id> --json
```

`--json` schema 稳定：dry-run 输出
`{mode, options, candidates[], reclaimableCount, blockedCounts?, confirmation}`，
执行输出
`{mode, result:{removed[], skipped[], failed[], warnings?, bytesFreed, candidateCount}}`。
存在失败条目时命令在完整打印结果后以非零码退出。

## 可回收判据（必须全部满足）

1. 不是该扩展的 `active_version_id`，也不是 `staged_version_id`。
2. 没有**开放状态**的 `extension_lifecycle_operations` 行以
   `(extension_id, extension_version, package_digest)` 为目标。终态行不阻塞
   ——它们是 append-only 证据，从不清理。
3. 没有被任何 `plugin_runtime_publication_members` 行引用。这些行受
   append-only 触发器保护（迁移 `202607160027`），因此**被发布过的版本行是
   永久钉住的**——报告为 `blocked=publication_history`，这不是错误。
4. 超出保留窗口：每扩展最新 `--retain` 个历史版本（默认 `1`，即
   active + 1）与最近 `--retain-days` 天内安装的版本（默认 `0` = 关闭）
   保留。
5. 磁盘目录存在。行在、盘缺的候选是 `blocked=disk_missing`，属于**缺失制品
   清理**流程（管理端 `POST /admin/extensions/missing-artifacts/uninstall`），
   不归 GC 管。

## 诚实的回收面

每次插件 enable/upgrade/disable 都会追加一条全量发布修订，所以**曾被启用过
的插件版本行永久保留**。GC 的实际回收面因此是：

- 主题历史版本（发布账本只覆盖插件——`enforce_plugin_runtime_member_type`
  触发器拒绝非 plugin 成员），以及
- 从未进入过发布的插件版本行（上传后未启用、启用前 staged 候选被替换、被
  discard 的候选）。

要回收"曾上线"的插件版本，前提是放宽 append-only 触发器或引入发布历史
归档机制——那是改变多节点收敛语义的独立决策，不属于版本 GC。

## 执行序（逐条）

1. 统计目录体积，然后**改名到暂存名**（原子操作）。改名失败该条中止，
   现场不变。
2. 单事务内：`FOR UPDATE` 锁版本行，**锁内复核全部判据**（防并发
   enable / upgrade / rollback 与跨节点竞态），`DELETE` 行，写
   `extension_version_gc_removals` 锚点，追加 `extension.version_gc` 审计
   事件。任何 FK RESTRICT 失败都会回滚该条。
3. 事务失败时，暂存目录恢复原路径——GC 绝不制造"行在、盘缺"的悬挂态。
   恢复也失败时，错误里显式携带暂存路径。
4. 提交后删除暂存目录。失败时该条仍报告为已移除，但附带携带残留路径的
   警告。

幂等：重复执行安全；已删除的行自然不再是候选。

## 阻塞原因

| 原因 | 含义 | 运维动作 |
| --- | --- | --- |
| `publication_history` | 被 append-only 发布账本钉住 | 无（设计上永久） |
| `open_operation` | 有开放 lifecycle operation 以该版本为目标 | 等它完成或带外关闭 |
| `within_retention` | 仍在 active + N / 最近 N 天窗口内 | 仅在明确意图下调大 `--retain` / `--retain-days` |
| `disk_missing` | 行在、盘缺 | 走缺失制品清理，不走 GC |

## 可审计性

每个被移除的版本产生一条 `extension.version_gc` 审计事件（扩展 id、版本、
摘要、路径、释放字节数）和一行持久的 `extension_version_gc_removals` 锚点，
两者与删除写在同一事务。审计事件遵循平台保留策略；锚点表是持久的逐条
移除台账。
