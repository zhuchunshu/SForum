package sitechrome

import "time"

const NavigationDocumentSchemaVersion = "sforum.site-navigation@1"

type NavigationDefinition struct {
	SourceKey      string `json:"sourceKey"`
	SourceKind     string `json:"sourceKind"`
	LinkKind       string `json:"linkKind"`
	LabelZhCN      string `json:"labelZhCN,omitempty"`
	LabelEnUS      string `json:"labelEnUS,omitempty"`
	Href           string `json:"href,omitempty"`
	Icon           string `json:"icon,omitempty"`
	OpenInNewTab   bool   `json:"openInNewTab,omitempty"`
	ExtensionID    string `json:"extensionId,omitempty"`
	ContributionID string `json:"contributionId,omitempty"`
}

type NavigationPlacement struct {
	SourceKey  string `json:"sourceKey"`
	Location   string `json:"location"`
	Order      int    `json:"order"`
	Enabled    bool   `json:"enabled"`
	Visibility string `json:"visibility"`
	Permission string `json:"permission,omitempty"`
	LabelZhCN  string `json:"labelZhCN,omitempty"`
	LabelEnUS  string `json:"labelEnUS,omitempty"`
	Icon       string `json:"icon,omitempty"`
}

type NavigationDocument struct {
	Revision       uint64                    `json:"revision"`
	Definitions    []NavigationDefinition    `json:"definitions"`
	Placements     []NavigationPlacement     `json:"placements"`
	ThemeLocations []NavigationThemeLocation `json:"themeLocations,omitempty"`
}

type NavigationThemeLocation struct {
	Location  string `json:"location"`
	Supported bool   `json:"supported"`
}

type NavigationSnapshot struct {
	ID                int64              `json:"id"`
	Revision          uint64             `json:"revision"`
	ActorUserID       int64              `json:"actorUserId,omitempty"`
	Operation         string             `json:"operation"`
	Reason            string             `json:"reason,omitempty"`
	AffectedLocations []string           `json:"affectedLocations"`
	CreatedAt         time.Time          `json:"createdAt"`
	Document          NavigationDocument `json:"document"`
}

type NavigationBackup struct {
	Schema      string                 `json:"schema"`
	ExportedAt  time.Time              `json:"exportedAt,omitempty"`
	Definitions []NavigationDefinition `json:"definitions"`
	Placements  []NavigationPlacement  `json:"placements"`
}

type NavigationPreview struct {
	PreviewToken     string                     `json:"previewToken"`
	ExpectedRevision uint64                     `json:"expectedRevision"`
	Mode             string                     `json:"mode"`
	Changes          []string                   `json:"changes"`
	Warnings         []string                   `json:"warnings"`
	ChangeEntries    []NavigationPreviewChange  `json:"changeEntries"`
	WarningEntries   []NavigationPreviewWarning `json:"warningEntries"`
}

type NavigationPreviewChange struct {
	Kind        string `json:"kind"`
	Location    string `json:"location,omitempty"`
	BeforeCount int    `json:"beforeCount"`
	AfterCount  int    `json:"afterCount"`
}

type NavigationPreviewWarning struct {
	Code        string `json:"code"`
	SourceKey   string `json:"sourceKey,omitempty"`
	ExtensionID string `json:"extensionId,omitempty"`
}

type NavigationDefaultsPreviewInput struct {
	ExpectedRevision uint64
	Scope            string
	Location         string
}

type NavigationImportPreviewInput struct {
	ExpectedRevision uint64
	Mode             string
	Backup           NavigationBackup
}

type NavigationApplyInput struct {
	ExpectedRevision uint64
	Reason           string
	Document         NavigationDocument
}

type NavigationPreviewApplyInput struct {
	ExpectedRevision uint64
	PreviewToken     string
	Reason           string
}

// NavigationThemeLocationProvider is a read-only projection of the active
// exact theme artifact. M6 binds it to declared, validated theme locations;
// M1 uses the Core fallback default so configuration is never discarded.
type NavigationThemeLocationProvider interface {
	SupportsNavigationLocation(location string) bool
}

type ResolvedNavigationItem struct {
	SourceKey    string `json:"sourceKey"`
	SourceKind   string `json:"sourceKind"`
	LinkKind     string `json:"linkKind"`
	Label        string `json:"label"`
	Href         string `json:"href,omitempty"`
	Icon         string `json:"icon,omitempty"`
	OpenInNewTab bool   `json:"openInNewTab,omitempty"`
}

type ResolvedNavigationLocation struct {
	Location  string                   `json:"location"`
	Supported bool                     `json:"supported"`
	Items     []ResolvedNavigationItem `json:"items"`
}

type ResolvedNavigation struct {
	SchemaVersion string                       `json:"schemaVersion"`
	Revision      uint64                       `json:"revision"`
	Locations     []ResolvedNavigationLocation `json:"locations"`
}
