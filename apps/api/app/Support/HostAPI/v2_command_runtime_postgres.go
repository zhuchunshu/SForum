package hostapi

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	moderation "github.com/zhuchunshu/sforum/apps/api/app/Models/Moderation"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

type ProtocolV2IdentityAuthorityMutationGate interface {
	RunSessionPolicyMutation(context.Context, func() error) error
}

type PostgresProtocolV2CommandRuntimeConfig struct {
	Pool               *pgxpool.Pool
	ActorDelegations   *ProtocolV2ActorDelegationAuthority
	Jobs               *supportjobs.Dispatcher
	Moderation         *moderation.PostgresStore
	AttachmentStatuses ProtocolV2AttachmentStatusMutator
	IdentityAuthority  ProtocolV2IdentityAuthorityMutationGate
}

// NewPostgresProtocolV2CommandRuntime publishes the complete immutable domain
// catalog. Missing dependencies fail boot instead of silently omitting a
// command and creating different API/worker capabilities.
func NewPostgresProtocolV2CommandRuntime(config PostgresProtocolV2CommandRuntimeConfig) (ProtocolV2CommandRuntime, error) {
	if config.Pool == nil || config.ActorDelegations == nil || config.Jobs == nil ||
		config.Moderation == nil || config.AttachmentStatuses == nil {
		return nil, errors.New("hostapi: complete PostgreSQL Host Command runtime dependencies are required")
	}
	identityCommand := newProtocolV2IdentityUserStatusCommandDefinition()
	if config.IdentityAuthority != nil {
		identityCommand.RunAuthorityMutation = config.IdentityAuthority.RunSessionPolicyMutation
	}
	engine, err := newProtocolV2CommandEngineWithInvalidationJobs(
		NewPostgresProtocolV2HostCommandBackend(config.Pool),
		config.ActorDelegations,
		config.Jobs,
		identityCommand,
		newProtocolV2TopicVisibilityCommandDefinition(config.Pool, config.Jobs),
		newProtocolV2EntityMetaCommandDefinition(config.Pool),
		newProtocolV2ModerationCommandDefinition(config.Moderation, config.Jobs),
		newProtocolV2EntitlementCommandDefinition(config.Pool),
		newProtocolV2AttachmentStatusCommandDefinition(config.AttachmentStatuses),
		newProtocolV2ExtensionPluginDisableCommandDefinition(),
		newProtocolV2ExtensionSettingsResetCommandDefinition(),
	)
	if err != nil {
		return nil, err
	}
	return newProtocolV2CommandRuntime(engine), nil
}
