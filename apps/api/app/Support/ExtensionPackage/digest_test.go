package extensionpackage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDigestTreeIgnoresCreationOrderDirectoryModesAndModificationTimes(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()

	writeDigestTestFile(t, first, "z-last.txt", "last", 0o640)
	writeDigestTestFile(t, first, "nested/a-first.txt", "first", 0o644)
	writeDigestTestFile(t, second, "nested/a-first.txt", "first", 0o644)
	writeDigestTestFile(t, second, "z-last.txt", "last", 0o640)

	firstDigest, err := DigestTree(first)
	if err != nil {
		t.Fatalf("digest first tree: %v", err)
	}
	secondDigest, err := DigestTree(second)
	if err != nil {
		t.Fatalf("digest second tree: %v", err)
	}
	if firstDigest != secondDigest {
		t.Fatalf("creation order changed digest: first=%s second=%s", firstDigest, secondDigest)
	}

	changedAt := time.Unix(1_900_000_000, 0)
	if err := os.Chtimes(filepath.Join(first, "z-last.txt"), changedAt, changedAt); err != nil {
		t.Fatalf("change file timestamps: %v", err)
	}
	if err := os.Chmod(filepath.Join(first, "nested"), 0o700); err != nil {
		t.Fatalf("change directory mode: %v", err)
	}
	afterMetadataChange, err := DigestTree(first)
	if err != nil {
		t.Fatalf("digest tree after metadata change: %v", err)
	}
	if afterMetadataChange != firstDigest {
		t.Fatalf("directory metadata or mtime changed digest: before=%s after=%s", firstDigest, afterMetadataChange)
	}
}

func TestDigestTreeChangesWhenFileBytesChange(t *testing.T) {
	root := t.TempDir()
	file := writeDigestTestFile(t, root, "component.vue", "alpha", 0o644)

	before, err := DigestTree(root)
	if err != nil {
		t.Fatalf("digest original tree: %v", err)
	}
	if err := os.WriteFile(file, []byte("bravo"), 0o644); err != nil {
		t.Fatalf("replace file body: %v", err)
	}
	after, err := DigestTree(root)
	if err != nil {
		t.Fatalf("digest changed tree: %v", err)
	}
	if before == after {
		t.Fatalf("file byte change did not change digest: %s", before)
	}
}

func TestDigestTreeChangesWhenPermissionBitsChange(t *testing.T) {
	root := t.TempDir()
	file := writeDigestTestFile(t, root, "component.vue", "same", 0o644)

	before, err := DigestTree(root)
	if err != nil {
		t.Fatalf("digest original mode: %v", err)
	}
	if err := os.Chmod(file, 0o600); err != nil {
		t.Fatalf("change file mode: %v", err)
	}
	after, err := DigestTree(root)
	if err != nil {
		t.Fatalf("digest changed mode: %v", err)
	}
	if before == after {
		t.Fatalf("permission change did not change digest: %s", before)
	}
}

func TestDigestTreeRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	writeDigestTestFile(t, root, "component.vue", "<template />", 0o644)
	if err := os.Symlink("component.vue", filepath.Join(root, "alias.vue")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	_, err := DigestTree(root)
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("expected ErrSymlink, got %v", err)
	}
}

func writeDigestTestFile(t *testing.T, root string, relativePath string, body string, mode os.FileMode) string {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("create parent for %s: %v", relativePath, err)
	}
	if err := os.WriteFile(target, []byte(body), mode); err != nil {
		t.Fatalf("write %s: %v", relativePath, err)
	}
	return target
}
