package extensions

import (
	"context"
)

type Store interface {
	List(ctx context.Context) ([]Extension, error)
	Get(ctx context.Context, id string) (Extension, error)
	SaveInstalled(ctx context.Context, input SaveInstalledInput) (Extension, error)
	PromoteStagedVersion(ctx context.Context, input StagedVersionCASInput) (Extension, error)
	DiscardStagedVersion(ctx context.Context, input StagedVersionCASInput) (Extension, error)
	SaveBuiltin(ctx context.Context, input SaveBuiltinInput) (Extension, error)
	PruneMissingBuiltins(ctx context.Context, activeIDs []string) error
	// Delete 删除扩展行（CASCADE settings/events/versions）。F2.4 卸载。
	Delete(ctx context.Context, id string) error
	Enable(ctx context.Context, id string, extensionType string) (Extension, error)
	Disable(ctx context.Context, id string) (Extension, error)
	ActivateTheme(ctx context.Context, id string) (Extension, error)
	ActiveTheme(ctx context.Context) (Extension, error)
	CreateEvent(ctx context.Context, input EventInput) (ExtensionEvent, error)
	ListEvents(ctx context.Context, extensionID string, limit int) ([]ExtensionEvent, error)
	ListSettings(ctx context.Context, extensionID string) (map[string]string, error)
	ReplaceSettings(ctx context.Context, extensionID string, values map[string]string) error
	CompareAndSwapSetting(ctx context.Context, extensionID, name, oldValue, newValue string) (bool, error)
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

type SettingsActionRuntime interface {
	ProbeSettingsAction(ctx context.Context, extension Extension, providerSlot string, values map[string]string) (SettingsActionProbeResult, error)
}
