// Package regioncatalog owns the neutral Host catalog of standard page regions
// that plugin contributions may place content into (forum.page.regions).
package regioncatalog

import (
	"slices"
	"sort"
	"strings"
)

// Region is one immutable Host-owned placement area on a public page.
type Region struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Kind            string `json:"kind"`
	Multiple        bool   `json:"multiple"`
}

const (
	RegionContentBefore = "content_before"
	RegionContentAfter  = "content_after"
	RegionSidebar       = "sidebar"

	// 以下区域把「一个页面只有三块」细化到具体语义位置。
	RegionListBefore         = "list_before"
	RegionListAfter          = "list_after"
	RegionDetailBefore       = "detail_before"
	RegionDetailAfter        = "detail_after"
	RegionCommentsBefore     = "comments_before"
	RegionCommentsAfter      = "comments_after"
	RegionSidebarTop         = "sidebar_top"
	RegionSidebarBottom      = "sidebar_bottom"
	RegionProfileHeaderAfter = "profile_header_after"
)

// 区域语义类别。仅作展示与排序提示；渲染由贡献 payload 的 type 决定。
const (
	RegionKindContent  = "content"
	RegionKindSidebar  = "sidebar"
	RegionKindList     = "list"
	RegionKindDetail   = "detail"
	RegionKindComments = "comments"
	RegionKindProfile  = "profile"
)

var standardRegions = []Region{
	{ID: RegionContentBefore, ContractVersion: "sforum.region.content_before@1", Kind: RegionKindContent, Multiple: true},
	{ID: RegionContentAfter, ContractVersion: "sforum.region.content_after@1", Kind: RegionKindContent, Multiple: true},
	{ID: RegionSidebar, ContractVersion: "sforum.region.sidebar@1", Kind: RegionKindSidebar, Multiple: true},
	{ID: RegionListBefore, ContractVersion: "sforum.region.list_before@1", Kind: RegionKindList, Multiple: true},
	{ID: RegionListAfter, ContractVersion: "sforum.region.list_after@1", Kind: RegionKindList, Multiple: true},
	{ID: RegionDetailBefore, ContractVersion: "sforum.region.detail_before@1", Kind: RegionKindDetail, Multiple: true},
	{ID: RegionDetailAfter, ContractVersion: "sforum.region.detail_after@1", Kind: RegionKindDetail, Multiple: true},
	{ID: RegionCommentsBefore, ContractVersion: "sforum.region.comments_before@1", Kind: RegionKindComments, Multiple: true},
	{ID: RegionCommentsAfter, ContractVersion: "sforum.region.comments_after@1", Kind: RegionKindComments, Multiple: true},
	{ID: RegionSidebarTop, ContractVersion: "sforum.region.sidebar_top@1", Kind: RegionKindSidebar, Multiple: true},
	{ID: RegionSidebarBottom, ContractVersion: "sforum.region.sidebar_bottom@1", Kind: RegionKindSidebar, Multiple: true},
	{ID: RegionProfileHeaderAfter, ContractVersion: "sforum.region.profile_header_after@1", Kind: RegionKindProfile, Multiple: true},
}

// pageRegionMatrix whitelists which standard regions each catalog page exposes.
// 每个条目都必须在宿主页面组件里有真实的 <SFRegionOutlet> 挂载点：声明一个
// 宿主从不渲染的区域等于对作者撒谎。apps/web/tests/regions/regionRenderSites
// .test.ts 逐页核对挂载点，所以「先加区域、后补渲染」无法蒙混过关。
//
// 页面没有右栏时不暴露 sidebar / sidebar_top / sidebar_bottom。
var pageRegionMatrix = map[string][]string{
	"forum.home": {
		RegionContentBefore, RegionContentAfter, RegionListBefore, RegionListAfter,
		RegionSidebar, RegionSidebarTop, RegionSidebarBottom,
	},
	"forum.search": {
		RegionContentBefore, RegionContentAfter, RegionListBefore, RegionListAfter, RegionSidebar,
	},
	"forum.category.index": {
		RegionContentBefore, RegionContentAfter, RegionListBefore, RegionListAfter, RegionSidebar,
	},
	// 分类详情页是两栏布局(导航 + 主列),无右栏,不暴露 sidebar。
	"forum.category.show": {
		RegionContentBefore, RegionContentAfter, RegionListBefore, RegionListAfter,
	},
	"forum.tag.index": {
		RegionContentBefore, RegionContentAfter, RegionListBefore, RegionListAfter, RegionSidebar,
	},
	"forum.tag.show": {
		RegionContentBefore, RegionContentAfter, RegionListBefore, RegionListAfter, RegionSidebar,
	},
	"forum.topic.show": {
		RegionContentBefore, RegionContentAfter, RegionDetailBefore, RegionDetailAfter,
		RegionCommentsBefore, RegionCommentsAfter, RegionSidebar, RegionSidebarTop, RegionSidebarBottom,
	},
	"forum.profile.show": {
		RegionContentBefore, RegionContentAfter, RegionSidebar, RegionProfileHeaderAfter,
	},
	"forum.topic.create": {RegionContentBefore, RegionContentAfter},
	"forum.topic.edit":   {RegionContentBefore, RegionContentAfter},
	"forum.notifications": {
		RegionContentBefore, RegionContentAfter, RegionListBefore, RegionListAfter,
	},
}

// StandardRegions returns a caller-owned copy of every Host region definition.
func StandardRegions() []Region {
	return append([]Region(nil), standardRegions...)
}

// FindRegion resolves a region id to its definition.
func FindRegion(regionID string) (Region, bool) {
	regionID = strings.TrimSpace(regionID)
	for _, region := range standardRegions {
		if region.ID == regionID {
			return region, true
		}
	}
	return Region{}, false
}

// PageRegions returns the regions a catalog page exposes, in stable render order.
func PageRegions(pageID string) []Region {
	ids, ok := pageRegionMatrix[strings.TrimSpace(pageID)]
	if !ok {
		return nil
	}
	result := make([]Region, 0, len(ids))
	for _, id := range ids {
		if region, found := FindRegion(id); found {
			result = append(result, region)
		}
	}
	return result
}

// Valid reports whether pageID exposes regionID.
func Valid(pageID, regionID string) bool {
	return slices.Contains(pageRegionMatrix[strings.TrimSpace(pageID)], strings.TrimSpace(regionID))
}

// Pages returns every page id that exposes at least one region, sorted.
func Pages() []string {
	result := make([]string, 0, len(pageRegionMatrix))
	for pageID := range pageRegionMatrix {
		result = append(result, pageID)
	}
	sort.Strings(result)
	return result
}

// PageRegionIDs returns the region ids a page exposes, in matrix order.
// 供代码生成器与一致性测试使用；调用方不应依赖返回切片的可变性。
func PageRegionIDs(pageID string) []string {
	return append([]string(nil), pageRegionMatrix[strings.TrimSpace(pageID)]...)
}

// UndeclaredRegions returns every matrix region id that is missing from
// standardRegions. 非空表示目录内部不一致，宿主应当 fail-closed。
func UndeclaredRegions() []string {
	missing := make([]string, 0)
	for _, pageID := range Pages() {
		for _, regionID := range pageRegionMatrix[pageID] {
			if _, ok := FindRegion(regionID); !ok {
				missing = append(missing, pageID+"/"+regionID)
			}
		}
	}
	return missing
}
