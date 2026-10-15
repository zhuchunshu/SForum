package ai

import "strings"

// 内联思考的归一化。
//
// 有些网关不把思考放进 reasoning_content，而是把它和回答拼在同一个 content 里，
// 用标签分隔：`<think>…</think>真正的回答`。这类正文如果原样进入 Text，就会
// 以评论的形式发布出去——同一个故障的第二种入口。协议适配器在把正文交给上层之前
// 统一剥掉它，因此「Text 是回答」这条约定在两套协议上只有一处实现。
//
// 只识别这几个标签，不做通配：标签集合越宽，越可能吃掉正文里合法的尖括号内容。
var inlineReasoningTags = []string{"think", "thinking", "reasoning", "analysis"}

// stripInlineReasoning 移除正文开头的内联思考块。
//
// 两条规则覆盖真实返回形态：
//   - 成对的 `<tag>…</tag>`：反复剥掉开头的块，直到正文真正开始；
//   - 未闭合的 `<tag>…`：输出预算被思考耗尽时标签永远等不到闭合，此时从标签起
//     丢弃到结尾——那一段整体都是思考，不是回答。
//
// 标签只被当作「正文开头」的标记：出现在中间的 `<think>` 可能是成员提问里的字面
// 内容，不动它。
func stripInlineReasoning(text string) string {
	trimmed := strings.TrimSpace(text)
	for {
		tag, rest, matched := matchLeadingReasoningTag(trimmed)
		if !matched {
			return trimmed
		}
		closing := "</" + tag + ">"
		index := indexFold(rest, closing)
		if index < 0 {
			// 未闭合：剩下的全是思考，没有回答。
			return ""
		}
		trimmed = strings.TrimSpace(rest[index+len(closing):])
	}
}

// matchLeadingReasoningTag 报告文本是否以某个思考标签开头，并返回标签名与标签之后
// 的内容。大小写不敏感；带属性的开标签（`<think level="high">`）不识别——真实网关
// 不这么写，为它放宽匹配规则只会增加误伤正文的可能。
func matchLeadingReasoningTag(text string) (string, string, bool) {
	if !strings.HasPrefix(text, "<") {
		return "", "", false
	}
	for _, tag := range inlineReasoningTags {
		opening := "<" + tag + ">"
		if len(text) >= len(opening) && strings.EqualFold(text[:len(opening)], opening) {
			return tag, text[len(opening):], true
		}
	}
	return "", "", false
}

// indexFold 返回 needle 在 haystack 中首次出现的位置（ASCII 大小写不敏感），未命中
// 返回 -1。它按字节位置比较，不用 strings.ToLower 复制整串——对含非 ASCII 的正文，
// ToLower 可能改变字节长度，换算回来的下标会指错位置。
func indexFold(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i] != '<' {
			continue
		}
		if strings.EqualFold(haystack[i:i+len(needle)], needle) {
			return i
		}
	}
	return -1
}
