package extensions

import (
	"context"
	"errors"
	"fmt"
)

var ErrRuntimeIdentityPublicationUnavailable = errors.New("extensions: runtime identity publication boundary is unavailable")

// RuntimeIdentityPublicationMutation is an exact Host-owned Identity Registry
// mutation. Rollback must use artifact CAS and must never overwrite a newer
// runtime publication.
type RuntimeIdentityPublicationMutation interface {
	Rollback() error
}

// RuntimeIdentityPublicationBoundary keeps Models independent from the
// production Identity Registry and process-owning Manager. Identity-declaring
// plugins without Lifecycle V2 use this boundary after runtime start and before
// effective public/auth provider availability can observe them.
type RuntimeIdentityPublicationBoundary interface {
	PublishRuntimeIdentity(context.Context, Extension, int64, int64) (RuntimeIdentityPublicationMutation, error)
	QuarantineRuntimeIdentity(context.Context, Extension, int64, int64) (RuntimeIdentityPublicationMutation, error)
}

// BindRuntimeIdentityPublications is late-bound by production bootstrap after
// the shared Manager and Identity Registry have been constructed.
func (s *Service) BindRuntimeIdentityPublications(boundary RuntimeIdentityPublicationBoundary) *Service {
	if s == nil {
		return nil
	}
	s.identityPublications = boundary
	return s
}

func hasRuntimeIdentityPublication(manifest Manifest) bool {
	return manifest.Identity != nil || len(manifest.PermissionDefinitions) > 0
}

func (s *Service) publishLegacyRuntimeIdentity(
	ctx context.Context,
	enabled Extension,
	actorUserID int64,
	auditEventID int64,
) (RuntimeIdentityPublicationMutation, error) {
	if !hasRuntimeIdentityPublication(enabled.Manifest) {
		return nil, nil
	}
	if s == nil || s.identityPublications == nil {
		return nil, nil
	}
	if auditEventID <= 0 {
		return nil, ErrRuntimeIdentityPublicationUnavailable
	}
	return s.identityPublications.PublishRuntimeIdentity(ctx, enabled, actorUserID, auditEventID)
}

func (s *Service) quarantineLegacyRuntimeIdentity(
	ctx context.Context,
	extension Extension,
	actorUserID int64,
	auditEventID int64,
) (RuntimeIdentityPublicationMutation, error) {
	if !hasRuntimeIdentityPublication(extension.Manifest) {
		return nil, nil
	}
	if s == nil || s.identityPublications == nil {
		return nil, nil
	}
	if auditEventID <= 0 {
		return nil, ErrRuntimeIdentityPublicationUnavailable
	}
	return s.identityPublications.QuarantineRuntimeIdentity(ctx, extension, actorUserID, auditEventID)
}

func (s *Service) compensateLegacyIdentityEnable(
	ctx context.Context,
	enabled Extension,
	assetMutation exactAssetMutation,
	queryMutation RuntimeQueryPublicationMutation,
	cacheMutation RuntimeCachePublicationMutation,
	actorUserID int64,
	cause error,
) error {
	errs := []error{cause}
	if s.pageRegistry != nil {
		s.pageRegistry.ClearExtension(enabled.ID)
	}
	if cacheMutation != nil {
		if err := cacheMutation.Rollback(); err != nil {
			errs = append(errs, fmt.Errorf("restore cache publication: %w", err))
		}
	}
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
	if _, err := s.disableLegacyPluginState(ctx, enabled, actorUserID); err != nil {
		errs = append(errs, fmt.Errorf("disable extension: %w", err))
	}
	if err := s.rollbackExactAssetMutation(assetMutation); err != nil {
		errs = append(errs, fmt.Errorf("restore asset publication: %w", err))
	}
	return errors.Join(errs...)
}

func (s *Service) compensateLegacyIdentityDisable(
	assetMutation exactAssetMutation,
	queryMutation RuntimeQueryPublicationMutation,
	cacheMutation RuntimeCachePublicationMutation,
	identityMutation RuntimeIdentityPublicationMutation,
	cause error,
) error {
	errs := []error{cause}
	if err := s.rollbackExactAssetMutation(assetMutation); err != nil {
		errs = append(errs, fmt.Errorf("restore asset publication: %w", err))
	}
	if cacheMutation != nil {
		if err := cacheMutation.Rollback(); err != nil {
			errs = append(errs, fmt.Errorf("restore cache publication and runtime admission: %w", err))
		}
	}
	if queryMutation != nil {
		if err := queryMutation.Rollback(); err != nil {
			errs = append(errs, fmt.Errorf("restore query publication and runtime admission: %w", err))
		}
	}
	if identityMutation != nil {
		if err := identityMutation.Rollback(); err != nil {
			errs = append(errs, fmt.Errorf("restore identity publication: %w", err))
		}
	}
	return errors.Join(errs...)
}
