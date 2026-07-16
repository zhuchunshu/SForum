package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	audit "github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

const (
	defaultRouteFailureQueueSize    = 128
	defaultRouteFailureAuditTimeout = 2 * time.Second
)

type ExactRuntimeIncidentQuarantiner interface {
	QuarantineRuntimeInstance(
		extensionsruntime.RuntimeInstanceArtifactIdentity,
		error,
	) (extensionsruntime.RuntimeAdmissionSnapshot, error)
}

type RouteFailureRecorder struct {
	runtime   ExactRuntimeIncidentQuarantiner
	auditor   audit.Writer
	logger    *slog.Logger
	timeout   time.Duration
	queue     chan recordedRouteFailure
	stop      chan struct{}
	done      chan struct{}
	enqueueMu sync.RWMutex
	stopOnce  sync.Once
	closed    bool
	dropped   atomic.Uint64
}

type recordedRouteFailure struct {
	event      routes.RouteCommittedAfterFailure
	quarantine string
}

func NewRouteFailureRecorder(
	runtime ExactRuntimeIncidentQuarantiner,
	auditor audit.Writer,
	logger *slog.Logger,
) (*RouteFailureRecorder, error) {
	return newRouteFailureRecorder(runtime, auditor, logger, defaultRouteFailureQueueSize, defaultRouteFailureAuditTimeout)
}

func newRouteFailureRecorder(
	runtime ExactRuntimeIncidentQuarantiner,
	auditor audit.Writer,
	logger *slog.Logger,
	queueSize int,
	timeout time.Duration,
) (*RouteFailureRecorder, error) {
	if runtime == nil || auditor == nil || queueSize <= 0 || timeout <= 0 {
		return nil, fmt.Errorf("route failure recorder is not configured")
	}
	if logger == nil {
		logger = slog.Default()
	}
	recorder := &RouteFailureRecorder{
		runtime: runtime, auditor: auditor, logger: logger, timeout: timeout,
		queue: make(chan recordedRouteFailure, queueSize), stop: make(chan struct{}), done: make(chan struct{}),
	}
	go recorder.run()
	return recorder, nil
}

func (r *RouteFailureRecorder) RecordCommittedAfterFailure(
	_ context.Context,
	event routes.RouteCommittedAfterFailure,
) {
	if r == nil {
		return
	}
	r.enqueueMu.RLock()
	defer r.enqueueMu.RUnlock()
	if r.closed {
		return
	}
	item := recordedRouteFailure{event: event, quarantine: "not_required"}
	if routeFailureRequiresQuarantine(event) {
		exact := extensionsruntime.RuntimeInstanceArtifactIdentity{
			RuntimeInstanceIdentity: extensionsruntime.RuntimeInstanceIdentity{
				ExtensionID: event.Artifact.ExtensionID, InstanceID: event.Artifact.RuntimeInstanceID,
			},
			ExtensionVersion: event.Artifact.ExtensionVersion,
			ArtifactDigest:   event.Artifact.PackageDigest,
		}
		cause := fmt.Errorf("%w: %s", extensionsruntime.ErrRuntimeRouteIncident, event.FailureCode)
		_, err := r.runtime.QuarantineRuntimeInstance(exact, cause)
		switch {
		case err == nil:
			item.quarantine = "quarantined"
		case errors.Is(err, extensionsruntime.ErrRuntimeInstanceNotFound):
			item.quarantine = "stale_missing"
		case errors.Is(err, extensionsruntime.ErrRuntimeInstanceConflict):
			item.quarantine = "stale_artifact"
		default:
			item.quarantine = "failed"
			r.logger.Error("route failure runtime quarantine failed",
				"extension_id", event.Artifact.ExtensionID,
				"runtime_instance_id", event.Artifact.RuntimeInstanceID,
				"failure_code", event.FailureCode,
				"error", err,
			)
		}
	} else if event.FailureCode == routes.RouteFailureTransportFailed {
		item.quarantine = "execution_not_observed"
	}
	select {
	case r.queue <- item:
	default:
		r.dropped.Add(1)
		r.logger.Error("route failure audit queue is full",
			"extension_id", event.Artifact.ExtensionID,
			"runtime_instance_id", event.Artifact.RuntimeInstanceID,
			"failure_code", event.FailureCode,
			"quarantine", item.quarantine,
		)
	}
}

func (r *RouteFailureRecorder) Dropped() uint64 {
	if r == nil {
		return 0
	}
	return r.dropped.Load()
}

func (r *RouteFailureRecorder) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf("route failure recorder close context is nil")
	}
	r.enqueueMu.Lock()
	if !r.closed {
		r.closed = true
		r.stopOnce.Do(func() { close(r.stop) })
	}
	r.enqueueMu.Unlock()
	select {
	case <-r.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *RouteFailureRecorder) run() {
	defer close(r.done)
	for {
		select {
		case item := <-r.queue:
			r.appendAudit(item)
		case <-r.stop:
			for {
				select {
				case item := <-r.queue:
					r.appendAudit(item)
				default:
					return
				}
			}
		}
	}
}

func (r *RouteFailureRecorder) appendAudit(item recordedRouteFailure) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	event := item.event
	err := r.auditor.Append(ctx, audit.Event{
		ActorUserID: event.ActorID,
		Action:      audit.ActionRouteCommittedAfterFailure,
		Metadata: map[string]any{
			"revision": event.Revision, "stepIndex": event.StepIndex,
			"phase": event.Phase, "action": event.Action,
			"routeId": event.RouteID, "contractVersion": event.ContractVersion,
			"method": event.Method, "pathSignature": event.PathSignature,
			"failureCode": event.FailureCode, "runtimeExecutionObserved": event.RuntimeExecutionObserved,
			"responseStatus": event.ResponseStatus, "commitState": event.CommitState,
			"extensionId": event.Artifact.ExtensionID, "extensionVersion": event.Artifact.ExtensionVersion,
			"packageDigest": event.Artifact.PackageDigest, "runtimeInstanceId": event.Artifact.RuntimeInstanceID,
			"quarantineResult": item.quarantine,
		},
	})
	if err != nil {
		r.logger.Error("record route committed-after failure audit failed",
			"extension_id", event.Artifact.ExtensionID,
			"runtime_instance_id", event.Artifact.RuntimeInstanceID,
			"failure_code", event.FailureCode,
			"error", err,
		)
	}
}

func routeFailureRequiresQuarantine(event routes.RouteCommittedAfterFailure) bool {
	return event.FailureCode == routes.RouteFailureResponseSchemaRejected ||
		event.FailureCode == routes.RouteFailureTransportFailed && event.RuntimeExecutionObserved
}

var _ routes.RouteFailureSink = (*RouteFailureRecorder)(nil)
