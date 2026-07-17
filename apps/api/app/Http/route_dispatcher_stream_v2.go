package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	stdhttp "net/http"
	"strings"
	"sync"
	"time"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

type exactRouteV2StreamRuntime interface {
	exactRouteV2Runtime
	OpenRouteStreamInstance(context.Context, extensionsruntime.RuntimeInstanceIdentity, extensionsruntime.ProtocolV2RouteStreamRequest) (*extensionsruntime.ProtocolV2RouteStream, error)
}

func (i *BufferedRouteStepInvoker) OpenStream(
	ctx context.Context,
	input routes.RouteInvocation,
) (routes.RouteStreamStart, error) {
	if i == nil || i.Runtime == nil || ctx == nil || input.Commit == nil || input.Step.Provider.Kind != routes.ProviderPlugin {
		return routes.RouteStreamStart{}, routes.ErrDispatchTransport
	}
	if input.Stage != routes.InvocationStageHandler ||
		input.Step.Action != extensionmanifest.RouteActionAdd && input.Step.Action != extensionmanifest.RouteActionReplace {
		return routes.RouteStreamStart{}, ErrRouteRuntimeTarget
	}
	authority, ok := input.RequestAuthority()
	if !ok {
		return routes.RouteStreamStart{}, ErrRouteRuntimeTarget
	}
	runtime, ok := i.Runtime.(exactRouteV2StreamRuntime)
	if !ok {
		return routes.RouteStreamStart{}, ErrRouteRuntimeTarget
	}
	artifact := input.Step.Provider.Artifact
	identity := extensionsruntime.RuntimeInstanceIdentity{
		ExtensionID: artifact.ExtensionID, InstanceID: artifact.RuntimeInstanceID,
	}
	snapshot, err := i.Runtime.InspectRuntimeInstance(identity)
	if err != nil {
		return routes.RouteStreamStart{}, err
	}
	idempotencyKey, err := exactProtocolV2RouteIdempotencyKey(input.Request.Headers)
	if err != nil {
		return routes.RouteStreamStart{}, err
	}
	if !snapshot.Active || snapshot.Identity != identity || snapshot.ExtensionVersion != artifact.ExtensionVersion ||
		snapshot.ArtifactDigest != artifact.PackageDigest || strings.TrimSpace(snapshot.Target.BaseURL) != "" {
		return routes.RouteStreamStart{}, ErrRouteRuntimeArtifact
	}
	lease, err := i.Runtime.AcquireRuntimeCall(ctx, identity, extensionsruntime.RuntimeCallRoute)
	if err != nil {
		return routes.RouteStreamStart{}, err
	}
	query, queryValues, err := exactProtocolV2RouteQuery(input.Request.Query)
	if err != nil {
		lease.Release()
		return routes.RouteStreamStart{}, err
	}
	headers := make(stdhttp.Header)
	if err := copyRouteRequestHeaders(headers, input.Request.Headers, authority); err != nil {
		lease.Release()
		return routes.RouteStreamStart{}, err
	}
	wireAuthority, err := protocolV2RequestAuthority(authority)
	if err != nil {
		lease.Release()
		return routes.RouteStreamStart{}, err
	}
	actor := extensionsruntime.NewProtocolV2RouteActor(
		input.Request.ActorID, input.Request.Authenticated, input.Request.Permissions,
	)
	correlationID, err := newRouteStreamCorrelationID()
	if err != nil {
		lease.Release()
		return routes.RouteStreamStart{}, err
	}
	if err := lease.Context.Err(); err != nil {
		lease.Release()
		return routes.RouteStreamStart{}, err
	}
	preflightTimeout, err := routeStreamRemainingBudget(lease.Context)
	if err != nil {
		lease.Release()
		return routes.RouteStreamStart{}, err
	}
	// Admission and all local validation completed. From this point a failed
	// unary preflight cannot prove that the plugin handler did not execute.
	input.Commit.SideEffectStarted()
	preflight, err := runtime.InvokeRouteInstance(lease.Context, identity, extensionsruntime.ProtocolV2RouteRequest{
		RouteID: input.Step.RouteID, ContractVersion: input.Step.ContractVersion,
		RouteAction: input.Step.Action, InvocationStage: extensionsruntime.ProtocolV2RouteInvocationStageHandler,
		Method: input.Request.Method, Path: input.Request.Path, Headers: headers,
		PathParameters: input.Request.Params, QueryParameters: query, QueryParameterValues: queryValues,
		RequestSchema: input.Step.RequestSchema, ResponseSchema: input.Step.ResponseSchema,
		Authority: wireAuthority, Actor: actor, IdempotencyKey: idempotencyKey, CorrelationID: correlationID,
		Timeout: preflightTimeout,
	})
	if err != nil {
		lease.Release()
		return routes.RouteStreamStart{}, err
	}
	if !routes.ValidTerminalResponseStatus(input.Step.Mode, preflight.StatusCode) ||
		!preflight.StreamFollows || preflight.BodyPresent {
		lease.Release()
		return routes.RouteStreamStart{}, fmt.Errorf("%w: runtime did not accept a streamed response", ErrRouteRuntimeTarget)
	}
	if err := lease.Context.Err(); err != nil {
		lease.Release()
		return routes.RouteStreamStart{}, err
	}
	requestTarget := input.Request.Path
	if input.Request.Query != "" {
		requestTarget += "?" + input.Request.Query
	}
	streamTimeout, err := routeStreamRemainingBudget(lease.Context)
	if err != nil {
		lease.Release()
		return routes.RouteStreamStart{}, err
	}
	stream, err := runtime.OpenRouteStreamInstance(lease.Context, identity, extensionsruntime.ProtocolV2RouteStreamRequest{
		RouteID: input.Step.RouteID, ContractVersion: input.Step.ContractVersion,
		Method: input.Request.Method, Path: requestTarget, Mode: input.Step.Mode,
		Headers: headers, Authority: wireAuthority, Actor: actor, IdempotencyKey: idempotencyKey,
		CorrelationID: correlationID,
		Timeout:       streamTimeout,
	})
	if err != nil {
		lease.Release()
		return routes.RouteStreamStart{}, err
	}
	session := newRouteV2StreamSession(stream, lease, preflight.StatusCode)
	// ForceCancel and parent cancel land on the lease context; arm owns that path.
	session.arm(lease.Context)
	if err := lease.Context.Err(); err != nil {
		session.Cancel()
		return routes.RouteStreamStart{}, err
	}
	return routes.RouteStreamStart{
		Response: routes.DispatchResponse{
			Status: preflight.StatusCode, Headers: filteredRouteResponseHeaders(preflight.Headers),
		},
		Session: session,
	}, nil
}

func newRouteStreamCorrelationID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return "route_" + hex.EncodeToString(value[:]), nil
}

// routeStreamRemainingBudget derives the remaining Host total budget for a child
// Protocol V2 call. Zero remaining fails closed so open cannot outlive the fence.
func routeStreamRemainingBudget(ctx context.Context) (time.Duration, error) {
	if ctx == nil {
		return 0, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return 0, nil
	}
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return 0, context.DeadlineExceeded
	}
	return remaining, nil
}

type routeV2WireStream interface {
	Send([]byte, bool) error
	CloseRequest() error
	Recv() (extensionsruntime.ProtocolV2RouteStreamChunk, error)
	Response() (extensionsruntime.ProtocolV2RouteStreamResponse, bool)
	Cancel()
}

const (
	routeV2StreamStateActive int32 = iota
	routeV2StreamStateTerminal
	routeV2StreamStateCanceled
)

// routeV2StreamSession owns wire cancellation and runtime lease release. The
// outer lifetime wrapper owns Host budget, caller callback, and detach.
type routeV2StreamSession struct {
	stream         routeV2WireStream
	lease          *extensionsruntime.RuntimeAdmissionLease
	expectedStatus int

	mu       sync.Mutex
	state    int32
	response *routes.DispatchResponse
	cause    error
	done     chan struct{}
	close    sync.Once
}

func newRouteV2StreamSession(
	stream routeV2WireStream,
	lease *extensionsruntime.RuntimeAdmissionLease,
	expectedStatus int,
) *routeV2StreamSession {
	return &routeV2StreamSession{
		stream: stream, lease: lease, expectedStatus: expectedStatus,
		state: routeV2StreamStateActive, done: make(chan struct{}),
	}
}

func (s *routeV2StreamSession) arm(ctx context.Context) {
	if s == nil || ctx == nil {
		return
	}
	// AfterFunc may run on a different goroutine; completeCancel is Once-safe.
	context.AfterFunc(ctx, func() {
		s.completeCancel(context.Cause(ctx))
	})
}

func (s *routeV2StreamSession) Send(data []byte, final bool) error {
	if s == nil || s.stream == nil {
		return routes.ErrDispatchTransport
	}
	return s.stream.Send(data, final)
}

func (s *routeV2StreamSession) CloseRequest() error {
	if s == nil || s.stream == nil {
		return routes.ErrDispatchTransport
	}
	err := s.stream.CloseRequest()
	if err == nil {
		return nil
	}
	// The peer terminal cancels the underlying gRPC context. A request-side
	// cleanup error after that exact terminal cannot rewrite a committed stream.
	terminal, ok := s.stream.Response()
	if ok && terminal.StatusCode == s.expectedStatus {
		return nil
	}
	return err
}

func (s *routeV2StreamSession) Recv() (routes.RouteStreamChunk, error) {
	if s == nil || s.stream == nil {
		return routes.RouteStreamChunk{}, routes.ErrDispatchTransport
	}
	// If cancel already won, surface the captured cause without touching the wire.
	if cause, canceled := s.canceledCause(); canceled {
		return routes.RouteStreamChunk{}, cause
	}
	chunk, err := s.stream.Recv()
	if err == nil {
		return routes.RouteStreamChunk{
			Sequence: chunk.Sequence, Data: append([]byte(nil), chunk.Data...), Final: chunk.Final,
		}, nil
	}
	if !errors.Is(err, io.EOF) {
		// Ordinary transport error: capture exact cause, cancel wire, release lease.
		s.completeCancel(err)
		return routes.RouteStreamChunk{}, s.Cause()
	}
	terminal, ok := s.stream.Response()
	if !ok || terminal.StatusCode != s.expectedStatus {
		s.completeCancel(ErrRouteRuntimeTarget)
		return routes.RouteStreamChunk{}, s.Cause()
	}
	resp := &routes.DispatchResponse{
		Status: terminal.StatusCode, Headers: filteredRouteResponseHeaders(terminal.Headers),
	}
	if !s.completeTerminal(resp) {
		// Cancel/ForceCancel/budget won the atomic race; Response must stay unpublished.
		return routes.RouteStreamChunk{}, s.Cause()
	}
	return routes.RouteStreamChunk{}, io.EOF
}

func (s *routeV2StreamSession) Response() (routes.DispatchResponse, bool) {
	if s == nil {
		return routes.DispatchResponse{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Only a terminal winner may publish Response.
	if s.state != routeV2StreamStateTerminal || s.response == nil {
		return routes.DispatchResponse{}, false
	}
	return routes.DispatchResponse{
		Status: s.response.Status, Headers: s.response.Headers.Clone(),
	}, true
}

func (s *routeV2StreamSession) Cancel() {
	if s != nil {
		s.completeCancel(nil)
	}
}

func (s *routeV2StreamSession) Done() <-chan struct{} {
	if s == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	s.mu.Lock()
	if s.done == nil {
		s.done = make(chan struct{})
		if s.state != routeV2StreamStateActive {
			close(s.done)
		}
	}
	done := s.done
	s.mu.Unlock()
	return done
}

func (s *routeV2StreamSession) Cause() error {
	if s == nil {
		return context.Canceled
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cause
}

func (s *routeV2StreamSession) canceledCause() (error, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != routeV2StreamStateCanceled {
		return nil, false
	}
	cause := s.cause
	if cause == nil {
		cause = context.Canceled
	}
	return cause, true
}

// publishDone closes Done only after state/cause and cleanup are published.
func (s *routeV2StreamSession) publishDone() {
	s.mu.Lock()
	if s.done == nil {
		s.done = make(chan struct{})
	}
	done := s.done
	s.mu.Unlock()
	close(done)
}

// completeTerminal races with completeCancel. Only the winner runs Once cleanup.
func (s *routeV2StreamSession) completeTerminal(resp *routes.DispatchResponse) bool {
	if s == nil {
		return false
	}
	won := false
	s.close.Do(func() {
		s.mu.Lock()
		s.state = routeV2StreamStateTerminal
		s.response = resp
		s.cause = nil
		s.mu.Unlock()
		// Terminal does not cancel the wire; peer already closed with EOF.
		if s.lease != nil {
			s.lease.Release()
		}
		s.publishDone()
		won = true
	})
	if won {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state == routeV2StreamStateTerminal
}

// completeCancel captures lease cause before Release (Release cancels with nil).
func (s *routeV2StreamSession) completeCancel(cause error) {
	if s == nil {
		return
	}
	s.close.Do(func() {
		// Capture before Release: Release itself cancels the lease context.
		if cause == nil && s.lease != nil && s.lease.Context != nil {
			cause = context.Cause(s.lease.Context)
		}
		if cause == nil {
			cause = context.Canceled
		}
		s.mu.Lock()
		s.state = routeV2StreamStateCanceled
		s.cause = cause
		s.response = nil
		s.mu.Unlock()
		if s.stream != nil {
			s.stream.Cancel()
		}
		if s.lease != nil {
			s.lease.Release()
		}
		// Done closes only after wire cancel, lease release, and cause publication.
		s.publishDone()
	})
}

var _ routes.StreamingStepInvoker = (*BufferedRouteStepInvoker)(nil)
var _ routes.RouteStreamSession = (*routeV2StreamSession)(nil)
var _ routes.RouteStreamLifetimeSource = (*routeV2StreamSession)(nil)
