package extensions

import (
	"context"
	"errors"
)

var ErrThemeRuntimeApplyFailed = errors.New("extensions: theme runtime publication apply failed")

// ApplyThemeRuntimePublication converges only process-local Page/ThemeRuntime
// state. The desired database state and publication row were committed by the
// activation transaction before this method runs.
func (s *Service) ApplyThemeRuntimePublication(ctx context.Context, publication ThemeRuntimePublication) error {
	if s == nil || ctx == nil || s.pageRegistry == nil || !validThemeRuntimePublication(publication) {
		return ErrThemeRuntimeApplyFailed
	}
	s.themeActivationMu.Lock()
	defer s.themeActivationMu.Unlock()

	items, err := s.store.List(ctx)
	if err != nil {
		return errors.Join(ErrThemeRuntimeApplyFailed, err)
	}
	if publication.DesiredState == ThemeRuntimePublicationNone {
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
	if target.ID != DefaultThemeID {
		defaultTheme, defaultErr := s.store.Get(ctx, DefaultThemeID)
		if defaultErr != nil {
			return errors.Join(ErrThemeRuntimeApplyFailed, defaultErr)
		}
		if defaultErr = s.pageRegistry.RegisterDefaultThemeFallback(ctx, defaultTheme); defaultErr != nil {
			return errors.Join(ErrThemeRuntimeApplyFailed, defaultErr)
		}
	}
	for _, item := range items {
		if item.Type == TypeTheme && item.ID != target.ID && item.ID != publication.SourceThemeID {
			s.pageRegistry.ClearExtension(item.ID)
		}
	}
	if err := s.pageRegistry.PreflightThemePackage(ctx, target, publication.SourceThemeID); err != nil {
		return errors.Join(ErrThemeRuntimeApplyFailed, err)
	}
	if publication.CoreReplacementsApproved {
		err = s.pageRegistry.RegisterThemePackageReplacingApproved(
			ctx, target, publication.SourceThemeID, publication.ActorUserID,
		)
	} else {
		err = s.pageRegistry.RegisterThemePackageReplacing(ctx, target, publication.SourceThemeID)
	}
	if err != nil {
		return errors.Join(ErrThemeRuntimeApplyFailed, err)
	}
	return nil
}
