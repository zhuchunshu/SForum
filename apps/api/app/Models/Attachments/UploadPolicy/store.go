package uploadpolicy

import "context"

type Store interface {
	RoleLimitsForUser(ctx context.Context, userID int64) ([]RoleLimit, error)
	UserLimit(ctx context.Context, userID int64) (*int64, error)
	ListRolePolicies(ctx context.Context) ([]StoredRolePolicy, error)
	GetUserPolicy(ctx context.Context, userID int64) (StoredUserPolicy, error)
	SetRoleLimit(ctx context.Context, actorUserID int64, roleKey string, maxBytes int64) error
	DeleteRoleLimit(ctx context.Context, actorUserID int64, roleKey string) error
	SetUserLimit(ctx context.Context, actorUserID int64, userID int64, maxBytes int64) error
	DeleteUserLimit(ctx context.Context, actorUserID int64, userID int64) error
}
