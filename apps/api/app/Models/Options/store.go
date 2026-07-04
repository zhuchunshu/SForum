package options

import "context"

type Store interface {
	List(ctx context.Context) ([]Option, error)
	Upsert(ctx context.Context, input UpdateInput) (Option, error)
}
