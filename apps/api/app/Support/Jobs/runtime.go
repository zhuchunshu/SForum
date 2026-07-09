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
	return NewClientWithPeriodic(pool, cfg, workers, nil)
}

// NewClientWithPeriodic 创建支持周期任务（PeriodicJobs）的 worker client。
// periodicJobs 为 nil/空时退化为普通 NewClient 行为（向后兼容）。
// 周期任务不经过队列调度，由 River 按 cron 周期触发；需在 worker 进程（非 insert-only）注册。
func NewClientWithPeriodic(pool *pgxpool.Pool, cfg Config, workers *river.Workers, periodicJobs []*river.PeriodicJob) (*Client, error) {
	rcfg := &river.Config{
		Queues:  cfg.RiverQueues(),
		Workers: workers,
	}
	if len(periodicJobs) > 0 {
		rcfg.PeriodicJobs = periodicJobs
	}
	return river.NewClient(riverpgxv5.New(pool), rcfg)
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
