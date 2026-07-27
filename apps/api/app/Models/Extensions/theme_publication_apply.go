package extensions

import (
	"context"
	"errors"
	"fmt"
)

var ErrThemeRuntimeApplyFailed = errors.New("extensions: theme runtime publication apply failed")

// ApplyThemeRuntimePublication converges process-local Page/ThemeRuntime and
// Component Registry state. The desired database state and publication row
// were committed by the activation transaction before this method runs.
func (s *ThemeService) ApplyThemeRuntimePublication(ctx context.Context, publication ThemeRuntimePublication) error {
	if s == nil || ctx == nil || s.pageRegistry == nil || !validThemeRuntimePublication(publication) {
		return ErrThemeRuntimeApplyFailed
	}
	s.themeActivationMu.Lock()
	defer s.themeActivationMu.Unlock()
	if s.themeRuntimeUnavailable {
		return errors.Join(ErrThemeRuntimeApplyFailed, ErrThemeRuntimeUnavailable)
	}
	s.assetPublicationMu.Lock()
	defer s.assetPublicationMu.Unlock()
	assetBefore := s.captureAssetPublicationSnapshot()

	items, err := s.store.List(ctx)
	if err != nil {
		return errors.Join(ErrThemeRuntimeApplyFailed, err)
	}
	source, err := s.themePublicationSource(ctx, publication)
	if err != nil {
		return errors.Join(ErrThemeRuntimeApplyFailed, err)
	}
	if publication.DesiredState == ThemeRuntimePublicationNone {
		if s.componentRegistry != nil {
			if err := s.componentRegistry.ValidateThemeTransition(nil, source); err != nil {
				return errors.Join(ErrThemeRuntimeApplyFailed, err)
			}
			if err := s.componentRegistry.PublishThemeTransition(nil, source, publication.Revision); err != nil {
				return errors.Join(ErrThemeRuntimeApplyFailed, err)
			}
		}
		if err := s.validateThemeAssetTransition(ctx, assetBefore, nil, source); err != nil {
			return errors.Join(ErrThemeRuntimeApplyFailed, err)
		}
		if _, err := s.publishThemeAssetTransition(ctx, assetBefore, nil, source); err != nil {
			if s.componentRegistry != nil {
				if rollbackErr := s.componentRegistry.RollbackThemeTransition(
					source, nil, publication.Revision,
				); rollbackErr != nil {
					err = errors.Join(err, fmt.Errorf("restore source component publication: %w", rollbackErr))
				}
			}
			return errors.Join(ErrThemeRuntimeApplyFailed, err)
		}
		for _, item := range items {
			if item.Type == TypeTheme {
				s.pageRegistry.ClearExtension(item.ID)
			}
		}
		return nil
	}

	target, err := s.store.Get(ctx, publication.ThemeID)
	if err != nil || target.Type != TypeTheme || target.Status != StatusEnabled ||
		target.Version != publication.ThemeVersion || target.PackageDigest != publication.PackageDigest {
		return errors.Join(ErrThemeRuntimeApplyFailed, err)
	}
	if err := s.verifyExtension(ctx, target); err != nil {
		return errors.Join(ErrThemeRuntimeApplyFailed, err)
	}
	if err := s.pageRegistry.PreflightThemePackage(ctx, target, publication.SourceThemeID); err != nil {
		return errors.Join(ErrThemeRuntimeApplyFailed, err)
	}
	if s.componentRegistry != nil {
		if err := s.componentRegistry.ValidateThemeTransition(&target, source); err != nil {
			return errors.Join(ErrThemeRuntimeApplyFailed, err)
		}
	}
	if err := s.validateThemeAssetTransition(ctx, assetBefore, &target, source); err != nil {
		return errors.Join(ErrThemeRuntimeApplyFailed, err)
	}
	if target.ID != DefaultThemeID {
		defaultTheme, defaultErr := s.store.Get(ctx, DefaultThemeID)
		if defaultErr != nil {
			return errors.Join(ErrThemeRuntimeApplyFailed, defaultErr)
		}
		if defaultErr = s.pageRegistry.RegisterDefaultThemeFallback(ctx, defaultTheme); defaultErr != nil {
			return errors.Join(ErrThemeRuntimeApplyFailed, defaultErr)
		}
	}
	componentPublished := false
	if s.componentRegistry != nil {
		if err := s.componentRegistry.PublishThemeTransition(&target, source, publication.Revision); err != nil {
			return errors.Join(ErrThemeRuntimeApplyFailed, err)
		}
		componentPublished = true
	}
	assetAfter, publishErr := s.publishThemeAssetTransition(ctx, assetBefore, &target, source)
	if publishErr != nil {
		if componentPublished {
			if rollbackErr := s.componentRegistry.RollbackThemeTransition(
				source, &target, publication.Revision,
			); rollbackErr != nil {
				publishErr = errors.Join(publishErr, fmt.Errorf("restore source component publication: %w", rollbackErr))
			}
		}
		return errors.Join(ErrThemeRuntimeApplyFailed, publishErr)
	}
	if publication.CoreReplacementsApproved {
		err = s.pageRegistry.RegisterThemePackageReplacingApproved(
			ctx, target, publication.SourceThemeID, publication.ActorUserID,
		)
	} else {
		err = s.pageRegistry.RegisterThemePackageReplacing(ctx, target, publication.SourceThemeID)
	}
	if err != nil {
		if componentPublished {
			if rollbackErr := s.componentRegistry.RollbackThemeTransition(
				source, &target, publication.Revision,
			); rollbackErr != nil {
				err = errors.Join(err, fmt.Errorf("restore source component publication: %w", rollbackErr))
			}
		}
		if _, rollbackErr := s.rollbackThemeAssetTransition(
			ctx, assetAfter, source, &target,
		); rollbackErr != nil {
			err = errors.Join(err, fmt.Errorf("restore source asset publication: %w", rollbackErr))
		}
		return errors.Join(ErrThemeRuntimeApplyFailed, err)
	}
	for _, item := range items {
		if item.Type == TypeTheme && item.ID != target.ID && item.ID != publication.SourceThemeID {
			s.pageRegistry.ClearExtension(item.ID)
		}
	}
	return nil
}

func (s *serviceCore) themePublicationSource(
	ctx context.Context,
	publication ThemeRuntimePublication,
) (*Extension, error) {
	if publication.SourceThemeID == "" {
		return nil, nil
	}
	expected := Extension{
		ID: publication.SourceThemeID, Type: TypeTheme,
		Version: publication.SourceThemeVersion, PackageDigest: publication.SourcePackageDigest,
	}
	base, err := s.store.Get(ctx, publication.SourceThemeID)
	if errors.Is(err, ErrExtensionNotFound) {
		// The inactive source package may already be uninstalled on a fresh
		// node. Its durable exact tuple remains sufficient to fence a stale
		// process-local publication; no source package code is loaded here.
		return &expected, nil
	}
	if err != nil {
		return nil, err
	}
	if sameThemeExactArtifact(base, expected) {
		value := base
		return &value, nil
	}
	versions, ok := s.store.(ExactExtensionVersionRepository)
	if !ok {
		return nil, ErrThemePublicationConflict
	}
	version, err := versions.GetExtensionVersion(ctx, ExactExtensionVersionInput{
		ExtensionID:   publication.SourceThemeID,
		Version:       publication.SourceThemeVersion,
		PackageDigest: publication.SourcePackageDigest,
	})
	if err != nil {
		return nil, err
	}
	exact := extensionFromExactVersion(base, version)
	if !sameThemeExactArtifact(exact, expected) {
		return nil, ErrThemePublicationConflict
	}
	return &exact, nil
}
