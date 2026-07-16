package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	queryregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/QueryRegistry"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
)

var errProductionQueryRegistryRuntimeStale = errors.New("bootstrap: Query Registry exact runtime is stale")

type productionQueryRegistry struct {
	Execution *queryregistry.ExecutionRuntime
	Service   *hostapi.ProtocolV2QueryRegistryService
}

// productionQueryActorAuthority never trusts a delegation's cached actor
// projection. Every release fence reloads the Host actor and its effective
// permissions from the authoritative Identity store.
type productionQueryActorAuthority struct {
	store identity.ActorStore
}

func (a *productionQueryActorAuthority) ResolveProtocolV2QueryActor(
	ctx context.Context,
	actorUserID int64,
) (hostapi.ProtocolV2QueryActorProjection, error) {
	actor, err := a.loadActiveActor(ctx, actorUserID)
	if err != nil {
		return hostapi.ProtocolV2QueryActorProjection{}, err
	}
	return productionQueryActorProjection(actor)
}

func (a *productionQueryActorAuthority) AuthorizeProtocolV2QueryActor(
	ctx context.Context,
	actorUserID int64,
	claim queryregistry.PermissionClaim,
) (hostapi.ProtocolV2QueryActorProjection, error) {
	actor, err := a.loadActiveActor(ctx, actorUserID)
	if err != nil {
		return hostapi.ProtocolV2QueryActorProjection{}, err
	}
	switch claim.PermissionPolicy {
	case queryregistry.PermissionPolicyPublic, queryregistry.PermissionPolicyLogin:
		// Delegated Query calls are authenticated. Reloading an active actor is
		// the live Host authority check for the two built-in policies.
	default:
		if !actor.Can(claim.PermissionPolicy) {
			return hostapi.ProtocolV2QueryActorProjection{}, hostapi.ErrProtocolV2QueryActorDenied
		}
	}
	return productionQueryActorProjection(actor)
}

func (a *productionQueryActorAuthority) loadActiveActor(ctx context.Context, actorUserID int64) (identity.Actor, error) {
	if a == nil || a.store == nil || ctx == nil || actorUserID <= 0 {
		return identity.Actor{}, hostapi.ErrProtocolV2QueryActorDenied
	}
	if err := ctx.Err(); err != nil {
		return identity.Actor{}, err
	}
	actor, err := a.store.LoadActor(ctx, actorUserID)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return identity.Actor{}, ctxErr
	}
	if err != nil || actor.ID != actorUserID || !actor.IsActive() {
		return identity.Actor{}, errors.Join(hostapi.ErrProtocolV2QueryActorDenied, err)
	}
	return actor, nil
}

func productionQueryActorProjection(actor identity.Actor) (hostapi.ProtocolV2QueryActorProjection, error) {
	createdAt := ""
	if !actor.CreatedAt.IsZero() {
		createdAt = actor.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	actorFingerprint, err := productionQueryFingerprint("actor", struct {
		ID        int64               `json:"id"`
		Status    identity.UserStatus `json:"status"`
		CreatedAt string              `json:"createdAt,omitempty"`
	}{ID: actor.ID, Status: actor.Status, CreatedAt: createdAt})
	if err != nil {
		return hostapi.ProtocolV2QueryActorProjection{}, hostapi.ErrProtocolV2QueryActorDenied
	}
	permissions := make([]string, 0, len(actor.Permissions))
	for permission, allowed := range actor.Permissions {
		if allowed {
			permissions = append(permissions, permission)
		}
	}
	policyFingerprint, err := productionQueryFingerprint("policy", struct {
		Status      identity.UserStatus `json:"status"`
		Roles       []string            `json:"roles"`
		Permissions []string            `json:"permissions"`
	}{
		Status: actor.Status, Roles: productionQuerySortedUnique(actor.RoleKeys),
		Permissions: productionQuerySortedUnique(permissions),
	})
	if err != nil {
		return hostapi.ProtocolV2QueryActorProjection{}, hostapi.ErrProtocolV2QueryActorDenied
	}
	return hostapi.ProtocolV2QueryActorProjection{
		ActorUserID: actor.ID, Authenticated: true,
		ActorFingerprint: actorFingerprint, PolicyFingerprint: policyFingerprint,
	}, nil
}

func productionQueryFingerprint(namespace string, value any) (string, error) {
	document, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(document)
	return namespace + ":" + hex.EncodeToString(digest[:]), nil
}

func productionQuerySortedUnique(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	if len(result) < 2 {
		return result
	}
	write := 1
	for read := 1; read < len(result); read++ {
		if result[read] == result[write-1] {
			continue
		}
		result[write] = result[read]
		write++
	}
	return result[:write]
}

type productionQueryRuntimeAcquirer func(
	context.Context,
	string,
	extensionsruntime.RuntimeCallClass,
) (extensionsruntime.RuntimeInstanceSnapshot, *extensionsruntime.RuntimeAdmissionLease, error)

// productionQueryRuntimeAdmission provides both the caller fence and the
// executable artifact fence from one exact-runtime linearization point.
type productionQueryRuntimeAdmission struct {
	acquire productionQueryRuntimeAcquirer
}

func newProductionQueryRuntimeAdmission(runtime *extensionsruntime.Manager) *productionQueryRuntimeAdmission {
	if runtime == nil {
		return nil
	}
	return &productionQueryRuntimeAdmission{acquire: runtime.AcquireActiveRuntimeCall}
}

func (a *productionQueryRuntimeAdmission) AuthorizeProtocolV2QueryCaller(
	ctx context.Context,
	identity *protocolv2.ExtensionIdentity,
) error {
	if a == nil || a.acquire == nil || ctx == nil || !productionQueryCallerIdentityValid(identity) {
		return errProductionQueryRegistryRuntimeStale
	}
	snapshot, lease, err := a.acquire(ctx, identity.GetExtensionId(), extensionsruntime.RuntimeCallHost)
	if err != nil {
		return errors.Join(errProductionQueryRegistryRuntimeStale, err)
	}
	if lease == nil {
		return errProductionQueryRegistryRuntimeStale
	}
	defer lease.Release()
	if err := productionQueryRuntimeSnapshotMatches(
		ctx, lease, snapshot, identity.GetExtensionId(), identity.GetExtensionVersion(),
		identity.GetArtifactDigest(), identity.GetInstanceId(),
	); err != nil {
		return err
	}
	return nil
}

func (a *productionQueryRuntimeAdmission) AcquireQueryExecution(
	ctx context.Context,
	artifact queryregistry.Artifact,
) (func(), error) {
	// ExecutionRuntime itself recognizes the unforgeable Core seal and bypasses
	// this adapter. Never trust the public Core boolean at this outer boundary.
	if a == nil || a.acquire == nil || ctx == nil || artifact.Core ||
		strings.TrimSpace(artifact.ExtensionID) == "" || strings.TrimSpace(artifact.ExtensionVersion) == "" ||
		strings.TrimSpace(artifact.PackageDigest) == "" || artifact.VersionID <= 0 ||
		strings.TrimSpace(artifact.RuntimeInstanceID) == "" {
		return nil, errProductionQueryRegistryRuntimeStale
	}
	snapshot, lease, err := a.acquire(ctx, artifact.ExtensionID, extensionsruntime.RuntimeCallProvider)
	if err != nil {
		return nil, errors.Join(errProductionQueryRegistryRuntimeStale, err)
	}
	if lease == nil {
		return nil, errProductionQueryRegistryRuntimeStale
	}
	if err := productionQueryRuntimeSnapshotMatches(
		ctx, lease, snapshot, artifact.ExtensionID, artifact.ExtensionVersion,
		artifact.PackageDigest, artifact.RuntimeInstanceID,
	); err != nil {
		lease.Release()
		return nil, err
	}
	return lease.Release, nil
}

func productionQueryCallerIdentityValid(identity *protocolv2.ExtensionIdentity) bool {
	if identity == nil || identity.GetRuntimeEpoch() == 0 || strings.TrimSpace(identity.GetTrustGrantId()) == "" {
		return false
	}
	for _, value := range []string{
		identity.GetExtensionId(), identity.GetExtensionVersion(), identity.GetArtifactDigest(), identity.GetInstanceId(),
	} {
		if value == "" || value != strings.TrimSpace(value) {
			return false
		}
	}
	return true
}

func productionQueryRuntimeSnapshotMatches(
	ctx context.Context,
	lease *extensionsruntime.RuntimeAdmissionLease,
	snapshot extensionsruntime.RuntimeInstanceSnapshot,
	extensionID string,
	extensionVersion string,
	artifactDigest string,
	instanceID string,
) error {
	if ctx == nil || lease == nil || lease.Context == nil {
		return errProductionQueryRegistryRuntimeStale
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := lease.Context.Err(); err != nil {
		return errors.Join(errProductionQueryRegistryRuntimeStale, err)
	}
	if !snapshot.Active || snapshot.Admission.Draining || snapshot.Admission.Forced ||
		snapshot.Identity.ExtensionID != extensionID || snapshot.Identity.InstanceID != instanceID ||
		snapshot.ExtensionVersion != extensionVersion || snapshot.ArtifactDigest != artifactDigest {
		return errProductionQueryRegistryRuntimeStale
	}
	return nil
}

func bindProductionQueryRegistry(
	registry *queryregistry.Registry,
	catalog *hostapi.QueryRegistryCoreCatalog,
	stableRuntime hostapi.ProtocolV2QueryRuntime,
	actors identity.ActorStore,
	runtime *extensionsruntime.Manager,
	gateway *hostapi.Gateway,
	trace hostapi.QueryTraceSink,
) (*productionQueryRegistry, error) {
	if registry == nil || catalog == nil || stableRuntime == nil || actors == nil || runtime == nil || gateway == nil || trace == nil {
		return nil, fmt.Errorf("bootstrap: production Query Registry dependency unavailable")
	}
	providers, err := hostapi.NewProtocolV2QueryRegistryProviderResolver(stableRuntime, catalog.Bindings())
	if err != nil {
		return nil, fmt.Errorf("create Query Registry provider resolver: %w", err)
	}
	schemas, err := queryregistry.NewJSONResultSchemaCatalog(catalog.Schemas())
	if err != nil {
		return nil, fmt.Errorf("create Query Registry result schema catalog: %w", err)
	}
	admission := newProductionQueryRuntimeAdmission(runtime)
	execution, err := queryregistry.NewExecutionRuntime(queryregistry.ExecutionConfig{
		Registry: registry, Providers: providers, Admission: admission, Schemas: schemas,
		Trace: hostapi.NewQueryRegistryTraceAdapter(trace),
	})
	if err != nil {
		return nil, fmt.Errorf("create Query Registry execution runtime: %w", err)
	}
	service, err := hostapi.NewProtocolV2QueryRegistryService(
		registry, execution, &productionQueryActorAuthority{store: actors}, admission,
	)
	if err != nil {
		return nil, fmt.Errorf("create Query Registry Protocol V2 service: %w", err)
	}
	if err := gateway.BindProtocolV2QueryRegistryService(service); err != nil {
		return nil, fmt.Errorf("bind Query Registry Protocol V2 service: %w", err)
	}
	return &productionQueryRegistry{Execution: execution, Service: service}, nil
}
