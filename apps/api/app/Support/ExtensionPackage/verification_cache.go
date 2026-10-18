package extensionpackage

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 扩展包的整树摘要与逐文件摘要都会被多条链路反复校验：后台 GuardPolicyCatalog
// 每 5 秒遍历一次全部扩展，每次保存设置又要跑 3–5 遍生命周期校验与可信身份计算。
// 每次校验都要把整棵包目录（含数十 MB 的已编译后端二进制）读出来做 SHA-256，
// 于是同一份不可变制品被重复哈希成千上万次。
//
// 这里按「树指纹」记忆化校验结果：指纹只读元数据（路径集合 + 大小 + mtime +
// 权限位），不读内容，因此比整树哈希便宜约三个数量级；内容一旦变化，大小或
// mtime 必然改变，指纹即告失效并触发完整重算。
//
// 取舍（见 knowledge/decisions/2026-10-19-package-verification-memoization.md）：
// 指纹不包含内容，所以「同时伪造文件大小与 mtime」的篡改在 TTL 窗口内不会被
// 发现。TTL 与显式失效点共同把该窗口限制在有界范围内。
const (
	// VerificationCacheTTL 是缓存条目的最长信任时长。到期后即使指纹未变也会
	// 完整重算一遍，作为指纹被绕过的兜底。
	VerificationCacheTTL = 5 * time.Minute
	// verificationCacheLimit 限制条目数量；超限时清空过期项，仍超限则整体清空。
	verificationCacheLimit = 1024
)

type verificationCacheEntry struct {
	fingerprint string
	digest      string
	verifiedAt  time.Time
}

var verificationCache = struct {
	mu      sync.Mutex
	entries map[string]verificationCacheEntry
}{entries: make(map[string]verificationCacheEntry)}

// VerifiedTreeDigest 与 DigestTree 语义一致（同一套遍历规则、同一份文件集合、
// 同一个摘要算法），区别仅在于指纹未变时复用上一次的校验结果。
func VerifiedTreeDigest(root string) (string, error) {
	files, err := collectDigestFiles(root, false)
	if err != nil {
		return "", err
	}
	return verifiedDigestFiles(root, fingerprintDigestFiles(files), files)
}

// VerifiedFileDigest 返回单个常规文件的 SHA-256。指纹未变时复用上次结果，
// 供可信身份校验避免重复读取数十 MB 的可执行文件。
func VerifiedFileDigest(fullPath string) (string, error) {
	return VerifiedFileDigestFunc(fullPath, func() (string, error) {
		handle, err := os.Open(fullPath)
		if err != nil {
			return "", err
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, handle)
		closeErr := handle.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		return hex.EncodeToString(hash.Sum(nil)), nil
	})
}

// VerifiedFileDigestFunc 与 VerifiedFileDigest 相同，但把真实校验交给 compute。
// 可信路径需要自己的 TOCTOU 防御（打开句柄前后比对同一文件），缓存只负责决定
// 「这一次是否需要再跑一遍」；compute 返回错误时结果不写入缓存。
func VerifiedFileDigestFunc(fullPath string, compute func() (string, error)) (string, error) {
	if compute == nil {
		return "", ErrNonRegular
	}
	info, err := os.Lstat(fullPath)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", ErrSymlink
	}
	if !info.Mode().IsRegular() {
		return "", ErrNonRegular
	}
	fingerprint := fingerprintDigest(entryFingerprint(fullPath, info))
	if digest, ok := lookupVerification(fullPath, fingerprint); ok {
		return digest, nil
	}
	digest, err := compute()
	if err != nil {
		return "", err
	}
	storeVerification(fullPath, fingerprint, digest)
	return digest, nil
}

// InvalidateVerification 丢弃 path 自身及其子树的记忆化结果。构建立即写回、
// 暂存、升级、卸载等「已知内容发生变化」的路径必须显式调用它，不要依赖指纹。
func InvalidateVerification(path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	prefix := filepath.Clean(path)
	verificationCache.mu.Lock()
	defer verificationCache.mu.Unlock()
	for key := range verificationCache.entries {
		if key == prefix || strings.HasPrefix(key, prefix+string(filepath.Separator)) {
			delete(verificationCache.entries, key)
		}
	}
}

func verifiedDigestFiles(root, fingerprint string, files []digestFile) (string, error) {
	if digest, ok := lookupVerification(root, fingerprint); ok {
		return digest, nil
	}
	digest, err := hashDigestFiles(files)
	if err != nil {
		return "", err
	}
	storeVerification(root, fingerprint, digest)
	return digest, nil
}

// fingerprintDigestFiles 把文件集合压缩成一个指纹。集合顺序由
// collectDigestFiles 固定，因此指纹对同一棵树是稳定的。
func fingerprintDigestFiles(files []digestFile) string {
	var builder strings.Builder
	for _, file := range files {
		builder.WriteString(entryFingerprint(file.path, file.info))
	}
	return fingerprintDigest(builder.String())
}

// entryFingerprint 只用元数据描述一个文件。inode 不在其中：该字段需要平台
// 相关的 syscall，而路径集合本身已经由遍历固定，收益不足以引入构建约束。
func entryFingerprint(name string, info os.FileInfo) string {
	if info == nil {
		return name + "\x00missing\x00"
	}
	return name + "\x00" +
		strconv.FormatInt(info.Size(), 10) + "\x00" +
		strconv.FormatInt(info.ModTime().UnixNano(), 10) + "\x00" +
		strconv.FormatUint(uint64(info.Mode().Perm()), 8) + "\x00"
}

func fingerprintDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func lookupVerification(key, fingerprint string) (string, bool) {
	now := time.Now()
	verificationCache.mu.Lock()
	defer verificationCache.mu.Unlock()
	entry, ok := verificationCache.entries[key]
	if !ok || entry.fingerprint != fingerprint {
		return "", false
	}
	if now.Sub(entry.verifiedAt) >= VerificationCacheTTL {
		return "", false
	}
	return entry.digest, true
}

func storeVerification(key, fingerprint, digest string) {
	now := time.Now()
	verificationCache.mu.Lock()
	defer verificationCache.mu.Unlock()
	if len(verificationCache.entries) >= verificationCacheLimit {
		pruneVerificationCacheLocked(now)
	}
	verificationCache.entries[key] = verificationCacheEntry{
		fingerprint: fingerprint, digest: digest, verifiedAt: now,
	}
}

func pruneVerificationCacheLocked(now time.Time) {
	for key, entry := range verificationCache.entries {
		if now.Sub(entry.verifiedAt) >= VerificationCacheTTL {
			delete(verificationCache.entries, key)
		}
	}
	if len(verificationCache.entries) < verificationCacheLimit {
		return
	}
	// 仍超限说明 TTL 内出现了大量互异路径（例如短命的临时目录）。整体清空比
	// 无界增长安全：丢失的只是复用机会，校验语义不变。
	verificationCache.entries = make(map[string]verificationCacheEntry)
}
