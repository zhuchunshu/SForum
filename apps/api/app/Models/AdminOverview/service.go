package adminoverview

import (
	"context"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

type Store interface {
	Snapshot(ctx context.Context, since time.Time) (StoreSnapshot, error)
}

type RuntimeProvider interface {
	Snapshot() RuntimeStats
}

type Option func(*Service)

type Service struct {
	store   Store
	runtime RuntimeProvider
	widgets DashboardWidgetProvider
	clock   func() time.Time
}

func NewService(store Store, runtime RuntimeProvider, options ...Option) *Service {
	service := &Service{
		store:   store,
		runtime: runtime,
		clock: func() time.Time {
			return time.Now().UTC()
		},
	}
	for _, option := range options {
		option(service)
	}
	return service
}

func WithClock(clock func() time.Time) Option {
	return func(service *Service) {
		if clock != nil {
			service.clock = clock
		}
	}
}

// WithDashboardWidgets 注入扩展仪表盘小部件（F4.3）。
func WithDashboardWidgets(provider DashboardWidgetProvider) Option {
	return func(service *Service) {
		service.widgets = provider
	}
}

func (s *Service) Overview(ctx context.Context, actor identity.Actor) (AdminOverview, error) {
	if !actor.Can(identity.PermissionAdminAccess) {
		return AdminOverview{}, identity.ErrPermissionDenied
	}

	now := s.clock().UTC()
	since := now.AddDate(0, 0, -(WindowDays - 1))
	snapshot, err := s.store.Snapshot(ctx, since)
	if err != nil {
		return AdminOverview{}, err
	}

	widgets, err := s.listWidgets(ctx)
	if err != nil {
		return AdminOverview{}, err
	}

	return AdminOverview{
		GeneratedAt:      now,
		WindowDays:       WindowDays,
		Runtime:          s.runtimeSnapshot(),
		Community:        snapshot.Community,
		Attachments:      snapshot.Attachments,
		Moderation:       snapshot.Moderation,
		Extensions:       snapshot.Extensions,
		Trends:           TrendStats{Days: snapshot.Trends},
		TopCategories:    snapshot.TopCategories,
		Actions:          overviewActions(snapshot),
		ExtensionWidgets: widgets,
	}, nil
}

func (s *Service) listWidgets(ctx context.Context) ([]ExtensionWidget, error) {
	if s == nil || s.widgets == nil {
		return nil, nil
	}
	return s.widgets.DashboardWidgets(ctx)
}

func (s *Service) runtimeSnapshot() RuntimeStats {
	if s.runtime == nil {
		return RuntimeStats{}
	}
	return s.runtime.Snapshot()
}

func overviewActions(snapshot StoreSnapshot) []OverviewAction {
	return []OverviewAction{
		{
			Key:      ActionModerationQueue,
			Count:    snapshot.Moderation.OpenCount + snapshot.Moderation.ReviewingCount,
			Severity: severityForCount(snapshot.Moderation.OpenCount+snapshot.Moderation.ReviewingCount, "warning"),
			Route:    "/moderation",
		},
		{
			Key:      ActionPendingTags,
			Count:    snapshot.Community.PendingTagCount,
			Severity: severityForCount(snapshot.Community.PendingTagCount, "warning"),
			Route:    "/forum/tags",
		},
		{
			Key:      ActionOrphanAttachments,
			Count:    snapshot.Attachments.OrphanCount,
			Severity: severityForCount(snapshot.Attachments.OrphanCount, "info"),
			Route:    "/attachments",
		},
		{
			Key:      ActionFailedExtensionEvents,
			Count:    snapshot.Extensions.FailedEventCount,
			Severity: severityForCount(snapshot.Extensions.FailedEventCount, "danger"),
			Route:    "/extensions/events",
		},
	}
}

func severityForCount(count int64, nonZero string) string {
	if count <= 0 {
		return "success"
	}
	return nonZero
}

type StaticRuntimeProvider struct {
	Stats RuntimeStats
}

func (p StaticRuntimeProvider) Snapshot() RuntimeStats {
	return p.Stats
}
