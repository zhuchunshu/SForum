package routes

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

func TestRouteStreamFailureSinkRecordsExactPayloadFreeIncidentOnce(t *testing.T) {
	for _, class := range []RouteStreamFailureClass{
		RouteStreamFailureRuntimeTransport,
		RouteStreamFailureHostBudget,
		RouteStreamFailureInvalidPreflight,
		RouteStreamFailureMissingTerminal,
	} {
		t.Run(string(class), func(t *testing.T) {
			sink := &recordingRouteStreamFailureSink{}
			step := streamIncidentStep("stream.incident." + string(class))
			dispatcher := streamIncidentDispatcher(step, &streamIncidentInvoker{}, sink)
			prepared, err := dispatcher.PrepareStream(context.Background(), DispatchRequest{
				Method: http.MethodGet, Path: "/incident", ActorID: 42,
			})
			if err != nil || prepared.Dispatch == nil {
				t.Fatalf("prepared=%#v err=%v", prepared, err)
			}
			start, err := prepared.Dispatch.Open(context.Background())
			if err != nil || start.Session == nil {
				t.Fatalf("start=%#v err=%v", start, err)
			}
			prepared.Dispatch.ResponseStarted()
			failure := errors.New("must not enter evidence")
			if class == RouteStreamFailureRuntimeTransport {
				err = prepared.Dispatch.StreamFailed(failure)
			} else {
				err = prepared.Dispatch.StreamFailedAs(class, failure)
			}
			if !errors.Is(err, ErrDispatchTransport) {
				t.Fatalf("failure error=%v", err)
			}
			_ = prepared.Dispatch.StreamFailedAs(class, errors.New("duplicate"))
			start.Session.Cancel()

			events := sink.snapshot()
			if len(events) != 1 {
				t.Fatalf("events=%#v", events)
			}
			event := events[0]
			if !ValidRouteStreamFailure(event) || event.CauseClass != class || event.Revision != 1 ||
				event.StepIndex != 0 || event.Phase != RoutePhaseHandler ||
				event.InvocationStage != InvocationStageHandler || event.Action != extensionmanifest.RouteActionAdd ||
				event.Mode != extensionmanifest.RouteModeStream || event.RouteID != step.RouteID ||
				event.ContractVersion != step.ContractVersion || event.Method != http.MethodGet ||
				event.PathSignature != routeStepPathSignature(step) || event.FailureCode != RouteFailureTransportFailed ||
				!event.RuntimeExecutionObserved || event.ActorID != 42 || event.ResponseStatus != http.StatusOK ||
				event.CommitState != RouteCommitResponseStarted || event.Artifact != step.Provider.Artifact {
				t.Fatalf("event=%#v", event)
			}
			body, marshalErr := json.Marshal(event)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			lower := strings.ToLower(string(body))
			for _, forbidden := range []string{"must not enter evidence", "requestbody", "responsebody", "rawerror", "headers", "query"} {
				if strings.Contains(lower, forbidden) {
					t.Fatalf("payload-free incident contains %q: %s", forbidden, body)
				}
			}
		})
	}
}

func TestRouteStreamFailureSinkExcludesHostAndCallerAbortions(t *testing.T) {
	for _, test := range []struct {
		name string
		fail func(*RouteStreamDispatch) error
	}{
		{name: "caller cancellation", fail: func(dispatch *RouteStreamDispatch) error {
			return dispatch.StreamFailed(context.Canceled)
		}},
		{name: "Host writer", fail: func(dispatch *RouteStreamDispatch) error {
			return dispatch.StreamAborted(io.ErrClosedPipe)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			sink := &recordingRouteStreamFailureSink{}
			step := streamIncidentStep("stream.abort." + strings.ReplaceAll(test.name, " ", "_"))
			dispatcher := streamIncidentDispatcher(step, &streamIncidentInvoker{}, sink)
			prepared, err := dispatcher.PrepareStream(context.Background(), DispatchRequest{
				Method: http.MethodGet, Path: "/incident",
			})
			if err != nil || prepared.Dispatch == nil {
				t.Fatalf("prepared=%#v err=%v", prepared, err)
			}
			start, err := prepared.Dispatch.Open(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			prepared.Dispatch.ResponseStarted()
			if err := test.fail(prepared.Dispatch); !errors.Is(err, ErrDispatchTransport) {
				t.Fatalf("abort error=%v", err)
			}
			start.Session.Cancel()
			if events := sink.snapshot(); len(events) != 0 {
				t.Fatalf("abort recorded incidents=%#v", events)
			}
		})
	}
}

func TestRouteStreamOpenRecordsInvalidPreflightAndBudgetAfterExecution(t *testing.T) {
	t.Run("invalid preflight", func(t *testing.T) {
		sink := &recordingRouteStreamFailureSink{}
		step := streamIncidentStep("stream.invalid_preflight")
		invoker := &streamIncidentInvoker{body: []byte("invalid")}
		dispatcher := streamIncidentDispatcher(step, invoker, sink)
		prepared, err := dispatcher.PrepareStream(context.Background(), DispatchRequest{
			Method: http.MethodGet, Path: "/incident",
		})
		if err != nil || prepared.Dispatch == nil {
			t.Fatalf("prepared=%#v err=%v", prepared, err)
		}
		if _, err := prepared.Dispatch.Open(context.Background()); !errors.Is(err, ErrDispatchTransport) {
			t.Fatalf("open error=%v", err)
		}
		events := sink.snapshot()
		if len(events) != 1 || events[0].CauseClass != RouteStreamFailureInvalidPreflight ||
			events[0].CommitState != RouteCommitSideEffectStarted {
			t.Fatalf("events=%#v", events)
		}
	})

	t.Run("Host budget", func(t *testing.T) {
		sink := &recordingRouteStreamFailureSink{}
		step := streamIncidentStep("stream.host_budget")
		step.TimeoutMS = 5
		dispatcher := streamIncidentDispatcher(step, &streamBudgetIncidentInvoker{}, sink)
		prepared, err := dispatcher.PrepareStream(context.Background(), DispatchRequest{
			Method: http.MethodGet, Path: "/incident",
		})
		if err != nil || prepared.Dispatch == nil {
			t.Fatalf("prepared=%#v err=%v", prepared, err)
		}
		if _, err := prepared.Dispatch.Open(context.Background()); !errors.Is(err, ErrDispatchTransport) || !errors.Is(err, ErrRouteStreamBudgetExceeded) {
			t.Fatalf("open error=%v", err)
		}
		events := sink.snapshot()
		if len(events) != 1 || events[0].CauseClass != RouteStreamFailureHostBudget ||
			events[0].CommitState != RouteCommitSideEffectStarted {
			t.Fatalf("events=%#v", events)
		}
	})
}

func streamIncidentStep(id string) RouteExecutionStep {
	step := dispatchPluginStep(RoutePhaseHandler, id, extensionmanifest.RouteActionAdd)
	step.Mode = extensionmanifest.RouteModeStream
	step.Path = "/incident"
	return step
}

func streamIncidentDispatcher(
	step RouteExecutionStep,
	invoker StreamingStepInvoker,
	sink RouteStreamFailureSink,
) *Dispatcher {
	return NewDispatcher(DispatcherConfig{
		Plans: dispatchPlanResolver{plan: dispatchPlan(
			http.MethodGet, "/incident", nil, []RouteExecutionStep{step}, 0,
		)},
		Guard: allowStreamGuard{}, Steps: streamIncidentStepInvoker{StreamingStepInvoker: invoker},
		StreamFailures: sink,
	})
}

type streamIncidentStepInvoker struct{ StreamingStepInvoker }

func (streamIncidentStepInvoker) SupportsMode(string) bool { return false }
func (streamIncidentStepInvoker) Invoke(context.Context, RouteInvocation) (RouteInvocationResult, error) {
	return RouteInvocationResult{}, ErrDispatchTransport
}

type streamIncidentInvoker struct{ body []byte }

func (i *streamIncidentInvoker) OpenStream(_ context.Context, input RouteInvocation) (RouteStreamStart, error) {
	input.Commit.SideEffectStarted()
	return RouteStreamStart{
		Response: DispatchResponse{Status: http.StatusOK, Body: append([]byte(nil), i.body...)},
		Session:  authorityStreamSession{},
	}, nil
}

type streamBudgetIncidentInvoker struct{}

func (*streamBudgetIncidentInvoker) OpenStream(ctx context.Context, input RouteInvocation) (RouteStreamStart, error) {
	input.Commit.SideEffectStarted()
	<-ctx.Done()
	return RouteStreamStart{}, context.Cause(ctx)
}

type recordingRouteStreamFailureSink struct {
	mu     sync.Mutex
	events []RouteStreamFailure
}

func (s *recordingRouteStreamFailureSink) RecordStreamFailure(_ context.Context, event RouteStreamFailure) {
	s.mu.Lock()
	s.events = append(s.events, event)
	s.mu.Unlock()
}

func (s *recordingRouteStreamFailureSink) snapshot() []RouteStreamFailure {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]RouteStreamFailure(nil), s.events...)
}

var _ RouteStreamFailureSink = (*recordingRouteStreamFailureSink)(nil)
