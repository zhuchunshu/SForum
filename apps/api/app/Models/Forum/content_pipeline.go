package forum

import (
	"context"
	"strings"
)

// ContentPostFilter is an optional Host seam after authoritative RenderContent.
// Nil implementations and empty Content Registry graphs MUST leave content
// unchanged so ordinary markdown / editor-document posts never fail because a
// plugin surface is incomplete.
type ContentPostFilter interface {
	AfterHostRender(ctx context.Context, in ContentPostFilterInput) (RenderedContent, error)
}

// ContentPostFilterInput carries Host-rendered content plus composition context.
type ContentPostFilterInput struct {
	Rendered   RenderedContent
	Resource   string // "topic" | "comment"
	ResourceID string // "new" or numeric id when known
	Scope      string // "public" | "author" | ...
}

// WithContentPostFilter injects the optional post-render content pipeline.
func (s *Service) WithContentPostFilter(filter ContentPostFilter) *Service {
	if s != nil {
		s.contentFilter = filter
	}
	return s
}

func (s *Service) applyContentPostFilter(ctx context.Context, content RenderedContent, resource, resourceID string) (RenderedContent, error) {
	if s == nil || s.contentFilter == nil {
		return content, nil
	}
	next, err := s.contentFilter.AfterHostRender(ctx, ContentPostFilterInput{
		Rendered:   content,
		Resource:   resource,
		ResourceID: resourceID,
		Scope:      "public",
	})
	if err != nil {
		return RenderedContent{}, err
	}
	// 空结果保护：过滤器不得擦除 Host 已验收正文。
	if strings.TrimSpace(next.HTMLContent) == "" && strings.TrimSpace(content.HTMLContent) != "" {
		return content, nil
	}
	return next, nil
}
