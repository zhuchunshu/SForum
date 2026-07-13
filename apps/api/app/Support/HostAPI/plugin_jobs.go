package hostapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/riverqueue/river"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

// PluginJobKind 是 River 上的宿主统一 job kind；真实插件 kind 放在 payload 内。
const PluginJobKind = "extension.plugin_job"

// PluginJobArgs 入队载荷：仅允许插件 manifest 声明过的 JobName。
type PluginJobArgs struct {
	EnvelopeVersion      int            `json:"envelopeVersion,omitempty"`
	ExtensionID          string         `json:"extensionId"`
	ExtensionVersion     string         `json:"extensionVersion,omitempty"`
	ArtifactDigest       string         `json:"artifactDigest,omitempty"`
	TrustGrantID         string         `json:"trustGrantId,omitempty"`
	JobName              string         `json:"kind"`
	JobContractVersion   string         `json:"jobContractVersion,omitempty"`
	PayloadSchemaID      string         `json:"payloadSchemaId,omitempty"`
	PayloadSchemaVersion string         `json:"payloadSchemaVersion,omitempty"`
	Payload              map[string]any `json:"payload,omitempty"`
	EnqueuedAt           time.Time      `json:"enqueuedAt"`
}

// Kind 返回 River 注册用的固定 kind（非插件声明的 JobName）。
func (PluginJobArgs) Kind() string { return PluginJobKind }

func (a PluginJobArgs) Contract() supportjobs.PluginJobContract {
	return supportjobs.PluginJobContract{
		ExtensionID: a.ExtensionID, ExtensionVersion: a.ExtensionVersion, ArtifactDigest: a.ArtifactDigest,
		JobName: a.JobName, JobContract: a.JobContractVersion,
		PayloadSchemaID: a.PayloadSchemaID, PayloadSchemaVersion: a.PayloadSchemaVersion,
	}
}

func (a PluginJobArgs) validEnvelope() bool {
	return a.EnvelopeVersion == supportjobs.PluginJobEnvelopeVersion &&
		a.Contract().Valid() && a.TrustGrantID != "" && !a.EnqueuedAt.IsZero()
}

// RiverJobEnqueuer 把插件 job 写入 River default 队列。
type RiverJobEnqueuer struct {
	Dispatcher *supportjobs.Dispatcher
}

func (e *RiverJobEnqueuer) EnqueuePluginJob(ctx context.Context, extensionID, kind string, payload map[string]any) error {
	if e == nil || e.Dispatcher == nil {
		return fmt.Errorf("%w: job dispatcher missing", ErrUnavailable)
	}
	args := PluginJobArgs{
		ExtensionID: extensionID,
		JobName:     kind,
		Payload:     payload,
		EnqueuedAt:  time.Now().UTC(),
	}
	_, err := e.Dispatcher.Enqueue(ctx, args, supportjobs.EnqueueOptions{
		Queue: supportjobs.QueueDefault,
	})
	return err
}

// EnqueueVersionedPluginJob persists the exact runtime and manifest contract
// that authorized a protocol-v2 enqueue request.
func (e *RiverJobEnqueuer) EnqueueVersionedPluginJob(
	ctx context.Context,
	contract supportjobs.PluginJobContract,
	trustGrantID string,
	payload map[string]any,
) error {
	if e == nil || e.Dispatcher == nil {
		return fmt.Errorf("%w: job dispatcher missing", ErrUnavailable)
	}
	if !contract.Valid() || trustGrantID == "" {
		return fmt.Errorf("%w: exact plugin job contract is required", ErrInvalidRequest)
	}
	args := PluginJobArgs{
		EnvelopeVersion: supportjobs.PluginJobEnvelopeVersion,
		ExtensionID:     contract.ExtensionID, ExtensionVersion: contract.ExtensionVersion,
		ArtifactDigest: contract.ArtifactDigest, TrustGrantID: trustGrantID,
		JobName: contract.JobName, JobContractVersion: contract.JobContract,
		PayloadSchemaID: contract.PayloadSchemaID, PayloadSchemaVersion: contract.PayloadSchemaVersion,
		Payload: payload, EnqueuedAt: time.Now().UTC(),
	}
	_, err := e.Dispatcher.Enqueue(ctx, args, supportjobs.EnqueueOptions{Queue: supportjobs.QueueDefault})
	return err
}

type VersionedPluginJobEnqueuer interface {
	EnqueueVersionedPluginJob(context.Context, supportjobs.PluginJobContract, string, map[string]any) error
}

type PluginJobRuntimeContract struct {
	Contract     supportjobs.PluginJobContract
	TrustGrantID string
}

type PluginJobRuntimeResolver interface {
	ResolvePluginJobRuntime(context.Context, string, string) (PluginJobRuntimeContract, error)
}

type PluginJobExecutor interface {
	ExecutePluginJob(context.Context, supportjobs.PluginJobInvocation) error
}

type PluginJobCompatibilityError struct {
	Decision supportjobs.PluginJobDecision
}

func (e *PluginJobCompatibilityError) Error() string {
	if e == nil {
		return "plugin job contract is incompatible"
	}
	return fmt.Sprintf("plugin job %s: %s", e.Decision.Action, e.Decision.Reason)
}

// PluginJobWorker resolves the live exact contract before any plugin code is
// invoked. Legacy or incompatible rows are permanently cancelled.
type PluginJobWorker struct {
	river.WorkerDefaults[PluginJobArgs]
	Resolver PluginJobRuntimeResolver
	Executor PluginJobExecutor
}

func (w *PluginJobWorker) Work(ctx context.Context, job *river.Job[PluginJobArgs]) error {
	if job == nil || !job.Args.validEnvelope() {
		return cancelPluginJob(supportjobs.PluginJobDecision{
			Action: supportjobs.PluginJobCancel, Reason: "plugin_job.envelope_invalid",
		})
	}
	if job.Args.Payload != nil {
		if _, err := json.Marshal(job.Args.Payload); err != nil {
			return err
		}
	}
	if w == nil || w.Resolver == nil {
		return errors.New("plugin job runtime resolver is not configured")
	}
	target, err := w.Resolver.ResolvePluginJobRuntime(ctx, job.Args.ExtensionID, job.Args.JobName)
	if err != nil {
		return fmt.Errorf("resolve plugin job runtime: %w", err)
	}
	decision := supportjobs.DecidePluginJobExecution(job.Args.Contract(), target.Contract, nil)
	if decision.Action != supportjobs.PluginJobExecute {
		return cancelPluginJob(decision)
	}
	if target.TrustGrantID == "" || target.TrustGrantID != job.Args.TrustGrantID {
		return cancelPluginJob(supportjobs.PluginJobDecision{
			Action: supportjobs.PluginJobCancel, Reason: "plugin_job.trust_grant_stale",
		})
	}
	if w.Executor == nil {
		return errors.New("plugin job runtime executor is not configured")
	}
	invocation := supportjobs.PluginJobInvocation{
		Contract: job.Args.Contract(), TrustGrant: job.Args.TrustGrantID,
		Payload: job.Args.Payload, EnqueuedAt: job.Args.EnqueuedAt,
	}
	if job.JobRow != nil {
		invocation.JobID = job.ID
		invocation.Attempt = job.Attempt
	}
	return w.Executor.ExecutePluginJob(ctx, invocation)
}

func cancelPluginJob(decision supportjobs.PluginJobDecision) error {
	return river.JobCancel(&PluginJobCompatibilityError{Decision: decision})
}
