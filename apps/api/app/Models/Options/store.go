package options

import "context"

type Store interface {
	List(ctx context.Context) ([]Option, error)
	InsertMissing(ctx context.Context, input UpdateInput) error
	Upsert(ctx context.Context, input UpdateInput) (Option, error)
}
