package identity

import "errors"

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
	UserStatusBanned   UserStatus = "banned"
)

var (
	ErrInvalidCredentials        = errors.New("identity: invalid credentials")
	ErrPermissionDenied          = errors.New("identity: permission denied")
	ErrSystemRoleLocked          = errors.New("identity: system role is locked")
	ErrDefaultRoleLocked         = errors.New("identity: default role is locked")
	ErrInitialSuperAdminLocked   = errors.New("identity: initial super admin is locked")
	ErrUsernameOrEmailNotUnique  = errors.New("identity: username or email is not unique")
	ErrPasswordDoesNotMeetPolicy = errors.New("identity: password does not meet policy")
)

type Actor struct {
	ID          int64
	Status      UserStatus
	RoleKeys    []string
	Permissions map[string]bool
}

type PostSummary struct {
	ID           int64
	AuthorUserID int64
}

type CurrentUser struct {
	ID                  int64      `json:"id"`
	Username            string     `json:"username"`
	DisplayName         string     `json:"displayName"`
	Locale              string     `json:"locale"`
	Status              UserStatus `json:"status"`
	IsInitialSuperAdmin bool       `json:"isInitialSuperAdmin"`
	RoleKeys            []string   `json:"roleKeys"`
	Permissions         []string   `json:"permissions"`
}

type RegistrationStatus struct {
	NextUserIsInitialSuperAdmin bool `json:"nextUserIsInitialSuperAdmin"`
}
