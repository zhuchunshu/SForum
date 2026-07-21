package forum

import (
	"context"
)

// ContentRegistryPostFilter adapts ContentRegistry's filter surface to the
// forum ContentPostFilter interface without importing ContentRegistry types
// into every call site (provider injects a concrete adapter).
type ContentRegistryPostFilter interface {
	AfterHostRender(ctx context.Context, html, plain, resource, resourceID string) (nextHTML, nextPlain string, err error)
}

// ContentRegistryBridge adapts ContentRegistryPostFilter → ContentPostFilter.
type ContentRegistryBridge struct {
	Inner ContentRegistryPostFilter
}

func (b ContentRegistryBridge) AfterHostRender(ctx context.Context, in ContentPostFilterInput) (RenderedContent, error) {
	if b.Inner == nil {
		return in.Rendered, nil
	}
	html, plain, err := b.Inner.AfterHostRender(
		ctx,
		in.Rendered.HTMLContent,
		in.Rendered.PlainText,
		in.Resource,
		in.ResourceID,
	)
	if err != nil {
		return RenderedContent{}, err
	}
	out := in.Rendered
	// 默认只允许替换 HTML；PlainText/Excerpt/ContentHash 保持 Host 权威，避免搜索漂移。
	if html != "" {
		out.HTMLContent = html
	}
	_ = plain
	return out, nil
}
