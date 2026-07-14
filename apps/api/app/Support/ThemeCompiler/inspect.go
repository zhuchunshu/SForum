package themecompiler

import (
	"errors"
	"fmt"
	stdhtml "html"
	htmltemplate "html/template"
	"io"
	"reflect"
	"regexp"
	"strings"
	"text/template/parse"

	nethtml "golang.org/x/net/html"
)

var (
	unsafeTagPattern  = regexp.MustCompile(`(?i)<\s*/?\s*(script|style|iframe|object|embed|svg|math|base|meta|link|form)(?:\s|/?>)`)
	unsafeAttrPattern = regexp.MustCompile(`(?i)\s(?:on[a-z0-9_-]+|style|srcdoc)\s*=`)
	allowedFunctions  = map[string]struct{}{
		"and": {}, "or": {}, "not": {},
		"eq": {}, "ne": {}, "lt": {}, "le": {}, "gt": {}, "ge": {},
		"len": {}, "index": {},
		"asset": {}, "route": {}, "i18n": {},
	}
	unsafeTags = map[string]struct{}{
		"script": {}, "style": {}, "iframe": {}, "object": {}, "embed": {},
		"svg": {}, "math": {}, "base": {}, "meta": {}, "link": {}, "form": {},
	}
	urlAttributes = map[string]struct{}{
		"action": {}, "background": {}, "cite": {}, "data": {}, "formaction": {},
		"href": {}, "longdesc": {}, "manifest": {}, "ping": {}, "poster": {},
		"profile": {}, "src": {}, "srcset": {}, "usemap": {},
	}
)

const templateActionMarker = "sforum-template-action-"

func inspectStaticHTML(name string, source []byte) error {
	decoded := strings.ToLower(stdhtml.UnescapeString(string(source)))
	if match := unsafeTagPattern.FindString(decoded); match != "" {
		return fmt.Errorf("%w: %s contains %q", ErrUnsafeStaticHTML, name, match)
	}
	if match := unsafeAttrPattern.FindString(decoded); match != "" {
		return fmt.Errorf("%w: %s contains unsafe attribute %q", ErrUnsafeStaticHTML, name, strings.TrimSpace(match))
	}
	compact := strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "", "\x00", "").Replace(decoded)
	for _, scheme := range []string{"javascript:", "vbscript:", "data:text/html"} {
		if strings.Contains(compact, scheme) {
			return fmt.Errorf("%w: %s contains forbidden URL scheme", ErrUnsafeStaticHTML, name)
		}
	}
	return inspectHTMLTokens(name, source)
}

// 正则负责保守拦截模板分支里的危险静态片段；HTML tokenizer 负责按浏览器
// 语义识别 `<svg/onload=...>` 这类非标准分隔，二者缺一都会留下绕过路径。
func inspectHTMLTokens(name string, source []byte) error {
	stripped, err := rewriteTemplateActions(string(source), false)
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrInvalidTemplate, name, err)
	}
	// Control actions emit no bytes themselves. Inspect that possible output so
	// `scr{{if ...}}{{end}}ipt` cannot reconstruct a forbidden static tag.
	if err := inspectHTMLTokenStream(name, stripped); err != nil {
		return err
	}
	masked, err := rewriteTemplateActions(string(source), true)
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrInvalidTemplate, name, err)
	}
	return inspectHTMLTokenStream(name, masked)
}

func inspectHTMLTokenStream(name, source string) error {
	tokenizer := nethtml.NewTokenizer(strings.NewReader(source))
	for {
		switch tokenType := tokenizer.Next(); tokenType {
		case nethtml.ErrorToken:
			if err := tokenizer.Err(); err != nil && !errors.Is(err, io.EOF) {
				return fmt.Errorf("%w: tokenize %s: %v", ErrInvalidTemplate, name, err)
			}
			return nil
		case nethtml.StartTagToken, nethtml.SelfClosingTagToken, nethtml.EndTagToken:
			token := tokenizer.Token()
			tag := strings.ToLower(token.Data)
			if _, forbidden := unsafeTags[tag]; forbidden {
				return fmt.Errorf("%w: %s contains forbidden tag %q", ErrUnsafeStaticHTML, name, tag)
			}
			if tokenType == nethtml.EndTagToken {
				continue
			}
			for _, attribute := range token.Attr {
				key := strings.ToLower(attribute.Key)
				if strings.HasPrefix(key, "on") || key == "style" || key == "srcdoc" {
					return fmt.Errorf("%w: %s contains unsafe attribute %q", ErrUnsafeStaticHTML, name, key)
				}
				if _, isURL := urlAttributes[key]; isURL {
					if err := inspectURLAttribute(attribute.Val); err != nil {
						return fmt.Errorf("%w: %s attribute %q: %v", ErrUnsafeStaticHTML, name, key, err)
					}
				}
			}
		}
	}
}

func inspectURLAttribute(value string) error {
	if !strings.Contains(value, templateActionMarker) {
		canonical := canonicalURLValue(value)
		for _, scheme := range []string{"javascript:", "vbscript:", "data:", "file:"} {
			if strings.Contains(canonical, scheme) {
				return fmt.Errorf("forbidden URL scheme %q", strings.TrimSuffix(scheme, ":"))
			}
		}
		return nil
	}

	// html/template filters a value supplied by one action, but cannot prevent
	// `{{.Prefix}}script:` from becoming `javascript:` after string assembly.
	if isSingleTemplateAction(value) || hasUnambiguousURLPrefix(value) {
		return nil
	}
	return fmt.Errorf("dynamic URL must be one action or follow a fixed safe prefix")
}

func canonicalURLValue(value string) string {
	value = strings.Map(func(r rune) rune {
		if r <= 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, value)
	return strings.ToLower(value)
}

func isSingleTemplateAction(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, templateActionMarker) {
		return false
	}
	suffix := strings.TrimPrefix(value, templateActionMarker)
	if suffix == "" {
		return false
	}
	for _, character := range suffix {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func hasUnambiguousURLPrefix(value string) bool {
	action := strings.Index(value, templateActionMarker)
	if action <= 0 {
		return false
	}
	prefix := canonicalURLValue(value[:action])
	if prefix == "" {
		return false
	}
	if colon := strings.IndexByte(prefix, ':'); colon >= 0 {
		scheme := prefix[:colon]
		return scheme == "http" || scheme == "https" || scheme == "mailto" || scheme == "tel"
	}
	// A character that cannot occur in a URL scheme fixes the value as a
	// relative or protocol-relative URL before any dynamic fragment is added.
	for _, character := range prefix {
		if !isURLSchemeCharacter(character) {
			return true
		}
	}
	return false
}

func isURLSchemeCharacter(character rune) bool {
	return character >= 'a' && character <= 'z' ||
		character >= '0' && character <= '9' ||
		character == '+' || character == '-' || character == '.'
}

// rewriteTemplateActions 在 HTML tokenization 前移除 action 语法；markers=true
// 时保留无引号占位符，供 URL 组合策略识别动态片段。否则 helper 参数里的引号会
// 被 tokenizer 误认为外层 HTML attribute 的结束符。
func rewriteTemplateActions(source string, markers bool) (string, error) {
	var output strings.Builder
	output.Grow(len(source))
	actionIndex := 0
	for offset := 0; offset < len(source); {
		start := strings.Index(source[offset:], "{{")
		if start < 0 {
			output.WriteString(source[offset:])
			break
		}
		start += offset
		output.WriteString(source[offset:start])
		end, ok := templateActionEnd(source, start+2)
		if !ok {
			return "", fmt.Errorf("unterminated template action")
		}
		if markers {
			fmt.Fprintf(&output, "%s%d", templateActionMarker, actionIndex)
		}
		actionIndex++
		offset = end
	}
	return output.String(), nil
}

func templateActionEnd(source string, offset int) (int, bool) {
	if end, matched, ok := templateCommentActionEnd(source, offset); matched {
		return end, ok
	}
	var quote byte
	escaped := false
	for index := offset; index < len(source); index++ {
		character := source[index]
		if quote != 0 {
			if quote != '`' && escaped {
				escaped = false
				continue
			}
			if quote != '`' && character == '\\' {
				escaped = true
				continue
			}
			if character == quote {
				quote = 0
			}
			continue
		}
		switch character {
		case '\'', '"', '`':
			quote = character
		case '}':
			if index+1 < len(source) && source[index+1] == '}' {
				return index + 2, true
			}
		}
	}
	return 0, false
}

func templateCommentActionEnd(source string, offset int) (end int, matched bool, ok bool) {
	index := offset
	if index < len(source) && source[index] == '-' {
		index++
	}
	for index < len(source) && (source[index] == ' ' || source[index] == '\t' || source[index] == '\r' || source[index] == '\n') {
		index++
	}
	if !strings.HasPrefix(source[index:], "/*") {
		return 0, false, false
	}
	commentEnd := strings.Index(source[index+2:], "*/")
	if commentEnd < 0 {
		return 0, true, false
	}
	commentEnd += index + 4
	closeOffset := strings.Index(source[commentEnd:], "}}")
	if closeOffset < 0 {
		return 0, true, false
	}
	return commentEnd + closeOffset + 2, true, true
}

func validateTemplateSet(set *htmltemplate.Template, entry string, maxDepth int) error {
	graphs := make(map[string][]string)
	for _, current := range set.Templates() {
		if current.Tree == nil || current.Name() == internalRootTemplate {
			continue
		}
		if err := validateTemplateName(current.Name()); err != nil {
			return err
		}
		calls, err := inspectNode(current.Tree.Root)
		if err != nil {
			return fmt.Errorf("%w: %s: %w", ErrInvalidTemplate, current.Name(), err)
		}
		for _, called := range calls {
			if err := validateTemplateName(called); err != nil {
				return err
			}
			if target := set.Lookup(called); target == nil || target.Tree == nil {
				return fmt.Errorf("%w: %s calls missing template %q", ErrInvalidPartial, current.Name(), called)
			}
		}
		graphs[current.Name()] = calls
	}
	if target := set.Lookup(entry); target == nil || target.Tree == nil {
		return fmt.Errorf("%w: entry %s", ErrTemplateNotFound, entry)
	}
	return validateCallDepth(graphs, maxDepth)
}

func inspectNode(node parse.Node) ([]string, error) {
	calls := make([]string, 0)
	var walk func(parse.Node) error
	walk = func(current parse.Node) error {
		if current == nil {
			return nil
		}
		currentValue := reflect.ValueOf(current)
		if currentValue.Kind() == reflect.Pointer && currentValue.IsNil() {
			return nil
		}
		switch value := current.(type) {
		case *parse.ListNode:
			for _, child := range value.Nodes {
				if err := walk(child); err != nil {
					return err
				}
			}
		case *parse.ActionNode:
			return walk(value.Pipe)
		case *parse.IfNode:
			if err := walk(value.Pipe); err != nil {
				return err
			}
			if err := walk(value.List); err != nil {
				return err
			}
			return walk(value.ElseList)
		case *parse.RangeNode:
			if err := walk(value.Pipe); err != nil {
				return err
			}
			if err := walk(value.List); err != nil {
				return err
			}
			return walk(value.ElseList)
		case *parse.WithNode:
			if err := walk(value.Pipe); err != nil {
				return err
			}
			if err := walk(value.List); err != nil {
				return err
			}
			return walk(value.ElseList)
		case *parse.TemplateNode:
			calls = append(calls, value.Name)
			return walk(value.Pipe)
		case *parse.PipeNode:
			for _, command := range value.Cmds {
				if err := walk(command); err != nil {
					return err
				}
			}
		case *parse.CommandNode:
			for _, argument := range value.Args {
				if err := walk(argument); err != nil {
					return err
				}
			}
		case *parse.ChainNode:
			return walk(value.Node)
		case *parse.IdentifierNode:
			if _, ok := allowedFunctions[value.Ident]; !ok {
				return fmt.Errorf("%w: %s", ErrForbiddenHelper, value.Ident)
			}
		case *parse.TextNode, *parse.CommentNode, *parse.BoolNode, *parse.NumberNode,
			*parse.StringNode, *parse.NilNode, *parse.DotNode, *parse.FieldNode,
			*parse.VariableNode, *parse.BreakNode, *parse.ContinueNode:
			return nil
		default:
			return fmt.Errorf("unsupported parse node %T", current)
		}
		return nil
	}
	if err := walk(node); err != nil {
		return nil, err
	}
	return calls, nil
}

func validateCallDepth(graph map[string][]string, limit int) error {
	const (
		unseen = iota
		visiting
		done
	)
	state := make(map[string]int, len(graph))
	depths := make(map[string]int, len(graph))
	var depth func(string) (int, error)
	depth = func(name string) (int, error) {
		switch state[name] {
		case visiting:
			return 0, fmt.Errorf("%w: cycle at %s", ErrTemplateRecursion, name)
		case done:
			return depths[name], nil
		}
		state[name] = visiting
		max := 1
		for _, child := range graph[name] {
			childDepth, err := depth(child)
			if err != nil {
				return 0, err
			}
			if childDepth+1 > max {
				max = childDepth + 1
			}
		}
		if max > limit {
			return 0, fmt.Errorf("%w: %s reaches depth %d (max %d)", ErrTemplateRecursion, name, max, limit)
		}
		state[name] = done
		depths[name] = max
		return max, nil
	}
	for name := range graph {
		if _, err := depth(name); err != nil {
			return err
		}
	}
	return nil
}

// html/template performs contextual analysis lazily on first execution. Run a
// zero-data validation pass before publication and only ignore ordinary
// missing-data execution errors.
func validateContextEscaping(set *htmltemplate.Template, entry string) error {
	err := set.ExecuteTemplate(io.Discard, entry, map[string]any{})
	if err == nil {
		return nil
	}
	var contextError *htmltemplate.Error
	if errors.As(err, &contextError) {
		return fmt.Errorf("%w: contextual escaping for %s: %v", ErrInvalidTemplate, entry, err)
	}
	return nil
}
