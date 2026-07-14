package extensionsruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	pluginwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	protocolwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
)

var ErrProtocolV2RouteStreamInvalid = errors.New("protocol v2 route stream is invalid")

const MaxProtocolV2RouteChunkSize = 1 << 20

// ProtocolV2RouteStreamRequest binds a bidirectional stream to one exact
// non-buffered route declaration. Path may include the raw query because the
// v2 stream open frame deliberately carries the original request target.
type ProtocolV2RouteStreamRequest struct {
	RouteID         string
	ContractVersion string
	Method          string
	Path            string
	Mode            string
	Headers         http.Header
	Actor           *ProtocolV2RouteActor
	CorrelationID   string
	Timeout         time.Duration
}

type ProtocolV2RouteStreamChunk struct {
	Sequence uint64
	Data     []byte
	Final    bool
}

type ProtocolV2RouteStreamResponse struct {
	StatusCode int
	Headers    http.Header
}

// ProtocolV2RouteStream owns the child deadline until the peer sends a valid
// terminal frame or the Host explicitly cancels the stream.
type ProtocolV2RouteStream struct {
	raw    pluginwire.PluginRuntimeService_StreamRouteClient
	cancel context.CancelFunc

	sendMu    sync.Mutex
	sendSeq   uint64
	sendFinal bool
	sendClose bool
	recvMu    sync.Mutex
	recvSeq   uint64
	recvFinal bool
	response  *ProtocolV2RouteStreamResponse
	closeOnce sync.Once
}

func (c *protocolV2Client) OpenRouteStreamContext(
	parent context.Context,
	input ProtocolV2RouteStreamRequest,
) (*ProtocolV2RouteStream, error) {
	if c == nil || c.client == nil || c.identity == nil || parent == nil {
		return nil, ErrProtocolV2RouteStreamInvalid
	}
	if err := validateProtocolV2RouteStreamRequest(input); err != nil {
		return nil, err
	}
	if err := c.validateFrozenRouteStream(input); err != nil {
		return nil, err
	}
	timeout := input.Timeout
	if timeout <= 0 {
		timeout = DefaultProtocolV2RequestTimeout
	}
	ctx, cancel := protocolV2Deadline(parent, timeout)
	requestContext := c.requestContext(ctx, input.CorrelationID)
	if input.Actor != nil {
		requestContext.Actor = &protocolwire.Actor{
			UserId: input.Actor.UserID, PermissionKeys: append([]string(nil), input.Actor.PermissionKeys...),
		}
	}
	raw, err := c.client.StreamRoute(ctx)
	if err == nil {
		err = raw.Send(&pluginwire.RouteStreamFrame{Frame: &pluginwire.RouteStreamFrame_Open{Open: &pluginwire.RouteStreamOpen{
			Context: requestContext, RouteId: input.RouteID, ContractVersion: input.ContractVersion,
			Method: input.Method, Path: input.Path, Headers: protocolV2RouteHeaders(input.Headers),
		}}})
	}
	if err != nil {
		if raw != nil {
			_ = raw.CloseSend()
		}
		cancel()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	return &ProtocolV2RouteStream{raw: raw, cancel: cancel}, nil
}

func validateProtocolV2RouteStreamRequest(input ProtocolV2RouteStreamRequest) error {
	if strings.TrimSpace(input.RouteID) == "" || strings.TrimSpace(input.ContractVersion) == "" ||
		strings.TrimSpace(input.Method) == "" || !strings.HasPrefix(input.Path, "/") ||
		!protocolV2StreamingRouteMode(input.Mode) {
		return ErrProtocolV2RouteStreamInvalid
	}
	if input.Actor != nil && input.Actor.UserID <= 0 {
		return fmt.Errorf("%w: authenticated actor id is invalid", ErrProtocolV2RouteStreamInvalid)
	}
	return nil
}

func protocolV2StreamingRouteMode(mode string) bool {
	switch mode {
	case extensionmanifest.RouteModeMultipart, extensionmanifest.RouteModeSSE,
		extensionmanifest.RouteModeStream, extensionmanifest.RouteModeWebSocket:
		return true
	default:
		return false
	}
}

func (c *protocolV2Client) validateFrozenRouteStream(input ProtocolV2RouteStreamRequest) error {
	for _, route := range c.routes {
		if route.ID != input.RouteID || route.ContractVersion != input.ContractVersion || route.Mode != input.Mode {
			continue
		}
		for _, method := range route.Methods {
			if method == "*" || strings.EqualFold(method, input.Method) {
				return nil
			}
		}
	}
	return fmt.Errorf(
		"%w: route %q contract %q mode %q is not frozen for method %q",
		ErrProtocolV2RouteStreamInvalid, input.RouteID, input.ContractVersion, input.Mode, input.Method,
	)
}

func (s *ProtocolV2RouteStream) Context() context.Context {
	if s == nil || s.raw == nil {
		return context.Background()
	}
	return s.raw.Context()
}

// Send copies one bounded request chunk. Sequence and checksum are authored by
// the Host so plugins cannot exploit caller-controlled framing metadata.
func (s *ProtocolV2RouteStream) Send(data []byte, final bool) error {
	if s == nil || s.raw == nil || len(data) > MaxProtocolV2RouteChunkSize {
		return ErrProtocolV2RouteStreamInvalid
	}
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if s.sendFinal || s.sendClose {
		return fmt.Errorf("%w: request stream is already terminal", ErrProtocolV2RouteStreamInvalid)
	}
	s.sendSeq++
	digest := sha256.Sum256(data)
	if err := s.raw.Send(&pluginwire.RouteStreamFrame{Frame: &pluginwire.RouteStreamFrame_Chunk{Chunk: &protocolwire.DataChunk{
		Sequence: s.sendSeq, Data: append([]byte(nil), data...), Checksum: digest[:], Final: final,
	}}}); err != nil {
		return err
	}
	s.sendFinal = final
	return nil
}

// CloseRequest half-closes the Host request direction while leaving the peer
// response readable. A final data chunk is optional for an empty request body.
func (s *ProtocolV2RouteStream) CloseRequest() error {
	if s == nil || s.raw == nil {
		return ErrProtocolV2RouteStreamInvalid
	}
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if s.sendClose {
		return fmt.Errorf("%w: request stream is already closed", ErrProtocolV2RouteStreamInvalid)
	}
	if err := s.raw.Send(&pluginwire.RouteStreamFrame{Frame: &pluginwire.RouteStreamFrame_Close{Close: &pluginwire.RouteStreamClose{}}}); err != nil {
		return err
	}
	if err := s.raw.CloseSend(); err != nil {
		return err
	}
	s.sendClose = true
	return nil
}

func (s *ProtocolV2RouteStream) Recv() (ProtocolV2RouteStreamChunk, error) {
	if s == nil || s.raw == nil {
		return ProtocolV2RouteStreamChunk{}, ErrProtocolV2RouteStreamInvalid
	}
	s.recvMu.Lock()
	defer s.recvMu.Unlock()
	if s.response != nil {
		return ProtocolV2RouteStreamChunk{}, io.EOF
	}
	frame, err := s.raw.Recv()
	if err != nil {
		if contextErr := s.Context().Err(); contextErr != nil {
			s.finish()
			return ProtocolV2RouteStreamChunk{}, contextErr
		}
		if errors.Is(err, io.EOF) {
			s.finish()
			return ProtocolV2RouteStreamChunk{}, ErrProtocolV2RouteStreamInvalid
		}
		return ProtocolV2RouteStreamChunk{}, err
	}
	if closeFrame := frame.GetClose(); closeFrame != nil {
		if err := protocolV2Error(closeFrame.GetError()); err != nil {
			s.finish()
			return ProtocolV2RouteStreamChunk{}, err
		}
		if err := s.captureResponseClose(closeFrame); err != nil {
			s.finish()
			return ProtocolV2RouteStreamChunk{}, err
		}
		s.finish()
		return ProtocolV2RouteStreamChunk{}, io.EOF
	}
	chunk := frame.GetChunk()
	if chunk == nil || len(chunk.GetData()) > MaxProtocolV2RouteChunkSize || s.recvFinal ||
		chunk.GetSequence() != s.recvSeq+1 || !validProtocolV2RouteChunkChecksum(chunk) {
		s.finish()
		return ProtocolV2RouteStreamChunk{}, ErrProtocolV2RouteStreamInvalid
	}
	s.recvSeq = chunk.GetSequence()
	s.recvFinal = chunk.GetFinal()
	return ProtocolV2RouteStreamChunk{
		Sequence: chunk.GetSequence(), Data: append([]byte(nil), chunk.GetData()...), Final: chunk.GetFinal(),
	}, nil
}

func validProtocolV2RouteChunkChecksum(chunk *protocolwire.DataChunk) bool {
	if len(chunk.GetChecksum()) == 0 {
		return true
	}
	digest := sha256.Sum256(chunk.GetData())
	return bytes.Equal(chunk.GetChecksum(), digest[:])
}

func (s *ProtocolV2RouteStream) captureResponseClose(closeFrame *pluginwire.RouteStreamClose) error {
	if closeFrame == nil || closeFrame.GetStatusCode() < 100 || closeFrame.GetStatusCode() > 599 {
		return ErrProtocolV2RouteStreamInvalid
	}
	headers, err := protocolV2RouteHTTPHeaders(closeFrame.GetHeaders())
	if err != nil {
		return err
	}
	s.response = &ProtocolV2RouteStreamResponse{StatusCode: int(closeFrame.GetStatusCode()), Headers: headers}
	return nil
}

func (s *ProtocolV2RouteStream) Response() (ProtocolV2RouteStreamResponse, bool) {
	if s == nil {
		return ProtocolV2RouteStreamResponse{}, false
	}
	s.recvMu.Lock()
	defer s.recvMu.Unlock()
	if s.response == nil {
		return ProtocolV2RouteStreamResponse{}, false
	}
	return ProtocolV2RouteStreamResponse{
		StatusCode: s.response.StatusCode, Headers: s.response.Headers.Clone(),
	}, true
}

// Cancel propagates browser disconnects and Host shutdown to the gRPC stream.
func (s *ProtocolV2RouteStream) Cancel() {
	if s != nil {
		s.finish()
	}
}

func (s *ProtocolV2RouteStream) finish() {
	if s == nil {
		return
	}
	s.closeOnce.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
	})
}
