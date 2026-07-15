package extensions

import (
	"context"
	"errors"
	"time"
)

const (
	TrustImpactSchemaV1 = "sforum.trust-impact@1"
	TrustImpactSchemaV2 = "sforum.trust-impact@2"
	TrustActionEnable   = "enable"

	CodeTrustChallengeRequired = "extension.trust_challenge_required"
	CodeTrustChallengeInvalid  = "extension.trust_challenge_invalid"
	CodeTrustChallengeExpired  = "extension.trust_challenge_expired"
	CodeTrustChallengeReplayed = "extension.trust_challenge_replayed"
	CodeTrustChallengeStale    = "extension.trust_challenge_stale"
	CodeTrustNotRequired       = "extension.trust_not_required"
)

var (
	ErrTrustChallengeRequired = errors.New("extensions: trust challenge required")
	ErrTrustChallengeInvalid  = errors.New("extensions: trust challenge invalid")
	ErrTrustChallengeExpired  = errors.New("extensions: trust challenge expired")
	ErrTrustChallengeReplayed = errors.New("extensions: trust challenge replayed")
	ErrTrustChallengeStale    = errors.New("extensions: trust challenge stale")
	ErrTrustNotRequired       = errors.New("extensions: executable trust not required")
	ErrTrustGrantNotFound     = errors.New("extensions: executable trust grant not found")
)

type TrustArtifact struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type TrustMigration struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type TrustGuard struct {
	Path       string   `json:"path"`
	Methods    []string `json:"methods"`
	Access     string   `json:"access"`
	Permission string   `json:"permission,omitempty"`
}

type TrustContracts struct {
	HostAPI     string `json:"hostApi"`
	FrontendAPI string `json:"frontendApi,omitempty"`
}

type TrustAuthority struct {
	BackendExecution       bool     `json:"backendExecution"`
	AdminFrontendExecution bool     `json:"adminFrontendExecution"`
	RawRequest             bool     `json:"rawRequest"`
	RawCoreDatabase        bool     `json:"rawCoreDatabase"`
	OutboundNetwork        bool     `json:"outboundNetwork"`
	PackageFiles           []string `json:"packageFiles"`
	Secrets                []string `json:"secrets"`
}

// TrustImpact 是服务端 canonicalize 后参与摘要的完整当前能力文档。
// P2 扩展 Manifest 时必须追加字段，不能从摘要中静默移除既有声明。
type TrustImpact struct {
	SchemaVersion         string                         `json:"schemaVersion"`
	Action                string                         `json:"action"`
	ExtensionID           string                         `json:"extensionId"`
	ExtensionVersion      string                         `json:"extensionVersion"`
	ExtensionType         string                         `json:"extensionType"`
	Source                string                         `json:"source"`
	PackageDigest         string                         `json:"packageDigest"`
	ManifestContract      string                         `json:"manifestContract"`
	ArtifactDigests       map[string]string              `json:"artifactDigests"`
	Binaries              []TrustArtifact                `json:"binaries"`
	Backend               ManifestBackend                `json:"backend"`
	Routes                []ManifestRoute                `json:"routes"`
	Guards                []TrustGuard                   `json:"guards"`
	GuardDeclarations     []ManifestGuard                `json:"guardDeclarations"`
	Hooks                 []ManifestHook                 `json:"hooks"`
	Events                []ManifestEvent                `json:"events"`
	Migrations            []TrustMigration               `json:"migrations"`
	MigrationDeclarations []ManifestMigration            `json:"migrationDeclarations"`
	Providers             []ManifestProvider             `json:"providers"`
	Jobs                  []ManifestJob                  `json:"jobs"`
	Schedules             []ManifestSchedule             `json:"schedules"`
	Components            []SettingsComponent            `json:"components"`
	RegistryComponents    []ManifestComponent            `json:"registryComponents"`
	Templates             []ManifestTemplate             `json:"templates"`
	Assets                []ManifestAsset                `json:"assets"`
	Content               []ManifestContent              `json:"content"`
	Database              *ManifestDatabase              `json:"database"`
	Cache                 []ManifestCache                `json:"cache"`
	Services              []ManifestService              `json:"services"`
	Commands              []ManifestCommand              `json:"commands"`
	AdminSurfaces         []ManifestAdminSurface         `json:"adminSurfaces"`
	Queries               []ManifestQuery                `json:"queries"`
	Identity              *ManifestIdentity              `json:"identity"`
	PermissionDefinitions []ManifestPermissionDefinition `json:"permissionDefinitions"`
	Media                 []ManifestMediaPipeline        `json:"media"`
	Navigation            []ManifestNavigation           `json:"navigation"`
	Regions               []ManifestRegion               `json:"regions"`
	Contributions         []ManifestContribution         `json:"contributions"`
	Capabilities          []CapabilityGrant              `json:"capabilities"`
	Permissions           []string                       `json:"permissions"`
	RequiredFeatures      []string                       `json:"requiredFeatures"`
	Dependencies          []ManifestDependency           `json:"dependencies"`
	Lifecycle             *ManifestLifecycle             `json:"lifecycle"`
	OpenAPI               []ManifestOpenAPIFragment      `json:"openapi"`
	PackageFiles          []ManifestPackageFile          `json:"packageFiles"`
	RequestedAuthority    TrustAuthority                 `json:"requestedAuthority"`
	Contracts             TrustContracts                 `json:"contracts"`
	Digest                string                         `json:"digest"`
}

type TrustChallenge struct {
	Token     string      `json:"token"`
	Impact    TrustImpact `json:"impact"`
	ExpiresAt time.Time   `json:"expiresAt"`
}

type ExecutableTrustStatus struct {
	Impact        TrustImpact `json:"impact"`
	TrustRequired bool        `json:"trustRequired"`
	Trusted       bool        `json:"trusted"`
}

type TrustGrant struct {
	ID               int64      `json:"id"`
	ExtensionID      string     `json:"extensionId"`
	ExtensionVersion string     `json:"extensionVersion"`
	PackageDigest    string     `json:"packageDigest"`
	Action           string     `json:"action"`
	ImpactDigest     string     `json:"impactDigest"`
	GrantedByUserID  int64      `json:"grantedByUserId"`
	GrantedAt        time.Time  `json:"grantedAt"`
	RevokedAt        *time.Time `json:"revokedAt,omitempty"`
	RevokedByUserID  int64      `json:"revokedByUserId,omitempty"`
	RevocationReason string     `json:"revocationReason,omitempty"`
	// created 仅供同包内的激活补偿判断；不得成为 HTTP/SDK 契约。
	created bool
}

// RuntimeTrustIdentity binds a subprocess handshake to the live P1 grant.
// Built-in artifacts use the explicit "builtin" provenance instead of a row.
type RuntimeTrustIdentity struct {
	TrustGrantID string
	ImpactDigest string
}

type TrustIdentity struct {
	ExtensionID      string
	ExtensionVersion string
	PackageDigest    string
	Action           string
	ImpactDigest     string
}

type TrustChallengeRecord struct {
	TokenHash       string
	ActorUserID     int64
	Identity        TrustIdentity
	ArtifactDigests map[string]string
	Impact          TrustImpact
	ExpiresAt       time.Time
}

type TrustConsumeInput struct {
	TokenHash   string
	ActorUserID int64
	Identity    TrustIdentity
}

type ExecutableTrustStore interface {
	CreateChallenge(context.Context, TrustChallengeRecord) error
	HasLiveGrant(context.Context, TrustIdentity) (bool, error)
	LiveGrant(context.Context, TrustIdentity) (TrustGrant, error)
	ConsumeChallenge(context.Context, TrustConsumeInput) (TrustGrant, error)
	RevokeAll(context.Context, string, int64, string) error
}
