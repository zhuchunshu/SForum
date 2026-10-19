package aireply

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// AI 产物里的出处标记形如 [[cite:topic:42]] / [[cite:comment:913]]。
//
// 为什么是这套语法：
//   - 双方括号加固定前缀不会与 Markdown 语法或社区里常见的方括号链接撞车；
//   - 只带一个整数，模型不容易写错（比起来「请按 JSON 输出引用」要它维持结构，
//     一次格式漂移就整段作废）；
//   - 正文里剥掉它之后仍然是一段人话，因此模型不遵守格式时降级为「无引用」
//     而不是让整次回复失败（任务书易错点 1）。
const (
	citationKindTopic   = "topic"
	citationKindComment = "comment"
)

// citationMarkerPattern 匹配一条出处标记。
//
// 三类容错，都来自「模型不会逐字节照抄格式」这一事实：
//   - 允许 cite 与冒号之间、冒号前后出现空白（[[cite: topic: 42]]）；
//   - id 允许带前导零、上限 19 位（int64 上限是 19 位十进制）；
//   - kind 大小写不敏感（Topic / COMMENT）。
//
// 不允许的是「负号」「多段 id」这类语义上无意义的东西：那不是格式漂移，那是
// 模型在编另一个东西，宁可当普通文本留着。
var citationMarkerPattern = regexp.MustCompile(`(?i)\[\[\s*cite\s*:\s*(topic|comment)\s*:\s*(\d{1,19})\s*\]\]`)

const (
	// MaxCitationsPerReply 是一次回复最多保留的引用条数。上限存在的理由是渲染：
	// 评论下面挂二十条依据已经是「证据链」而不是「依据两句」，再多的折叠块
	// 没人会展开，但它仍然要进库、要被前端渲染。
	MaxCitationsPerReply = 20
	// MaxCitationExcerptRunes 是单条引文的长度上限。引文是用来让人快速确认
	// 「依据是不是这条」的，不是用来复制整楼内容的；超出部分由链接补足。
	MaxCitationExcerptRunes = 240
	// MaxCitationMarkersScanned 是解析阶段最多消费的标记数。它防的不是渲染，
	// 是「模型刷了一屏标记」这种病态输出：先按出现顺序消费到这里为止，剩下的
	// 从正文里剥掉但不参与校验（不浪费查询，也不放大日志）。
	MaxCitationMarkersScanned = 64
)

// CitationRef 是解析产物：一条待校验的引用，只有指向，没有引文。
//
// 它是导出类型，因为 CitationVerifier 的实现方（含测试假件）必须能构造它。
// 字段含义与 Citation 一致：CommentID 非零时 TopicID 可缺（校验阶段会从库里
// 补回评论所属主题）。
type CitationRef struct {
	TopicID   int64
	CommentID int64
}

// IsZero 报告这条引用是否什么都没有指向：一个既无主题又无评论的引用是解析产物
// 的残渣，不该被校验、不该被渲染，也不该被计入「这次回答有依据」。
func (r CitationRef) IsZero() bool {
	return r.TopicID <= 0 && r.CommentID <= 0
}

// parseCitationMarkers 从模型正文里抽出出处标记，并返回剥离标记后的正文。
//
// 它只做纯文本工作，不碰数据库：解析必须可以在没有装配的单元测试里跑，而且
// 「格式不遵守」这个最常见的失败必须与「引用指向不存在的楼层」分开报告。
//
// 剥离策略：所有匹配到的标记都从正文里删掉（包括超出上限、无法解析的那些），
// 因为标记是给解析器的元数据，留在正文里就是一段 [[cite:topic:42]] 噪声。
// 但只有前 MaxCitationsMarkersScanned 条、且 id 能装进 int64 的，才进入返回的
// 引用列表。
func parseCitationMarkers(text string) (string, []CitationRef) {
	if !strings.Contains(text, "[[") {
		// 绝大多数回复没有引用标记：这条短路让常规路径零正则开销。
		return text, nil
	}
	matches := citationMarkerPattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text, nil
	}
	refs := make([]CitationRef, 0, len(matches))
	var builder strings.Builder
	builder.Grow(len(text))
	cursor := 0
	for index, match := range matches {
		// match 是 [start, end, kindStart, kindEnd, idStart, idEnd]。
		builder.WriteString(text[cursor:match[0]])
		cursor = match[1]
		if index >= MaxCitationMarkersScanned {
			continue
		}
		ref, ok := citationRefFromMatch(text, match)
		if !ok {
			continue
		}
		refs = append(refs, ref)
	}
	builder.WriteString(text[cursor:])
	return builder.String(), refs
}

// citationRefFromMatch 把一次正则命中翻译成引用。id 溢出 int64 时返回 false：
// 那不是一条引用，是一个贴着引用语法的长数字。
func citationRefFromMatch(text string, match []int) (CitationRef, bool) {
	kind := strings.ToLower(text[match[2]:match[3]])
	id, err := strconv.ParseInt(text[match[4]:match[5]], 10, 64)
	if err != nil || id <= 0 {
		return CitationRef{}, false
	}
	if kind == citationKindComment {
		// 评论引用此时只知道评论 id：它属于哪个主题要靠校验阶段从库里读回来。
		return CitationRef{CommentID: id}, true
	}
	return CitationRef{TopicID: id}, true
}

// normalizeCitationBody 清理剥离标记之后剩下的正文。
//
// 三件事，都是为了不产出「看起来有内容、其实是空的」评论：
//   - 去掉行尾因标记消失而留下的空白（"依据。 [[cite:x]]" → "依据。"）；
//   - 合并连续空行（模型常在标记前后各留一个换行）；
//   - 去首尾空白。
//
// 它**不**判断正文是否为空：那是调用方的决定（空正文要走 ErrEmptyReply 的既有
// 语义）。把两件事混在一起会让「只有标记」这种情形无法被单独测试。
//
// 注意它只在**确实剥掉了标记**时才被调用：没有标记的回复走原来的
// strings.TrimSpace，逐字节保持既有行为（连续空行是模型正常的排版，不该被
// 这次改动顺手抹平）。
func normalizeCitationBody(body string) string {
	lines := strings.Split(body, "\n")
	cleaned := make([]string, 0, len(lines))
	blank := 0
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			blank++
			// 最多保留一个空行：段落分隔有意义，连续空行只是排版噪声。
			if blank > 1 {
				continue
			}
			cleaned = append(cleaned, "")
			continue
		}
		blank = 0
		cleaned = append(cleaned, line)
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

// CitationVerifier 校验引用是否真实存在且对访客可见，并取回它的引文。
//
// 「可见」的口径与工具、回复上下文完全一致：主题 active/locked + 未删除 + 公开
// 分类，评论 active。任何一项不满足都丢弃——不区分「不存在」与「不可见」，因为
// 这条区分本身就是一种信息泄露（私密板块的主题存在与否不该由 AI 的引用列表
// 泄漏出去）。
//
// 它是生成器的可选能力：未装配时降级为「无引用」（见 Generator.verifyCitations）。
type CitationVerifier interface {
	Verify(ctx context.Context, refs []CitationRef) ([]Citation, error)
}

// ErrCitationsUnavailable 表示引用能力没有装配。它与「引用全部无效」是两件事：
// 后者是模型写错了（正常降级），前者是部署缺能力（值得记一条 Warn）。
var ErrCitationsUnavailable = errors.New("aireply: citation store is not wired")

// clipExcerpt 把引文裁到上限并保持合法 UTF-8。裁切按 rune，不按字节：中文引文
// 被从中间切断会变成一串乱码，而运营者要靠这段引文确认「依据是不是这条」。
func clipExcerpt(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= MaxCitationExcerptRunes {
		return value
	}
	return strings.TrimSpace(string(runes[:MaxCitationExcerptRunes])) + "…"
}

// dedupeCitations 按 (topicId, commentId) 去重并保持首次出现的顺序。
//
// 模型常在同一段话里重复标注同一处依据（「参见 42 楼……（42 楼）」）。重复的
// 引用在渲染上就是两个一样的链接，在库里就是两行同事实——两者都是噪声。
func dedupeCitations(items []Citation) []Citation {
	if len(items) <= 1 {
		return items
	}
	seen := make(map[CitationRef]struct{}, len(items))
	unique := make([]Citation, 0, len(items))
	for _, item := range items {
		key := CitationRef{TopicID: item.TopicID, CommentID: item.CommentID}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, item)
	}
	return unique
}

// citationPromptHint 是追加到提示词里的引用格式要求。
//
// 它刻意写得像「可选建议」而不是「硬性格式」：格式遵守是概率性的，把要求写成
// 命令只会让模型在没有站内依据时编一个标记出来凑格式——那正是引用校验要拦的
// 东西。这里明确写了「没有查证过就不要写标记」，让「零引用」成为一个正当结果。
//
// 注意：它改变模型读到的内容，因此 B4 落地时必须配套升 CoreReplyPromptVersion
// （见任务书偏离登记 D8：B2 = @7、B4 = @8）。
func citationPromptHint() string {
	var builder strings.Builder
	builder.WriteString("\n引用出处（可选）：\n")
	builder.WriteString("- 你引用了站内某个主题或某条评论时，在该句末尾标注出处：主题用 ")
	builder.WriteString("[[cite:topic:主题id]]，评论用 [[cite:comment:评论id]]。\n")
	builder.WriteString("- 标记里的 id 必须来自工具结果或上面的对话，不要自己编。\n")
	builder.WriteString("- 没有查证过的内容不要标注出处；没有依据时可以不写任何标记，这是正常结果。\n")
	builder.WriteString("- 标记会被系统从正文里移除并渲染成「依据」列表，不要在正文里解释这些标记。\n")
	return builder.String()
}

// 以下是给外部测试包（aireply_test）的薄钩子。它们是纯函数的再导出，不含逻辑：
// 公开它们是为了让「标记解析」「正文清理」这两条最容易被改坏的规则能被独立
// 测试，而不是只能通过 Generate 间接观察。
var (
	ParseCitationMarkers  = parseCitationMarkers
	NormalizeCitationBody = normalizeCitationBody
	ClipExcerpt           = clipExcerpt
	DedupeCitations       = dedupeCitations
	CitationPromptHint    = citationPromptHint
	CitationMarkerPattern = citationMarkerPattern
)
