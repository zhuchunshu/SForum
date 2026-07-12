package apitokens

import (
	"context"
	"time"
)

type Store interface {
	Create(ctx context.Context, userID int64, publicID, tokenHash, name string, scopes []string, expiresAt *time.Time) (Record, error)
	ListByUser(ctx context.Context, userID int64, includeRevoked bool) ([]Record, error)
	GetByPublicID(ctx context.Context, publicID string) (Record, error)
	GetByIDForUser(ctx context.Context, userID, id int64) (Record, error)
	Revoke(ctx context.Context, userID, id int64) error
	// Rotate：撤销旧 token 并创建新 token（同名/scopes）。
	TouchLastUsed(ctx context.Context, id int64) error
}
