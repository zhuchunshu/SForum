package extensions

import (
	"context"
	"errors"
	"fmt"

	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

var ErrRuntimeContentPublicationUnavailable = errors.New("extensions: runtime content publication boundary is unavailable")
var ErrRuntimeEditorPublicationUnavailable = errors.New("extensions: runtime editor publication boundary is unavailable")

type RuntimeContentPublicationMutation interface {
	Rollback() error
}

// RuntimeContentPublicationBoundary keeps Models independent from the Host
// Content Registry. Plugins without Lifecycle V2 still publish and quarantine
// declarations against the exact process artifact during legacy lifecycle.
type RuntimeContentPublicationBoundary interface {
	PublishRuntimeContent(context.Context, Extension) (RuntimeContentPublicationMutation, error)
	QuarantineRuntimeContent(context.Context, Extension) (RuntimeContentPublicationMutation, error)
}

// RuntimeEditorPublicationMutation is an exact Host-owned Editor Registry
// mutation. Rollback is artifact-CAS guarded and cannot replace a newer graph.
type RuntimeEditorPublicationMutation interface {
	Rollback() error
}

// RuntimeEditorPublicationBoundary keeps legacy lifecycle paths independent
// from the process-local Editor Registry implementation.
type RuntimeEditorPublicationBoundary interface {
	PublishRuntimeEditor(context.Context, Extension) (RuntimeEditorPublicationMutation, error)
	QuarantineRuntimeEditor(context.Context, Extension) (RuntimeEditorPublicationMutation, error)
}

func WithRuntimeContentPublications(boundary RuntimeContentPublicationBoundary) ServiceOption {
	return func(s *Service) { s.contentPublications = boundary }
}

func WithRuntimeEditorPublications(boundary RuntimeEditorPublicationBoundary) ServiceOption {
	return func(s *Service) {
		if s != nil && s.lifecycle != nil {
			s.lifecycle.editorPublications = boundary
		}
	}
}

func hasRuntimeContentPublication(manifest Manifest) bool {
	return len(manifest.Content) > 0
}

func hasRuntimeEditorPublication(manifest Manifest) bool {
	return len(manifest.Editor) > 0
}

func rollbackRuntimeContentMutation(mutation RuntimeContentPublicationMutation, cause error) error {
	if mutation == nil {
		return cause
	}
	if err := mutation.Rollback(); err != nil {
		return errors.Join(cause, fmt.Errorf("restore content publication: %w", err))
	}
	return cause
}

func rollbackRuntimeEditorMutation(mutation RuntimeEditorPublicationMutation, cause error) error {
	if mutation == nil {
		return cause
	}
	if err := mutation.Rollback(); err != nil {
		return errors.Join(cause, fmt.Errorf("restore editor publication: %w", err))
	}
	return cause
}

// compensateLegacyEditorEnable unwinds declarations in reverse publication
// order before reusing the existing runtime/query/cache compensation paths.
func compensateLegacyEditorEnable(
	s *serviceCore,
	ctx context.Context,
	enabled Extension,
	assetMutation exactAssetMutation,
	queryMutation RuntimeQueryPublicationMutation,
	cacheMutation RuntimeCachePublicationMutation,
	contentMutation RuntimeContentPublicationMutation,
	editorMutation RuntimeEditorPublicationMutation,
	actorUserID int64,
	cause error,
) error {
	cause = rollbackRuntimeEditorMutation(editorMutation, cause)
	cause = rollbackRuntimeContentMutation(contentMutation, cause)
	if cacheMutation != nil {
		return s.compensateLegacyCacheEnable(
			ctx, enabled, assetMutation, queryMutation, cacheMutation, actorUserID, cause,
		)
	}
	if queryMutation != nil {
		return s.compensateLegacyQueryEnable(ctx, enabled, assetMutation, queryMutation, actorUserID, cause)
	}
	return compensateLegacyContentEnable(s, ctx, enabled, assetMutation, nil, actorUserID, cause)
}

func publishLegacyRuntimeEditor(
	s *LifecycleService,
	ctx context.Context,
	enabled Extension,
	assetMutation exactAssetMutation,
	queryMutation RuntimeQueryPublicationMutation,
	cacheMutation RuntimeCachePublicationMutation,
	contentMutation RuntimeContentPublicationMutation,
	actorUserID int64,
) (RuntimeEditorPublicationMutation, error) {
	if !hasRuntimeEditorPublication(enabled.Manifest) {
		return nil, nil
	}
	mutation, err := s.editorPublications.PublishRuntimeEditor(ctx, enabled)
	if err == nil && mutation != nil {
		return mutation, nil
	}
	if err == nil {
		err = ErrRuntimeEditorPublicationUnavailable
	}
	return nil, compensateLegacyEditorEnable(
		s.serviceCore, ctx, enabled, assetMutation, queryMutation, cacheMutation,
		contentMutation, mutation, actorUserID, err,
	)
}

func quarantineLegacyRuntimeEditor(
	s *LifecycleService,
	ctx context.Context,
	extension Extension,
	assetMutation exactAssetMutation,
	queryMutation RuntimeQueryPublicationMutation,
	cacheMutation RuntimeCachePublicationMutation,
	contentMutation RuntimeContentPublicationMutation,
	hasRuntimeCaches bool,
	hasRuntimeQuerySurfaces bool,
) (RuntimeEditorPublicationMutation, error) {
	if !hasRuntimeEditorPublication(extension.Manifest) {
		return nil, nil
	}
	mutation, err := s.editorPublications.QuarantineRuntimeEditor(ctx, extension)
	if err == nil && mutation != nil {
		return mutation, nil
	}
	if err == nil {
		err = ErrRuntimeEditorPublicationUnavailable
	}
	err = rollbackRuntimeEditorMutation(mutation, err)
	err = rollbackRuntimeContentMutation(contentMutation, err)
	if hasRuntimeCaches {
		err = s.compensateLegacyCacheDisable(assetMutation, queryMutation, cacheMutation, nil, err)
	} else if hasRuntimeQuerySurfaces {
		err = s.compensateLegacyQueryDisable(assetMutation, queryMutation, nil, err)
	} else if restoreErr := s.rollbackExactAssetMutation(assetMutation); restoreErr != nil {
		err = errors.Join(err, restoreErr)
	}
	return nil, err
}

func compensateLegacyContentEnable(
	s *serviceCore,
	ctx context.Context,
	enabled Extension,
	assetMutation exactAssetMutation,
	contentMutation RuntimeContentPublicationMutation,
	actorUserID int64,
	cause error,
) error {
	errs := []error{rollbackRuntimeContentMutation(contentMutation, cause)}
	if s.runtime != nil {
		if err := s.runtime.Stop(ctx, enabled); err != nil {
			errs = append(errs, fmt.Errorf("stop runtime: %w", err))
		}
	}
	if _, err := s.disableLegacyPluginState(ctx, enabled, actorUserID); err != nil {
		errs = append(errs, fmt.Errorf("disable extension: %w", err))
	}
	if err := s.rollbackExactAssetMutation(assetMutation); err != nil {
		errs = append(errs, fmt.Errorf("restore asset publication: %w", err))
	}
	return errors.Join(errs...)
}

func compensateLegacyContentDisable(
	s *serviceCore,
	assetMutation exactAssetMutation,
	contentMutation RuntimeContentPublicationMutation,
	cause error,
) error {
	err := s.rollbackExactAssetMutation(assetMutation)
	return rollbackRuntimeContentMutation(contentMutation, errors.Join(cause, err))
}

func disableLegacyContentPlugin(
	s *serviceCore,
	ctx context.Context,
	extension Extension,
	assetMutation exactAssetMutation,
	contentMutation RuntimeContentPublicationMutation,
	actorUserID int64,
) (Extension, error) {
	if contentMutation == nil {
		return Extension{}, ErrRuntimeContentPublicationUnavailable
	}
	if err := s.clearPluginProviderSelections(ctx, extension.ID); err != nil {
		return Extension{}, compensateLegacyContentDisable(s, assetMutation, contentMutation, err)
	}
	disabled, err := s.disableLegacyPluginState(ctx, extension, actorUserID)
	if err != nil {
		return Extension{}, compensateLegacyContentDisable(s, assetMutation, contentMutation, err)
	}
	if s.pageRegistry != nil {
		s.pageRegistry.ClearExtension(extension.ID)
	}
	if s.runtime != nil {
		_ = s.runtime.Stop(ctx, extension)
		if extension.Status == StatusEnabled {
			s.runtime.EmitHook(ctx, appevents.ExtensionDisabled, map[string]any{
				"extensionId": extension.ID, "reason": "lifecycle_drain",
			})
		}
	}
	return disabled, nil
}

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
