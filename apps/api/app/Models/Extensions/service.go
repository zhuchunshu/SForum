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

func (s *Service) List(ctx context.Context, actor identity.Actor) ([]Extension, error) {
	if !actor.Can(identity.PermissionExtensionManage) {
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
			manifestPath := filepath.Join(root, entry.Name(), ManifestFileName)
			body, err := os.ReadFile(manifestPath)
			if err != nil {
				return nil, err
			}
			var manifest Manifest
			if err := json.Unmarshal(body, &manifest); err != nil {
				return nil, ErrInvalidManifest
			}
			manifest = normalizeManifest(manifest)
			if manifest.Type != group.extensionType {
				return nil, ErrInvalidManifest
			}
			if err := validateManifest(manifest); err != nil {
				return nil, err
			}
			packageRoot := filepath.Dir(manifestPath)
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
	if !actor.Can(identity.PermissionExtensionManage) {
		return nil, identity.ErrPermissionDenied
	}
	return s.store.ListEvents(ctx, normalizeID(extensionID), limit)
}

func (s *Service) EventDefinitions(ctx context.Context, actor identity.Actor) ([]appevents.Definition, error) {
	if !actor.Can(identity.PermissionExtensionManage) {
		return nil, identity.ErrPermissionDenied
	}
	return appevents.Definitions(), nil
}

func (s *Service) EventDeliveries(ctx context.Context, actor identity.Actor, input EventDeliveryListInput) ([]ExtensionEventDelivery, error) {
	if !actor.Can(identity.PermissionExtensionManage) {
		return nil, identity.ErrPermissionDenied
	}
	input.ExtensionID = normalizeID(input.ExtensionID)
	return s.store.ListEventDeliveries(ctx, input)
}

func (s *Service) ContributionPoints(_ context.Context, actor identity.Actor) ([]ContributionPointDefinition, error) {
	if !actor.Can(identity.PermissionExtensionManage) {
		return nil, identity.ErrPermissionDenied
	}
	return extensionmanifest.ContributionPointDefinitions(), nil
}

func (s *Service) Contributions(ctx context.Context, actor identity.Actor) ([]EffectiveContribution, error) {
	if !actor.Can(identity.PermissionExtensionManage) {
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
	if !actor.Can(identity.PermissionExtensionManage) {
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
	values, err := s.store.ListSettings(ctx, extension.ID)
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
	current, err := s.store.ListSettings(ctx, extension.ID)
	if err != nil {
		return ExtensionSettings{}, err
	}
	values, err := sanitizeSettingValues(extension.Manifest, input.Values, current)
	if err != nil {
		return ExtensionSettings{}, err
	}
	if err := s.store.ReplaceSettings(ctx, extension.ID, values); err != nil {
		return ExtensionSettings{}, err
	}
	if err := s.restartPluginForSettings(ctx, extension); err != nil {
		return ExtensionSettings{}, err
	}
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
	if err := s.store.ResetSettings(ctx, extension.ID); err != nil {
		return ExtensionSettings{}, err
	}
	if err := s.restartPluginForSettings(ctx, extension); err != nil {
		return ExtensionSettings{}, err
	}
	return resolveExtensionSettings(extension, map[string]string{}, locale), nil
}

func canManageExtensionSettings(actor identity.Actor, extension Extension) bool {
	if actor.Can(identity.PermissionExtensionManage) {
		return true
	}
	if !actor.Can(identity.PermissionSettingsManage) {
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

func (s *Service) InstallArchive(ctx context.Context, actor identity.Actor, input ArchiveInput) (Extension, error) {
	if !actor.Can(identity.PermissionExtensionManage) {
		return Extension{}, identity.ErrPermissionDenied
	}
	if len(input.Data) == 0 || len(input.Data) > maxArchiveBytes {
		return Extension{}, ErrInvalidArchive
	}

	manifest, files, err := readArchive(input.Data)
	if err != nil {
		return Extension{}, err
	}
	if err := validateManifest(manifest); err != nil {
		return Extension{}, err
	}
	if manifest.ID == DefaultThemeID {
		return Extension{}, ErrInvalidManifest
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return Extension{}, err
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
			return Extension{}, ErrInvalidManifest
		case errors.Is(err, extensionpackage.ErrInvalidPath),
			errors.Is(err, extensionpackage.ErrNonRegular),
			errors.Is(err, extensionpackage.ErrSymlink):
			return Extension{}, ErrInvalidArchive
		}
		return Extension{}, err
	}

	installed, err := s.store.SaveInstalled(ctx, SaveInstalledInput{
		Manifest:      manifest,
		PackagePath:   snapshot.Root,
		PackageDigest: snapshot.Digest,
	})
	if err != nil {
		return Extension{}, err
	}
	_, _ = s.store.CreateEvent(ctx, EventInput{
		ExtensionID: installed.ID,
		ActorUserID: actor.ID,
		Action:      EventInstalled,
		Message:     "Extension archive installed.",
	})
	return s.decorateRuntime(ctx, installed), nil
}

func (s *Service) Enable(ctx context.Context, actor identity.Actor, id string) (Extension, error) {
	if !actor.Can(identity.PermissionExtensionManage) {
		return Extension{}, identity.ErrPermissionDenied
	}
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return Extension{}, err
	}
	if extension.Type == TypeTheme {
		return Extension{}, ErrThemeActivationRequired
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
	_, _ = s.store.CreateEvent(ctx, EventInput{
		ExtensionID: enabled.ID,
		ActorUserID: actor.ID,
		Action:      EventEnabled,
		Message:     "Extension enabled.",
	})
	if enabled.Type == TypePlugin && s.runtime != nil {
		s.runtime.EmitHook(ctx, appevents.ExtensionEnabled, map[string]any{"extensionId": enabled.ID})
	}
	return s.decorateRuntime(ctx, enabled), nil
}

func (s *Service) Disable(ctx context.Context, actor identity.Actor, id string) (Extension, error) {
	if !actor.Can(identity.PermissionExtensionManage) {
		return Extension{}, identity.ErrPermissionDenied
	}
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return Extension{}, err
	}
	if extension.Type == TypeTheme {
		return Extension{}, ErrThemeActivationRequired
	}
	disabled, err := s.store.Disable(ctx, extension.ID)
	if err != nil {
		return Extension{}, err
	}
	if selectionStore, ok := s.store.(interface {
		SelectedMailProvider(context.Context) (string, error)
		RestoreMailProvider(context.Context) error
	}); ok {
		if selected, selectErr := selectionStore.SelectedMailProvider(ctx); selectErr == nil && selected == disabled.ID {
			if err := selectionStore.RestoreMailProvider(ctx); err != nil {
				return Extension{}, err
			}
		}
	}
	if disabled.Type == TypePlugin && s.runtime != nil {
		_ = s.runtime.Stop(ctx, disabled)
	}
	_, _ = s.store.CreateEvent(ctx, EventInput{
		ExtensionID: disabled.ID,
		ActorUserID: actor.ID,
		Action:      EventDisabled,
		Message:     "Extension disabled.",
	})
	if disabled.Type == TypePlugin && s.runtime != nil {
		s.runtime.EmitHook(ctx, appevents.ExtensionDisabled, map[string]any{"extensionId": disabled.ID})
	}
	return s.decorateRuntime(ctx, disabled), nil
}

func (s *Service) VerifyExtension(ctx context.Context, actor identity.Actor, id string) (Extension, error) {
	if !actor.Can(identity.PermissionExtensionManage) {
		return Extension{}, identity.ErrPermissionDenied
	}
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
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
	if !actor.Can(identity.PermissionExtensionManage) {
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
	if item.Type == TypePlugin && s.runtime != nil {
		status := s.runtime.Status(ctx, item)
		item.Runtime = &status
	}
	if item.Type == TypeTheme {
		if release, err := s.store.LatestThemeRelease(ctx, item.ID); err == nil {
			item.ThemeRelease = &release
		}
	}
	return item
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

	var manifest Manifest
	manifestFound := false
	files := []archiveFile{}
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
		total += file.UncompressedSize64
		if total > maxArchiveBytes {
			return Manifest{}, nil, ErrInvalidArchive
		}
		body, err := readZipFile(file)
		if err != nil {
			return Manifest{}, nil, ErrInvalidArchive
		}
		if name == ManifestFileName {
			if err := json.Unmarshal(body, &manifest); err != nil {
				return Manifest{}, nil, ErrInvalidManifest
			}
			manifestFound = true
			continue
		}
		files = append(files, archiveFile{name: name, mode: file.Mode(), body: body})
	}
	if !manifestFound {
		return Manifest{}, nil, ErrInvalidArchive
	}
	return normalizeManifest(manifest), files, nil
}

func readZipFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
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

func sanitizeSettingValues(manifest Manifest, input, current map[string]string) (map[string]string, error) {
	allowed := map[string]ManifestSetting{}
	for _, setting := range manifest.Settings {
		allowed[setting.Key] = setting
	}
	values := map[string]string{}
	for key, value := range input {
		key = strings.TrimSpace(key)
		if _, ok := allowed[key]; !ok {
			return nil, ErrInvalidManifest
		}
		normalized := strings.TrimSpace(value)
		if allowed[key].Type == "secret" && normalized == "" {
			normalized = current[key]
		}
		values[key] = normalized
	}
	return values, nil
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
