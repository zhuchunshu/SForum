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
	themecompiler "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeCompiler"
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
// V3 静态上传只保存不可变候选；活动 runtime、状态、信任和 provider 选择均保持不变。
func (s *Service) InstallOrUpgradeArchive(ctx context.Context, actor identity.Actor, input ArchiveInput) (InstallResult, error) {
	// 上传前先挡住无扩展管理权限的调用者；包类型只能在有界静态解析后确定。
	if !canManagePlugins(actor) && !canManageThemes(actor) {
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
	switch manifest.Type {
	case TypePlugin:
		if !canManagePlugins(actor) {
			return InstallResult{}, identity.ErrPermissionDenied
		}
	case TypeTheme:
		if !canManageThemes(actor) {
			return InstallResult{}, identity.ErrPermissionDenied
		}
	default:
		return InstallResult{}, ErrInvalidManifest
	}
	if manifest.ID == DefaultThemeID {
		return InstallResult{}, ErrInvalidManifest
	}

	// V3 静态安装只校验并保存惰性包，不执行包代码；迁移开关关闭时保留 v1 边界。
	if !s.trustChallengesEnabled {
		if err := requireSuperAdminForUntrustedBackend(actor, SourceUploaded, manifest); err != nil {
			s.denyUntrustedBackend(ctx, actor, manifest.ID, "install")
			return InstallResult{}, err
		}
	}

	var previous Extension
	var isUpgrade bool
	if existing, getErr := s.store.Get(ctx, manifest.ID); getErr == nil {
		previous = existing
		isUpgrade = true
		if existing.Source == SourceBuiltin || existing.IsSystem {
			return InstallResult{}, ErrNotDeletable
		}
	} else if !errors.Is(getErr, ErrExtensionNotFound) {
		return InstallResult{}, getErr
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
	ownedSnapshot, err := extensionpackage.SnapshotUploadedOwned(s.extensionRoot, manifestJSON, packageFiles)
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
	defer ownedSnapshot.Release()
	snapshot := ownedSnapshot.Snapshot
	if manifest.Type == TypeTheme {
		// 上传安装仍是惰性的，但所有 L1 模板必须在进入权威 Store 前完成
		// 静态安全检查、受限 AST 校验和 html/template 上下文编译。
		if err := themecompiler.NewCompiler(themecompiler.Limits{}).PreflightFS(os.DirFS(snapshot.Root)); err != nil {
			installErr := fmt.Errorf("%w: theme template preflight: %w", ErrInvalidManifest, err)
			return InstallResult{}, errors.Join(installErr, s.discardUnreferencedUploadedSnapshot(ctx, ownedSnapshot))
		}
	}

	adminFrontendDigest, err := ComputeAdminFrontendDigest(manifest, snapshot.Root)
	if err != nil {
		return InstallResult{}, errors.Join(ErrInvalidManifest, s.discardUnreferencedUploadedSnapshot(ctx, ownedSnapshot))
	}
	installed, err := s.store.SaveInstalled(ctx, SaveInstalledInput{
		Manifest:            manifest,
		PackagePath:         snapshot.Root,
		PackageDigest:       snapshot.Digest,
		AdminFrontendDigest: adminFrontendDigest,
	})
	if err != nil {
		// SaveInstalled may have committed before returning an error (for example,
		// a failed post-commit reload). Retain the bytes for later reconciliation.
		return InstallResult{}, err
	}
	_ = ownedSnapshot.Release()

	result := InstallResult{
		Extension:         s.decorateRuntime(ctx, installed),
		Upgraded:          isUpgrade,
		RequiredReEnable:  false,
		ActivationPending: isUpgrade && installed.StagedVersion != nil,
	}
	if isUpgrade {
		candidateVersion := installed.Version
		candidateDigest := installed.PackageDigest
		candidateAdminDigest := installed.AdminFrontendDigest
		if installed.StagedVersion != nil {
			candidateVersion = installed.StagedVersion.Version
			candidateDigest = installed.StagedVersion.PackageDigest
			candidateAdminDigest = installed.StagedVersion.AdminFrontendDigest
		}
		result.PreviousVersion = previous.Version
		result.PreviousDigest = previous.PackageDigest
		_, _ = s.store.CreateEvent(ctx, EventInput{
			ExtensionID: installed.ID,
			ActorUserID: actor.ID,
			Action:      EventUpgraded,
			Message:     fmt.Sprintf("Staged upgrade from %s to %s; activation is pending.", previous.Version, candidateVersion),
		})
		s.appendAudit(ctx, actor, audit.ActionExtensionUpgraded, map[string]any{
			"extensionId":          installed.ID,
			"type":                 installed.Type,
			"previousVersion":      previous.Version,
			"candidateVersion":     candidateVersion,
			"previousDigest":       previous.PackageDigest,
			"candidateDigest":      candidateDigest,
			"candidateAdminDigest": candidateAdminDigest,
			"activationPending":    installed.StagedVersion != nil,
			"trustRevoked":         false,
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

// discardUnreferencedUploadedSnapshot runs while the per-digest lock is held.
// Any Store uncertainty or active/staged reference retains the immutable bytes.
func (s *Service) discardUnreferencedUploadedSnapshot(ctx context.Context, snapshot *extensionpackage.OwnedSnapshot) error {
	if snapshot == nil || !snapshot.Created() {
		return nil
	}
	if store, ok := s.store.(interface {
		PackagePathReferenced(context.Context, string) (bool, error)
	}); ok {
		referenced, err := store.PackagePathReferenced(ctx, snapshot.Root)
		if err != nil || referenced {
			return nil
		}
		if err := snapshot.RemoveIfCreated(); err != nil {
			return fmt.Errorf("discard unreferenced uploaded snapshot: %w", err)
		}
		return nil
	}
	items, err := s.store.List(ctx)
	if err != nil {
		return nil
	}
	for _, item := range items {
		if samePackagePath(item.PackagePath, snapshot.Root) ||
			(item.StagedVersion != nil && samePackagePath(item.StagedVersion.PackagePath, snapshot.Root)) {
			return nil
		}
	}
	if err := snapshot.RemoveIfCreated(); err != nil {
		return fmt.Errorf("discard unreferenced uploaded snapshot: %w", err)
	}
	return nil
}

func samePackagePath(left string, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	return left != "" && right != "" && filepath.Clean(left) == filepath.Clean(right)
}

func (s *Service) Uninstall(ctx context.Context, actor identity.Actor, id string, input UninstallInput) error {
	_, err := s.UninstallWithResult(ctx, actor, id, input)
	return err
}

// UninstallWithResult 先检查 durable replay，再选择 V2 coordinator 或 V1 兼容路径。
// V2 的 package/runtime/authority 会保留到 terminal success，物理删除只能由 exact-receipt finalizer 执行。
func (s *Service) UninstallWithResult(ctx context.Context, actor identity.Actor, id string, input UninstallInput) (UninstallResult, error) {
	if !canManagePlugins(actor) {
		return UninstallResult{}, identity.ErrPermissionDenied
	}
	id = normalizeID(id)
	if id == "" {
		return UninstallResult{}, ErrExtensionNotFound
	}
	if validLifecycleServiceIdempotencyKey(input.IdempotencyKey) && s.lifecycleAuthority != nil {
		operation, err := s.lifecycleAuthority.OperationByIdempotencyKey(ctx, id, input.IdempotencyKey)
		switch {
		case err == nil:
			return s.replayLifecycleUninstall(ctx, actor, id, input, operation)
		case !errors.Is(err, ErrLifecycleOperationNotFound):
			return UninstallResult{}, errors.Join(ErrLifecycleCoordinatorUnavailable, err)
		}
	}
	extension, err := s.store.Get(ctx, id)
	if err != nil {
		return UninstallResult{}, err
	}
	if extension.Source == SourceBuiltin || extension.IsSystem || !extension.IsDeletable {
		return UninstallResult{}, ErrNotDeletable
	}
	if extension.ID == DefaultThemeID {
		return UninstallResult{}, ErrNotDeletable
	}
	if usesLifecycleV2(extension) && (extension.Status == StatusEnabled || extension.Status == StatusDisabled) {
		return s.uninstallLifecycleV2(ctx, actor, extension, input)
	}
	if err := requireSuperAdminForUntrustedBackend(actor, extension.Source, extension.Manifest); err != nil {
		s.denyUntrustedBackend(ctx, actor, extension.ID, "uninstall")
		return UninstallResult{}, err
	}
	if extension.Status == StatusEnabled {
		return UninstallResult{}, ErrMustDisableFirst
	}
	s.assetPublicationMu.Lock()
	defer s.assetPublicationMu.Unlock()
	extension, err = s.store.Get(ctx, extension.ID)
	if err != nil {
		return UninstallResult{}, err
	}
	if extension.Status == StatusEnabled {
		return UninstallResult{}, ErrMustDisableFirst
	}
	assetBefore := s.captureAssetPublicationSnapshot()
	assetMutation, err := s.quarantineExactAssetPublication(ctx, assetBefore, extension)
	if err != nil {
		return UninstallResult{}, fmt.Errorf("remove exact asset publication before uninstall: %w", err)
	}

	_ = s.drainPluginRuntime(ctx, extension)
	// 立即清除页面贡献（即便已 disable 也应幂等清理）
	if s.pageRegistry != nil {
		s.pageRegistry.ClearExtension(extension.ID)
	}
	packagePath := extension.PackagePath
	identityRetained := false
	if err := s.store.Delete(ctx, extension.ID); err != nil {
		// 历史 plugin_runtime_publication_members 对 extension_versions 为 ON DELETE RESTRICT：
		// 一旦进入 desired full-set，精确版本行不可物理删除（见 lifecycle publication 门禁）。
		// 目录卸载仍须成功：保留不可变身份，删除包文件，扩展保持 disabled。
		if isPublishedPluginRuntimeIdentityRetained(err) {
			identityRetained = true
		} else {
			if restoreErr := s.rollbackExactAssetMutation(assetMutation); restoreErr != nil {
				return UninstallResult{}, errors.Join(err, fmt.Errorf("restore asset publication after uninstall failure: %w", restoreErr))
			}
			return UninstallResult{}, err
		}
	}

	if !input.RetainPackage && packagePath != "" {
		if safe, ok := packagePathUnderRoot(s.extensionRoot, packagePath); ok {
			_ = os.RemoveAll(safe)
		}
	}

	// extension_events 在硬删除时 CASCADE；身份保留时仅写宿主 audit。
	s.appendAudit(ctx, actor, audit.ActionExtensionUninstalled, map[string]any{
		"extensionId":       extension.ID,
		"type":              extension.Type,
		"version":           extension.Version,
		"retainSettings":    input.RetainSettings,
		"retainPackage":     input.RetainPackage,
		"settingsDeleted":   !input.RetainSettings && !identityRetained,
		"identityRetained":  identityRetained,
		// v1：settings 随 extensions CASCADE 删除；RetainSettings 记入审计供后续独立备份表使用。
	})
	return UninstallResult{Uninstalled: true, ExtensionID: extension.ID}, nil
}

// isPublishedPluginRuntimeIdentityRetained 判断删除是否被不可变 runtime publication 历史挡住。
func isPublishedPluginRuntimeIdentityRetained(err error) bool {
	if err == nil {
		return false
	}
	// pgx/pgconn 包装：SQLSTATE 23503 + plugin_runtime_publication_members。
	msg := err.Error()
	return strings.Contains(msg, "plugin_runtime_publication_members") ||
		strings.Contains(msg, "extension_version_id_extensi_fkey")
}

func (s *Service) replayLifecycleUninstall(
	ctx context.Context,
	actor identity.Actor,
	extensionID string,
	input UninstallInput,
	operation LifecycleOperation,
) (UninstallResult, error) {
	removalMode, err := normalizeLifecycleRemovalMode(input.RemovalMode)
	if err != nil {
		return UninstallResult{}, err
	}
	if operation.ExtensionID != extensionID || operation.Operation != string(LifecycleMachineUninstall) ||
		operation.RemovalMode != removalMode ||
		(operation.RequestedByUserID > 0 && operation.RequestedByUserID != actor.ID) {
		return UninstallResult{}, ErrLifecycleFingerprintConflict
	}
	if operation.CompletedAt != nil {
		switch operation.TerminalResult {
		case LifecycleTerminalSucceeded:
			return s.finalizeLifecycleUninstall(ctx, extensionID, removalMode, operation, true)
		case LifecycleTerminalFailed, LifecycleTerminalCancelled:
			return UninstallResult{}, ErrLifecycleCoordinatorRetryRequired
		default:
			return UninstallResult{}, ErrLifecycleCoordinatorInvalid
		}
	}
	current, err := s.store.Get(ctx, extensionID)
	if err != nil {
		return UninstallResult{}, errors.Join(ErrLifecycleCoordinatorUnavailable, err)
	}
	_, found, err := s.replayLifecycleV2(ctx, actor, current, input.IdempotencyKey)
	if err != nil {
		return UninstallResult{}, err
	}
	if !found {
		return UninstallResult{}, ErrLifecycleCoordinatorInvalid
	}
	latest, err := s.lifecycleAuthority.OperationByIdempotencyKey(ctx, extensionID, input.IdempotencyKey)
	if err != nil {
		return UninstallResult{}, errors.Join(ErrLifecycleCoordinatorUnavailable, err)
	}
	return s.finalizeLifecycleUninstall(ctx, extensionID, removalMode, latest, true)
}

// ApplyDeclaredMigrations 将 manifest.migrations 登记到账本（F2.4 v1）。
// 不执行任意 SQL：只校验文件并记录 checksum，避免插件写核心库。
// 非内置后端插件的迁移登记仍限 super_admin，与启用边界一致。
func (s *Service) ApplyDeclaredMigrations(ctx context.Context, actor identity.Actor, id string) ([]MigrationRecord, error) {
	if !canManagePlugins(actor) {
		return nil, identity.ErrPermissionDenied
	}
	if s.safeMode {
		return nil, ErrSafeModeActive
	}
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return nil, err
	}
	if err := requireSuperAdminForUntrustedBackend(actor, extension.Source, extension.Manifest); err != nil {
		s.denyUntrustedBackend(ctx, actor, extension.ID, "migrations")
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

// StorageSelectionClearer 在禁用存储插件时把 attachment.provider 从 plugin:<id> 回落 local（E6.1）。
type StorageSelectionClearer interface {
	ClearStorageProviderSelectionIfMatch(ctx context.Context, extensionID string) error
}

type mailProviderSelectionStore interface {
	SelectedMailProvider(context.Context) (string, error)
	RestoreMailProvider(context.Context) error
}

type RouteProviderSelectionInvalidator interface {
	InvalidateRouteProviderSelections(
		ctx context.Context,
		extensionID string,
		actorUserID int64,
		auditEventID int64,
		reasonCode string,
	) error
}

type ProviderSlotSelectionInvalidator interface {
	InvalidateProviderSlotSelections(
		ctx context.Context,
		extensionID string,
		actorUserID int64,
		auditEventID int64,
		reasonCode string,
	) error
}

// clearPluginProviderSelections 是 V1/V2 共用的 Host-owned 幂等清理边界。
// 它不触碰 runtime；V2 的进程 drain 只能由 durable coordinator 管理。
func (s *Service) clearPluginProviderSelections(ctx context.Context, extensionID string) error {
	return s.clearPluginNonRouteProviderSelections(ctx, extensionID)
}

func (s *Service) clearPluginProviderSelectionsWithAudit(
	ctx context.Context,
	extensionID string,
	actorUserID int64,
	auditEventID int64,
	reasonCode string,
) error {
	if s.providerSlotSelections != nil {
		if actorUserID <= 0 || auditEventID <= 0 || reasonCode == "" {
			return fmt.Errorf("provider slot selection invalidation requires actor and audit evidence")
		}
		if err := s.providerSlotSelections.InvalidateProviderSlotSelections(
			ctx, extensionID, actorUserID, auditEventID, reasonCode,
		); err != nil {
			return err
		}
	}
	if s.routeProviderSelections != nil {
		if actorUserID <= 0 || auditEventID <= 0 || reasonCode == "" {
			return fmt.Errorf("route provider selection invalidation requires actor and audit evidence")
		}
		if err := s.routeProviderSelections.InvalidateRouteProviderSelections(
			ctx, extensionID, actorUserID, auditEventID, reasonCode,
		); err != nil {
			return err
		}
	}
	return s.clearPluginNonRouteProviderSelections(ctx, extensionID)
}

func (s *Service) clearPluginNonRouteProviderSelections(ctx context.Context, extensionID string) error {
	if selectionStore, ok := s.store.(mailProviderSelectionStore); ok {
		selected, err := selectionStore.SelectedMailProvider(ctx)
		if err != nil {
			return err
		}
		if selected == extensionID {
			if err := selectionStore.RestoreMailProvider(ctx); err != nil {
				return err
			}
		}
	}
	// E6.1：若附件存储选中了本插件，回落 local，避免孤儿 plugin: 选择。
	if s.storageSelection != nil {
		if err := s.storageSelection.ClearStorageProviderSelectionIfMatch(ctx, extensionID); err != nil {
			return err
		}
	}
	return nil
}

// drainPluginRuntime 禁用/升级/卸载共用的 V1 drain：停子进程、清 provider、发 disabled hook。
func (s *Service) drainPluginRuntime(ctx context.Context, extension Extension) error {
	if extension.Type != TypePlugin {
		return nil
	}
	if err := s.clearPluginProviderSelections(ctx, extension.ID); err != nil {
		return err
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
