package editordocument

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var (
	shortcodeBracketName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	shortcodeBracketKey  = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

type shortcodeSourceLine struct {
	raw   string
	start int
}

type shortcodeTextParser struct {
	source       string
	lines        []shortcodeSourceLine
	opaque       []bool
	resourceKind string
	count        int
	maxDepth     int
	invalid      bool
}

// ParseShortcodeMarkdown imports bracket syntax through Goldmark's native AST
// and returns a complete Host document only when at least one declaration is
// admitted. Invalid/unknown/over-budget syntax returns activated=false so the
// caller can keep the original Markdown path byte-for-byte.
func ParseShortcodeMarkdown(markdown, resourceKind string) (Document, bool, error) {
	parser := &shortcodeTextParser{
		source:       markdown,
		lines:        splitShortcodeSourceLines(markdown),
		resourceKind: strings.TrimSpace(resourceKind),
	}
	parser.opaque = shortcodeOpaqueLines(markdown, parser.lines)
	content, activated := parser.parseRange(0, len(parser.lines), 0)
	if !activated || parser.invalid || parser.count > ShortcodeMaxNodes || parser.maxDepth > ShortcodeMaxDepth {
		return Document{}, false, nil
	}
	doc := Document{Type: "doc", Content: content}
	if len(content) == 0 {
		return Document{}, false, nil
	}
	return doc, true, nil
}

func splitShortcodeSourceLines(source string) []shortcodeSourceLine {
	if source == "" {
		return nil
	}
	parts := strings.SplitAfter(source, "\n")
	lines := make([]shortcodeSourceLine, 0, len(parts))
	offset := 0
	for _, part := range parts {
		if part == "" {
			continue
		}
		lines = append(lines, shortcodeSourceLine{raw: part, start: offset})
		offset += len(part)
	}
	return lines
}

func (p *shortcodeTextParser) parseRange(start, end, depth int) ([]Node, bool) {
	var content []Node
	activated := false
	ordinaryStart := start
	flushOrdinary := func(until int) {
		if until <= ordinaryStart {
			return
		}
		content = append(content, parseOrdinaryMarkdown(p.joinLines(ordinaryStart, until))...)
	}

	for index := start; index < end; index++ {
		if p.opaque[index] {
			continue
		}
		rawLine := strings.TrimSuffix(strings.TrimSuffix(p.lines[index].raw, "\n"), "\r")
		if strings.HasPrefix(rawLine, "\t") || len(rawLine)-len(strings.TrimLeft(rawLine, " ")) > 3 {
			continue
		}
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, `\[`) {
			continue
		}
		if declaration, arguments, ok := parseShortcodeReferenceLine(line, p.resourceKind); ok {
			flushOrdinary(index)
			content = append(content, shortcodeNodeForDeclaration(declaration, arguments, nil))
			p.count++
			currentDepth := depth + 1
			if currentDepth > p.maxDepth {
				p.maxDepth = currentDepth
			}
			activated = true
			ordinaryStart = index + 1
			continue
		}
		declaration, arguments, ok := parseShortcodeOpeningLine(line, p.resourceKind)
		if !ok || declaration.Kind != shortcodeProtected {
			continue
		}
		closing, matched := p.findProtectedClose(index, end, declaration.Name)
		if !matched || closing == index+1 || strings.TrimSpace(p.joinLines(index+1, closing)) == "" {
			continue
		}
		children, childActivated := p.parseRange(index+1, closing, depth+1)
		_ = childActivated
		if len(children) == 0 {
			continue
		}
		flushOrdinary(index)
		content = append(content, shortcodeNodeForDeclaration(declaration, arguments, children))
		p.count++
		currentDepth := depth + 1
		if currentDepth > p.maxDepth {
			p.maxDepth = currentDepth
		}
		activated = true
		index = closing
		ordinaryStart = closing + 1
	}
	flushOrdinary(end)
	return content, activated
}

func (p *shortcodeTextParser) findProtectedClose(open, end int, openingName string) (int, bool) {
	stack := []string{openingName}
	for index := open + 1; index < end; index++ {
		if p.opaque[index] {
			continue
		}
		line := strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(p.lines[index].raw, "\n"), "\r"))
		if strings.HasPrefix(line, `\[`) {
			continue
		}
		if declaration, _, ok := parseShortcodeOpeningLine(line, p.resourceKind); ok && declaration.Kind == shortcodeProtected {
			stack = append(stack, declaration.Name)
			continue
		}
		name, closing := parseShortcodeClosingLine(line)
		if !closing {
			continue
		}
		declaration, declared := shortcodeByName[name]
		if !declared || declaration.Kind != shortcodeProtected {
			continue
		}
		if stack[len(stack)-1] != name {
			return 0, false
		}
		previous := strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(p.lines[index-1].raw, "\n"), "\r"))
		if shortcodeLazyContinuationLine(previous) {
			return 0, false
		}
		stack = stack[:len(stack)-1]
		if len(stack) == 0 {
			return index, true
		}
	}
	return 0, false
}

func shortcodeLazyContinuationLine(line string) bool {
	return strings.HasPrefix(line, ">") || regexp.MustCompile(`^(?:[-+*]|\d+[.)])\s+`).MatchString(line)
}

func (p *shortcodeTextParser) joinLines(start, end int) string {
	var builder strings.Builder
	for _, line := range p.lines[start:end] {
		builder.WriteString(line.raw)
	}
	return builder.String()
}

func parseShortcodeReferenceLine(line, resourceKind string) (shortcodeDeclaration, map[string]any, bool) {
	closingBracket := strings.IndexByte(line, ']')
	if closingBracket < 0 {
		return shortcodeDeclaration{}, nil, false
	}
	declaration, arguments, ok := parseShortcodeOpeningLine(line[:closingBracket+1], resourceKind)
	openingInner := line[1:closingBracket]
	syntaxName := openingInner
	if split := strings.IndexAny(openingInner, " \t"); split >= 0 {
		syntaxName = openingInner[:split]
	}
	if !ok || declaration.Kind != shortcodeReference || line[closingBracket+1:] != "[/"+syntaxName+"]" {
		return shortcodeDeclaration{}, nil, false
	}
	return declaration, arguments, true
}

func parseShortcodeOpeningLine(line, resourceKind string) (shortcodeDeclaration, map[string]any, bool) {
	if len(line) < 3 || line[0] != '[' || line[len(line)-1] != ']' || strings.HasPrefix(line, "[/") {
		return shortcodeDeclaration{}, nil, false
	}
	inner := line[1 : len(line)-1]
	name := inner
	argumentText := ""
	if split := strings.IndexAny(inner, " \t"); split >= 0 {
		name = inner[:split]
		argumentText = strings.TrimSpace(inner[split+1:])
	}
	if !shortcodeBracketName.MatchString(name) {
		return shortcodeDeclaration{}, nil, false
	}
	declaration, declared := shortcodeByName[name]
	alias := false
	if !declared && name == "topic-tag" {
		declaration = shortcodeByName["category"]
		declared = true
		alias = true
	}
	if !declared || (declaration.CommentOnly && resourceKind == "topic") {
		return shortcodeDeclaration{}, nil, false
	}
	if declaration.CanonicalArg == "" {
		if argumentText != "" {
			return shortcodeDeclaration{}, nil, false
		}
		return declaration, map[string]any{}, true
	}
	key, value, ok := parseSingleShortcodeArgument(argumentText)
	if !ok {
		return shortcodeDeclaration{}, nil, false
	}
	expectedKey := declaration.BracketArg
	if alias {
		expectedKey = "tag_id"
	}
	if key != expectedKey {
		return shortcodeDeclaration{}, nil, false
	}
	return declaration, map[string]any{declaration.CanonicalArg: value}, true
}

func parseSingleShortcodeArgument(value string) (string, int64, bool) {
	if value == "" {
		return "", 0, false
	}
	equals := strings.IndexByte(value, '=')
	if equals <= 0 || strings.Contains(value[equals+1:], "=") {
		return "", 0, false
	}
	key := strings.TrimSpace(value[:equals])
	raw := strings.TrimSpace(value[equals+1:])
	if !shortcodeBracketKey.MatchString(key) || len(key) > ShortcodeMaxKeyBytes || raw == "" || strings.ContainsAny(key, " \t") {
		return "", 0, false
	}
	if strings.HasPrefix(raw, `"`) || strings.HasSuffix(raw, `"`) {
		if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
			return "", 0, false
		}
		raw = raw[1 : len(raw)-1]
	}
	for _, character := range raw {
		if character < '0' || character > '9' {
			return "", 0, false
		}
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed <= 0 {
		return "", 0, false
	}
	return key, parsed, true
}

func parseShortcodeClosingLine(line string) (string, bool) {
	if len(line) < 4 || !strings.HasPrefix(line, "[/") || line[len(line)-1] != ']' {
		return "", false
	}
	name := line[2 : len(line)-1]
	return name, shortcodeBracketName.MatchString(name)
}

func shortcodeOpaqueLines(source string, lines []shortcodeSourceLine) []bool {
	result := make([]bool, len(lines))
	if len(lines) == 0 {
		return result
	}
	markdown := goldmark.New(goldmark.WithExtensions(extension.GFM))
	root := markdown.Parser().Parse(text.NewReader([]byte(source)))
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		blocked := node.Kind() == ast.KindCodeBlock || node.Kind() == ast.KindFencedCodeBlock ||
			node.Kind() == ast.KindHTMLBlock || node.Kind().String() == "Table"
		for ancestor := node.Parent(); !blocked && ancestor != nil; ancestor = ancestor.Parent() {
			blocked = ancestor.Kind() == ast.KindListItem || ancestor.Kind() == ast.KindBlockquote || ancestor.Kind().String() == "Table"
		}
		if !blocked || node.Type() != ast.TypeBlock || node.Lines() == nil {
			return ast.WalkContinue, nil
		}
		segments := node.Lines()
		for index := 0; index < segments.Len(); index++ {
			lineIndex := shortcodeLineAtOffset(lines, segments.At(index).Start)
			if lineIndex >= 0 {
				result[lineIndex] = true
			}
		}
		return ast.WalkContinue, nil
	})
	return result
}

func shortcodeLineAtOffset(lines []shortcodeSourceLine, offset int) int {
	for index := len(lines) - 1; index >= 0; index-- {
		if offset >= lines[index].start {
			return index
		}
	}
	return -1
}

func parseOrdinaryMarkdown(source string) []Node {
	if strings.TrimSpace(source) == "" {
		return nil
	}
	markdown := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)
	root := markdown.Parser().Parse(text.NewReader([]byte(source)))
	return goldmarkBlockChildren(root, []byte(source))
}

func goldmarkBlockChildren(parent ast.Node, source []byte) []Node {
	var result []Node
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		result = append(result, goldmarkBlockNode(child, source)...)
	}
	return result
}

func goldmarkBlockNode(node ast.Node, source []byte) []Node {
	switch typed := node.(type) {
	case *ast.Paragraph, *ast.TextBlock:
		return []Node{{Type: "paragraph", Content: goldmarkInlineChildren(node, source, nil)}}
	case *ast.Heading:
		return []Node{{Type: "heading", Attrs: map[string]any{"level": float64(typed.Level)}, Content: goldmarkInlineChildren(node, source, nil)}}
	case *ast.Blockquote:
		return []Node{{Type: "blockquote", Content: goldmarkBlockChildren(node, source)}}
	case *ast.CodeBlock:
		return []Node{{Type: "codeBlock", Content: []Node{{Type: "text", Text: goldmarkLinesValue(node.Lines(), source)}}}}
	case *ast.FencedCodeBlock:
		attrs := map[string]any{}
		if language := string(typed.Language(source)); language != "" {
			attrs["language"] = language
		}
		return []Node{{Type: "codeBlock", Attrs: attrs, Content: []Node{{Type: "text", Text: goldmarkLinesValue(node.Lines(), source)}}}}
	case *ast.List:
		nodeType := "bulletList"
		attrs := map[string]any(nil)
		if typed.IsOrdered() {
			nodeType = "orderedList"
			if typed.Start != 1 {
				attrs = map[string]any{"start": float64(typed.Start)}
			}
		}
		return []Node{{Type: nodeType, Attrs: attrs, Content: goldmarkBlockChildren(node, source)}}
	case *ast.ListItem:
		return []Node{{Type: "listItem", Content: goldmarkBlockChildren(node, source)}}
	case *ast.ThematicBreak:
		return []Node{{Type: "horizontalRule"}}
	case *ast.HTMLBlock:
		return literalMarkdownBlock(goldmarkLinesValue(node.Lines(), source))
	default:
		if node.Kind().String() == "Table" {
			return literalMarkdownBlock(goldmarkLinesValue(node.Lines(), source))
		}
		if node.HasChildren() {
			return goldmarkBlockChildren(node, source)
		}
		return literalMarkdownBlock(string(node.Text(source)))
	}
}

func goldmarkInlineChildren(parent ast.Node, source []byte, marks []Mark) []Node {
	var result []Node
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		result = append(result, goldmarkInlineNode(child, source, marks)...)
	}
	return result
}

func goldmarkInlineNode(node ast.Node, source []byte, marks []Mark) []Node {
	cloneMarks := func(extra Mark) []Mark {
		result := append([]Mark(nil), marks...)
		return append(result, extra)
	}
	switch typed := node.(type) {
	case *ast.Text:
		result := []Node{{Type: "text", Text: string(typed.Segment.Value(source)), Marks: append([]Mark(nil), marks...)}}
		if typed.HardLineBreak() || typed.SoftLineBreak() {
			result = append(result, Node{Type: "hardBreak"})
		}
		return result
	case *ast.String:
		return []Node{{Type: "text", Text: string(typed.Value), Marks: append([]Mark(nil), marks...)}}
	case *ast.Emphasis:
		markType := "italic"
		if typed.Level == 2 {
			markType = "bold"
		}
		return goldmarkInlineChildren(node, source, cloneMarks(Mark{Type: markType}))
	case *ast.CodeSpan:
		return []Node{{Type: "text", Text: string(typed.Text(source)), Marks: cloneMarks(Mark{Type: "code"})}}
	case *ast.Link:
		return goldmarkInlineChildren(node, source, cloneMarks(Mark{Type: "link", Attrs: map[string]any{"href": string(typed.Destination)}}))
	case *ast.AutoLink:
		value := string(typed.URL(source))
		return []Node{{Type: "text", Text: value, Marks: cloneMarks(Mark{Type: "link", Attrs: map[string]any{"href": value}})}}
	case *ast.Image:
		return []Node{{Type: "image", Attrs: map[string]any{"src": string(typed.Destination), "alt": string(node.Text(source))}}}
	case *ast.RawHTML:
		var builder bytes.Buffer
		for index := 0; index < typed.Segments.Len(); index++ {
			segment := typed.Segments.At(index)
			builder.Write(segment.Value(source))
		}
		return []Node{{Type: "text", Text: builder.String(), Marks: append([]Mark(nil), marks...)}}
	default:
		switch node.Kind().String() {
		case "Strikethrough":
			return goldmarkInlineChildren(node, source, cloneMarks(Mark{Type: "strike"}))
		case "TaskCheckBox":
			return []Node{{Type: "text", Text: "[ ] ", Marks: append([]Mark(nil), marks...)}}
		}
		if node.HasChildren() {
			return goldmarkInlineChildren(node, source, marks)
		}
		return nil
	}
}

func goldmarkLinesValue(lines *text.Segments, source []byte) string {
	if lines == nil {
		return ""
	}
	var builder strings.Builder
	for index := 0; index < lines.Len(); index++ {
		segment := lines.At(index)
		builder.Write(segment.Value(source))
	}
	return strings.TrimSuffix(builder.String(), "\n")
}

func literalMarkdownBlock(value string) []Node {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return []Node{{Type: "paragraph", Content: []Node{{Type: "text", Text: value}}}}
}
