package themecompiler

import (
	"bytes"
	"errors"
	"fmt"
	stdhtml "html"
	htmltemplate "html/template"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	nethtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var islandTagPattern = regexp.MustCompile(`^sf-[a-z][a-z0-9-]*$`)
var islandPropPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

const (
	islandPlaceholderAttribute   = "data-sforum-island"
	maxIslandFallbackSegments    = 256
	maxIslandFallbackOutputBytes = DefaultMaxOutputBytes
)

// requiredPageComponents 仅强制 mutation / 身份相关页必须嵌入宿主岛。
// 其余可替换页的 body 岛由参考主题产品门禁（P13）校验，避免把编译器
// 通用夹具绑死在完整公开页矩阵上。
var requiredPageComponents = map[string]string{
	"forum.topic.create":      "forum.component.topic_composer",
	"forum.topic.reply":       "forum.component.topic_reply",
	"forum.settings.profile":  "profile.component.settings_form",
	"forum.settings.security": "identity.component.security_settings",
	"auth.login":              "identity.component.login_form",
	"auth.register":           "identity.component.register_form",
	"auth.forgot_password":    "identity.component.recovery_request_form",
	"auth.reset_password":     "identity.component.recovery_confirm_form",
}

var protectedPageComponents = func() map[string]struct{} {
	result := make(map[string]struct{}, len(requiredPageComponents))
	for _, componentID := range requiredPageComponents {
		result[componentID] = struct{}{}
	}
	return result
}()

func validateIslandBindings(bindings map[string]IslandBinding) error {
	for tag, binding := range bindings {
		if !islandTagPattern.MatchString(tag) || !viewModelIDPattern.MatchString(binding.ComponentID) {
			return fmt.Errorf("%w: invalid binding %q", ErrInvalidIsland, tag)
		}
		seen := make(map[string]struct{}, len(binding.Props))
		for _, prop := range binding.Props {
			if !islandPropPattern.MatchString(prop.Name) || prop.Name == islandPlaceholderAttribute || sensitiveViewModelName(prop.Name) || !validIslandPropType(prop.Type) {
				return fmt.Errorf("%w: invalid prop %q for %s", ErrInvalidIsland, prop.Name, tag)
			}
			if _, exists := seen[prop.Name]; exists {
				return fmt.Errorf("%w: duplicate prop %q for %s", ErrInvalidIsland, prop.Name, tag)
			}
			seen[prop.Name] = struct{}{}
		}
	}
	return nil
}

func validIslandPropType(value IslandPropType) bool {
	switch value {
	case IslandPropString, IslandPropBoolean, IslandPropInteger, IslandPropURL:
		return true
	default:
		return false
	}
}

func validateTemplateIslandDeclarations(name string, source []byte, bindings map[string]IslandBinding) error {
	_, err := templateIslandComponents(name, string(source), bindings)
	return err
}

func templateIslandComponents(name, source string, bindings map[string]IslandBinding) ([]string, error) {
	stripped, err := rewriteTemplateActions(source, false)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrInvalidTemplate, name, err)
	}
	tokenizer := nethtml.NewTokenizer(strings.NewReader(stripped))
	components := make([]string, 0, 2)
	for {
		tokenType := tokenizer.Next()
		if tokenType == nethtml.ErrorToken {
			if err := tokenizer.Err(); err != nil && !errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("%w: tokenize %s: %v", ErrInvalidTemplate, name, err)
			}
			return components, nil
		}
		if tokenType != nethtml.StartTagToken && tokenType != nethtml.SelfClosingTagToken && tokenType != nethtml.EndTagToken {
			continue
		}
		token := tokenizer.Token()
		tag := strings.ToLower(token.Data)
		if tokenType != nethtml.EndTagToken {
			for _, attribute := range token.Attr {
				if strings.EqualFold(attribute.Key, islandPlaceholderAttribute) {
					return nil, fmt.Errorf("%w: %s uses reserved island placeholder", ErrInvalidIsland, name)
				}
			}
		}
		if !strings.HasPrefix(tag, "sf-") {
			continue
		}
		binding, ok := bindings[tag]
		if !ok {
			return nil, fmt.Errorf("%w: %s uses %s", ErrUnknownIsland, name, tag)
		}
		if tokenType != nethtml.EndTagToken {
			if err := validateIslandPropNames(token.Attr, binding); err != nil {
				return nil, fmt.Errorf("%w: %s: %v", ErrInvalidIsland, name, err)
			}
			components = append(components, binding.ComponentID)
		}
	}
}

func validateRequiredPageIsland(name string, set *htmltemplate.Template, pageID string, bindings map[string]IslandBinding) error {
	required, sensitive := requiredPageComponents[pageID]
	if !sensitive {
		return nil
	}
	seenTemplates := map[string]struct{}{}
	queue := []string{name}
	components := make([]string, 0, 2)
	for len(queue) != 0 {
		currentName := queue[0]
		queue = queue[1:]
		if _, seen := seenTemplates[currentName]; seen {
			continue
		}
		seenTemplates[currentName] = struct{}{}
		current := set.Lookup(currentName)
		if current == nil || current.Tree == nil || current.Tree.Root == nil {
			return fmt.Errorf("%w: %s calls unavailable template %q", ErrRequiredIsland, name, currentName)
		}
		declared, err := templateIslandComponents(currentName, current.Tree.Root.String(), bindings)
		if err != nil {
			return err
		}
		components = append(components, declared...)
		calls, err := inspectNode(current.Tree.Root)
		if err != nil {
			return err
		}
		queue = append(queue, calls...)
	}
	return validateRequiredComponentSet(pageID, required, components)
}

func validateRenderedRequiredPageIsland(pageID string, islands []IslandDescriptor) error {
	required, sensitive := requiredPageComponents[pageID]
	if !sensitive {
		return nil
	}
	components := make([]string, 0, len(islands))
	for _, island := range islands {
		components = append(components, island.ComponentID)
	}
	return validateRequiredComponentSet(pageID, required, components)
}

func validateRequiredComponentSet(pageID, required string, components []string) error {
	count := 0
	for _, componentID := range components {
		if componentID == required {
			count++
			continue
		}
		if _, protected := protectedPageComponents[componentID]; protected {
			return fmt.Errorf("%w: %s contains protected component %s instead of %s", ErrRequiredIsland, pageID, componentID, required)
		}
	}
	if count != 1 {
		return fmt.Errorf("%w: %s requires exactly one %s, found %d", ErrRequiredIsland, pageID, required, count)
	}
	return nil
}

func validateIslandPropNames(attributes []nethtml.Attribute, binding IslandBinding) error {
	contracts := make(map[string]IslandPropContract, len(binding.Props))
	for _, contract := range binding.Props {
		contracts[contract.Name] = contract
	}
	seen := make(map[string]struct{}, len(attributes))
	for _, attribute := range attributes {
		name := strings.ToLower(attribute.Key)
		if _, ok := contracts[name]; attribute.Namespace != "" || !ok {
			return fmt.Errorf("component %s rejects prop %q", binding.ComponentID, name)
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("duplicate prop %q", name)
		}
		seen[name] = struct{}{}
	}
	for _, contract := range binding.Props {
		if _, exists := seen[contract.Name]; contract.Required && !exists {
			return fmt.Errorf("component %s requires prop %q", binding.ComponentID, contract.Name)
		}
	}
	return nil
}

func segmentRenderedHTML(source string, bindings map[string]IslandBinding) ([]RenderedHTML, []IslandDescriptor, error) {
	tokenizer := nethtml.NewTokenizer(strings.NewReader(source))
	var skeleton strings.Builder
	islands := make([]IslandDescriptor, 0, 2)
	islandIndex := 0
	fallbackSegments := 0
	var fallbackBytes int64

	for {
		tokenType := tokenizer.Next()
		if tokenType == nethtml.ErrorToken {
			if err := tokenizer.Err(); err != nil && !errors.Is(err, io.EOF) {
				return nil, nil, fmt.Errorf("%w: tokenize output: %v", ErrExecution, err)
			}
			segments, err := balanceHTMLSkeleton(skeleton.String())
			if err != nil {
				return nil, nil, err
			}
			return segments, islands, nil
		}
		raw := string(tokenizer.Raw())
		if tokenType != nethtml.StartTagToken && tokenType != nethtml.SelfClosingTagToken && tokenType != nethtml.EndTagToken {
			skeleton.WriteString(raw)
			continue
		}
		token := tokenizer.Token()
		tag := strings.ToLower(token.Data)
		if !strings.HasPrefix(tag, "sf-") {
			if tokenType != nethtml.EndTagToken {
				for _, attribute := range token.Attr {
					if strings.EqualFold(attribute.Key, islandPlaceholderAttribute) {
						return nil, nil, fmt.Errorf("%w: rendered HTML contains reserved island placeholder", ErrInvalidIsland)
					}
				}
			}
			skeleton.WriteString(raw)
			continue
		}
		if tokenType == nethtml.EndTagToken {
			return nil, nil, fmt.Errorf("%w: unexpected closing tag %s", ErrInvalidIsland, tag)
		}
		binding, ok := bindings[tag]
		if !ok {
			return nil, nil, fmt.Errorf("%w: %s", ErrUnknownIsland, tag)
		}
		props, err := typedIslandProps(token.Attr, binding)
		if err != nil {
			return nil, nil, err
		}
		var fallback []string
		if tokenType == nethtml.StartTagToken {
			if binding.AllowFallback {
				fallback, err = consumeIslandFallback(tokenizer, tag)
			} else {
				err = consumeEmptyIsland(tokenizer, tag)
			}
			if err != nil {
				return nil, nil, err
			}
			fallbackSegments += len(fallback)
			for _, segment := range fallback {
				fallbackBytes += int64(len(segment))
			}
			if fallbackSegments > maxIslandFallbackSegments || fallbackBytes > maxIslandFallbackOutputBytes {
				return nil, nil, ErrOutputLimit
			}
		}
		islandIndex++
		descriptor := IslandDescriptor{
			ID: fmt.Sprintf("%s:%d", binding.ComponentID, islandIndex), ComponentID: binding.ComponentID,
			Props: props, FallbackHTMLSegments: fallback,
		}
		islands = append(islands, descriptor)
		fmt.Fprintf(&skeleton, `<template %s="%s"></template>`, islandPlaceholderAttribute, stdhtml.EscapeString(descriptor.ID))
	}
}

func consumeIslandFallback(tokenizer *nethtml.Tokenizer, tag string) ([]string, error) {
	var source strings.Builder
	for {
		tokenType := tokenizer.Next()
		if tokenType == nethtml.ErrorToken {
			return nil, fmt.Errorf("%w: unclosed %s", ErrInvalidIsland, tag)
		}
		raw := string(tokenizer.Raw())
		if tokenType == nethtml.StartTagToken || tokenType == nethtml.SelfClosingTagToken || tokenType == nethtml.EndTagToken {
			token := tokenizer.Token()
			currentTag := strings.ToLower(token.Data)
			if tokenType == nethtml.EndTagToken && currentTag == tag {
				segments, err := balanceHTMLSkeleton(source.String())
				if err != nil {
					return nil, err
				}
				result := make([]string, len(segments))
				for index, segment := range segments {
					result[index] = segment.String()
				}
				return result, nil
			}
			if strings.HasPrefix(currentTag, "sf-") || currentTag == "template" {
				return nil, fmt.Errorf("%w: %s fallback contains nested island %s", ErrInvalidIsland, tag, currentTag)
			}
			if tokenType != nethtml.EndTagToken {
				for _, attribute := range token.Attr {
					if strings.EqualFold(attribute.Key, islandPlaceholderAttribute) {
						return nil, fmt.Errorf("%w: %s fallback contains reserved island placeholder", ErrInvalidIsland, tag)
					}
				}
			}
		}
		source.WriteString(raw)
	}
}

func balanceHTMLSkeleton(source string) ([]RenderedHTML, error) {
	contextNode := &nethtml.Node{Type: nethtml.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := nethtml.ParseFragment(strings.NewReader(source), contextNode)
	if err != nil {
		return nil, fmt.Errorf("%w: parse rendered HTML: %v", ErrExecution, err)
	}
	segments := make([]RenderedHTML, 0, len(nodes))
	for _, node := range nodes {
		var output bytes.Buffer
		if err := nethtml.Render(&output, node); err != nil {
			return nil, fmt.Errorf("%w: serialize rendered HTML: %v", ErrExecution, err)
		}
		if output.Len() != 0 {
			segments = append(segments, RenderedHTML{value: output.String()})
		}
	}
	return segments, nil
}

func consumeEmptyIsland(tokenizer *nethtml.Tokenizer, tag string) error {
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case nethtml.ErrorToken:
			return fmt.Errorf("%w: unclosed %s", ErrInvalidIsland, tag)
		case nethtml.TextToken:
			if strings.TrimSpace(string(tokenizer.Raw())) != "" {
				return fmt.Errorf("%w: %s must not contain HTML or text", ErrInvalidIsland, tag)
			}
		case nethtml.CommentToken:
			continue
		case nethtml.EndTagToken:
			token := tokenizer.Token()
			if strings.EqualFold(token.Data, tag) {
				return nil
			}
			return fmt.Errorf("%w: mismatched closing tag %s", ErrInvalidIsland, token.Data)
		default:
			return fmt.Errorf("%w: %s must be empty", ErrInvalidIsland, tag)
		}
	}
}

func typedIslandProps(attributes []nethtml.Attribute, binding IslandBinding) ([]IslandProp, error) {
	contracts := make(map[string]IslandPropContract, len(binding.Props))
	for _, contract := range binding.Props {
		contracts[contract.Name] = contract
	}
	seen := make(map[string]struct{}, len(attributes))
	result := make([]IslandProp, 0, len(attributes))
	for _, attribute := range attributes {
		name := strings.ToLower(attribute.Key)
		contract, ok := contracts[name]
		if attribute.Namespace != "" || !ok {
			return nil, fmt.Errorf("%w: component %s rejects prop %q", ErrInvalidIsland, binding.ComponentID, name)
		}
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("%w: duplicate prop %q", ErrInvalidIsland, name)
		}
		seen[name] = struct{}{}
		prop, err := parseIslandProp(contract, attribute.Val)
		if err != nil {
			return nil, err
		}
		result = append(result, prop)
	}
	for _, contract := range binding.Props {
		if _, exists := seen[contract.Name]; contract.Required && !exists {
			return nil, fmt.Errorf("%w: component %s requires prop %q", ErrInvalidIsland, binding.ComponentID, contract.Name)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func parseIslandProp(contract IslandPropContract, value string) (IslandProp, error) {
	if len(value) > 4096 || strings.IndexByte(value, 0) >= 0 {
		return IslandProp{}, fmt.Errorf("%w: prop %q exceeds its value boundary", ErrInvalidIsland, contract.Name)
	}
	result := IslandProp{Name: contract.Name, Type: contract.Type}
	switch contract.Type {
	case IslandPropString:
		result.StringValue = strings.Clone(value)
	case IslandPropURL:
		if err := inspectURLAttribute(value); err != nil {
			return IslandProp{}, fmt.Errorf("%w: prop %q: %v", ErrInvalidIsland, contract.Name, err)
		}
		result.StringValue = strings.Clone(value)
	case IslandPropBoolean:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "", "true":
			result.BooleanValue = true
		case "false":
			result.BooleanValue = false
		default:
			return IslandProp{}, fmt.Errorf("%w: prop %q requires boolean", ErrInvalidIsland, contract.Name)
		}
	case IslandPropInteger:
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return IslandProp{}, fmt.Errorf("%w: prop %q requires integer", ErrInvalidIsland, contract.Name)
		}
		result.IntegerValue = parsed
	default:
		return IslandProp{}, fmt.Errorf("%w: prop %q has unknown type", ErrInvalidIsland, contract.Name)
	}
	return result, nil
}
