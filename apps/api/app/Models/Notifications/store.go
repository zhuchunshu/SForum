package notifications

import (
	"context"

	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
)

type Store interface {
	List(context.Context, ListInput) (Page, error)
	UnreadCount(context.Context, int64) (int64, error)
	MarkRead(context.Context, int64, int64) error
	MarkAllRead(context.Context, int64) (int64, error)
	GetDelivery(context.Context, int64) (MailDelivery, error)
	UpdateDelivery(context.Context, DeliveryUpdate) error
	ListDeliveries(context.Context, int) ([]MailDelivery, error)
}

// CorePolicyStore is the compatibility projection used while /admin/mail/policy
// remains in API LTS. The V2 tables are still the authority.
type CorePolicyStore interface {
	NotificationPolicy(context.Context) (options.NotificationPolicy, error)
	UpdateCoreNotificationPolicy(context.Context, options.NotificationPolicy) error
	RestoreCoreNotificationPolicy(context.Context) error
}

type RecipientRevisionStore interface {
	RecipientRevision(context.Context, int64) (int64, error)
}

type RecipientRevisionWakeStore interface {
	SubscribeRevision(int64) (<-chan struct{}, func(), error)
}
