package editordocument

// CoreSchema returns the Host-owned Tiptap starter surface that mirrors
// createSFEditorExtensions without plugin L2 modules.
func CoreSchema() Schema {
	return Schema{
		Nodes: map[string]NodeSpec{
			"doc":        {},
			"paragraph":  {},
			"text":       {Atom: true},
			"heading":    {AllowAttrs: map[string]bool{"level": true}},
			"blockquote": {},
			"codeBlock":  {AllowAttrs: map[string]bool{"language": true}},
			"bulletList": {},
			"orderedList": {
				AllowAttrs: map[string]bool{"start": true},
			},
			"listItem":  {},
			"hardBreak": {Atom: true},
			"horizontalRule": {
				Atom: true, FallbackHTML: "<hr>",
			},
			"image": {
				Atom: true,
				AllowAttrs: map[string]bool{
					"src": true, "alt": true, "title": true,
					"width": true, "height": true, "displaySize": true,
					"attachmentId": true, "attachmentPublicId": true,
				},
				FallbackHTML: `<span class="sf-editor-fallback" data-fallback="image">[image]</span>`,
			},
			"sforumEmoji": {
				Atom: true,
				AllowAttrs: map[string]bool{
					"name": true, "label": true, "native": true,
				},
				FallbackHTML: `<span class="sf-editor-fallback" data-fallback="emoji">[emoji]</span>`,
			},
		},
		Marks: map[string]MarkSpec{
			"bold":          {},
			"italic":        {},
			"strike":        {},
			"code":          {},
			"underline":     {},
			"link":          {AllowAttrs: map[string]bool{"href": true, "target": true, "rel": true}},
			"demoHighlight": {AllowAttrs: map[string]bool{"color": true}}, // reserved test hook name
		},
	}
}

// MergeSchema adds plugin node/mark specs without replacing Core ownership of
// the same type names. Plugin names that collide with Core are ignored.
func MergeSchema(base Schema, nodes map[string]NodeSpec, marks map[string]MarkSpec) Schema {
	result := Schema{
		Nodes: map[string]NodeSpec{},
		Marks: map[string]MarkSpec{},
	}
	for name, spec := range base.Nodes {
		result.Nodes[name] = spec
	}
	for name, spec := range base.Marks {
		result.Marks[name] = spec
	}
	for name, spec := range nodes {
		if _, exists := result.Nodes[name]; exists {
			continue
		}
		result.Nodes[name] = spec
	}
	for name, spec := range marks {
		if _, exists := result.Marks[name]; exists {
			continue
		}
		result.Marks[name] = spec
	}
	return result
}

// SchemaFromEditorCatalog builds NodeSpec/MarkSpec entries from Editor Registry
// contributions. Unknown plugin types fall back to atom placeholders.
func SchemaFromEditorNames(nodeNames, markNames []string) Schema {
	nodes := map[string]NodeSpec{}
	marks := map[string]MarkSpec{}
	for _, name := range nodeNames {
		if name == "" {
			continue
		}
		nodes[name] = NodeSpec{
			Atom: true,
			FallbackHTML: `<span class="sf-editor-fallback" data-fallback="` + htmlAttrEscape(name) + `">[` +
				htmlTextEscape(name) + `]</span>`,
		}
	}
	for _, name := range markNames {
		if name == "" {
			continue
		}
		marks[name] = MarkSpec{}
	}
	return MergeSchema(CoreSchema(), nodes, marks)
}
