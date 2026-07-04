package identity

import "context"

type Store interface {
	WithBootstrapTx(ctx context.Context, fn func(context.Context, TxStore) error) error
	GetCurrentUser(ctx context.Context, userID int64) (CurrentUser, error)
	GetCredentialByLogin(ctx context.Context, login string) (CredentialUser, error)
	LoadActor(ctx context.Context, userID int64) (Actor, error)
	ListRoles(ctx context.Context) ([]Role, error)
	CreateRole(ctx context.Context, input RoleInput) (Role, error)
	UpdateRole(ctx context.Context, roleKey string, input RoleInput) (Role, error)
	DeleteRole(ctx context.Context, roleKey string) error
	ReplaceRolePermissions(ctx context.Context, actorUserID int64, roleKey string, permissions []string) error
}

type TxStore interface {
	AnyUserExists(ctx context.Context) (bool, error)
	CreateUser(ctx context.Context, input CreateUserInput) (CurrentUser, error)
	CreateCredential(ctx context.Context, userID int64, passwordHash string) error
	GetDefaultRole(ctx context.Context) (Role, error)
	GetRole(ctx context.Context, roleKey string) (Role, error)
	AssignRole(ctx context.Context, userID int64, roleID int64) error
}

type CreateUserInput struct {
	Username            string
	Email               string
	DisplayName         string
	Locale              string
	IsInitialSuperAdmin bool
}

type CredentialUser struct {
	CurrentUser
	PasswordHash string
}

type Role struct {
	ID          int64  `json:"id"`
	Key         string `json:"key"`
	Alias       string `json:"alias"`
	Description string `json:"description"`
	IsSystem    bool   `json:"isSystem"`
	IsDefault   bool   `json:"isDefault"`
	IsDeletable bool   `json:"isDeletable"`
	IsEnabled   bool   `json:"isEnabled"`
}

type RoleInput struct {
	Key         string
	Alias       string
	Description string
}
