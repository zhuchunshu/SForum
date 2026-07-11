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
