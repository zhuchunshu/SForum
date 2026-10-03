package forum

type RenderedContent struct {
	ID            int64  `json:"id"`
	RawContent    string `json:"rawContent"`
	HTMLContent   string `json:"htmlContent"`
	PlainText     string `json:"plainText"`
	Excerpt       string `json:"excerpt"`
	SourceFormat  string `json:"sourceFormat"`
	EditorType    string `json:"editorType"`
	EditorVersion string `json:"editorVersion"`
	RenderVersion string `json:"renderVersion"`
	ContentHash   string `json:"contentHash"`
}

// PublicRenderedContent is the only content projection allowed on public
// topic/comment surfaces. Canonical source and source-derived metadata stay
// behind edit-source, revision, or admin authorization boundaries.
type PublicRenderedContent struct {
	ID            int64  `json:"id"`
	HTMLContent   string `json:"htmlContent"`
	PlainText     string `json:"plainText"`
	Excerpt       string `json:"excerpt"`
	RenderVersion string `json:"renderVersion"`
}

// EditableContentSource contains the current canonical source needed to
// re-open an editor. It must never be embedded in public DTOs or public cache
// entries.
type EditableContentSource struct {
	RawContent      string  `json:"rawContent"`
	SourceFormat    string  `json:"sourceFormat"`
	EditorType      string  `json:"editorType"`
	EditorVersion   string  `json:"editorVersion"`
	ContentHash     string  `json:"contentHash"`
	AttachmentIDs   []int64 `json:"attachmentIds"`
	CurrentRevision int64   `json:"currentRevision"`
}

func ToPublicRenderedContent(content RenderedContent) PublicRenderedContent {
	return PublicRenderedContent{
		ID:            content.ID,
		HTMLContent:   content.HTMLContent,
		PlainText:     content.PlainText,
		Excerpt:       content.Excerpt,
		RenderVersion: content.RenderVersion,
	}
}
