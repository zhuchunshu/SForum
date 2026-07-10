package extensionjobs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/riverqueue/river"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

type WebReleaseCleanupArgs struct{}

func (WebReleaseCleanupArgs) Kind() string { return "extension.web_release_cleanup" }

func (WebReleaseCleanupArgs) EnqueueOptions() supportjobs.EnqueueOptions {
	return supportjobs.EnqueueOptions{
		Queue:       supportjobs.QueueMaintenance,
		MaxAttempts: 3,
		Unique:      river.UniqueOpts{ByArgs: true},
	}
}

type WebReleaseCleanupStore interface {
	CleanupWebReleases(context.Context, time.Time) (extensions.WebReleaseCleanupResult, error)
}

type WebReleaseCleanupWorker struct {
	river.WorkerDefaults[WebReleaseCleanupArgs]
	Store       WebReleaseCleanupStore
	ReleaseRoot string
	Now         func() time.Time
}

func (w *WebReleaseCleanupWorker) Work(ctx context.Context, _ *river.Job[WebReleaseCleanupArgs]) error {
	if w.Store == nil {
		return fmt.Errorf("web release cleanup requires a store")
	}
	now := time.Now
	if w.Now != nil {
		now = w.Now
	}
	result, err := w.Store.CleanupWebReleases(ctx, now().UTC())
	if err != nil {
		return err
	}
	root, err := filepath.Abs(w.ReleaseRoot)
	if err != nil {
		return fmt.Errorf("resolve web release root: %w", err)
	}
	for _, releaseID := range result.ArtifactReleaseIDs {
		if releaseID <= 0 {
			return fmt.Errorf("invalid web release cleanup id %d", releaseID)
		}
		releaseDir := filepath.Join(root, "releases", strconv.FormatInt(releaseID, 10))
		if err := os.RemoveAll(releaseDir); err != nil {
			return fmt.Errorf("remove web release %d artifacts: %w", releaseID, err)
		}
	}
	return nil
}

func RegisterWebReleaseCleanupWorker(registry *supportjobs.Registry, store WebReleaseCleanupStore, releaseRoot string) {
	registry.Add(func(workers *river.Workers) error {
		return river.AddWorkerSafely[WebReleaseCleanupArgs](workers, &WebReleaseCleanupWorker{Store: store, ReleaseRoot: releaseRoot})
	})
}
