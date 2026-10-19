package ai

import "strings"

// RichContentPromptVersion 是「富内容围栏」这一段提示词自身的版本。
//
// 它不参与 CoreReplyPromptVersion 的编号（那是整份回复提示词的版本），但两者必须
// 一起动：改这一段就等于改变模型读到的内容，历史执行记录与缓存键都要跟着失效。
// 独立版本号让「契约变了、提示词忘了跟」在评审里一眼可见——真正的把关是
// reply_prompt_test.go 里读 contracts/fixtures/rich-content-v1.json 的一致性测试。
// rich-content@1：首版，与 rich-content-v1.json 的围栏语言、定界符与上限对齐。
const RichContentPromptVersion = "rich-content@1"

// richContentPromptLines 是富内容段的正文，按行拼装。
//
// 用行切片而不是一个大 raw string，是因为正文里必须出现 ```mermaid / ```chart
// 这两种围栏写法，而 Go 的 raw string 不能包含反引号。这里只做常量拼接：段内容
// 不读文件、不读配置、不随运行期状态变化——同一版二进制永远得到逐字相同的文本，
// 缓存键与 eval 指纹才可比。
var richContentPromptLines = []string{
	"富内容（宿主会把它渲染成图）：",
	"",
	"需要画图时用围栏代码块，不要用 HTML、SVG、图片链接或字符画代替：",
	"- 流程、层级、结构、时序、状态、时间线、脑图：用 ```mermaid 围栏，里面直接写",
	"  mermaid 语法，不要包 JSON。支持 mindmap / flowchart / sequenceDiagram /",
	"  stateDiagram-v2 / classDiagram / erDiagram / gantt / timeline / pie 等。",
	"- 数值对比或趋势：用 ```chart 围栏，内容是一段合法 JSON（双引号、无注释、无尾逗号）：",
	`  {"type":"bar","title":"标题","labels":["A","B"],"series":[{"name":"系列名","data":[1,2]}]}`,
	"  type 只能是 bar、line、area、pie、doughnut 之一；labels 1–60 项；series 1–6 组，",
	"  每组 data 的长度与 labels 相同。",
	"  数字只能来自工具结果或提问者给出的数据，绝不编造；拿不到数字就直接说明，不要画图。",
	"- 数学公式：行内用 $…$，独立成行用 $$…$$；单条公式不超过 2 KiB。",
	"",
	"限制：",
	"- 单个 mermaid 围栏不超过 12 KiB，单个 chart 围栏不超过 8 KiB（按源码字节计算）；",
	"  超限会退回源码显示，图不会出现。",
	"- 一个回答最多一张图（渲染层每篇最多 6 个围栏，那是硬上限，不是配额）。",
	"- 只在确实比文字更清楚时才画：简单问题、一两句话能说清的事不要配图。",
	"- 图里的文字用提问者的语言，与回答正文保持一致。",
	"- mermaid 源码里不要写 click、链接或 HTML 标签：渲染时会按安全策略剥掉。",
	"- 不要使用 [topic]、[user]、[login] 这类短代码语法：那是给人类引用用的，你无法",
	"  知道其中的 ID；要引用站内内容就给出工具返回的链接。",
}

// RichContentSystemPrompt 返回追加在基础提示词之后的富内容指令段。
//
// 它与 ReplyPromptSafetyAppendix 一样由宿主拥有：渲染能力是部署事实（前端装了哪
// 些渲染器、上限是多少），不是运营者的文案选择。运营者覆盖 SystemPrompt 时删不
// 掉这一段，否则「AI 会不会画图」会变成每个站点各自猜的配置项，而站点并没有任何
// 开关能真的关掉渲染——那只会让提示词与真实能力脱节。
func RichContentSystemPrompt() string {
	return strings.Join(richContentPromptLines, "\n")
}
