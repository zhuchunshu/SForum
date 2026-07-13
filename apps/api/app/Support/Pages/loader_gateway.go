package pages

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// LoaderRouteTarget 将 loopback 地址与同一 runtime instance 的 admission lease 绑定。
type LoaderRouteTarget struct {
	BaseURL string
	Context context.Context
	Release func()
}

// RouteTargetSource 从运行中的插件 runtime 原子取得目标和调用 lease。
type RouteTargetSource interface {
	AcquireRouteTarget(context.Context, string) (LoaderRouteTarget, bool)
}

// ExtensionPackageRoot 解析扩展包内容根（用于读取 schema 文件）。
type ExtensionPackageRoot interface {
	PackageRoot(extensionID string) (string, bool)
}

// LoaderGateway 受控的生产 SSR loader 入口。
// 仅在 access/permission 通过后由 Pages Controller 调用。
type LoaderGateway struct {
	Loader   *PageDataLoader
	Targets  RouteTargetSource
	Packages ExtensionPackageRoot
}

// NewLoaderGateway 组装网关；targets 为 nil 时 loader 始终失败回退。
func NewLoaderGateway(loader *PageDataLoader, targets RouteTargetSource) *LoaderGateway {
	if loader == nil {
		loader = NewPageDataLoader(nil)
	}
	return &LoaderGateway{Loader: loader, Targets: targets}
}

// WithPackages 注入包根解析（schema 文件）。
func (g *LoaderGateway) WithPackages(p ExtensionPackageRoot) *LoaderGateway {
	if g != nil {
		g.Packages = p
	}
	return g
}

// LoadForContribution 为 add/replace 贡献拉取数据。
// 要求：DataSource=plugin 且 DataRoute 非空；否则跳过。
func (g *LoaderGateway) LoadForContribution(ctx context.Context, contrib PageContribution, params map[string]string, locale string, actorID int64) LoaderResult {
	if g == nil || g.Loader == nil {
		return LoaderResult{Error: "pages: loader unavailable", Fallback: true, Status: 503}
	}
	source := strings.TrimSpace(strings.ToLower(contrib.DataSource))
	route := strings.TrimSpace(contrib.DataRoute)
	if route == "" || (source != "" && source != "plugin") {
		// 无插件数据源：不算错误
		return LoaderResult{}
	}
	if source == "" {
		// 有 route 无 source 时按 plugin 处理
		source = "plugin"
	}
	if g.Targets == nil {
		return LoaderResult{Error: "pages: plugin runtime unavailable", Fallback: true, Status: 503}
	}
	target, ok := g.Targets.AcquireRouteTarget(ctx, contrib.ExtensionID)
	if !ok {
		return LoaderResult{Error: "pages: plugin not enabled or runtime unavailable", Fallback: true, Status: 503}
	}
	if target.Release == nil {
		return LoaderResult{Error: "pages: plugin runtime lease unavailable", Fallback: true, Status: 503}
	}
	defer target.Release()
	if strings.TrimSpace(target.BaseURL) == "" {
		return LoaderResult{Error: "pages: plugin not enabled or runtime unavailable", Fallback: true, Status: 503}
	}
	if target.Context != nil {
		ctx = target.Context
	}
	schemaJSON := ""
	if schemaRel := strings.TrimSpace(contrib.DataSchema); schemaRel != "" && g.Packages != nil {
		if root, ok := g.Packages.PackageRoot(contrib.ExtensionID); ok {
			// 仅允许包内相对路径
			if !strings.Contains(schemaRel, "..") && !strings.HasPrefix(schemaRel, "/") {
				full := filepath.Join(root, filepath.FromSlash(schemaRel))
				if raw, err := os.ReadFile(full); err == nil {
					schemaJSON = string(raw)
				}
			}
		}
	}
	return g.Loader.Fetch(ctx, LoaderRequest{
		ExtensionID: contrib.ExtensionID,
		Route:       route,
		Params:      params,
		Locale:      locale,
		ActorID:     actorID,
		TargetBase:  target.BaseURL,
		SchemaJSON:  schemaJSON,
	})
}

// LoadForResolved 对 Resolve 结果中的 replace 贡献拉数据。
// 必须转发 DataSchema，否则 replace 页会跳过 manifest 声明的 schema 校验。
func (g *LoaderGateway) LoadForResolved(ctx context.Context, resolved ResolvedPage, locale string, actorID int64) LoaderResult {
	if resolved.DataSource != "plugin" || strings.TrimSpace(resolved.DataRoute) == "" {
		return LoaderResult{}
	}
	contrib := PageContribution{
		ExtensionID: resolved.ExtensionID,
		DataSource:  resolved.DataSource,
		DataRoute:   resolved.DataRoute,
		DataSchema:  resolved.DataSchema,
	}
	if contrib.ExtensionID == "" {
		contrib.ExtensionID = resolved.Provider
	}
	return g.LoadForContribution(ctx, contrib, nil, locale, actorID)
}

// DecodeLoaderData 将 RawMessage 解为 any 供 JSON 响应。
func DecodeLoaderData(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	return v
}
