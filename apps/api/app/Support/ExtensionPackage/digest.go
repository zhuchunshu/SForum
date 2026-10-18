package extensionpackage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var (
	ErrSymlink     = errors.New("extension package: symbolic links are not allowed")
	ErrNonRegular  = errors.New("extension package: non-regular files are not allowed")
	ErrInvalidPath = errors.New("extension package: invalid path")
)

type digestFile struct {
	path string
	full string
	// info 是 WalkDir 已经取得的条目元数据。指纹判定复用它，避免为同一批
	// 文件再 stat 一遍；hashFile 仍会自行 Lstat 以发现「哈希期间被改写」。
	info fs.FileInfo
}

// DigestTree returns a stable SHA-256 digest for the complete regular-file tree.
// Directory metadata and modification times intentionally do not participate.
func DigestTree(root string) (string, error) {
	return digestTree(root, false)
}

// NonArtifactDirectory 判断目录名是否属于打包阶段必然排除的开发/依赖目录。
// 这些目录从不进入发布 zip，因此也不构成扩展 artifact。
func NonArtifactDirectory(name string) bool {
	switch name {
	case ".git", "node_modules", "vendor":
		return true
	default:
		return false
	}
}

// DigestAuthorTree 是 DigestTree 的作者源码树变体：规则完全一致，只是跳过
// NonArtifactDirectory 目录。作者在本地跑过 bun/npm install 后，包内会出现
// node_modules（含 .bin 符号链接），而打包阶段本就会排除它；预检若把依赖目录
// 也算进 artifact，就会让「脚手架 → extension build」这条主循环直接失败。
//
// 信任身份仍由 DigestTree 计算：安装路径的树来自发布 zip，永远不含这些目录，
// 且 snapshot/archive 校验各自独立拒绝符号链接。
func DigestAuthorTree(root string) (string, error) {
	return digestTree(root, true)
}

func digestTree(root string, skipNonArtifactDirs bool) (string, error) {
	files, err := collectDigestFiles(root, skipNonArtifactDirs)
	if err != nil {
		return "", err
	}
	return hashDigestFiles(files)
}

// collectDigestFiles 收集受摘要约束的完整文件集合。它是 DigestTree 与
// DigestAuthorTree 唯一的遍历实现；指纹缓存必须复用同一集合，否则指纹命中
// 时的校验口径会与摘要不一致。
func collectDigestFiles(root string, skipNonArtifactDirs bool) ([]digestFile, error) {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return nil, err
	}
	if rootInfo.Mode()&fs.ModeSymlink != 0 {
		return nil, fmt.Errorf("%w: %s", ErrSymlink, root)
	}
	if !rootInfo.IsDir() {
		return nil, fmt.Errorf("%w: snapshot root %s", ErrNonRegular, root)
	}

	files := make([]digestFile, 0)
	seen := make(map[string]struct{})
	err = filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == root {
			return nil
		}
		if skipNonArtifactDirs && entry.IsDir() && NonArtifactDirectory(entry.Name()) {
			return filepath.SkipDir
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%w: %s", ErrSymlink, current)
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%w: %s", ErrNonRegular, current)
		}

		relative, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		normalized, ok := canonicalRelativePath(relative)
		if !ok {
			return fmt.Errorf("%w: %s", ErrInvalidPath, relative)
		}
		if _, duplicate := seen[normalized]; duplicate {
			return fmt.Errorf("%w: duplicate normalized path %s", ErrInvalidPath, normalized)
		}
		seen[normalized] = struct{}{}
		files = append(files, digestFile{path: normalized, full: current, info: info})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	return files, nil
}

// hashDigestFiles 计算文件集合的稳定摘要（顺序由 collectDigestFiles 固定）。
func hashDigestFiles(files []digestFile) (string, error) {
	digest := sha256.New()
	for _, file := range files {
		if err := hashFile(digest, file); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func hashFile(digest hash.Hash, file digestFile) error {
	linkInfo, err := os.Lstat(file.full)
	if err != nil {
		return err
	}
	if linkInfo.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("%w: %s", ErrSymlink, file.full)
	}
	if !linkInfo.Mode().IsRegular() {
		return fmt.Errorf("%w: %s", ErrNonRegular, file.full)
	}

	handle, err := os.Open(file.full)
	if err != nil {
		return err
	}
	info, statErr := handle.Stat()
	if statErr != nil {
		_ = handle.Close()
		return statErr
	}
	if !info.Mode().IsRegular() || !os.SameFile(linkInfo, info) {
		_ = handle.Close()
		return fmt.Errorf("%w: file changed while hashing %s", ErrNonRegular, file.full)
	}

	_, _ = io.WriteString(digest, file.path)
	_, _ = digest.Write([]byte{0})
	// Permission bits use octal text, matching the conventional representation of Unix modes.
	_, _ = io.WriteString(digest, strconv.FormatUint(uint64(info.Mode().Perm()), 8))
	_, _ = digest.Write([]byte{0})
	_, _ = io.WriteString(digest, strconv.FormatInt(info.Size(), 10))
	_, _ = digest.Write([]byte{0})
	written, copyErr := io.Copy(digest, handle)
	closeErr := handle.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written != info.Size() {
		return fmt.Errorf("extension package: file changed while hashing %s", file.full)
	}
	return nil
}

func canonicalRelativePath(value string) (string, bool) {
	value = strings.ReplaceAll(filepath.ToSlash(value), "\\", "/")
	if value == "" || strings.ContainsRune(value, '\x00') || strings.HasPrefix(value, "/") {
		return "", false
	}
	if len(value) >= 2 && value[1] == ':' && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) {
		return "", false
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}
