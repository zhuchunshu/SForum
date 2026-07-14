package extensionmanifest

type ManifestDatabase struct {
	ContractVersion string `json:"contractVersion"`
	// Authority is accepted only as a legacy manifest input. Normalize expands
	// it into the cumulative exact Grants set before runtime consumers use it.
	Authority         string                      `json:"authority,omitempty"`
	Grants            []string                    `json:"grants,omitempty"`
	Schema            string                      `json:"schema,omitempty"`
	Role              string                      `json:"role,omitempty"`
	CoreCompatibility string                      `json:"coreCompatibility,omitempty"`
	Backup            ManifestBackupPolicy        `json:"backup"`
	Retention         ManifestRetention           `json:"retention"`
	Operations        []ManifestDatabaseOperation `json:"operations,omitempty"`
}

// ManifestDatabaseOperation 将一个宿主管理的 SQL 文件绑定到有界运行时契约。
// SQL 字节不得内联到 manifest，也不会穿过 RPC 边界。
type ManifestDatabaseOperation struct {
	ID               string                      `json:"id"`
	StatementVersion string                      `json:"statementVersion"`
	Kind             string                      `json:"kind"`
	Path             string                      `json:"path"`
	Digest           string                      `json:"digest"`
	Parameters       []ManifestDatabaseParameter `json:"parameters"`
	ResultSchema     string                      `json:"resultSchema,omitempty"`
	Columns          []ManifestDatabaseColumn    `json:"columns"`
	MaxRows          int                         `json:"maxRows,omitempty"`
	MaxAffectedRows  uint64                      `json:"maxAffectedRows,omitempty"`
	TimeoutMS        int                         `json:"timeoutMs"`
}

type ManifestDatabaseParameter struct {
	Schema   string `json:"schema"`
	Field    string `json:"field"`
	Kind     string `json:"kind"`
	Nullable bool   `json:"nullable"`
	MaxBytes int    `json:"maxBytes"`
}

type ManifestDatabaseColumn struct {
	Name     string `json:"name"`
	Nullable bool   `json:"nullable"`
}

type ManifestBackupPolicy struct {
	Required bool   `json:"required"`
	Strategy string `json:"strategy,omitempty"`
}

type ManifestRetention struct {
	OnDisable   string `json:"onDisable"`
	OnUninstall string `json:"onUninstall"`
	Days        int    `json:"days,omitempty"`
}

type ManifestCache struct {
	ID              string   `json:"id"`
	ContractVersion string   `json:"contractVersion"`
	Namespace       string   `json:"namespace"`
	Policy          string   `json:"policy"`
	Tags            []string `json:"tags,omitempty"`
	Provider        string   `json:"provider,omitempty"`
	Invalidators    []string `json:"invalidators,omitempty"`
}

type ManifestAdminSurface struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Kind            string `json:"kind"`
	Action          string `json:"action"`
	TargetID        string `json:"targetId,omitempty"`
	Label           string `json:"label"`
	Handler         string `json:"handler,omitempty"`
	Schema          string `json:"schema,omitempty"`
	Permission      string `json:"permission,omitempty"`
	Priority        int    `json:"priority,omitempty"`
}

type ManifestQuery struct {
	ID               string   `json:"id"`
	ContractVersion  string   `json:"contractVersion"`
	Entity           string   `json:"entity"`
	PlanVersion      string   `json:"planVersion"`
	Fields           []string `json:"fields"`
	Relations        []string `json:"relations,omitempty"`
	Filters          []string `json:"filters,omitempty"`
	Sort             []string `json:"sort,omitempty"`
	Pagination       string   `json:"pagination"`
	ResultSchema     string   `json:"resultSchema"`
	PermissionPolicy string   `json:"permissionPolicy"`
	CacheTags        []string `json:"cacheTags,omitempty"`
}

type ManifestIdentity struct {
	ContractVersion string                      `json:"contractVersion"`
	UserFields      []ManifestIdentityUserField `json:"userFields,omitempty"`
	Providers       []ManifestIdentityProvider  `json:"providers,omitempty"`
	SessionPolicy   string                      `json:"sessionPolicy,omitempty"`
	RiskHooks       []string                    `json:"riskHooks,omitempty"`
}

type ManifestIdentityUserField struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Type            string `json:"type"`
	Schema          string `json:"schema"`
	ReadPermission  string `json:"readPermission,omitempty"`
	WritePermission string `json:"writePermission,omitempty"`
}

type ManifestIdentityProvider struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Kind            string `json:"kind"`
	Handler         string `json:"handler"`
	Priority        int    `json:"priority,omitempty"`
}

type ManifestPermissionDefinition struct {
	Key              string   `json:"key"`
	ContractVersion  string   `json:"contractVersion"`
	Label            string   `json:"label"`
	Description      string   `json:"description"`
	RecommendedRoles []string `json:"recommendedRoles,omitempty"`
	AssignmentPolicy string   `json:"assignmentPolicy"`
}

type ManifestMediaPipeline struct {
	ID              string                   `json:"id"`
	ContractVersion string                   `json:"contractVersion"`
	Action          string                   `json:"action"`
	TargetID        string                   `json:"targetId,omitempty"`
	MIMEs           []string                 `json:"mimes"`
	Transforms      []ManifestMediaTransform `json:"transforms,omitempty"`
	Handler         string                   `json:"handler"`
	Permission      string                   `json:"permission,omitempty"`
	Priority        int                      `json:"priority,omitempty"`
}

type ManifestMediaTransform struct {
	ID      string `json:"id"`
	Variant string `json:"variant"`
	Format  string `json:"format,omitempty"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
}

type ManifestNavigation struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Kind            string `json:"kind"`
	Action          string `json:"action"`
	TargetID        string `json:"targetId,omitempty"`
	Label           string `json:"label"`
	Href            string `json:"href,omitempty"`
	Permission      string `json:"permission,omitempty"`
	Order           int    `json:"order,omitempty"`
}

type ManifestRegion struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Action          string `json:"action"`
	TargetID        string `json:"targetId,omitempty"`
	Kind            string `json:"kind"`
	Label           string `json:"label"`
	Multiple        bool   `json:"multiple,omitempty"`
}

type ManifestDependency struct {
	ID         string `json:"id,omitempty"`
	Capability string `json:"capability,omitempty"`
	Version    string `json:"version"`
	Kind       string `json:"kind"`
}

type ManifestLifecycle struct {
	ContractVersion string                      `json:"contractVersion"`
	Install         *ManifestLifecycleOperation `json:"install,omitempty"`
	Enable          *ManifestLifecycleOperation `json:"enable,omitempty"`
	Disable         *ManifestLifecycleOperation `json:"disable,omitempty"`
	Upgrade         *ManifestLifecycleOperation `json:"upgrade,omitempty"`
	Rollback        *ManifestLifecycleOperation `json:"rollback,omitempty"`
	Uninstall       *ManifestLifecycleOperation `json:"uninstall,omitempty"`
}

type ManifestLifecycleOperation struct {
	Plan             string `json:"plan"`
	Execute          string `json:"execute"`
	ProgressSchema   string `json:"progressSchema"`
	CheckpointSchema string `json:"checkpointSchema"`
}
