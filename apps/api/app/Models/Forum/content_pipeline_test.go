package forum

import (
	"context"
	"strings"
	"testing"
)

type stubContentPostFilter struct {
	marker string
	err    error
	calls  int
}

func (s *stubContentPostFilter) AfterHostRender(_ context.Context, in ContentPostFilterInput) (RenderedContent, error) {
	s.calls++
	if s.err != nil {
		return RenderedContent{}, s.err
	}
	out := in.Rendered
	out.HTMLContent = out.HTMLContent + s.marker
	return out, nil
}

func TestApplyContentPostFilterNilIsIdentity(t *testing.T) {
	t.Parallel()
	service := NewService(ServiceConfig{Store: nil})
	content := RenderedContent{HTMLContent: "<p>hi</p>", PlainText: "hi"}
	got, err := service.applyContentPostFilter(context.Background(), content, "topic", "new")
	if err != nil {
		t.Fatal(err)
	}
	if got.HTMLContent != content.HTMLContent || got.PlainText != content.PlainText {
		t.Fatalf("nil filter must be identity: %#v", got)
	}
}

func TestApplyContentPostFilterStubMutatesHTML(t *testing.T) {
	t.Parallel()
	stub := &stubContentPostFilter{marker: "<!--sf-filter-->"}
	service := NewService(ServiceConfig{Store: nil}).WithContentPostFilter(stub)
	content := RenderedContent{HTMLContent: "<p>body</p>", PlainText: "body"}
	got, err := service.applyContentPostFilter(context.Background(), content, "topic", "new")
	if err != nil {
		t.Fatal(err)
	}
	if stub.calls != 1 {
		t.Fatalf("calls = %d", stub.calls)
	}
	if !strings.HasSuffix(got.HTMLContent, "<!--sf-filter-->") {
		t.Fatalf("html = %q", got.HTMLContent)
	}
	if got.PlainText != "body" {
		t.Fatalf("plain must stay Host-owned: %q", got.PlainText)
	}
}

func TestApplyContentPostFilterRejectsEmptyHTMLOverwrite(t *testing.T) {
	t.Parallel()
	service := NewService(ServiceConfig{Store: nil}).WithContentPostFilter(ContentPostFilterFunc(func(_ context.Context, in ContentPostFilterInput) (RenderedContent, error) {
		out := in.Rendered
		out.HTMLContent = "   "
		return out, nil
	}))
	content := RenderedContent{HTMLContent: "<p>keep</p>", PlainText: "keep"}
	got, err := service.applyContentPostFilter(context.Background(), content, "comment", "1")
	if err != nil {
		t.Fatal(err)
	}
	if got.HTMLContent != "<p>keep</p>" {
		t.Fatalf("empty overwrite must keep host: %q", got.HTMLContent)
	}
}

// ContentPostFilterFunc adapts a function to ContentPostFilter for tests.
type ContentPostFilterFunc func(context.Context, ContentPostFilterInput) (RenderedContent, error)

func (f ContentPostFilterFunc) AfterHostRender(ctx context.Context, in ContentPostFilterInput) (RenderedContent, error) {
	return f(ctx, in)
}
