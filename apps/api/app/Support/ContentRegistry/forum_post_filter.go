package contentregistry

import (
	"context"
	"strings"
)

// HostPostBodyTargetID is reserved for a future Host composition target that
// echoes Host-rendered HTML into ContentRegistry Execute for filter chains.
// Until Protocol dispatches exact handler leases, ForumPostFilter remains an
// identity adapter when no production invoker is present.
const HostPostBodyTargetID = "sforum.core.content.post-body"

// ForumPostFilter is the production-safe ContentRegistry adapter for forum
// write paths. Empty graphs and missing invokers leave Host HTML unchanged.
type ForumPostFilter struct {
	registry *Registry
	// keepHostOnError is always true for product safety in this leaf.
	keepHostOnError bool
}

// NewForumPostFilter builds a nil-safe identity filter over the process-local
// Content Registry. Full Execute composition requires Protocol handler
// dispatch and is intentionally not invented here.
func NewForumPostFilter(registry *Registry) *ForumPostFilter {
	return &ForumPostFilter{registry: registry, keepHostOnError: true}
}

// AfterHostRender implements forum.ContentPostFilter without importing forum
// (avoids cycle). Callers map HTML/plain fields themselves when needed.
//
// This leaf returns identity for all production cases until a Host post-body
// target + lease-dispatched filters exist. It still proves the seam is live:
// non-nil registry is inspected so Safe Mode / empty graphs stay no-ops.
func (f *ForumPostFilter) AfterHostRender(ctx context.Context, html, plain, resource, resourceID string) (nextHTML, nextPlain string, err error) {
	if f == nil || f.registry == nil || ctx == nil {
		return html, plain, nil
	}
	if err := ctx.Err(); err != nil {
		return html, plain, err
	}
	// 仅当图中存在 render_filter / sanitizer 声明时才有组合意义；
	// 当前无 Protocol 调度时仍保持 Host HTML，避免假 ProviderSet。
	snapshot := f.registry.Snapshot()
	hasFilter := false
	for _, contribution := range snapshot.Content {
		if contribution.Kind == KindRenderFilter || contribution.Kind == KindSanitizer {
			hasFilter = true
			break
		}
	}
	if !hasFilter {
		return html, plain, nil
	}
	// 生产 invoker 未接线：过滤器存在也不改变 Host 正文（fail-closed for plugins）。
	_ = resource
	_ = resourceID
	_ = strings.TrimSpace(html)
	return html, plain, nil
}

// HasFilterContributions reports whether any filter/sanitizer is published.
// Used by tests and inspectors; not a security boundary by itself.
func (f *ForumPostFilter) HasFilterContributions() bool {
	if f == nil || f.registry == nil {
		return false
	}
	for _, contribution := range f.registry.Snapshot().Content {
		if contribution.Kind == KindRenderFilter || contribution.Kind == KindSanitizer {
			return true
		}
	}
	return false
}
