package extensionpackage

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// 记忆化只允许跳过「重复校验」，绝不允许改变校验结果：下面这些用例同时锁定
// 复用行为和正确性（内容变化必须重算、TTL 到期必须重算、symlink 仍被拒绝）。
func resetVerificationCacheForTest() {
	verificationCache.mu.Lock()
	verificationCache.entries = make(map[string]verificationCacheEntry)
	verificationCache.mu.Unlock()
}

func writeTestPackageFile(t *testing.T, root, name string, body []byte) string {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, body, 0o644); err != nil {
		t.Fatal(err)
	}
	return full
}

func TestVerifiedTreeDigestMatchesDigestTree(t *testing.T) {
	resetVerificationCacheForTest()
	root := t.TempDir()
	writeTestPackageFile(t, root, "sforum.extension.json", []byte(`{"manifestVersion":3}`))
	writeTestPackageFile(t, root, "backend/plugin", []byte("binary-bytes"))

	want, err := DigestTree(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := VerifiedTreeDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("verified digest %s != DigestTree %s", got, want)
	}
}

// 复用必须发生：把一个哨兵值写进缓存，命中时应当原样返回。
func TestVerifiedTreeDigestReusesCacheWithinTTL(t *testing.T) {
	resetVerificationCacheForTest()
	root := t.TempDir()
	writeTestPackageFile(t, root, "a.txt", []byte("one"))

	if _, err := VerifiedTreeDigest(root); err != nil {
		t.Fatal(err)
	}
	verificationCache.mu.Lock()
	entry := verificationCache.entries[root]
	entry.digest = "sentinel-digest"
	verificationCache.entries[root] = entry
	verificationCache.mu.Unlock()

	got, err := VerifiedTreeDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != "sentinel-digest" {
		t.Fatalf("fingerprint hit did not reuse the cached verification, got %s", got)
	}
}

// 内容变化（即便字节数相同）必须击穿缓存并重新校验。
func TestVerifiedTreeDigestRecomputesWhenContentChanges(t *testing.T) {
	resetVerificationCacheForTest()
	root := t.TempDir()
	path := writeTestPackageFile(t, root, "a.txt", []byte("AAAA"))

	first, err := VerifiedTreeDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	// 同长度、不同内容的改写：mtime 必然变化，指纹必须失效。
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(path, []byte("BBBB"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := VerifiedTreeDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("content change was masked by the verification cache")
	}
	want, err := DigestTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if second != want {
		t.Fatalf("recomputed digest %s != DigestTree %s", second, want)
	}
}

// TTL 到期后即便指纹未变也必须重算，作为指纹被绕过的兜底。
func TestVerifiedTreeDigestRecomputesAfterTTL(t *testing.T) {
	resetVerificationCacheForTest()
	root := t.TempDir()
	writeTestPackageFile(t, root, "a.txt", []byte("one"))

	want, err := DigestTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifiedTreeDigest(root); err != nil {
		t.Fatal(err)
	}
	verificationCache.mu.Lock()
	entry := verificationCache.entries[root]
	entry.verifiedAt = time.Now().Add(-2 * VerificationCacheTTL)
	verificationCache.entries[root] = entry
	verificationCache.mu.Unlock()

	got, err := VerifiedTreeDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("expired entry was reused: got %s want %s", got, want)
	}
}

func TestInvalidateVerificationDropsSelfAndSubtree(t *testing.T) {
	resetVerificationCacheForTest()
	root := t.TempDir()
	child := filepath.Join(root, "backend")
	writeTestPackageFile(t, root, "backend/plugin", []byte("bin"))

	unrelated := filepath.Join(t.TempDir(), "unrelated")
	for _, path := range []string{root, child, filepath.Join(root, "other"), unrelated} {
		verificationCache.mu.Lock()
		verificationCache.entries[filepath.Clean(path)] = verificationCacheEntry{
			fingerprint: "fp", digest: "d", verifiedAt: time.Now(),
		}
		verificationCache.mu.Unlock()
	}
	InvalidateVerification(root)

	verificationCache.mu.Lock()
	defer verificationCache.mu.Unlock()
	if _, ok := verificationCache.entries[root]; ok {
		t.Fatal("invalidated root entry survived")
	}
	if _, ok := verificationCache.entries[child]; ok {
		t.Fatal("invalidated subtree entry survived")
	}
	if _, ok := verificationCache.entries[filepath.Join(root, "other")]; ok {
		t.Fatal("invalidated subtree sibling entry survived")
	}
	if _, ok := verificationCache.entries[unrelated]; !ok {
		t.Fatal("invalidation removed an unrelated entry")
	}
}

// 可信路径自带 TOCTOU 防御：compute 必须仍在未命中时执行，且其错误不得被缓存。
func TestVerifiedFileDigestFuncKeepsCallerVerification(t *testing.T) {
	resetVerificationCacheForTest()
	root := t.TempDir()
	path := writeTestPackageFile(t, root, "plugin", []byte("binary"))

	calls := 0
	compute := func() (string, error) {
		calls++
		return "caller-digest", nil
	}
	first, err := VerifiedFileDigestFunc(path, compute)
	if err != nil {
		t.Fatal(err)
	}
	if first != "caller-digest" || calls != 1 {
		t.Fatalf("first call must run the caller verification: digest=%s calls=%d", first, calls)
	}
	second, err := VerifiedFileDigestFunc(path, compute)
	if err != nil {
		t.Fatal(err)
	}
	if second != "caller-digest" || calls != 1 {
		t.Fatalf("second call must reuse the cached result: digest=%s calls=%d", second, calls)
	}

	failing := func() (string, error) { return "", ErrNonRegular }
	if _, err := VerifiedFileDigestFunc(path, failing); err != nil {
		t.Fatalf("cached path should not consult a failing compute: %v", err)
	}
	verificationCache.mu.Lock()
	delete(verificationCache.entries, path)
	verificationCache.mu.Unlock()
	if _, err := VerifiedFileDigestFunc(path, failing); err == nil {
		t.Fatal("compute error must surface on a cache miss")
	}
	verificationCache.mu.Lock()
	_, cached := verificationCache.entries[path]
	verificationCache.mu.Unlock()
	if cached {
		t.Fatal("failed verification must not be cached")
	}
}

func TestVerifiedTreeDigestRejectsSymlinkRoot(t *testing.T) {
	resetVerificationCacheForTest()
	root := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(root, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if _, err := VerifiedTreeDigest(link); err == nil {
		t.Fatal("symlink root must fail exactly like DigestTree")
	}
}

func TestVerifiedTreeDigestConcurrentAccess(t *testing.T) {
	resetVerificationCacheForTest()
	root := t.TempDir()
	writeTestPackageFile(t, root, "a.txt", []byte("one"))
	want, err := DigestTree(root)
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	for i := 0; i < 16; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			got, err := VerifiedTreeDigest(root)
			if err != nil || got != want {
				t.Errorf("concurrent digest mismatch: %s %v", got, err)
			}
		}()
	}
	wait.Wait()
}
