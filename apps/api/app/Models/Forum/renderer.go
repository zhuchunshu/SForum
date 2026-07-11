package forum

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"golang.org/x/net/html"
)

const defaultExcerptRuneLimit = RecommendedExcerptRuneLimit

func RenderContent(input ContentInput) (RenderedContent, error) {
	return RenderContentWithExcerptLimit(input, defaultExcerptRuneLimit)
}

// RenderContentWithExcerptLimit 使用运营配置的摘要长度截断 plain text。
func RenderContentWithExcerptLimit(input ContentInput, excerptLimit int) (RenderedContent, error) {
	raw := strings.TrimSpace(input.RawContent)
	if raw == "" {
		return RenderedContent{}, ErrInvalidContent
	}

	sourceFormat := strings.TrimSpace(input.SourceFormat)
	if sourceFormat == "" {
		sourceFormat = SourceFormatMarkdown
	}
	editorType := strings.TrimSpace(input.EditorType)
	if editorType == "" {
		editorType = sourceFormat
	}

	var renderedHTML string
	safeRaw := stripUnsafeHTMLBlocks(raw)
	switch sourceFormat {
	case SourceFormatMarkdown:
		var buffer bytes.Buffer
		// 启用 GFM 扩展：表格、删除线、自动链接、任务列表。
		// 与前端 Tiptap 编辑器的 gfm:true 保持一致，避免"编辑器预览 ≠ 发布结果"。
		md := goldmark.New(goldmark.WithExtensions(extension.GFM))
		if err := md.Convert([]byte(safeRaw), &buffer); err != nil {
			return RenderedContent{}, err
		}
		renderedHTML = sanitizeHTML(buffer.String())
	case SourceFormatHTML:
		renderedHTML = sanitizeHTML(safeRaw)
	case SourceFormatJSON:
		return RenderedContent{}, ErrInvalidContent
	default:
		return RenderedContent{}, ErrInvalidContent
	}

	plain := normalizePlainText(htmlToText(renderedHTML))
	if plain == "" {
		return RenderedContent{}, ErrInvalidContent
	}

	limit := excerptLimit
	if limit < HardExcerptMinRunes || limit > HardExcerptMaxRunes {
		limit = defaultExcerptRuneLimit
	}

	hash := sha256.Sum256([]byte(sourceFormat + "\x00" + raw))
	return RenderedContent{
		RawContent:    raw,
		HTMLContent:   renderedHTML,
		PlainText:     plain,
		Excerpt:       makeExcerpt(plain, limit),
		SourceFormat:  sourceFormat,
		EditorType:    editorType,
		EditorVersion: strings.TrimSpace(input.EditorVersion),
		RenderVersion: RenderVersion,
		ContentHash:   hex.EncodeToString(hash[:]),
	}, nil
}

var unsafeHTMLBlockPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`),
	regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style>`),
	regexp.MustCompile(`(?is)<iframe\b[^>]*>.*?</iframe>`),
	regexp.MustCompile(`(?is)<object\b[^>]*>.*?</object>`),
	regexp.MustCompile(`(?is)<embed\b[^>]*>.*?</embed>`),
}

func stripUnsafeHTMLBlocks(value string) string {
	for _, pattern := range unsafeHTMLBlockPatterns {
		value = pattern.ReplaceAllString(value, "")
	}
	return value
}

// codeClassPattern 匹配 goldmark 代码块输出的 language-<lang> class，供 highlight.js 精确识别语言。
// 仅允许字母/数字与少量符号，避免 class 成为属性注入入口。
var codeClassPattern = regexp.MustCompile(`^language-[a-z0-9+#.-]+$`)

// checkboxTypePattern 仅允许 type="checkbox"，用于 GFM 任务列表（TaskList 输出只读 checkbox）。
var checkboxTypePattern = regexp.MustCompile(`^checkbox$`)

func sanitizeHTML(value string) string {
	policy := bluemonday.UGCPolicy()
	policy.RequireNoFollowOnLinks(true)
	policy.RequireNoReferrerOnLinks(true)

	// 保留代码块 language-* class，供前端 highlight.js 识别语言。
	policy.AllowAttrs("class").Matching(codeClassPattern).OnElements("code")

	// 放开 GFM 任务列表的 checkbox：仅允许 type=checkbox，且只保留 checked/disabled。
	// 事件属性（onclick 等）与非 checkbox input 仍会被 bluemonday 剔除，放开是收敛的。
	policy.AllowElements("input")
	policy.AllowAttrs("type").Matching(checkboxTypePattern).OnElements("input")
	policy.AllowAttrs("checked").OnElements("input")
	policy.AllowAttrs("disabled").OnElements("input")

	return policy.Sanitize(value)
}

func htmlToText(value string) string {
	node, err := html.Parse(strings.NewReader(value))
	if err != nil {
		return value
	}
	var builder strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			builder.WriteString(n.Data)
			builder.WriteByte(' ')
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return builder.String()
}

func normalizePlainText(value string) string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return unicode.IsSpace(r)
	})
	return strings.Join(fields, " ")
}

func makeExcerpt(value string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:limit])) + "..."
}
