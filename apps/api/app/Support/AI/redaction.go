package ai

import (
	"encoding/json"
	"regexp"
	"strings"
)

// 出站脱敏：RedactionSettings 的实现。
//
// 为什么放在网关：所有带内容的出站（补全、工具循环的每一步、诊断、插件命令）都经过
// Gateway.Execute，只有在这一处做才不存在第二条漏掉的路径。
//
// 为什么在缓存键之前做：缓存键包含请求正文。先用原文算键、再用脱敏文出站，会让同一段
// 内容产生两个缓存条目，而且等于把原文留在了缓存键的输入里。
//
// 为什么用显式占位符而不是空串：模型需要知道「这里原本有内容」才会正确表述，而删成
// 空串会把两句话粘在一起，反而制造新的错误理解。
const (
	RedactedEmailPlaceholder = "[redacted-email]"
	RedactedPhonePlaceholder = "[redacted-phone]"
	RedactedIPPlaceholder    = "[redacted-ip]"
)

var (
	// 邮箱：必须有域名点号与 2 位以上 TLD，因此社区里的 @用户名 不会被命中。
	emailPattern = regexp.MustCompile(
		`[A-Za-z0-9._%+\-]+@[A-Za-z0-9](?:[A-Za-z0-9\-]*[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9\-]*[A-Za-z0-9])?)*\.[A-Za-z]{2,}`,
	)
	// IPv4：四段八位组且逐段限值，因此 1.2.3（版本号）与 10.0.0 不会被当成地址。
	ipv4Pattern = regexp.MustCompile(
		`\b(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(?:\.(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}\b`,
	)
	// IPv6：冒号分组形态，含压缩写法。用启发式而不是完整 RFC 正则——完整正则会长得
	// 无法审阅，而「三段以上冒号分组的十六进制」在自然语言里几乎不会误伤。
	ipv6Pattern = regexp.MustCompile(
		`\b(?:[0-9A-Fa-f]{1,4}:){3,}[0-9A-Fa-f]{0,4}\b|\b::(?:[0-9A-Fa-f]{1,4}:){0,6}[0-9A-Fa-f]{0,4}\b`,
	)
	// 电话：只认三种无歧义写法（+ 国际号、括号区号、中国大陆手机号）。刻意不认「一串
	// 裸数字」：论坛里订单号、时间戳、楼层号都是裸数字，抹掉它们会让提示词面目全非，
	// 而误伤造成的错答比漏掉更难排查。
	phonePattern = regexp.MustCompile(
		`\+\d{1,3}[\s\-]?\d[\d\s\-]{6,12}\d|\(\d{2,4}\)[\s\-]?\d[\d\s\-]{4,10}\d|\b1[3-9]\d{9}\b`,
	)
)

// RedactText 按设置脱敏一段文本。未启用、空串或没有命中时原样返回，因此调用方可以
// 用「返回值与入参是否相等」判断这次出站是否真的被改写。
func RedactText(settings RedactionSettings, text string) string {
	if !settings.Enabled || text == "" {
		return text
	}
	out := text
	if settings.RedactEmail {
		out = emailPattern.ReplaceAllString(out, RedactedEmailPlaceholder)
	}
	if settings.RedactIP {
		out = ipv4Pattern.ReplaceAllString(out, RedactedIPPlaceholder)
		out = ipv6Pattern.ReplaceAllString(out, RedactedIPPlaceholder)
	}
	if settings.RedactPhone {
		out = phonePattern.ReplaceAllString(out, RedactedPhonePlaceholder)
	}
	return out
}

// ApplyRedaction 返回脱敏后的请求副本，用于出站与缓存键。
//
// 它不修改入参持有的切片：调用方可能仍要用原文回显或记录，网关只保证「出站的那一份」
// 已脱敏。脱敏关闭时直接返回原值，零拷贝、零改写。
func ApplyRedaction(settings RedactionSettings, request CompletionRequest) CompletionRequest {
	if !settings.Enabled {
		return request
	}
	request.System = RedactText(settings, request.System)
	if len(request.Messages) == 0 {
		return request
	}
	messages := make([]Message, len(request.Messages))
	copy(messages, request.Messages)
	for index := range messages {
		message := messages[index]
		if len(message.Parts) > 0 {
			parts := make([]Part, len(message.Parts))
			copy(parts, message.Parts)
			for partIndex := range parts {
				parts[partIndex].Text = RedactText(settings, parts[partIndex].Text)
			}
			message.Parts = parts
		}
		if len(message.ToolCalls) > 0 {
			calls := make([]ToolCall, len(message.ToolCalls))
			copy(calls, message.ToolCalls)
			for callIndex := range calls {
				calls[callIndex].Arguments = redactToolArguments(settings, calls[callIndex].Arguments)
			}
			message.ToolCalls = calls
		}
		messages[index] = message
	}
	request.Messages = messages
	return request
}

// redactToolArguments 只改 JSON 里的字符串值：工具参数是一份 JSON 文档，直接按文本
// 做正则替换可能把引号或结构改坏，下一步模型就会收到一段非法 JSON。解析失败时退回文本
// 脱敏——那种内容本来就不合法，不会再损失什么。
func redactToolArguments(settings RedactionSettings, arguments string) string {
	trimmed := strings.TrimSpace(arguments)
	if trimmed == "" {
		return arguments
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return RedactText(settings, arguments)
	}
	redacted, changed := redactJSONValue(settings, decoded)
	if !changed {
		return arguments
	}
	payload, err := json.Marshal(redacted)
	if err != nil {
		return RedactText(settings, arguments)
	}
	return string(payload)
}

func redactJSONValue(settings RedactionSettings, value any) (any, bool) {
	switch typed := value.(type) {
	case string:
		replaced := RedactText(settings, typed)
		if replaced == typed {
			return typed, false
		}
		return replaced, true
	case []any:
		changed := false
		for index := range typed {
			item, itemChanged := redactJSONValue(settings, typed[index])
			if itemChanged {
				typed[index] = item
				changed = true
			}
		}
		return typed, changed
	case map[string]any:
		changed := false
		for key, item := range typed {
			redacted, itemChanged := redactJSONValue(settings, item)
			if itemChanged {
				typed[key] = redacted
				changed = true
			}
		}
		return typed, changed
	default:
		return value, false
	}
}
