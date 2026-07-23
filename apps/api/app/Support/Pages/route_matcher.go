package pages

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"unicode/utf8"
)

// 路由段类型（signature 用类型，不保留参数名）。
const (
	segStatic   = "S"
	segParam    = "P"
	segCatchAll = "C"
)

// CompiledRoute 编译后的 add 路由，用于确定性匹配。
type CompiledRoute struct {
	// Signature 规范化语义签名，如 /docs/P、/files/C；参数名差异不影响签名。
	Signature string
	// Pattern 原始规范化路径（保留参数名，供调试/展示）。
	Pattern string
	// Segments 编译段。
	Segments []routeSegment
	// Specificity 越大越优先：静态 > 参数 > catch-all；同分按 pattern 字典序。
	Specificity int
	// Contribution 绑定的贡献。
	Contribution PageContribution
}

type routeSegment struct {
	kind  string // S | P | C
	value string // 静态段原文；参数名为 :slug 中的 slug；catch-all 名
}

// RouteMatch 一次成功匹配的结果。
type RouteMatch struct {
	Contribution PageContribution
	Params       map[string]string
	Signature    string
	Pattern      string
}

// CanonicalRouteSignature 将路径模式规范为语义签名。
// /docs/:slug 与 /docs/:id → /docs/P；catch-all 为 C；静态段保留小写原文。
func CanonicalRouteSignature(pathPattern string) (string, error) {
	path := normalizePublicPath(pathPattern)
	if path == "/" {
		return "/", nil
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	out := make([]string, 0, len(parts))
	for i, part := range parts {
		if part == "" {
			return "", fmt.Errorf("%w: empty path segment", ErrInvalidContribution)
		}
		kind, _, err := classifySegment(part, i == len(parts)-1)
		if err != nil {
			return "", err
		}
		switch kind {
		case segStatic:
			out = append(out, strings.ToLower(part))
		case segParam:
			out = append(out, segParam)
		case segCatchAll:
			out = append(out, segCatchAll)
		}
	}
	return "/" + strings.Join(out, "/"), nil
}

// CompileRoute 编译 add 路径模式。
func CompileRoute(pathPattern string, contrib PageContribution) (CompiledRoute, error) {
	path := normalizePublicPath(pathPattern)
	sig, err := CanonicalRouteSignature(path)
	if err != nil {
		return CompiledRoute{}, err
	}
	if path == "/" {
		return CompiledRoute{
			Signature:    "/",
			Pattern:      "/",
			Segments:     nil,
			Specificity:  1_000_000,
			Contribution: contrib,
		}, nil
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	segs := make([]routeSegment, 0, len(parts))
	specificity := 0
	for i, part := range parts {
		kind, name, err := classifySegment(part, i == len(parts)-1)
		if err != nil {
			return CompiledRoute{}, err
		}
		segs = append(segs, routeSegment{kind: kind, value: name})
		switch kind {
		case segStatic:
			// 静态段权重最高；越靠前越重要
			specificity += 1000 + (len(parts)-i)*10
		case segParam:
			specificity += 100 + (len(parts) - i)
		case segCatchAll:
			specificity += 1
		}
	}
	return CompiledRoute{
		Signature:    sig,
		Pattern:      path,
		Segments:     segs,
		Specificity:  specificity,
		Contribution: contrib,
	}, nil
}

func classifySegment(part string, isLast bool) (kind, name string, err error) {
	// catch-all：*rest、:path(.*)、:path...、**
	if part == "**" || part == "*" {
		if !isLast {
			return "", "", fmt.Errorf("%w: catch-all must be final segment", ErrInvalidContribution)
		}
		return segCatchAll, "path", nil
	}
	if strings.HasPrefix(part, "*") && len(part) > 1 {
		if !isLast {
			return "", "", fmt.Errorf("%w: catch-all must be final segment", ErrInvalidContribution)
		}
		return segCatchAll, strings.TrimPrefix(part, "*"), nil
	}
	if strings.Contains(part, "(.*)") || strings.HasSuffix(part, "...") {
		if !isLast {
			return "", "", fmt.Errorf("%w: catch-all must be final segment", ErrInvalidContribution)
		}
		name = strings.TrimPrefix(part, ":")
		name = strings.TrimSuffix(name, "(.*)")
		name = strings.TrimSuffix(name, "...")
		if name == "" {
			name = "path"
		}
		return segCatchAll, name, nil
	}
	if strings.HasPrefix(part, ":") {
		name = strings.TrimPrefix(part, ":")
		if name == "" {
			return "", "", fmt.Errorf("%w: empty param name", ErrInvalidContribution)
		}
		return segParam, name, nil
	}
	if strings.ContainsAny(part, ":*") {
		return "", "", fmt.Errorf("%w: invalid path segment %q", ErrInvalidContribution, part)
	}
	return segStatic, part, nil
}

// MatchRequestPath 对已排序的编译路由表做确定性匹配；返回首个命中。
// routes 必须已按 Specificity 降序、Pattern 升序排序。
func MatchRequestPath(routes []CompiledRoute, requestPath string) (RouteMatch, bool) {
	path := normalizePublicPath(stripLocalePrefix(requestPath))
	if path == "/" {
		for _, r := range routes {
			if r.Pattern == "/" {
				return RouteMatch{
					Contribution: r.Contribution,
					Params:       map[string]string{},
					Signature:    r.Signature,
					Pattern:      r.Pattern,
				}, true
			}
		}
		return RouteMatch{}, false
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for _, r := range routes {
		params, ok := matchSegments(r.Segments, parts)
		if ok {
			return RouteMatch{
				Contribution: r.Contribution,
				Params:       params,
				Signature:    r.Signature,
				Pattern:      r.Pattern,
			}, true
		}
	}
	return RouteMatch{}, false
}

// MatchCorePagePath verifies that a catalog page id and a browser-controlled
// path describe the same core page before route parameters enter a ViewModel.
func MatchCorePagePath(pageID, requestPath string) (map[string]string, bool) {
	page, ok := Find(strings.TrimSpace(pageID))
	if !ok {
		return nil, false
	}
	// 虚拟系统错误页没有固定路由模式，它描述的是 Host 已确认的当前错误路径。
	if page.Virtual && strings.TrimSpace(page.PathPattern) == "" {
		return map[string]string{}, true
	}
	if strings.TrimSpace(page.PathPattern) == "" {
		return nil, false
	}
	route, err := CompileRoute(page.PathPattern, PageContribution{})
	if err != nil {
		return nil, false
	}
	matched, ok := MatchRequestPath([]CompiledRoute{route}, requestPath)
	if !ok {
		return nil, false
	}
	params := make(map[string]string, len(matched.Params))
	for key, value := range matched.Params {
		params[key] = value
	}
	return params, true
}

func matchSegments(segs []routeSegment, parts []string) (map[string]string, bool) {
	params := map[string]string{}
	if len(segs) == 0 {
		return params, len(parts) == 0 || (len(parts) == 1 && parts[0] == "")
	}
	si, pi := 0, 0
	for si < len(segs) {
		seg := segs[si]
		switch seg.kind {
		case segStatic:
			if pi >= len(parts) || !strings.EqualFold(parts[pi], seg.value) {
				return nil, false
			}
			si++
			pi++
		case segParam:
			if pi >= len(parts) || parts[pi] == "" {
				return nil, false
			}
			value, ok := decodeRouteParamPart(parts[pi], false)
			if !ok {
				return nil, false
			}
			params[seg.value] = value
			si++
			pi++
		case segCatchAll:
			if pi >= len(parts) {
				return nil, false
			}
			decoded := make([]string, 0, len(parts)-pi)
			for _, part := range parts[pi:] {
				value, ok := decodeRouteParamPart(part, true)
				if !ok {
					return nil, false
				}
				decoded = append(decoded, value)
			}
			params[seg.value] = strings.Join(decoded, "/")
			si++
			pi = len(parts)
		default:
			return nil, false
		}
	}
	return params, pi == len(parts) && si == len(segs)
}

func decodeRouteParamPart(value string, allowSlash bool) (string, bool) {
	decoded, err := url.PathUnescape(value)
	if err != nil || !utf8.ValidString(decoded) || (!allowSlash && strings.Contains(decoded, "/")) {
		return "", false
	}
	return decoded, true
}

// SortCompiledRoutes 静态优先、参数次之、catch-all 最后；同分按 pattern 字典序保证稳定。
func SortCompiledRoutes(routes []CompiledRoute) {
	sort.SliceStable(routes, func(i, j int) bool {
		if routes[i].Specificity != routes[j].Specificity {
			return routes[i].Specificity > routes[j].Specificity
		}
		return routes[i].Pattern < routes[j].Pattern
	})
}

// signaturesConflict 语义签名相同即冲突（忽略参数名）。
func signaturesConflict(a, b string) bool {
	return a != "" && a == b
}
