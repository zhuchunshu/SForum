package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
	themecompiler "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeCompiler"
)

const (
	ExternalDiagnosticRootUnavailable = "external.root_unavailable"
	ExternalDiagnosticPackageInvalid  = "external.package_invalid"
	ExternalDiagnosticIDConflict      = "external.id_conflict"
	ExternalDiagnosticBuiltinConflict = "external.builtin_conflict"
	ExternalDiagnosticSnapshotFailed  = "external.snapshot_failed"
)

// ExternalSourceDiagnostic 描述单个外部根或包的静态扫描失败。
// 诊断不会让 Host 执行不可信代码，也不会删除此前已安装的不可变制品。
type ExternalSourceDiagnostic struct {
	Code        string
	Root        string
	PackagePath string
	ExtensionID string
	Message     string
}

type ExternalSyncResult struct {
	Items       []Extension
	Diagnostics []ExternalSourceDiagnostic
}

type externalSourceCandidate struct {
	root          string
	packagePath   string
	extensionType string
	manifest      Manifest
}

// SyncExternalSources 扫描显式配置的第三方源码集合。
// 新包以 uploaded/installed 保存；已有活动包的不同摘要只进入 staged。
func (s *Service) SyncExternalSources(ctx context.Context) (ExternalSyncResult, error) {
	var result ExternalSyncResult
	if s == nil || len(s.externalRoots) == 0 {
		return result, nil
	}
	if s.store == nil {
		return result, fmt.Errorf("sync external extensions: store is required")
	}

	candidates := make([]externalSourceCandidate, 0)
	byID := make(map[string][]int)
	groups := []struct {
		dir           string
		extensionType string
	}{
		{dir: "plugins", extensionType: TypePlugin},
		{dir: "themes", extensionType: TypeTheme},
	}

	for _, root := range s.externalRoots {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			message := "not a directory"
			if err != nil {
				message = err.Error()
			}
			result.Diagnostics = append(result.Diagnostics, ExternalSourceDiagnostic{
				Code: ExternalDiagnosticRootUnavailable, Root: root, Message: message,
			})
			continue
		}
		for _, group := range groups {
			groupRoot := filepath.Join(root, group.dir)
			entries, err := os.ReadDir(groupRoot)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				result.Diagnostics = append(result.Diagnostics, ExternalSourceDiagnostic{
					Code: ExternalDiagnosticRootUnavailable, Root: root,
					PackagePath: groupRoot, Message: err.Error(),
				})
				continue
			}
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				packagePath := filepath.Join(groupRoot, entry.Name())
				manifest, err := extensionmanifest.LoadPackage(packagePath)
				if err != nil || validateManifest(manifest) != nil || manifest.Type != group.extensionType || manifest.ID == DefaultThemeID {
					if err == nil {
						err = ErrInvalidManifest
					}
					result.Diagnostics = append(result.Diagnostics, ExternalSourceDiagnostic{
						Code: ExternalDiagnosticPackageInvalid, Root: root,
						PackagePath: packagePath, ExtensionID: manifest.ID, Message: err.Error(),
					})
					continue
				}
				index := len(candidates)
				candidates = append(candidates, externalSourceCandidate{
					root: root, packagePath: packagePath,
					extensionType: group.extensionType, manifest: manifest,
				})
				byID[manifest.ID] = append(byID[manifest.ID], index)
			}
		}
	}

	conflicted := make(map[int]bool)
	for extensionID, indexes := range byID {
		if len(indexes) < 2 {
			continue
		}
		for _, index := range indexes {
			conflicted[index] = true
			candidate := candidates[index]
			result.Diagnostics = append(result.Diagnostics, ExternalSourceDiagnostic{
				Code: ExternalDiagnosticIDConflict, Root: candidate.root,
				PackagePath: candidate.packagePath, ExtensionID: extensionID,
				Message: "extension id is declared by multiple external source packages",
			})
		}
	}

	for index, candidate := range candidates {
		if conflicted[index] {
			continue
		}
		existing, err := s.store.Get(ctx, candidate.manifest.ID)
		existingFound := err == nil
		switch {
		case existingFound && (existing.Source == SourceBuiltin || existing.IsSystem):
			result.Diagnostics = append(result.Diagnostics, ExternalSourceDiagnostic{
				Code: ExternalDiagnosticBuiltinConflict, Root: candidate.root,
				PackagePath: candidate.packagePath, ExtensionID: candidate.manifest.ID,
				Message: "external package cannot replace a built-in extension",
			})
			continue
		case err != nil && !errors.Is(err, ErrExtensionNotFound):
			return result, fmt.Errorf("load external extension %s: %w", candidate.manifest.ID, err)
		}

		owned, err := extensionpackage.SnapshotExternalSourceOwned(candidate.packagePath, s.extensionRoot)
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, ExternalSourceDiagnostic{
				Code: ExternalDiagnosticSnapshotFailed, Root: candidate.root,
				PackagePath: candidate.packagePath, ExtensionID: candidate.manifest.ID, Message: err.Error(),
			})
			continue
		}
		snapshot := owned.Snapshot
		var manifest Manifest
		if err := json.Unmarshal([]byte(snapshot.Manifest), &manifest); err != nil ||
			manifest.ID != candidate.manifest.ID || manifest.Type != candidate.extensionType {
			if err == nil {
				err = errors.New("package identity changed while scanning")
			}
			result.Diagnostics = append(result.Diagnostics, ExternalSourceDiagnostic{
				Code: ExternalDiagnosticPackageInvalid, Root: candidate.root,
				PackagePath: candidate.packagePath, ExtensionID: candidate.manifest.ID, Message: err.Error(),
			})
			_ = s.discardUnreferencedUploadedSnapshot(ctx, owned)
			_ = owned.Release()
			continue
		}
		if manifest.Type == TypeTheme {
			if err := themecompiler.NewCompiler(themecompiler.Limits{}).PreflightFS(os.DirFS(snapshot.Root)); err != nil {
				result.Diagnostics = append(result.Diagnostics, ExternalSourceDiagnostic{
					Code: ExternalDiagnosticPackageInvalid, Root: candidate.root,
					PackagePath: candidate.packagePath, ExtensionID: manifest.ID, Message: err.Error(),
				})
				_ = s.discardUnreferencedUploadedSnapshot(ctx, owned)
				_ = owned.Release()
				continue
			}
		}
		adminFrontendDigest, err := ComputeAdminFrontendDigest(manifest, snapshot.Root)
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, ExternalSourceDiagnostic{
				Code: ExternalDiagnosticPackageInvalid, Root: candidate.root,
				PackagePath: candidate.packagePath, ExtensionID: manifest.ID, Message: err.Error(),
			})
			_ = s.discardUnreferencedUploadedSnapshot(ctx, owned)
			_ = owned.Release()
			continue
		}

		unchanged := existingFound && externalArtifactAlreadyKnown(existing, manifest.Version, snapshot.Digest)
		if unchanged {
			result.Items = append(result.Items, existing)
			_ = owned.Release()
			continue
		}
		installed, saveErr := s.store.SaveInstalled(ctx, SaveInstalledInput{
			Manifest: manifest, PackagePath: snapshot.Root, PackageDigest: snapshot.Digest,
			AdminFrontendDigest: adminFrontendDigest,
		})
		if saveErr != nil {
			_ = s.discardUnreferencedUploadedSnapshot(ctx, owned)
			_ = owned.Release()
			if errors.Is(saveErr, ErrNotDeletable) || errors.Is(saveErr, ErrInvalidManifest) {
				result.Diagnostics = append(result.Diagnostics, ExternalSourceDiagnostic{
					Code: ExternalDiagnosticBuiltinConflict, Root: candidate.root,
					PackagePath: candidate.packagePath, ExtensionID: manifest.ID, Message: saveErr.Error(),
				})
				continue
			}
			return result, fmt.Errorf("save external extension %s: %w", manifest.ID, saveErr)
		}
		_ = owned.Release()
		message := "External source package discovered and installed as an inert snapshot."
		action := EventInstalled
		if existingFound {
			message = "External source package changed; immutable candidate staged for operator review."
			action = EventUpgraded
		}
		_, _ = s.store.CreateEvent(ctx, EventInput{
			ExtensionID: installed.ID, Action: action, Message: message,
		})
		result.Items = append(result.Items, installed)
	}

	sort.Slice(result.Items, func(i, j int) bool { return result.Items[i].ID < result.Items[j].ID })
	sort.Slice(result.Diagnostics, func(i, j int) bool {
		left := strings.Join([]string{result.Diagnostics[i].Root, result.Diagnostics[i].PackagePath, result.Diagnostics[i].Code}, "\x00")
		right := strings.Join([]string{result.Diagnostics[j].Root, result.Diagnostics[j].PackagePath, result.Diagnostics[j].Code}, "\x00")
		return left < right
	})
	return result, nil
}

func externalArtifactAlreadyKnown(existing Extension, version, digest string) bool {
	if existing.ID == "" {
		return false
	}
	if existing.Version == version && strings.EqualFold(existing.PackageDigest, digest) {
		return true
	}
	return existing.StagedVersion != nil && existing.StagedVersion.Version == version &&
		strings.EqualFold(existing.StagedVersion.PackageDigest, digest)
}
