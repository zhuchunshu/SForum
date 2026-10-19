# AI 会话状态：失效必须落到存储，不只是注入

Date: 2026-10-19
Status: accepted
Task: B5（`2026-10-19-ai-observability-and-capabilities.md`）

## 决定

`TopicState` 在每一轮写入时，以**本轮按三重失效裁决后仍然有效的事实**作为合并基底
（`survivingFacts`），而不是以上一轮读到的原始文档作为基底。

## 背景

B5 的三重失效里，第 1、2 条是整份失效（提示词版本不符、超过 14 天），第 3 条是
**逐条剔除**（被引用的楼层被删除或转为不可见）。任务书对它只有一句要求：

> 引用的楼层被删除或转为不可见 → 该条 `VerifiedFact` 在下次读取时剔除（不是整份失效）。

这句话只规定了「注入时不能出现」，没规定「存储里要不要清掉」。实现时很自然会写成：

```go
// 错的那一版
state.Effective(...)            // 注入用：已剔除失效条目 → 提示词干净
state.VerifiedFacts = MergeFacts(previous.VerifiedFacts, added)  // 写回用：原始文档作基底
```

## 后果（实测出来的）

被剔除的事实只从**提示词**里消失，从未从 **JSONB 文档**里消失。于是下一轮：

1. `Load` 读回包含失效楼层的文档；
2. `Effective` 再次发现它失效、再次剔除、再次记一条 `dropped=1, reason=reference_invalid`；
3. `Save` 又把它原样写回去。

外层表现：**状态永远干净不了**。日志里每一轮都记 `dropped=1`（看起来「正在生效」），
而库里那条依据一直都在。运维按 `reason=reference_invalid` 的计数排查看不到问题，
因为它每轮都发生一次、数值恒定。

更糟的是它的隐蔽性：这个缺陷**在单元测试里看不出来**，因为 fake 只断言「注入的
事实对不对」。它被 `TestGenerateDropsOnlyTheInvalidatedFact` 的第二半抓出来——那条
断言检查的是**写回文档**里只剩有效事实。

## 为什么不能用「整份删」绕过

把「有任一事实失效」升级成「整份状态失效」也能让存储干净，但它违反任务书
易错点 5 的另一半：

> 失败事实剔除是逐条，不整份删——整份删会让状态在一次可见性变更后全部丢失。

一次楼层隐藏就让整份记忆归零，等于把「有价值的部分记忆」也一起惩罚了。

## 实现约束（后来者请保持）

1. `Generator.saveTopicState` 的 `survivingFacts` 参数是 `Effective` 的返回值，
   **不得**替换成 `previous.VerifiedFacts`。
2. 整份失效（`prompt_version` / `stale`）时基底整体清空：这两条的理由是「旧事实
   按旧措辞采集或已过期」，把它们合并回去会让失效规则形同虚设。
3. 读取失败（`unreadable`）时 `Revision=0` 走 insert-only，**不得**退化成
   `UPDATE`——「宁可这一轮不写，也不覆盖别人」是精确行为，不是缺陷。
4. 回归守卫：`TestGenerateDropsOnlyTheInvalidatedFact`（写回文档只留有效事实）、
   `TestGenerateInvalidatesStateWhenPromptVersionMoves`（整份失效不得把旧事实当
   基底合并回来）。

## 与「读取容错」的边界

`Load` 遇到坏 `document` 时按「无状态」处理，但**不删行**：坏数据是可观测事实，
删掉它等于掩盖一次数据损坏。重建走 insert-only；如果与既有行冲突则放弃写入
（`ON CONFLICT DO NOTHING`）——那份坏文档因此会留在库里直到有人手工处理，这是
有意的：自动「修复」别人的数据比留着一条可观测的坏行更危险。

## 相关

- 任务书 B5 易错点 5、记忆连续性 · 层 2 第 1/2/3/4 条。
- `apps/api/app/Models/AIReply/topic_state.go`（`Effective` 的注释）。
- `apps/api/app/Models/AIReply/topic_state_store.go`（`Save` 的两种路径）。
