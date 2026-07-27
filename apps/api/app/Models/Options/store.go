package options

import "context"

type Store interface {
	List(ctx context.Context) ([]Option, error)
	InsertMissing(ctx context.Context, input UpdateInput) error
	Upsert(ctx context.Context, input UpdateInput) (Option, error)
}

// BatchUpsertStore is deliberately optional so existing non-PostgreSQL stores
// keep their current behavior. PostgreSQL uses it for registration-policy
// writes, where enabled and mode must become visible as one transaction.
type BatchUpsertStore interface {
	UpsertMany(context.Context, []UpdateInput) ([]Option, error)
}
