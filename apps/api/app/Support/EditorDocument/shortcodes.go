package editordocument

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	ShortcodeRefNode   = "sforumShortcodeRef"
	ShortcodeBlockNode = "sforumShortcodeBlock"

	ShortcodeTextVersion    = "sforum.shortcode-text@1"
	ShortcodeMaxNodes       = 32
	ShortcodeMaxDepth       = 4
	ShortcodeMaxArguments   = 16
	ShortcodeMaxKeyBytes    = 64
	ShortcodeMaxStringRunes = 512
)

type shortcodeKind uint8

const (
	shortcodeReference shortcodeKind = iota + 1
	shortcodeProtected
)

type shortcodeDeclaration struct {
	Name          string
	ID            string
	Version       string
	Kind          shortcodeKind
	CanonicalArg  string
	BracketArg    string
	CommentOnly   bool
	FallbackCode  string
	FallbackLabel string
}

var shortcodeDeclarations = []shortcodeDeclaration{
	{Name: "user", ID: "sforum-shortcodes.user", Version: "sforum-shortcodes.user@1", Kind: shortcodeReference, CanonicalArg: "userId", BracketArg: "user_id", FallbackCode: "shortcode.user.unavailable", FallbackLabel: "用户引用暂不可用"},
	{Name: "topic", ID: "sforum-shortcodes.topic", Version: "sforum-shortcodes.topic@1", Kind: shortcodeReference, CanonicalArg: "topicId", BracketArg: "topic_id", FallbackCode: "shortcode.topic.unavailable", FallbackLabel: "主题引用暂不可用"},
	{Name: "comment", ID: "sforum-shortcodes.comment", Version: "sforum-shortcodes.comment@1", Kind: shortcodeReference, CanonicalArg: "commentId", BracketArg: "comment_id", FallbackCode: "shortcode.comment.unavailable", FallbackLabel: "评论引用暂不可用"},
	{Name: "category", ID: "sforum-shortcodes.category", Version: "sforum-shortcodes.category@1", Kind: shortcodeReference, CanonicalArg: "categoryId", BracketArg: "category_id", FallbackCode: "shortcode.category.unavailable", FallbackLabel: "分类引用暂不可用"},
	{Name: "friend-links", ID: "sforum-shortcodes.friend-links", Version: "sforum-shortcodes.friend-links@1", Kind: shortcodeReference, FallbackCode: "shortcode.friend_links.omitted"},
	{Name: "login", ID: "sforum-shortcodes.login", Version: "sforum-shortcodes.login@1", Kind: shortcodeProtected, FallbackCode: "shortcode.protected.unavailable", FallbackLabel: "受保护内容暂不可用"},
	{Name: "reply", ID: "sforum-shortcodes.reply", Version: "sforum-shortcodes.reply@1", Kind: shortcodeProtected, FallbackCode: "shortcode.protected.unavailable", FallbackLabel: "受保护内容暂不可用"},
	{Name: "only-author", ID: "sforum-shortcodes.only-author", Version: "sforum-shortcodes.only-author@1", Kind: shortcodeProtected, CommentOnly: true, FallbackCode: "shortcode.protected.unavailable", FallbackLabel: "受保护内容暂不可用"},
}

var (
	shortcodeCanonicalKey = regexp.MustCompile(`^[a-z][A-Za-z0-9]*$`)
	shortcodeByID         = indexShortcodesByID()
	shortcodeByName       = indexShortcodesByName()
)

func shortcodeNodeAttrs() map[string]bool {
	return map[string]bool{"id": true, "contractVersion": true, "arguments": true}
}

func indexShortcodesByID() map[string]shortcodeDeclaration {
	result := make(map[string]shortcodeDeclaration, len(shortcodeDeclarations))
	for _, declaration := range shortcodeDeclarations {
		result[declaration.ID] = declaration
	}
	return result
}

func indexShortcodesByName() map[string]shortcodeDeclaration {
	result := make(map[string]shortcodeDeclaration, len(shortcodeDeclarations)+1)
	for _, declaration := range shortcodeDeclarations {
		result[declaration.Name] = declaration
	}
	return result
}

type shortcodeValidationState struct {
	count          int
	shortcodeDepth int
	resourceKind   string
	referencePath  map[ShortcodeReference]struct{}
}

func newShortcodeValidationState(input Input) *shortcodeValidationState {
	path := make(map[ShortcodeReference]struct{}, len(input.ReferencePath))
	for _, reference := range input.ReferencePath {
		path[reference] = struct{}{}
	}
	return &shortcodeValidationState{
		resourceKind:  strings.TrimSpace(input.ResourceKind),
		referencePath: path,
	}
}

func normalizeShortcodeNode(node Node, parentType string, state *shortcodeValidationState) (Node, string, error) {
	if parentType != "doc" && parentType != ShortcodeBlockNode {
		return Node{}, "", ErrInvalidShortcode
	}
	if len(node.Attrs) != 3 {
		return Node{}, "", ErrInvalidShortcode
	}
	for key := range node.Attrs {
		if !shortcodeNodeAttrs()[key] {
			return Node{}, "", ErrInvalidShortcode
		}
	}

	id, idOK := node.Attrs["id"].(string)
	version, versionOK := node.Attrs["contractVersion"].(string)
	arguments, argumentsOK := node.Attrs["arguments"].(map[string]any)
	declaration, declared := shortcodeByID[id]
	if !idOK || !versionOK || !argumentsOK || !declared || version != declaration.Version {
		return Node{}, "", ErrInvalidShortcode
	}
	if declaration.CommentOnly && state.resourceKind == "topic" {
		return Node{}, "", ErrInvalidShortcode
	}
	if node.Type == ShortcodeRefNode && declaration.Kind != shortcodeReference {
		return Node{}, "", ErrInvalidShortcode
	}
	if node.Type == ShortcodeBlockNode && declaration.Kind != shortcodeProtected {
		return Node{}, "", ErrInvalidShortcode
	}
	if len(arguments) > ShortcodeMaxArguments {
		return Node{}, "", ErrInvalidShortcode
	}

	normalizedArguments, stableID, err := normalizeShortcodeArguments(declaration, arguments)
	if err != nil {
		return Node{}, "", err
	}
	if stableID > 0 {
		if _, cyclic := state.referencePath[ShortcodeReference{ID: id, StableID: stableID}]; cyclic {
			return Node{}, "", ErrInvalidShortcode
		}
	}

	state.count++
	if state.count > ShortcodeMaxNodes {
		return Node{}, "", ErrInvalidShortcode
	}

	return Node{
		Type: node.Type,
		Attrs: map[string]any{
			"id": id, "contractVersion": version, "arguments": normalizedArguments,
		},
	}, declaration.FallbackCode, nil
}

func normalizeShortcodeArguments(declaration shortcodeDeclaration, arguments map[string]any) (map[string]any, int64, error) {
	for key, value := range arguments {
		if len(key) == 0 || len(key) > ShortcodeMaxKeyBytes || !shortcodeCanonicalKey.MatchString(key) {
			return nil, 0, ErrInvalidShortcode
		}
		if err := validateShortcodeScalar(value); err != nil {
			return nil, 0, err
		}
	}
	if declaration.CanonicalArg == "" {
		if len(arguments) != 0 {
			return nil, 0, ErrInvalidShortcode
		}
		return map[string]any{}, 0, nil
	}
	if len(arguments) != 1 {
		return nil, 0, ErrInvalidShortcode
	}
	value, exists := arguments[declaration.CanonicalArg]
	if !exists {
		return nil, 0, ErrInvalidShortcode
	}
	stableID, ok := shortcodePositiveInt64(value)
	if !ok {
		return nil, 0, ErrInvalidShortcode
	}
	return map[string]any{declaration.CanonicalArg: stableID}, stableID, nil
}

func validateShortcodeScalar(value any) error {
	switch typed := value.(type) {
	case string:
		if strings.ContainsRune(typed, '\x00') || utf8.RuneCountInString(typed) > ShortcodeMaxStringRunes {
			return ErrInvalidShortcode
		}
		return nil
	case bool, int, int32, int64, json.Number:
		return nil
	default:
		return ErrInvalidShortcode
	}
}

func shortcodePositiveInt64(value any) (int64, bool) {
	var parsed int64
	switch typed := value.(type) {
	case int:
		parsed = int64(typed)
	case int32:
		parsed = int64(typed)
	case int64:
		parsed = typed
	case json.Number:
		value, err := strconv.ParseInt(string(typed), 10, 64)
		if err != nil {
			return 0, false
		}
		parsed = value
	default:
		return 0, false
	}
	return parsed, parsed > 0
}

func shortcodeFallback(node Node) (code, label string, omit bool) {
	return shortcodeFallbackForLocale(node, "")
}

func shortcodeFallbackForLocale(node Node, locale string) (code, label string, omit bool) {
	id, _ := node.Attrs["id"].(string)
	declaration, ok := shortcodeByID[id]
	if !ok {
		code = "shortcode.reference.unavailable"
		return code, shortcodeFallbackLabel(code, locale), false
	}
	return declaration.FallbackCode, shortcodeFallbackLabel(declaration.FallbackCode, locale), declaration.Name == "friend-links"
}

func shortcodeFallbackLabel(code, locale string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "en") {
		switch code {
		case "shortcode.user.unavailable":
			return "User reference unavailable"
		case "shortcode.topic.unavailable":
			return "Topic reference unavailable"
		case "shortcode.comment.unavailable":
			return "Comment reference unavailable"
		case "shortcode.category.unavailable":
			return "Category reference unavailable"
		case "shortcode.protected.unavailable":
			return "Protected content unavailable"
		case "shortcode.login.required":
			return "Log in to view this content"
		case "shortcode.reply.required":
			return "Reply to view this content"
		case "shortcode.only_author.private":
			return "This content is private"
		default:
			return "Shortcode unavailable"
		}
	}
	switch code {
	case "shortcode.user.unavailable":
		return "用户引用暂不可用"
	case "shortcode.topic.unavailable":
		return "主题引用暂不可用"
	case "shortcode.comment.unavailable":
		return "评论引用暂不可用"
	case "shortcode.category.unavailable":
		return "分类引用暂不可用"
	case "shortcode.protected.unavailable":
		return "受保护内容暂不可用"
	case "shortcode.login.required":
		return "登录后可查看此内容"
	case "shortcode.reply.required":
		return "回复后可查看此内容"
	case "shortcode.only_author.private":
		return "此内容仅限相关作者查看"
	default:
		return "短代码暂不可用"
	}
}

func shortcodeCanonicalMarkdown(node Node) (string, bool) {
	id, _ := node.Attrs["id"].(string)
	declaration, ok := shortcodeByID[id]
	if !ok {
		return "", false
	}
	var opening strings.Builder
	opening.WriteByte('[')
	opening.WriteString(declaration.Name)
	if declaration.CanonicalArg != "" {
		arguments, _ := node.Attrs["arguments"].(map[string]any)
		stableID, valid := shortcodePositiveInt64(arguments[declaration.CanonicalArg])
		if !valid {
			return "", false
		}
		opening.WriteByte(' ')
		opening.WriteString(declaration.BracketArg)
		opening.WriteString(`="`)
		opening.WriteString(strconv.FormatInt(stableID, 10))
		opening.WriteByte('"')
	}
	opening.WriteByte(']')
	if declaration.Kind == shortcodeReference {
		opening.WriteString("[/")
		opening.WriteString(declaration.Name)
		opening.WriteByte(']')
	}
	return opening.String(), true
}

func shortcodeName(node Node) (string, bool) {
	id, _ := node.Attrs["id"].(string)
	declaration, ok := shortcodeByID[id]
	return declaration.Name, ok
}

func escapeLiteralShortcodeOpening(value string) string {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, `\[`) {
		return value
	}
	closing := strings.IndexByte(trimmed, ']')
	if closing < 2 {
		return value
	}
	nameEnd := strings.IndexAny(trimmed[1:closing], " \t")
	name := trimmed[1:closing]
	if nameEnd >= 0 {
		name = trimmed[1 : 1+nameEnd]
	}
	if _, declared := shortcodeByName[name]; !declared && name != "topic-tag" {
		return value
	}
	indent := len(value) - len(strings.TrimLeft(value, " \t"))
	return value[:indent] + `\` + value[indent:]
}

func shortcodeNodeForDeclaration(declaration shortcodeDeclaration, arguments map[string]any, content []Node) Node {
	nodeType := ShortcodeRefNode
	if declaration.Kind == shortcodeProtected {
		nodeType = ShortcodeBlockNode
	}
	if arguments == nil {
		arguments = map[string]any{}
	}
	return Node{Type: nodeType, Attrs: map[string]any{
		"id": declaration.ID, "contractVersion": declaration.Version, "arguments": arguments,
	}, Content: content}
}
