package pages

import (
	"context"
	"errors"
	"testing"
)

func TestMergeCompositionHTMLSegmentsRetainsPrimary(t *testing.T) {
	merged := MergeCompositionHTMLSegments([]string{"<main>primary</main>"}, []string{"<aside>plugin</aside>"})
	if len(merged) != 2 || merged[0] != "<aside>plugin</aside>" || merged[1] != "<main>primary</main>" {
		t.Fatalf("merged = %#v", merged)
	}
	if got := MergeCompositionHTMLSegments([]string{"primary"}, nil); len(got) != 1 || got[0] != "primary" {
		t.Fatalf("nil composition = %#v", got)
	}
}

type stubPageComposition struct {
	html []string
	err  error
}

func (s stubPageComposition) ComposePageHTML(context.Context, string, map[string]any) ([]string, error) {
	return s.html, s.err
}

func TestApplyPageCompositionFailOpenAndSEOFence(t *testing.T) {
	page := ThemeRenderedPage{HTMLSegments: []string{"<main>core</main>"}}
	applied, err := ApplyPageComposition(
		context.Background(), page, stubPageComposition{html: []string{"<aside>x</aside>"}}, "forum.home", nil,
	)
	if err != nil || len(applied.HTMLSegments) != 2 || applied.HTMLSegments[1] != "<main>core</main>" {
		t.Fatalf("apply = %#v err=%v", applied, err)
	}

	open, err := ApplyPageComposition(
		context.Background(), page, stubPageComposition{err: errors.New("transient")}, "forum.home", nil,
	)
	if err != nil || len(open.HTMLSegments) != 1 || open.HTMLSegments[0] != "<main>core</main>" {
		t.Fatalf("fail-open = %#v err=%v", open, err)
	}

	if _, err := ApplyPageComposition(
		context.Background(), page, stubPageComposition{err: ErrPageCompositionSEO}, "forum.home", nil,
	); !errors.Is(err, ErrPageCompositionSEO) {
		t.Fatalf("SEO fence error=%v", err)
	}
}
