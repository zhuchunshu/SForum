package routes

import (
	"sort"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

func inspectConflicts(routes []preparedRoute) []Conflict {
	conflicts := make([]Conflict, 0)
	pathGroups := make(map[string][]preparedRoute)
	identityGroups := make(map[string][]preparedRoute)
	replacements := make(map[string][]preparedRoute)
	baseByID := make(map[string][]preparedRoute)
	for _, item := range routes {
		if addressableAction(item.route.Action) {
			pathGroups[item.route.Method+"\x00"+item.path.signature] = append(pathGroups[item.route.Method+"\x00"+item.path.signature], item)
			identityGroups[item.route.ID+"\x00"+item.route.Method] = append(identityGroups[item.route.ID+"\x00"+item.route.Method], item)
			baseByID[item.route.ID] = append(baseByID[item.route.ID], item)
		}
		if item.route.Action == extensionmanifest.RouteActionReplace {
			replacements[item.route.TargetID+"\x00"+item.route.Method] = append(replacements[item.route.TargetID+"\x00"+item.route.Method], item)
		}
	}
	for _, group := range pathGroups {
		if len(group) > 1 {
			conflicts = append(conflicts, conflictFromPrepared(ConflictPathMethod, "", group))
		}
	}
	for _, group := range identityGroups {
		if len(group) > 1 {
			conflicts = append(conflicts, conflictFromPrepared(ConflictRouteIdentity, group[0].route.ID, group))
		}
	}
	for _, group := range replacements {
		candidates := append([]preparedRoute(nil), group...)
		for _, base := range baseByID[group[0].route.TargetID] {
			if methodsOverlap(group[0].route.Method, base.route.Method) && group[0].path.signature == base.path.signature {
				candidates = append(candidates, base)
			}
		}
		conflicts = append(conflicts, conflictFromPrepared(ConflictProviderSelection, group[0].route.TargetID, candidates))
	}
	sort.SliceStable(conflicts, func(i, j int) bool {
		left, right := conflicts[i], conflicts[j]
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		if left.RouteID != right.RouteID {
			return left.RouteID < right.RouteID
		}
		if left.Method != right.Method {
			return left.Method < right.Method
		}
		return left.PathSignature < right.PathSignature
	})
	return conflicts
}

func conflictFromPrepared(kind ConflictKind, routeID string, items []preparedRoute) Conflict {
	routes := make([]Route, 0, len(items))
	for _, item := range items {
		routes = append(routes, item.route)
	}
	sortRoutes(routes)
	first := routes[0]
	contract := ""
	if routeID != "" {
		for _, route := range routes {
			if route.ID == routeID {
				contract = route.ContractVersion
				break
			}
		}
	}
	return Conflict{
		Kind: kind, RouteID: routeID, ContractVersion: contract, Method: first.Method,
		PathSignature: first.PathSignature, Candidates: cloneRoutes(routes),
	}
}
