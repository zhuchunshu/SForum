package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	stdhttp "net/http"
	"strings"
	"sync"

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
	if i == nil || i.Runtime == nil || ctx == nil || input.Step.Provider.Kind != routes.ProviderPlugin {
		return routes.RouteStreamStart{}, routes.ErrDispatchTransport
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
	if !snapshot.Active || snapshot.Identity != identity || snapshot.ExtensionVersion != artifact.ExtensionVersion ||
		snapshot.ArtifactDigest != artifact.PackageDigest || strings.TrimSpace(snapshot.Target.BaseURL) != "" {
		return routes.RouteStreamStart{}, ErrRouteRuntimeArtifact
	}
	lease, err := i.Runtime.AcquireRuntimeCall(ctx, identity, extensionsruntime.RuntimeCallRoute)
	if err != nil {
		return routes.RouteStreamStart{}, err
	}
	query, err := exactProtocolV2RouteQuery(input.Request.Query)
	if err != nil {
		lease.Release()
		return routes.RouteStreamStart{}, err
	}
	headers := make(stdhttp.Header)
	copyRouteRequestHeaders(headers, input.Request.Headers)
	actor := extensionsruntime.NewProtocolV2RouteActor(
		input.Request.ActorID, input.Request.Authenticated, input.Request.Permissions,
	)
	preflight, err := runtime.InvokeRouteInstance(lease.Context, identity, extensionsruntime.ProtocolV2RouteRequest{
		RouteID: input.Step.RouteID, ContractVersion: input.Step.ContractVersion,
		Method: input.Request.Method, Path: input.Request.Path, Headers: headers,
		PathParameters: input.Request.Params, QueryParameters: query,
		RequestSchema: input.Step.RequestSchema, ResponseSchema: input.Step.ResponseSchema,
		Actor: actor,
	})
	if err != nil {
		lease.Release()
		return routes.RouteStreamStart{}, err
	}
	if !preflight.StreamFollows || preflight.BodyPresent {
		lease.Release()
		return routes.RouteStreamStart{}, fmt.Errorf("%w: runtime did not accept a streamed response", ErrRouteRuntimeTarget)
	}
	requestTarget := input.Request.Path
	if input.Request.Query != "" {
		requestTarget += "?" + input.Request.Query
	}
	stream, err := runtime.OpenRouteStreamInstance(lease.Context, identity, extensionsruntime.ProtocolV2RouteStreamRequest{
		RouteID: input.Step.RouteID, ContractVersion: input.Step.ContractVersion,
		Method: input.Request.Method, Path: requestTarget, Mode: input.Step.Mode,
		Headers: headers, Actor: actor,
	})
	if err != nil {
		lease.Release()
		return routes.RouteStreamStart{}, err
	}
	session := &routeV2StreamSession{
		stream: stream, lease: lease, expectedStatus: preflight.StatusCode,
	}
	return routes.RouteStreamStart{
		Response: routes.DispatchResponse{
			Status: preflight.StatusCode, Headers: filteredRouteResponseHeaders(preflight.Headers),
		},
		Session: session,
	}, nil
}

type routeV2WireStream interface {
	Send([]byte, bool) error
	CloseRequest() error
	Recv() (extensionsruntime.ProtocolV2RouteStreamChunk, error)
	Response() (extensionsruntime.ProtocolV2RouteStreamResponse, bool)
	Cancel()
}

type routeV2StreamSession struct {
	stream         routeV2WireStream
	lease          *extensionsruntime.RuntimeAdmissionLease
	expectedStatus int

	mu       sync.Mutex
	response *routes.DispatchResponse
	close    sync.Once
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
	return s.stream.CloseRequest()
}

func (s *routeV2StreamSession) Recv() (routes.RouteStreamChunk, error) {
	if s == nil || s.stream == nil {
		return routes.RouteStreamChunk{}, routes.ErrDispatchTransport
	}
	chunk, err := s.stream.Recv()
	if err == nil {
		return routes.RouteStreamChunk{
			Sequence: chunk.Sequence, Data: append([]byte(nil), chunk.Data...), Final: chunk.Final,
		}, nil
	}
	if !errors.Is(err, io.EOF) {
		s.finish(true)
		return routes.RouteStreamChunk{}, err
	}
	terminal, ok := s.stream.Response()
	if !ok || terminal.StatusCode != s.expectedStatus {
		s.finish(true)
		return routes.RouteStreamChunk{}, ErrRouteRuntimeTarget
	}
	s.mu.Lock()
	s.response = &routes.DispatchResponse{
		Status: terminal.StatusCode, Headers: filteredRouteResponseHeaders(terminal.Headers),
	}
	s.mu.Unlock()
	s.finish(false)
	return routes.RouteStreamChunk{}, io.EOF
}

func (s *routeV2StreamSession) Response() (routes.DispatchResponse, bool) {
	if s == nil {
		return routes.DispatchResponse{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.response == nil {
		return routes.DispatchResponse{}, false
	}
	return routes.DispatchResponse{
		Status: s.response.Status, Headers: s.response.Headers.Clone(),
	}, true
}

func (s *routeV2StreamSession) Cancel() {
	if s != nil {
		s.finish(true)
	}
}

func (s *routeV2StreamSession) finish(cancel bool) {
	if s == nil {
		return
	}
	s.close.Do(func() {
		if cancel && s.stream != nil {
			s.stream.Cancel()
		}
		if s.lease != nil {
			s.lease.Release()
		}
	})
}

var _ routes.StreamingStepInvoker = (*BufferedRouteStepInvoker)(nil)
var _ routes.RouteStreamSession = (*routeV2StreamSession)(nil)
