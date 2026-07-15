package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"

	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

const routeWebSocketControlTimeout = 2 * time.Second

func serveRouteWebSocket(c fiber.Ctx, dispatch *routes.RouteStreamDispatch) error {
	if c == nil || dispatch == nil || !validRouteWebSocketUpgrade(c) {
		return fiber.NewError(fiber.StatusUpgradeRequired, "extensions.websocket_upgrade_required")
	}
	// The hijacked connection outlives Fiber's pooled request context. Actor,
	// permissions, headers, and params were already detached by PrepareStream.
	start, err := dispatch.Open(context.Background())
	if err != nil {
		return mapRouteDispatchError(err)
	}
	if start.Response.Status != fiber.StatusSwitchingProtocols || !validRouteWebSocketSubprotocol(c, start.Response.Headers.Get("Sec-WebSocket-Protocol")) {
		start.Session.Cancel()
		dispatch.Fail()
		return mapRouteDispatchError(fmt.Errorf("%w: invalid websocket preflight", routes.ErrDispatchTransport))
	}
	c.Response().Reset()
	for name, values := range start.Response.Headers {
		for _, value := range values {
			c.Response().Header.Add(name, value)
		}
	}
	upgrader := websocket.FastHTTPUpgrader{}
	if err := upgrader.Upgrade(c.RequestCtx(), func(connection *websocket.Conn) {
		dispatch.ResponseStarted()
		bridgeRouteWebSocket(connection, start.Session, dispatch)
	}); err != nil {
		start.Session.Cancel()
		dispatch.Fail()
		// FastHTTPUpgrader has already authored the exact handshake error.
		return nil
	}
	return nil
}

func validRouteWebSocketUpgrade(c fiber.Ctx) bool {
	if c == nil || c.Method() != fiber.MethodGet || !websocket.FastHTTPIsWebSocketUpgrade(c.RequestCtx()) ||
		strings.TrimSpace(c.Get("Sec-WebSocket-Key")) == "" || !strings.Contains(c.Get("Sec-WebSocket-Version"), "13") {
		return false
	}
	origin := strings.TrimSpace(c.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	return err == nil && (strings.EqualFold(parsed.Scheme, "http") || strings.EqualFold(parsed.Scheme, "https")) &&
		parsed.User == nil && (parsed.Path == "" || parsed.Path == "/") && parsed.RawQuery == "" && parsed.Fragment == "" &&
		strings.EqualFold(parsed.Host, c.Get("Host"))
}

func validRouteWebSocketSubprotocol(c fiber.Ctx, selected string) bool {
	selected = strings.TrimSpace(selected)
	if selected == "" {
		return true
	}
	for _, candidate := range strings.Split(c.Get("Sec-WebSocket-Protocol"), ",") {
		if strings.TrimSpace(candidate) == selected {
			return true
		}
	}
	return false
}

type routeWebSocketPumpResult struct {
	direction string
	normal    bool
	err       error
}

func bridgeRouteWebSocket(connection *websocket.Conn, session routes.RouteStreamSession, dispatch *routes.RouteStreamDispatch) {
	if connection == nil || session == nil || dispatch == nil {
		if session != nil {
			session.Cancel()
		}
		return
	}
	connection.SetReadLimit(extensionsruntime.MaxProtocolV2RouteChunkSize)
	results := make(chan routeWebSocketPumpResult, 2)
	go func() { results <- pumpWebSocketRequests(connection, session) }()
	go func() { results <- pumpWebSocketResponses(connection, session) }()

	first := <-results
	if first.normal {
		_ = dispatch.Complete()
	} else {
		_ = dispatch.StreamFailed(first.err)
	}
	session.Cancel()
	_ = connection.Close()
	// Closing both transports unblocks the opposite pump before this hijacked
	// handler releases the exact runtime admission lease.
	<-results
}

func pumpWebSocketRequests(connection *websocket.Conn, session routes.RouteStreamSession) routeWebSocketPumpResult {
	for {
		messageType, payload, err := connection.ReadMessage()
		if err != nil {
			var closeError *websocket.CloseError
			normal := errors.As(err, &closeError) &&
				(closeError.Code == websocket.CloseNormalClosure || closeError.Code == websocket.CloseGoingAway)
			if normal {
				_ = session.CloseRequest()
			}
			return routeWebSocketPumpResult{direction: "request", normal: normal, err: err}
		}
		if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
			continue
		}
		if len(payload) > extensionsruntime.MaxProtocolV2RouteChunkSize {
			return routeWebSocketPumpResult{direction: "request", err: fmt.Errorf("websocket message exceeds route chunk limit")}
		}
		// RouteStreamFrame has one bounded binary DataChunk. One WebSocket
		// message maps to one chunk, preserving message boundaries without a
		// second framing protocol.
		if err := session.Send(payload, false); err != nil {
			return routeWebSocketPumpResult{direction: "request", err: err}
		}
	}
}

func pumpWebSocketResponses(connection *websocket.Conn, session routes.RouteStreamSession) routeWebSocketPumpResult {
	for {
		chunk, err := session.Recv()
		if errors.Is(err, io.EOF) {
			if response, ok := session.Response(); !ok || response.Status != fiber.StatusSwitchingProtocols {
				return routeWebSocketPumpResult{direction: "response", err: routes.ErrDispatchTransport}
			}
			deadline := time.Now().Add(routeWebSocketControlTimeout)
			_ = connection.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), deadline)
			return routeWebSocketPumpResult{direction: "response", normal: true}
		}
		if err != nil {
			return routeWebSocketPumpResult{direction: "response", err: err}
		}
		if len(chunk.Data) > extensionsruntime.MaxProtocolV2RouteChunkSize {
			return routeWebSocketPumpResult{direction: "response", err: routes.ErrDispatchTransport}
		}
		if err := connection.WriteMessage(websocket.BinaryMessage, chunk.Data); err != nil {
			return routeWebSocketPumpResult{direction: "response", err: err}
		}
	}
}
