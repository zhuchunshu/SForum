# AI 范围决定：MCP/Skills 不进 Core，streaming 不做

日期：2026-10-12
状态：accepted（运营者在方向讨论中确认）
相关：`decisions/2026-10-02-ai-provider-gateway.md`、
`plans/2026-10-12-ai-forward-ideas.md`

## 背景

AI 平台 M0-M2 已落地（网关、只读工具循环、`ai.tool` 贡献面、插件
`sforum.ai.complete` 命令）。2026-10-12 方向讨论中评估了三件事：是否引入
MCP、是否引入 Agent Skills、是否做流式输出。

## 决定

1. **MCP 不进 Core。** Host 已有等价面：`ai.tool` manifest 贡献 +
   ToolRegistry + trust/审查语义，覆盖「给工具循环注册工具」的全部需求；
   MCP 作为 client 消费的是外部工具生态，属运营者级可选增强，不是主干。
   若未来有真实需求，以独立 bridge 插件形态引入：把外部 MCP Server 翻译成
   已注册 ai.tool，接受既有 trust/配额/闸门语义检验，验证通过再谈正式贡献
   契约。MCP server 形态（SForum 暴露为 MCP Server 供本地 AI 客户端连接）
   为低优先级 optional 扩展：只读、复用现有 reader。
2. **Skills 不实现。** 指令包/操作手册职责已由 reply prompt + 各 purpose 的
   prompt 承担；等 M4 purpose registry 落地后，purpose 参数即等价物。
3. **Streaming 不做。** 当前唯一 AI 消费场景是异步回帖（River 队列），等待
   结果的只有 worker，不存在「人在同步请求里逐 token 等输出」的界面——
   论坛程序不是 IM。未来若出现人机实时界面（前台 AI 面板、实时编辑建议），
   在 `wire_openai` / `wire_anthropic` 的 SSE 转换层加流式支持，网关契约与
   闸门语义不动。

## 后果

- 未来会话遇到「要不要上 MCP/Skills/流式」时，以本记录为准，不再重新论证。
- 任何 MCP bridge 插件提案必须先声明 outbound authority 与 trust 面，走
  `ai.tool` 语义审查。
- 与 embeddings/多模态的排除（2026-10-02 决定）互不影响：bridge 插件若引入
  embedding 类工具，仍需先重开该决定。
