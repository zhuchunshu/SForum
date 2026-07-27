package extensions

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	settingslifecycle "github.com/zhuchunshu/sforum/apps/api/app/Support/SettingsLifecycle"
)

// ErrSettingsRevisionConflict is returned when concurrent settings saves race on CAS.
var ErrSettingsRevisionConflict = errors.New("extensions: settings revision conflict")

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
	actorID := settingsActorID(actor)
	// 合并：未提交字段由 lifecycle Put 保留；secret 空串保留。
	doc, err := s.settingsLifecycle.Put(ctx, extension.ID, actorID, input.Values, true)
	if err != nil {
		return ExtensionSettings{}, mapSettingsLifecycleError(err)
	}
	// 重启插件以加载新设置（与旧路径一致）。
	restart, err := s.preparePluginSettingsRestart(ctx, extension)
	if err != nil {
		return ExtensionSettings{}, err
	}
	if err := s.restartPluginForSettings(ctx, extension, restart); err != nil {
		// 设置已 CAS 提交；重启失败时尽力回读并表面错误（与旧路径 restore 不同：
		// lifecycle 权威已前进，不倒退 revision）。
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
	doc, err := s.settingsLifecycle.ResetDefaults(ctx, extension.ID, settingsActorID(actor), settingslifecycle.ResetOptions{
		// 初学者友好：重置表单默认值时保留密钥引用。
		PreserveSecrets: true,
	})
	if err != nil {
		return ExtensionSettings{}, mapSettingsLifecycleError(err)
	}
	restart, err := s.preparePluginSettingsRestart(ctx, extension)
	if err != nil {
		return ExtensionSettings{}, err
	}
	if err := s.restartPluginForSettings(ctx, extension, restart); err != nil {
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
	doc, err := s.settingsLifecycle.Import(ctx, extension.ID, settingsActorID(actor), bundle)
	if err != nil {
		return ExtensionSettings{}, mapSettingsLifecycleError(err)
	}
	restart, err := s.preparePluginSettingsRestart(ctx, extension)
	if err != nil {
		return ExtensionSettings{}, err
	}
	if err := s.restartPluginForSettings(ctx, extension, restart); err != nil {
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
