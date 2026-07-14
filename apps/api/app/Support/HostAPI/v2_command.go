package hostapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	hostv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/host/v2"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const protocolV2CommandRollbackTimeout = 5 * time.Second

type protocolV2CommandServer struct {
	hostv2.UnimplementedHostCommandServiceServer
	core *protocolV2Core
}

func (s *protocolV2CommandServer) Plan(ctx context.Context, request *hostv2.CommandRequest) (*hostv2.CommandPlan, error) {
	if s == nil || s.core == nil || s.core.commands == nil {
		return unavailableProtocolV2CommandPlan(request), nil
	}
	return s.core.commands.plan(ctx, request)
}

func (s *protocolV2CommandServer) Execute(ctx context.Context, request *hostv2.CommandRequest) (*hostv2.CommandResult, error) {
	if s == nil || s.core == nil || s.core.commands == nil {
		return &hostv2.CommandResult{
			Context: protocolV2ResponseContext(request.GetContext()),
			State:   hostv2.CommandState_COMMAND_STATE_REJECTED,
			Error:   protocolV2CommandUnavailable(),
		}, nil
	}
	return s.core.commands.execute(ctx, request)
}

// protocolV2CommandBackend keeps the idempotency ledger and audit event in the
// same pgx transaction as the command's business writes. LockIdempotency must
// serialize one extension/key pair until tx ends. Domain commands are bound by
// a separate Host-owned catalog.
type protocolV2CommandBackend interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	ResolveScope(ctx context.Context, tx pgx.Tx, requested protocolV2CommandScope) (protocolV2CommandScope, error)
	LockIdempotency(ctx context.Context, tx pgx.Tx, scope protocolV2CommandScope) (*protocolV2CommandReceipt, error)
	SaveResult(ctx context.Context, tx pgx.Tx, scope protocolV2CommandScope, receipt protocolV2CommandReceipt) error
	AppendAudit(ctx context.Context, tx pgx.Tx, event protocolV2CommandAudit) (string, error)
}

type protocolV2CommandScope struct {
	ExtensionID        string
	ExtensionVersionID int64
	ExtensionVersion   string
	PackageDigest      string
	AuthorityType      string
	TrustGrantID       int64
	CommandID          string
	CommandVersion     string
	IdempotencyKey     string
}

type protocolV2CommandReceipt struct {
	Fingerprint string
	Result      *hostv2.CommandResult
}

type protocolV2CommandAudit struct {
	Scope          protocolV2CommandScope
	ExtensionID    string
	ActorUserID    int64
	CommandID      string
	CommandVersion string
	TransactionID  string
	IdempotencyKey string
	Impact         []*hostv2.ImpactItem
}

type protocolV2CommandPreparation struct {
	Policy          []*hostv2.PolicyDecision
	Impact          []*hostv2.ImpactItem
	ProjectedResult *protocolv2.TypedDocument
}

type protocolV2CommandExecution struct {
	Output            *protocolv2.TypedDocument
	CommittedRevision string
}

type protocolV2CommandDefinition struct {
	ID                  string
	Version             string
	InputSchemaID       string
	InputSchemaVersion  string
	OutputSchemaID      string
	OutputSchemaVersion string
	// Preview is read-only guidance for Plan. Execute never trusts its result.
	Preview func(context.Context, *hostv2.CommandRequest) (*protocolV2CommandPreparation, error)
	// Prepare performs authoritative reads through tx immediately before writes.
	Prepare func(context.Context, pgx.Tx, *hostv2.CommandRequest) (*protocolV2CommandPreparation, error)
	Execute func(context.Context, pgx.Tx, *hostv2.CommandRequest, *protocolV2CommandPreparation) (*protocolV2CommandExecution, error)
}

type protocolV2CommandKey struct {
	id      string
	version string
}

// protocolV2CommandEngine is immutable after construction, so planning and
// execution can run concurrently without a mutable registration surface.
type protocolV2CommandEngine struct {
	backend     protocolV2CommandBackend
	definitions map[protocolV2CommandKey]protocolV2CommandDefinition
}

// ProtocolV2CommandRuntime is a sealed Host-owned command catalog. P5's
// PostgreSQL command package constructs one and binds it before broker boot.
type ProtocolV2CommandRuntime interface {
	commandEngine() *protocolV2CommandEngine
}

type protocolV2CommandRuntime struct {
	engine *protocolV2CommandEngine
}

func (r *protocolV2CommandRuntime) commandEngine() *protocolV2CommandEngine {
	if r == nil {
		return nil
	}
	return r.engine
}

func newProtocolV2CommandRuntime(engine *protocolV2CommandEngine) ProtocolV2CommandRuntime {
	return &protocolV2CommandRuntime{engine: engine}
}

func newProtocolV2CommandEngine(backend protocolV2CommandBackend, definitions ...protocolV2CommandDefinition) (*protocolV2CommandEngine, error) {
	registered := make(map[protocolV2CommandKey]protocolV2CommandDefinition, len(definitions))
	for _, definition := range definitions {
		definition.ID = strings.TrimSpace(definition.ID)
		definition.Version = strings.TrimSpace(definition.Version)
		key := protocolV2CommandKey{id: definition.ID, version: definition.Version}
		if definition.ID == "" || definition.Version == "" || definition.Preview == nil || definition.Prepare == nil || definition.Execute == nil {
			return nil, fmt.Errorf("hostapi: command id, version, preview, prepare, and execute are required")
		}
		if (definition.InputSchemaID == "") != (definition.InputSchemaVersion == "") ||
			(definition.OutputSchemaID == "") != (definition.OutputSchemaVersion == "") {
			return nil, fmt.Errorf("hostapi: command schema id and version must be declared together")
		}
		if _, exists := registered[key]; exists {
			return nil, fmt.Errorf("hostapi: duplicate command %s@%s", definition.ID, definition.Version)
		}
		registered[key] = definition
	}
	return &protocolV2CommandEngine{backend: backend, definitions: registered}, nil
}

func (e *protocolV2CommandEngine) plan(ctx context.Context, request *hostv2.CommandRequest) (*hostv2.CommandPlan, error) {
	definition, plan := e.definition(request)
	if plan.GetError() != nil {
		return plan, nil
	}
	preparation, err := definition.Preview(ctx, request)
	plan = finalizeProtocolV2CommandPlan(request, definition, preparation, err, plan)
	return plan, nil
}

func (e *protocolV2CommandEngine) definition(request *hostv2.CommandRequest) (protocolV2CommandDefinition, *hostv2.CommandPlan) {
	plan := &hostv2.CommandPlan{
		Context:        protocolV2ResponseContext(request.GetContext()),
		CommandId:      strings.TrimSpace(request.GetCommandId()),
		CommandVersion: strings.TrimSpace(request.GetCommandVersion()),
	}
	if e == nil {
		plan.Error = protocolV2CommandUnavailable()
		return protocolV2CommandDefinition{}, plan
	}
	definition, ok := e.definitions[protocolV2CommandKey{id: plan.GetCommandId(), version: plan.GetCommandVersion()}]
	if !ok {
		plan.Error = commandError(protocolv2.ErrorCode_ERROR_CODE_FAILED_PRECONDITION, "host.command_unsupported", "The command id or version is not registered.", false)
		return protocolV2CommandDefinition{}, plan
	}
	if strings.TrimSpace(request.GetContext().GetExtension().GetExtensionId()) == "" {
		plan.Error = commandError(protocolv2.ErrorCode_ERROR_CODE_INVALID_ARGUMENT, "host.command_extension_required", "The authenticated extension identity is required.", false)
		return definition, plan
	}
	if request.GetContext().GetActor() != nil {
		plan.Error = commandError(protocolv2.ErrorCode_ERROR_CODE_PERMISSION_DENIED, "host.command_actor_unattested", "Plugin-initiated Host Commands cannot supply an actor.", false)
		return definition, plan
	}
	if detail := validateProtocolV2CommandDocument(request.GetInput(), definition.InputSchemaID, definition.InputSchemaVersion, "input"); detail != nil {
		plan.Error = detail
		return definition, plan
	}
	return definition, plan
}

func finalizeProtocolV2CommandPlan(
	request *hostv2.CommandRequest,
	definition protocolV2CommandDefinition,
	preparation *protocolV2CommandPreparation,
	err error,
	plan *hostv2.CommandPlan,
) *hostv2.CommandPlan {
	if err != nil {
		plan.Error = protocolV2CommandErrorDetail(err, "host.command_plan_failed")
		return plan
	}
	if preparation == nil {
		plan.Error = commandError(protocolv2.ErrorCode_ERROR_CODE_INTERNAL, "host.command_plan_invalid", "The command returned no impact plan.", false)
		return plan
	}
	plan.Policy = cloneProtocolV2Policies(preparation.Policy)
	plan.Impact = cloneProtocolV2Impact(preparation.Impact)
	plan.ProjectedResult = cloneProtocolV2Document(preparation.ProjectedResult)
	if len(plan.GetPolicy()) == 0 {
		plan.Error = commandError(protocolv2.ErrorCode_ERROR_CODE_INTERNAL, "host.command_policy_required", "The command returned no Host policy decision.", false)
		return plan
	}
	if detail := validateProtocolV2CommandDocument(plan.GetProjectedResult(), definition.OutputSchemaID, definition.OutputSchemaVersion, "projected result"); detail != nil {
		plan.Error = detail
		return plan
	}
	for _, decision := range plan.GetPolicy() {
		if decision == nil || strings.TrimSpace(decision.GetPolicyId()) == "" {
			plan.Error = commandError(protocolv2.ErrorCode_ERROR_CODE_INTERNAL, "host.command_policy_invalid", "The command returned an invalid policy decision.", false)
			return plan
		}
		if !decision.GetAllowed() {
			plan.Error = commandError(protocolv2.ErrorCode_ERROR_CODE_PERMISSION_DENIED, "host.command_policy_denied", "Host policy denied the command.", false)
		}
	}
	plan.PlanId = protocolV2CommandPlanID(request, plan)
	return plan
}

func (e *protocolV2CommandEngine) execute(ctx context.Context, request *hostv2.CommandRequest) (*hostv2.CommandResult, error) {
	definition, plan := e.definition(request)
	result := &hostv2.CommandResult{Context: protocolV2ResponseContext(request.GetContext())}
	if plan.GetError() != nil {
		result.State = hostv2.CommandState_COMMAND_STATE_REJECTED
		result.Error = cloneProtocolV2CommandError(plan.GetError())
		return result, nil
	}
	if request.GetDryRun() {
		result.State = hostv2.CommandState_COMMAND_STATE_REJECTED
		result.Error = commandError(protocolv2.ErrorCode_ERROR_CODE_FAILED_PRECONDITION, "host.command_dry_run_requires_plan", "Use HostCommandService.Plan for dry-run impact data.", false)
		return result, nil
	}
	idempotencyKey, detail := protocolV2CommandIdempotencyKey(request, true)
	if detail != nil {
		result.State = hostv2.CommandState_COMMAND_STATE_REJECTED
		result.Error = detail
		return result, nil
	}
	if e.backend == nil {
		result.State = hostv2.CommandState_COMMAND_STATE_REJECTED
		result.Error = protocolV2CommandUnavailable()
		return result, nil
	}
	fingerprint, err := protocolV2CommandFingerprint(request)
	if err != nil {
		result.State = hostv2.CommandState_COMMAND_STATE_REJECTED
		result.Error = protocolV2CommandErrorDetail(err, "host.command_fingerprint_failed")
		return result, nil
	}
	scope := protocolV2CommandScope{
		ExtensionID: strings.TrimSpace(request.GetContext().GetExtension().GetExtensionId()),
		CommandID:   definition.ID, CommandVersion: definition.Version, IdempotencyKey: idempotencyKey,
	}
	tx, err := e.backend.Begin(ctx)
	if err != nil {
		result.State = hostv2.CommandState_COMMAND_STATE_REJECTED
		result.Error = commandError(protocolv2.ErrorCode_ERROR_CODE_UNAVAILABLE, "host.command_transaction_unavailable", "The Host command transaction could not be started.", true)
		return result, nil
	}

	resolvedScope, err := e.backend.ResolveScope(ctx, tx, scope)
	if err != nil {
		rollbackCtx, cancelRollback := context.WithTimeout(context.WithoutCancel(ctx), protocolV2CommandRollbackTimeout)
		defer cancelRollback()
		if rollbackErr := tx.Rollback(rollbackCtx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return nil, status.Error(codes.Internal, "Host command identity rollback failed; transaction outcome is unknown")
		}
		result.State = hostv2.CommandState_COMMAND_STATE_REJECTED
		result.Error = protocolV2CommandErrorDetail(err, "host.command_identity_invalid")
		return result, nil
	}
	committed, txnErr := e.executeInTransaction(ctx, tx, resolvedScope, fingerprint, definition, request)
	if txnErr != nil {
		rollbackCtx, cancelRollback := context.WithTimeout(context.WithoutCancel(ctx), protocolV2CommandRollbackTimeout)
		defer cancelRollback()
		if rollbackErr := tx.Rollback(rollbackCtx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return nil, status.Error(codes.Internal, "Host command rollback failed; transaction outcome is unknown")
		}
		result.State = hostv2.CommandState_COMMAND_STATE_ROLLED_BACK
		result.Error = protocolV2CommandErrorDetail(txnErr, "host.command_rolled_back")
		return result, nil
	}
	if err := tx.Commit(ctx); err != nil {
		if errors.Is(err, pgx.ErrTxCommitRollback) {
			result.State = hostv2.CommandState_COMMAND_STATE_ROLLED_BACK
			result.Error = commandError(protocolv2.ErrorCode_ERROR_CODE_INTERNAL, "host.command_commit_rolled_back", "The database rejected the commit and rolled the command back.", true)
			return result, nil
		}
		return nil, status.Error(codes.Internal, "Host command commit failed; transaction outcome is unknown")
	}
	committed.Context = result.GetContext()
	return committed, nil
}

func (e *protocolV2CommandEngine) executeInTransaction(
	ctx context.Context,
	tx pgx.Tx,
	scope protocolV2CommandScope,
	fingerprint string,
	definition protocolV2CommandDefinition,
	request *hostv2.CommandRequest,
) (*hostv2.CommandResult, error) {
	receipt, err := e.backend.LockIdempotency(ctx, tx, scope)
	if err != nil {
		return nil, fmt.Errorf("lock idempotency key: %w", err)
	}
	if receipt != nil {
		if receipt.Fingerprint != fingerprint || receipt.Result == nil {
			return nil, newProtocolV2CommandError(protocolv2.ErrorCode_ERROR_CODE_CONFLICT, "host.command_idempotency_conflict", "The idempotency key was already used for a different command request.", false)
		}
		replayed := proto.Clone(receipt.Result).(*hostv2.CommandResult)
		replayed.State = hostv2.CommandState_COMMAND_STATE_REPLAYED
		return replayed, nil
	}
	preparation, err := definition.Prepare(ctx, tx, request)
	if err != nil {
		return nil, fmt.Errorf("prepare command: %w", err)
	}
	plan := finalizeProtocolV2CommandPlan(request, definition, preparation, nil, &hostv2.CommandPlan{
		CommandId: definition.ID, CommandVersion: definition.Version,
	})
	if plan.GetError() != nil {
		return nil, &protocolV2CommandError{detail: cloneProtocolV2CommandError(plan.GetError())}
	}

	execution, err := definition.Execute(ctx, tx, request, preparation)
	if err != nil {
		return nil, err
	}
	if execution == nil {
		return nil, newProtocolV2CommandError(protocolv2.ErrorCode_ERROR_CODE_INTERNAL, "host.command_result_invalid", "The command returned no result.", false)
	}
	if detail := validateProtocolV2CommandDocument(execution.Output, definition.OutputSchemaID, definition.OutputSchemaVersion, "output"); detail != nil {
		return nil, &protocolV2CommandError{detail: detail}
	}
	transactionID, err := newProtocolV2CommandID("txn")
	if err != nil {
		return nil, fmt.Errorf("create transaction id: %w", err)
	}
	auditID, err := e.backend.AppendAudit(ctx, tx, protocolV2CommandAudit{
		Scope:       scope,
		ExtensionID: scope.ExtensionID,
		// Plugin -> Host calls currently have no Host-attested actor channel.
		ActorUserID: 0,
		CommandID:   definition.ID, CommandVersion: definition.Version,
		TransactionID: transactionID, IdempotencyKey: scope.IdempotencyKey,
		Impact: cloneProtocolV2Impact(preparation.Impact),
	})
	if err != nil {
		return nil, fmt.Errorf("append command audit: %w", err)
	}
	if strings.TrimSpace(auditID) == "" {
		return nil, fmt.Errorf("append command audit: empty event id")
	}
	committed := &hostv2.CommandResult{
		State:             hostv2.CommandState_COMMAND_STATE_COMMITTED,
		TransactionId:     transactionID,
		AuditEventId:      auditID,
		CommittedRevision: execution.CommittedRevision,
		Output:            cloneProtocolV2Document(execution.Output),
	}
	if err := e.backend.SaveResult(ctx, tx, scope, protocolV2CommandReceipt{Fingerprint: fingerprint, Result: proto.Clone(committed).(*hostv2.CommandResult)}); err != nil {
		return nil, fmt.Errorf("save command result: %w", err)
	}
	return committed, nil
}
