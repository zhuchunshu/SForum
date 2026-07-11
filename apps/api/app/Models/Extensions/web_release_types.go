package extensions

import (
	"encoding/json"
	"time"
)

const (
	WebReleaseReloadPrompt = "prompt"
	WebReleaseReloadForce  = "force"

	WebReleaseCheckpointPending = "pending"
)

type WebRelease struct {
	ID                   int64            `json:"id"`
	DesiredGeneration    int64            `json:"desiredGeneration"`
	TriggerKind          string           `json:"triggerKind"`
	TriggerExtensionID   string           `json:"triggerExtensionId,omitempty"`
	CompositionHash      string           `json:"compositionHash"`
	CompositionSnapshot  json.RawMessage  `json:"compositionSnapshot"`
	ActiveThemeID        string           `json:"activeThemeId"`
	ThemeVersion         string           `json:"themeVersion"`
	ThemeLayerPath       string           `json:"themeLayerPath"`
	ThemePackageDigest   string           `json:"themePackageDigest"`
	Status               WebReleaseStatus `json:"status"`
	ActivationCheckpoint string           `json:"activationCheckpoint"`
	ReloadMode           string           `json:"reloadMode"`
	ArtifactPath         string           `json:"artifactPath,omitempty"`
	ArtifactDigest       string           `json:"artifactDigest,omitempty"`
	ServerEntry          string           `json:"serverEntry,omitempty"`
	BuildLog             string           `json:"buildLog,omitempty"`
	PublicReason         string           `json:"publicReason,omitempty"`
	PublicMessage        string           `json:"publicMessage,omitempty"`
	PreviousReleaseID    *int64           `json:"previousReleaseId,omitempty"`
	RequestedByUserID    *int64           `json:"requestedByUserId,omitempty"`
	ActivatedByUserID    *int64           `json:"activatedByUserId,omitempty"`
	CreatedAt            time.Time        `json:"createdAt"`
	UpdatedAt            time.Time        `json:"updatedAt"`
	ReadyAt              *time.Time       `json:"readyAt,omitempty"`
	ActivationStartedAt  *time.Time       `json:"activationStartedAt,omitempty"`
	ActivatedAt          *time.Time       `json:"activatedAt,omitempty"`
	CompletedAt          *time.Time       `json:"completedAt,omitempty"`
}

type WebReleaseExtension struct {
	WebReleaseID                     int64                  `json:"webReleaseId"`
	ExtensionID                      string                 `json:"extensionId"`
	ExtensionVersion                 string                 `json:"extensionVersion"`
	PackageDigest                    string                 `json:"packageDigest"`
	FrontendRoot                     string                 `json:"frontendRoot"`
	ComponentMap                     map[string]string      `json:"componentMap"`
	APIVersion                       int                    `json:"apiVersion"`
	TrustedComponents                []ManifestContribution `json:"trustedComponents"`
	LocaleMap                        map[string]string      `json:"localeMap"`
	LocaleMapDigest                  string                 `json:"localeMapDigest"`
	LockfileDigest                   string                 `json:"lockfileDigest"`
	ResolvedDependencies             []Dependency           `json:"resolvedDependencies"`
	ResolvedDependencySnapshotDigest string                 `json:"resolvedDependencySnapshotDigest,omitempty"`
	SortOrder                        int                    `json:"sortOrder"`
	CreatedAt                        time.Time              `json:"createdAt"`
}

type WebReleaseExtensionEffect struct {
	WebReleaseID         int64      `json:"webReleaseId"`
	ExtensionID          string     `json:"extensionId"`
	PreviousStatus       string     `json:"previousStatus"`
	TargetStatus         string     `json:"targetStatus"`
	ActivationCheckpoint string     `json:"activationCheckpoint"`
	PublicReason         string     `json:"publicReason,omitempty"`
	PublicMessage        string     `json:"publicMessage,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
	CompensatedAt        *time.Time `json:"compensatedAt,omitempty"`
}

type WebReleaseEvent struct {
	ID             int64             `json:"id"`
	WebReleaseID   int64             `json:"webReleaseId"`
	ActorUserID    *int64            `json:"actorUserId,omitempty"`
	PreviousStatus *WebReleaseStatus `json:"previousStatus,omitempty"`
	NextStatus     WebReleaseStatus  `json:"nextStatus"`
	Reason         string            `json:"reason"`
	Message        string            `json:"message"`
	CreatedAt      time.Time         `json:"createdAt"`
}

type WebReleaseDetail struct {
	WebRelease
	Extensions []WebReleaseExtension       `json:"extensions"`
	Effects    []WebReleaseExtensionEffect `json:"effects"`
	Events     []WebReleaseEvent           `json:"events"`
}

type WebReleaseSummary struct {
	ID                 int64            `json:"id"`
	Status             WebReleaseStatus `json:"status"`
	CompositionHash    string           `json:"compositionHash"`
	ReloadMode         string           `json:"reloadMode"`
	TriggerKind        string           `json:"triggerKind,omitempty"`
	TriggerExtensionID string           `json:"triggerExtensionId,omitempty"`
	PublicReason       string           `json:"publicReason,omitempty"`
	PublicMessage      string           `json:"publicMessage,omitempty"`
}

type WebReleasePage struct {
	Items   []WebRelease `json:"items"`
	Total   int64        `json:"total"`
	Page    int          `json:"page"`
	PerPage int          `json:"perPage"`
}

type WebReleaseCreateInput struct {
	TriggerKind         string
	TriggerExtensionID  string
	CompositionHash     string
	CompositionSnapshot json.RawMessage
	ActiveThemeID       string
	ThemeVersion        string
	ThemeLayerPath      string
	ThemePackageDigest  string
	ReloadMode          string
	PreviousReleaseID   *int64
	RequestedByUserID   *int64
	Extensions          []WebReleaseExtensionInput
	Effects             []WebReleaseEffectInput
	Reason              string
	Message             string
}

type WebReleaseExtensionInput struct {
	ExtensionID       string
	ExtensionVersion  string
	PackageDigest     string
	FrontendRoot      string
	ComponentMap      map[string]string
	APIVersion        int
	TrustedComponents []ManifestContribution
	LocaleMap         map[string]string
	LocaleMapDigest   string
	LockfileDigest    string
	SortOrder         int
}

type WebReleaseEffectInput struct {
	ExtensionID    string
	PreviousStatus string
	TargetStatus   string
}

type WebReleaseTransitionInput struct {
	ID                   int64
	ExpectedStatus       WebReleaseStatus
	NextStatus           WebReleaseStatus
	ActivationCheckpoint string
	ArtifactPath         string
	ArtifactDigest       string
	ServerEntry          string
	BuildLog             string
	PublicReason         string
	PublicMessage        string
	ActorUserID          *int64
	ActivatedByUserID    *int64
	Reason               string
	Message              string
	Compensation         bool
}

type WebReleaseListInput struct {
	Status  WebReleaseStatus
	Page    int
	PerPage int
}

type WebReleaseDependencySnapshotInput struct {
	WebReleaseID         int64
	ExtensionID          string
	ResolvedDependencies []Dependency
	Digest               string
}
