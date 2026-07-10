package webreleaseruntime

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func TestPointerStoreWritesAndReadsAcknowledgements(t *testing.T) {
	root := t.TempDir()
	store := NewPointerStore(root, nil)
	current := CurrentRelease{ReleaseID: 12, CompositionHash: "composition", ArtifactDigest: "artifact"}
	if err := store.WriteCurrent(context.Background(), current); err != nil {
		t.Fatal(err)
	}
	var written CurrentRelease
	if err := readJSON(filepath.Join(root, "current.json"), &written); err != nil {
		t.Fatal(err)
	}
	if written.SchemaVersion != ReleaseManifestSchemaVersion || written.RequestedAt.IsZero() {
		t.Fatalf("current pointer defaults were not written: %#v", written)
	}

	writePointerFixture(t, filepath.Join(root, "active.json"), ActiveRelease{ReleaseID: 12, CompositionHash: "composition"})
	active, err := store.ReadActive(context.Background())
	if err != nil || active.ReleaseID != 12 {
		t.Fatalf("read active acknowledgement: active=%#v err=%v", active, err)
	}
	writePointerFixture(t, filepath.Join(root, "failures", "12.json"), Failure{ReleaseID: 12, Reason: "failed"})
	failure, err := store.ReadFailure(context.Background(), 12)
	if err != nil || failure.Reason != "failed" {
		t.Fatalf("read failure acknowledgement: failure=%#v err=%v", failure, err)
	}
}

func TestPointerStoreRejectsMismatchedFailureAndRestoresPrevious(t *testing.T) {
	root := t.TempDir()
	previousID := int64(5)
	reader := pointerReleaseReader{detail: extensions.WebReleaseDetail{WebRelease: extensions.WebRelease{
		ID: previousID, CompositionHash: "previous", ArtifactDigest: "old-artifact",
		ArtifactPath: "/releases/5", ServerEntry: "/releases/5/server/index.mjs",
	}}}
	store := NewPointerStore(root, reader)
	writePointerFixture(t, filepath.Join(root, "failures", "7.json"), Failure{ReleaseID: 8})
	if _, err := store.ReadFailure(context.Background(), 7); err == nil {
		t.Fatal("expected mismatched failure acknowledgement to be rejected")
	}
	if err := store.RestorePrevious(context.Background(), extensions.WebReleaseDetail{WebRelease: extensions.WebRelease{PreviousReleaseID: &previousID}}); err != nil {
		t.Fatal(err)
	}
	var current CurrentRelease
	if err := readJSON(filepath.Join(root, "current.json"), &current); err != nil {
		t.Fatal(err)
	}
	if current.ReleaseID != previousID || current.CompositionHash != "previous" {
		t.Fatalf("unexpected restored pointer: %#v", current)
	}
	if err := store.RestorePrevious(context.Background(), extensions.WebReleaseDetail{}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadActive(context.Background()); !errors.Is(err, ErrPointerNotFound) {
		t.Fatalf("expected missing active pointer error, got %v", err)
	}
	if err := readJSON(filepath.Join(root, "current.json"), &current); !errors.Is(err, ErrPointerNotFound) {
		t.Fatalf("expected current pointer removal, got %v", err)
	}
}

type pointerReleaseReader struct{ detail extensions.WebReleaseDetail }

func (r pointerReleaseReader) WebRelease(context.Context, int64) (extensions.WebReleaseDetail, error) {
	return r.detail, nil
}

func writePointerFixture(t *testing.T, path string, value any) {
	t.Helper()
	if err := writeJSONAtomic(path, value); err != nil {
		t.Fatal(err)
	}
}
