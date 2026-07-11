package notificationjobs

import (
	"context"
	"fmt"
	"strconv"

	"github.com/riverqueue/river"
	notifications "github.com/zhuchunshu/sforum/apps/api/app/Models/Notifications"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

type DeliverMailArgs struct {
	DeliveryID int64 `json:"delivery_id" river:"unique"`
}

func (DeliverMailArgs) Kind() string { return "mail.deliver" }
func (DeliverMailArgs) EnqueueOptions() supportjobs.EnqueueOptions {
	return supportjobs.EnqueueOptions{Queue: supportjobs.QueueMail, MaxAttempts: 5, Unique: river.UniqueOpts{ByArgs: true}}
}

type DeliveryStore interface {
	GetDelivery(context.Context, int64) (notifications.MailDelivery, error)
	UpdateDelivery(context.Context, notifications.DeliveryUpdate) error
}
type ProviderResolver interface {
	Selected(context.Context) (extensionsruntime.MailProviderSelection, bool, error)
}
type ProviderSender interface {
	SendMail(context.Context, string, extensionsruntime.MailProviderRequest) (extensionsruntime.MailProviderResponse, error)
}

type DeliverMailWorker struct {
	river.WorkerDefaults[DeliverMailArgs]
	Store     DeliveryStore
	Providers ProviderResolver
	Sender    ProviderSender
}

func (w *DeliverMailWorker) Work(ctx context.Context, job *river.Job[DeliverMailArgs]) error {
	if w.Store == nil || w.Providers == nil {
		return fmt.Errorf("mail delivery worker is not configured")
	}
	delivery, err := w.Store.GetDelivery(ctx, job.Args.DeliveryID)
	if err != nil {
		return err
	}
	if delivery.Status == notifications.DeliverySent || delivery.Status == notifications.DeliverySkipped {
		return nil
	}
	selection, ok, err := w.Providers.Selected(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return w.Store.UpdateDelivery(ctx, notifications.DeliveryUpdate{ID: delivery.ID, Status: notifications.DeliverySkipped, Reason: "provider_unavailable", AttemptCount: delivery.AttemptCount + 1})
	}
	if w.Sender == nil {
		return fmt.Errorf("mail provider sender is not configured")
	}
	response, err := w.Sender.SendMail(ctx, selection.ExtensionID, extensionsruntime.MailProviderRequest{DeliveryID: strconv.FormatInt(delivery.ID, 10), CorrelationID: delivery.CorrelationID, To: []string{delivery.Recipient}, Subject: delivery.TemplateKey, TextBody: string(delivery.TemplateData)})
	if err != nil {
		return err
	}
	update := notifications.DeliveryUpdate{ID: delivery.ID, ExtensionID: selection.ExtensionID, AttemptCount: delivery.AttemptCount + 1, Reason: response.Reason, ErrorSummary: response.Message}
	if response.OK {
		update.Status = notifications.DeliverySent
		return w.Store.UpdateDelivery(ctx, update)
	}
	if response.Classification == "temporary" {
		return fmt.Errorf("temporary mail provider failure: %s", response.Reason)
	}
	update.Status = notifications.DeliveryFailed
	return w.Store.UpdateDelivery(ctx, update)
}

func Register(registry *supportjobs.Registry, worker *DeliverMailWorker) {
	registry.Add(func(workers *river.Workers) error { return river.AddWorkerSafely[DeliverMailArgs](workers, worker) })
}
