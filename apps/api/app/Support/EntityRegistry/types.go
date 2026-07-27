package entityregistry

import "errors"

// SchemaVersion is the stable identity of this immutable entity graph.
const SchemaVersion = "sforum.entity-registry@1"

// Frozen Manifest entity surface kinds (entity type / taxonomy / field schema).
const (
	KindEntity   = "entity"
	KindTaxonomy = "taxonomy"
	KindField    = "field"
)

// Import/export and deletion policies are Host-owned enums. Plugins declare
// intent only; Host storage and RBAC remain authoritative for actual data.
const (
	ImportExportAllow      = "allow"
	ImportExportDeny       = "deny"
	ImportExportExportOnly = "export_only"
	ImportExportImportOnly = "import_only"

	DeletionSoft   = "soft"
	DeletionHard   = "hard"
	DeletionRetain = "retain"

	IndexNone    = "none"
	IndexKeyword = "keyword"
	IndexText    = "text"
	IndexNumeric = "numeric"
	IndexBoolean = "boolean"
)

// EntityAction is a Host permission action against one entity type.
const (
	ActionCreate      = "create"
	ActionRead        = "read"
	ActionUpdate      = "update"
	ActionDelete      = "delete"
	ActionImport      = "import"
	ActionExport      = "export"
	ActionManageTerms = "manage_terms"
	ActionAssignTerms = "assign_terms"
	ActionReadField   = "read_field"
	ActionWriteField  = "write_field"
)

var (
	ErrInvalid          = errors.New("entity registry declaration is invalid")
	ErrConflict         = errors.New("entity registry conflicts with the active graph")
	ErrArtifactConflict = errors.New("entity registry artifact does not own the active publication")
	ErrRevisionConflict = errors.New("entity registry revision changed during replacement")
	ErrSafeMode         = errors.New("entity registry rejects third-party publication in safe mode")
	ErrNotFound         = errors.New("entity registry declaration is not found")
	ErrPermissionDenied = errors.New("entity registry permission denied")
	ErrPolicyDenied     = errors.New("entity registry policy denied")
)

// Artifact binds every contribution to one exact package. Core publications
// carry a package-private Host seal from NewCoreArtifact.
type Artifact struct {
	ExtensionID       string `json:"extensionId"`
	ExtensionVersion  string `json:"extensionVersion"`
	PackageDigest     string `json:"packageDigest"`
	VersionID         int64  `json:"versionId,omitempty"`
	RuntimeInstanceID string `json:"runtimeInstanceId,omitempty"`
	Core              bool   `json:"core,omitempty"`
	// coreSeal is deliberately not serializable. Core authority must enter
	// through NewCoreArtifact only.
	coreSeal [32]byte
}

// Declaration is one frozen entity/taxonomy/field contribution.
//
// entity: product type with CRUD permissions and data lifecycle policies.
// taxonomy: term vocabulary bound to one or more same-package entity types.
// field: field schema + UI + index + permission bound to one entity type.
type Declaration struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Kind            string `json:"kind"`
	// Label is the operator-facing display name.
	Label string `json:"label,omitempty"`
	// StorageKey is a Host-stable key for entity/taxonomy storage namespaces.
	// It must be package-prefixed and never collide across publications.
	StorageKey string `json:"storageKey,omitempty"`

	// Entity permissions (kind=entity). Empty permission denies the action.
	PermissionCreate string `json:"permissionCreate,omitempty"`
	PermissionRead   string `json:"permissionRead,omitempty"`
	PermissionUpdate string `json:"permissionUpdate,omitempty"`
	PermissionDelete string `json:"permissionDelete,omitempty"`
	PermissionImport string `json:"permissionImport,omitempty"`
	PermissionExport string `json:"permissionExport,omitempty"`

	// ImportExportPolicy and DeletionPolicy are Host-enforced lifecycle intent.
	ImportExportPolicy string `json:"importExportPolicy,omitempty"`
	DeletionPolicy     string `json:"deletionPolicy,omitempty"`

	// TaxonomyIDs optionally lists same-package taxonomies usable on an entity.
	TaxonomyIDs []string `json:"taxonomyIds,omitempty"`

	// Taxonomy fields (kind=taxonomy).
	Hierarchical     bool     `json:"hierarchical,omitempty"`
	EntityIDs        []string `json:"entityIds,omitempty"`
	PermissionManage string   `json:"permissionManage,omitempty"`
	PermissionAssign string   `json:"permissionAssign,omitempty"`

	// Field fields (kind=field).
	EntityID             string `json:"entityId,omitempty"`
	Schema               string `json:"schema,omitempty"`
	UIComponent          string `json:"uiComponent,omitempty"`
	UIModule             string `json:"uiModule,omitempty"`
	UIDigest             string `json:"uiDigest,omitempty"`
	Required             bool   `json:"required,omitempty"`
	Indexed              bool   `json:"indexed,omitempty"`
	IndexKind            string `json:"indexKind,omitempty"`
	PermissionFieldRead  string `json:"permissionFieldRead,omitempty"`
	PermissionFieldWrite string `json:"permissionFieldWrite,omitempty"`
	Validation           string `json:"validation,omitempty"`
	Order                int    `json:"order,omitempty"`
	// Priority orders competing field/taxonomy providers. Higher wins.
	Priority int `json:"priority,omitempty"`
}

// Publication is one exact-artifact owner of zero or more entity declarations.
type Publication struct {
	Artifact Artifact      `json:"artifact"`
	Entities []Declaration `json:"entities,omitempty"`
}

// Contribution exposes a frozen declaration together with its exact owner.
type Contribution struct {
	Declaration
	Artifact Artifact `json:"artifact"`
}

// Snapshot is an immutable, inspectable view of the active entity graph.
type Snapshot struct {
	SchemaVersion string         `json:"schemaVersion"`
	Revision      uint64         `json:"revision"`
	Digest        string         `json:"digest"`
	SafeMode      bool           `json:"safeMode,omitempty"`
	Publications  []Publication  `json:"publications"`
	Entities      []Contribution `json:"entities"`
}

// PermissionDecision is the Host evaluation result for one action.
type PermissionDecision struct {
	Allowed       bool   `json:"allowed"`
	Action        string `json:"action"`
	TargetID      string `json:"targetId"`
	PermissionKey string `json:"permissionKey,omitempty"`
	Policy        string `json:"policy,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

// IndexPlan is the Host-derived search/index projection for one entity type.
type IndexPlan struct {
	EntityID   string           `json:"entityId"`
	StorageKey string           `json:"storageKey"`
	Fields     []IndexFieldPlan `json:"fields"`
}

// IndexFieldPlan describes one indexed field under an entity.
type IndexFieldPlan struct {
	FieldID   string `json:"fieldId"`
	IndexKind string `json:"indexKind"`
	Required  bool   `json:"required"`
	Schema    string `json:"schema"`
}

// ImportExportPlan is the Host-derived import/export contract for one entity.
type ImportExportPlan struct {
	EntityID    string   `json:"entityId"`
	Policy      string   `json:"policy"`
	CanImport   bool     `json:"canImport"`
	CanExport   bool     `json:"canExport"`
	FieldIDs    []string `json:"fieldIds"`
	TaxonomyIDs []string `json:"taxonomyIds,omitempty"`
}

// DeletionPlan is the Host-derived deletion contract for one entity type.
type DeletionPlan struct {
	EntityID   string `json:"entityId"`
	Policy     string `json:"policy"`
	SoftDelete bool   `json:"softDelete"`
	HardDelete bool   `json:"hardDelete"`
	Retain     bool   `json:"retain"`
}
