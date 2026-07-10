package extensionjobs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/riverqueue/river"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

func TestWebReleaseCleanupArgsUseMaintenanceQueue(t *testing.T) {
	opts := (WebReleaseCleanupArgs{}).EnqueueOptions()
	if (WebReleaseCleanupArgs{}).Kind() != "extension.web_release_cleanup" {
		t.Fatal("unexpected cleanup job kind")
	}
	if opts.Queue != supportjobs.QueueMaintenance || opts.MaxAttempts != 3 || !opts.Unique.ByArgs {
		t.Fatalf("unexpected cleanup options: %#v", opts)
	}
}

func TestSelectWebReleaseCleanupKeepsActiveRollbackAndFiveNewestSuccesses(t *testing.T) {
	now := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	rollbackID := int64(8)
	releases := []extensions.WebReleaseCleanupRecord{
		{ID: 10, Status: extensions.WebReleaseActive, PreviousReleaseID: &rollbackID, CompletedAt: ptrTime(now.Add(-time.Hour)), HasArtifact: true},
		{ID: 9, Status: extensions.WebReleaseInactive, CompletedAt: ptrTime(now.Add(-2 * time.Hour)), HasArtifact: true},
		{ID: 8, Status: extensions.WebReleaseInactive, CompletedAt: ptrTime(now.Add(-3 * time.Hour)), HasArtifact: true},
		{ID: 7, Status: extensions.WebReleaseInactive, CompletedAt: ptrTime(now.Add(-4 * time.Hour)), HasArtifact: true},
		{ID: 6, Status: extensions.WebReleaseInactive, CompletedAt: ptrTime(now.Add(-5 * time.Hour)), HasArtifact: true},
		{ID: 5, Status: extensions.WebReleaseInactive, CompletedAt: ptrTime(now.Add(-6 * time.Hour)), HasArtifact: true},
		{ID: 4, Status: extensions.WebReleaseInactive, CompletedAt: ptrTime(now.Add(-7 * time.Hour)), HasArtifact: true},
		{ID: 3, Status: extensions.WebReleaseFailed, CompletedAt: ptrTime(now.Add(-8 * 24 * time.Hour)), HasArtifact: true},
		{ID: 2, Status: extensions.WebReleaseSuperseded, CompletedAt: ptrTime(now.Add(-6 * 24 * time.Hour)), HasArtifact: true},
		{ID: 1, Status: extensions.WebReleaseFailed, CompletedAt: ptrTime(now.Add(-31 * 24 * time.Hour)), HasArtifact: false, HasBuildLog: true},
	}

	result := extensions.SelectWebReleaseCleanup(releases, now)
	if !reflect.DeepEqual(result.ArtifactReleaseIDs, []int64{3, 4}) {
		t.Fatalf("unexpected artifact cleanup ids: %v", result.ArtifactReleaseIDs)
	}
	if !reflect.DeepEqual(result.BuildLogReleaseIDs, []int64{1}) {
		t.Fatalf("unexpected build log cleanup ids: %v", result.BuildLogReleaseIDs)
	}
}

func TestWebReleaseCleanupWorkerDeletesOnlyCanonicalReleaseDirectories(t *testing.T) {
	root := t.TempDir()
	releaseDir := filepath.Join(root, "releases", "42")
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(releaseDir, "artifact.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := &fakeWebReleaseCleanupStore{result: extensions.WebReleaseCleanupResult{ArtifactReleaseIDs: []int64{42}}}
	worker := &WebReleaseCleanupWorker{Store: store, ReleaseRoot: root, Now: func() time.Time { return time.Unix(1, 0) }}

	if err := worker.Work(context.Background(), &river.Job[WebReleaseCleanupArgs]{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(releaseDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected release directory removed, got %v", err)
	}
	if !store.now.Equal(time.Unix(1, 0)) {
		t.Fatalf("worker did not pass its clock to the store: %s", store.now)
	}
}

func ptrTime(value time.Time) *time.Time { return &value }

type fakeWebReleaseCleanupStore struct {
	result extensions.WebReleaseCleanupResult
	now    time.Time
}

func (s *fakeWebReleaseCleanupStore) CleanupWebReleases(_ context.Context, now time.Time) (extensions.WebReleaseCleanupResult, error) {
	s.now = now
	return s.result, nil
}
