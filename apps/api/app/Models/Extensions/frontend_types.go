package extensions

import "time"

const (
	AdminFrontendKindNone              = "none"
	AdminFrontendKindLegacyWebRelease  = "legacy_web_release"
	AdminFrontendKindPrebuiltComponent = "prebuilt_component"

	FrontendTrustNone              = "none"
	FrontendTrustRequired          = "required"
	FrontendTrustTrusted           = "trusted"
	FrontendTrustSourceTrusted     = "source_trusted"
	FrontendTrustRevocationPending = "revocation_pending"
	FrontendTrustRevoked           = "revoked"
	FrontendTrustInvalidated       = "invalidated"
)

type FrontendStatus struct {
	ExtensionID    string                 `json:"extensionId"`
	Kind           string                 `json:"kind"`
	Declaration    *ManifestAdminFrontend `json:"declaration,omitempty"`
	Component      *SettingsComponent     `json:"component,omitempty"`
	TrustState     string                 `json:"trustState"`
	Digest         string                 `json:"digest,omitempty"`
	ArtifactActive bool                   `json:"artifactActive,omitempty"`
	BuildRequired  bool                   `json:"buildRequired,omitempty"`
	Dependencies   DependencySummary      `json:"dependencies"`
}

type GrantFrontendInput struct {
	PackageDigest string                     `json:"packageDigest"`
	Confirmation  *FrontendTrustConfirmation `json:"confirmation,omitempty"`
}

// FrontendTrustConfirmation 只证明管理员有意授权当前声明；真正边界仍是服务端保存的 digest grant。
type FrontendTrustConfirmation struct {
	ChallengeID  string `json:"challengeId"`
	Code         string `json:"code"`
	ExtensionID  string `json:"extensionId"`
	Version      string `json:"version"`
	Digest       string `json:"digest"`
	APIVersion   int    `json:"apiVersion"`
	ComponentID  string `json:"componentId"`
	Phrase       string `json:"phrase"`
	Acknowledged bool   `json:"acknowledged"`
}

type FrontendTrustChallenge struct {
	ChallengeID string    `json:"challengeId"`
	Code        string    `json:"code"`
	ExtensionID string    `json:"extensionId"`
	Version     string    `json:"version"`
	Digest      string    `json:"digest"`
	APIVersion  int       `json:"apiVersion"`
	ComponentID string    `json:"componentId"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type FrontendAsset struct {
	Body        []byte
	ContentType string
	ETag        string
}

type ExtensionOperation struct {
	Extension  Extension          `json:"extension"`
	Frontend   *FrontendStatus    `json:"frontend,omitempty"`
	WebRelease *WebReleaseSummary `json:"webRelease,omitempty"`
	Queued     bool               `json:"queued"`
}

type FrontendTrustGrant struct {
	ID                          int64      `json:"id"`
	ExtensionID                 string     `json:"extensionId"`
	ExtensionVersion            string     `json:"extensionVersion"`
	PackageDigest               string     `json:"packageDigest"`
	AdminFrontendDigest         string     `json:"adminFrontendDigest"`
	APIVersion                  int        `json:"apiVersion"`
	ContributionPoints          []string   `json:"contributionPoints"`
	ComponentIDs                []string   `json:"componentIds"`
	GrantedByUserID             int64      `json:"grantedByUserId"`
	GrantedAt                   time.Time  `json:"grantedAt"`
	RevocationRequestedAt       *time.Time `json:"revocationRequestedAt,omitempty"`
	RevocationRequestedByUserID int64      `json:"revocationRequestedByUserId,omitempty"`
	RevokedAt                   *time.Time `json:"revokedAt,omitempty"`
	RevokedByUserID             int64      `json:"revokedByUserId,omitempty"`
}

type FrontendTrustGrantInput struct {
	ExtensionID         string
	ExtensionVersion    string
	PackageDigest       string
	AdminFrontendDigest string
	APIVersion          int
	ContributionPoints  []string
	ComponentIDs        []string
	GrantedByUserID     int64
}

type FrontendRevocationInput struct {
	ExtensionID         string
	ExtensionVersion    string
	PackageDigest       string
	AdminFrontendDigest string
	RequestedByUserID   int64
}

type FrontendFinalizeInput struct {
	ExtensionID         string
	ExtensionVersion    string
	PackageDigest       string
	AdminFrontendDigest string
}
