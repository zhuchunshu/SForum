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
	var builder strings.Builder
	for _, node := range doc.Content {
		renderNode(&builder, node, schema)
	}
	return builder.String()
}

func renderNode(builder *strings.Builder, node Node, schema Schema) {
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
			renderNode(builder, child, schema)
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
			renderNode(builder, child, schema)
		}
		builder.WriteString("</ul>")
	case "orderedList":
		builder.WriteString("<ol>")
		for _, child := range node.Content {
			renderNode(builder, child, schema)
		}
		builder.WriteString("</ol>")
	case "listItem":
		builder.WriteString("<li>")
		for _, child := range node.Content {
			renderNode(builder, child, schema)
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
			renderNode(builder, node, schema)
			continue
		}
		if node.Type != "text" {
			renderNode(builder, node, schema)
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

func renderMarkdownNode(builder *strings.Builder, node Node) {
	switch node.Type {
	case "paragraph":
		renderMarkdownInline(builder, node.Content)
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
