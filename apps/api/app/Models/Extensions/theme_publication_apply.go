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
	switch {
	case publication.CoreReplacementsApproved && publication.ApprovalSource == ThemeApprovalSourceHost:
		// Host 批准只可能来自受保护内置默认主题（store 层已强制），
		// 绑定写 host 来源而不是伪造审批用户。
		err = s.pageRegistry.RegisterThemePackageReplacingHostApproved(
			ctx, target, publication.SourceThemeID,
		)
	case publication.CoreReplacementsApproved:
		err = s.pageRegistry.RegisterThemePackageReplacingApproved(
			ctx, target, publication.SourceThemeID, publication.ActorUserID,
		)
	default:
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

// EnsureDefaultThemeActive 是首次安装、启动修复与恢复路径共用的默认主题激活入口。
// 受保护内置默认主题由 Host 批准核心页替换；其它主题仍走人工审批。
func (s *ThemeService) EnsureDefaultThemeActive(ctx context.Context) (Extension, error) {
	active, err := s.store.ActiveTheme(ctx)
	if err == nil && active.ID == DefaultThemeID && active.Source == SourceBuiltin {
		return active, nil
	}
	defaultTheme, getErr := s.store.Get(ctx, DefaultThemeID)
	if getErr != nil {
		if err != nil {
			return Extension{}, err
		}
		return Extension{}, getErr
	}
	if defaultTheme.Type != TypeTheme || defaultTheme.Source != SourceBuiltin {
		return Extension{}, ErrInvalidManifest
	}
	result, err := s.store.ActivateTheme(ctx, DefaultThemeID)
	if err != nil {
		return Extension{}, err
	}
	if result.Publication.CoreReplacementsApproved &&
		result.Publication.ApprovalSource == ThemeApprovalSourceHost {
		// 首次安装/启动修复没有人类 actor：Host 批准自己的受保护默认主题，
		// 用一条事件让运维在扩展日志里能看到这次自动批准。
		_, _ = s.store.CreateEvent(ctx, EventInput{
			ExtensionID: DefaultThemeID,
			Action:      EventThemeActivated,
			Message:     "host-approved protected built-in default theme (first install or startup repair)",
		})
	}
	return result.Extension, nil
}
