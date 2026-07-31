# 路线图

[← 中文文档首页](./README.md)

状态摘要（2026-07）：核心论坛、身份、后台、附件、邮件通知、审核和默认站点搜索已落地。扩展平台 V3 的 P0–P12 **阶段清单**已完成，但跨阶段生产重接线仍未关闭；这不等于扩展平台已经达到稳定版生产承诺。下列为**方向性**列表，细节以 `knowledge/plans/` 与代码为准。

在生产重接线 M3/M5/M6/M7 关闭前，多节点 RuntimeRollout、Marketplace / Privacy 运营入口、完整兼容矩阵与 commerce Dispatcher 证明都属于预览或未支持范围。当前发行必须继续标记为 prerelease，不应描述为第一个公开稳定版。

## 已完成（精简）

- 基础脚手架与 Compose 开发/生产路径  
- 身份 / RBAC / 会话 / 账户安全  
- 论坛读写、分类标签、策略执行  
- 运行时选项、个性化、SEO 基础  
- 附件与存储槽位  
- 邮件框架 + 内置 SMTP 插件、站内通知  
- 扩展 Manifest V3、信任、生命周期、Page Registry 主题基础；运营/多节点能力仍有下述生产残项
- 默认 PostgreSQL 站点搜索；Meili 可选插件  

## 进行中 / 近期产品

| 方向 | 说明 | 参考 |
| --- | --- | --- |
| 参与循环 | 浏览量累加、点赞/反应、收藏等 | `knowledge/plans/2026-07-12-iteration-a-engagement-loop.md` |
| 设置丰富度 | 更多运营策略面与分组 | `knowledge/plans/2026-07-12-admin-settings-richness.md` |
| 扩展密度 | 更多贡献点与服务槽位打磨 | `knowledge/plans/2026-07-12-extension-surface-density.md` |
| V3 生产重接线 | RuntimeRollout、Marketplace/Privacy 消费端、CompatFarm 与 commerce Dispatcher 残项 | `knowledge/plans/2026-07-22-v3-production-rewire-honesty-remediation.md` |
| V3 P13 LTS | 协议 v1 / 兼容路径删除（到期 + 零 shim） | `docs/extensions/v3/p13-migration-and-lts.md` |

## 中期候选

- 分类级 ACL  
- 支付中性框架 + 插件网关  
- 更完整的资料关注/私信等社区循环  
- 性能容量证明与多节点运营手册  

## 文档自身

- 本目录提供中英使用与开发手册  
- 扩展契约仍在 `docs/extensions/`  
- 历史设计稿在 `docs/archive/`  

归档的早期里程碑原文：`docs/archive/legacy-root/roadmap.md`。
