package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type testArgs struct {
	ID int64 `json:"id" river:"unique"`
}

func (testArgs) Kind() string { return "test.args" }

type fakeRiverClient struct {
	insertArgs   river.JobArgs
	insertOpts   *river.InsertOpts
	insertTx     pgx.Tx
	insertManyN  int
	insertManyOk bool
	err          error
}

func (f *fakeRiverClient) Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	f.insertArgs = args
	f.insertOpts = opts
	return &rivertype.JobInsertResult{}, f.err
}

func (f *fakeRiverClient) InsertTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	f.insertTx = tx
	f.insertArgs = args
	f.insertOpts = opts
	return &rivertype.JobInsertResult{}, f.err
}

func (f *fakeRiverClient) InsertMany(_ context.Context, params []river.InsertManyParams) ([]*rivertype.JobInsertResult, error) {
	f.insertManyN = len(params)
	f.insertManyOk = true
	if f.err != nil {
		return nil, f.err
	}
	results := make([]*rivertype.JobInsertResult, len(params))
	for i := range results {
		results[i] = &rivertype.JobInsertResult{}
	}
	return results, nil
}

func TestDispatcherEnqueueConvertsOptions(t *testing.T) {
	client := &fakeRiverClient{}
	dispatcher := NewDispatcher(client)
	runAt := time.Now().UTC().Add(time.Hour)

	_, err := dispatcher.Enqueue(context.Background(), testArgs{ID: 42}, EnqueueOptions{
		Queue:       QueueSearch,
		MaxAttempts: 3,
		ScheduledAt: runAt,
		Unique:      river.UniqueOpts{ByArgs: true},
	})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if client.insertArgs.Kind() != "test.args" {
		t.Fatalf("expected args kind test.args, got %q", client.insertArgs.Kind())
	}
	if client.insertOpts.Queue != QueueSearch {
		t.Fatalf("expected search queue, got %q", client.insertOpts.Queue)
	}
	if client.insertOpts.MaxAttempts != 3 {
		t.Fatalf("expected max attempts 3, got %d", client.insertOpts.MaxAttempts)
	}
	if !client.insertOpts.ScheduledAt.Equal(runAt) {
		t.Fatalf("expected scheduled time %s, got %s", runAt, client.insertOpts.ScheduledAt)
	}
	if !client.insertOpts.UniqueOpts.ByArgs {
		t.Fatal("expected unique by args")
	}
}

func TestDispatcherEnqueueTxForwardsTransaction(t *testing.T) {
	client := &fakeRiverClient{}
	dispatcher := NewDispatcher(client)

	_, err := dispatcher.EnqueueTx(context.Background(), nil, testArgs{ID: 7}, EnqueueOptions{
		Queue: QueueCritical,
	})
	if err != nil {
		t.Fatalf("enqueue tx: %v", err)
	}

	if client.insertArgs.Kind() != "test.args" {
		t.Fatalf("expected args kind test.args, got %q", client.insertArgs.Kind())
	}
	if client.insertOpts.Queue != QueueCritical {
		t.Fatalf("expected critical queue, got %q", client.insertOpts.Queue)
	}
	if client.insertTx != nil {
		t.Fatalf("expected nil test tx to be forwarded, got %#v", client.insertTx)
	}
}

func TestDispatcherEnqueueReturnsClientErrors(t *testing.T) {
	expected := errors.New("insert failed")
	dispatcher := NewDispatcher(&fakeRiverClient{err: expected})

	if _, err := dispatcher.Enqueue(context.Background(), testArgs{ID: 1}, EnqueueOptions{}); !errors.Is(err, expected) {
		t.Fatalf("expected insert error, got %v", err)
	}
}

func TestDispatcherEnqueueManyBatchesInsert(t *testing.T) {
	client := &fakeRiverClient{}
	dispatcher := NewDispatcher(client)

	results, err := dispatcher.EnqueueMany(context.Background(),
		[]river.JobArgs{testArgs{ID: 1}, testArgs{ID: 2}, testArgs{ID: 3}},
		EnqueueOptions{Queue: QueueSearch},
	)
	if err != nil {
		t.Fatalf("enqueue many: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if !client.insertManyOk {
		t.Fatal("expected InsertMany to be invoked")
	}
	if client.insertManyN != 3 {
		t.Fatalf("expected 3 params batched, got %d", client.insertManyN)
	}
}

func TestDispatcherEnqueueManyEmptyIsNoop(t *testing.T) {
	client := &fakeRiverClient{}
	dispatcher := NewDispatcher(client)

	results, err := dispatcher.EnqueueMany(context.Background(), nil, EnqueueOptions{})
	if err != nil {
		t.Fatalf("enqueue many empty: %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results for empty input, got %v", results)
	}
	if client.insertManyOk {
		t.Fatal("expected InsertMany NOT to be invoked for empty input")
	}
}
