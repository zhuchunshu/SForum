package pages

import (
	"sort"
	"strings"
)

// ThemeRuntimeInspection is a redacted admin view of staged theme/plugin
// runtime snapshots and the active/default theme selection.
type ThemeRuntimeInspection struct {
	Revision      uint64                    `json:"revision"`
	ActiveTheme   string                    `json:"activeTheme,omitempty"`
	DefaultTheme  string                    `json:"defaultTheme,omitempty"`
	SnapshotCount int                       `json:"snapshotCount"`
	OverrideCount int                       `json:"overrideCount"`
	Snapshots     []ThemeRuntimeInspectItem `json:"snapshots"`
}

// ThemeRuntimeInspectItem summarizes one staged runtime package without
// exposing package filesystem roots or compiled template bodies.
type ThemeRuntimeInspectItem struct {
	ExtensionID         string   `json:"extensionId"`
	ExtensionVersion    string   `json:"extensionVersion"`
	PackageDigest       string   `json:"packageDigest"`
	Kind                string   `json:"kind"`
	ContributionIDs     []string `json:"contributionIds,omitempty"`
	OverrideTargets     []string `json:"overrideTargets,omitempty"`
	NavigationLocations []string `json:"navigationLocations,omitempty"`
	Active              bool     `json:"active,omitempty"`
	Default             bool     `json:"default,omitempty"`
}

// InspectSnapshot returns a detached redacted inspection of the live registry.
func (r *ThemeRuntimeRegistry) InspectSnapshot() ThemeRuntimeInspection {
	if r == nil {
		return ThemeRuntimeInspection{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]ThemeRuntimeInspectItem, 0, len(r.snapshots))
	overrideCount := 0
	for artifact, snapshot := range r.snapshots {
		if snapshot == nil {
			continue
		}
		contribs := make([]string, 0, len(snapshot.providers))
		for _, binding := range snapshot.providers {
			if id := strings.TrimSpace(binding.ContributionID); id != "" {
				contribs = append(contribs, id)
			}
		}
		sort.Strings(contribs)
		overrides := make([]string, 0, len(snapshot.overrides))
		for target := range snapshot.overrides {
			overrides = append(overrides, target)
		}
		sort.Strings(overrides)
		overrideCount += len(overrides)
		kind := "theme"
		if snapshot.kind == RuntimeTemplatePlugin {
			kind = "plugin"
		}
		navigationLocations := make([]string, 0, len(snapshot.navigationLocations))
		for location := range snapshot.navigationLocations {
			navigationLocations = append(navigationLocations, location)
		}
		sort.Strings(navigationLocations)
		items = append(items, ThemeRuntimeInspectItem{
			ExtensionID: artifact.ExtensionID, ExtensionVersion: artifact.ExtensionVersion,
			PackageDigest: artifact.PackageDigest, Kind: kind,
			ContributionIDs: contribs, OverrideTargets: overrides, NavigationLocations: navigationLocations,
			Active: r.active == artifact, Default: r.defaultArtifact == artifact,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ExtensionID == items[j].ExtensionID {
			return items[i].PackageDigest < items[j].PackageDigest
		}
		return items[i].ExtensionID < items[j].ExtensionID
	})
	return ThemeRuntimeInspection{
		Revision: r.revision, ActiveTheme: r.active.ExtensionID, DefaultTheme: r.defaultArtifact.ExtensionID,
		SnapshotCount: len(items), OverrideCount: overrideCount, Snapshots: items,
	}
}
