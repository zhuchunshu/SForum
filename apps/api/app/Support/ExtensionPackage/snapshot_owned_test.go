package extensionpackage

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestSnapshotUploadedOwnedReportsCreationAndReuse(t *testing.T) {
	destination := t.TempDir()
	manifest := snapshotTestManifest("owned.plugin", "1.0.0")
	files := []File{{Path: "component.vue", Mode: 0o644, Body: []byte("owned")}}

	first, err := SnapshotUploadedOwned(destination, manifest, files)
	if err != nil {
		t.Fatalf("create owned snapshot: %v", err)
	}
	if !first.Created() {
		t.Fatal("first publisher did not receive creation ownership")
	}
	firstRoot := first.Root
	if err := first.Release(); err != nil {
		t.Fatalf("release first snapshot: %v", err)
	}

	second, err := SnapshotUploadedOwned(destination, manifest, files)
	if err != nil {
		t.Fatalf("reuse owned snapshot: %v", err)
	}
	defer second.Release()
	if second.Created() || second.Root != firstRoot {
		t.Fatalf("reused snapshot ownership = created:%t root:%s", second.Created(), second.Root)
	}
	if err := second.RemoveIfCreated(); err != nil {
		t.Fatalf("discard reused snapshot: %v", err)
	}
	if _, err := os.Stat(firstRoot); err != nil {
		t.Fatalf("reused snapshot was removed: %v", err)
	}
}

func TestSnapshotUploadedOwnedSerializesSameDigest(t *testing.T) {
	destination := t.TempDir()
	manifest := snapshotTestManifest("serialized.plugin", "1.0.0")
	files := []File{{Path: "component.vue", Mode: 0o644, Body: []byte("serialized")}}

	first, err := SnapshotUploadedOwned(destination, manifest, files)
	if err != nil {
		t.Fatalf("create first owned snapshot: %v", err)
	}
	t.Cleanup(func() { _ = first.Release() })
	if uploadedSnapshotMu.TryLock() {
		uploadedSnapshotMu.Unlock()
		t.Fatal("owned snapshot did not retain the process-local publication gate")
	}

	if err := first.Release(); err != nil {
		t.Fatalf("release first owned snapshot: %v", err)
	}
	second, err := SnapshotUploadedOwned(destination, manifest, files)
	if err != nil {
		t.Fatalf("second publisher failed after release: %v", err)
	}
	defer second.Release()
	if second.Created() {
		t.Fatal("serialized identical publisher unexpectedly acquired creation ownership")
	}
}

func TestSnapshotUploadedOwnedSerializesAcrossProcesses(t *testing.T) {
	const helperEnv = "SFORUM_OWNED_SNAPSHOT_PROCESS_HELPER"
	if os.Getenv(helperEnv) == "1" {
		destination := os.Getenv("SFORUM_OWNED_SNAPSHOT_ROOT")
		ready := os.Getenv("SFORUM_OWNED_SNAPSHOT_READY")
		release := os.Getenv("SFORUM_OWNED_SNAPSHOT_RELEASE")
		owned, err := SnapshotUploadedOwned(
			destination,
			snapshotTestManifest("process-lock.plugin", "1.0.0"),
			[]File{{Path: "component.vue", Mode: 0o644, Body: []byte("process-lock")}},
		)
		if err != nil {
			t.Fatalf("helper acquire owned snapshot: %v", err)
		}
		defer owned.Release()
		if !owned.Created() {
			t.Fatal("helper must create the first process snapshot")
		}
		if err := os.WriteFile(ready, []byte("ready"), 0o600); err != nil {
			t.Fatalf("write helper ready signal: %v", err)
		}
		if err := waitForSnapshotSignal(context.Background(), release, 10*time.Second); err != nil {
			t.Fatalf("wait for helper release signal: %v", err)
		}
		return
	}

	destination := t.TempDir()
	ready := filepath.Join(destination, "helper.ready")
	release := filepath.Join(destination, "helper.release")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSnapshotUploadedOwnedSerializesAcrossProcesses$")
	command.Env = append(os.Environ(),
		helperEnv+"=1",
		"SFORUM_OWNED_SNAPSHOT_ROOT="+destination,
		"SFORUM_OWNED_SNAPSHOT_READY="+ready,
		"SFORUM_OWNED_SNAPSHOT_RELEASE="+release,
	)
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Start(); err != nil {
		t.Fatalf("start snapshot lock helper: %v", err)
	}
	if err := waitForSnapshotSignal(ctx, ready, 5*time.Second); err != nil {
		cancel()
		_ = command.Wait()
		t.Fatalf("wait for helper readiness: %v\n%s", err, output.String())
	}

	lockFiles, err := filepath.Glob(filepath.Join(destination, "process-lock.plugin", "1.0.0", ".locks", "*.lock"))
	if err != nil || len(lockFiles) != 1 {
		cancel()
		_ = command.Wait()
		t.Fatalf("locate helper lock file: files=%#v err=%v", lockFiles, err)
	}
	probe, err := os.OpenFile(lockFiles[0], os.O_RDWR, 0o600)
	if err != nil {
		cancel()
		_ = command.Wait()
		t.Fatalf("open helper lock probe: %v", err)
	}
	acquired, probeErr := tryLockUploadedSnapshotFile(probe)
	if acquired {
		_ = unlockUploadedSnapshotFile(probe)
	}
	if closeErr := probe.Close(); probeErr == nil {
		probeErr = closeErr
	}
	if probeErr != nil {
		cancel()
		_ = command.Wait()
		t.Fatalf("probe helper advisory lock: %v", probeErr)
	}
	if acquired {
		cancel()
		_ = command.Wait()
		t.Fatal("second process acquired a live digest advisory lock")
	}

	if err := os.WriteFile(release, []byte("release"), 0o600); err != nil {
		t.Fatalf("write helper release signal: %v", err)
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("snapshot lock helper failed: %v\n%s", err, output.String())
	}
	owned, err := SnapshotUploadedOwned(
		destination,
		snapshotTestManifest("process-lock.plugin", "1.0.0"),
		[]File{{Path: "component.vue", Mode: 0o644, Body: []byte("process-lock")}},
	)
	if err != nil {
		t.Fatalf("parent acquire failed after helper release: %v", err)
	}
	defer owned.Release()
	if owned.Created() {
		t.Fatal("parent should reuse the helper's published snapshot")
	}
}

func waitForSnapshotSignal(ctx context.Context, path string, timeout time.Duration) error {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(path); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return errors.New("snapshot signal timeout")
		case <-ticker.C:
		}
	}
}
