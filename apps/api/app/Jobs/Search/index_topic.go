package searchjobs

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"

	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

type TopicIndexer interface {
	IndexTopic(ctx context.Context, topicID int64) error
}

type IndexTopicArgs struct {
	TopicID int64 `json:"topic_id" river:"unique"`
}

func (IndexTopicArgs) Kind() string {
	return "search.index_topic"
}

func (IndexTopicArgs) EnqueueOptions() supportjobs.EnqueueOptions {
	return supportjobs.EnqueueOptions{
		Queue:       supportjobs.QueueSearch,
		MaxAttempts: 10,
		Unique:      river.UniqueOpts{ByArgs: true},
	}
}

type IndexTopicWorker struct {
	river.WorkerDefaults[IndexTopicArgs]
	Indexer TopicIndexer
}

func (w *IndexTopicWorker) Work(ctx context.Context, job *river.Job[IndexTopicArgs]) error {
	if job.Args.TopicID <= 0 {
		return fmt.Errorf("index topic job requires positive topic id: %d", job.Args.TopicID)
	}
	if w.Indexer == nil {
		return fmt.Errorf("index topic worker requires indexer")
	}
	return w.Indexer.IndexTopic(ctx, job.Args.TopicID)
}

func Register(registry *supportjobs.Registry, indexer TopicIndexer) {
	registry.Add(func(workers *river.Workers) error {
		return river.AddWorkerSafely[IndexTopicArgs](workers, &IndexTopicWorker{Indexer: indexer})
	})
}
