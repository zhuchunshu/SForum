package contentregistry

import "sort"

// CatalogSchemaVersion is the Host-owned content catalog contract.
const CatalogSchemaVersion = "sforum.content-catalog@1"

// CatalogEntry is one inspectable content contribution (block/filter/etc.).
type CatalogEntry struct {
	ID               string `json:"id"`
	ContractVersion  string `json:"contractVersion"`
	Kind             string `json:"kind"`
	Handler          string `json:"handler,omitempty"`
	Schema           string `json:"schema,omitempty"`
	Renderer         string `json:"renderer,omitempty"`
	Migration        string `json:"migration,omitempty"`
	ExtensionID      string `json:"extensionId"`
	ExtensionVersion string `json:"extensionVersion"`
	PackageDigest    string `json:"packageDigest"`
	Core             bool   `json:"core,omitempty"`
}

// Catalog is the inspectable Content Registry surface for Host/admin/devtools.
// It projects declaration metadata only — no execution bindings or secrets.
type Catalog struct {
	SchemaVersion string         `json:"schemaVersion"`
	Revision      uint64         `json:"revision"`
	Digest        string         `json:"digest"`
	SafeMode      bool           `json:"safeMode,omitempty"`
	Content       []CatalogEntry `json:"content"`
}

// BuildCatalog projects the active Content Registry into a loadable catalog.
// Safe Mode still returns core contributions only (third-party already filtered).
func (r *Registry) BuildCatalog() Catalog {
	if r == nil {
		return Catalog{SchemaVersion: CatalogSchemaVersion, Content: []CatalogEntry{}}
	}
	snapshot := r.Snapshot()
	catalog := Catalog{
		SchemaVersion: CatalogSchemaVersion,
		Revision:      snapshot.Revision,
		Digest:        snapshot.Digest,
		SafeMode:      snapshot.SafeMode,
		Content:       []CatalogEntry{},
	}
	for _, contribution := range snapshot.Content {
		catalog.Content = append(catalog.Content, CatalogEntry{
			ID:               contribution.ID,
			ContractVersion:  contribution.ContractVersion,
			Kind:             contribution.Kind,
			Handler:          contribution.Handler,
			Schema:           contribution.Schema,
			Renderer:         contribution.Renderer,
			Migration:        contribution.Migration,
			ExtensionID:      contribution.Artifact.ExtensionID,
			ExtensionVersion: contribution.Artifact.ExtensionVersion,
			PackageDigest:    contribution.Artifact.PackageDigest,
			Core:             contribution.Artifact.Core,
		})
	}
	sort.Slice(catalog.Content, func(i, j int) bool {
		if catalog.Content[i].Kind != catalog.Content[j].Kind {
			return catalog.Content[i].Kind < catalog.Content[j].Kind
		}
		return catalog.Content[i].ID < catalog.Content[j].ID
	})
	return catalog
}
