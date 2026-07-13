package hostapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
)

// Gateway 在 loopback 上暴露 Host API，供插件子进程调用（F2.2）。
// 每个已启动插件持有独立 token；请求必须带 X-SForum-Extension-Id 与 Bearer token。
type Gateway struct {
	mu                       sync.RWMutex
	service                  *Service
	services                 *ServiceRegistry
	commands                 *protocolV2CommandEngine
	protocolV2CommandsFrozen bool
	server                   *http.Server
	ln                       net.Listener
	baseURL                  string
	tokens                   map[string]string // extensionID → token
}

// RegisterProtocolV2 exposes typed Host services only on the caller's
// runtime-bound go-plugin broker server.
func (g *Gateway) RegisterProtocolV2(server grpc.ServiceRegistrar) {
	if g == nil || server == nil {
		return
	}
	g.mu.Lock()
	service := g.service
	services := g.services
	commands := g.commands
	g.protocolV2CommandsFrozen = true
	g.mu.Unlock()
	registerProtocolV2(server, service, services, commands)
}

// BindProtocolV2CommandRuntime installs one immutable Host-owned command
// catalog. RegisterProtocolV2 freezes the catalog; replacement then requires a
// new Gateway/server boot so running brokers never observe contract drift.
func (g *Gateway) BindProtocolV2CommandRuntime(runtime ProtocolV2CommandRuntime) error {
	if g == nil {
		return fmt.Errorf("hostapi gateway is nil")
	}
	if runtime == nil {
		return fmt.Errorf("hostapi: protocol v2 command runtime is required")
	}
	engine := runtime.commandEngine()
	if engine == nil {
		return fmt.Errorf("hostapi: protocol v2 command runtime is required")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.protocolV2CommandsFrozen {
		return fmt.Errorf("hostapi: protocol v2 command runtime is frozen until the next Gateway boot")
	}
	g.commands = engine
	return nil
}

// NewGateway 创建未启动的网关；Start 后才监听。
func NewGateway(service *Service) *Gateway {
	return &Gateway{
		service:  service,
		services: NewServiceRegistry(),
		tokens:   map[string]string{},
	}
}

// ReplaceProtocolV2Services atomically publishes one runtime's complete
// handshake service declaration set.
func (g *Gateway) ReplaceProtocolV2Services(extensionID string, registrations []ServiceRegistration) error {
	registry := g.ProtocolV2ServiceRegistry()
	if registry == nil {
		return fmt.Errorf("hostapi gateway is nil")
	}
	return registry.ReplaceExtension(extensionID, registrations)
}

// UnregisterProtocolV2Services removes all services owned by one runtime.
func (g *Gateway) UnregisterProtocolV2Services(extensionID string) {
	if registry := g.ProtocolV2ServiceRegistry(); registry != nil {
		registry.UnregisterExtension(extensionID)
	}
}

// UnregisterProtocolV2ServiceInstance removes services only when the current
// registry snapshot still belongs to the stopping runtime instance.
func (g *Gateway) UnregisterProtocolV2ServiceInstance(extensionID, instanceID string) bool {
	registry := g.ProtocolV2ServiceRegistry()
	return registry != nil && registry.UnregisterProtocolV2ServiceInstance(extensionID, instanceID)
}

// ProtocolV2ServiceRegistry exposes the shared immutable-snapshot registry to
// runtime assembly and host-owned inspectors.
func (g *Gateway) ProtocolV2ServiceRegistry() *ServiceRegistry {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	registry := g.services
	g.mu.RUnlock()
	return registry
}

// Start 绑定 127.0.0.1:0 并开始服务。
func (g *Gateway) Start() error {
	if g == nil || g.service == nil {
		return fmt.Errorf("hostapi gateway: service required")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.ln != nil {
		return nil
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/call", g.handleCall)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	g.ln = ln
	g.baseURL = "http://" + ln.Addr().String()
	g.server = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		_ = g.server.Serve(ln)
	}()
	return nil
}

// BaseURL 返回 loopback 根地址。
func (g *Gateway) BaseURL() string {
	if g == nil {
		return ""
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.baseURL
}

// RegisterExtension 为插件签发 token，返回应注入子进程的环境变量。
func (g *Gateway) RegisterExtension(extensionID string) (token string, env []string, err error) {
	if g == nil {
		return "", nil, fmt.Errorf("hostapi gateway is nil")
	}
	if err := g.Start(); err != nil {
		return "", nil, err
	}
	extensionID = strings.TrimSpace(extensionID)
	if extensionID == "" {
		return "", nil, fmt.Errorf("extension id required")
	}
	token, err = randomToken()
	if err != nil {
		return "", nil, err
	}
	g.mu.Lock()
	g.tokens[extensionID] = token
	base := g.baseURL
	g.mu.Unlock()

	env = []string{
		"SFORUM_HOST_API_URL=" + base,
		"SFORUM_HOST_API_TOKEN=" + token,
		"SFORUM_EXTENSION_ID=" + extensionID,
		"SFORUM_HOST_API_VERSION=" + Version,
	}
	return token, env, nil
}

// UnregisterExtension 撤销 token。
func (g *Gateway) UnregisterExtension(extensionID string) {
	if g == nil {
		return
	}
	extensionID = strings.TrimSpace(extensionID)
	g.mu.Lock()
	delete(g.tokens, extensionID)
	services := g.services
	g.mu.Unlock()
	if services != nil {
		services.UnregisterExtension(extensionID)
	}
}

// Close 停止监听。
func (g *Gateway) Close() error {
	if g == nil {
		return nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.tokens = map[string]string{}
	if g.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = g.server.Shutdown(ctx)
		g.server = nil
	}
	g.ln = nil
	g.baseURL = ""
	return nil
}

func (g *Gateway) handleCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	extensionID := strings.TrimSpace(r.Header.Get("X-SForum-Extension-Id"))
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	token := strings.TrimPrefix(auth, "Bearer ")
	token = strings.TrimSpace(token)
	if extensionID == "" || token == "" {
		writeJSON(w, http.StatusUnauthorized, Response{OK: false, Reason: "host.unauthorized", Message: "missing extension credentials"})
		return
	}
	g.mu.RLock()
	expected := g.tokens[extensionID]
	service := g.service
	g.mu.RUnlock()
	if expected == "" || expected != token {
		writeJSON(w, http.StatusUnauthorized, Response{OK: false, Reason: "host.unauthorized", Message: "invalid extension credentials"})
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Reason: "host.invalid_request", Message: "invalid JSON body"})
		return
	}
	// 强制 extensionId 为已认证身份，防止插件冒充其它扩展。
	req.ExtensionID = extensionID
	resp := service.Call(r.Context(), req)
	status := http.StatusOK
	if !resp.OK && resp.Reason == "host.capability_denied" {
		status = http.StatusForbidden
	}
	writeJSON(w, status, resp)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func randomToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// Client 是插件侧最小 Host API 客户端（也可作 SDK stub）。
type Client struct {
	BaseURL     string
	Token       string
	ExtensionID string
	HTTP        *http.Client
}

// Call 发起一次 Host API 调用。
func (c *Client) Call(ctx context.Context, method string, payload map[string]any) (Response, error) {
	if c == nil || c.BaseURL == "" {
		return Response{}, fmt.Errorf("hostapi client is not configured")
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultTimeout + time.Second}
	}
	body, err := json.Marshal(Request{Method: method, ExtensionID: c.ExtensionID, Payload: payload})
	if err != nil {
		return Response{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/v1/call", bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-SForum-Extension-Id", c.ExtensionID)
	res, err := httpClient.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer res.Body.Close()
	var resp Response
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return Response{}, err
	}
	return resp, nil
}

// ClientFromEnv 从插件进程环境变量构造客户端。
func ClientFromEnv() (*Client, error) {
	base := strings.TrimSpace(os.Getenv("SFORUM_HOST_API_URL"))
	token := strings.TrimSpace(os.Getenv("SFORUM_HOST_API_TOKEN"))
	ext := strings.TrimSpace(os.Getenv("SFORUM_EXTENSION_ID"))
	if base == "" || token == "" || ext == "" {
		return nil, fmt.Errorf("hostapi env not configured")
	}
	return &Client{BaseURL: base, Token: token, ExtensionID: ext}, nil
}
