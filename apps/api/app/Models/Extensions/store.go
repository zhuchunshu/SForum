package extensions

import (
	"context"

	themeruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeRuntime"
)

type Store interface {
	List(ctx context.Context) ([]Extension, error)
	Get(ctx context.Context, id string) (Extension, error)
	SaveInstalled(ctx context.Context, input SaveInstalledInput) (Extension, error)
	SaveBuiltin(ctx context.Context, input SaveBuiltinInput) (Extension, error)
	PruneMissingBuiltins(ctx context.Context, activeIDs []string) error
	// Delete 删除扩展行（CASCADE settings/events/versions）。F2.4 卸载。
	Delete(ctx context.Context, id string) error
	Enable(ctx context.Context, id string, extensionType string) (Extension, error)
	Disable(ctx context.Context, id string) (Extension, error)
	ActivateTheme(ctx context.Context, id string) (Extension, error)
	ActiveTheme(ctx context.Context) (Extension, error)
	CreateThemeRelease(ctx context.Context, input ThemeReleaseInput) (ThemeRelease, error)
	UpdateThemeRelease(ctx context.Context, input ThemeReleaseUpdate) (ThemeRelease, error)
	LatestThemeRelease(ctx context.Context, extensionID string) (ThemeRelease, error)
	ActiveThemeRelease(ctx context.Context) (ThemeRelease, error)
	CreateEvent(ctx context.Context, input EventInput) (ExtensionEvent, error)
	ListEvents(ctx context.Context, extensionID string, limit int) ([]ExtensionEvent, error)
	ListSettings(ctx context.Context, extensionID string) (map[string]string, error)
	ReplaceSettings(ctx context.Context, extensionID string, values map[string]string) error
	ResetSettings(ctx context.Context, extensionID string) error
	// ListMigrationLedger / RecordMigration 插件 SQL 迁移账本（F2.4）。
	ListMigrationLedger(ctx context.Context, extensionID string) ([]MigrationRecord, error)
	RecordMigration(ctx context.Context, extensionID string, record MigrationRecord) error
	CreateEventDelivery(ctx context.Context, input EventDeliveryInput) (ExtensionEventDelivery, error)
	UpdateEventDelivery(ctx context.Context, input EventDeliveryUpdateInput) error
	ListEventDeliveries(ctx context.Context, input EventDeliveryListInput) ([]ExtensionEventDelivery, error)
}

type RuntimePreflight interface {
	Check(ctx context.Context, extension Extension) error
}

type RuntimeManager interface {
	RuntimePreflight
	Start(ctx context.Context, extension Extension) error
	Stop(ctx context.Context, extension Extension) error
	Status(ctx context.Context, extension Extension) RuntimeStatus
	EmitHook(ctx context.Context, name string, payload map[string]any)
}

type ThemeBuilder interface {
	Build(ctx context.Context, extension Extension) error
}

type ThemeActivationDispatcher interface {
	EnqueueThemeActivation(ctx context.Context, release ThemeRelease) error
}

// ThemeCurrentWriter 负责写 theme-releases/current.json。
// 生产 runtime.mjs 与本地 dev supervisor 都依赖这个文件来切换主题，
// 因此恢复默认主题（同步路径）也需要更新它，而不只是改 DB 状态。
type ThemeCurrentWriter interface {
	WriteCurrent(ctx context.Context, current themeruntime.CurrentRelease) error
}
