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
)

var standardRegions = []Region{
	{ID: RegionContentBefore, ContractVersion: "sforum.region.content_before@1", Kind: "content", Multiple: true},
	{ID: RegionContentAfter, ContractVersion: "sforum.region.content_after@1", Kind: "content", Multiple: true},
	{ID: RegionSidebar, ContractVersion: "sforum.region.sidebar@1", Kind: "sidebar", Multiple: true},
}

// pageRegionMatrix whitelists which standard regions each catalog page exposes.
// Pages without a right rail do not expose the sidebar region.
var pageRegionMatrix = map[string][]string{
	"forum.home":           {RegionContentBefore, RegionContentAfter, RegionSidebar},
	"forum.category.index": {RegionContentBefore, RegionContentAfter, RegionSidebar},
	// 分类详情页是两栏布局(导航 + 主列),无右栏,不暴露 sidebar。
	"forum.category.show": {RegionContentBefore, RegionContentAfter},
	"forum.tag.index":      {RegionContentBefore, RegionContentAfter, RegionSidebar},
	"forum.tag.show":       {RegionContentBefore, RegionContentAfter, RegionSidebar},
	"forum.topic.show":     {RegionContentBefore, RegionContentAfter, RegionSidebar},
	"forum.profile.show":   {RegionContentBefore, RegionContentAfter, RegionSidebar},
	"forum.topic.create":   {RegionContentBefore, RegionContentAfter},
	"forum.topic.reply":    {RegionContentBefore, RegionContentAfter},
	"forum.topic.edit":     {RegionContentBefore, RegionContentAfter},
	"forum.notifications":  {RegionContentBefore, RegionContentAfter},
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
