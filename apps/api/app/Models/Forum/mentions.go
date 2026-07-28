package forum

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var mentionPattern = regexp.MustCompile(`(?:^|[^\pL\pN_])@([\pL\pN_]{1,50})`)

// MentionedUsernames 从已经通过 Host 过滤的 Markdown 源中提取提及。代码块和
// 行内代码被忽略，供创建与审批后的通知重放共用。
func MentionedUsernames(markdown string) []string {
	source := []byte(markdown)
	document := goldmark.DefaultParser().Parse(text.NewReader(source))
	var visible strings.Builder
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node.Kind() {
		case ast.KindCodeBlock, ast.KindFencedCodeBlock, ast.KindCodeSpan:
			visible.WriteByte(' ')
			return ast.WalkSkipChildren, nil
		case ast.KindParagraph, ast.KindHeading:
			visible.WriteByte(' ')
		case ast.KindText:
			visible.Write(node.(*ast.Text).Segment.Value(source))
		}
		return ast.WalkContinue, nil
	})

	seen := map[string]struct{}{}
	items := []string{}
	for _, match := range mentionPattern.FindAllStringSubmatch(visible.String(), -1) {
		key := strings.ToLower(match[1])
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, match[1])
	}
	return items
}
