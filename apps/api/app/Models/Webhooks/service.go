package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

type TxEnqueuer interface {
	EnqueueTx(context.Context, pgx.Tx, river.JobArgs, supportjobs.EnqueueOptions) (*rivertype.JobInsertResult, error)
}

// Service 管理端点配置，并把 core observe 事件扇出为 webhook 投递。
type Service struct {
	store Store
	pool  *pgxpool.Pool
	jobs  TxEnqueuer
}

func NewService(store Store, pool *pgxpool.Pool, jobs TxEnqueuer) *Service {
	return &Service{store: store, pool: pool, jobs: jobs}
}

func (s *Service) ListEndpoints(ctx context.Context, actor identity.Actor) ([]Endpoint, error) {
	if err := requireManage(actor); err != nil {
		return nil, err
	}
	records, err := s.store.ListEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Endpoint, 0, len(records))
	for _, record := range records {
		out = append(out, PublicEndpoint(record))
	}
	return out, nil
}

func (s *Service) CreateEndpoint(ctx context.Context, actor identity.Actor, input CreateEndpointInput) (Endpoint, error) {
	if err := requireManage(actor); err != nil {
		return Endpoint{}, err
	}
	if err := validateEndpointInput(input.Name, input.TargetURL); err != nil {
		return Endpoint{}, err
	}
	record, err := s.store.CreateEndpoint(ctx, input)
	if err != nil {
		return Endpoint{}, err
	}
	return PublicEndpoint(record), nil
}

func (s *Service) UpdateEndpoint(ctx context.Context, actor identity.Actor, id int64, input UpdateEndpointInput) (Endpoint, error) {
	if err := requireManage(actor); err != nil {
		return Endpoint{}, err
	}
	if input.Name != nil || input.TargetURL != nil {
		name := ""
		if input.Name != nil {
			name = *input.Name
		}
		target := ""
		if input.TargetURL != nil {
			target = *input.TargetURL
		}
		// 部分更新时只校验提供的字段。
		if input.Name != nil && strings.TrimSpace(name) == "" {
			return Endpoint{}, ErrInvalidEndpoint
		}
		if input.TargetURL != nil {
			if err := validateURL(target); err != nil {
				return Endpoint{}, err
			}
		}
	}
	record, err := s.store.UpdateEndpoint(ctx, id, input)
	if err != nil {
		return Endpoint{}, err
	}
	return PublicEndpoint(record), nil
}

func (s *Service) DeleteEndpoint(ctx context.Context, actor identity.Actor, id int64) error {
	if err := requireManage(actor); err != nil {
		return err
	}
	return s.store.DeleteEndpoint(ctx, id)
}

func (s *Service) ListDeliveries(ctx context.Context, actor identity.Actor, endpointID int64, limit int) ([]Delivery, error) {
	if err := requireManage(actor); err != nil {
		return nil, err
	}
	return s.store.ListDeliveries(ctx, endpointID, limit)
}

// Fanout 将 observe 事件写入各订阅端点的投递记录并入队（异步，不阻塞业务）。
func (s *Service) Fanout(ctx context.Context, envelope appevents.Envelope) {
	if s == nil || s.store == nil || s.pool == nil || s.jobs == nil {
		return
	}
	if envelope.Name == "" {
		return
	}
	// 仅 observe 类事件对外投递；filter 是同步业务钩子。
	if envelope.Kind != "" && envelope.Kind != appevents.KindObserve {
		return
	}
	endpoints, err := s.store.ListEnabledEndpointsForEvent(ctx, envelope.Name)
	if err != nil || len(endpoints) == 0 {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"id":            envelope.ID,
		"name":          envelope.Name,
		"kind":          envelope.Kind,
		"actorUserId":   envelope.ActorUserID,
		"resourceType":  envelope.ResourceType,
		"resourceId":    envelope.ResourceID,
		"correlationId": envelope.CorrelationID,
		"payload":       envelope.Payload,
		"occurredAt":    envelope.OccurredAt,
	})
	if err != nil {
		return
	}
	for _, endpoint := range endpoints {
		if err := s.enqueueDelivery(ctx, endpoint.ID, envelope, payload); err != nil {
			// 单端点失败不影响其它订阅方。
			continue
		}
	}
}

func (s *Service) enqueueDelivery(ctx context.Context, endpointID int64, envelope appevents.Envelope, payload []byte) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	delivery, err := s.store.CreateDeliveryTx(ctx, tx, CreateDeliveryInput{
		EndpointID:    endpointID,
		EventName:     envelope.Name,
		EventID:       envelope.ID,
		CorrelationID: envelope.CorrelationID,
		Payload:       payload,
	})
	if err != nil {
		return err
	}
	// 与 Jobs/Webhooks.DeliverArgs 同 kind/JSON，避免 Models→Jobs 循环依赖。
	args := deliverJobArgs{DeliveryID: delivery.ID}
	if _, err := s.jobs.EnqueueTx(ctx, tx, args, args.enqueueOptions()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type deliverJobArgs struct {
	DeliveryID int64 `json:"delivery_id" river:"unique"`
}

func (deliverJobArgs) Kind() string { return "webhook.deliver" }
func (deliverJobArgs) enqueueOptions() supportjobs.EnqueueOptions {
	return supportjobs.EnqueueOptions{
		Queue:       supportjobs.QueueDefault,
		MaxAttempts: DefaultMaxAttempts,
		Unique:      river.UniqueOpts{ByArgs: true},
	}
}

func requireManage(actor identity.Actor) error {
	if !actor.Can(identity.PermissionSettingsManage) && !actor.Can(identity.PermissionSettingsSiteManage) {
		return identity.ErrPermissionDenied
	}
	return nil
}

func validateEndpointInput(name, targetURL string) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidEndpoint
	}
	return validateURL(targetURL)
}

func validateURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ErrInvalidURL
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return ErrInvalidURL
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https", "http":
		return nil
	default:
		return ErrInvalidURL
	}
}

func FormatError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprint(err)
}
