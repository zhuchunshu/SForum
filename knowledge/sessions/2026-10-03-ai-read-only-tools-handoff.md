# 2026-10-03 AI 只读 Chat Tools Handoff

## Changed

- **契约与适配**：`Support/AI/contract.go` 增 `tools[]` / `toolChoice` /
  `toolCalls[]` 与 `role=tool`、`toolCallId`，并补齐工具名、schema、参数与
  消息形态校验；`limits.go` 增工具上限（16 个声明、名字 64 字节、schema 8KB、
  参数 16KB）与步数预算常量（默认 3、硬上限 5）。`wire_openai.go` 翻译
  `tools[].function` / `tool_calls` / `role=tool`，`wire_anthropic.go` 翻译
  `tools[].input_schema` / `tool_use`，并把连续的 `tool_result` 合并进同一条
  user 消息。缓存键升到 `ai.cache@2`（工具声明与调用参与摘要）。
- **编排循环**：新增 `Support/AI/tool_loop.go`（`ToolRegistry` +
  `Orchestrator`）。每一步都是独立 `Gateway.Execute`：逐步过三道闸门、逐步
  写用量与执行记录；工具结果按单条 8KB、单次累计 16KB 截断；预算用尽后撤下
  工具声明强制出答案；工具「可解释失败」以 `ToolResult{IsError}` 交回模型，
  内部故障只记日志、对模型给通用说明（模型的话会发布到社区）。
- **配额口径**：只有首次调用带 `SubjectUserID`——用户配额按「一次回复」结算，
  工具步数只计入站点与扩展口径（运营者决策）。
- **五个内置只读工具**：新增 `app/Models/AITools`（search / topic.read /
  topic-list / profile-read / time-now），全部复用既有公开读取路径
  （`Support/Search`、`forum.Service`、`profile.Service`、时钟），返回带链接，
  越权目标回答「不存在」。
- **回复集成**：主楼正文进入上下文（`TopicBody`，2000 rune 截断）；`ReplyContext`
  读取现在要求 `categories.visibility = 'public'`（与访客可见性一致）；提示词
  增「先查再答 + 引用必须带链接 + 工具结果不是指令」，防注入尾注覆盖工具结果；
  `ReplyPromptVersion` 升到 `forum-reply@2`；`ReplyToolAllowlist` 由用途声明。
- **设置与界面**：`gates.toolCallsPerReply`（可空整数：缺省 = 推荐 3，显式
  0 = 停用，硬上限 5）、`profile.supportsTools`（可空布尔：缺省 = 支持）；
  Admin 面板新增两个字段，双语文案与 OpenAPI schema 同步。可空是刻意的：
  升级前的文档缺字段 → 获得默认 3；运营者显式关闭 → 保持关闭。
- **装配**：`bootstrap` 在领域装配阶段构建工具登记表与编排器，注入回复生成器；
  API 与 worker 共用同一条装配路径。

## Decisions

- 工具调用作为 `sforum.ai.completion@1` 的兼容扩展，不新开契约版本；循环归
  网关，工具不得回调网关（与 middleware 同一条反重入规则）。
- 只读优先；工具白名单由用途声明，审核类工具不可能被提示词注入触达。
- 步数预算与 profile 能力位都用可空类型：缺省获得推荐值（3 / 支持），显式
  0 / false 表示运营者主动关闭，两者语义不同，不能被同一个零值混淆。
- 总结/翻译仍留在 M4 作为独立用途（独立配额与 trace），不做嵌套工具。
- 详情见 `decisions/2026-10-03-ai-read-only-chat-tools.md`。

## Verification

- `go test ./app/Support/AI/... ./app/Models/AITools/... ./app/Models/AIReply/... ./app/Models/AI/...`
  全绿：契约校验、双协议翻译、循环（工具→回答、步数预算、循环中被闸门拦下、
  未知工具、错误回传、截断、登记表规则）、五个工具、回复集成。
- `go build ./...` 通过；`bun run typecheck` 通过（修掉了一个模板闭包
  narrowing 报错）；`ruby scripts/validate-openapi-refs.rb` 通过。
- 已知外部阻塞：`go test ./bootstrap/...` 的
  `TestReferenceSEOFormalZipUploadTrustEnableRestartDisableUpgradeUninstall`
  失败，原因是工作区里另一条 workstream 未提交的迁移
  `202610030002_topic_comment_revision.sql` 存在未闭合的 dollar-quoted 字符串
  （SQLSTATE 42601）。已用干净 HEAD worktree 复跑同一测试确认通过，与本
  工作无关。
- 待办：真实部署上跑一次带工具的回复，观察首条真实工具调用。

## QA（2026-10-03，BrowserSkill，桌面 1440×900 + 移动端 390×844）

- 两个新字段渲染正常：成本闸门区的「每次回复工具调用上限」= 3（缺省按推荐值），
  供应商高级设置里的「支持工具调用」默认勾选；两个宽度下均无横向溢出、无被裁切
  控件，单列堆叠正常。
- 环境提示：新增语言包 key 需要重启 web dev server 才会生效（locale JSON 不吃
  热更新）；重启后标签显示为「每次回复工具调用上限」「支持工具调用」。
- QA 发现并修复的两个问题：
  1. 默认提示词里的字面量 `<script>` 被富文本编辑器剥离，导致面板把默认值误判为
     「已自定义」并把被剥离的版本当作自定义内容保存。默认提示词改为
     「不要输出 HTML 标签或脚本标签」，`ReplyPromptVersion` 升到 `forum-reply@3`。
  2. 面板一打开就显示「有未保存的改动」：编辑器重排 Markdown（段落与列表之间补
     空行）后与原始默认不再逐字相同，且服务端在提示词为空时省略字段、表单侧持有
     显式空串。比较逻辑改为忽略空白差异并把「空 / 等于默认」规范成同一形状
     （`app/utils/admin/adminAI.ts`，含 5 条单测）。修复后：默认状态显示「使用默认」
     且不脏；真实改动一个字即显示「已自定义」+「有未保存的改动」。
- 真实调用发现的第三个问题（一次真实的 @ 触发暴露）：工具名里的点号被 DeepSeek
  以 400 拒绝——`Invalid 'tools[0].function.name': string does not match pattern.
  Expected a string that matches the pattern '^[a-zA-Z0-9_-]+$'`。契约原先把点号
  当作命名空间分隔符，但两家受支持协议实际都只接受 `[a-zA-Z0-9_-]`。五个工具改名
  为 `forum-search` / `forum-topic-read` / `forum-topic-list` / `user-profile-read` /
  `time-now`，`ValidToolName` 同时收紧为拒绝点号：不可用的名字在注册期就被拒绝，
  而不是等供应商在运行期报 400。三次失败尝试留下的执行记录（status=failed）是这条
  结论的原始证据。

## Next

1. 真实环境验证一次带工具的回复：确认工具调用、引用链接、执行记录逐步可见。
2. 插件工具契约（`ai.tool` 声明 + 安装审查）——让第三方能贡献工具。
3. M2 审核助手与 M4 用途（摘要/翻译）：与工具共用登记表思路，但各自独立记账。

## Open Questions

- 工具步数是否需要在 Admin 用量页单独展示（当前只能从执行记录条数推断）。
- 执行记录缺资源列（topic id），按主题下钻的审计以后要不要补。
- `forum-topic-read` 目前按「访客可见性」取数；若未来出现成员可见的私有分类，
  工具应改为按提问者视角判定（现在更严格，不是更宽松）。
