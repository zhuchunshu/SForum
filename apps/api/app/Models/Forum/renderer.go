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
	"golang.org/x/net/html"
)

const excerptRuneLimit = 180

func RenderContent(input ContentInput) (RenderedContent, error) {
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
		if err := goldmark.Convert([]byte(safeRaw), &buffer); err != nil {
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

	hash := sha256.Sum256([]byte(sourceFormat + "\x00" + raw))
	return RenderedContent{
		RawContent:    raw,
		HTMLContent:   renderedHTML,
		PlainText:     plain,
		Excerpt:       makeExcerpt(plain, excerptRuneLimit),
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

func sanitizeHTML(value string) string {
	policy := bluemonday.UGCPolicy()
	policy.RequireNoFollowOnLinks(true)
	policy.RequireNoReferrerOnLinks(true)
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
