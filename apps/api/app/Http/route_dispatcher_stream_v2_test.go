package http

import (
	"errors"
	"io"
	stdhttp "net/http"
	"testing"

	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

func TestRouteV2StreamSessionCopiesChunksAndAcceptsBoundTerminal(t *testing.T) {
	wire := &fakeRouteV2WireStream{
		chunks: []extensionsruntime.ProtocolV2RouteStreamChunk{{Sequence: 1, Data: []byte("first"), Final: true}},
		response: extensionsruntime.ProtocolV2RouteStreamResponse{
			StatusCode: stdhttp.StatusCreated,
			Headers:    stdhttp.Header{"X-Result": {"done"}, "Set-Cookie": {"private=1"}},
		},
	}
	session := &routeV2StreamSession{stream: wire, expectedStatus: stdhttp.StatusCreated}
	if err := session.Send([]byte("request"), true); err != nil || err == nil && string(wire.sent) != "request" {
		t.Fatalf("send=%q err=%v", wire.sent, err)
	}
	if err := session.CloseRequest(); err != nil || !wire.requestClosed {
		t.Fatalf("request closed=%t err=%v", wire.requestClosed, err)
	}
	chunk, err := session.Recv()
	if err != nil || chunk.Sequence != 1 || string(chunk.Data) != "first" || !chunk.Final {
		t.Fatalf("chunk=%#v err=%v", chunk, err)
	}
	chunk.Data[0] = 'x'
	if _, err := session.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("terminal err=%v", err)
	}
	response, ok := session.Response()
	if !ok || response.Status != stdhttp.StatusCreated || response.Headers.Get("X-Result") != "done" || response.Headers.Get("Set-Cookie") != "" {
		t.Fatalf("response=%#v ok=%t", response, ok)
	}
	if wire.cancelled {
		t.Fatal("successful terminal cancelled the runtime stream")
	}
}

func TestRouteV2StreamSessionCancelsOnTransportFailureOrStatusDrift(t *testing.T) {
	for name, wire := range map[string]*fakeRouteV2WireStream{
		"transport": {recvErr: errors.New("closed")},
		"status": {
			response: extensionsruntime.ProtocolV2RouteStreamResponse{StatusCode: stdhttp.StatusNoContent},
		},
	} {
		t.Run(name, func(t *testing.T) {
			session := &routeV2StreamSession{stream: wire, expectedStatus: stdhttp.StatusOK}
			_, err := session.Recv()
			if err == nil || !wire.cancelled {
				t.Fatalf("error=%v cancelled=%t", err, wire.cancelled)
			}
			if _, ok := session.Response(); ok {
				t.Fatal("failed stream exposed a terminal response")
			}
		})
	}
}

type fakeRouteV2WireStream struct {
	chunks        []extensionsruntime.ProtocolV2RouteStreamChunk
	response      extensionsruntime.ProtocolV2RouteStreamResponse
	recvErr       error
	sent          []byte
	requestClosed bool
	cancelled     bool
}

func (s *fakeRouteV2WireStream) Send(data []byte, _ bool) error {
	s.sent = append([]byte(nil), data...)
	return nil
}

func (s *fakeRouteV2WireStream) CloseRequest() error {
	s.requestClosed = true
	return nil
}

func (s *fakeRouteV2WireStream) Recv() (extensionsruntime.ProtocolV2RouteStreamChunk, error) {
	if s.recvErr != nil {
		return extensionsruntime.ProtocolV2RouteStreamChunk{}, s.recvErr
	}
	if len(s.chunks) == 0 {
		return extensionsruntime.ProtocolV2RouteStreamChunk{}, io.EOF
	}
	chunk := s.chunks[0]
	s.chunks = s.chunks[1:]
	return chunk, nil
}

func (s *fakeRouteV2WireStream) Response() (extensionsruntime.ProtocolV2RouteStreamResponse, bool) {
	return s.response, s.response.StatusCode != 0
}

func (s *fakeRouteV2WireStream) Cancel() { s.cancelled = true }

var _ routeV2WireStream = (*fakeRouteV2WireStream)(nil)
