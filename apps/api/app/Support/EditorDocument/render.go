package editordocument

import (
	"html"
	"strconv"
	"strings"

	nethtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// RenderHTML walks accepted native JSON into HTML before sanitization.
func RenderHTML(doc Document, schema Schema) string {
	return RenderHTMLForLocale(doc, schema, "")
}

// RenderHTMLForLocale renders Host-owned fallback copy in locale. Stored
// public projections call this with the site's current default locale.
func RenderHTMLForLocale(doc Document, schema Schema, locale string) string {
	var builder strings.Builder
	for _, node := range doc.Content {
		renderNode(&builder, node, schema, nil, 0, locale)
	}
	return builder.String()
}

// ShortcodeRenderSlot is a public-safe render-plan entry. Reference nodes are
// atoms, so the copied node can contain only the frozen identity and arguments.
type ShortcodeRenderSlot struct {
	Placeholder string
	Node        Node
	Depth       int
}

// ProtectedShortcodeFragment is request-only accepted source. It must never be
// logged, traced, hashed, or stored in a shared cache.
type ProtectedShortcodeFragment struct {
	Placeholder     string
	ID              string
	ContractVersion string
	Depth           int
	Document        Document
}

type shortcodeRenderPlanState struct {
	slots            []ShortcodeRenderSlot
	protected        []ProtectedShortcodeFragment
	includeProtected bool
	locale           string
}

// RenderHTMLWithShortcodeSlots renders accepted Host nodes while replacing
// public reference atoms with opaque server-side placeholders. Protected block
// nodes keep their closed Host fallback until the actor-sensitive M8 pipeline.
func RenderHTMLWithShortcodeSlots(doc Document, schema Schema) (string, []ShortcodeRenderSlot) {
	var builder strings.Builder
	state := &shortcodeRenderPlanState{}
	for _, node := range doc.Content {
		renderNode(&builder, node, schema, state, 0, "")
	}
	return builder.String(), append([]ShortcodeRenderSlot(nil), state.slots...)
}

// RenderHTMLWithAllShortcodeSlots produces a public-safe template and keeps
// protected children in a separate request-only slice. Protected slot metadata
// contains no child, arguments, actor, output, or source-derived hash.
func RenderHTMLWithAllShortcodeSlots(doc Document, schema Schema, locale string) (string, []ShortcodeRenderSlot, []ProtectedShortcodeFragment) {
	var builder strings.Builder
	state := &shortcodeRenderPlanState{includeProtected: true, locale: locale}
	for _, node := range doc.Content {
		renderNode(&builder, node, schema, state, 0, locale)
	}
	return builder.String(), append([]ShortcodeRenderSlot(nil), state.slots...), append([]ProtectedShortcodeFragment(nil), state.protected...)
}

// RenderProtectedShortcodeFallbackHTML is lifecycle-independent Host output.
// It never includes protected child content or declaration arguments.
func RenderProtectedShortcodeFallbackHTML(locale string) string {
	return RenderProtectedShortcodeFallbackHTMLForCode("shortcode.protected.unavailable", locale)
}

// RenderProtectedShortcodeFallbackHTMLForCode accepts only stable Host reason
// codes. Unknown input is collapsed to the generic closed fallback.
func RenderProtectedShortcodeFallbackHTMLForCode(code, locale string) string {
	switch code {
	case "shortcode.protected.unavailable", "shortcode.login.required", "shortcode.reply.required", "shortcode.only_author.private":
	default:
		code = "shortcode.protected.unavailable"
	}
	label := shortcodeFallbackLabel(code, locale)
	return `<span class="sf-editor-fallback" data-fallback="` + html.EscapeString(code) + `">` + html.EscapeString(label) + `</span>`
}

// RenderShortcodeFallbackHTML renders one accepted node through the Host
// fallback path. It is used only to close unresolved public reference slots.
func RenderShortcodeFallbackHTML(node Node, schema Schema, locale string) string {
	return RenderHTMLForLocale(Document{Type: "doc", Content: []Node{node}}, schema, locale)
}

func renderNode(builder *strings.Builder, node Node, schema Schema, plan *shortcodeRenderPlanState, shortcodeDepth int, locale string) {
	switch node.Type {
	case "paragraph":
		builder.WriteString("<p>")
		renderInline(builder, node.Content, schema)
		builder.WriteString("</p>")
	case "heading":
		level := 2
		if raw, ok := node.Attrs["level"].(float64); ok {
			level = int(raw)
		}
		if level < 1 {
			level = 1
		}
		if level > 6 {
			level = 6
		}
		tag := "h" + strconv.Itoa(level)
		builder.WriteString("<" + tag + ">")
		renderInline(builder, node.Content, schema)
		builder.WriteString("</" + tag + ">")
	case "blockquote":
		builder.WriteString("<blockquote>")
		for _, child := range node.Content {
			renderNode(builder, child, schema, plan, shortcodeDepth, locale)
		}
		builder.WriteString("</blockquote>")
	case "codeBlock":
		lang, _ := node.Attrs["language"].(string)
		builder.WriteString("<pre><code")
		if lang != "" {
			builder.WriteString(` class="language-` + html.EscapeString(lang) + `"`)
		}
		builder.WriteString(">")
		for _, child := range node.Content {
			if child.Type == "text" {
				builder.WriteString(html.EscapeString(child.Text))
			}
		}
		builder.WriteString("</code></pre>")
	case "bulletList":
		builder.WriteString("<ul>")
		for _, child := range node.Content {
			renderNode(builder, child, schema, plan, shortcodeDepth, locale)
		}
		builder.WriteString("</ul>")
	case "orderedList":
		start := 1
		if normalized, ok := normalizedOrderedListStart(node.Attrs["start"]); ok {
			start = normalized
		}
		if start != 1 {
			builder.WriteString(`<ol start="` + strconv.Itoa(start) + `">`)
		} else {
			builder.WriteString("<ol>")
		}
		for _, child := range node.Content {
			renderNode(builder, child, schema, plan, shortcodeDepth, locale)
		}
		builder.WriteString("</ol>")
	case "listItem":
		builder.WriteString("<li>")
		for _, child := range node.Content {
			renderNode(builder, child, schema, plan, shortcodeDepth, locale)
		}
		builder.WriteString("</li>")
	case "horizontalRule":
		builder.WriteString("<hr>")
	case "hardBreak":
		builder.WriteString("<br>")
	case "image":
		src, _ := node.Attrs["src"].(string)
		alt, _ := node.Attrs["alt"].(string)
		displaySize, _ := node.Attrs["displaySize"].(string)
		if _, ok := imageDisplaySizes[displaySize]; !ok {
			displaySize = "standard"
		}
		width, widthOK := normalizedImageDimension(node.Attrs["width"])
		height, heightOK := normalizedImageDimension(node.Attrs["height"])
		viewerSrc := src
		if publicID, ok := node.Attrs["attachmentPublicId"].(string); ok && safeMediaPublicID(publicID) {
			viewerSrc = "/media/attachments/" + publicID + "/original"
		}

		builder.WriteString(`<a href="` + html.EscapeString(viewerSrc) + `" class="sf-content-image-link" data-sforum-image-viewer="1" data-sforum-image-size="` + displaySize + `" target="_blank" rel="noopener noreferrer">`)
		builder.WriteString(`<img src="` + html.EscapeString(src) + `" alt="` + html.EscapeString(alt) + `" data-sforum-image-size="` + displaySize + `"`)
		if widthOK && heightOK {
			builder.WriteString(` width="` + strconv.Itoa(width) + `" height="` + strconv.Itoa(height) + `"`)
			if height >= width*5/2 {
				builder.WriteString(` data-sforum-image-long="1"`)
			}
		}
		builder.WriteString(` loading="lazy" decoding="async" referrerpolicy="no-referrer"></a>`)
	case "sforumEmoji":
		name, _ := node.Attrs["name"].(string)
		label, _ := node.Attrs["label"].(string)
		native, _ := node.Attrs["native"].(string)
		if native == "" {
			native = ":" + name + ":"
		}
		builder.WriteString(`<span class="sf-editor-emoji-node" data-sforum-emoji="` + html.EscapeString(name) +
			`" data-label="` + html.EscapeString(label) + `" title="` + html.EscapeString(label) + `">` +
			html.EscapeString(native) + `</span>`)
	case ShortcodeRefNode:
		if plan != nil {
			placeholder := "<!--sforum-shortcode-slot:" + strconv.Itoa(len(plan.slots)) + "-->"
			plan.slots = append(plan.slots, ShortcodeRenderSlot{
				Placeholder: placeholder,
				Node:        cloneShortcodeRenderNode(node),
				Depth:       shortcodeDepth + 1,
			})
			builder.WriteString(placeholder)
			return
		}
		code, label, omit := shortcodeFallbackForLocale(node, locale)
		if omit {
			return
		}
		builder.WriteString(`<span class="sf-editor-fallback" data-fallback="` + html.EscapeString(code) + `">`)
		builder.WriteString(html.EscapeString(label))
		builder.WriteString("</span>")
	case ShortcodeBlockNode:
		if plan != nil && plan.includeProtected {
			placeholder := "<!--sforum-protected-slot:" + strconv.Itoa(len(plan.protected)) + "-->"
			id, _ := node.Attrs["id"].(string)
			version, _ := node.Attrs["contractVersion"].(string)
			plan.protected = append(plan.protected, ProtectedShortcodeFragment{
				Placeholder: placeholder, ID: id, ContractVersion: version,
				Depth:    shortcodeDepth + 1,
				Document: Document{Type: "doc", Content: cloneNodes(node.Content)},
			})
			builder.WriteString(placeholder)
			return
		}
		code, label, omit := shortcodeFallbackForLocale(node, locale)
		if omit {
			return
		}
		builder.WriteString(`<span class="sf-editor-fallback" data-fallback="` + html.EscapeString(code) + `">`)
		builder.WriteString(html.EscapeString(label))
		builder.WriteString("</span>")
	default:
		if spec, ok := schema.Nodes[node.Type]; ok && spec.FallbackHTML != "" {
			builder.WriteString(spec.FallbackHTML)
			return
		}
		builder.WriteString("<p>")
		renderInline(builder, node.Content, schema)
		builder.WriteString("</p>")
	}
}

func safeMediaPublicID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '_' || character == '-' {
			continue
		}
		return false
	}
	return true
}

func renderInline(builder *strings.Builder, nodes []Node, schema Schema) {
	for _, node := range nodes {
		if node.Type == "hardBreak" {
			builder.WriteString("<br>")
			continue
		}
		if node.Type == "sforumEmoji" || node.Type == "image" {
			renderNode(builder, node, schema, nil, 0, "")
			continue
		}
		if node.Type != "text" {
			renderNode(builder, node, schema, nil, 0, "")
			continue
		}
		text := html.EscapeString(node.Text)
		// Apply marks outer-to-inner for stable nesting.
		for i := len(node.Marks) - 1; i >= 0; i-- {
			mark := node.Marks[i]
			switch mark.Type {
			case "bold":
				text = "<strong>" + text + "</strong>"
			case "italic":
				text = "<em>" + text + "</em>"
			case "strike":
				text = "<s>" + text + "</s>"
			case "code":
				text = "<code>" + text + "</code>"
			case "underline":
				text = "<u>" + text + "</u>"
			case "link":
				href, _ := mark.Attrs["href"].(string)
				text = `<a href="` + html.EscapeString(href) + `" rel="noopener noreferrer nofollow ugc" target="_blank">` + text + `</a>`
			default:
				// Unknown marks already stripped in normalize.
			}
		}
		builder.WriteString(text)
	}
}

func cloneShortcodeRenderNode(node Node) Node {
	clone := Node{Type: node.Type, Text: node.Text}
	if node.Attrs != nil {
		clone.Attrs = make(map[string]any, len(node.Attrs))
		for key, value := range node.Attrs {
			if arguments, ok := value.(map[string]any); ok {
				copied := make(map[string]any, len(arguments))
				for argument, scalar := range arguments {
					copied[argument] = scalar
				}
				clone.Attrs[key] = copied
				continue
			}
			clone.Attrs[key] = value
		}
	}
	return clone
}

func cloneNodes(nodes []Node) []Node {
	result := make([]Node, len(nodes))
	for index, node := range nodes {
		result[index] = cloneShortcodeRenderNode(node)
		result[index].Content = cloneNodes(node.Content)
		if node.Marks != nil {
			result[index].Marks = make([]Mark, len(node.Marks))
			for markIndex, mark := range node.Marks {
				result[index].Marks[markIndex] = Mark{Type: mark.Type}
				if mark.Attrs != nil {
					result[index].Marks[markIndex].Attrs = make(map[string]any, len(mark.Attrs))
					for key, value := range mark.Attrs {
						result[index].Marks[markIndex].Attrs[key] = value
					}
				}
			}
		}
	}
	return result
}

// RenderMarkdown produces a lossy but readable Markdown export from accepted
// native structure for audits and editable source fallback.
func RenderMarkdown(doc Document) string {
	var builder strings.Builder
	for index, node := range doc.Content {
		if index > 0 {
			builder.WriteString("\n\n")
		}
		renderMarkdownNode(&builder, node)
	}
	return strings.TrimSpace(builder.String())
}

// RenderPublicSideEffectMarkdown preserves Markdown code/link semantics for
// existing authoring analyzers while removing protected descendants first.
// It is for write-time mentions/previews only, never for editable projection.
func RenderPublicSideEffectMarkdown(doc Document) string {
	return RenderMarkdown(Document{Type: "doc", Content: publicSideEffectNodes(doc.Content)})
}

func publicSideEffectNodes(nodes []Node) []Node {
	result := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		if node.Type == ShortcodeBlockNode {
			_, label, _ := shortcodeFallback(node)
			result = append(result, Node{Type: "paragraph", Content: []Node{{Type: "text", Text: label}}})
			continue
		}
		if node.Type == ShortcodeRefNode {
			_, label, omit := shortcodeFallback(node)
			if !omit {
				result = append(result, Node{Type: "paragraph", Content: []Node{{Type: "text", Text: label}}})
			}
			continue
		}
		node.Content = publicSideEffectNodes(node.Content)
		result = append(result, node)
	}
	return result
}

func renderMarkdownNode(builder *strings.Builder, node Node) {
	switch node.Type {
	case "paragraph":
		var paragraph strings.Builder
		renderMarkdownInline(&paragraph, node.Content)
		builder.WriteString(escapeLiteralShortcodeOpening(paragraph.String()))
	case "heading":
		level := 2
		if raw, ok := node.Attrs["level"].(float64); ok {
			level = int(raw)
		}
		if level < 1 {
			level = 1
		}
		if level > 6 {
			level = 6
		}
		builder.WriteString(strings.Repeat("#", level))
		builder.WriteByte(' ')
		renderMarkdownInline(builder, node.Content)
	case "blockquote":
		var inner strings.Builder
		for _, child := range node.Content {
			renderMarkdownNode(&inner, child)
			inner.WriteByte('\n')
		}
		for _, line := range strings.Split(strings.TrimSpace(inner.String()), "\n") {
			builder.WriteString("> ")
			builder.WriteString(line)
			builder.WriteByte('\n')
		}
	case "codeBlock":
		lang, _ := node.Attrs["language"].(string)
		builder.WriteString("```")
		builder.WriteString(lang)
		builder.WriteByte('\n')
		for _, child := range node.Content {
			if child.Type == "text" {
				builder.WriteString(child.Text)
			}
		}
		builder.WriteString("\n```")
	case "bulletList", "orderedList":
		for i, child := range node.Content {
			if child.Type != "listItem" {
				continue
			}
			if node.Type == "orderedList" {
				builder.WriteString(strconv.Itoa(i+1) + ". ")
			} else {
				builder.WriteString("- ")
			}
			renderMarkdownInline(builder, flattenListItem(child))
			builder.WriteByte('\n')
		}
	case "horizontalRule":
		builder.WriteString("---")
	case "image":
		src, _ := node.Attrs["src"].(string)
		alt, _ := node.Attrs["alt"].(string)
		builder.WriteString("![")
		builder.WriteString(alt)
		builder.WriteString("](")
		builder.WriteString(src)
		builder.WriteString(")")
	case ShortcodeRefNode:
		if canonical, ok := shortcodeCanonicalMarkdown(node); ok {
			builder.WriteString(canonical)
		}
	case ShortcodeBlockNode:
		canonical, ok := shortcodeCanonicalMarkdown(node)
		name, nameOK := shortcodeName(node)
		if !ok || !nameOK {
			return
		}
		builder.WriteString(canonical)
		builder.WriteByte('\n')
		builder.WriteString(RenderMarkdown(Document{Type: "doc", Content: node.Content}))
		builder.WriteString("\n[/")
		builder.WriteString(name)
		builder.WriteByte(']')
	default:
		renderMarkdownInline(builder, node.Content)
	}
}

func flattenListItem(node Node) []Node {
	var result []Node
	for _, child := range node.Content {
		if child.Type == "paragraph" {
			result = append(result, child.Content...)
		} else {
			result = append(result, child)
		}
	}
	return result
}

func renderMarkdownInline(builder *strings.Builder, nodes []Node) {
	for _, node := range nodes {
		if node.Type == "hardBreak" {
			builder.WriteString("  \n")
			continue
		}
		if node.Type == "sforumEmoji" {
			native, _ := node.Attrs["native"].(string)
			if native == "" {
				name, _ := node.Attrs["name"].(string)
				native = ":" + name + ":"
			}
			builder.WriteString(native)
			continue
		}
		if node.Type != "text" {
			continue
		}
		text := node.Text
		for _, mark := range node.Marks {
			switch mark.Type {
			case "bold":
				text = "**" + text + "**"
			case "italic":
				text = "*" + text + "*"
			case "strike":
				text = "~~" + text + "~~"
			case "code":
				text = "`" + text + "`"
			case "link":
				href, _ := mark.Attrs["href"].(string)
				text = "[" + text + "](" + href + ")"
			}
		}
		builder.WriteString(text)
	}
}

func htmlToPlain(value string) string {
	node, err := nethtml.Parse(strings.NewReader(value))
	if err != nil {
		return value
	}
	var builder strings.Builder
	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		if n.Type == nethtml.TextNode {
			builder.WriteString(n.Data)
			builder.WriteByte(' ')
		}
		// Prefer semantic breaks between blocks.
		if n.Type == nethtml.ElementNode {
			switch n.DataAtom {
			case atom.P, atom.Div, atom.Br, atom.Li, atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
				builder.WriteByte(' ')
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return builder.String()
}

func normalizePlainText(value string) string {
	fields := strings.Fields(value)
	return strings.Join(fields, " ")
}

func htmlAttrEscape(value string) string {
	return html.EscapeString(value)
}

func htmlTextEscape(value string) string {
	return html.EscapeString(value)
}
