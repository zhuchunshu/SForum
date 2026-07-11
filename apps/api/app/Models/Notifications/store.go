package notifications

import "context"

type Store interface {
	List(context.Context, ListInput) (Page, error)
	UnreadCount(context.Context, int64) (int64, error)
	MarkRead(context.Context, int64, int64) error
	MarkAllRead(context.Context, int64) (int64, error)
	GetDelivery(context.Context, int64) (MailDelivery, error)
	UpdateDelivery(context.Context, DeliveryUpdate) error
}
