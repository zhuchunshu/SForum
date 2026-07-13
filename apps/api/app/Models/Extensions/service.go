package extensions

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Capabilities"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Crypto"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

const maxArchiveBytes = 50 * 1024 * 1024

type Service struct {
	store         Store
	extensionRoot string
	builtinRoot   string
	runtime       RuntimeManager
	// auditor 写入宿主 audit_events（F1.4）；与 extension_events 互补。
	auditor audit.Writer
	// trustRevoker 升级时吊销前端信任（F2.4）。
	trustRevoker TrustRevoker
	// featureFlags 检查 requiresFeatures（F4.5）；未注入时跳过门禁（测试兼容）。
	featureFlags FeatureFlagSource
	// cipher 加密 manifest type=secret 的设置；nil/透明时开发环境可存明文。
	cipher *crypto.OptionCipher
	// storageSelection 禁用存储插件时回落 attachment.provider（E6.1）。
	storageSelection StorageSelectionClearer
	// pageRegistry 运行时主题页面贡献（L0/L1）；nil 时跳过注册。
	pageRegistry    PageRegistry
	settingsActions SettingsActionRuntime
	// executableTrust 在 V3 P1 开关开启时取代 confirmCapabilities 布尔确认。
	executableTrust        *ExecutableTrustService
	trustChallengesEnabled bool
	safeMode               bool
	activation             *ActivationCoordinator
}

// PageRegistry 主题/插件页面贡献注册（避免 extensions 直接依赖 pages 包实现细节）。
type PageRegistry interface {
	// PreflightThemePackage 激活前完整预检，不修改 Registry。
	// previousActiveThemeID 非空时按「替换旧主题」的最终状态校验 add 路径。
	PreflightThemePackage(ctx context.Context, extension Extension, previousActiveThemeID string) error
	// RegisterThemePackage 校验并注册主题页面贡献（仅候选，不批准 replace）。
	RegisterThemePackage(ctx context.Context, extension Extension) error
	// RegisterThemePackageReplacing 原子替换旧活动主题贡献（同路径切换允许）。
	RegisterThemePackageReplacing(ctx context.Context, extension Extension, previousActiveThemeID string) error
	// RegisterPluginPackage 插件 enable 时注册页面贡献（统一 theme.json pages 契约）。
	RegisterPluginPackage(ctx context.Context, extension Extension) error
	// ClearExtension 禁用/卸载/切换时撤销贡献。
	ClearExtension(extensionID string)
}

// FeatureFlagSource 返回 requiresFeatures 中当前关闭的 key。
type FeatureFlagSource interface {
	MissingRequiredFeatures(ctx context.Context, required []string) ([]string, error)
}

// ServiceOption 注入扩展服务的可选宿主能力。
type ServiceOption func(*Service)

// WithAuditor 注入宿主 audit_events 写入（F1.4 扩展生命周期审计）。
func WithAuditor(w audit.Writer) ServiceOption {
	return func(s *Service) {
		s.auditor = w
	}
}

// WithFeatureFlags 注入站点产品开关检查（F4.5 requiresFeatures）。
func WithFeatureFlags(source FeatureFlagSource) ServiceOption {
	return func(s *Service) {
		s.featureFlags = source
	}
}

// WithStorageSelectionClearer 注入附件存储选择回落（E6.1 禁用插件时）。
func WithStorageSelectionClearer(clearer StorageSelectionClearer) ServiceOption {
	return func(s *Service) {
		s.storageSelection = clearer
	}
}

// WithPageRegistry 注入运行时 Page Registry（主题激活无 Nuxt 重建）。
func WithPageRegistry(registry PageRegistry) ServiceOption {
	return func(s *Service) {
		s.pageRegistry = registry
	}
}

// WithCipher 注入与 web_options 相同的 AES-GCM 加密器（secret 设置静态加密）。
func WithCipher(c *crypto.OptionCipher) ServiceOption {
	return func(s *Service) {
		s.cipher = c
	}
}

// WithRuntimeManager 用于 bootstrap 的两阶段装配：先构造带 Cipher 的 Service，
// 再把使用该 Service 读取解密设置的插件 runtime 绑定回来。
func WithRuntimeManager(runtime RuntimeManager) ServiceOption {
	return func(s *Service) {
		if runtime != nil {
			s.runtime = runtime
			if actions, ok := runtime.(SettingsActionRuntime); ok {
				s.settingsActions = actions
			}
		}
	}
}

func WithSettingsActionRuntime(runtime SettingsActionRuntime) ServiceOption {
	return func(s *Service) { s.settingsActions = runtime }
}

// WithExecutableTrust 先注入兼容层；enabled=false 时完整保留 v1 启用行为。
func WithExecutableTrust(service *ExecutableTrustService, enabled bool) ServiceOption {
	return func(s *Service) {
		s.executableTrust = service
		s.trustChallengesEnabled = enabled
	}
}

func WithSafeMode(enabled bool) ServiceOption {
	return func(s *Service) { s.safeMode = enabled }
}

func WithActivationCoordinator(coordinator *ActivationCoordinator) ServiceOption {
	return func(s *Service) { s.activation = coordinator }
}

func (s *Service) ExecutableTrustStatus(ctx context.Context, actor identity.Actor, extensionID string) (ExecutableTrustStatus, error) {
	if !s.trustChallengesEnabled || s.executableTrust == nil {
		return ExecutableTrustStatus{}, ErrTrustNotRequired
	}
	return s.executableTrust.Status(ctx, actor, extensionID)
}

func (s *Service) IssueExecutableTrustChallenge(ctx context.Context, actor identity.Actor, extensionID string) (TrustChallenge, error) {
	if !s.trustChallengesEnabled || s.executableTrust == nil {
		return TrustChallenge{}, ErrTrustNotRequired
	}
	return s.executableTrust.Challenge(ctx, actor, extensionID)
}

func (s *Service) RevokeExecutableTrust(ctx context.Context, actor identity.Actor, extensionID string) (ExecutableTrustStatus, error) {
	if !s.trustChallengesEnabled || s.executableTrust == nil {
		return ExecutableTrustStatus{}, ErrTrustNotRequired
	}
	return s.executableTrust.Revoke(ctx, actor, extensionID)
}

func (s *Service) appendAudit(ctx context.Context, actor identity.Actor, action string, metadata map[string]any) {
	if s == nil || s.auditor == nil || action == "" {
		return
	}
	_ = s.auditor.Append(ctx, audit.Event{
		ActorUserID: actor.ID,
		Action:      action,
		Metadata:    metadata,
	})
}

func (s *Service) ensureRequiredFeatures(ctx context.Context, required []string) error {
	if s == nil || s.featureFlags == nil || len(required) == 0 {
		return nil
	}
	missing, err := s.featureFlags.MissingRequiredFeatures(ctx, required)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrFeaturesRequired, strings.Join(missing, ","))
	}
	return nil
}

func NewService(store Store, extensionRoot string) *Service {
	return NewServiceWithHooks(store, extensionRoot, nil)
}

func NewServiceWithHooks(store Store, extensionRoot string, runtime RuntimePreflight) *Service {
	return NewServiceWithBuiltinsAndHooks(store, extensionRoot, "", runtime)
}

func NewServiceWithBuiltins(store Store, extensionRoot string, builtinRoot string) *Service {
	return NewServiceWithBuiltinsAndRuntime(store, extensionRoot, builtinRoot, nil)
}

func NewServiceWithBuiltinsAndHooks(store Store, extensionRoot string, builtinRoot string, runtime RuntimePreflight) *Service {
	var manager RuntimeManager
	if runtime != nil {
		if full, ok := runtime.(RuntimeManager); ok {
			manager = full
		} else {
			manager = preflightRuntimeManager{RuntimePreflight: runtime}
		}
	}
	return NewServiceWithBuiltinsAndRuntime(store, extensionRoot, builtinRoot, manager)
}

func NewServiceWithRuntime(store Store, extensionRoot string, runtime RuntimeManager) *Service {
	return NewServiceWithBuiltinsAndRuntime(store, extensionRoot, "", runtime)
}

func NewServiceWithBuiltinsAndRuntime(store Store, extensionRoot string, builtinRoot string, runtime RuntimeManager) *Service {
	if strings.TrimSpace(extensionRoot) == "" {
		extensionRoot = "storage/extensions"
	}
	if runtime == nil {
		runtime = LocalRuntimeManager{}
	}
	return &Service{
		store:         store,
		extensionRoot: extensionRoot,
		builtinRoot:   strings.TrimSpace(builtinRoot),
		runtime:       runtime,
	}
}

func NewServiceWithOptions(store Store, extensionRoot string, builtinRoot string, runtime RuntimeManager, options ...ServiceOption) *Service {
	service := NewServiceWithBuiltinsAndRuntime(store, extensionRoot, builtinRoot, runtime)
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service
}

type LocalRuntimePreflight struct{}

func (LocalRuntimePreflight) Check(_ context.Context, extension Extension) error {
	if extension.Manifest.Backend.Entry == "" {
		return nil
	}
	entry, ok := installedFilePath(extension, extension.Manifest.Backend.Entry)
	if !ok {
		return ErrInvalidManifest
	}
	info, err := os.Stat(entry)
	if err != nil || info.IsDir() {
		return fmt.Errorf("backend entry %s is not available", extension.Manifest.Backend.Entry)
	}
	return nil
}

type LocalRuntimeManager struct {
	LocalRuntimePreflight
}

func (LocalRuntimeManager) Start(context.Context, Extension) error {
	return nil
}

func (LocalRuntimeManager) Stop(context.Context, Extension) error {
	return nil
}

func (LocalRuntimeManager) Status(_ context.Context, extension Extension) RuntimeStatus {
	return RuntimeStatus{
		State:         RuntimeStopped,
		RouteCount:    len(extension.Manifest.Routes),
		HookCount:     len(manifestEvents(extension.Manifest)),
		EventCount:    len(manifestEvents(extension.Manifest)),
		ProviderCount: len(extension.Manifest.Providers),
	}
}

func (LocalRuntimeManager) EmitHook(context.Context, string, map[string]any) {}

type preflightRuntimeManager struct {
	RuntimePreflight
}

func (preflightRuntimeManager) Start(context.Context, Extension) error {
	return nil
}

func (preflightRuntimeManager) Stop(context.Context, Extension) error {
	return nil
}

func (preflightRuntimeManager) Status(_ context.Context, extension Extension) RuntimeStatus {
	return RuntimeStatus{
		State:         RuntimeStopped,
		RouteCount:    len(extension.Manifest.Routes),
		HookCount:     len(manifestEvents(extension.Manifest)),
		EventCount:    len(manifestEvents(extension.Manifest)),
		ProviderCount: len(extension.Manifest.Providers),
	}
}

func (preflightRuntimeManager) EmitHook(context.Context, string, map[string]any) {}

// canViewExtensions 扩展目录只读（列表/事件/贡献/导航）。
func canViewExtensions(actor identity.Actor) bool {
	return actor.Can(identity.PermissionExtensionView) || actor.Can(identity.PermissionExtensionManage)
}

// canManagePlugins 插件启停/安装/校验与插件设置。
func canManagePlugins(actor identity.Actor) bool {
	return actor.Can(identity.PermissionExtensionPluginManage) || actor.Can(identity.PermissionExtensionManage)
}

// canManageThemes 主题激活。
func canManageThemes(actor identity.Actor) bool {
	return actor.Can(identity.PermissionExtensionThemeManage) || actor.Can(identity.PermissionExtensionManage)
}

func (s *Service) List(ctx context.Context, actor identity.Actor) ([]Extension, error) {
	if !canViewExtensions(actor) {
		return nil, identity.ErrPermissionDenied
	}
	items, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index] = s.decorateRuntime(ctx, items[index])
	}
	return items, nil
}

func (s *Service) SyncBuiltins(ctx context.Context) ([]Extension, error) {
	if strings.TrimSpace(s.builtinRoot) == "" {
		return nil, nil
	}

	groups := []struct {
		dir           string
		extensionType string
	}{
		{dir: "plugins", extensionType: TypePlugin},
		{dir: "themes", extensionType: TypeTheme},
	}
	items := []Extension{}
	activeBuiltinIDs := map[string]bool{}
	for _, group := range groups {
		root := filepath.Join(s.builtinRoot, group.dir)
		entries, err := os.ReadDir(root)
		if errorsIsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			packageRoot := filepath.Join(root, entry.Name())
			// LoadPackage 解析 includes 并合并为单一 Manifest，与上传安装路径一致。
			manifest, err := extensionmanifest.LoadPackage(packageRoot)
			if err != nil {
				return nil, fmt.Errorf("builtin %s: %w", entry.Name(), err)
			}
			if manifest.Type != group.extensionType {
				return nil, ErrInvalidManifest
			}
			snapshot, err := extensionpackage.SnapshotBuiltin(packageRoot, s.extensionRoot)
			if err != nil {
				return nil, err
			}
			adminFrontendDigest, err := ComputeAdminFrontendDigest(manifest, snapshot.Root)
			if err != nil {
				return nil, fmt.Errorf("builtin %s admin frontend: %w", entry.Name(), err)
			}
			item, err := s.store.SaveBuiltin(ctx, SaveBuiltinInput{
				Manifest:            manifest,
				PackagePath:         snapshot.Root,
				PackageDigest:       snapshot.Digest,
				AdminFrontendDigest: adminFrontendDigest,
			})
			if err != nil {
				return nil, err
			}
			activeBuiltinIDs[item.ID] = true
			_, _ = s.store.CreateEvent(ctx, EventInput{
				ExtensionID: item.ID,
				Action:      EventBuiltinSynced,
				Message:     "Built-in extension synchronized from Git.",
			})
			items = append(items, item)
		}
	}
	if len(activeBuiltinIDs) > 0 {
		ids := make([]string, 0, len(activeBuiltinIDs))
		for id := range activeBuiltinIDs {
			ids = append(ids, id)
		}
		if err := s.store.PruneMissingBuiltins(ctx, ids); err != nil {
			return nil, err
		}
	}
	if _, err := s.EnsureDefaultThemeActive(ctx); err != nil && !errors.Is(err, ErrExtensionNotFound) {
		return nil, err
	}
	return items, nil
}

func (s *Service) Events(ctx context.Context, actor identity.Actor, extensionID string, limit int) ([]ExtensionEvent, error) {
	if !canViewExtensions(actor) {
		return nil, identity.ErrPermissionDenied
	}
	return s.store.ListEvents(ctx, normalizeID(extensionID), limit)
}

func (s *Service) EventDefinitions(ctx context.Context, actor identity.Actor) ([]appevents.Definition, error) {
	if !canViewExtensions(actor) {
		return nil, identity.ErrPermissionDenied
	}
	return appevents.Definitions(), nil
}

func (s *Service) EventDeliveries(ctx context.Context, actor identity.Actor, input EventDeliveryListInput) ([]ExtensionEventDelivery, error) {
	if !canViewExtensions(actor) {
		return nil, identity.ErrPermissionDenied
	}
	input.ExtensionID = normalizeID(input.ExtensionID)
	return s.store.ListEventDeliveries(ctx, input)
}

func (s *Service) ContributionPoints(_ context.Context, actor identity.Actor) ([]ContributionPointDefinition, error) {
	if !canViewExtensions(actor) {
		return nil, identity.ErrPermissionDenied
	}
	return extensionmanifest.ContributionPointDefinitions(), nil
}

func (s *Service) Contributions(ctx context.Context, actor identity.Actor) ([]EffectiveContribution, error) {
	if !canViewExtensions(actor) {
		return nil, identity.ErrPermissionDenied
	}
	return s.EffectiveContributions(ctx)
}

func (s *Service) EffectiveContributions(ctx context.Context) ([]EffectiveContribution, error) {
	if s.safeMode {
		return []EffectiveContribution{}, nil
	}
	items, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	contributions := []EffectiveContribution{}
	for _, item := range items {
		if item.Type != TypePlugin || item.Status != StatusEnabled {
			continue
		}
		manifest := normalizeManifest(item.Manifest)
		for _, contribution := range manifest.Contributions {
			contributions = append(contributions, EffectiveContribution{
				ExtensionID:   item.ID,
				ExtensionName: item.Name,
				ExtensionType: item.Type,
				Point:         contribution.Point,
				ID:            contribution.ID,
				Order:         contribution.Order,
				Label:         contribution.Label,
				Icon:          contribution.Icon,
				Payload:       contribution.Payload,
			})
		}
	}
	sort.SliceStable(contributions, func(left, right int) bool {
		if contributions[left].Order != contributions[right].Order {
			return contributions[left].Order < contributions[right].Order
		}
		if contributions[left].ExtensionID != contributions[right].ExtensionID {
			return contributions[left].ExtensionID < contributions[right].ExtensionID
		}
		return contributions[left].ID < contributions[right].ID
	})
	return contributions, nil
}

func (s *Service) Navigation(ctx context.Context, actor identity.Actor) ([]ExtensionAdminNavigationItem, error) {
	if !canViewExtensions(actor) {
		return nil, identity.ErrPermissionDenied
	}
	if s.safeMode {
		return []ExtensionAdminNavigationItem{}, nil
	}
	items, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	navigation := []ExtensionAdminNavigationItem{}
	for _, item := range items {
		if !extensionInjectsAdminNavigation(item) {
			continue
		}
		for _, page := range normalizedMenuAdminPages(item.Manifest) {
			navigation = append(navigation, ExtensionAdminNavigationItem{
				ExtensionID:     item.ID,
				ExtensionName:   item.Name,
				ExtensionType:   item.Type,
				ExtensionStatus: item.Status,
				Path:            page.Path,
				Label:           page.Label,
				Description:     page.Description,
				Icon:            page.Icon,
				View:            page.View,
				Order:           page.Order,
			})
		}
	}
	sort.SliceStable(navigation, func(left, right int) bool {
		if navigation[left].Order == navigation[right].Order {
			if navigation[left].ExtensionName == navigation[right].ExtensionName {
				return navigation[left].Path < navigation[right].Path
			}
			return navigation[left].ExtensionName < navigation[right].ExtensionName
		}
		return navigation[left].Order < navigation[right].Order
	})
	return navigation, nil
}

func (s *Service) Settings(ctx context.Context, actor identity.Actor, extensionID string, locale string) (ExtensionSettings, error) {
	extension, err := s.store.Get(ctx, normalizeID(extensionID))
	if err != nil {
		return ExtensionSettings{}, err
	}
	if !canManageExtensionSettings(actor, extension) {
		return ExtensionSettings{}, identity.ErrPermissionDenied
	}
	values, err := s.listDecryptedSettings(ctx, extension)
	if err != nil {
		return ExtensionSettings{}, err
	}
	return resolveExtensionSettings(extension, values, locale), nil
}

func (s *Service) UpdateSettings(ctx context.Context, actor identity.Actor, extensionID string, input UpdateSettingsInput, locale string) (ExtensionSettings, error) {
	extension, err := s.store.Get(ctx, normalizeID(extensionID))
	if err != nil {
		return ExtensionSettings{}, err
	}
	if !canManageExtensionSettings(actor, extension) {
		return ExtensionSettings{}, identity.ErrPermissionDenied
	}
	current, err := s.listDecryptedSettings(ctx, extension)
	if err != nil {
		return ExtensionSettings{}, err
	}
	values, err := sanitizeSettingValues(extension.Manifest, input.Values, current)
	if err != nil {
		return ExtensionSettings{}, err
	}
	stored, err := s.encryptSecretSettings(extension.Manifest, values)
	if err != nil {
		return ExtensionSettings{}, err
	}
	// 持久化前保留 previous 密文快照，便于重启失败时回滚。
	previousRaw, err := s.store.ListSettings(ctx, extension.ID)
	if err != nil {
		return ExtensionSettings{}, err
	}
	if err := s.store.ReplaceSettings(ctx, extension.ID, stored); err != nil {
		return ExtensionSettings{}, err
	}
	if err := s.restartPluginForSettings(ctx, extension); err != nil {
		return ExtensionSettings{}, s.restoreSettingsAfterRestartFailure(ctx, extension.ID, previousRaw, err)
	}
	// 返回解密后的视图（secret 仍在 resolve 中掩码）。
	return resolveExtensionSettings(extension, values, locale), nil
}

func (s *Service) ResetSettings(ctx context.Context, actor identity.Actor, extensionID string, locale string) (ExtensionSettings, error) {
	extension, err := s.store.Get(ctx, normalizeID(extensionID))
	if err != nil {
		return ExtensionSettings{}, err
	}
	if !canManageExtensionSettings(actor, extension) {
		return ExtensionSettings{}, identity.ErrPermissionDenied
	}
	previousRaw, err := s.store.ListSettings(ctx, extension.ID)
	if err != nil {
		return ExtensionSettings{}, err
	}
	if err := s.store.ResetSettings(ctx, extension.ID); err != nil {
		return ExtensionSettings{}, err
	}
	if err := s.restartPluginForSettings(ctx, extension); err != nil {
		return ExtensionSettings{}, s.restoreSettingsAfterRestartFailure(ctx, extension.ID, previousRaw, err)
	}
	return resolveExtensionSettings(extension, map[string]string{}, locale), nil
}

func (s *Service) restoreSettingsAfterRestartFailure(ctx context.Context, extensionID string, previous map[string]string, restartErr error) error {
	if restoreErr := s.store.ReplaceSettings(ctx, extensionID, previous); restoreErr != nil {
		return errors.Join(
			ErrSettingsRollbackFailed,
			fmt.Errorf("restart extension after settings change: %w", restartErr),
			fmt.Errorf("restore previous extension settings: %w", restoreErr),
		)
	}
	return fmt.Errorf("restart extension after settings change: %w", restartErr)
}

func canManageExtensionSettings(actor identity.Actor, extension Extension) bool {
	// 主题设置：extension.theme.manage（或兼容父权限 extension.manage）。
	if extension.Type == TypeTheme {
		return canManageThemes(actor)
	}
	// 插件设置：extension.plugin.manage；邮件提供商插件也允许 settings.mail.manage。
	if canManagePlugins(actor) {
		return true
	}
	if !actor.Can(identity.PermissionSettingsMailManage) {
		return false
	}
	for _, provider := range extension.Manifest.Providers {
		if provider.Slot == "mail.provider" {
			return true
		}
	}
	return false
}

// PublicActiveThemeSettings 返回当前激活主题的非 secret 设置（含默认值）。
// 供前台主题 layer 读取可运营配置；secret 永不出现在公开响应中。
func (s *Service) PublicActiveThemeSettings(ctx context.Context) (PublicActiveThemeSettings, error) {
	var theme Extension
	var err error
	if s.safeMode {
		theme, err = s.store.Get(ctx, DefaultThemeID)
	} else {
		theme, err = s.store.ActiveTheme(ctx)
	}
	if err != nil {
		return PublicActiveThemeSettings{Settings: map[string]string{}}, nil
	}
	if theme.Type != TypeTheme {
		return PublicActiveThemeSettings{Settings: map[string]string{}}, nil
	}
	values, err := s.listDecryptedSettings(ctx, theme)
	if err != nil {
		return PublicActiveThemeSettings{}, err
	}
	settings := map[string]string{}
	for _, setting := range theme.Manifest.Settings {
		if setting.Type == "secret" {
			continue
		}
		key := strings.TrimSpace(setting.Key)
		if key == "" {
			continue
		}
		if stored, ok := values[key]; ok {
			settings[key] = stored
			continue
		}
		settings[key] = setting.Default
	}
	return PublicActiveThemeSettings{
		ThemeID:  theme.ID,
		Settings: settings,
	}, nil
}

func (s *Service) restartPluginForSettings(ctx context.Context, extension Extension) error {
	if s.safeMode {
		return nil
	}
	if extension.Type != TypePlugin || extension.Status != StatusEnabled || extension.Manifest.Backend.Entry == "" || s.runtime == nil {
		return nil
	}
	if err := s.runtime.Stop(ctx, extension); err != nil {
		return err
	}
	return s.runtime.Start(ctx, extension)
}

func (s *Service) MatchRoute(ctx context.Context, extensionID string, method string, routePath string) (MatchedRoute, error) {
	if s.safeMode {
		return MatchedRoute{}, ErrRouteNotFound
	}
	extension, err := s.store.Get(ctx, normalizeID(extensionID))
	if err != nil {
		return MatchedRoute{}, err
	}
	if extension.Type != TypePlugin || extension.Status != StatusEnabled {
		return MatchedRoute{}, ErrRouteNotFound
	}
	normalizedPath := normalizeRoutePath(routePath)
	pathExists := false
	for _, route := range extension.Manifest.Routes {
		if normalizeRoutePath(route.Path) != normalizedPath {
			continue
		}
		pathExists = true
		for _, allowed := range route.Methods {
			if strings.EqualFold(allowed, method) {
				return MatchedRoute{Extension: extension, Route: route, Path: normalizedPath}, nil
			}
		}
	}
	if pathExists {
		return MatchedRoute{}, ErrRouteMethodNotAllowed
	}
	return MatchedRoute{}, ErrRouteNotFound
}

func (s *Service) Enable(ctx context.Context, actor identity.Actor, id string, input EnableInput) (Extension, error) {
	if !canManagePlugins(actor) {
		return Extension{}, identity.ErrPermissionDenied
	}
	if s.safeMode {
		return Extension{}, ErrSafeModeActive
	}
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return Extension{}, err
	}
	if extension.Type == TypeTheme {
		return Extension{}, ErrThemeActivationRequired
	}
	if s.trustChallengesEnabled {
		if s.executableTrust == nil {
			return Extension{}, ErrTrustChallengeRequired
		}
		if err := s.executableTrust.ConfirmEnable(ctx, actor, extension, input.ConfirmationToken); err != nil {
			return Extension{}, err
		}
	} else {
		// v1 兼容：非内置后端插件启动子进程仅允许 super_admin。
		if err := requireSuperAdminForUntrustedBackend(actor, extension.Source, extension.Manifest); err != nil {
			s.denyUntrustedBackend(ctx, actor, extension.ID, "enable")
			return Extension{}, err
		}
	}

	// 首次启用（非 restart）且存在 Host 能力时，要求运营显式确认（F2.1）。
	if !s.trustChallengesEnabled && extension.Status != StatusEnabled {
		capKeys, _ := extensionmanifest.ResolvedCapabilities(extension.Manifest)
		if capabilities.RequiresConfirmation(capKeys) && !input.ConfirmCapabilities {
			return Extension{}, ErrCapabilityConfirmationRequired
		}
	}

	// F4.5：manifest requiresFeatures 必须全部开启。
	if err := s.ensureRequiredFeatures(ctx, extension.Manifest.RequiresFeatures); err != nil {
		return Extension{}, err
	}

	if _, err := s.preflightActivationDependencies(ctx, extension); err != nil {
		s.recordEnableFailure(ctx, actor, extension.ID, err)
		return Extension{}, err
	}

	if err := s.verifyExtension(ctx, extension); err != nil {
		s.recordEnableFailure(ctx, actor, extension.ID, err)
		return Extension{}, err
	}
	enabled, err := s.store.Enable(ctx, extension.ID, extension.Type)
	if err != nil {
		return Extension{}, err
	}
	if enabled.Type == TypePlugin && enabled.Manifest.Backend.Entry != "" && s.runtime != nil {
		var startErr error
		if s.activation != nil {
			startErr = s.activation.Start(ctx, s.runtime, enabled, ActivationTriggerEnable, actor.ID, NewActivationBootID())
		} else {
			startErr = s.runtime.Start(ctx, enabled)
		}
		if startErr != nil {
			_, _ = s.store.Disable(ctx, enabled.ID)
			s.recordEnableFailure(ctx, actor, enabled.ID, startErr)
			return Extension{}, fmt.Errorf("%w: %v", ErrRuntimeFailed, startErr)
		}
	}
	// 插件 enable：注册页面贡献（add/replace 候选）；replace 仍需 super_admin 批准。
	if enabled.Type == TypePlugin && s.pageRegistry != nil {
		if err := s.pageRegistry.RegisterPluginPackage(ctx, enabled); err != nil {
			// 页面贡献失败不静默：回滚 enable，避免半启用状态
			if s.runtime != nil {
				_ = s.runtime.Stop(ctx, enabled)
			}
			_, _ = s.store.Disable(ctx, enabled.ID)
			s.pageRegistry.ClearExtension(enabled.ID)
			s.recordEnableFailure(ctx, actor, enabled.ID, err)
			return Extension{}, fmt.Errorf("%w: page contributions: %v", ErrPreflightFailed, err)
		}
	}
	capKeys, _ := extensionmanifest.ResolvedCapabilities(enabled.Manifest)
	_, _ = s.store.CreateEvent(ctx, EventInput{
		ExtensionID: enabled.ID,
		ActorUserID: actor.ID,
		Action:      EventEnabled,
		Message:     "Extension enabled.",
	})
	s.appendAudit(ctx, actor, audit.ActionExtensionEnable, map[string]any{
		"extensionId":  enabled.ID,
		"type":         enabled.Type,
		"capabilities": capKeys,
	})
	if enabled.Type == TypePlugin && s.runtime != nil {
		s.runtime.EmitHook(ctx, appevents.ExtensionEnabled, map[string]any{"extensionId": enabled.ID})
	}
	return s.decorateRuntime(ctx, enabled), nil
}

func (s *Service) Disable(ctx context.Context, actor identity.Actor, id string) (Extension, error) {
	if !canManagePlugins(actor) {
		return Extension{}, identity.ErrPermissionDenied
	}
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return Extension{}, err
	}
	if extension.Type == TypeTheme {
		return Extension{}, ErrThemeActivationRequired
	}
	// F2.4：先 drain runtime（停进程、清 provider），再改 DB 状态。
	if err := s.drainPluginRuntime(ctx, extension); err != nil {
		return Extension{}, err
	}
	// 立即撤销页面贡献，使 replace 绑定在 Resolve 时回退 core。
	if s.pageRegistry != nil {
		s.pageRegistry.ClearExtension(extension.ID)
	}
	disabled, err := s.store.Disable(ctx, extension.ID)
	if err != nil {
		return Extension{}, err
	}
	_, _ = s.store.CreateEvent(ctx, EventInput{
		ExtensionID: disabled.ID,
		ActorUserID: actor.ID,
		Action:      EventDisabled,
		Message:     "Extension disabled.",
	})
	s.appendAudit(ctx, actor, audit.ActionExtensionDisable, map[string]any{
		"extensionId": disabled.ID,
		"type":        disabled.Type,
	})
	return s.decorateRuntime(ctx, disabled), nil
}

func (s *Service) VerifyExtension(ctx context.Context, actor identity.Actor, id string) (Extension, error) {
	if !canManagePlugins(actor) {
		return Extension{}, identity.ErrPermissionDenied
	}
	if s.safeMode {
		return Extension{}, ErrSafeModeActive
	}
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return Extension{}, err
	}
	// 预检会触碰后端入口；非内置后端包同样仅 super_admin。
	if err := requireSuperAdminForUntrustedBackend(actor, extension.Source, extension.Manifest); err != nil {
		s.denyUntrustedBackend(ctx, actor, extension.ID, "verify")
		return Extension{}, err
	}
	if err := s.verifyExtension(ctx, extension); err != nil {
		_, _ = s.store.CreateEvent(ctx, EventInput{
			ExtensionID: extension.ID,
			ActorUserID: actor.ID,
			Action:      EventEnableFailed,
			Message:     err.Error(),
		})
		return Extension{}, err
	}
	_, _ = s.store.CreateEvent(ctx, EventInput{
		ExtensionID: extension.ID,
		ActorUserID: actor.ID,
		Action:      EventVerified,
		Message:     "Extension preflight verified.",
	})
	return s.decorateRuntime(ctx, extension), nil
}

func (s *Service) ActivateTheme(ctx context.Context, actor identity.Actor, id string) (Extension, error) {
	if !canManageThemes(actor) {
		return Extension{}, identity.ErrPermissionDenied
	}
	if s.safeMode {
		return Extension{}, ErrSafeModeActive
	}
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return Extension{}, err
	}
	if extension.Type != TypeTheme {
		return Extension{}, ErrThemeActivationRequired
	}
	if err := s.verifyExtension(ctx, extension); err != nil {
		s.recordEnableFailure(ctx, actor, extension.ID, err)
		return Extension{}, err
	}

	// 记录旧主题，失败时完整回滚。
	var previous *Extension
	if prev, prevErr := s.store.ActiveTheme(ctx); prevErr == nil {
		p := prev
		previous = &p
	}

	// 1) 完整预检（manifest / theme.json / 模板 / CSS / routes / 贡献）—— 不改 DB、不改 Registry。
	// 按「用新主题替换旧活动主题」的最终状态校验，允许新旧主题相同 add 路径。
	// 主题预检失败使用 ErrBuildFailed，避免前端误显示「后端入口」插件文案。
	prevThemeID := ""
	if previous != nil {
		prevThemeID = previous.ID
	}
	if s.pageRegistry != nil {
		if err := s.pageRegistry.PreflightThemePackage(ctx, extension, prevThemeID); err != nil {
			wrapped := fmt.Errorf("%w: %v", ErrBuildFailed, err)
			s.recordEnableFailure(ctx, actor, extension.ID, wrapped)
			return Extension{}, wrapped
		}
	}

	// 2) DB 事务切换活动主题（store.ActivateTheme 内部事务；同主题再激活也幂等写状态）。
	active, err := s.store.ActivateTheme(ctx, extension.ID)
	if err != nil {
		return Extension{}, err
	}

	// 3) 事务成功后原子替换 Registry（ReplaceThemeContributions：新旧同路径允许）。
	//    失败则回滚 DB 活动主题并恢复旧 Registry。
	if s.pageRegistry != nil {
		if err := s.pageRegistry.RegisterThemePackageReplacing(ctx, active, prevThemeID); err != nil {
			s.pageRegistry.ClearExtension(active.ID)
			if previous != nil && previous.ID != active.ID {
				s.rollbackThemeActivation(ctx, previous, active.ID)
			}
			return Extension{}, fmt.Errorf("%w: registry register failed: %v", ErrBuildFailed, err)
		}
	}

	_, _ = s.store.CreateEvent(ctx, EventInput{
		ExtensionID: active.ID,
		ActorUserID: actor.ID,
		Action:      EventThemeActivated,
		Message:     "Theme activated via runtime page registry (no site rebuild).",
	})
	s.appendAudit(ctx, actor, audit.ActionExtensionActivate, map[string]any{
		"extensionId": active.ID,
		"previousId":  themeIDOrEmpty(previous),
		"queued":      false,
		"runtime":     true,
	})
	return active, nil
}

func themeIDOrEmpty(e *Extension) string {
	if e == nil {
		return ""
	}
	return e.ID
}

// rollbackThemeActivation 激活失败时恢复旧主题 DB + Registry。
func (s *Service) rollbackThemeActivation(ctx context.Context, previous *Extension, failedID string) {
	if previous == nil {
		return
	}
	if _, err := s.store.ActivateTheme(ctx, previous.ID); err != nil {
		// 记录诊断；调用方已返回主错误
		_, _ = s.store.CreateEvent(ctx, EventInput{
			ExtensionID: failedID,
			Action:      EventEnableFailed,
			Message:     "theme activation rollback failed: " + err.Error(),
		})
		return
	}
	if s.pageRegistry != nil {
		s.pageRegistry.ClearExtension(failedID)
		_ = s.pageRegistry.RegisterThemePackage(ctx, *previous)
	}
}

// RestoreActiveThemeRegistry API 启动时恢复活动主题 + 已启用插件的页面贡献。
// 无效/缺失主题时安全回退默认主题并写诊断事件。
func (s *Service) RestoreActiveThemeRegistry(ctx context.Context) error {
	if s == nil || s.pageRegistry == nil {
		return nil
	}
	// 恢复已启用插件页面贡献
	items, err := s.store.List(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Type != TypePlugin || item.Status != StatusEnabled {
			continue
		}
		if err := s.pageRegistry.RegisterPluginPackage(ctx, item); err != nil {
			// 插件页面失败不阻断启动；清掉该扩展贡献并记事件
			s.pageRegistry.ClearExtension(item.ID)
			_, _ = s.store.CreateEvent(ctx, EventInput{
				ExtensionID: item.ID,
				Action:      EventEnableFailed,
				Message:     "restore plugin page contributions failed: " + err.Error(),
			})
		}
	}

	active, err := s.store.ActiveTheme(ctx)
	if err != nil {
		// 无活动主题 → 尝试默认
		if def, derr := s.EnsureDefaultThemeActive(ctx); derr == nil {
			active = def
		} else {
			return derr
		}
	}
	if err := s.pageRegistry.PreflightThemePackage(ctx, active, ""); err != nil {
		// 无效主题 → 回退默认
		_, _ = s.store.CreateEvent(ctx, EventInput{
			ExtensionID: active.ID,
			Action:      EventEnableFailed,
			Message:     "active theme registry restore preflight failed, falling back to default: " + err.Error(),
		})
		s.pageRegistry.ClearExtension(active.ID)
		if active.ID != DefaultThemeID {
			def, derr := s.store.ActivateTheme(ctx, DefaultThemeID)
			if derr != nil {
				return derr
			}
			active = def
		}
	}
	if err := s.pageRegistry.RegisterThemePackage(ctx, active); err != nil {
		return fmt.Errorf("restore active theme registry: %w", err)
	}
	return nil
}

// RestoreSafeModeThemeRegistry 忽略数据库 desired theme 与全部插件贡献，只加载受保护默认主题。
func (s *Service) RestoreSafeModeThemeRegistry(ctx context.Context) error {
	if s == nil || s.pageRegistry == nil {
		return nil
	}
	items, err := s.store.List(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		s.pageRegistry.ClearExtension(item.ID)
	}
	defaultTheme, err := s.store.Get(ctx, DefaultThemeID)
	if err != nil {
		return err
	}
	if defaultTheme.Type != TypeTheme || defaultTheme.Source != SourceBuiltin || !defaultTheme.IsSystem {
		return ErrInvalidManifest
	}
	if err := s.pageRegistry.PreflightThemePackage(ctx, defaultTheme, ""); err != nil {
		return err
	}
	return s.pageRegistry.RegisterThemePackage(ctx, defaultTheme)
}

func (s *Service) EnsureDefaultThemeActive(ctx context.Context) (Extension, error) {
	active, err := s.store.ActiveTheme(ctx)
	if err == nil && active.ID == DefaultThemeID && active.Source == SourceBuiltin {
		return active, nil
	}
	defaultTheme, getErr := s.store.Get(ctx, DefaultThemeID)
	if getErr != nil {
		if err != nil {
			return Extension{}, err
		}
		return Extension{}, getErr
	}
	if defaultTheme.Type != TypeTheme || defaultTheme.Source != SourceBuiltin {
		return Extension{}, ErrInvalidManifest
	}
	return s.store.ActivateTheme(ctx, DefaultThemeID)
}

func (s *Service) verifyExtension(ctx context.Context, extension Extension) error {
	if err := validateInstalledPackage(extension); err != nil {
		if extension.Type == TypeTheme {
			return fmt.Errorf("%w: %v", ErrBuildFailed, err)
		}
		return fmt.Errorf("%w: %v", ErrPreflightFailed, err)
	}
	if extension.Type == TypePlugin && extension.Manifest.Backend.Entry != "" && s.runtime != nil {
		if err := s.runtime.Check(ctx, extension); err != nil {
			return fmt.Errorf("%w: %v", ErrPreflightFailed, err)
		}
	}
	if extension.Type == TypeTheme {
		if !themeRuntimePackagePresent(extension.PackagePath) {
			return fmt.Errorf("%w: theme requires theme.json/assets (L0/L1)", ErrBuildFailed)
		}
	}
	return nil
}

// themeRuntimePackagePresent 检测 L0/L1 运行时主题包。
func themeRuntimePackagePresent(packagePath string) bool {
	root := strings.TrimSpace(packagePath)
	if root == "" {
		return false
	}
	// 与 PackageContentRoot 一致：zip 旁 files/、同级目录或内容寻址目录。
	candidates := []string{root}
	if st, err := os.Stat(root); err == nil && !st.IsDir() {
		candidates = []string{
			filepath.Join(filepath.Dir(root), "files"),
			filepath.Dir(root),
		}
	}
	for _, base := range candidates {
		for _, name := range []string{"theme.json", "assets/theme.css", "assets"} {
			if st, err := os.Stat(filepath.Join(base, name)); err == nil {
				if name == "assets" && !st.IsDir() {
					continue
				}
				return true
			}
		}
	}
	return false
}

func validateInstalledPackage(extension Extension) error {
	packagePath := strings.TrimSpace(extension.PackagePath)
	if packagePath == "" {
		return fmt.Errorf("extension package path is empty")
	}
	if strings.TrimSpace(extension.PackageDigest) != "" {
		info, err := os.Lstat(packagePath)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("extension package snapshot %s is not available", packagePath)
		}
		if err := requireInstalledManifest(packagePath); err != nil {
			return err
		}
		digest, err := extensionpackage.DigestTree(packagePath)
		if err != nil {
			return fmt.Errorf("extension package snapshot %s is invalid: %w", packagePath, err)
		}
		if digest != extension.PackageDigest {
			return fmt.Errorf("extension package snapshot %s digest does not match its installed version", packagePath)
		}
		return nil
	}
	if extension.Source == SourceBuiltin {
		info, err := os.Stat(packagePath)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("builtin extension package %s is not available", packagePath)
		}
		return requireInstalledManifest(packagePath)
	}

	info, err := os.Stat(packagePath)
	if err != nil || info.IsDir() {
		return fmt.Errorf("extension archive %s is not available", packagePath)
	}
	return requireInstalledManifest(filepath.Dir(packagePath))
}

func requireInstalledManifest(root string) error {
	manifestPath := filepath.Join(root, ManifestFileName)
	info, err := os.Stat(manifestPath)
	if err != nil || info.IsDir() {
		return fmt.Errorf("extension manifest %s is not available", manifestPath)
	}
	return nil
}

func (s *Service) recordEnableFailure(ctx context.Context, actor identity.Actor, extensionID string, cause error) {
	_, _ = s.store.CreateEvent(ctx, EventInput{
		ExtensionID: extensionID,
		ActorUserID: actor.ID,
		Action:      EventEnableFailed,
		Message:     cause.Error(),
	})
}

func (s *Service) decorateRuntime(ctx context.Context, item Extension) Extension {
	if item.Type == TypePlugin {
		item.CapabilityGrants = extensionmanifest.CapabilityGrants(item.Manifest)
		if s.runtime != nil {
			status := s.runtime.Status(ctx, item)
			item.Runtime = &status
		}
	}
	return item
}

// CapabilitiesFor 返回已启用插件的有效能力集合（Host API CapabilitySource）。
func (s *Service) CapabilitiesFor(ctx context.Context, extensionID string) (capabilities.Set, error) {
	if s.safeMode {
		return nil, ErrExtensionDisabled
	}
	extension, err := s.store.Get(ctx, normalizeID(extensionID))
	if err != nil {
		return nil, err
	}
	if extension.Type != TypePlugin {
		return nil, ErrExtensionNotFound
	}
	if extension.Status != StatusEnabled {
		return nil, ErrExtensionDisabled
	}
	keys, _ := extensionmanifest.ResolvedCapabilities(extension.Manifest)
	return capabilities.NewSet(keys), nil
}

// DeclaredJobKinds 返回插件 manifest 声明的 job names。
func (s *Service) DeclaredJobKinds(ctx context.Context, extensionID string) ([]string, error) {
	if s.safeMode {
		return []string{}, nil
	}
	extension, err := s.store.Get(ctx, normalizeID(extensionID))
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(extension.Manifest.Jobs))
	for _, job := range extension.Manifest.Jobs {
		if name := strings.TrimSpace(job.Name); name != "" {
			out = append(out, name)
		}
	}
	return out, nil
}

// CapabilityCatalog 返回宿主能力目录（管理端审查文案）。
func (s *Service) CapabilityCatalog(_ context.Context, actor identity.Actor) ([]capabilities.Definition, error) {
	if !canViewExtensions(actor) && !canManagePlugins(actor) {
		return nil, identity.ErrPermissionDenied
	}
	return capabilities.Catalog(), nil
}

type archiveFile struct {
	name string
	mode os.FileMode
	body []byte
}

func readArchive(data []byte) (Manifest, []archiveFile, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return Manifest{}, nil, ErrInvalidArchive
	}

	var rootBody []byte
	files := []archiveFile{}
	fileMap := extensionmanifest.FileMapFS{}
	seen := map[string]struct{}{}
	var total uint64
	for _, file := range reader.File {
		name, ok := safeArchivePath(file.Name)
		if !ok {
			return Manifest{}, nil, ErrInvalidArchive
		}
		if file.Mode()&os.ModeSymlink != 0 {
			return Manifest{}, nil, ErrInvalidArchive
		}
		if file.FileInfo().IsDir() {
			continue
		}
		if _, duplicate := seen[name]; duplicate {
			return Manifest{}, nil, ErrInvalidArchive
		}
		seen[name] = struct{}{}
		// 不信任 UncompressedSize64：zip bomb 可虚报小体积。按真实读出字节累计并硬顶。
		remaining := int64(maxArchiveBytes) - int64(total)
		if remaining <= 0 {
			return Manifest{}, nil, ErrInvalidArchive
		}
		body, err := readZipFileLimited(file, remaining)
		if err != nil {
			return Manifest{}, nil, ErrInvalidArchive
		}
		total += uint64(len(body))
		if total > maxArchiveBytes {
			return Manifest{}, nil, ErrInvalidArchive
		}
		fileMap[name] = body
		if name == ManifestFileName {
			rootBody = body
			continue
		}
		files = append(files, archiveFile{name: name, mode: file.Mode(), body: body})
	}
	if rootBody == nil {
		return Manifest{}, nil, ErrInvalidArchive
	}
	// 合并 includes partials 后再交给校验与快照；files 仍保留除入口外的原文。
	manifest, err := extensionmanifest.LoadRootBytes(rootBody, fileMap)
	if err != nil {
		return Manifest{}, nil, ErrInvalidManifest
	}
	return manifest, files, nil
}

// readZipFileLimited 读取 zip 条目，最多 maxBytes 字节；超出视为炸弹/恶意包。
func readZipFileLimited(file *zip.File, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return nil, ErrInvalidArchive
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	// +1 探测是否超过上限，避免无界 ReadAll。
	limited := io.LimitReader(reader, maxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, ErrInvalidArchive
	}
	return body, nil
}

func writeManifest(versionDir string, manifest Manifest) error {
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(versionDir, ManifestFileName), body, 0o600)
}

// PackageContentRoot 返回扩展包内可读文件的目录根。
// - 内容寻址快照 / builtin：PackagePath 本身就是目录。
// - 旧版上传包：PackagePath 指向 package.zip，解压内容在同级 files/。
// 主题 L0/L1 预检、皮肤资源与模板加载都必须走这里，不能直接拼 package.zip 路径。
func PackageContentRoot(extension Extension) string {
	path := strings.TrimSpace(extension.PackagePath)
	if path == "" {
		return ""
	}
	path = filepath.Clean(path)
	// 有 digest 的快照或内置包：PackagePath 即为内容根目录。
	if strings.TrimSpace(extension.PackageDigest) != "" || extension.Source == SourceBuiltin {
		return path
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return path
	}
	// 旧上传布局：.../1.0.0/package.zip + .../1.0.0/files/*
	files := filepath.Join(filepath.Dir(path), "files")
	if info, err := os.Stat(files); err == nil && info.IsDir() {
		return files
	}
	// 兜底：zip 同级目录（manifest 等同级时）
	return filepath.Dir(path)
}

func installedFilePath(extension Extension, manifestPath string) (string, bool) {
	name, ok := safeArchivePath(manifestPath)
	if !ok {
		return "", false
	}
	root := PackageContentRoot(extension)
	if root == "" {
		return "", false
	}
	target := filepath.Join(root, filepath.FromSlash(name))
	return target, strings.HasPrefix(target, root+string(os.PathSeparator))
}

func InstalledFilePathForRuntime(extension Extension, manifestPath string) (string, bool) {
	return installedFilePath(extension, manifestPath)
}

func validateManifest(manifest Manifest) error {
	return extensionmanifest.Validate(manifest)
}

func ValidateManifest(manifest Manifest) error {
	return extensionmanifest.Validate(manifest)
}

func normalizeManifest(manifest Manifest) Manifest {
	return extensionmanifest.Normalize(manifest)
}

func normalizeID(id string) string {
	return extensionmanifest.NormalizeID(id)
}

func safeArchivePath(name string) (string, bool) {
	return extensionmanifest.SafeArchivePath(name)
}

func normalizeRoutePath(value string) string {
	return extensionmanifest.NormalizeRoutePath(value)
}

func extensionInjectsAdminNavigation(extension Extension) bool {
	if extension.Type == TypePlugin {
		return extension.Status == StatusEnabled
	}
	return extension.Type == TypeTheme && extension.Status == StatusEnabled
}

func normalizedAdminPages(manifest Manifest) []ManifestAdminPage {
	pages := make([]ManifestAdminPage, 0, len(extensionmanifest.EffectiveAdminPages(manifest))+1)
	pages = append(pages, ManifestAdminPage{
		Path:        "/about",
		Label:       manifest.Name,
		Description: manifest.Description,
		Icon:        defaultExtensionIcon(manifest.Type),
		View:        "about",
		Order:       0,
	})
	for _, page := range extensionmanifest.EffectiveAdminPages(manifest) {
		if strings.TrimSpace(page.Path) == "" {
			continue
		}
		pages = append(pages, normalizeAdminPageForDisplay(manifest.Type, page))
	}
	sort.SliceStable(pages, func(left, right int) bool {
		if pages[left].Order == pages[right].Order {
			return pages[left].Path < pages[right].Path
		}
		return pages[left].Order < pages[right].Order
	})
	return pages
}

func normalizedMenuAdminPages(manifest Manifest) []ManifestAdminPage {
	pages := make([]ManifestAdminPage, 0, len(extensionmanifest.MenuAdminPages(manifest)))
	for _, page := range extensionmanifest.MenuAdminPages(manifest) {
		if strings.TrimSpace(page.Path) == "" {
			continue
		}
		pages = append(pages, normalizeAdminPageForDisplay(manifest.Type, page))
	}
	sort.SliceStable(pages, func(left, right int) bool {
		if pages[left].Order == pages[right].Order {
			return pages[left].Path < pages[right].Path
		}
		return pages[left].Order < pages[right].Order
	})
	return pages
}

func normalizeAdminPageForDisplay(extensionType string, page ManifestAdminPage) ManifestAdminPage {
	page.Path = extensionmanifest.NormalizeRoutePath(page.Path)
	page.Label = strings.TrimSpace(page.Label)
	page.Description = strings.TrimSpace(page.Description)
	page.Icon = strings.TrimSpace(page.Icon)
	page.View = strings.TrimSpace(page.View)
	page.Permission = strings.TrimSpace(page.Permission)
	if page.Icon == "" {
		page.Icon = defaultExtensionIcon(extensionType)
	}
	if page.View == "" {
		page.View = "about"
	}
	return page
}

func defaultExtensionIcon(extensionType string) string {
	if extensionType == TypeTheme {
		return "i-lucide-palette"
	}
	return "i-lucide-plug"
}

func resolveExtensionSettings(extension Extension, values map[string]string, locale string) ExtensionSettings {
	items := make([]ExtensionSettingValue, 0, len(extension.Manifest.Settings))
	for _, setting := range extension.Manifest.Settings {
		value := setting.Default
		secretSet := false
		if values != nil {
			if stored, ok := values[setting.Key]; ok {
				value = stored
				secretSet = setting.Type == "secret" && stored != ""
			}
		}
		if setting.Type == "secret" {
			value = ""
		}
		// API 响应始终返回当前 locale 下的纯字符串，避免前端处理 locale map。
		presentation := extensionmanifest.ResolveSettingPresentation(setting, locale)
		options := make([]ExtensionSettingOption, 0, len(presentation.Options))
		for _, option := range presentation.Options {
			options = append(options, ExtensionSettingOption{
				Value:       option.Value,
				Label:       option.Label,
				Description: option.Description,
			})
		}
		items = append(items, ExtensionSettingValue{
			Key:              setting.Key,
			Label:            presentation.Label,
			Description:      presentation.Description,
			Type:             setting.Type,
			Default:          setting.Default,
			Value:            value,
			Placeholder:      presentation.Placeholder,
			RecommendedValue: setting.RecommendedValue,
			Width:            setting.Width,
			Group:            presentation.Group,
			GroupID:          setting.GroupID,
			Column:           setting.Column,
			Options:          options,
			SecretSet:        secretSet,
		})
	}
	document := extension.Manifest.SettingsDocument
	renderer := ExtensionSettingsRenderer{Mode: document.UI.Mode, Layout: document.UI.Layout, Source: "document", Fallback: "schema"}
	if !document.Explicit {
		renderer.Source = "legacy_array"
	}
	if document.UI.Component != nil {
		component := document.UI.Component
		renderer.Component = &ExtensionSettingsComponent{ID: component.ID, Kind: "prebuilt", APIVersion: component.APIVersion, Entry: component.Entry, CSS: component.CSS}
	}
	tabs := make([]ExtensionSettingsTab, 0, len(document.UI.Tabs))
	for _, tab := range document.UI.Tabs {
		tabs = append(tabs, ExtensionSettingsTab{ID: tab.ID, Label: tab.Label.Resolve(locale), Description: tab.Description.Resolve(locale), Groups: append([]string(nil), tab.Groups...)})
	}
	groups := make([]ExtensionSettingsGroup, 0, len(document.UI.Groups))
	for _, group := range document.UI.Groups {
		groups = append(groups, ExtensionSettingsGroup{ID: group.ID, Label: group.Label.Resolve(locale), Description: group.Description.Resolve(locale), Columns: group.Columns})
	}
	callouts := make([]ExtensionSettingsCallout, 0, len(document.UI.Callouts))
	for _, callout := range document.UI.Callouts {
		callouts = append(callouts, ExtensionSettingsCallout{ID: callout.ID, Tone: callout.Tone, Title: callout.Title.Resolve(locale), Body: callout.Body.Resolve(locale), Tab: callout.Tab, Group: callout.Group})
	}
	actions := make([]ExtensionSettingsAction, 0, len(document.Actions))
	for _, action := range document.Actions {
		available := extension.Type == TypePlugin && extension.Manifest.Backend.Entry != "" && len(extension.Manifest.Providers) > 0
		reason := ""
		if !available {
			reason = "extension.settings_action_unavailable"
		}
		actions = append(actions, ExtensionSettingsAction{
			ID: action.ID, Kind: action.Kind, Label: action.Label.Resolve(locale), Description: action.Description.Resolve(locale),
			Placement: action.Placement, UseDraftValues: action.UseDraftValues, Fields: append([]string(nil), action.Fields...),
			Available: available, UnavailableReason: reason,
		})
	}
	return ExtensionSettings{
		ExtensionID: extension.ID, ExtensionType: extension.Type, ExtensionVersion: extension.Version, ExtensionStatus: extension.Status,
		Renderer: renderer, Tabs: tabs, Groups: groups, Callouts: callouts, Items: items, Actions: actions,
	}
}

// sanitizeSettingValues 将 PUT 解析为完整候选集：提交值 → 已存值 → 默认。
// 省略的 secret 始终保留已存值；未知 key 拒绝且不写库。
func sanitizeSettingValues(manifest Manifest, input, current map[string]string) (map[string]string, error) {
	allowed := map[string]ManifestSetting{}
	for _, setting := range manifest.Settings {
		allowed[setting.Key] = setting
	}
	// 先拒绝未知键，避免部分写入。
	for key := range input {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := allowed[key]; !ok {
			return nil, ErrInvalidManifest
		}
	}
	values := map[string]string{}
	for _, setting := range manifest.Settings {
		key := setting.Key
		if submitted, ok := input[key]; ok {
			normalized := strings.TrimSpace(submitted)
			if setting.Type == "secret" && normalized == "" {
				if cur, has := current[key]; has {
					values[key] = cur
				}
				// 无已存 secret 且提交空：不写入覆盖行。
				continue
			}
			values[key] = normalized
			continue
		}
		// 未提交：保留已存；无已存则不写入（运行时用 default）。
		if cur, has := current[key]; has {
			values[key] = cur
		}
	}
	return values, nil
}

// listDecryptedSettings 读取并解密 secret；错误密文 fail closed（不交给插件/API 明文路径）。
// 历史明文在 cipher 启用时异步迁移写回密文。
func (s *Service) listDecryptedSettings(ctx context.Context, extension Extension) (map[string]string, error) {
	raw, err := s.store.ListSettings(ctx, extension.ID)
	if err != nil {
		return nil, err
	}
	return s.decryptSettingsMap(ctx, extension, raw)
}

func (s *Service) decryptSettingsMap(ctx context.Context, extension Extension, raw map[string]string) (map[string]string, error) {
	if raw == nil {
		return map[string]string{}, nil
	}
	secretKeys := secretSettingKeys(extension.Manifest)
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		if !secretKeys[key] {
			out[key] = value
			continue
		}
		plain, migrated, err := s.decryptSecretValue(value)
		if err != nil {
			// 错误密钥/损坏密文：禁止静默清空，也禁止把密文交给插件。
			return nil, fmt.Errorf("%w: setting %s", err, key)
		}
		out[key] = plain
		if migrated {
			enc, encErr := s.encryptSecretValue(plain)
			if encErr != nil {
				return nil, fmt.Errorf("encrypt legacy secret setting %s: %w", key, encErr)
			}
			if _, casErr := s.store.CompareAndSwapSetting(ctx, extension.ID, key, value, enc); casErr != nil {
				return nil, fmt.Errorf("migrate legacy secret setting %s: %w", key, casErr)
			}
		}
	}
	return out, nil
}

func secretSettingKeys(manifest Manifest) map[string]bool {
	out := map[string]bool{}
	for _, setting := range manifest.Settings {
		if setting.Type == "secret" {
			out[setting.Key] = true
		}
	}
	return out
}

func (s *Service) encryptSecretSettings(manifest Manifest, values map[string]string) (map[string]string, error) {
	if values == nil {
		return map[string]string{}, nil
	}
	secretKeys := secretSettingKeys(manifest)
	out := make(map[string]string, len(values))
	for key, value := range values {
		if secretKeys[key] && value != "" {
			enc, err := s.encryptSecretValue(value)
			if err != nil {
				return nil, err
			}
			out[key] = enc
			continue
		}
		out[key] = value
	}
	return out, nil
}

func (s *Service) encryptSecretValue(plaintext string) (string, error) {
	if s == nil || s.cipher == nil {
		return plaintext, nil
	}
	return s.cipher.Encrypt(plaintext)
}

// decryptSecretValue 返回明文；migrated=true 表示存储仍是历史明文且 cipher 已启用，应回写。
func (s *Service) decryptSecretValue(stored string) (plain string, migrated bool, err error) {
	if stored == "" {
		return "", false, nil
	}
	if s == nil || s.cipher == nil || !s.cipher.Enabled() {
		// 透明模式：密文前缀也无法解密，fail closed。
		if crypto.IsEncrypted(stored) {
			return "", false, fmt.Errorf("extensions: encrypted secret requires option encryption key")
		}
		return stored, false, nil
	}
	if !crypto.IsEncrypted(stored) {
		// 历史明文：可读，并标记迁移。
		return stored, true, nil
	}
	plain, err = s.cipher.Decrypt(stored)
	if err != nil {
		return "", false, fmt.Errorf("extensions: secret decrypt failed: %w", err)
	}
	return plain, false, nil
}

// ListSettingsForRuntime 供插件子进程注入：返回解密后的设置；解密失败则错误。
func (s *Service) ListSettingsForRuntime(ctx context.Context, extensionID string) (map[string]string, error) {
	extension, err := s.store.Get(ctx, normalizeID(extensionID))
	if err != nil {
		return nil, err
	}
	return s.listDecryptedSettings(ctx, extension)
}

// ListSettings 实现插件 ProtocolStarter 与 Host API 共用的只读设置接口。
// 所有运行时读取必须经过这里，禁止直接把 Store 注入插件边界。
func (s *Service) ListSettings(ctx context.Context, extensionID string) (map[string]string, error) {
	return s.ListSettingsForRuntime(ctx, extensionID)
}

func manifestEvents(manifest Manifest) []ManifestEvent {
	return extensionmanifest.DeclaredEvents(manifest)
}

func DeclaredManifestEvents(manifest Manifest) []ManifestEvent {
	return extensionmanifest.DeclaredEvents(manifest)
}

func errorsIsNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}
