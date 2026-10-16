# Decision: 移除模型身份注入，出站 system 只由调用方决定

## Status

Accepted（2026-10-16）。取代 `2026-10-15-ai-reasoning-separation-and-model-identity.md`
的第 2、3 条（身份声明由网关注入；公开性可配置且缺省公开）。该决策的第 1、4、5 条
（推理正文与回答分离、回复预算跟随 profile、确定性空回复不重试）继续有效。

## Context

2026-10-15 引入的身份注入让模型每次出站都能拿到一段宿主写下的声明：

```
【你的身份】你由本站的 AI 网关调用，本次使用的模型是 abc（glm-5.3-flash，openai-chat 协议）。
成员问「你是哪个模型」「谁提供的」时，按这个标识如实回答……
```

它的初衷是消除「模型不知道自己是谁于是乱猜」。实际效果是把后台配置事实送进了会话：
成员问一句「你是哪个模型」，助手就会念出 profile 的 `Label`（供应商标识）、
`Model`（底层模型名）与 `Protocol`（协议名），这些全部来自站点配置而不是模型本身。
同一段声明还告诉模型「你由本站的 AI 网关调用」，等于把部署结构复述给提问者。

运营者要求的语义与此相反：**站内 AI 回答自身身份时，内容只能来自模型自身**——训练
数据里的东西与模型正常返回的东西。任何由宿主追加的事实、开关或口径都属于要移除的
注入，而不是要调参的配置项。

## Decision

1. **网关不再向 system 追加任何文本。** `Gateway.Execute` 只保留出站脱敏，随后原样
   发出调用方给出的 system。删除 `Support/AI/identity.go`（`IdentitySettings` /
   `ApplyIdentity` / `IdentityAppendix` / `IdentityAppendixVersion`）。
2. **不留配置开关。** `settings.identity` 从 Go `Settings`、REST `updateSettingsRequest`、
   OpenAPI（`AIIdentitySettings` 与两处 `$ref`）、控制台 `SFAdminAIIdentityBlock` 与
   双语文案中整体删除。一个「打开就会把后台配置送进会话」的开关是可用性陷阱，保留它
   等于把这个行为当作可选项继续维护。
3. **删除走目录账本的合法路径。** 控制台组件身份
   `core.component.admin.sf_admin_ai_identity_block` 在 `catalog-identities.json` 转为
   `state: "retired"`，并向 `catalog-retired-identities.json` 与
   `catalogs/ui-retired-identities.json` 追加同一条墓碑；随后由
   `node scripts/v3-catalog/generate.mjs` 的产物（`ui-surfaces.*`、
   `core_catalog_gen.go`）确认无漂移。
4. **提示词版本递增。** `forum-reply@6`、`forum-reply-triage@3`、`diagnose@3`：版本号是
   历史执行记录与「产出它的那版实际生效系统指令」之间唯一的对应关系，而这次三个用途
   生效的 system 都变了（尽管提示词正文一字未动）。

## Consequences

- 成员问「你是哪个模型」时，答案完全由模型自身决定：可能准确、可能含糊、也可能自称
  别的产品。这是本决策有意接受的取舍——宿主不再替模型回答一个它本来不知道的事实。
  需要「站点自称」的运营者应当把它写进自己配置的系统提示词（那是运营者的正文，不是
  宿主注入的事实），或干脆不承诺回答该问题。
- 出站 system 的字节内容从此只由调用方决定，因此 `CacheKey` 摘要里的 `request.System`
  稳定可预测；键同时带 `profile.ID` / `profile.Model` / `revision`，移除注入不会造成
  跨供应商串答案。旧缓存因 system 变化自然失效，无需显式清理。
- `MaxSystemBytes`（100 KB）校验之后不再有宿主追加文本，原先「注入可能把贴着上限的
  请求推过界」的已知边界随之消失。
- 存储兼容：`ai_settings.document` 里残留的 `identity` 字段被 `decodeSettingsDocument`
  的普通 `json.Unmarshal` 忽略，下次保存自然清除，无需 Goose 迁移。
- 回归守卫：Go 侧 `TestGatewaySendsCallerSystemUnchanged`（哨兵 profile，断言出站
  system 与调用方逐字节一致）与 `TestGatewayKeepsEmptySystemEmpty`；Web 侧
  `tests/admin/adminAINoModelIdentity.test.ts`（契约无身份 schema、控制台无开关、
  双语键不存在、历史脏值在比较时被剥离）。
- 已知非本次范围：`AISettings` 的 OpenAPI 镜像里 `reply:` 键重复出现两次（既有 YAML
  重复映射键），本次不动，避免把无关清理混进这次收敛。
