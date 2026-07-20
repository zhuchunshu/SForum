package pages

import (
	"context"
	"errors"
)

// ErrPageCompositionSEO is returned when composition would strip primary SSR
// content. Callers must fail closed rather than emit an incomplete page.
var ErrPageCompositionSEO = errors.New("pages: component composition removed primary SSR content")

// PageCompositionRenderer is an optional Host composition seam. Theme L1 render
// remains authoritative; composition only prefaces/appends Host-sanitized HTML.
type PageCompositionRenderer interface {
	// ComposePageHTML returns additional Host-sanitized HTML segments for a
	// Page Registry id (e.g. forum.home). Empty slice means no-op.
	ComposePageHTML(ctx context.Context, pageID string, props map[string]any) ([]string, error)
}

// MergeCompositionHTMLSegments prefaces composition HTML while retaining every
// primary theme segment. Composition must never remove primary content here.
func MergeCompositionHTMLSegments(primary []string, composition []string) []string {
	if len(composition) == 0 {
		return append([]string(nil), primary...)
	}
	if len(primary) == 0 {
		return append([]string(nil), composition...)
	}
	result := make([]string, 0, len(composition)+len(primary))
	result = append(result, composition...)
	result = append(result, primary...)
	return result
}

// ApplyPageComposition merges optional composition HTML into a rendered page.
// Fail-open: non-SEO errors leave the theme output unchanged. SEO fence errors
// propagate so the caller can fail closed.
func ApplyPageComposition(
	ctx context.Context,
	page ThemeRenderedPage,
	renderer PageCompositionRenderer,
	pageID string,
	props map[string]any,
) (ThemeRenderedPage, error) {
	if renderer == nil || ctx == nil {
		return page, nil
	}
	composition, err := renderer.ComposePageHTML(ctx, pageID, props)
	if err != nil {
		if errors.Is(err, ErrPageCompositionSEO) {
			return ThemeRenderedPage{}, err
		}
		// 可选 composition 不得破坏 Core/主题主渲染。
		return page, nil
	}
	if len(composition) == 0 {
		return page, nil
	}
	page.HTMLSegments = MergeCompositionHTMLSegments(page.HTMLSegments, composition)
	return page, nil
}
