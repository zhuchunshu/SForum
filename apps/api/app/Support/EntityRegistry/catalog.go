package entityregistry

import "sort"

// CatalogSchemaVersion is the Host-owned entity catalog contract.
const CatalogSchemaVersion = "sforum.entity-catalog@1"

// CatalogEntity is one inspectable entity type with Host-derived plan summaries.
type CatalogEntity struct {
	ID                 string            `json:"id"`
	ContractVersion    string            `json:"contractVersion"`
	StorageKey         string            `json:"storageKey,omitempty"`
	ExtensionID        string            `json:"extensionId"`
	ExtensionVersion   string            `json:"extensionVersion"`
	PackageDigest      string            `json:"packageDigest"`
	ImportExportPolicy string            `json:"importExportPolicy,omitempty"`
	DeletionPolicy     string            `json:"deletionPolicy,omitempty"`
	Index              IndexPlan         `json:"index"`
	ImportExport       ImportExportPlan  `json:"importExport"`
	Deletion           DeletionPlan      `json:"deletion"`
	Fields             []CatalogField    `json:"fields"`
	Taxonomies         []CatalogTaxonomy `json:"taxonomies"`
}

// CatalogField is a field schema projection for one entity.
type CatalogField struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Schema          string `json:"schema,omitempty"`
	Required        bool   `json:"required,omitempty"`
	Indexed         bool   `json:"indexed,omitempty"`
	IndexKind       string `json:"indexKind,omitempty"`
	UIComponent     string `json:"uiComponent,omitempty"`
	Order           int    `json:"order,omitempty"`
}

// CatalogTaxonomy is a taxonomy bound to one entity.
type CatalogTaxonomy struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Hierarchical    bool   `json:"hierarchical,omitempty"`
}

// Catalog is the inspectable Entity Registry surface for Host/admin/devtools.
type Catalog struct {
	SchemaVersion string          `json:"schemaVersion"`
	Revision      uint64          `json:"revision"`
	Digest        string          `json:"digest"`
	SafeMode      bool            `json:"safeMode,omitempty"`
	Entities      []CatalogEntity `json:"entities"`
}

// BuildCatalog projects the active Entity Registry into a loadable catalog.
// Safe Mode still returns core entities only (third-party already filtered).
func (r *Registry) BuildCatalog() Catalog {
	if r == nil {
		return Catalog{SchemaVersion: CatalogSchemaVersion, Entities: []CatalogEntity{}}
	}
	snapshot := r.Snapshot()
	catalog := Catalog{
		SchemaVersion: CatalogSchemaVersion,
		Revision:      snapshot.Revision,
		Digest:        snapshot.Digest,
		SafeMode:      snapshot.SafeMode,
		Entities:      []CatalogEntity{},
	}
	for _, entity := range snapshot.Entities {
		if entity.Kind != KindEntity {
			continue
		}
		entry := CatalogEntity{
			ID:                 entity.ID,
			ContractVersion:    entity.ContractVersion,
			StorageKey:         entity.StorageKey,
			ExtensionID:        entity.Artifact.ExtensionID,
			ExtensionVersion:   entity.Artifact.ExtensionVersion,
			PackageDigest:      entity.Artifact.PackageDigest,
			ImportExportPolicy: entity.ImportExportPolicy,
			DeletionPolicy:     entity.DeletionPolicy,
		}
		if plan, err := r.IndexPlanForEntity(entity.ID); err == nil {
			entry.Index = plan
		}
		if plan, err := r.ImportExportPlanForEntity(entity.ID); err == nil {
			entry.ImportExport = plan
		}
		if plan, err := r.DeletionPlanForEntity(entity.ID); err == nil {
			entry.Deletion = plan
		}
		for _, field := range r.ListFieldsForEntity(entity.ID) {
			entry.Fields = append(entry.Fields, CatalogField{
				ID: field.ID, ContractVersion: field.ContractVersion,
				Schema: field.Schema, Required: field.Required,
				Indexed: field.Indexed, IndexKind: field.IndexKind,
				UIComponent: field.UIComponent, Order: field.Order,
			})
		}
		sort.Slice(entry.Fields, func(i, j int) bool {
			if entry.Fields[i].Order != entry.Fields[j].Order {
				return entry.Fields[i].Order < entry.Fields[j].Order
			}
			return entry.Fields[i].ID < entry.Fields[j].ID
		})
		for _, taxonomy := range r.ListTaxonomiesForEntity(entity.ID) {
			entry.Taxonomies = append(entry.Taxonomies, CatalogTaxonomy{
				ID: taxonomy.ID, ContractVersion: taxonomy.ContractVersion,
				Hierarchical: taxonomy.Hierarchical,
			})
		}
		sort.Slice(entry.Taxonomies, func(i, j int) bool {
			return entry.Taxonomies[i].ID < entry.Taxonomies[j].ID
		})
		catalog.Entities = append(catalog.Entities, entry)
	}
	sort.Slice(catalog.Entities, func(i, j int) bool {
		return catalog.Entities[i].ID < catalog.Entities[j].ID
	})
	return catalog
}
