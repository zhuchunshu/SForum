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
}

// DigestTree returns a stable SHA-256 digest for the complete regular-file tree.
// Directory metadata and modification times intentionally do not participate.
func DigestTree(root string) (string, error) {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return "", err
	}
	if rootInfo.Mode()&fs.ModeSymlink != 0 {
		return "", fmt.Errorf("%w: %s", ErrSymlink, root)
	}
	if !rootInfo.IsDir() {
		return "", fmt.Errorf("%w: snapshot root %s", ErrNonRegular, root)
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
		files = append(files, digestFile{path: normalized, full: current})
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
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
