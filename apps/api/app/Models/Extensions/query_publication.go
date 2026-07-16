package extensions

import (
	"context"
	"errors"
	"fmt"

	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

var ErrRuntimeQueryPublicationUnavailable = errors.New("extensions: runtime query publication boundary is unavailable")

// RuntimeQueryPublicationMutation is an exact Host-owned Registry mutation.
// Rollback must use artifact CAS and must never overwrite or remove a newer
// runtime publication.
type RuntimeQueryPublicationMutation interface {
	Rollback() error
}

// RuntimeQueryPublicationBoundary keeps Models independent from QueryRegistry
// and the process-owning Manager. Query-bearing plugins without Lifecycle V2
// use this boundary after runtime start and before legacy disable side effects.
type RuntimeQueryPublicationBoundary interface {
	PublishRuntimeQueries(context.Context, Extension) (RuntimeQueryPublicationMutation, error)
	QuarantineRuntimeQueries(context.Context, Extension) (RuntimeQueryPublicationMutation, error)
}

// BindRuntimeQueryPublications is late-bound by production bootstrap after the
// shared Manager and Query Registry have been constructed.
func (s *Service) BindRuntimeQueryPublications(boundary RuntimeQueryPublicationBoundary) *Service {
	if s == nil {
		return nil
	}
	s.queryPublications = boundary
	return s
}

func (s *Service) compensateLegacyQueryEnable(
	ctx context.Context,
	enabled Extension,
	assetMutation exactAssetMutation,
	queryMutation RuntimeQueryPublicationMutation,
	cause error,
) error {
	errs := []error{cause}
	// Close the newly published query before stopping its exact runtime.
	if queryMutation != nil {
		if err := queryMutation.Rollback(); err != nil {
			errs = append(errs, fmt.Errorf("restore query publication: %w", err))
		}
	}
	if s.runtime != nil {
		if err := s.runtime.Stop(ctx, enabled); err != nil {
			errs = append(errs, fmt.Errorf("stop runtime: %w", err))
		}
	}
	if _, err := s.store.Disable(ctx, enabled.ID); err != nil {
		errs = append(errs, fmt.Errorf("disable extension: %w", err))
	}
	if err := s.rollbackExactAssetMutation(assetMutation); err != nil {
		errs = append(errs, fmt.Errorf("restore asset publication: %w", err))
	}
	return errors.Join(errs...)
}

func (s *Service) disableLegacyQueryPlugin(
	ctx context.Context,
	extension Extension,
	assetMutation exactAssetMutation,
	queryMutation RuntimeQueryPublicationMutation,
) (Extension, error) {
	if queryMutation == nil {
		return Extension{}, ErrRuntimeQueryPublicationUnavailable
	}
	// QuarantineRuntimeQueries has already closed exact runtime admission. Keep
	// the process retained until Store.Disable commits so compensation can resume
	// that same instance rather than starting an unbound replacement.
	if err := s.clearPluginProviderSelections(ctx, extension.ID); err != nil {
		return Extension{}, s.compensateLegacyQueryDisable(assetMutation, queryMutation, err)
	}
	disabled, err := s.store.Disable(ctx, extension.ID)
	if err != nil {
		return Extension{}, s.compensateLegacyQueryDisable(assetMutation, queryMutation, err)
	}
	if s.pageRegistry != nil {
		s.pageRegistry.ClearExtension(extension.ID)
	}
	if s.runtime != nil {
		_ = s.runtime.Stop(ctx, extension)
		if extension.Status == StatusEnabled {
			s.runtime.EmitHook(ctx, appevents.ExtensionDisabled, map[string]any{
				"extensionId": extension.ID,
				"reason":      "lifecycle_drain",
			})
		}
	}
	return disabled, nil
}

func (s *Service) compensateLegacyQueryDisable(
	assetMutation exactAssetMutation,
	queryMutation RuntimeQueryPublicationMutation,
	cause error,
) error {
	// Runtime admission must remain closed if another surface cannot be restored.
	if err := s.rollbackExactAssetMutation(assetMutation); err != nil {
		return errors.Join(cause, fmt.Errorf("restore asset publication: %w", err))
	}
	if err := queryMutation.Rollback(); err != nil {
		return errors.Join(cause, fmt.Errorf("restore query publication and runtime admission: %w", err))
	}
	return cause
}
