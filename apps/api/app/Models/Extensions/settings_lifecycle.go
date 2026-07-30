package extensions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	settingslifecycle "github.com/zhuchunshu/sforum/apps/api/app/Support/SettingsLifecycle"
)

// ErrSettingsRevisionConflict is returned when concurrent settings saves race on CAS.
var ErrSettingsRevisionConflict = errors.New("extensions: settings revision conflict")

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
			Required:         setting.Required,
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
		callouts = append(callouts, ExtensionSettingsCallout{
			ID: callout.ID, Tone: callout.Tone, Title: callout.Title.Resolve(locale), Body: callout.Body.Resolve(locale),
			LinkLabel: callout.LinkLabel.Resolve(locale), LinkURL: callout.LinkURL, Tab: callout.Tab, Group: callout.Group,
		})
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

// WithSettingsLifecycle 注入生产 SettingsLifecycle（后台保存/重置/导入/升级必经）。
func WithSettingsLifecycle(svc *settingslifecycle.Service) ServiceOption {
	return func(s *Service) {
		s.settingsLifecycle = svc
	}
}

// BindSettingsLifecycle late-binds production SettingsLifecycle after bootstrap
// creates the platform services (extensionService is constructed first).
func (s *Service) BindSettingsLifecycle(svc *settingslifecycle.Service) *Service {
	if s == nil {
		return nil
	}
	s.assetPublicationMu.Lock()
	s.settingsLifecycle = svc
	s.assetPublicationMu.Unlock()
	return s
}

// SettingsLifecycle returns the bound lifecycle service (may be nil in tests).
func (s *SettingsService) SettingsLifecycle() SettingsLifecycleRuntime {
	if s == nil {
		return nil
	}
	return s.settingsLifecycle
}

// RegisterSettingsLifecycleFromManifest 从 Manifest 恢复 Schema（启用/升级时调用）。
// Migration 若已由插件 Host API / 注册表声明则保留；否则按 dataVersion 目标注册空路径。
func (s *SettingsService) RegisterSettingsLifecycleFromManifest(extension Extension) error {
	if s == nil || s.settingsLifecycle == nil {
		return nil
	}
	fields := manifestSettingsFields(extension.Manifest)
	if len(fields) == 0 {
		return nil
	}
	dataVersion := settingsDataVersion(extension.Manifest)
	if err := s.settingsLifecycle.RegisterSchema(extension.ID, dataVersion, fields); err != nil {
		return fmt.Errorf("register settings schema for %s: %w", extension.ID, err)
	}
	return nil
}

// UpdateSettings 后台保存：有 SettingsLifecycle 时走文档 CAS + 迁移；否则旧路径。
func (s *serviceCore) updateSettingsViaLifecycle(
	ctx context.Context,
	actor identity.Actor,
	extension Extension,
	input UpdateSettingsInput,
	locale string,
) (ExtensionSettings, error) {
	if err := s.host.RegisterSettingsLifecycleFromManifest(extension); err != nil {
		return ExtensionSettings{}, err
	}
	restart, err := s.preparePluginSettingsRestart(ctx, actor, extension)
	if err != nil {
		return ExtensionSettings{}, err
	}
	actorID := settingsActorID(actor)
	// 合并：未提交字段由 lifecycle Put 保留；secret 空串保留。
	doc, err := s.settingsLifecycle.Put(ctx, extension.ID, actorID, input.Values, true)
	if err != nil {
		return ExtensionSettings{}, mapSettingsLifecycleError(err)
	}
	// 重启插件以加载新设置（与旧路径一致）。
	if err := s.restartPluginForSettings(ctx, actor, extension, restart, settingsRestartMutationKey(extension, doc)); err != nil {
		// 设置已 CAS 提交；Lifecycle 权威已前进，不倒退 revision。明确返回
		// “已保存但重启失败”，由生命周期账本保留安全恢复状态。
		return ExtensionSettings{}, err
	}
	maybeBumpPublicSurfaceRevision(s.host, ctx, extension)
	return resolveExtensionSettings(extension, settingsDocumentViewValues(doc), locale), nil
}

// resetSettingsViaLifecycle 重置为字段默认值（推荐路径保留 SecretStore refs）。
func (s *serviceCore) resetSettingsViaLifecycle(
	ctx context.Context,
	actor identity.Actor,
	extension Extension,
	locale string,
) (ExtensionSettings, error) {
	if err := s.host.RegisterSettingsLifecycleFromManifest(extension); err != nil {
		return ExtensionSettings{}, err
	}
	restart, err := s.preparePluginSettingsRestart(ctx, actor, extension)
	if err != nil {
		return ExtensionSettings{}, err
	}
	doc, err := s.settingsLifecycle.ResetDefaults(ctx, extension.ID, settingsActorID(actor), settingslifecycle.ResetOptions{
		// 初学者友好：重置表单默认值时保留密钥引用。
		PreserveSecrets: true,
	})
	if err != nil {
		return ExtensionSettings{}, mapSettingsLifecycleError(err)
	}
	if err := s.restartPluginForSettings(ctx, actor, extension, restart, settingsRestartMutationKey(extension, doc)); err != nil {
		return ExtensionSettings{}, err
	}
	maybeBumpPublicSurfaceRevision(s.host, ctx, extension)
	return resolveExtensionSettings(extension, settingsDocumentViewValues(doc), locale), nil
}

// ImportSettings 从导出包恢复设置（经 SettingsLifecycle；拒绝密文泄漏）。
func (s *SettingsService) ImportSettings(
	ctx context.Context,
	actor identity.Actor,
	extensionID string,
	bundle settingslifecycle.ExportBundle,
	locale string,
) (ExtensionSettings, error) {
	s.assetPublicationMu.Lock()
	defer s.assetPublicationMu.Unlock()

	extension, err := s.store.Get(ctx, normalizeID(extensionID))
	if err != nil {
		return ExtensionSettings{}, err
	}
	if !canManageExtensionSettings(actor, extension) {
		return ExtensionSettings{}, identity.ErrPermissionDenied
	}
	if s.settingsLifecycle == nil {
		return ExtensionSettings{}, fmt.Errorf("extensions: settings lifecycle is not bound")
	}
	if err := s.host.RegisterSettingsLifecycleFromManifest(extension); err != nil {
		return ExtensionSettings{}, err
	}
	restart, err := s.preparePluginSettingsRestart(ctx, actor, extension)
	if err != nil {
		return ExtensionSettings{}, err
	}
	doc, err := s.settingsLifecycle.Import(ctx, extension.ID, settingsActorID(actor), bundle)
	if err != nil {
		return ExtensionSettings{}, mapSettingsLifecycleError(err)
	}
	if err := s.restartPluginForSettings(ctx, actor, extension, restart, settingsRestartMutationKey(extension, doc)); err != nil {
		return ExtensionSettings{}, err
	}
	return resolveExtensionSettings(extension, settingsDocumentViewValues(doc), locale), nil
}

// ExportSettings 导出掩码设置包（无密钥明文）。
func (s *SettingsService) ExportSettings(ctx context.Context, actor identity.Actor, extensionID string) (settingslifecycle.ExportBundle, error) {
	extension, err := s.store.Get(ctx, normalizeID(extensionID))
	if err != nil {
		return settingslifecycle.ExportBundle{}, err
	}
	if !canManageExtensionSettings(actor, extension) {
		return settingslifecycle.ExportBundle{}, identity.ErrPermissionDenied
	}
	if s.settingsLifecycle == nil {
		return settingslifecycle.ExportBundle{}, fmt.Errorf("extensions: settings lifecycle is not bound")
	}
	if err := s.host.RegisterSettingsLifecycleFromManifest(extension); err != nil {
		return settingslifecycle.ExportBundle{}, err
	}
	return s.settingsLifecycle.Export(ctx, extension.ID)
}

// migrateSettingsOnUpgrade 升级时恢复 Schema 并触发迁移（失败不得改 revision）。
func (s *serviceCore) migrateSettingsOnUpgrade(ctx context.Context, actor identity.Actor, extension Extension) error {
	if s == nil || s.settingsLifecycle == nil {
		return nil
	}
	if err := s.host.RegisterSettingsLifecycleFromManifest(extension); err != nil {
		return err
	}
	// 读当前文档；若不存在则无迁移。Put 空 map + preserve 会跑 migrate。
	_, err := s.settingsLifecycle.Get(ctx, extension.ID)
	if err != nil {
		if err == settingslifecycle.ErrNotFound {
			return nil
		}
		return mapSettingsLifecycleError(err)
	}
	// 空 Put：保留全部值/密钥，仅推进 dataVersion（迁移失败则整单回滚）。
	_, err = s.settingsLifecycle.Put(ctx, extension.ID, settingsActorID(actor), map[string]string{}, true)
	if err != nil {
		return mapSettingsLifecycleError(err)
	}
	return nil
}

func manifestSettingsFields(manifest Manifest) []settingslifecycle.FieldSchema {
	out := make([]settingslifecycle.FieldSchema, 0, len(manifest.Settings))
	for _, field := range manifest.Settings {
		key := strings.TrimSpace(field.Key)
		if key == "" {
			continue
		}
		typ := strings.ToLower(strings.TrimSpace(field.Type))
		if typ == "" {
			typ = "string"
		}
		schema := settingslifecycle.FieldSchema{
			Name:    key,
			Type:    typ,
			Default: field.Default,
			Secret:  typ == "secret",
		}
		if typ == "select" || typ == "enum" {
			schema.Type = "select"
			for _, opt := range field.Options {
				schema.Options = append(schema.Options, opt.Value)
			}
		}
		out = append(out, schema)
	}
	return out
}

func settingsDataVersion(manifest Manifest) int {
	// SettingsDocument.SchemaVersion 是 UI 契约；数据版本至少为 1。
	// 若未来 Manifest 声明 dataVersion，可在此读取。
	v := manifest.SettingsDocument.SchemaVersion
	if v < 1 {
		return 1
	}
	return v
}

func settingsActorID(actor identity.Actor) string {
	if actor.ID > 0 {
		return "user:" + strconv.FormatInt(actor.ID, 10)
	}
	return "actor:system"
}

func mapSettingsLifecycleError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, settingslifecycle.ErrPermissionDenied):
		return identity.ErrPermissionDenied
	case errors.Is(err, settingslifecycle.ErrNotFound):
		return ErrExtensionNotFound
	case errors.Is(err, settingslifecycle.ErrConflict):
		return fmt.Errorf("%w: settings revision conflict", ErrSettingsRevisionConflict)
	case errors.Is(err, settingslifecycle.ErrValidation), errors.Is(err, settingslifecycle.ErrInvalid):
		return fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	case errors.Is(err, settingslifecycle.ErrMigration):
		return fmt.Errorf("%w: %v", ErrPreflightFailed, err)
	default:
		// 包装 migration/secret 等错误，保留 errors.Is 链。
		return err
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func settingsDocumentViewValues(doc settingslifecycle.Document) map[string]string {
	values := cloneStringMap(doc.Values)
	for name, set := range doc.SecretSet {
		if set {
			// resolveExtensionSettings 会把 secret value 清空；这里的非空占位只用于
			// 保留 secretSet=true，避免保存/重置响应让运营误以为密码已丢失。
			values[name] = "__sforum.secret_set"
		}
	}
	return values
}

type settingsRestartCoordinator struct {
	service *serviceCore
}

func newSettingsRestartCoordinator(service *serviceCore) settingsRestartCoordinator {
	return settingsRestartCoordinator{service: service}
}

// preflightLifecycleV2 在设置文档或 SecretStore 修改前检查停用、启用两个阶段。
// 实际执行时协调器仍会重新校验运行时状态。
func (c settingsRestartCoordinator) preflightLifecycleV2(
	ctx context.Context,
	actor identity.Actor,
	extension Extension,
) error {
	s := c.service
	if !canManageExtensionSettings(actor, extension) {
		return identity.ErrPermissionDenied
	}
	if s.lifecycleCoordinator == nil || s.lifecyclePreflight == nil || s.lifecycleAuthority == nil {
		return ErrLifecycleCoordinatorUnavailable
	}
	if err := s.preflightRestartTarget(ctx, extension); err != nil {
		return err
	}
	disable := lifecycleServiceRequest{
		operation: LifecycleMachineDisable,
		source:    exactLifecycleCopy(extension), target: extension,
		idempotencyKey: "settings-preflight-disable", frozenAuthority: true,
	}
	if _, err := s.lifecycleServiceAuthority(ctx, actor, disable); err != nil {
		return err
	}
	if err := s.lifecyclePreflight(ctx, disable.operation, disable.source, disable.target); err != nil {
		return errors.Join(ErrPreflightFailed, err)
	}
	enable := lifecycleServiceRequest{
		operation: LifecycleMachineEnable, target: extension,
		idempotencyKey: "settings-preflight-enable",
	}
	if _, err := s.lifecycleServiceAuthority(ctx, actor, enable); err != nil {
		return err
	}
	if err := s.lifecyclePreflight(ctx, enable.operation, nil, enable.target); err != nil {
		return errors.Join(ErrPreflightFailed, err)
	}
	return nil
}

// restartLifecycleV2 只重启精确 active 制品。设置保存不是升级，
// 因此即使存在 staged 制品也不得顺便晋升。
func (c settingsRestartCoordinator) restartLifecycleV2(
	ctx context.Context,
	actor identity.Actor,
	extension Extension,
	mutationKey string,
) error {
	s := c.service
	if mutationKey == "" {
		return fmt.Errorf("%w: settings mutation identity is missing", ErrLifecycleCoordinatorInvalid)
	}
	if _, err := s.disableLifecycleV2(ctx, actor, extension, LifecycleRequestInput{
		IdempotencyKey: settingsRestartPhaseKey(mutationKey, "disable"),
	}); err != nil {
		return fmt.Errorf("disable exact plugin for settings restart: %w", err)
	}

	disabled, err := s.store.Get(ctx, extension.ID)
	if err != nil {
		return fmt.Errorf("reload plugin after settings disable: %w", err)
	}
	if disabled.Status != StatusDisabled || !sameRestartArtifact(disabled, extension) {
		return fmt.Errorf("%w: settings restart exact artifact changed while disabled", ErrLifecycleCoordinatorInvalid)
	}
	if _, err := s.enableLifecycleV2(ctx, actor, disabled, EnableInput{
		IdempotencyKey: settingsRestartPhaseKey(mutationKey, "enable"),
	}); err != nil {
		return fmt.Errorf("enable exact plugin after settings change: %w", err)
	}
	return nil
}

func settingsRestartMutationKey(extension Extension, doc settingslifecycle.Document) string {
	updatedAt := doc.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = time.Unix(0, 0).UTC()
	}
	digest := sha256.Sum256([]byte(
		extension.ID + "\x00" + extension.Version + "\x00" + extension.PackageDigest + "\x00" +
			doc.UpdatedBy + "\x00" + strconv.FormatInt(updatedAt.UnixNano(), 10),
	))
	return hex.EncodeToString(digest[:])
}

func legacySettingsRestartMutationKey(extension Extension, actor identity.Actor) string {
	doc := settingslifecycle.Document{
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: settingsActorID(actor),
	}
	return settingsRestartMutationKey(extension, doc)
}

func settingsRestartPhaseKey(mutationKey string, phase string) string {
	digest := sha256.Sum256([]byte("sforum.extension.settings.restart\x00" + phase + "\x00" + mutationKey))
	return "settings-" + phase + "-" + hex.EncodeToString(digest[:])
}
