package extensions

import (
	"context"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

type ExtensionFrontendLifecycle interface {
	Frontend(context.Context, identity.Actor, string) (FrontendStatus, error)
}

func frontendRequiresWebRelease(status FrontendStatus) bool {
	switch status.TrustState {
	case FrontendTrustTrusted, FrontendTrustSourceTrusted, FrontendTrustRevocationPending:
		return true
	default:
		return false
	}
}

func (s *Service) EnableOperation(ctx context.Context, actor identity.Actor, id string) (ExtensionOperation, error) {
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return ExtensionOperation{}, err
	}
	if extension.Type == TypePlugin && extension.Manifest.Frontend.Admin != nil && s.frontendLifecycle != nil && s.webReleaseLifecycle != nil {
		status, err := s.frontendLifecycle.Frontend(ctx, actor, extension.ID)
		if err != nil {
			return ExtensionOperation{}, err
		}
		if frontendRequiresWebRelease(status) {
			// 需要 Web Release 时同时要求 plugin + release，避免仅有插件权即可触发发布。
			if err := s.verifyLifecyclePermissionAndPackage(ctx, actor, extension); err != nil {
				return ExtensionOperation{}, err
			}
			if !canManageReleases(actor) {
				return ExtensionOperation{}, identity.ErrPermissionDenied
			}
			queued, err := s.webReleaseLifecycle.PlanAndQueue(ctx, QueueWebReleaseInput{
				Plan:    PlanWebReleaseInput{TriggerKind: WebReleaseTriggerPluginEnable, TriggerExtensionID: extension.ID, RequestedBy: actor.ID, ReloadMode: WebReleaseReloadPrompt},
				Effects: []WebReleaseEffectInput{{ExtensionID: extension.ID, PreviousStatus: extension.Status, TargetStatus: StatusEnabled}},
			})
			if err != nil {
				return ExtensionOperation{}, err
			}
			summary := webReleaseSummary(queued.Release)
			decorated := s.decorateRuntime(ctx, extension)
			// 排队瞬间列表可能尚未读到 DB 行，用本次 summary 保证前端立刻有进度。
			decorated.WebRelease = summary
			return ExtensionOperation{Extension: decorated, Frontend: &status, WebRelease: summary, Queued: true}, nil
		}
	}
	enabled, err := s.Enable(ctx, actor, extension.ID)
	return ExtensionOperation{Extension: enabled}, err
}

func (s *Service) DisableOperation(ctx context.Context, actor identity.Actor, id string) (ExtensionOperation, error) {
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return ExtensionOperation{}, err
	}
	if extension.Type == TypePlugin && extension.Manifest.Frontend.Admin != nil && s.frontendLifecycle != nil && s.webReleaseLifecycle != nil {
		status, err := s.frontendLifecycle.Frontend(ctx, actor, extension.ID)
		if err != nil {
			return ExtensionOperation{}, err
		}
		if frontendRequiresWebRelease(status) {
			if !canManageReleases(actor) {
				return ExtensionOperation{}, identity.ErrPermissionDenied
			}
			queued, err := s.webReleaseLifecycle.PlanAndQueue(ctx, QueueWebReleaseInput{
				Plan:    PlanWebReleaseInput{TriggerKind: WebReleaseTriggerPluginDisable, TriggerExtensionID: extension.ID, RequestedBy: actor.ID, ReloadMode: WebReleaseReloadForce},
				Effects: []WebReleaseEffectInput{{ExtensionID: extension.ID, PreviousStatus: extension.Status, TargetStatus: StatusDisabled}},
			})
			if err != nil {
				return ExtensionOperation{}, err
			}
			summary := webReleaseSummary(queued.Release)
			decorated := s.decorateRuntime(ctx, extension)
			decorated.WebRelease = summary
			return ExtensionOperation{Extension: decorated, Frontend: &status, WebRelease: summary, Queued: true}, nil
		}
	}
	disabled, err := s.Disable(ctx, actor, extension.ID)
	return ExtensionOperation{Extension: disabled}, err
}

func (s *Service) ActivateThemeOperation(ctx context.Context, actor identity.Actor, id string) (ExtensionOperation, error) {
	extension, err := s.store.Get(ctx, normalizeID(id))
	if err != nil {
		return ExtensionOperation{}, err
	}
	if extension.Type != TypeTheme || s.webReleaseLifecycle == nil {
		active, err := s.ActivateTheme(ctx, actor, extension.ID)
		queued := active.ThemeRelease != nil && active.ThemeRelease.Status == ThemeReleaseQueued
		return ExtensionOperation{Extension: active, Queued: queued}, err
	}
	if err := s.verifyLifecyclePermissionAndPackage(ctx, actor, extension); err != nil {
		return ExtensionOperation{}, err
	}
	effects := []WebReleaseEffectInput{{ExtensionID: extension.ID, PreviousStatus: extension.Status, TargetStatus: StatusEnabled}}
	if current, activeErr := s.store.ActiveTheme(ctx); activeErr == nil && current.ID != extension.ID {
		effects = append(effects, WebReleaseEffectInput{ExtensionID: current.ID, PreviousStatus: current.Status, TargetStatus: StatusDisabled})
	}
	queued, err := s.webReleaseLifecycle.PlanAndQueue(ctx, QueueWebReleaseInput{
		Plan:    PlanWebReleaseInput{TriggerKind: WebReleaseTriggerTheme, TriggerExtensionID: extension.ID, TargetThemeID: extension.ID, RequestedBy: actor.ID, ReloadMode: WebReleaseReloadForce},
		Effects: effects,
	})
	if err != nil {
		return ExtensionOperation{}, err
	}
	return ExtensionOperation{Extension: extension, WebRelease: webReleaseSummary(queued.Release), Queued: true}, nil
}

func (s *Service) verifyLifecyclePermissionAndPackage(ctx context.Context, actor identity.Actor, extension Extension) error {
	// 主题激活走 theme.manage；插件生命周期走 plugin.manage。
	ok := canManagePlugins(actor)
	if extension.Type == TypeTheme {
		ok = canManageThemes(actor)
	}
	if !ok {
		return identity.ErrPermissionDenied
	}
	if err := s.verifyExtension(ctx, extension); err != nil {
		s.recordEnableFailure(ctx, actor, extension.ID, err)
		return err
	}
	return nil
}
