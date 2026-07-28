package markdown

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// RenderSafe renders operator-authored Markdown and sanitizes the result before
// it crosses an HTTP presentation boundary.
func RenderSafe(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}

	var buffer bytes.Buffer
	parser := goldmark.New(goldmark.WithExtensions(extension.GFM))
	if err := parser.Convert([]byte(value), &buffer); err != nil {
		return "", err
	}

	policy := bluemonday.NewPolicy()
	policy.AllowElements("a", "p", "br", "strong", "b", "em", "i", "ul", "ol", "li")
	policy.AllowAttrs("href").Matching(regexp.MustCompile(`^(https?://|/|#|mailto:)`)).OnElements("a")
	policy.AllowAttrs("title").OnElements("a")
	policy.AllowStandardURLs()
	policy.RequireParseableURLs(true)
	policy.AllowURLSchemes("http", "https", "mailto")
	policy.RequireNoFollowOnLinks(true)
	policy.RequireNoReferrerOnLinks(true)
	return policy.Sanitize(buffer.String()), nil
}
