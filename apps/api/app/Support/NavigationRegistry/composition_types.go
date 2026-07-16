package navigationregistry

import "context"

const CompositionSchemaVersion = "sforum.navigation-region-composition@1"

const (
	maxComposedItems         = 512
	maxCompositionBytes      = 256 * 1024
	maxRuntimeContentRunes   = 16 * 1024
	maxRuntimeAttributes     = 32
	maxRuntimeAttributeRunes = 512
)

// ComposedItem is a typed server-side presentation node. ID remains the stable
// target identity when a provider replaces it; ProviderID and Artifact retain
// exact attribution for inspection.
type ComposedItem struct {
	ID                      string `json:"id"`
	ContractVersion         string `json:"contractVersion"`
	ProviderID              string `json:"providerId"`
	ProviderContractVersion string `json:"providerContractVersion"`
	Kind                    string `json:"kind"`
	Order                   int    `json:"order,omitempty"`
	Priority                int    `json:"priority,omitempty"`
	Label                   string `json:"label"`
	Href                    string `json:"href,omitempty"`
	// Content is plain text. Browser and SSR consumers must never interpret it
	// as HTML; rich fragments belong to the typed Component/Template Registry.
	Content    string            `json:"content,omitempty"`
	Multiple   bool              `json:"multiple,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Artifact   Artifact          `json:"artifact"`
	Wrappers   []ComposedItem    `json:"wrappers,omitempty"`
	Children   []ComposedItem    `json:"children,omitempty"`
}

type CompositionRequest struct {
	Locale                 string                    `json:"locale,omitempty"`
	Visibility             VisibilityInput           `json:"visibility"`
	NavigationKinds        []string                  `json:"navigationKinds,omitempty"`
	RegionKinds            []string                  `json:"regionKinds,omitempty"`
	BaseNavigationChildren map[string][]ComposedItem `json:"baseNavigationChildren,omitempty"`
}

type Composition struct {
	SchemaVersion string         `json:"schemaVersion"`
	Revision      uint64         `json:"revision"`
	Digest        string         `json:"digest"`
	SafeMode      bool           `json:"safeMode,omitempty"`
	Locale        string         `json:"locale"`
	CacheKey      string         `json:"cacheKey"`
	Navigation    []ComposedItem `json:"navigation"`
	Regions       []ComposedItem `json:"regions"`
}

type RuntimeInvocation struct {
	Family          string          `json:"family"`
	Action          string          `json:"action"`
	TargetID        string          `json:"targetId"`
	ContributionID  string          `json:"contributionId"`
	ContractVersion string          `json:"contractVersion"`
	Handler         string          `json:"handler"`
	Locale          string          `json:"locale"`
	Visibility      VisibilityInput `json:"visibility"`
	Current         ComposedItem    `json:"current"`
	Artifact        Artifact        `json:"artifact"`
}

type RuntimeOutput struct {
	Label string `json:"label,omitempty"`
	Href  string `json:"href,omitempty"`
	// Content is plain text, not an HTML escape hatch.
	Content    string            `json:"content,omitempty"`
	Hidden     bool              `json:"hidden,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// RuntimeRenderer invokes a handler through the already-acquired exact lease.
// Declarative contributions with no Handler never call this interface.
type RuntimeRenderer interface {
	RenderNavigation(context.Context, RuntimeInvocation) (RuntimeOutput, error)
	RenderRegion(context.Context, RuntimeInvocation) (RuntimeOutput, error)
}
