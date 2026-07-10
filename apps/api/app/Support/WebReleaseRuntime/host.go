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
	"strings"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

const (
	AdminSDKVersion      = 1
	BuildContractVersion = 1
	BunVersion           = "1.3.14"
)

func HostPeers() extensionpackage.HostPeers {
	return extensionpackage.HostPeers{
		"vue":               "3.5.39",
		"nuxt":              "4.4.8",
		"@nuxt/ui":          "4.9.0",
		"vue-router":        "5.1.0",
		"@sforum/admin-sdk": "1.0.0",
	}
}

func CompositionHost(webRoot string) (extensions.WebCompositionHost, error) {
	root := resolveWebRoot(webRoot)
	lock, err := os.ReadFile(filepath.Join(root, "bun.lock"))
	if err != nil {
		return extensions.WebCompositionHost{}, fmt.Errorf("read web lockfile: %w", err)
	}
	lockDigest := sha256.Sum256(lock)
	sourceDigest, err := digestWebSource(root)
	if err != nil {
		return extensions.WebCompositionHost{}, err
	}
	return extensions.WebCompositionHost{
		WebSource: sourceDigest, WebLock: hex.EncodeToString(lockDigest[:]),
		SDKVersion: AdminSDKVersion, BunVersion: BunVersion, Contract: BuildContractVersion,
		HostPeers: HostPeers(),
	}, nil
}

func digestWebSource(root string) (string, error) {
	type sourceFile struct{ relative, absolute string }
	files := make([]sourceFile, 0)
	err := filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if hostSourceExcluded(filepath.ToSlash(relative), entry.IsDir()) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("web source contains symbolic link: %s", relative)
		}
		if info.Mode().IsRegular() {
			files = append(files, sourceFile{filepath.ToSlash(relative), current})
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].relative < files[j].relative })
	digest := sha256.New()
	for _, file := range files {
		handle, err := os.Open(file.absolute)
		if err != nil {
			return "", err
		}
		_, _ = io.WriteString(digest, file.relative)
		_, _ = io.WriteString(digest, "\x00")
		if _, err := io.Copy(digest, handle); err != nil {
			_ = handle.Close()
			return "", err
		}
		if err := handle.Close(); err != nil {
			return "", err
		}
	}
	return strings.ToLower(hex.EncodeToString(digest.Sum(nil))), nil
}
