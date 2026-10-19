package forum

import (
	"strings"
	"testing"
)

// TestRenderMarkdownTableKeepsAlignAttribute 验证 legacy markdown 表格对齐在
// 净化后仍以 align 保留（bluemonday 会剥掉 style）。
func TestRenderMarkdownTableKeepsAlignAttribute(t *testing.T) {
	rendered, err := RenderContent(ContentInput{
		RawContent:   "| 左 | 中 | 右 |\n| :--- | :---: | ---: |\n| a | b | c |\n",
		SourceFormat: SourceFormatMarkdown,
		EditorType:   EditorTypeMarkdown,
	})
	if err != nil {
		t.Fatalf("RenderContent returned error: %v", err)
	}
	for _, want := range []string{
		`<th align="left">`, `<th align="center">`, `<th align="right">`,
		`<td align="left">`, `<td align="center">`, `<td align="right">`,
	} {
		if !strings.Contains(rendered.HTMLContent, want) {
			t.Fatalf("expected %q, got %q", want, rendered.HTMLContent)
		}
	}
	if strings.Contains(rendered.HTMLContent, "style=") {
		t.Fatalf("expected align attribute instead of style, got %q", rendered.HTMLContent)
	}
}
