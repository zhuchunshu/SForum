package extensions

import "context"

type Store interface {
	List(ctx context.Context) ([]Extension, error)
	Get(ctx context.Context, id string) (Extension, error)
	SaveInstalled(ctx context.Context, input SaveInstalledInput) (Extension, error)
	SaveBuiltin(ctx context.Context, input SaveBuiltinInput) (Extension, error)
	Enable(ctx context.Context, id string, extensionType string) (Extension, error)
	Disable(ctx context.Context, id string) (Extension, error)
	ActivateTheme(ctx context.Context, id string) (Extension, error)
	ActiveTheme(ctx context.Context) (Extension, error)
	CreateEvent(ctx context.Context, input EventInput) (ExtensionEvent, error)
	ListEvents(ctx context.Context, extensionID string, limit int) ([]ExtensionEvent, error)
}

type RuntimePreflight interface {
	Check(ctx context.Context, extension Extension) error
}

type ThemeBuilder interface {
	Build(ctx context.Context, extension Extension) error
}
