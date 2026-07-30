package uploadpolicy

import (
	"errors"
	"time"
)

const (
	SourceSite = "site"
	SourceRole = "role"
	SourceUser = "user"

	ReasonAllowed          = "attachment.upload_allowed"
	ReasonUploadDisabled   = "attachment.upload_disabled"
	ReasonPermissionDenied = "permission.denied"

	MultipartOverheadReserveBytes int64 = 1024 * 1024
)

var (
	ErrInvalidPolicy  = errors.New("attachment upload policy: invalid policy")
	ErrProtectedActor = errors.New("attachment upload policy: protected actor")
)

type GlobalPolicy struct {
	UploadEnabled         bool
	SiteMaxFileSizeBytes  int64
	TransportMaxBodyBytes int64
}

type EffectivePolicy struct {
	Allowed                   bool   `json:"allowed"`
	Reason                    string `json:"reason"`
	Source                    string `json:"source"`
	EffectiveMaxFileSizeBytes int64  `json:"effectiveMaxFileSizeBytes"`
	SiteMaxFileSizeBytes      int64  `json:"siteMaxFileSizeBytes"`
	TransportMaxFileSizeBytes int64  `json:"transportMaxFileSizeBytes"`
}

type RoleLimit struct {
	RoleKey          string
	MaxFileSizeBytes *int64
}

type RolePolicy struct {
	RoleKey                   string     `json:"roleKey"`
	Alias                     string     `json:"alias"`
	Enabled                   bool       `json:"enabled"`
	GrantsUpload              bool       `json:"grantsUpload"`
	Protected                 bool       `json:"protected"`
	ConfiguredMaxFileSizeMB   *int       `json:"configuredMaxFileSizeMb"`
	EffectiveMaxFileSizeBytes int64      `json:"effectiveMaxFileSizeBytes"`
	UpdatedAt                 *time.Time `json:"updatedAt,omitempty"`
}

type RolePolicyCatalog struct {
	UploadEnabled             bool         `json:"uploadEnabled"`
	SiteMaxFileSizeBytes      int64        `json:"siteMaxFileSizeBytes"`
	TransportMaxFileSizeBytes int64        `json:"transportMaxFileSizeBytes"`
	Items                     []RolePolicy `json:"items"`
}

type UserPolicy struct {
	UserID                    int64      `json:"userId"`
	Username                  string     `json:"username"`
	DisplayName               string     `json:"displayName"`
	Status                    string     `json:"status"`
	RoleKeys                  []string   `json:"roleKeys"`
	CanUpload                 bool       `json:"canUpload"`
	Protected                 bool       `json:"protected"`
	ConfiguredMaxFileSizeMB   *int       `json:"configuredMaxFileSizeMb"`
	EffectiveMaxFileSizeBytes int64      `json:"effectiveMaxFileSizeBytes"`
	Source                    string     `json:"source"`
	Reason                    string     `json:"reason"`
	UpdatedAt                 *time.Time `json:"updatedAt,omitempty"`
}

type LimitInput struct {
	MaxFileSizeMB int `json:"maxFileSizeMb"`
}

type StoredRolePolicy struct {
	RoleKey          string
	Alias            string
	Enabled          bool
	GrantsUpload     bool
	MaxFileSizeBytes *int64
	UpdatedAt        *time.Time
}

type StoredUserPolicy struct {
	UserID           int64
	Username         string
	DisplayName      string
	Status           string
	MaxFileSizeBytes *int64
	UpdatedAt        *time.Time
}
