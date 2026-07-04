package searchjobs

import (
	"context"
	"testing"

	"github.com/riverqueue/river"

	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

type fakeTopicIndexer struct {
	topicID int64
	err     error
}

func (f *fakeTopicIndexer) IndexTopic(ctx context.Context, topicID int64) error {
	f.topicID = topicID
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
	if indexer.topicID != 42 {
		t.Fatalf("expected topic 42, got %d", indexer.topicID)
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
