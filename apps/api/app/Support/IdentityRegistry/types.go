package identityregistry

import "errors"

const SchemaVersion = "sforum.identity-registry@1"

const (
	ProviderKindAuth     = "auth"
	ProviderKindProfile  = "profile"
	ProviderKindRecovery = "recovery"
	ProviderKindSession  = "session"
	ProviderKindRisk     = "risk"
)

const (
	TombstoneKindPermission = "permission"
	TombstoneKindUserField  = "user_field"
	TombstoneKindProvider   = "provider"
)

var (
	ErrInvalid            = errors.New("identity registry declaration is invalid")
	ErrConflict           = errors.New("identity registry conflicts with the active graph")
	ErrArtifactConflict   = errors.New("identity registry artifact does not own the active publication")
	ErrRevisionConflict   = errors.New("identity registry revision changed during replacement")
	ErrSafeMode           = errors.New("identity registry rejects third-party publication in safe mode")
	ErrNotFound           = errors.New("identity registry declaration is not found")
	ErrSchemaUnavailable  = errors.New("identity registry schema is unavailable")
	ErrSchemaValueInvalid = errors.New("identity registry schema rejected the value")
)

// Artifact binds declarations to one immutable extension version. Third-party
// executable identity surfaces additionally require RuntimeInstanceID; inert
// permission and field catalogs remain useful without a backend. Host-sealed
// Core handlers execute in-process and therefore have no plugin runtime id.
type Artifact struct {
	ExtensionID       string `json:"extensionId"`
	ExtensionVersion  string `json:"extensionVersion"`
	PackageDigest     string `json:"packageDigest"`
	VersionID         int64  `json:"versionId,omitempty"`
	RuntimeInstanceID string `json:"runtimeInstanceId,omitempty"`
	Core              bool   `json:"core,omitempty"`
	coreSeal          [32]byte
}

// PermissionDefinition is descriptive catalog material only. There is
// intentionally no grant/assign flag in this contract: Host role management is
// the sole authority that may turn a recommendation into an assignment.
type PermissionDefinition struct {
	Key                string            `json:"key"`
	ContractVersion    string            `json:"contractVersion"`
	Label              string            `json:"label"`
	Description        string            `json:"description"`
	LabelLocales       map[string]string `json:"labelLocales,omitempty"`
	DescriptionLocales map[string]string `json:"descriptionLocales,omitempty"`
	RecommendedRoles   []string          `json:"recommendedRoles,omitempty"`
	AssignmentPolicy   string            `json:"assignmentPolicy"`
}

type UserField struct {
	ID                  string `json:"id"`
	ContractVersion     string `json:"contractVersion"`
	Type                string `json:"type"`
	Schema              string `json:"schema"`
	SchemaWireReference string `json:"schemaWireReference,omitempty"`
	SchemaDigest        string `json:"schemaDigest,omitempty"`
	// Empty permission fields defer to a future Host-owned field policy and
	// must be consumed as default-deny, never as implicit public read/write.
	ReadPermission  string `json:"readPermission,omitempty"`
	WritePermission string `json:"writePermission,omitempty"`

	boundSchema *compiledIdentitySchema
}

const (
	ProviderFailureFailClosed = "fail_closed"
	ProviderFailureOmit       = "omit"
)

type ProviderOperation struct {
	Name                      string `json:"name"`
	InputSchema               string `json:"inputSchema"`
	InputSchemaWireReference  string `json:"inputSchemaWireReference,omitempty"`
	InputSchemaDigest         string `json:"inputSchemaDigest,omitempty"`
	OutputSchema              string `json:"outputSchema"`
	OutputSchemaWireReference string `json:"outputSchemaWireReference,omitempty"`
	OutputSchemaDigest        string `json:"outputSchemaDigest,omitempty"`
	TimeoutMS                 int    `json:"timeoutMs"`
	FailurePolicy             string `json:"failurePolicy"`

	boundInputSchema  *compiledIdentitySchema
	boundOutputSchema *compiledIdentitySchema
}

// Provider stays inspectable when Operations is empty. Non-empty operations
// require exact package Schema material and one live exact runtime.
//
// Label / LabelLocales / Icon 是插件声明的展示元数据（非可执行状态）。
// Host 公共 catalog 按请求 locale 解析后注入前端；Core 不得硬编码供应商品牌文案。
type Provider struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Kind            string `json:"kind"`
	Handler         string `json:"handler"`
	Priority        int    `json:"priority,omitempty"`
	// Label 默认展示名（无 locale 匹配时回退）。
	Label string `json:"label,omitempty"`
	// LabelLocales 插件提供的多语文案，例如 {"zh-CN":"GitHub","en-US":"GitHub"}。
	LabelLocales map[string]string `json:"labelLocales,omitempty"`
	// Icon 为 Iconify / Nuxt Icon 名称（如 i-tabler-brand-github）；空则 Host 用通用图标。
	Icon       string              `json:"icon,omitempty"`
	Operations []ProviderOperation `json:"operations,omitempty"`
}

type IdentityDeclaration struct {
	ContractVersion string      `json:"contractVersion"`
	UserFields      []UserField `json:"userFields,omitempty"`
	Providers       []Provider  `json:"providers,omitempty"`
	SessionPolicy   string      `json:"sessionPolicy,omitempty"`
	RiskHooks       []string    `json:"riskHooks,omitempty"`
}

type Publication struct {
	Artifact    Artifact               `json:"artifact"`
	Identity    *IdentityDeclaration   `json:"identity,omitempty"`
	Permissions []PermissionDefinition `json:"permissions,omitempty"`
}

type PermissionContribution struct {
	PermissionDefinition
	Artifact Artifact `json:"artifact"`
}

type UserFieldContribution struct {
	UserField
	Artifact Artifact `json:"artifact"`
}

type ProviderContribution struct {
	Provider
	Artifact Artifact `json:"artifact"`
}

// ProviderResolution is one atomic process-local provider claim. Revision and
// digest are inspection evidence; validation is scoped to Safe Mode and the
// exact provider/artifact so unrelated Registry publications do not interrupt
// an admitted call.
type ProviderResolution struct {
	Revision uint64               `json:"revision"`
	Digest   string               `json:"digest"`
	SafeMode bool                 `json:"safeMode,omitempty"`
	Provider ProviderContribution `json:"provider"`
}

// Tombstone permanently reserves a stable declaration identity for its first
// owner. Durable storage must restore this list after restart before publishing
// third-party declarations; otherwise old role grants could be reinterpreted by
// a different plugin.
type Tombstone struct {
	Kind             string `json:"kind"`
	ID               string `json:"id"`
	ContractVersion  string `json:"contractVersion"`
	OwnerExtensionID string `json:"ownerExtensionId"`
}

type Snapshot struct {
	SchemaVersion string                   `json:"schemaVersion"`
	Revision      uint64                   `json:"revision"`
	Digest        string                   `json:"digest"`
	SafeMode      bool                     `json:"safeMode,omitempty"`
	Publications  []Publication            `json:"publications"`
	Permissions   []PermissionContribution `json:"permissions"`
	UserFields    []UserFieldContribution  `json:"userFields"`
	Providers     []ProviderContribution   `json:"providers"`
	Tombstones    []Tombstone              `json:"tombstones"`
}
