# 2026-10-19 扩展包校验记忆化

## 状态

已实施。

## 背景

「保存设置」的响应时间长期在 24–39 秒。实测拆解（`extension settings save stages`）：

- `put_ms=5` —— 设置文档落库确实是毫秒级；
- `restart_ms=12249` —— 其余全部花在「重启插件以加载新设置」上。

重启之所以必须存在：插件配置通过 `SFORUM_SETTING_*` 环境变量在子进程启动时注入
（插件侧 `config.go` 与宿主 `pluginSettingEnvName` 同源），进程不重启，新值到不了插件。

40 秒 CPU profile 显示瓶颈不是数据库，而是对**同一份不可变扩展包**的重复整树哈希：

| 观测项 | 数值 |
|---|---|
| `crypto/internal/fips140/sha256.blockAVX2` | 85.18% of samples |
| `ExtensionPackage.DigestTree` | 42.63 s（同一窗口） |
| `GuardPolicyCatalog.RunRefresh` | 27.39 s / 8 个 tick ≈ 3.4 s 每轮 |
| `ExtensionManifest.ValidatePackageFiles` | 8.52 s |
| 单次保存的全量 registry 校验次数 | 3–5 遍 |

三条互相叠加的重复来源：

1. `bootstrap/api_assembly_http.go` 用 `RunRefresh(ctx, 5s)` 起了一个常驻循环，
   每 5 秒遍历全部扩展，对每个 `source=uploaded` 的扩展调用
   `freezeEntry → artifactTrusted`（当前制品 + review 制品各一次）→
   `RuntimeIdentity → buildTrustImpact → verifyInstalledPackageIdentity → DigestTree`。
   库内 16 个扩展即约 32 次整树哈希 / 5 秒。
2. `preparePublication` 内 `ValidateLifecycleRegistries` 与
   `PrepareLifecycleRegistryPublication` **各自**再跑一遍
   `validatePreparedLifecycleRegistries`，同 gate 内同 material 重复校验。
3. `buildTrustImpact` 的 `digestInstalledFile` 与 `LoadPackage` 的
   `ValidatePackageFiles` 又会各自完整读取一次 `backend/plugin`（25–60 MB）。

## 决策

引入**指纹记忆化**：以「树指纹」为键复用一次已经完成的校验结果。

- 指纹只读元数据（路径集合 + 大小 + mtime(ns) + 权限位），不读内容，比整树哈希便宜
  约三个数量级；遍历规则与 `DigestTree` 共用同一份文件集合
  （`collectDigestFiles`），避免指纹口径与摘要口径分叉。
- 新入口 `VerifiedTreeDigest` / `VerifiedFileDigest` / `VerifiedFileDigestFunc`
  （`app/Support/ExtensionPackage/verification_cache.go`）。`DigestTree` 与
  `DigestAuthorTree` 本身不改，仍是永远新鲜的底层原语（CLI 校验、测试继续用它）。
- 可信路径保留自己的校验实现：`VerifiedFileDigestFunc` 把真实校验交给调用方
  （`digestStableExtensionFile` 的 TOCTOU 防御原样保留），缓存只决定「这次要不要再跑」。
- `VerificationCacheTTL = 5 分钟`，条目上限 1024；到期即使指纹未变也完整重算，
  作为指纹被绕过的兜底。
- 显式失效点：`BuildAndDigestDevPackage` 在构建前调用 `InvalidateVerification(root)`，
  不依赖指纹。

另外把 `preparePublication` 内的重复校验合并：`LifecycleBoundaryRequest` 携带一个
`artifactLoader`（按值随克隆传递），使同一 gate 内「显式校验 + 发布准备」共享同一批
已加载制品。

## 已知取舍与残余风险

- **指纹不含内容**：同时伪造文件大小与 mtime 的篡改，在 TTL 窗口（≤5 分钟）内不会被
  发现。窗口由 TTL 与显式失效点共同约束。
- inode 未纳入指纹：需要平台相关的 `syscall`，而路径集合已由遍历固定，收益不足以引入
  构建约束。
- 缓存是进程内的：多节点部署下每个节点各校验一次，符合既有「精确制品」不变量。
- 该决策**不改变**「同一路径必须对应同一内容」这一前提的适用范围：
  `SnapshotUploadedOwned` 的落盘布局是 `<root>/<id>/<version>/<digest>`，目录名即内容摘要，
  内容寻址由构造保证。

## 边界（明确不做）

- 不缩小 `GuardPolicyCatalog.Refresh` 的扫描范围（不跳过 disabled/installed）：该目录同时
  供后台管理 UI 使用，改范围有 UI 语义风险，且记忆化后每轮成本已可忽略。
- 不做设置热加载（protocol v2 新增热更新通知）：那是治本方案，涉及契约版本，单独立项。
- 未改 `DigestTree` / `DigestAuthorTree` 的语义与所有调用点。

## 结果（实测）

| 阶段 | 优化前 | 第一轮（loader 去重） | 本轮（记忆化） |
|---|---|---|---|
| disable 相位 | 9.9 – 15.5 s | 4.6 – 7.1 s | **0.89 s** |
| enable 相位 | 14.6 – 23.5 s | 6.7 – 12.1 s | **2.11 s** |
| 单次保存 | 24.5 – 39 s | 11.4 – 18 s | **≈3.0 s** |
| 后台 `RunRefresh` 占 40 s 窗口 | 27.39 s | — | **0.18 s** |
| 进程总 CPU | 2.03 核 | — | **0.64 核** |

关键步骤：`disable.03` 9894 → 771 ms、`enable.04` 4119 → 484 ms、`enable.05` 13304 → 1433 ms。
