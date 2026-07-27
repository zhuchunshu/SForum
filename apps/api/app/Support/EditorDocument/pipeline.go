package editordocument

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
)

const (
	maxDocumentBytes    = 1 << 20 // 1 MiB native JSON
	maxNodeCount        = 20000
	maxDepth            = 64
	defaultExcerptRunes = 160
	maxExcerptRunes     = 500
)

// Accept runs the ordered parse→validate→normalize→render→sanitize pipeline
// and returns the Host storage triple. Markdown is derived from accepted native
// structure; client HTML is never trusted.
func Accept(input Input) (Accepted, error) {
	doc, err := Parse(input)
	if err != nil {
		return Accepted{}, fmt.Errorf("%w: %v", ErrPipeline, err)
	}
	schema := input.Schema
	if len(schema.Nodes) == 0 {
		schema = CoreSchema()
	}
	normalized, fallbacks, err := ValidateAndNormalize(doc, schema)
	if err != nil {
		return Accepted{}, fmt.Errorf("%w: %v", ErrPipeline, err)
	}
	htmlRaw := RenderHTML(normalized, schema)
	htmlSanitized := SanitizeHTML(htmlRaw)
	plain := normalizePlainText(htmlToPlain(htmlSanitized))
	if plain == "" && len(normalized.Content) == 0 {
		return Accepted{}, ErrInvalid
	}
	markdown := RenderMarkdown(normalized)
	search := buildSearchText(plain, normalized)
	excerptLimit := input.ExcerptLimit
	if excerptLimit <= 0 || excerptLimit > maxExcerptRunes {
		excerptLimit = defaultExcerptRunes
	}
	nativeBytes, _ := json.Marshal(normalized)
	hash := sha256.Sum256(append(append([]byte(StorageVersion), 0), nativeBytes...))
	return Accepted{
		StorageVersion: StorageVersion,
		Native:         normalized,
		Markdown:       markdown,
		HTMLSanitized:  htmlSanitized,
		PlainText:      plain,
		Excerpt:        excerptRunes(plain, excerptLimit),
		SearchText:     search,
		ContentHash:    hex.EncodeToString(hash[:]),
		Fallbacks:      fallbacks,
	}, nil
}

// Parse decodes native JSON or builds a minimal paragraph doc from Markdown.
func Parse(input Input) (Document, error) {
	if len(input.NativeJSON) > maxDocumentBytes {
		return Document{}, ErrInvalid
	}
	if len(input.NativeJSON) > 0 {
		var doc Document
		if err := json.Unmarshal(input.NativeJSON, &doc); err != nil {
			return Document{}, ErrInvalid
		}
		if strings.TrimSpace(doc.Type) == "" {
			doc.Type = "doc"
		}
		return doc, nil
	}
	markdown := strings.TrimSpace(input.Markdown)
	if markdown == "" {
		return Document{}, ErrInvalid
	}
	// Compatibility path: each non-empty line becomes a paragraph of text.
	// Full markdown→tiptap conversion remains outside this Host contract.
	var content []Node
	for _, line := range strings.Split(markdown, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		content = append(content, Node{
			Type: "paragraph",
			Content: []Node{{
				Type: "text", Text: line,
			}},
		})
	}
	if len(content) == 0 {
		return Document{}, ErrInvalid
	}
	return Document{Type: "doc", Content: content}, nil
}

// ValidateAndNormalize allowlists types/attrs, rewrites unsafe URLs, and
// replaces unknown nodes with stable fallbacks while preserving structure
// metadata for later re-enable.
func ValidateAndNormalize(doc Document, schema Schema) (Document, []string, error) {
	if doc.Type != "doc" {
		return Document{}, nil, ErrInvalid
	}
	var fallbacks []string
	var nodeCount int
	content, err := normalizeNodes(doc.Content, schema, 1, &nodeCount, &fallbacks)
	if err != nil {
		return Document{}, nil, err
	}
	return Document{Type: "doc", Content: content}, uniqueSorted(fallbacks), nil
}

func normalizeNodes(nodes []Node, schema Schema, depth int, nodeCount *int, fallbacks *[]string) ([]Node, error) {
	if depth > maxDepth {
		return nil, ErrInvalid
	}
	result := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		*nodeCount++
		if *nodeCount > maxNodeCount {
			return nil, ErrInvalid
		}
		spec, ok := schema.Nodes[node.Type]
		if !ok {
			*fallbacks = append(*fallbacks, node.Type)
			// Preserve source type under data-fallback so re-enable can migrate.
			result = append(result, Node{
				Type: "paragraph",
				Content: []Node{{
					Type: "text",
					Text: "[" + node.Type + "]",
				}},
				Attrs: map[string]any{"data-fallback-for": node.Type},
			})
			continue
		}
		normalized := Node{
			Type:  node.Type,
			Text:  node.Text,
			Attrs: filterAttrs(node.Attrs, spec.AllowAttrs),
		}
		if node.Type == "text" {
			normalized.Text = node.Text
			if marks, err := normalizeMarks(node.Marks, schema); err != nil {
				return nil, err
			} else {
				normalized.Marks = marks
			}
			result = append(result, normalized)
			continue
		}
		if node.Type == "image" {
			src, _ := normalized.Attrs["src"].(string)
			if clean := allowURL(src, true); clean == "" {
				*fallbacks = append(*fallbacks, "image")
				result = append(result, Node{Type: "paragraph", Content: []Node{{Type: "text", Text: "[image]"}}})
				continue
			} else {
				normalized.Attrs["src"] = clean
			}
		}
		if !spec.Atom {
			children, err := normalizeNodes(node.Content, schema, depth+1, nodeCount, fallbacks)
			if err != nil {
				return nil, err
			}
			normalized.Content = children
		}
		if marks, err := normalizeMarks(node.Marks, schema); err != nil {
			return nil, err
		} else {
			normalized.Marks = marks
		}
		result = append(result, normalized)
	}
	return result, nil
}

func normalizeMarks(marks []Mark, schema Schema) ([]Mark, error) {
	if len(marks) == 0 {
		return nil, nil
	}
	result := make([]Mark, 0, len(marks))
	for _, mark := range marks {
		spec, ok := schema.Marks[mark.Type]
		if !ok {
			// Drop unknown marks; text remains.
			continue
		}
		attrs := filterAttrs(mark.Attrs, spec.AllowAttrs)
		if mark.Type == "link" {
			href, _ := attrs["href"].(string)
			if clean := allowURL(href, false); clean == "" {
				continue
			} else {
				attrs["href"] = clean
				attrs["rel"] = "noopener noreferrer nofollow ugc"
				attrs["target"] = "_blank"
			}
		}
		result = append(result, Mark{Type: mark.Type, Attrs: attrs})
	}
	return result, nil
}

func filterAttrs(input map[string]any, allow map[string]bool) map[string]any {
	if len(input) == 0 || len(allow) == 0 {
		return nil
	}
	result := map[string]any{}
	for key, value := range input {
		if !allow[key] {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.ContainsRune(typed, '\x00') {
				continue
			}
			result[key] = typed
		case float64:
			result[key] = typed
		case bool:
			result[key] = typed
		case int:
			result[key] = typed
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func allowURL(raw string, image bool) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return value
	}
	if strings.HasPrefix(value, "#") && !image {
		return value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return parsed.String()
	case "mailto":
		if image {
			return ""
		}
		return parsed.String()
	default:
		return ""
	}
}

// MigrateStorage upgrades an Accepted document when StorageVersion differs.
// Unknown future versions fail closed; same version is a no-op.
func MigrateStorage(accepted Accepted, targetVersion string, schema Schema) (Accepted, error) {
	if targetVersion == "" {
		targetVersion = StorageVersion
	}
	if accepted.StorageVersion == targetVersion {
		return accepted, nil
	}
	// Only the current Host version is supported as a migration target.
	if targetVersion != StorageVersion {
		return Accepted{}, ErrStorageVersion
	}
	// Re-accept native JSON under the current schema (strips retired nodes).
	body, err := json.Marshal(accepted.Native)
	if err != nil {
		return Accepted{}, ErrInvalid
	}
	return Accept(Input{NativeJSON: body, Schema: schema, ExcerptLimit: defaultExcerptRunes})
}

func uniqueSorted(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func excerptRunes(plain string, limit int) string {
	plain = strings.TrimSpace(plain)
	if plain == "" || limit <= 0 {
		return ""
	}
	if utf8.RuneCountInString(plain) <= limit {
		return plain
	}
	runes := []rune(plain)
	return string(runes[:limit]) + "…"
}

func buildSearchText(plain string, doc Document) string {
	// Search prefers plain text; append stable custom-node tokens for indexing.
	var extra []string
	var walk func([]Node)
	walk = func(nodes []Node) {
		for _, node := range nodes {
			if node.Type == "sforumEmoji" {
				if name, _ := node.Attrs["name"].(string); name != "" {
					extra = append(extra, "emoji:"+name)
				}
			}
			walk(node.Content)
		}
	}
	walk(doc.Content)
	if len(extra) == 0 {
		return plain
	}
	return strings.TrimSpace(plain + " " + strings.Join(extra, " "))
}

func SanitizeHTML(value string) string {
	policy := bluemonday.UGCPolicy()
	policy.RequireNoFollowOnLinks(true)
	policy.RequireNoReferrerOnLinks(true)
	policy.AllowAttrs("class").OnElements("span", "code", "pre")
	policy.AllowAttrs("data-sforum-emoji", "data-label", "data-fallback", "data-fallback-for", "title").OnElements("span")
	policy.AllowAttrs("loading", "decoding", "referrerpolicy").OnElements("img")
	return policy.Sanitize(value)
}
