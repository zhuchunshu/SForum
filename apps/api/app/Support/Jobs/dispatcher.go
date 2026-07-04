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
