package webreleaseruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type artifactDigestEntry struct {
	relative string
	absolute string
	link     string
}

// ArtifactDigestTree 支持 Nitro 产物内部的相对 symlink，但拒绝任何根目录逃逸。
func ArtifactDigestTree(root string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
		return "", fmt.Errorf("artifact root must be a regular directory")
	}
	entries := make([]artifactDigestEntry, 0)
	err = filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == root || entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		entryInfo, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if entryInfo.Mode()&fs.ModeSymlink != 0 {
			target, err := os.Readlink(current)
			if err != nil {
				return err
			}
			resolved := filepath.Clean(filepath.Join(filepath.Dir(current), target))
			if filepath.IsAbs(target) || !pathWithin(root, resolved) {
				return fmt.Errorf("artifact symlink escapes artifact: %s", relative)
			}
			if _, err := os.Stat(resolved); err != nil {
				return fmt.Errorf("artifact symlink target is unavailable: %s", relative)
			}
			entries = append(entries, artifactDigestEntry{relative: relative, absolute: current, link: filepath.ToSlash(target)})
			return nil
		}
		if !entryInfo.Mode().IsRegular() {
			return fmt.Errorf("artifact contains non-regular file: %s", relative)
		}
		entries = append(entries, artifactDigestEntry{relative: relative, absolute: current})
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].relative < entries[j].relative })
	digest := sha256.New()
	for _, entry := range entries {
		_, _ = io.WriteString(digest, entry.relative)
		_, _ = digest.Write([]byte{0})
		if entry.link != "" {
			_, _ = io.WriteString(digest, "link")
			_, _ = digest.Write([]byte{0})
			_, _ = io.WriteString(digest, strconv.Itoa(len([]byte(entry.link))))
			_, _ = digest.Write([]byte{0})
			_, _ = io.WriteString(digest, entry.link)
			continue
		}
		info, err := os.Lstat(entry.absolute)
		if err != nil || !info.Mode().IsRegular() {
			return "", fmt.Errorf("artifact file changed while hashing: %s", entry.relative)
		}
		_, _ = io.WriteString(digest, strconv.FormatUint(uint64(info.Mode().Perm()), 8))
		_, _ = digest.Write([]byte{0})
		_, _ = io.WriteString(digest, strconv.FormatInt(info.Size(), 10))
		_, _ = digest.Write([]byte{0})
		file, err := os.Open(entry.absolute)
		if err != nil {
			return "", err
		}
		_, copyErr := io.Copy(digest, file)
		closeErr := file.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
	}
	return strings.ToLower(hex.EncodeToString(digest.Sum(nil))), nil
}
