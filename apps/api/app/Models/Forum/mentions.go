package forum

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var mentionPattern = regexp.MustCompile(`(?:^|[^\pL\pN_])@([\pL\pN_]{1,50})`)

func mentionedUsernames(markdown string) []string {
	source := []byte(markdown)
	document := goldmark.DefaultParser().Parse(text.NewReader(source))
	seen := map[string]struct{}{}
	items := []string{}
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node.Kind() {
		case ast.KindCodeBlock, ast.KindFencedCodeBlock, ast.KindCodeSpan:
			return ast.WalkSkipChildren, nil
		case ast.KindText:
			value := string(node.(*ast.Text).Segment.Value(source))
			for _, match := range mentionPattern.FindAllStringSubmatch(value, -1) {
				key := strings.ToLower(match[1])
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				items = append(items, match[1])
			}
		}
		return ast.WalkContinue, nil
	})
	return items
}
