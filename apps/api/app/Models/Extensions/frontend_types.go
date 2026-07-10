package extensions

import "time"

type FrontendTrustGrant struct {
	ID                          int64      `json:"id"`
	ExtensionID                 string     `json:"extensionId"`
	ExtensionVersion            string     `json:"extensionVersion"`
	PackageDigest               string     `json:"packageDigest"`
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
	ExtensionID        string
	ExtensionVersion   string
	PackageDigest      string
	APIVersion         int
	ContributionPoints []string
	ComponentIDs       []string
	GrantedByUserID    int64
}

type FrontendRevocationInput struct {
	ExtensionID       string
	ExtensionVersion  string
	PackageDigest     string
	RequestedByUserID int64
}
