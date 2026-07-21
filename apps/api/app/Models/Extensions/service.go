package extensions

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	assetregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/AssetRegistry"
	audit "github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	crypto "github.com/zhuchunshu/sforum/apps/api/app/Support/Crypto"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

const (
	maxArchiveBytes               = 50 * 1024 * 1024
	maxArchiveEntries             = 4096
	themeTrustCompensationTimeout = 5 * time.Second
	settingsCompensationTimeout   = 5 * time.Second
)

type Service struct {
	themeActivationMu       sync.Mutex
	themeRuntimeUnavailable bool
	assetPublicationMu      sync.Mutex
	store                   Store
	extensionRoot           string
	builtinRoot             string
	runtime                 RuntimeManager
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
	// routeProviderSelections 在进程/扩展身份清理前撤销精确 replace 选择。
	routeProviderSelections RouteProviderSelectionInvalidator
	// providerSlotSelections 撤销 contract owner/candidate 的精确服务选择。
	providerSlotSelections ProviderSlotSelectionInvalidator
	// pageRegistry 运行时主题页面贡献（L0/L1）；nil 时跳过注册。
	pageRegistry      PageRegistry
	componentRegistry ComponentRegistry
	// assetRegistry 与公开 L2 / 生命周期共享的 Host Asset Registry。
	assetRegistry   *assetregistry.Registry
	settingsActions SettingsActionRuntime
	// executableTrust 在 V3 P1 开关开启时取代 confirmCapabilities 布尔确认。
	executableTrust        *ExecutableTrustService
	trustChallengesEnabled bool
	safeMode               bool
	activation             *ActivationCoordinator
	lifecycleInspector     LifecycleInspectionRepository
	lifecycleCoordinator   LifecycleCoordinatorRunner
	lifecyclePreflight     LifecycleStaticPreflight
	lifecycleAuthority     LifecycleAuthorityRepository
	lifecycleFinalizer     LifecycleCleanupFinalizer
	queryPublications      RuntimeQueryPublicationBoundary
	cachePublications      RuntimeCachePublicationBoundary
	// pluginMemorySampler 可选；测试注入固定 RSS 映射。nil 时用 OS 进程采样。
	pluginMemorySampler func() map[string]uint64
}

// PageRegistry 主题/插件页面贡献注册（避免 extensions 直接依赖 pages 包实现细节）。
type PageRegistry interface {
	// PreflightThemePackage 激活前完整预检，不修改 Registry。
	// previousActiveThemeID 非空时按「替换旧主题」的最终状态校验 add 路径。
	PreflightThemePackage(ctx context.Context, extension Extension, previousActiveThemeID string) error
	// RegisterThemePackage 校验并注册主题页面贡献（仅候选，不批准 replace）。
	RegisterThemePackage(ctx context.Context, extension Extension) error
	RegisterThemePackageRestoring(ctx context.Context, extension Extension, staleThemeIDs []string) error
	// RegisterDefaultThemeFallback compiles the protected default without
	// making its Page Registry contributions active.
	RegisterDefaultThemeFallback(ctx context.Context, extension Extension) error
	// RegisterThemePackageReplacing 原子替换旧活动主题贡献（同路径切换允许）。
	RegisterThemePackageReplacing(ctx context.Context, extension Extension, previousActiveThemeID string) error
	RegisterThemePackageReplacingApproved(ctx context.Context, extension Extension, previousActiveThemeID string, approvedBy int64) error
	// RegisterPluginPackage 插件 enable 时注册页面贡献（统一 theme.json pages 契约）。
	RegisterPluginPackage(ctx context.Context, extension Extension) error
	// ClearExtension 禁用/卸载/切换时撤销贡献。
	ClearExtension(extensionID string)
}

// ComponentRegistry 是主题同步激活所需的最小声明式边界。
// 实现必须使用 Host 的 exact package runtime identity，并且原子地
// 将新主题发布与旧主题撤销放在同一个快照中。
type ComponentRegistry interface {
	ValidateThemeTransition(target, source *Extension) error
	PublishThemeTransition(target, source *Extension, publicationRevision int64) error
	RollbackThemeTransition(target, source *Extension, publicationRevision int64) error
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

func WithRouteProviderSelectionInvalidator(invalidator RouteProviderSelectionInvalidator) ServiceOption {
	return func(s *Service) {
		s.routeProviderSelections = invalidator
	}
}

func WithProviderSlotSelectionInvalidator(invalidator ProviderSlotSelectionInvalidator) ServiceOption {
	return func(s *Service) {
		s.providerSlotSelections = invalidator
	}
}

// WithPageRegistry 注入运行时 Page Registry（主题激活无 Nuxt 重建）。
func WithPageRegistry(registry PageRegistry) ServiceOption {
	return func(s *Service) {
		s.pageRegistry = registry
	}
}

// WithComponentRegistry 注入与公开 L2 admission 共享的 Component Registry。
func WithComponentRegistry(registry ComponentRegistry) ServiceOption {
	return func(s *Service) {
		s.componentRegistry = registry
	}
}

// WithAssetRegistry 注入与公开 L2 请求路径共享的 Host Asset Registry。
func WithAssetRegistry(registry *assetregistry.Registry) ServiceOption {
	return func(s *Service) {
		s.assetRegistry = registry
	}
}

// BindAssetRegistry late-binds the shared Host Registry after authoritative
// lifecycle startup reconciliation. Binding never performs a second restore.
func (s *Service) BindAssetRegistry(registry *assetregistry.Registry) *Service {
	if s == nil {
		return nil
	}
	s.assetPublicationMu.Lock()
	s.assetRegistry = registry
	s.assetPublicationMu.Unlock()
	return s
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

// WithRuntimeCachePublications injects the shared Host Cache Registry boundary.
// A nil boundary preserves the legacy behavior until production bootstrap binds it.
func WithRuntimeCachePublications(boundary RuntimeCachePublicationBoundary) ServiceOption {
	return func(s *Service) { s.cachePublications = boundary }
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

// WithPluginMemorySampler 注入扩展列表/详情用的插件 RSS 映射（测试或禁用采样）。
func WithPluginMemorySampler(sampler func() map[string]uint64) ServiceOption {
	return func(s *Service) { s.pluginMemorySampler = sampler }
}

func WithActivationCoordinator(coordinator *ActivationCoordinator) ServiceOption {
	return func(s *Service) { s.activation = coordinator }
}

// WithLifecycleCoordinator 注入 V2 durable coordinator 及其静态前置检查。
// preflight 只接收不可变制品快照，不得持有或启动 runtime process。
func WithLifecycleCoordinator(
	coordinator LifecycleCoordinatorRunner,
	preflight LifecycleStaticPreflight,
	authority LifecycleAuthorityRepository,
) ServiceOption {
	return func(s *Service) {
		s.lifecycleCoordinator = coordinator
		s.lifecyclePreflight = preflight
		s.lifecycleAuthority = authority
	}
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
	// 整表只采样一次 ps，避免每个扩展各扫一遍进程表。
	memoryByID := s.sampleOwnedPluginMemory()
	for index := range items {
		items[index] = applyPluginMemory(s.decorateRuntime(ctx, items[index]), memoryByID)
	}
	return items, nil
}

// Detail 只读取并装饰一个扩展，避免详情页把完整扩展目录写入 SSR payload。
func (s *Service) Detail(ctx context.Context, actor identity.Actor, extensionID string) (Extension, error) {
	if !canViewExtensions(actor) {
		return Extension{}, identity.ErrPermissionDenied
	}
	item, err := s.store.Get(ctx, normalizeID(extensionID))
	if err != nil {
		return Extension{}, err
	}
	return applyPluginMemory(s.decorateRuntime(ctx, item), s.sampleOwnedPluginMemory()), nil
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
	activeTheme, activeThemeErr := s.store.ActiveTheme(ctx)
	switch {
	case errors.Is(activeThemeErr, ErrExtensionNotFound):
		if _, err := s.EnsureDefaultThemeActive(ctx); err != nil && !errors.Is(err, ErrExtensionNotFound) {
			return nil, err
		}
	case activeThemeErr != nil:
		return nil, activeThemeErr
	case activeTheme.Source == SourceBuiltin && !activeBuiltinIDs[activeTheme.ID]:
		// 仅当活动内置主题已经从发行包移除时回退默认主题；有效的运营选择必须跨重启保留。
		if _, err := s.EnsureDefaultThemeActive(ctx); err != nil && !errors.Is(err, ErrExtensionNotFound) {
			return nil, err
		}
	}
	// 第一轮会保留仍活动但已从内置目录移除的主题；发布默认主题修复后再安全清理。
	if len(activeBuiltinIDs) > 0 {
		ids := make([]string, 0, len(activeBuiltinIDs))
		for id := range activeBuiltinIDs {
			ids = append(ids, id)
		}
		if err := s.store.PruneMissingBuiltins(ctx, ids); err != nil {
			return nil, err
		}
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

// AdminPageBootstrap 一次加载扩展详情页：store.Get 仅一次；未知 path 不报错，page/settings 为 null。
// 仅当匹配页 View 声明为 settings 时才加载掩码设置，不根据 path 文本推断。
func (s *Service) AdminPageBootstrap(ctx context.Context, actor identity.Actor, extensionID, pagePath, locale string) (AdminPageBootstrap, error) {
	// Settings 管理权限不反向继承 extension.view；先允许潜在设置管理员进入精确
	// 扩展判定，普通/未知页面仍在下方严格要求只读目录权限。
	if !canViewExtensions(actor) && !canManagePlugins(actor) && !canManageThemes(actor) && !actor.Can(identity.PermissionSettingsMailManage) {
		return AdminPageBootstrap{}, identity.ErrPermissionDenied
	}
	extension, err := s.store.Get(ctx, normalizeID(extensionID))
	if err != nil {
		return AdminPageBootstrap{}, err
	}
	extension = applyPluginMemory(s.decorateRuntime(ctx, extension), s.sampleOwnedPluginMemory())
	result := AdminPageBootstrap{Extension: extension}

	want := normalizeRoutePath(pagePath)
	for _, page := range normalizedAdminPages(extension.Manifest) {
		if page.Path != want {
			continue
		}
		matched := page
		result.Page = &matched
		break
	}
	if result.Page == nil || result.Page.View != "settings" {
		if !canViewExtensions(actor) {
			return AdminPageBootstrap{}, identity.ErrPermissionDenied
		}
		return result, nil
	}
	if !canManageExtensionSettings(actor, extension) {
		return AdminPageBootstrap{}, identity.ErrPermissionDenied
	}
	values, err := s.listDecryptedSettings(ctx, extension)
	if err != nil {
		return AdminPageBootstrap{}, err
	}
	settings := resolveExtensionSettings(extension, values, locale)
	result.Settings = &settings
	return result, nil
}

func (s *Service) UpdateSettings(ctx context.Context, actor identity.Actor, extensionID string, input UpdateSettingsInput, locale string) (ExtensionSettings, error) {
	s.assetPublicationMu.Lock()
	defer s.assetPublicationMu.Unlock()

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
	restart, err := s.preparePluginSettingsRestart(ctx, extension)
	if err != nil {
		return ExtensionSettings{}, err
	}
	if mutationErr := s.store.ReplaceSettings(ctx, extension.ID, stored); mutationErr != nil {
		if err := s.resolvePreparedSettingsMutation(ctx, extension.ID, previousRaw, stored, restart, mutationErr); err != nil {
			return ExtensionSettings{}, err
		}
	}
	if err := s.restartPluginForSettings(ctx, extension, restart); err != nil {
		return ExtensionSettings{}, s.restoreSettingsAfterRestartFailure(ctx, extension.ID, previousRaw, restart, err)
	}
	// 返回解密后的视图（secret 仍在 resolve 中掩码）。
	return resolveExtensionSettings(extension, values, locale), nil
}

func (s *Service) ResetSettings(ctx context.Context, actor identity.Actor, extensionID string, locale string) (ExtensionSettings, error) {
	s.assetPublicationMu.Lock()
	defer s.assetPublicationMu.Unlock()

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
	restart, err := s.preparePluginSettingsRestart(ctx, extension)
	if err != nil {
		return ExtensionSettings{}, err
	}
	if mutationErr := s.store.ResetSettings(ctx, extension.ID); mutationErr != nil {
		if err := s.resolvePreparedSettingsMutation(ctx, extension.ID, previousRaw, map[string]string{}, restart, mutationErr); err != nil {
			return ExtensionSettings{}, err
		}
	}
	if err := s.restartPluginForSettings(ctx, extension, restart); err != nil {
		return ExtensionSettings{}, s.restoreSettingsAfterRestartFailure(ctx, extension.ID, previousRaw, restart, err)
	}
	return resolveExtensionSettings(extension, map[string]string{}, locale), nil
}
