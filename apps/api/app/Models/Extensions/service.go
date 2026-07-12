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
	themeruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeRuntime"
)

const maxArchiveBytes = 50 * 1024 * 1024

type Service struct {
	store                     Store
	extensionRoot             string
	builtinRoot               string
	runtime                   RuntimeManager
	themeBuilder              ThemeBuilder
	themeActivationDispatcher ThemeActivationDispatcher
	themeCurrentWriter        ThemeCurrentWriter
	frontendLifecycle         ExtensionFrontendLifecycle
	webReleaseLifecycle       FrontendReleaseManager
	// webReleaseProgress 可选：把插件相关的 live/failed Web Release 挂到列表项上。
	webReleaseProgress WebReleaseProgressReader
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
}

// FeatureFlagSource 返回 requiresFeatures 中当前关闭的 key。
type FeatureFlagSource interface {
	MissingRequiredFeatures(ctx context.Context, required []string) ([]string, error)
}

// WebReleaseProgressReader 读取扩展相关的进行中/失败 Web 发布，用于管理端进度条。
type WebReleaseProgressReader interface {
	LatestProgressWebReleaseForExtension(context.Context, string) (WebRelease, error)
}

// ServiceOption 用于在保留现有构造函数签名的同时注入可选依赖。
// 目前仅用于给 API 进程（同步恢复默认主题路径）注入 ThemeCurrentWriter。
type ServiceOption func(*Service)

// WithThemeCurrentWriter 让 ActivateTheme 在恢复内置默认主题（同步路径）
// 时写入 current.json 的 default 状态，供前端运行时感知"切回默认"。
func WithThemeCurrentWriter(writer ThemeCurrentWriter) ServiceOption {
	return func(s *Service) {
		s.themeCurrentWriter = writer
	}
}

// WithWebReleaseLifecycle 将会改变完整 Web composition 的扩展操作交给统一发布流水线。
func WithWebReleaseLifecycle(frontend ExtensionFrontendLifecycle, releases FrontendReleaseManager) ServiceOption {
	return func(s *Service) {
		s.frontendLifecycle = frontend
		s.webReleaseLifecycle = releases
	}
}

// WithWebReleaseProgress 注入插件列表用的 Web Release 进度读取器。
func WithWebReleaseProgress(reader WebReleaseProgressReader) ServiceOption {
	return func(s *Service) {
		s.webReleaseProgress = reader
	}
}

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

// WithCipher 注入与 web_options 相同的 AES-GCM 加密器（secret 设置静态加密）。
func WithCipher(c *crypto.OptionCipher) ServiceOption {
	return func(s *Service) {
		s.cipher = c
	}
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
	return NewServiceWithHooks(store, extensionRoot, nil, nil)
}

func NewServiceWithHooks(store Store, extensionRoot string, runtime RuntimePreflight, themeBuilder ThemeBuilder) *Service {
	return NewServiceWithBuiltinsAndHooks(store, extensionRoot, "", runtime, themeBuilder)
}

func NewServiceWithBuiltins(store Store, extensionRoot string, builtinRoot string) *Service {
	return NewServiceWithBuiltinsAndRuntime(store, extensionRoot, builtinRoot, nil, nil)
}

func NewServiceWithBuiltinsAndHooks(store Store, extensionRoot string, builtinRoot string, runtime RuntimePreflight, themeBuilder ThemeBuilder) *Service {
	var manager RuntimeManager
	if runtime != nil {
		if full, ok := runtime.(RuntimeManager); ok {
			manager = full
		} else {
			manager = preflightRuntimeManager{RuntimePreflight: runtime}
		}
	}
	return NewServiceWithBuiltinsAndRuntime(store, extensionRoot, builtinRoot, manager, themeBuilder)
}

func NewServiceWithRuntime(store Store, extensionRoot string, runtime RuntimeManager, themeBuilder ThemeBuilder) *Service {
	return NewServiceWithBuiltinsAndRuntime(store, extensionRoot, "", runtime, themeBuilder)
}

func NewServiceWithBuiltinsAndRuntime(store Store, extensionRoot string, builtinRoot string, runtime RuntimeManager, themeBuilder ThemeBuilder) *Service {
	if strings.TrimSpace(extensionRoot) == "" {
		extensionRoot = "storage/extensions"
	}
	if runtime == nil {
		runtime = LocalRuntimeManager{}
	}
	if themeBuilder == nil {
		themeBuilder = LocalThemeBuilder{}
	}
	return &Service{
		store:         store,
		extensionRoot: extensionRoot,
		builtinRoot:   strings.TrimSpace(builtinRoot),
		runtime:       runtime,
		themeBuilder:  themeBuilder,
	}
}

func NewServiceWithThemeActivation(store Store, extensionRoot string, builtinRoot string, runtime RuntimeManager, themeBuilder ThemeBuilder, dispatcher ThemeActivationDispatcher) *Service {
	service := NewServiceWithBuiltinsAndRuntime(store, extensionRoot, builtinRoot, runtime, themeBuilder)
	service.themeActivationDispatcher = dispatcher
	return service
}

// NewServiceWithThemeActivationWithOptions 在 NewServiceWithThemeActivation 基础上
// 注入可选依赖（如 ThemeCurrentWriter），保持原有调用方签名不变。
func NewServiceWithThemeActivationWithOptions(store Store, extensionRoot string, builtinRoot string, runtime RuntimeManager, themeBuilder ThemeBuilder, dispatcher ThemeActivationDispatcher, options ...ServiceOption) *Service {
	service := NewServiceWithThemeActivation(store, extensionRoot, builtinRoot, runtime, themeBuilder, dispatcher)
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

type LocalThemeBuilder struct{}

func (LocalThemeBuilder) Build(_ context.Context, extension Extension) error {
	if extension.Manifest.Frontend.Layer == "" {
		return nil
	}
	layer, ok := installedFilePath(extension, extension.Manifest.Frontend.Layer)
	if !ok {
		return ErrInvalidManifest
	}
	info, err := os.Stat(layer)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("nuxt layer %s is not available", extension.Manifest.Frontend.Layer)
	}
	return nil
}


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

// canManageReleases Web Release 队列与发布。
func canManageReleases(actor identity.Actor) bool {
	return actor.Can(identity.PermissionExtensionReleaseManage) || actor.Can(identity.PermissionExtensionManage)
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
			item, err := s.store.SaveBuiltin(ctx, SaveBuiltinInput{
				Manifest:      manifest,
				PackagePath:   snapshot.Root,
				PackageDigest: snapshot.Digest,
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
	// 禁用插件不得再暴露/改写运行时配置（页面与 provider 能力一并下线）。
	if err := requireExtensionEnabledForSettings(extension); err != nil {
		return ExtensionSettings{}, err
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
	if err := requireExtensionEnabledForSettings(extension); err != nil {
		return ExtensionSettings{}, err
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
	previousRaw, _ := s.store.ListSettings(ctx, extension.ID)
	if err := s.store.ReplaceSettings(ctx, extension.ID, stored); err != nil {
		return ExtensionSettings{}, err
	}
	if err := s.restartPluginForSettings(ctx, extension); err != nil {
		// 插件重启失败：回滚设置，避免“已写库但进程仍用旧配置/失败态”不一致。
		_ = s.store.ReplaceSettings(ctx, extension.ID, previousRaw)
		return ExtensionSettings{}, err
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
	if err := requireExtensionEnabledForSettings(extension); err != nil {
		return ExtensionSettings{}, err
	}
	if err := s.store.ResetSettings(ctx, extension.ID); err != nil {
		return ExtensionSettings{}, err
	}
	if err := s.restartPluginForSettings(ctx, extension); err != nil {
		return ExtensionSettings{}, err
	}
	return resolveExtensionSettings(extension, map[string]string{}, locale), nil
}

// requireExtensionEnabledForSettings 限制仅启用中的扩展可读写 settings。
// 主题未激活 / 插件已禁用时配置页应不可用，避免“禁了但功能还在”。
func requireExtensionEnabledForSettings(extension Extension) error {
	if extension.Status == StatusEnabled {
		return nil
	}
	return ErrExtensionDisabled
}

func canManageExtensionSettings(actor identity.Actor, extension Extension) bool {
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

func (s *Service) restartPluginForSettings(ctx context.Context, extension Extension) error {
	if extension.Type != TypePlugin || extension.Status != StatusEnabled || extension.Manifest.Backend.Entry == "" || s.runtime == nil {
		return nil
	}
	if err := s.runtime.Stop(ctx, extension); err != nil {
		return err
	}
	return s.runtime.Start(ctx, extension)
}

func (s *Service) MatchRoute(ctx context.Context, extensionID string, method string, routePath string) (MatchedRoute, error) {
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
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return Extension{}, err
	}
	if extension.Type == TypeTheme {
		return Extension{}, ErrThemeActivationRequired
	}
	// 非内置后端插件：启动子进程 = 主机代码执行，仅 super_admin。
	if err := requireSuperAdminForUntrustedBackend(actor, extension.Source, extension.Manifest); err != nil {
		s.denyUntrustedBackend(ctx, actor, extension.ID, "enable")
		return Extension{}, err
	}

	// 首次启用（非 restart）且存在 Host 能力时，要求运营显式确认（F2.1）。
	if extension.Status != StatusEnabled {
		capKeys, _ := extensionmanifest.ResolvedCapabilities(extension.Manifest)
		if capabilities.RequiresConfirmation(capKeys) && !input.ConfirmCapabilities {
			return Extension{}, ErrCapabilityConfirmationRequired
		}
	}

	// F4.5：manifest requiresFeatures 必须全部开启。
	if err := s.ensureRequiredFeatures(ctx, extension.Manifest.RequiresFeatures); err != nil {
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
		if err := s.runtime.Start(ctx, enabled); err != nil {
			_, _ = s.store.Disable(ctx, enabled.ID)
			s.recordEnableFailure(ctx, actor, enabled.ID, err)
			return Extension{}, fmt.Errorf("%w: %v", ErrRuntimeFailed, err)
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
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return Extension{}, err
	}
	if extension.Type != TypeTheme {
		return Extension{}, ErrThemeActivationRequired
	}
	if extension.ID != DefaultThemeID || extension.Source != SourceBuiltin {
		if err := s.verifyExtension(ctx, extension); err != nil {
			s.recordEnableFailure(ctx, actor, extension.ID, err)
			return Extension{}, err
		}
		layerPath, ok := installedFilePath(extension, extension.Manifest.Frontend.Layer)
		if !ok {
			return Extension{}, fmt.Errorf("%w: theme layer is unavailable", ErrBuildFailed)
		}
		release, err := s.store.CreateThemeRelease(ctx, ThemeReleaseInput{
			ExtensionID: extension.ID,
			Version:     extension.Version,
			LayerPath:   layerPath,
		})
		if err != nil {
			return Extension{}, err
		}
		if s.themeActivationDispatcher != nil {
			if err := s.themeActivationDispatcher.EnqueueThemeActivation(ctx, release); err != nil {
				_, _ = s.store.UpdateThemeRelease(ctx, ThemeReleaseUpdate{
					ID:      release.ID,
					Status:  ThemeReleaseFailed,
					Message: err.Error(),
				})
				return Extension{}, err
			}
		}
		_, _ = s.store.CreateEvent(ctx, EventInput{
			ExtensionID: extension.ID,
			ActorUserID: actor.ID,
			Action:      EventThemeActivationQueued,
			Message:     "Theme activation queued.",
		})
		s.appendAudit(ctx, actor, audit.ActionExtensionActivate, map[string]any{
			"extensionId": extension.ID,
			"queued":      true,
			"releaseId":   release.ID,
		})
		extension.ThemeRelease = &release
		return extension, nil
	}
	if err := s.verifyExtension(ctx, extension); err != nil {
		s.recordEnableFailure(ctx, actor, extension.ID, err)
		return Extension{}, err
	}
	active, err := s.store.ActivateTheme(ctx, extension.ID)
	if err != nil {
		return Extension{}, err
	}
	// 切回默认主题前，把上一个上传主题遗留的 active release 置为 rolled_back，
	// 否则前端会继续把它当作"当前主题"显示 100% 进度，与实际渲染的默认主题不一致。
	if err := s.rollBackActiveThemeRelease(ctx); err != nil {
		return Extension{}, err
	}
	// 恢复默认主题是同步路径，没有 worker 会再写 current.json。
	// 这里主动写一个 default 状态，让前端运行时（runtime.mjs / dev supervisor）
	// 能感知到"切回默认主题"，而不是继续显示上一个上传主题。
	if s.themeCurrentWriter != nil {
		if err := s.themeCurrentWriter.WriteCurrent(ctx, themeruntime.CurrentRelease{
			ExtensionID: DefaultThemeID,
			Mode:        themeruntime.CurrentModeDefault,
		}); err != nil {
			return Extension{}, err
		}
	}
	_, _ = s.store.CreateEvent(ctx, EventInput{
		ExtensionID: active.ID,
		ActorUserID: actor.ID,
		Action:      EventThemeActivated,
		Message:     "Theme activated.",
	})
	s.appendAudit(ctx, actor, audit.ActionExtensionActivate, map[string]any{
		"extensionId": active.ID,
		"queued":      false,
	})
	return active, nil
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

// rollBackActiveThemeRelease 把当前 active 的上传主题 release 置为 rolled_back。
// 调用方是"切回默认主题"的同步路径：worker 不会再走一次 release 状态机，
// 所以这里要主动清理遗留的 active release，避免前端继续显示旧的"当前主题"。
// 没有 active release（比如首次激活默认主题）时静默返回 nil。
func (s *Service) rollBackActiveThemeRelease(ctx context.Context) error {
	current, err := s.store.ActiveThemeRelease(ctx)
	if err != nil {
		if errors.Is(err, ErrExtensionNotFound) {
			return nil
		}
		return err
	}
	_, err = s.store.UpdateThemeRelease(ctx, ThemeReleaseUpdate{
		ID:      current.ID,
		Status:  ThemeReleaseRolledBack,
		Message: "Rolled back because the default theme was activated.",
	})
	return err
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
		if strings.TrimSpace(extension.Manifest.Frontend.Layer) == "" {
			return fmt.Errorf("%w: theme frontend layer is empty", ErrBuildFailed)
		}
		if s.themeBuilder != nil {
			err := s.themeBuilder.Build(ctx, extension)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrBuildFailed, err)
			}
		}
	}
	return nil
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
	if item.Type == TypeTheme {
		if release, err := s.store.LatestThemeRelease(ctx, item.ID); err == nil {
			item.ThemeRelease = &release
		}
	}
	// 带管理端前端的插件启停会排队 Web Release；挂到列表供进度条轮询。
	if item.Type == TypePlugin && s.webReleaseProgress != nil {
		if release, err := s.webReleaseProgress.LatestProgressWebReleaseForExtension(ctx, item.ID); err == nil {
			item.WebRelease = webReleaseSummary(release)
		}
	}
	return item
}

// CapabilitiesFor 返回已启用插件的有效能力集合（Host API CapabilitySource）。
func (s *Service) CapabilitiesFor(ctx context.Context, extensionID string) (capabilities.Set, error) {
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

func installedFilePath(extension Extension, manifestPath string) (string, bool) {
	name, ok := safeArchivePath(manifestPath)
	if !ok {
		return "", false
	}
	root := filepath.Clean(extension.PackagePath)
	if strings.TrimSpace(extension.PackageDigest) == "" && extension.Source != SourceBuiltin {
		// 仅旧版上传包使用 package.zip 旁边的 files 目录；内容寻址快照直接以 PackagePath 为根。
		root = filepath.Join(filepath.Dir(extension.PackagePath), "files")
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
			Group:            presentation.Group,
			Options:          options,
			SecretSet:        secretSet,
		})
	}
	return ExtensionSettings{ExtensionID: extension.ID, Items: items}
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
	migrate := map[string]string{}
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
			if enc, encErr := s.encryptSecretValue(plain); encErr == nil {
				migrate[key] = enc
			}
		}
	}
	if len(migrate) > 0 {
		// 事务性写回：以当前 raw 为底，只更新需迁移的 secret。
		next := make(map[string]string, len(raw))
		for k, v := range raw {
			next[k] = v
		}
		for k, v := range migrate {
			next[k] = v
		}
		_ = s.store.ReplaceSettings(ctx, extension.ID, next)
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

func manifestEvents(manifest Manifest) []ManifestEvent {
	return extensionmanifest.DeclaredEvents(manifest)
}

func DeclaredManifestEvents(manifest Manifest) []ManifestEvent {
	return extensionmanifest.DeclaredEvents(manifest)
}

func errorsIsNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}
