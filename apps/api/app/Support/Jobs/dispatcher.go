package jobs

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type RiverClient interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
	InsertTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
	InsertMany(ctx context.Context, params []river.InsertManyParams) ([]*rivertype.JobInsertResult, error)
}

type Dispatcher struct {
	client RiverClient
}

func NewDispatcher(client RiverClient) *Dispatcher {
	return &Dispatcher{client: client}
}

func (d *Dispatcher) Enqueue(ctx context.Context, args river.JobArgs, opts EnqueueOptions) (*rivertype.JobInsertResult, error) {
	return d.client.Insert(ctx, args, opts.RiverInsertOpts())
}

func (d *Dispatcher) EnqueueTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts EnqueueOptions) (*rivertype.JobInsertResult, error) {
	return d.client.InsertTx(ctx, tx, args, opts.RiverInsertOpts())
}

// EnqueueMany 批量入队同质 job（共享同一 EnqueueOptions）。
// 内部用 River InsertMany 单次批量 INSERT，比循环单条快一个数量级，
// 适合重建索引这类一次性入队成千上万个 job 的场景。
// River 的 Unique ByArgs 会自动去重，重复入队安全幂等。
func (d *Dispatcher) EnqueueMany(ctx context.Context, argsList []river.JobArgs, opts EnqueueOptions) ([]*rivertype.JobInsertResult, error) {
	if len(argsList) == 0 {
		return nil, nil
	}
	insertOpts := opts.RiverInsertOpts()
	params := make([]river.InsertManyParams, 0, len(argsList))
	for _, args := range argsList {
		params = append(params, river.InsertManyParams{Args: args, InsertOpts: insertOpts})
	}
	return d.client.InsertMany(ctx, params)
}
