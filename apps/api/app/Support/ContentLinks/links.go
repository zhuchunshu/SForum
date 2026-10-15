// Package contentlinks 是宿主「正文链接呈现策略」的唯一实现点。
//
// 契约（帖子/评论正文、运营 Markdown）：
//   - 正文里的链接一律补 target="_blank"，并保证 rel 含 noopener/noreferrer/nofollow。
//     bluemonday 已写入 nofollow/noreferrer，这里只补齐缺失项并去重，保留原顺序。
//   - 绝对 http(s) 链接改写成站内中转页 /outbound?u=<base64url>。
//     「是否启用中转」「免中转域名白名单」一律在点击时由中转页判定，因此存储 HTML
//     不随运营配置漂移，历史内容无需重渲染。
//   - 只重写命中锚点的 <a ...> 起始标签，文档其余字节原样保留：不做整树重排，
//     避免存量 HTML 形态与既有渲染断言漂移。
//
// 明确不处理的形态：无 href、页内锚点 (#...)、mailto:/tel:、图片灯箱触发器
// (a[data-sforum-image-viewer="1"])、协议相对 (//host/path，无法在不改变语义的前提下
// 归一协议，仅补 target/rel)。
package contentlinks

import (
	"encoding/base64"
	"html"
	"net/url"
	"strings"
)

const (
	// InterstitialPath 是站外链接中转页的固定路径。它是存储态契约：
	// 已渲染的历史 HTML 会内嵌该前缀，改名必须保留永久别名。
	InterstitialPath = "/outbound"
	// TargetQueryKey 是中转页携带目标地址的查询参数名。
	TargetQueryKey = "u"

	blankTarget = "_blank"
	// canonicalRelValue 与 bluemonday require-no-follow/no-referrer 策略的规范化结果一致：
	// 非白名单令牌（ugc/external 等）由净化器丢弃，顺序固定为 nofollow → noreferrer → noopener。
	canonicalRelValue = "nofollow noreferrer noopener"
)

type hrefKind int

const (
	hrefUnmanaged        hrefKind = iota // 不参与策略（无 href / 锚点 / mailto / tel / 非 http 协议）
	hrefLocal                            // 站内相对路径
	hrefAbsolute                         // 绝对 http(s)，可改写为中转链接
	hrefProtocolRelative                 // //host/path
)

// Apply 对一段已净化的 HTML 应用链接呈现策略。
// 输入必须是净化（bluemonday）之后的可信结构；本函数只做属性层面的收敛改写。
func Apply(value string) string {
	if value == "" || !strings.Contains(value, "<") {
		return value
	}
	var builder strings.Builder
	builder.Grow(len(value) + 64)
	cursor := 0
	for {
		start, end, ok := nextAnchorTag(value, cursor)
		if !ok {
			break
		}
		builder.WriteString(value[cursor:start])
		builder.WriteString(rewriteAnchorTag(value[start:end]))
		cursor = end
	}
	if cursor == 0 {
		return value
	}
	builder.WriteString(value[cursor:])
	return builder.String()
}

// InterstitialHref 构造中转页地址；target 已是解码后的绝对 http(s) 地址。
func InterstitialHref(target string) string {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(target))
	return InterstitialPath + "?" + TargetQueryKey + "=" + encoded
}

// IsOffsiteHref 判定 href 是否为站外绝对 http(s) 地址（供测试与前端契约对照）。
func IsOffsiteHref(href string) bool {
	return classifyHref(href) == hrefAbsolute
}

// nextAnchorTag 返回下一个 <a ...> 起始标签的 [start,end) 区间。
// 扫描过程跟踪引号，属性值里出现 '>' 时不会提前截断。
func nextAnchorTag(value string, from int) (int, int, bool) {
	for index := from; index+1 < len(value); index++ {
		if value[index] != '<' {
			continue
		}
		// 大小写不敏感匹配 "<a"，且后一个字符必须是空白/'>'/'/'，避免命中 <abbr>。
		if lowerASCII(value[index+1]) != 'a' {
			continue
		}
		if index+2 < len(value) {
			switch next := value[index+2]; {
			case next == '>', next == '/', isSpace(next):
			default:
				continue
			}
		}
		quote := byte(0)
		for cursor := index + 2; cursor < len(value); cursor++ {
			char := value[cursor]
			if quote != 0 {
				if char == quote {
					quote = 0
				}
				continue
			}
			switch char {
			case '"', '\'':
				quote = char
			case '>':
				return index, cursor + 1, true
			}
		}
		return index, len(value), true
	}
	return 0, 0, false
}

type attribute struct {
	name  string
	value string
	// hasValue 为 false 表示布尔属性（如 disabled）。
	hasValue bool
}

// rewriteAnchorTag 只重写单个 <a ...> 起始标签，保持属性顺序与写法。
func rewriteAnchorTag(tag string) string {
	attrs, ok := parseAttributes(tag)
	if !ok {
		return tag
	}
	hrefIndex := -1
	interstitialOnly := false
	imageViewer := false
	for index, attr := range attrs {
		switch strings.ToLower(attr.name) {
		case "href":
			if hrefIndex < 0 {
				hrefIndex = index
			}
		case "data-sforum-image-viewer":
			if attr.hasValue && strings.TrimSpace(attr.value) == "1" {
				imageViewer = true
			}
		}
	}
	if hrefIndex < 0 || imageViewer {
		return tag
	}

	href := strings.TrimSpace(html.UnescapeString(attrs[hrefIndex].value))
	// 已经写过中转前缀的锚点保持幂等：只补 target/rel，不二次编码。
	if isInterstitialHref(href) {
		interstitialOnly = true
	}
	switch classifyHref(href) {
	case hrefUnmanaged:
		return tag
	case hrefAbsolute:
		if !interstitialOnly {
			attrs[hrefIndex].value = InterstitialHref(href)
		}
	}

	// bluemonday 的正文策略会把 rel 规范成固定令牌集并落到标签末尾，target 紧随其前。
	// 这里直接产出同一形态，保证「再净化一次结果不变」的固定点性质——
	// ContentRegistry 的 SSR 段落校验（Sanitize(x) == x）与 H3 纵深防御都依赖它。
	normalized := make([]attribute, 0, len(attrs)+2)
	for _, attr := range attrs {
		switch strings.ToLower(attr.name) {
		case "target", "rel":
			continue
		}
		normalized = append(normalized, attr)
	}
	normalized = append(normalized,
		attribute{name: "target", value: blankTarget, hasValue: true},
		attribute{name: "rel", value: canonicalRelValue, hasValue: true},
	)
	return renderAttributes(tag, normalized)
}

// parseAttributes 解析起始标签内的属性列表；标签本身不合法时返回 false。
func parseAttributes(tag string) ([]attribute, bool) {
	if len(tag) < 3 || tag[0] != '<' {
		return nil, false
	}
	attrs := make([]attribute, 0, 4)
	cursor := 2
	// 跳过标签名剩余字符（a）、可选的 '/' 与空白。
	for cursor < len(tag) && isTagNameChar(tag[cursor]) {
		cursor++
	}
	for cursor < len(tag) {
		for cursor < len(tag) && (isSpace(tag[cursor]) || tag[cursor] == '/') {
			cursor++
		}
		if cursor >= len(tag) || tag[cursor] == '>' {
			break
		}
		nameStart := cursor
		for cursor < len(tag) && !isSpace(tag[cursor]) && tag[cursor] != '=' && tag[cursor] != '>' && tag[cursor] != '/' {
			cursor++
		}
		name := tag[nameStart:cursor]
		for cursor < len(tag) && isSpace(tag[cursor]) {
			cursor++
		}
		if cursor >= len(tag) || tag[cursor] != '=' {
			if name != "" {
				attrs = append(attrs, attribute{name: name})
			}
			continue
		}
		cursor++ // '='
		for cursor < len(tag) && isSpace(tag[cursor]) {
			cursor++
		}
		if cursor >= len(tag) {
			return nil, false
		}
		var value string
		switch quote := tag[cursor]; quote {
		case '"', '\'':
			cursor++
			valueStart := cursor
			for cursor < len(tag) && tag[cursor] != quote {
				cursor++
			}
			if cursor >= len(tag) {
				return nil, false
			}
			value = tag[valueStart:cursor]
			cursor++
		default:
			valueStart := cursor
			for cursor < len(tag) && !isSpace(tag[cursor]) && tag[cursor] != '>' {
				cursor++
			}
			value = tag[valueStart:cursor]
		}
		if name == "" {
			continue
		}
		attrs = append(attrs, attribute{name: name, value: value, hasValue: true})
	}
	return attrs, true
}

// renderAttributes 以 key="value" 形式重写起始标签；值按 HTML 属性规则转义。
func renderAttributes(tag string, attrs []attribute) string {
	var builder strings.Builder
	builder.Grow(len(tag) + 32)
	builder.WriteString("<a")
	for _, attr := range attrs {
		builder.WriteByte(' ')
		builder.WriteString(attr.name)
		if !attr.hasValue {
			continue
		}
		builder.WriteString(`="`)
		builder.WriteString(escapeAttributeValue(attr.value))
		builder.WriteByte('"')
	}
	builder.WriteString(">")
	return builder.String()
}

func escapeAttributeValue(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, char := range value {
		switch char {
		case '&':
			builder.WriteString("&amp;")
		case '"':
			builder.WriteString("&quot;")
		case '<':
			builder.WriteString("&lt;")
		default:
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func classifyHref(href string) hrefKind {
	if href == "" {
		return hrefUnmanaged
	}
	switch {
	case strings.HasPrefix(href, "#"):
		return hrefUnmanaged
	case strings.HasPrefix(href, "//"):
		return hrefProtocolRelative
	}
	parsed, err := url.Parse(href)
	if err != nil || parsed == nil {
		return hrefUnmanaged
	}
	switch scheme := strings.ToLower(parsed.Scheme); scheme {
	case "http", "https":
		if parsed.Host == "" {
			return hrefUnmanaged
		}
		return hrefAbsolute
	case "":
		// 无 scheme 即相对引用（/t/1、./a、a/b、?q=1、裸域名都按站内路径处理，
		// 与浏览器解析一致）。
		if parsed.Host != "" {
			return hrefProtocolRelative
		}
		return hrefLocal
	default:
		// mailto:/tel:/javascript:/data: 等一律不参与策略。
		return hrefUnmanaged
	}
}

func isInterstitialHref(href string) bool {
	prefix := InterstitialPath + "?" + TargetQueryKey + "="
	return strings.HasPrefix(strings.ToLower(href), prefix)
}

func lowerASCII(char byte) byte {
	if char >= 'A' && char <= 'Z' {
		return char + ('a' - 'A')
	}
	return char
}

func isSpace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\n' || char == '\r' || char == '\f'
}

func isTagNameChar(char byte) bool {
	return !isSpace(char) && char != '>' && char != '/' && char != '<'
}
