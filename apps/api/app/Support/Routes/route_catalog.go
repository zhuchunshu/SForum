package routes

import (
	"errors"
	"fmt"
	"sort"
)

// ErrCatalogUnavailable 表示目录快照的所有者未装配或快照不可读。
var ErrCatalogUnavailable = errors.New("routes: route catalog unavailable")

// RouteCatalogRoute 是注册表目录里的一条路由，附加运行时准入判定与它参与的冲突类型。
// Admitted 只对 plugin 提供者有额外语义：admission 闭包未放行的精确制品声明仍留在
// 注册表里，但不会匹配任何请求。
type RouteCatalogRoute struct {
	Route
	Admitted      bool
	ConflictKinds []ConflictKind
}

// RouteCatalogSnapshot 是 caller-owned 的目录快照，不持有 registry 内部切片。
type RouteCatalogSnapshot struct {
	Revision uint64
	SafeMode bool
	Routes   []RouteCatalogRoute
}

// Catalog 从生产绑定的 ProviderSelectionAPI 派生，保证目录、检查器与提供者选择
// 观察同一个 immutable Registry，而不是构造第二个快照所有者。
type Catalog struct {
	registry *Registry
}

func NewProviderSelectionCatalog(providers *ProviderSelectionAPI) *Catalog {
	if providers == nil {
		return nil
	}
	return &Catalog{registry: providers.registry}
}

// Snapshot 返回当前不可变注册表的目录视图。写入并发时按 revision 重试，
// 与 Inspector 的读取契约一致。
func (c *Catalog) Snapshot() (RouteCatalogSnapshot, error) {
	if c == nil || c.registry == nil {
		return RouteCatalogSnapshot{}, ErrCatalogUnavailable
	}
	for attempt := 0; attempt < 4; attempt++ {
		snapshot := c.registry.Snapshot()
		result := buildRouteCatalog(c.registry, snapshot)
		if c.registry.Revision() == snapshot.Revision {
			return result, nil
		}
	}
	return RouteCatalogSnapshot{}, fmt.Errorf("%w: registry revision changed during catalog snapshot", ErrRevisionConflict)
}

func buildRouteCatalog(registry *Registry, snapshot Snapshot) RouteCatalogSnapshot {
	result := RouteCatalogSnapshot{Revision: snapshot.Revision, SafeMode: snapshot.SafeMode}
	conflictKinds := routeConflictKinds(snapshot.Conflicts)
	result.Routes = make([]RouteCatalogRoute, 0, len(snapshot.Routes))
	for _, route := range snapshot.Routes {
		result.Routes = append(result.Routes, RouteCatalogRoute{
			Route:         route,
			Admitted:      registry.routeAdmitted(route),
			ConflictKinds: conflictKinds[route.ID],
		})
	}
	sort.SliceStable(result.Routes, func(i, j int) bool {
		return routeSortLess(result.Routes[i].Route, result.Routes[j].Route)
	})
	return result
}

// routeConflictKinds 反查每条路由参与的冲突类型：同一冲突的候选共享该冲突类型。
func routeConflictKinds(conflicts []Conflict) map[string][]ConflictKind {
	result := make(map[string][]ConflictKind)
	seen := make(map[string]map[ConflictKind]struct{})
	for _, conflict := range conflicts {
		for _, candidate := range conflict.Candidates {
			kinds, ok := seen[candidate.ID]
			if !ok {
				kinds = make(map[ConflictKind]struct{})
				seen[candidate.ID] = kinds
			}
			if _, duplicate := kinds[conflict.Kind]; duplicate {
				continue
			}
			kinds[conflict.Kind] = struct{}{}
			result[candidate.ID] = append(result[candidate.ID], conflict.Kind)
		}
	}
	return result
}
