package extensions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

// TrustRevoker 在包 digest 变化时吊销前端信任（F2.4 升级重审批）。
type TrustRevoker interface {
	RevokeAllForExtension(ctx context.Context, extensionID string, actorUserID int64) error
}

// WithTrustRevoker 注入升级时的信任吊销器。
func WithTrustRevoker(revoker TrustRevoker) ServiceOption {
	return func(s *Service) {
		s.trustRevoker = revoker
	}
}

// InstallArchive 安装或同 id 升级上传包（兼容旧调用方）。
func (s *Service) InstallArchive(ctx context.Context, actor identity.Actor, input ArchiveInput) (Extension, error) {
	result, err := s.InstallOrUpgradeArchive(ctx, actor, input)
	if err != nil {
		return Extension{}, err
	}
	return result.Extension, nil
}

// InstallOrUpgradeArchive 返回完整升级元数据（F2.4）。
// 升级：同 id 时 drain runtime、状态回 installed、吊销前端信任、保留 settings。
func (s *Service) InstallOrUpgradeArchive(ctx context.Context, actor identity.Actor, input ArchiveInput) (InstallResult, error) {
	if !canManagePlugins(actor) {
		return InstallResult{}, identity.ErrPermissionDenied
	}
	if len(input.Data) == 0 || len(input.Data) > maxArchiveBytes {
		return InstallResult{}, ErrInvalidArchive
	}

	manifest, files, err := readArchive(input.Data)
	if err != nil {
		return InstallResult{}, err
	}
	if err := validateManifest(manifest); err != nil {
		return InstallResult{}, err
	}
	if manifest.ID == DefaultThemeID {
		return InstallResult{}, ErrInvalidManifest
	}

	var previous Extension
	var isUpgrade bool
	if existing, getErr := s.store.Get(ctx, manifest.ID); getErr == nil {
		previous = existing
		isUpgrade = true
		if existing.Source == SourceBuiltin || existing.IsSystem {
			return InstallResult{}, ErrNotDeletable
		}
		if err := s.drainPluginRuntime(ctx, existing); err != nil {
			return InstallResult{}, err
		}
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return InstallResult{}, err
	}
	packageFiles := make([]extensionpackage.File, 0, len(files))
	for _, file := range files {
		packageFiles = append(packageFiles, extensionpackage.File{
			Path: file.name,
			Mode: file.mode,
			Body: file.body,
		})
	}
	snapshot, err := extensionpackage.SnapshotUploaded(s.extensionRoot, manifestJSON, packageFiles)
	if err != nil {
		switch {
		case errors.Is(err, extensionpackage.ErrInvalidManifest):
			return InstallResult{}, ErrInvalidManifest
		case errors.Is(err, extensionpackage.ErrInvalidPath),
			errors.Is(err, extensionpackage.ErrNonRegular),
			errors.Is(err, extensionpackage.ErrSymlink):
			return InstallResult{}, ErrInvalidArchive
		}
		return InstallResult{}, err
	}

	installed, err := s.store.SaveInstalled(ctx, SaveInstalledInput{
		Manifest:      manifest,
		PackagePath:   snapshot.Root,
		PackageDigest: snapshot.Digest,
	})
	if err != nil {
		return InstallResult{}, err
	}

	_, _ = s.recordDeclaredMigrations(ctx, installed)

	result := InstallResult{
		Extension:        s.decorateRuntime(ctx, installed),
		Upgraded:         isUpgrade,
		RequiredReEnable: isUpgrade && previous.Type == TypePlugin,
	}
	if isUpgrade {
		result.PreviousVersion = previous.Version
		result.PreviousDigest = previous.PackageDigest
		if previous.PackageDigest != "" && previous.PackageDigest != installed.PackageDigest {
			result.TrustRevoked = true
			if s.trustRevoker != nil {
				_ = s.trustRevoker.RevokeAllForExtension(ctx, installed.ID, actor.ID)
			}
		}
		_, _ = s.store.CreateEvent(ctx, EventInput{
			ExtensionID: installed.ID,
			ActorUserID: actor.ID,
			Action:      EventUpgraded,
			Message:     fmt.Sprintf("Upgraded from %s to %s.", previous.Version, installed.Version),
		})
		s.appendAudit(ctx, actor, audit.ActionExtensionUpgraded, map[string]any{
			"extensionId":     installed.ID,
			"type":            installed.Type,
			"previousVersion": previous.Version,
			"version":         installed.Version,
			"previousDigest":  previous.PackageDigest,
			"packageDigest":   installed.PackageDigest,
			"trustRevoked":    result.TrustRevoked,
		})
	} else {
		_, _ = s.store.CreateEvent(ctx, EventInput{
			ExtensionID: installed.ID,
			ActorUserID: actor.ID,
			Action:      EventInstalled,
			Message:     "Extension archive installed.",
		})
		s.appendAudit(ctx, actor, audit.ActionExtensionInstalled, map[string]any{
			"extensionId": installed.ID,
			"type":        installed.Type,
		})
	}
	return result, nil
}

// Uninstall 删除可删除扩展（F2.4）。
// enabled 插件/主题须先禁用；系统/内置不可删；默认删除 settings（CASCADE）与包目录。
func (s *Service) Uninstall(ctx context.Context, actor identity.Actor, id string, input UninstallInput) error {
	if !canManagePlugins(actor) {
		return identity.ErrPermissionDenied
	}
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return err
	}
	if extension.Source == SourceBuiltin || extension.IsSystem || !extension.IsDeletable {
		return ErrNotDeletable
	}
	if extension.ID == DefaultThemeID {
		return ErrNotDeletable
	}
	if extension.Status == StatusEnabled {
		return ErrMustDisableFirst
	}

	_ = s.drainPluginRuntime(ctx, extension)

	packagePath := extension.PackagePath
	if err := s.store.Delete(ctx, extension.ID); err != nil {
		return err
	}

	if !input.RetainPackage && packagePath != "" {
		if safe, ok := packagePathUnderRoot(s.extensionRoot, packagePath); ok {
			_ = os.RemoveAll(safe)
		}
	}

	// extension_events 已 CASCADE；仅写宿主 audit。
	s.appendAudit(ctx, actor, audit.ActionExtensionUninstalled, map[string]any{
		"extensionId":     extension.ID,
		"type":            extension.Type,
		"version":         extension.Version,
		"retainSettings":  input.RetainSettings,
		"retainPackage":   input.RetainPackage,
		"settingsDeleted": !input.RetainSettings,
		// v1：settings 随 extensions CASCADE 删除；RetainSettings 记入审计供后续独立备份表使用。
	})
	return nil
}

// ApplyDeclaredMigrations 将 manifest.migrations 登记到账本（F2.4 v1）。
// 不执行任意 SQL：只校验文件并记录 checksum，避免插件写核心库。
func (s *Service) ApplyDeclaredMigrations(ctx context.Context, actor identity.Actor, id string) ([]MigrationRecord, error) {
	if !canManagePlugins(actor) {
		return nil, identity.ErrPermissionDenied
	}
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return nil, err
	}
	applied, err := s.recordDeclaredMigrations(ctx, extension)
	if err != nil {
		return nil, err
	}
	if len(applied) > 0 {
		_, _ = s.store.CreateEvent(ctx, EventInput{
			ExtensionID: extension.ID,
			ActorUserID: actor.ID,
			Action:      EventMigrationsApplied,
			Message:     fmt.Sprintf("Recorded %d declared migration path(s).", len(applied)),
		})
	}
	return s.store.ListMigrationLedger(ctx, extension.ID)
}

// ListMigrations 读取迁移账本。
func (s *Service) ListMigrations(ctx context.Context, actor identity.Actor, id string) ([]MigrationRecord, error) {
	if !canViewExtensions(actor) && !canManagePlugins(actor) {
		return nil, identity.ErrPermissionDenied
	}
	if _, err := s.store.Get(ctx, normalizeID(id)); err != nil {
		return nil, err
	}
	return s.store.ListMigrationLedger(ctx, normalizeID(id))
}

// drainPluginRuntime 禁用/升级/卸载共用的 drain：停子进程、清 mail provider、发 disabled hook。
func (s *Service) drainPluginRuntime(ctx context.Context, extension Extension) error {
	if extension.Type != TypePlugin {
		return nil
	}
	if selectionStore, ok := s.store.(interface {
		SelectedMailProvider(context.Context) (string, error)
		RestoreMailProvider(context.Context) error
	}); ok {
		if selected, selectErr := selectionStore.SelectedMailProvider(ctx); selectErr == nil && selected == extension.ID {
			if err := selectionStore.RestoreMailProvider(ctx); err != nil {
				return err
			}
		}
	}
	if s.runtime != nil {
		_ = s.runtime.Stop(ctx, extension)
		if extension.Status == StatusEnabled {
			s.runtime.EmitHook(ctx, appevents.ExtensionDisabled, map[string]any{
				"extensionId": extension.ID,
				"reason":      "lifecycle_drain",
			})
		}
	}
	return nil
}

func (s *Service) recordDeclaredMigrations(ctx context.Context, extension Extension) ([]MigrationRecord, error) {
	if len(extension.Manifest.Migrations) == 0 {
		return nil, nil
	}
	existing, _ := s.store.ListMigrationLedger(ctx, extension.ID)
	known := map[string]bool{}
	for _, row := range existing {
		known[row.Path] = true
	}
	recorded := []MigrationRecord{}
	for _, migration := range extension.Manifest.Migrations {
		path := strings.TrimSpace(migration.Path)
		if path == "" || known[path] {
			continue
		}
		checksum := ""
		message := "declared path recorded; SQL not executed by host v1 runner"
		status := "recorded"
		if full, ok := InstalledFilePathForRuntime(extension, path); ok {
			if body, err := os.ReadFile(full); err == nil {
				sum := sha256.Sum256(body)
				checksum = hex.EncodeToString(sum[:])
			} else {
				status = "failed"
				message = "migration file missing in package: " + err.Error()
			}
		} else {
			status = "failed"
			message = "unsafe or missing migration path"
		}
		record := MigrationRecord{Path: path, Checksum: checksum, Status: status, Message: message}
		if err := s.store.RecordMigration(ctx, extension.ID, record); err != nil {
			return recorded, fmt.Errorf("%w: %v", ErrMigrationFailed, err)
		}
		recorded = append(recorded, record)
	}
	return recorded, nil
}

func packagePathUnderRoot(root, packagePath string) (string, bool) {
	root = filepath.Clean(strings.TrimSpace(root))
	packagePath = filepath.Clean(strings.TrimSpace(packagePath))
	if root == "" || packagePath == "" {
		return "", false
	}
	rel, err := filepath.Rel(root, packagePath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return packagePath, true
}
