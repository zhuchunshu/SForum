package http

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	audit "github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

func TestRouteFailureRecorderQuarantinesOnlyRuntimeIncidentsAndAuditsDetachedMetadata(t *testing.T) {
	runtime := &recordingRouteIncidentRuntime{}
	auditor := &recordingRouteFailureAuditor{}
	recorder, err := newRouteFailureRecorder(runtime, auditor, discardRouteFailureLogger(), 8, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	events := []routes.RouteCommittedAfterFailure{
		routeFailureRecorderEvent(routes.RouteFailureGuardDenied, false),
		routeFailureRecorderEvent(routes.RouteFailureRequestSchemaRejected, false),
		routeFailureRecorderEvent(routes.RouteFailureTransportFailed, false),
		routeFailureRecorderEvent(routes.RouteFailureTransportFailed, true),
		routeFailureRecorderEvent(routes.RouteFailureResponseSchemaRejected, false),
		routeFailureRecorderEvent(routes.RouteFailureResponseSchemaRejected, true),
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, event := range events {
		recorder.RecordCommittedAfterFailure(cancelled, event)
	}
	closeCtx, closeCancel := context.WithTimeout(context.Background(), time.Second)
	defer closeCancel()
	if err := recorder.Close(closeCtx); err != nil {
		t.Fatal(err)
	}
	if err := recorder.Close(closeCtx); err != nil {
		t.Fatal(err)
	}

	runtime.mu.Lock()
	quarantines := append([]recordedRouteIncident(nil), runtime.calls...)
	runtime.mu.Unlock()
	if len(quarantines) != 2 {
		t.Fatalf("quarantines=%#v", quarantines)
	}
	for _, call := range quarantines {
		if call.exact.RuntimeInstanceIdentity != (extensionsruntime.RuntimeInstanceIdentity{
			ExtensionID: "failure.plugin", InstanceID: "runtime-1",
		}) || call.exact.ExtensionVersion != "1.0.0" ||
			call.exact.ArtifactDigest != strings.Repeat("a", 64) ||
			!errors.Is(call.cause, extensionsruntime.ErrRuntimeRouteIncident) {
			t.Fatalf("quarantine=%#v", call)
		}
	}

	auditor.mu.Lock()
	audits := append([]audit.Event(nil), auditor.events...)
	auditor.mu.Unlock()
	if len(audits) != len(events) {
		t.Fatalf("audits=%#v", audits)
	}
	wantQuarantine := []string{
		"not_required", "not_required", "execution_not_observed", "quarantined", "not_required", "quarantined",
	}
	for index, recorded := range audits {
		if recorded.Action != audit.ActionRouteCommittedAfterFailure || recorded.ActorUserID != 42 ||
			recorded.Metadata["failureCode"] != events[index].FailureCode ||
			recorded.Metadata["invocationStage"] != routes.InvocationStageResponse ||
			recorded.Metadata["quarantineResult"] != wantQuarantine[index] ||
			recorded.Metadata["packageDigest"] != strings.Repeat("a", 64) {
			t.Fatalf("audit[%d]=%#v", index, recorded)
		}
		for _, forbidden := range []string{"path", "query", "headers", "cookie", "authorization", "body", "error"} {
			if _, exists := recorded.Metadata[forbidden]; exists {
				t.Fatalf("audit metadata contains %q: %#v", forbidden, recorded.Metadata)
			}
		}
	}
}

func TestRouteFailureRecorderRejectsMissingInvalidOrMismatchedInvocationStage(t *testing.T) {
	runtime := &recordingRouteIncidentRuntime{}
	auditor := &recordingRouteFailureAuditor{}
	recorder, err := newRouteFailureRecorder(runtime, auditor, discardRouteFailureLogger(), 2, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	for _, stage := range []routes.InvocationStage{"", "forged"} {
		event := routeFailureRecorderEvent(routes.RouteFailureTransportFailed, true)
		event.InvocationStage = stage
		recorder.RecordCommittedAfterFailure(context.Background(), event)
	}
	for _, stage := range []routes.InvocationStage{routes.InvocationStageRequest, routes.InvocationStageHandler} {
		event := routeFailureRecorderEvent(routes.RouteFailureTransportFailed, true)
		event.InvocationStage = stage
		recorder.RecordCommittedAfterFailure(context.Background(), event)
	}
	mismatched := routeFailureRecorderEvent(routes.RouteFailureTransportFailed, true)
	mismatched.Phase = routes.RoutePhaseBefore
	recorder.RecordCommittedAfterFailure(context.Background(), mismatched)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := recorder.Close(ctx); err != nil {
		t.Fatal(err)
	}
	runtime.mu.Lock()
	quarantineCalls := len(runtime.calls)
	runtime.mu.Unlock()
	auditor.mu.Lock()
	auditCalls := len(auditor.events)
	auditor.mu.Unlock()
	if quarantineCalls != 0 || auditCalls != 0 || recorder.Dropped() != 0 {
		t.Fatalf("invalid stage reached recorder: quarantines=%d audits=%d dropped=%d", quarantineCalls, auditCalls, recorder.Dropped())
	}
}

func TestRouteFailureRecorderBoundsAuditBackpressureWithoutSkippingQuarantine(t *testing.T) {
	runtime := &recordingRouteIncidentRuntime{}
	auditor := &recordingRouteFailureAuditor{started: make(chan struct{}), release: make(chan struct{})}
	recorder, err := newRouteFailureRecorder(runtime, auditor, discardRouteFailureLogger(), 1, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	event := routeFailureRecorderEvent(routes.RouteFailureTransportFailed, true)
	recorder.RecordCommittedAfterFailure(context.Background(), event)
	select {
	case <-auditor.started:
	case <-time.After(time.Second):
		t.Fatal("audit worker did not start")
	}
	recorder.RecordCommittedAfterFailure(context.Background(), event)
	started := time.Now()
	recorder.RecordCommittedAfterFailure(context.Background(), event)
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("full audit queue blocked response path for %s", elapsed)
	}
	if recorder.Dropped() != 1 {
		t.Fatalf("dropped=%d", recorder.Dropped())
	}
	runtime.mu.Lock()
	quarantineCalls := len(runtime.calls)
	runtime.mu.Unlock()
	if quarantineCalls != 3 {
		t.Fatalf("queue pressure skipped quarantine: calls=%d", quarantineCalls)
	}
	close(auditor.release)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := recorder.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestRouteFailureRecorderAuditsStaleArtifactWithoutQuarantiningReplacement(t *testing.T) {
	runtime := &recordingRouteIncidentRuntime{err: extensionsruntime.ErrRuntimeInstanceConflict}
	auditor := &recordingRouteFailureAuditor{}
	recorder, err := newRouteFailureRecorder(runtime, auditor, discardRouteFailureLogger(), 2, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	recorder.RecordCommittedAfterFailure(context.Background(), routeFailureRecorderEvent(
		routes.RouteFailureResponseSchemaRejected, true,
	))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := recorder.Close(ctx); err != nil {
		t.Fatal(err)
	}
	auditor.mu.Lock()
	defer auditor.mu.Unlock()
	if len(auditor.events) != 1 || auditor.events[0].Metadata["quarantineResult"] != "stale_artifact" {
		t.Fatalf("audits=%#v", auditor.events)
	}
}

func routeFailureRecorderEvent(
	code routes.RouteFailureCode,
	observed bool,
) routes.RouteCommittedAfterFailure {
	return routes.RouteCommittedAfterFailure{
		Revision: 7, StepIndex: 2, Phase: routes.RoutePhaseAfter,
		InvocationStage: routes.InvocationStageResponse, Action: "after",
		RouteID: "failure.plugin.after", ContractVersion: "failure.plugin.after@1",
		Method: "POST", PathSignature: "/s:failure", FailureCode: code,
		RuntimeExecutionObserved: observed, ActorID: 42, ResponseStatus: 201,
		CommitState: routes.RouteCommitFinal,
		Artifact: routes.PluginArtifact{
			ExtensionID: "failure.plugin", ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("a", 64), RuntimeInstanceID: "runtime-1",
		},
	}
}

type recordedRouteIncident struct {
	exact extensionsruntime.RuntimeInstanceArtifactIdentity
	cause error
}

type recordingRouteIncidentRuntime struct {
	mu    sync.Mutex
	calls []recordedRouteIncident
	err   error
}

func (r *recordingRouteIncidentRuntime) QuarantineRuntimeInstance(
	exact extensionsruntime.RuntimeInstanceArtifactIdentity,
	cause error,
) (extensionsruntime.RuntimeAdmissionSnapshot, error) {
	r.mu.Lock()
	r.calls = append(r.calls, recordedRouteIncident{exact: exact, cause: cause})
	err := r.err
	r.mu.Unlock()
	return extensionsruntime.RuntimeAdmissionSnapshot{}, err
}

type recordingRouteFailureAuditor struct {
	mu      sync.Mutex
	events  []audit.Event
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (a *recordingRouteFailureAuditor) Append(ctx context.Context, event audit.Event) error {
	if a.started != nil {
		a.once.Do(func() { close(a.started) })
		select {
		case <-a.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	a.mu.Lock()
	a.events = append(a.events, event)
	a.mu.Unlock()
	return nil
}

func discardRouteFailureLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
