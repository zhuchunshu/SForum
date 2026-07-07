package searchjobs

import (
	"context"
	"testing"

	"github.com/riverqueue/river"

	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

type fakeTopicIndexer struct {
	indexedIDs []int64
	deletedIDs []int64
	err        error
}

func (f *fakeTopicIndexer) IndexTopic(_ context.Context, topicID int64) error {
	f.indexedIDs = append(f.indexedIDs, topicID)
	return f.err
}

func (f *fakeTopicIndexer) DeleteTopic(_ context.Context, topicID int64) error {
	f.deletedIDs = append(f.deletedIDs, topicID)
	return f.err
}

func TestIndexTopicArgsKindAndOptions(t *testing.T) {
	args := IndexTopicArgs{TopicID: 42}

	if args.Kind() != "search.index_topic" {
		t.Fatalf("expected search.index_topic kind, got %q", args.Kind())
	}

	opts := args.EnqueueOptions()
	if opts.Queue != supportjobs.QueueSearch {
		t.Fatalf("expected search queue, got %q", opts.Queue)
	}
	if !opts.Unique.ByArgs {
		t.Fatal("expected unique by args")
	}
	if opts.MaxAttempts != 10 {
		t.Fatalf("expected max attempts 10, got %d", opts.MaxAttempts)
	}
}

func TestIndexTopicWorkerCallsIndexer(t *testing.T) {
	indexer := &fakeTopicIndexer{}
	worker := &IndexTopicWorker{Indexer: indexer}

	err := worker.Work(context.Background(), &river.Job[IndexTopicArgs]{
		Args: IndexTopicArgs{TopicID: 42},
	})
	if err != nil {
		t.Fatalf("work: %v", err)
	}
	if len(indexer.indexedIDs) != 1 || indexer.indexedIDs[0] != 42 {
		t.Fatalf("expected topic 42 indexed, got %v", indexer.indexedIDs)
	}
}

func TestIndexTopicWorkerRejectsInvalidTopicID(t *testing.T) {
	worker := &IndexTopicWorker{Indexer: &fakeTopicIndexer{}}

	err := worker.Work(context.Background(), &river.Job[IndexTopicArgs]{
		Args: IndexTopicArgs{TopicID: 0},
	})
	if err == nil {
		t.Fatal("expected invalid topic id error")
	}
}

func TestRegisterAddsWorker(t *testing.T) {
	registry := supportjobs.NewRegistry()
	Register(registry, &fakeTopicIndexer{})

	workers, err := registry.Build()
	if err != nil {
		t.Fatalf("build workers: %v", err)
	}
	if workers == nil {
		t.Fatal("expected workers bundle")
	}
}

func TestDeleteTopicArgsKindAndOptions(t *testing.T) {
	args := DeleteTopicArgs{TopicID: 7}
	if args.Kind() != "search.delete_topic" {
		t.Fatalf("expected search.delete_topic kind, got %q", args.Kind())
	}
	opts := args.EnqueueOptions()
	if opts.Queue != supportjobs.QueueSearch {
		t.Fatalf("expected search queue, got %q", opts.Queue)
	}
	if opts.MaxAttempts != 10 {
		t.Fatalf("expected max attempts 10, got %d", opts.MaxAttempts)
	}
}

func TestDeleteTopicWorkerCallsIndexer(t *testing.T) {
	indexer := &fakeTopicIndexer{}
	worker := &DeleteTopicWorker{Indexer: indexer}

	err := worker.Work(context.Background(), &river.Job[DeleteTopicArgs]{
		Args: DeleteTopicArgs{TopicID: 9},
	})
	if err != nil {
		t.Fatalf("work: %v", err)
	}
	if len(indexer.deletedIDs) != 1 || indexer.deletedIDs[0] != 9 {
		t.Fatalf("expected topic 9 deleted, got %v", indexer.deletedIDs)
	}
}

func TestDeleteTopicWorkerRejectsInvalidTopicID(t *testing.T) {
	worker := &DeleteTopicWorker{Indexer: &fakeTopicIndexer{}}

	err := worker.Work(context.Background(), &river.Job[DeleteTopicArgs]{
		Args: DeleteTopicArgs{TopicID: 0},
	})
	if err == nil {
		t.Fatal("expected invalid topic id error")
	}
}
