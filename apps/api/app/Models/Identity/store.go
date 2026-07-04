package identity

import "context"

type Store interface {
	ActorStore
	WithBootstrapTx(ctx context.Context, fn func(context.Context, TxStore) error) error
	AnyUserExists(ctx context.Context) (bool, error)
	FindRegistrationConflicts(ctx context.Context, username string, email string) (RegistrationConflicts, error)
	GetCurrentUser(ctx context.Context, userID int64) (CurrentUser, error)
	GetCredentialByLogin(ctx context.Context, login string) (CredentialUser, error)
	ListPermissions(ctx context.Context) ([]Permission, error)
	ListPermissionMatrix(ctx context.Context) (PermissionMatrix, error)
	ListUsers(ctx context.Context, input UserListInput) (AdminUserList, error)
	GetAdminUser(ctx context.Context, userID int64) (AdminUserDetail, error)
	ListRoles(ctx context.Context) ([]Role, error)
	CreateRole(ctx context.Context, input RoleInput) (Role, error)
	UpdateRole(ctx context.Context, roleKey string, input RoleInput) (Role, error)
	DeleteRole(ctx context.Context, roleKey string) error
	ReplaceRolePermissions(ctx context.Context, actorUserID int64, roleKey string, permissions []string) error
	ReplaceUserRoles(ctx context.Context, actorUserID int64, targetUserID int64, roleKeys []string) (AdminUserDetail, error)
	ReplaceUserPermissionOverrides(ctx context.Context, actorUserID int64, targetUserID int64, overrides PermissionOverrides) (AdminUserDetail, error)
	RecordLoginAudit(ctx context.Context, input LoginAudit) error
}

type ActorStore interface {
	LoadActor(ctx context.Context, userID int64) (Actor, error)
}

type TxStore interface {
	AnyUserExists(ctx context.Context) (bool, error)
	FindRegistrationConflicts(ctx context.Context, username string, email string) (RegistrationConflicts, error)
	CreateUser(ctx context.Context, input CreateUserInput) (CurrentUser, error)
	CreateCredential(ctx context.Context, userID int64, passwordHash string) error
	GetDefaultRole(ctx context.Context) (Role, error)
	GetRole(ctx context.Context, roleKey string) (Role, error)
	AssignRole(ctx context.Context, userID int64, roleID int64) error
	LoadCurrentUserAccess(ctx context.Context, current *CurrentUser) error
}

type CreateUserInput struct {
	Username            string
	Email               string
	DisplayName         string
	Locale              string
	IsInitialSuperAdmin bool
}

type RegistrationConflicts struct {
	UsernameTaken bool
	EmailTaken    bool
}

type CredentialUser struct {
	CurrentUser
	PasswordHash string
}

type Role struct {
	ID             int64    `json:"id"`
	Key            string   `json:"key"`
	Alias          string   `json:"alias"`
	Description    string   `json:"description"`
	IsSystem       bool     `json:"isSystem"`
	IsDefault      bool     `json:"isDefault"`
	IsDeletable    bool     `json:"isDeletable"`
	IsEnabled      bool     `json:"isEnabled"`
	PermissionKeys []string `json:"permissionKeys"`
}

type RoleInput struct {
	Key         string
	Alias       string
	Description string
}
