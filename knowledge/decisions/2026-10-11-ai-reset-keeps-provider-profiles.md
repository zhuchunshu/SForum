# Decision: AI Settings Reset Keeps Provider Profiles

## Status

Accepted

## Date

2026-10-11

## Context

`Service.ResetSettings` 原本直接写入 `RecommendedSettings()`，把整份文档替换成推荐
默认。这带来一个状态错位：Secret Store 中的密钥不受重置影响（密钥不在配置文档里），
但 `Profiles`（含 `APIKeyRef`）被清空了。于是重置后界面显示所有供应商「未配置」，
而凭证实际仍然存在——运营者要么以为密钥丢了去重填，要么看着「未配置」不敢启用。

同时，供应商配置（base_url、model、单价）是运营者填过的资产，一键重置把它们清掉
代价过高。

## Decision

- 重置保留 `Profiles` 与 `CostClassProfiles`，只把策略字段（gates、redaction、
  autoAction、reply prompt、purposeFailurePosture、enabled）恢复为推荐值。
- 没有任何已存 profiles 时（首次接入）退回完整 `RecommendedSettings()`，保证面板
  不是空的。
- Toast 文案沿用 `admin.ai.resetDone`：「已恢复推荐默认值。密钥不会被清除。」

## Consequences

- 重置的承诺变成明确的「恢复推荐策略」，而不是「清空我的配置」——符合仓库的
  Beginner-Friendly 原则：新手点重置是为了兜底，不该付出重填配置的代价。
- 成本绑定必须跟着保留的 profiles 走，否则会指向不存在的供应商（后端
  `Validate` 会拒绝）。
- 若运营者真的想清掉自定义供应商，需要逐个删除而不是重置。当前 UI 没有删除入口，
  这是已知缺口。

## Revisit when

若控制台增加「删除供应商」动作，重置与删除的语义边界要重新对齐，并在 UI 上区分。
