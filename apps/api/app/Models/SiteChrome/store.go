package sitechrome

import "context"

// Store 持久化导航、友情链接与公告。
type Store interface {
	ListNavItems(ctx context.Context, enabledOnly bool) ([]NavItem, error)
	CreateNavItem(ctx context.Context, input CreateNavItemInput) (NavItem, error)
	UpdateNavItem(ctx context.Context, input UpdateNavItemInput) (NavItem, error)
	DeleteNavItem(ctx context.Context, id int64) error

	ListFriendLinks(ctx context.Context, enabledOnly bool) ([]FriendLink, error)
	CreateFriendLink(ctx context.Context, input CreateFriendLinkInput) (FriendLink, error)
	UpdateFriendLink(ctx context.Context, input UpdateFriendLinkInput) (FriendLink, error)
	DeleteFriendLink(ctx context.Context, id int64) error

	// ListAnnouncements：admin 用 enabledOnly=false；公开 active 用 enabledOnly=true 且按时间窗过滤。
	ListAnnouncements(ctx context.Context, enabledOnly bool, activeNow bool) ([]Announcement, error)
	CreateAnnouncement(ctx context.Context, input CreateAnnouncementInput) (Announcement, error)
	UpdateAnnouncement(ctx context.Context, input UpdateAnnouncementInput) (Announcement, error)
	DeleteAnnouncement(ctx context.Context, id int64) error
}
