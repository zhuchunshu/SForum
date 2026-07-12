package webhooks

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Store 持久化端点与投递记录。
type Store interface {
	ListEndpoints(ctx context.Context) ([]EndpointRecord, error)
	GetEndpoint(ctx context.Context, id int64) (EndpointRecord, error)
	CreateEndpoint(ctx context.Context, input CreateEndpointInput) (EndpointRecord, error)
	UpdateEndpoint(ctx context.Context, id int64, input UpdateEndpointInput) (EndpointRecord, error)
	DeleteEndpoint(ctx context.Context, id int64) error

	CreateDeliveryTx(ctx context.Context, tx pgx.Tx, input CreateDeliveryInput) (Delivery, error)
	GetDelivery(ctx context.Context, id int64) (Delivery, error)
	UpdateDelivery(ctx context.Context, input DeliveryUpdate) error
	ListDeliveries(ctx context.Context, endpointID int64, limit int) ([]Delivery, error)
	ListEnabledEndpointsForEvent(ctx context.Context, eventName string) ([]EndpointRecord, error)
}
