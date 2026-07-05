package forum

import (
	"strings"
	"testing"
)

func TestRenderMarkdownContentSanitizesHTML(t *testing.T) {
	rendered, err := RenderContent(ContentInput{
		RawContent:   "你好 **SForum** <script>alert(1)</script><a href=\"javascript:alert(1)\">坏链接</a>",
		SourceFormat: SourceFormatMarkdown,
		EditorType:   EditorTypeMarkdown,
	})
	if err != nil {
		t.Fatalf("RenderContent returned error: %v", err)
	}

	if !strings.Contains(rendered.HTMLContent, "<strong>SForum</strong>") {
		t.Fatalf("expected rendered markdown strong tag, got %q", rendered.HTMLContent)
	}
	if strings.Contains(rendered.HTMLContent, "<script") || strings.Contains(rendered.HTMLContent, "javascript:") {
		t.Fatalf("expected dangerous HTML to be removed, got %q", rendered.HTMLContent)
	}
	if rendered.PlainText == "" || strings.Contains(rendered.PlainText, "alert(1)") {
		t.Fatalf("expected safe plain text excerpt source, got %q", rendered.PlainText)
	}
	if rendered.Excerpt == "" {
		t.Fatal("expected excerpt to be generated")
	}
	if rendered.ContentHash == "" {
		t.Fatal("expected content hash to be generated")
	}
}

func TestRenderHTMLContentSanitizesRawHTML(t *testing.T) {
	rendered, err := RenderContent(ContentInput{
		RawContent:   `<p onclick="alert(1)">安全内容</p><iframe src="https://example.com"></iframe>`,
		SourceFormat: SourceFormatHTML,
		EditorType:   "richtext",
	})
	if err != nil {
		t.Fatalf("RenderContent returned error: %v", err)
	}

	if !strings.Contains(rendered.HTMLContent, "<p>安全内容</p>") {
		t.Fatalf("expected safe paragraph to remain, got %q", rendered.HTMLContent)
	}
	if strings.Contains(rendered.HTMLContent, "onclick") || strings.Contains(rendered.HTMLContent, "iframe") {
		t.Fatalf("expected unsafe HTML to be removed, got %q", rendered.HTMLContent)
	}
}

func TestRenderContentRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name  string
		input ContentInput
	}{
		{name: "empty raw content", input: ContentInput{RawContent: " ", SourceFormat: SourceFormatMarkdown}},
		{name: "unknown source format", input: ContentInput{RawContent: "hello", SourceFormat: "bbcode"}},
		{name: "json reserved for later", input: ContentInput{RawContent: `{"type":"doc"}`, SourceFormat: SourceFormatJSON}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := RenderContent(tc.input); err == nil {
				t.Fatal("expected RenderContent to reject invalid input")
			}
		})
	}
}
