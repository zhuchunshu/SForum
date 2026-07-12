package extensions

import (
	"context"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

type WebReleaseAdminStore interface {
	ListWebReleases(context.Context, WebReleaseListInput) (WebReleasePage, error)
	WebRelease(context.Context, int64) (WebReleaseDetail, error)
}

type WebReleaseCommander interface {
	// Rebuild 按当前激活主题与已启用可信插件重新规划 composition 并入队。
	PlanAndQueue(context.Context, QueueWebReleaseInput) (WebReleaseQueueResult, error)
	Retry(context.Context, int64, int64) (WebReleaseQueueResult, error)
	Rollback(context.Context, int64, int64) (WebReleaseQueueResult, error)
}

type WebReleaseOperation struct {
	WebRelease WebRelease `json:"webRelease"`
	Queued     bool       `json:"queued"`
}

type WebReleaseAdminService struct {
	store    WebReleaseAdminStore
	commands WebReleaseCommander
}

func NewWebReleaseAdminService(store WebReleaseAdminStore, commands WebReleaseCommander) *WebReleaseAdminService {
	return &WebReleaseAdminService{store: store, commands: commands}
}

func (s *WebReleaseAdminService) List(
	ctx context.Context,
	actor identity.Actor,
	input WebReleaseListInput,
) (WebReleasePage, error) {
	if !canManageReleases(actor) {
		return WebReleasePage{}, identity.ErrPermissionDenied
	}
	return s.store.ListWebReleases(ctx, input)
}

func (s *WebReleaseAdminService) Detail(ctx context.Context, actor identity.Actor, releaseID int64) (WebReleaseDetail, error) {
	if !canManageReleases(actor) {
		return WebReleaseDetail{}, identity.ErrPermissionDenied
	}
	return s.store.WebRelease(ctx, releaseID)
}

// Rebuild 手动触发一次 Web Release（例如内置主题 admin 前端更新后）。
// 不改插件/主题启用态，只按当前 composition 重新构建。
func (s *WebReleaseAdminService) Rebuild(ctx context.Context, actor identity.Actor) (WebReleaseOperation, error) {
	if !canManageReleases(actor) {
		return WebReleaseOperation{}, identity.ErrPermissionDenied
	}
	if s.commands == nil {
		return WebReleaseOperation{}, ErrFrontendTrustUnavailable
	}
	result, err := s.commands.PlanAndQueue(ctx, QueueWebReleaseInput{
		Plan: PlanWebReleaseInput{
			TriggerKind: WebReleaseTriggerRebuild,
			RequestedBy: actor.ID,
			// 管理端自定义组件更新后需要提示刷新；不强行 force 打断操作中页面。
			ReloadMode: WebReleaseReloadPrompt,
		},
	})
	if err != nil {
		return WebReleaseOperation{}, err
	}
	return WebReleaseOperation{WebRelease: result.Release, Queued: true}, nil
}

func (s *WebReleaseAdminService) Retry(ctx context.Context, actor identity.Actor, releaseID int64) (WebReleaseOperation, error) {
	if !canManageReleases(actor) {
		return WebReleaseOperation{}, identity.ErrPermissionDenied
	}
	result, err := s.commands.Retry(ctx, releaseID, actor.ID)
	if err != nil {
		return WebReleaseOperation{}, err
	}
	return WebReleaseOperation{WebRelease: result.Release, Queued: true}, nil
}

func (s *WebReleaseAdminService) Rollback(ctx context.Context, actor identity.Actor, releaseID int64) (WebReleaseOperation, error) {
	if !canManageReleases(actor) {
		return WebReleaseOperation{}, identity.ErrPermissionDenied
	}
	result, err := s.commands.Rollback(ctx, releaseID, actor.ID)
	if err != nil {
		return WebReleaseOperation{}, err
	}
	return WebReleaseOperation{WebRelease: result.Release, Queued: true}, nil
}
