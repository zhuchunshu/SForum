# 2026-10-19 扩展设置保存提速（包校验记忆化）

## Changed

- **根因**：设置保存不是纯写库，而是「写库 + 重启插件」。实测 `put_ms=5`、
  `restart_ms=12249`。CPU profile 显示 85% 采样落在 `sha256.blockAVX2`，
  即对同一份不可变扩展包反复做整树哈希。
- **后台饱和源**：`bootstrap/api_assembly_http.go` 用 `RunRefresh(ctx, 5s)` 起了常驻循环，
  每 5 秒遍历全部扩展、每个 `source=uploaded` 扩展调用两次 `artifactTrusted`，
  最终落到 `DigestTree`；库内 16 个扩展 ≈ 32 次整树哈希 / 5 秒。
  40 秒窗口内 `GuardPolicyCatalog.RunRefresh` 独占 27.39 s CPU。
- **新增** `app/Support/ExtensionPackage/verification_cache.go`：
  `VerifiedTreeDigest` / `VerifiedFileDigest` / `VerifiedFileDigestFunc` / `InvalidateVerification`。
  指纹只读元数据（路径集合 + 大小 + mtime(ns) + 权限位），与 `DigestTree` 共用
  `collectDigestFiles` 同一份文件集合；`VerificationCacheTTL = 5 分钟`，上限 1024 条。
- **`digest.go`** 抽出 `collectDigestFiles` + `hashDigestFiles`，`DigestTree` /
  `DigestAuthorTree` 语义不变；`digestFile` 增加 `info` 字段供指纹复用。
- **调用点切换**：`verifyInstalledPackageIdentity`、`loadArtifact`、
  `digestInstalledFile`（保留 `digestStableExtensionFile` 的 TOCTOU 防御作为未命中路径）、
  `extensionpackageRootDigest`、`verifyExistingSnapshot`。
- **删除重复校验**：`LifecycleBoundaryRequest` 携带 `artifactLoader`（按值随克隆传递），
  同一 gate 内「显式校验 + 发布准备」共享一批已加载制品；`validatePreparedLifecycleRegistries`
  与 `validateLifecycleRoutePolicyStates` 增加 loader 参数。
- **dev 构建显式失效**：`BuildAndDigestDevPackage` 构建前调用 `InvalidateVerification`。

## Decisions

- 见 `decisions/2026-10-19-package-verification-memoization.md`：指纹记忆化的取舍、
  残余风险（同尺寸同 mtime 的篡改在 TTL 内不被发现）、以及明确不做的两项
  （缩小 `Refresh` 扫描范围、设置热加载）。
- 曾用 ctx 携带 loader，但 `TestExactLifecycleCoordinatorHostRequiresComposedBoundary`
  断言的契约是「boundary 收到的 ctx 必须是调用方那一个对象」，因此改为显式传参，
  不去削弱既有的上下文透传契约。

## Result

| 阶段 | 优化前 | 第一轮（loader 去重） | 本轮（记忆化） |
|---|---|---|---|
| disable 相位 | 9.9 – 15.5 s | 4.6 – 7.1 s | **0.89 – 1.19 s** |
| enable 相位 | 14.6 – 23.5 s | 6.7 – 12.1 s | **1.81 – 2.11 s** |
| 单次保存 | 24.5 – 39 s | 11.4 – 18 s | **≈3.0 s** |
| 后台 `RunRefresh` / 窗口 | 27.39 s | — | **0.18 s** |
| 进程总 CPU | 2.03 核 | — | **0.64 核** |

## Next

- 后台仍余约 8.9 s/45 s 的整树哈希，来自 dev watcher 的快照管线
  （`SnapshotUploadedOwned` 必须哈希新内容，属正常；`verifyExistingSnapshot` 已切记忆化）。
- `ExtensionManifest.ValidatePackageFiles` 未加记忆（`ExtensionManifest` 不能反向
  import `ExtensionPackage`，记忆需落在该包内）；当前保存已达 ~3 s，未触发实施条件。

## Open Questions

- dev watcher 仍会因包内容变化触发重建；重建后摘要变化会使既有可信授权失效，
  保存期间可能弹出可信复核对话框并把插件留在 `disabled` 直到确认。这是设计内的
  失败关闭行为，但 dev 下的重建频率会放大它的可见度。
