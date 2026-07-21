package mediaregistry

import "sort"

// CatalogSchemaVersion is the Host-owned media catalog contract for inspectors.
const CatalogSchemaVersion = "sforum.media-catalog@1"

// CatalogPolicy is one inspectable MIME policy contribution (no secrets).
type CatalogPolicy struct {
	ID                 string   `json:"id"`
	ContractVersion    string   `json:"contractVersion"`
	Purpose            string   `json:"purpose"`
	Priority           int      `json:"priority,omitempty"`
	RequiredPermission string   `json:"requiredPermission,omitempty"`
	AllowedMIMEs       []string `json:"allowedMimes,omitempty"`
	DeniedMIMEs        []string `json:"deniedMimes,omitempty"`
	AllowedExtensions  []string `json:"allowedExtensions,omitempty"`
	ExtensionID        string   `json:"extensionId"`
	ExtensionVersion   string   `json:"extensionVersion"`
	PackageDigest      string   `json:"packageDigest"`
	Core               bool     `json:"core,omitempty"`
}

// CatalogProcessor is one inspectable pipeline processor contribution.
type CatalogProcessor struct {
	ID                 string   `json:"id"`
	ContractVersion    string   `json:"contractVersion"`
	Stage              string   `json:"stage"`
	Purpose            string   `json:"purpose"`
	Handler            string   `json:"handler,omitempty"`
	Priority           int      `json:"priority,omitempty"`
	Mode               string   `json:"mode,omitempty"`
	Execution          string   `json:"execution,omitempty"`
	FailureMode        string   `json:"failureMode,omitempty"`
	RequiredPermission string   `json:"requiredPermission,omitempty"`
	MIMEs              []string `json:"mimes,omitempty"`
	ExtensionID        string   `json:"extensionId"`
	ExtensionVersion   string   `json:"extensionVersion"`
	PackageDigest      string   `json:"packageDigest"`
	Core               bool     `json:"core,omitempty"`
}

// CatalogVariant is one inspectable regenerable variant declaration.
type CatalogVariant struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Purpose         string `json:"purpose"`
	Name            string `json:"name"`
	ProcessorID     string `json:"processorId,omitempty"`
	OutputMIME      string `json:"outputMime,omitempty"`
	Priority        int    `json:"priority,omitempty"`
	// BindingStatus is active/pending from the current graph (not a receipt).
	BindingStatus    string `json:"bindingStatus,omitempty"`
	BindingReason    string `json:"bindingReason,omitempty"`
	ExtensionID      string `json:"extensionId"`
	ExtensionVersion string `json:"extensionVersion"`
	PackageDigest    string `json:"packageDigest"`
	Core             bool   `json:"core,omitempty"`
}

// Catalog is the inspectable Media Registry surface for Host/admin/devtools.
// Declaration metadata only — no plan/execute receipts, bytes, or credentials.
type Catalog struct {
	SchemaVersion string             `json:"schemaVersion"`
	Revision      uint64             `json:"revision"`
	Digest        string             `json:"digest"`
	SafeMode      bool               `json:"safeMode,omitempty"`
	Policies      []CatalogPolicy    `json:"policies"`
	Processors    []CatalogProcessor `json:"processors"`
	Variants      []CatalogVariant   `json:"variants"`
}

// BuildCatalog projects the active Media Registry into a loadable catalog.
// Safe Mode still returns core contributions only (third-party already filtered).
func (r *Registry) BuildCatalog() Catalog {
	if r == nil {
		return Catalog{
			SchemaVersion: CatalogSchemaVersion,
			Policies:      []CatalogPolicy{},
			Processors:    []CatalogProcessor{},
			Variants:      []CatalogVariant{},
		}
	}
	snapshot := r.Snapshot()
	catalog := Catalog{
		SchemaVersion: CatalogSchemaVersion,
		Revision:      snapshot.Revision,
		Digest:        snapshot.Digest,
		SafeMode:      snapshot.SafeMode,
		Policies:      []CatalogPolicy{},
		Processors:    []CatalogProcessor{},
		Variants:      []CatalogVariant{},
	}
	bindingByID := map[string]VariantBinding{}
	for _, binding := range snapshot.VariantBindings {
		bindingByID[binding.Variant.ID] = binding
	}
	for _, policy := range snapshot.Policies {
		catalog.Policies = append(catalog.Policies, CatalogPolicy{
			ID:                 policy.ID,
			ContractVersion:    policy.ContractVersion,
			Purpose:            policy.Purpose,
			Priority:           policy.Priority,
			RequiredPermission: policy.RequiredPermission,
			AllowedMIMEs:       append([]string(nil), policy.AllowedMIMEs...),
			DeniedMIMEs:        append([]string(nil), policy.DeniedMIMEs...),
			AllowedExtensions:  append([]string(nil), policy.AllowedExtensions...),
			ExtensionID:        policy.Artifact.ExtensionID,
			ExtensionVersion:   policy.Artifact.ExtensionVersion,
			PackageDigest:      policy.Artifact.PackageDigest,
			Core:               policy.Artifact.Core,
		})
	}
	for _, processor := range snapshot.Processors {
		catalog.Processors = append(catalog.Processors, CatalogProcessor{
			ID:                 processor.ID,
			ContractVersion:    processor.ContractVersion,
			Stage:              processor.Stage,
			Purpose:            processor.Purpose,
			Handler:            processor.Handler,
			Priority:           processor.Priority,
			Mode:               processor.Mode,
			Execution:          processor.Execution,
			FailureMode:        processor.FailureMode,
			RequiredPermission: processor.RequiredPermission,
			MIMEs:              append([]string(nil), processor.MIMEs...),
			ExtensionID:        processor.Artifact.ExtensionID,
			ExtensionVersion:   processor.Artifact.ExtensionVersion,
			PackageDigest:      processor.Artifact.PackageDigest,
			Core:               processor.Artifact.Core,
		})
	}
	for _, variant := range snapshot.Variants {
		entry := CatalogVariant{
			ID:               variant.ID,
			ContractVersion:  variant.ContractVersion,
			Purpose:          variant.Purpose,
			Name:             variant.Name,
			ProcessorID:      variant.ProcessorID,
			OutputMIME:       variant.OutputMIME,
			Priority:         variant.Priority,
			ExtensionID:      variant.Artifact.ExtensionID,
			ExtensionVersion: variant.Artifact.ExtensionVersion,
			PackageDigest:    variant.Artifact.PackageDigest,
			Core:             variant.Artifact.Core,
		}
		if binding, ok := bindingByID[variant.ID]; ok {
			entry.BindingStatus = binding.Status
			entry.BindingReason = binding.Reason
		}
		catalog.Variants = append(catalog.Variants, entry)
	}
	sort.Slice(catalog.Policies, func(i, j int) bool {
		if catalog.Policies[i].Purpose != catalog.Policies[j].Purpose {
			return catalog.Policies[i].Purpose < catalog.Policies[j].Purpose
		}
		return catalog.Policies[i].ID < catalog.Policies[j].ID
	})
	sort.Slice(catalog.Processors, func(i, j int) bool {
		if catalog.Processors[i].Stage != catalog.Processors[j].Stage {
			return catalog.Processors[i].Stage < catalog.Processors[j].Stage
		}
		return catalog.Processors[i].ID < catalog.Processors[j].ID
	})
	sort.Slice(catalog.Variants, func(i, j int) bool {
		return catalog.Variants[i].ID < catalog.Variants[j].ID
	})
	return catalog
}
