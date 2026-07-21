package pluginfiles

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteReadDeleteAndUsage(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.EnsureNamespace(Namespace{ExtensionID: "demo.files"}); err != nil {
		t.Fatal(err)
	}
	info, err := svc.Write(WriteRequest{
		ExtensionID: "demo.files", Kind: KindPrivate, RelativePath: "data/a.json",
		Data: []byte(`{"a":1}`), Actor: "plugin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if info.Size != 7 || info.RelativePath != "data/a.json" || info.ExtensionID != "demo.files" {
		t.Fatalf("info = %#v", info)
	}
	data, got, err := svc.Read(ReadRequest{
		ExtensionID: "demo.files", Kind: KindPrivate, RelativePath: "data/a.json",
	})
	if err != nil || string(data) != `{"a":1}` || got.Size != 7 {
		t.Fatalf("read = %q %#v err=%v", data, got, err)
	}
	usage, err := svc.Usage("demo.files")
	if err != nil || usage.PrivateUsed != 7 {
		t.Fatalf("usage = %#v err=%v", usage, err)
	}
	if err := svc.Delete(DeleteRequest{
		ExtensionID: "demo.files", Kind: KindPrivate, RelativePath: "data/a.json", Actor: "plugin",
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Read(ReadRequest{
		ExtensionID: "demo.files", Kind: KindPrivate, RelativePath: "data/a.json",
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete = %v", err)
	}
}

func TestTraversalAndSymlinkRejected(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.EnsureNamespace(Namespace{ExtensionID: "demo.files"}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../escape", "..", "/etc/passwd", "a/../../b", `a\..\b`} {
		if _, err := svc.Write(WriteRequest{
			ExtensionID: "demo.files", Kind: KindPrivate, RelativePath: path,
			Data: []byte("x"), Actor: "p",
		}); !errors.Is(err, ErrTraversal) && !errors.Is(err, ErrInvalid) {
			t.Fatalf("path %q = %v", path, err)
		}
	}

	// Plant a symlink pointing outside and ensure write/read reject.
	ns, _ := svc.namespace("demo.files")
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(ns.Root, KindPrivate, "link-out")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Write(WriteRequest{
		ExtensionID: "demo.files", Kind: KindPrivate, RelativePath: "link-out",
		Data: []byte("overwrite"), Actor: "p",
	}); !errors.Is(err, ErrSymlink) {
		t.Fatalf("symlink write = %v", err)
	}
	if _, _, err := svc.Read(ReadRequest{
		ExtensionID: "demo.files", Kind: KindPrivate, RelativePath: "link-out",
	}); !errors.Is(err, ErrSymlink) {
		t.Fatalf("symlink read = %v", err)
	}
}

func TestQuotaExceeded(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.EnsureNamespace(Namespace{
		ExtensionID: "demo.quota", PrivateQuotaBytes: 10,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Write(WriteRequest{
		ExtensionID: "demo.quota", Kind: KindPrivate, RelativePath: "a.bin",
		Data: []byte("12345"), Actor: "p",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Write(WriteRequest{
		ExtensionID: "demo.quota", Kind: KindPrivate, RelativePath: "b.bin",
		Data: []byte("123456"), Actor: "p",
	}); !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("quota = %v", err)
	}
}

func TestUserIsolationAndCleanup(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.EnsureNamespace(Namespace{ExtensionID: "demo.user"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Write(WriteRequest{
		ExtensionID: "demo.user", Kind: KindUser, UserID: "42", RelativePath: "avatar.bin",
		Data: []byte("img"), Actor: "p",
	}); err != nil {
		t.Fatal(err)
	}
	// Missing user id invalid.
	if _, err := svc.Write(WriteRequest{
		ExtensionID: "demo.user", Kind: KindUser, RelativePath: "x", Data: []byte("x"), Actor: "p",
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing user = %v", err)
	}
	// Other user cannot read.
	if _, _, err := svc.Read(ReadRequest{
		ExtensionID: "demo.user", Kind: KindUser, UserID: "99", RelativePath: "avatar.bin",
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross user = %v", err)
	}
	// 默认卸载策略：保留 user-owned；需显式 DeleteUser。
	kept, err := svc.CleanupNamespace("demo.user")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Read(ReadRequest{
		ExtensionID: "demo.user", Kind: KindUser, UserID: "42", RelativePath: "avatar.bin",
	}); err != nil {
		t.Fatalf("user data should be retained by default: %v (cleanup=%#v)", err, kept)
	}
	result, err := svc.CleanupNamespaceWithOptions("demo.user", CleanupOptions{DeleteUser: true})
	if err != nil || result.RemovedFiles < 1 {
		t.Fatalf("cleanup delete user = %#v err=%v", result, err)
	}
	if _, _, err := svc.Read(ReadRequest{
		ExtensionID: "demo.user", Kind: KindUser, UserID: "42", RelativePath: "avatar.bin",
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("user data after DeleteUser = %v", err)
	}
}

func TestCrossPluginIsolationAndRestart(t *testing.T) {
	dir := t.TempDir()
	svc, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.EnsureNamespace(Namespace{ExtensionID: "plugin.a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.EnsureNamespace(Namespace{ExtensionID: "plugin.b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Write(WriteRequest{
		ExtensionID: "plugin.a", Kind: KindPrivate, RelativePath: "secret.txt",
		Data: []byte("only-a"), Actor: "a",
	}); err != nil {
		t.Fatal(err)
	}
	// B 不能通过相对路径读到 A 的内容（不同 namespace 根）。
	if _, _, err := svc.Read(ReadRequest{
		ExtensionID: "plugin.b", Kind: KindPrivate, RelativePath: "secret.txt",
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross plugin = %v", err)
	}
	// 重启：新 Service 同一 baseDir 可恢复 private。
	svc2, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	data, _, err := svc2.Read(ReadRequest{
		ExtensionID: "plugin.a", Kind: KindPrivate, RelativePath: "secret.txt",
	})
	if err != nil || string(data) != "only-a" {
		t.Fatalf("restart read = %q err=%v", data, err)
	}
}

func TestTempCleanupByAge(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.EnsureNamespace(Namespace{ExtensionID: "demo.temp"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Write(WriteRequest{
		ExtensionID: "demo.temp", Kind: KindTemp, RelativePath: "old.tmp",
		Data: []byte("tmp"), Actor: "p",
	}); err != nil {
		t.Fatal(err)
	}
	ns, _ := svc.namespace("demo.temp")
	path := filepath.Join(ns.Root, KindTemp, "old.tmp")
	past := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(path, past, past); err != nil {
		t.Fatal(err)
	}
	removed, err := svc.CleanupTemp("demo.temp", time.Hour)
	if err != nil || removed != 1 {
		t.Fatalf("cleanup temp = %d err=%v", removed, err)
	}
}

func TestActorRequired(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.EnsureNamespace(Namespace{ExtensionID: "demo.x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Write(WriteRequest{
		ExtensionID: "demo.x", Kind: KindPrivate, RelativePath: "a", Data: []byte("x"),
	}); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("write actor = %v", err)
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	svc, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return svc
}
