package extensions

import "context"

// LegacyPluginRuntimePublicationStore keeps the V1 Service call surface while
// allowing the production PostgreSQL store to commit mutable extension state
// and the immutable P12 desired full-set in one transaction. Test and
// third-party Store implementations that do not expose this additive boundary
// retain the pre-P12 behavior.
type LegacyPluginRuntimePublicationStore interface {
	EnableLegacyPluginRuntime(
		context.Context,
		Extension,
		int64,
	) (Extension, PluginRuntimePublication, error)
	DisableLegacyPluginRuntime(
		context.Context,
		Extension,
		int64,
	) (Extension, PluginRuntimePublication, error)
}

func (s *serviceCore) enableLegacyPluginState(
	ctx context.Context,
	target Extension,
	actorUserID int64,
) (Extension, error) {
	if publisher, ok := s.store.(LegacyPluginRuntimePublicationStore); ok {
		enabled, _, err := publisher.EnableLegacyPluginRuntime(ctx, target, actorUserID)
		return enabled, err
	}
	return s.store.Enable(ctx, target.ID, target.Type)
}

func (s *serviceCore) disableLegacyPluginState(
	ctx context.Context,
	target Extension,
	actorUserID int64,
) (Extension, error) {
	if publisher, ok := s.store.(LegacyPluginRuntimePublicationStore); ok {
		disabled, _, err := publisher.DisableLegacyPluginRuntime(ctx, target, actorUserID)
		return disabled, err
	}
	return s.store.Disable(ctx, target.ID)
}
