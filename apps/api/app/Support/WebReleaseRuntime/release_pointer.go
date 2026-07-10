package webreleaseruntime

import "time"

type CurrentRelease struct {
	SchemaVersion   int       `json:"schemaVersion"`
	ReleaseID       int64     `json:"releaseId"`
	CompositionHash string    `json:"compositionHash"`
	ArtifactPath    string    `json:"artifactPath"`
	ArtifactDigest  string    `json:"artifactDigest"`
	ServerEntry     string    `json:"serverEntry"`
	ThemeID         string    `json:"themeId"`
	ThemeVersion    string    `json:"themeVersion"`
	ReloadMode      string    `json:"reloadMode"`
	RequestedAt     time.Time `json:"requestedAt"`
}

type ActiveRelease struct {
	ReleaseID       int64     `json:"releaseId"`
	CompositionHash string    `json:"compositionHash"`
	ArtifactDigest  string    `json:"artifactDigest"`
	ServerEntry     string    `json:"serverEntry"`
	ThemeID         string    `json:"themeId"`
	ThemeVersion    string    `json:"themeVersion"`
	ReloadMode      string    `json:"reloadMode"`
	SwitchedAt      time.Time `json:"switchedAt"`
}

type Failure struct {
	ReleaseID int64     `json:"releaseId"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	FailedAt  time.Time `json:"failedAt"`
}
