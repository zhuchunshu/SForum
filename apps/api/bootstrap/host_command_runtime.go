package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	attachments "github.com/zhuchunshu/sforum/apps/api/app/Models/Attachments"
	moderation "github.com/zhuchunshu/sforum/apps/api/app/Models/Moderation"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

type protocolV2AttachmentStatusMutator struct {
	store *attachments.PostgresStore
}

type protocolV2CommandRuntimeBinder func(*hostapi.Gateway) error

func (m protocolV2AttachmentStatusMutator) MutateProtocolV2AttachmentStatus(
	ctx context.Context,
	tx pgx.Tx,
	attachmentID int64,
	status string,
) (hostapi.ProtocolV2AttachmentStatusResult, error) {
	if m.store == nil {
		return hostapi.ProtocolV2AttachmentStatusResult{}, fmt.Errorf("attachment status store is unavailable")
	}
	attachment, err := m.store.UpdateStatusTx(ctx, tx, attachmentID, status, false)
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, attachments.ErrAttachmentNotFound) {
		return hostapi.ProtocolV2AttachmentStatusResult{}, hostapi.ErrProtocolV2AttachmentNotFound
	}
	if err != nil {
		return hostapi.ProtocolV2AttachmentStatusResult{}, err
	}
	return hostapi.ProtocolV2AttachmentStatusResult{
		ID: attachment.ID, Status: attachment.Status,
		ReferenceCount: attachment.ReferenceCount, UpdatedAt: attachment.UpdatedAt,
	}, nil
}

func bindPostgresProtocolV2CommandRuntime(
	gateway *hostapi.Gateway,
	pool *pgxpool.Pool,
	jobs *supportjobs.Dispatcher,
	moderationStore *moderation.PostgresStore,
	attachmentStore *attachments.PostgresStore,
) error {
	if gateway == nil {
		return fmt.Errorf("Host Command gateway is required")
	}
	delegations, err := hostapi.NewProtocolV2ActorDelegationAuthority()
	if err != nil {
		return fmt.Errorf("create Host Command actor delegation authority: %w", err)
	}
	runtime, err := hostapi.NewPostgresProtocolV2CommandRuntime(hostapi.PostgresProtocolV2CommandRuntimeConfig{
		Pool: pool, ActorDelegations: delegations, Jobs: jobs, Moderation: moderationStore,
		AttachmentStatuses: protocolV2AttachmentStatusMutator{store: attachmentStore},
	})
	if err != nil {
		return fmt.Errorf("create Host Command runtime: %w", err)
	}
	if err := gateway.BindProtocolV2CommandRuntime(runtime); err != nil {
		return fmt.Errorf("bind Host Command runtime: %w", err)
	}
	return nil
}

func postgresProtocolV2CommandRuntimeBinder(
	pool *pgxpool.Pool,
	jobs *supportjobs.Dispatcher,
	moderationStore *moderation.PostgresStore,
	attachmentStore *attachments.PostgresStore,
) protocolV2CommandRuntimeBinder {
	return func(gateway *hostapi.Gateway) error {
		return bindPostgresProtocolV2CommandRuntime(gateway, pool, jobs, moderationStore, attachmentStore)
	}
}

var _ hostapi.ProtocolV2AttachmentStatusMutator = protocolV2AttachmentStatusMutator{}
