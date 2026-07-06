package jobs

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

type Client = river.Client[pgx.Tx]

func NewClient(pool *pgxpool.Pool, cfg Config, workers *river.Workers) (*Client, error) {
	return river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:  cfg.RiverQueues(),
		Workers: workers,
	})
}

func NewInsertOnlyClient(pool *pgxpool.Pool, _ Config) (*Client, error) {
	// River 省略 Queues/Workers 时按 insert-only client 处理，允许 API 进程只入队、不注册执行器。
	return river.NewClient(riverpgxv5.New(pool), &river.Config{})
}

func Start(ctx context.Context, client *Client) error {
	if client == nil {
		return nil
	}
	return client.Start(ctx)
}

func Stop(ctx context.Context, client *Client) error {
	if client == nil {
		return nil
	}
	return client.Stop(ctx)
}
