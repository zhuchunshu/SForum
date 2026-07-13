package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

const (
	hashicorpGoPluginRPC = "hashicorp-go-plugin"
	pluginProtocolName   = "sforum-plugin-v1"
)

var (
	ErrUnsupportedProtocol = errors.New("unsupported plugin protocol")
	// ErrUnsafePluginRouteTarget 插件回报的 BaseURL 不允许被 API 代理（SSRF 防护）。
	ErrUnsafePluginRouteTarget = errors.New("plugin route target is not allowed")

	handshakeConfig = plugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "SFORUM_PLUGIN",
		MagicCookieValue: "sforum-plugin-v1",
	}

	// 插件子进程环境白名单：不继承 DATABASE_URL / SESSION_* 等宿主密钥。
	pluginEnvAllowlist = map[string]bool{
		"PATH": true, "HOME": true, "TMPDIR": true, "TEMP": true, "TMP": true,
		"LANG": true, "LC_ALL": true, "LC_CTYPE": true, "TZ": true,
		// go-plugin / 测试 helper 需要的变量由 go-plugin 自行注入；宿主仅保留基础运行环境。
		"SFORUM_PLUGIN_HELPER": true,
	}
)

type PluginSettings interface {
	ListSettings(context.Context, string) (map[string]string, error)
}

// HostAPIRegistrar 在插件启动时签发 Host API 凭证（F2.2）。
type HostAPIRegistrar interface {
	RegisterExtension(extensionID string) (token string, env []string, err error)
	UnregisterExtension(extensionID string)
}

type ProtocolStarterConfig struct {
	Settings PluginSettings
	// HostAPI 可选；注入后向子进程写入 SFORUM_HOST_API_* 环境变量。
	HostAPI HostAPIRegistrar
}

type ProtocolStarter struct {
	mu        sync.Mutex
	clients   map[string]*plugin.Client
	protocols map[string]PluginProtocol
	settings  PluginSettings
	hostAPI   HostAPIRegistrar
}

type PluginProtocol interface {
	Health() (PluginHealth, error)
	RouteTarget() (PluginRouteTarget, error)
	InvokeHook(PluginHookRequest) (PluginHookResponse, error)
	ProviderProbe(ProviderProbeRequest) (ProviderProbeResponse, error)
	SendMail(MailProviderRequest) (MailProviderResponse, error)
	// 附件存储槽 attachment.storage.provider（E6.2，分块 Put/Open）。
	StoragePutBegin(StoragePutBeginRequest) (StorageSessionResponse, error)
	StoragePutChunk(StoragePutChunkRequest) (StorageResult, error)
	StorageOpen(StorageOpenRequest) (StorageSessionResponse, error)
	StorageGetChunk(StorageGetChunkRequest) (StorageGetChunkResponse, error)
	StorageClose(StorageCloseRequest) (StorageResult, error)
	StorageDelete(StorageObjectRequest) (StorageResult, error)
	StorageStat(StorageStatRequest) (StorageStatResponse, error)
	StorageExists(StorageExistsRequest) (StorageExistsResponse, error)
	StoragePublicURL(StoragePublicURLRequest) (StorageURLResponse, error)
	StorageSignedURL(StorageSignedURLRequest) (StorageURLResponse, error)
	StorageProbe(StorageProbeRequest) (StorageProbeResponse, error)
}

type PluginHealth struct {
	OK bool
}

type PluginRouteTarget struct {
	BaseURL string
}

type PluginHookRequest struct {
	Name          string
	Kind          string
	DeliveryID    int64
	CorrelationID string
	TimeoutMS     int
	Payload       map[string]any
	PatchFields   []string
}

type PluginHookResponse struct {
	OK      bool
	Reason  string
	Message string
	Patch   map[string]any
}

type ProviderProbeRequest struct {
	Slot string
}

type ProviderProbeResponse struct {
	OK          bool
	Reason      string
	Message     string
	Details     map[string]string
	Suggestions []string
}

type MailProviderRequest struct {
	DeliveryID    string
	CorrelationID string
	FromAddress   string
	FromName      string
	To            []string
	Subject       string
	TextBody      string
	HTMLBody      string
}

type MailProviderResponse struct {
	OK             bool
	Classification string
	Reason         string
	Message        string
}

type PluginEmptyRequest struct{}

func NewProtocolStarter(config ProtocolStarterConfig) *ProtocolStarter {
	return &ProtocolStarter{
		clients:   map[string]*plugin.Client{},
		protocols: map[string]PluginProtocol{},
		settings:  config.Settings,
		hostAPI:   config.HostAPI,
	}
}

func (s *ProtocolStarter) Start(ctx context.Context, extension extensions.Extension) (RouteTarget, error) {
	if extension.Manifest.Backend.RPC != "" && extension.Manifest.Backend.RPC != hashicorpGoPluginRPC {
		return RouteTarget{}, ErrUnsupportedProtocol
	}
	if extension.Manifest.Backend.Entry == "" {
		return RouteTarget{}, fmt.Errorf("backend entry is required")
	}
	if extension.Manifest.Backend.ProtocolVersion > 0 && uint(extension.Manifest.Backend.ProtocolVersion) != handshakeConfig.ProtocolVersion {
		return RouteTarget{}, ErrUnsupportedProtocol
	}
	path, ok := extensions.InstalledFilePathForRuntime(extension, extension.Manifest.Backend.Entry)
	if !ok {
		return RouteTarget{}, extensions.ErrInvalidManifest
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return RouteTarget{}, fmt.Errorf("backend entry %s is not available", extension.Manifest.Backend.Entry)
	}

	cmd := exec.CommandContext(ctx, path)
	cmd.Env = buildPluginProcessEnv(os.Environ())
	if s.settings != nil {
		values, err := s.settings.ListSettings(ctx, extension.ID)
		if err != nil {
			return RouteTarget{}, fmt.Errorf("load plugin settings: %w", err)
		}
		keys := make([]string, 0, len(values))
		for key := range values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			cmd.Env = append(cmd.Env, pluginSettingEnvName(key)+"="+values[key])
		}
	}
	// F2.2：为子进程签发 Host API loopback 凭证。
	if s.hostAPI != nil {
		if _, hostEnv, err := s.hostAPI.RegisterExtension(extension.ID); err != nil {
			return RouteTarget{}, fmt.Errorf("register host api: %w", err)
		} else {
			cmd.Env = append(cmd.Env, hostEnv...)
		}
	}
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: handshakeConfig,
		Logger:          hclog.NewNullLogger(),
		Plugins: map[string]plugin.Plugin{
			pluginProtocolName: &netRPCPlugin{},
		},
		Cmd:              cmd,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC},
	})
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return RouteTarget{}, err
	}
	raw, err := rpcClient.Dispense(pluginProtocolName)
	if err != nil {
		client.Kill()
		return RouteTarget{}, err
	}
	protocol, ok := raw.(PluginProtocol)
	if !ok {
		client.Kill()
		return RouteTarget{}, fmt.Errorf("plugin protocol implementation mismatch")
	}
	health, err := protocol.Health()
	if err != nil || !health.OK {
		client.Kill()
		if err != nil {
			return RouteTarget{}, err
		}
		return RouteTarget{}, fmt.Errorf("plugin health check failed")
	}
	target, err := protocol.RouteTarget()
	if err != nil {
		client.Kill()
		return RouteTarget{}, err
	}
	// 纯 provider/RPC 插件（如 SMTP）不暴露 HTTP 路由：允许空或历史哨兵值。
	baseURL := strings.TrimSpace(target.BaseURL)
	if isPluginRouteTargetNone(baseURL) {
		baseURL = ""
	} else if err := validatePluginRouteTarget(baseURL); err != nil {
		client.Kill()
		return RouteTarget{}, err
	}

	s.mu.Lock()
	if s.clients == nil {
		s.clients = map[string]*plugin.Client{}
	}
	if s.protocols == nil {
		s.protocols = map[string]PluginProtocol{}
	}
	if previous := s.clients[extension.ID]; previous != nil {
		previous.Kill()
	}
	s.clients[extension.ID] = client
	s.protocols[extension.ID] = protocol
	s.mu.Unlock()
	return RouteTarget{BaseURL: baseURL}, nil
}

// isPluginRouteTargetNone 表示插件不提供可代理的 HTTP BaseURL。
// 兼容旧哨兵 "disabled"（SSRF 加固前 SMTP 等插件使用）。
func isPluginRouteTargetNone(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "disabled", "none":
		return true
	default:
		return false
	}
}

// validatePluginRouteTarget 限制插件 RouteTarget 仅允许 loopback http(s)，阻断 SSRF。
// 调用方应先用 isPluginRouteTargetNone 处理无路由插件。
func validatePluginRouteTarget(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnsafePluginRouteTarget, err)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("%w: scheme %q not allowed", ErrUnsafePluginRouteTarget, parsed.Scheme)
	}
	if parsed.User != nil {
		return fmt.Errorf("%w: userinfo not allowed", ErrUnsafePluginRouteTarget)
	}
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("%w: empty host", ErrUnsafePluginRouteTarget)
	}
	// 字面量 loopback 主机名直接放行，避免依赖解析器。
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("%w: resolve %q: %v", ErrUnsafePluginRouteTarget, host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("%w: no addresses for %q", ErrUnsafePluginRouteTarget, host)
	}
	for _, ip := range ips {
		if !ip.IsLoopback() {
			return fmt.Errorf("%w: host %q resolves to non-loopback %s", ErrUnsafePluginRouteTarget, host, ip)
		}
		// 双保险：拒绝 link-local / 未指定地址（IsLoopback 通常已排除）。
		if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("%w: host %q resolves to disallowed address %s", ErrUnsafePluginRouteTarget, host, ip)
		}
	}
	return nil
}

// buildPluginProcessEnv 从宿主环境中挑选最小白名单，并保留已有 SFORUM_SETTING_*。
// 不把 DATABASE_URL、SESSION_HASH_SECRET 等密钥传给插件子进程。
func buildPluginProcessEnv(hostEnv []string) []string {
	out := make([]string, 0, 16)
	for _, entry := range hostEnv {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if pluginEnvAllowlist[key] || strings.HasPrefix(key, "SFORUM_SETTING_") {
			out = append(out, entry)
		}
	}
	return out
}

func pluginSettingEnvName(key string) string {
	var value strings.Builder
	value.WriteString("SFORUM_SETTING_")
	for _, char := range strings.ToUpper(key) {
		if char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' {
			value.WriteRune(char)
		} else {
			value.WriteByte('_')
		}
	}
	return value.String()
}

func (s *ProtocolStarter) Stop(_ context.Context, extension extensions.Extension) error {
	s.mu.Lock()
	client := s.clients[extension.ID]
	delete(s.clients, extension.ID)
	delete(s.protocols, extension.ID)
	s.mu.Unlock()
	if client != nil {
		client.Kill()
	}
	if s.hostAPI != nil {
		s.hostAPI.UnregisterExtension(extension.ID)
	}
	return nil
}

func (s *ProtocolStarter) InvokeHook(ctx context.Context, extension extensions.Extension, input HookInput) HookResult {
	s.mu.Lock()
	protocol := s.protocols[extension.ID]
	s.mu.Unlock()
	if protocol == nil {
		return HookResult{OK: false, Reason: "extension.runtime_unavailable", Message: "Plugin runtime is not available."}
	}
	timeoutMS := int(input.Timeout / time.Millisecond)
	if timeoutMS <= 0 && input.Timeout > 0 {
		timeoutMS = 1
	}
	req := PluginHookRequest{
		Name:          input.Name,
		Kind:          input.Kind,
		DeliveryID:    input.DeliveryID,
		CorrelationID: input.CorrelationID,
		TimeoutMS:     timeoutMS,
		Payload:       input.Payload,
		PatchFields:   input.PatchFields,
	}
	// net/rpc 无原生 context；用 goroutine + select 保证宿主 deadline 生效（F2.3）。
	type outcome struct {
		resp PluginHookResponse
		err  error
	}
	done := make(chan outcome, 1)
	go func() {
		resp, err := protocol.InvokeHook(req)
		done <- outcome{resp: resp, err: err}
	}()
	select {
	case <-ctx.Done():
		return HookResult{
			OK:      false,
			Reason:  "extension.hook_timeout",
			Message: "Plugin hook exceeded the host timeout. Heavy work must enqueue a job.",
		}
	case out := <-done:
		if out.err != nil {
			return HookResult{OK: false, Reason: "extension.hook_failed", Message: out.err.Error()}
		}
		return HookResult{OK: out.resp.OK, Reason: out.resp.Reason, Message: out.resp.Message, Patch: out.resp.Patch}
	}
}

func (s *ProtocolStarter) SendMail(ctx context.Context, extensionID string, request MailProviderRequest) (MailProviderResponse, error) {
	s.mu.Lock()
	protocol := s.protocols[extensionID]
	s.mu.Unlock()
	if protocol == nil {
		return MailProviderResponse{}, extensions.ErrRuntimeUnavailable
	}
	type outcome struct {
		resp MailProviderResponse
		err  error
	}
	done := make(chan outcome, 1)
	go func() {
		resp, err := protocol.SendMail(request)
		done <- outcome{resp: resp, err: err}
	}()
	select {
	case <-ctx.Done():
		return MailProviderResponse{
			OK:             false,
			Classification: "temporary",
			Reason:         "extension.hook_timeout",
			Message:        "Mail provider call exceeded the host timeout.",
		}, nil
	case out := <-done:
		return out.resp, out.err
	}
}

func (s *ProtocolStarter) ProviderProbe(ctx context.Context, extensionID string, request ProviderProbeRequest) (ProviderProbeResponse, error) {
	return callStorage(ctx, s.protocolFor(extensionID), func(protocol PluginProtocol) (ProviderProbeResponse, error) {
		return protocol.ProviderProbe(request)
	}, ProviderProbeResponse{Reason: "extension.action_timeout", Message: "Provider probe exceeded the host timeout."})
}

// protocolFor 返回已启动扩展的协议面；未运行时返回 nil。
func (s *ProtocolStarter) protocolFor(extensionID string) PluginProtocol {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.protocols[extensionID]
}

// callStorage 在 ctx 截止前执行一次存储 RPC（net/rpc 无原生 context）。
// 超时返回 onTimeout（err=nil），与 SendMail 一致，便于宿主按 OK/Reason 处理。
func callStorage[T any](ctx context.Context, protocol PluginProtocol, fn func(PluginProtocol) (T, error), onTimeout T) (T, error) {
	var zero T
	if protocol == nil {
		return zero, extensions.ErrRuntimeUnavailable
	}
	type outcome struct {
		resp T
		err  error
	}
	done := make(chan outcome, 1)
	go func() {
		resp, err := fn(protocol)
		done <- outcome{resp: resp, err: err}
	}()
	select {
	case <-ctx.Done():
		return onTimeout, nil
	case out := <-done:
		return out.resp, out.err
	}
}

func (s *ProtocolStarter) StoragePutBegin(ctx context.Context, extensionID string, request StoragePutBeginRequest) (StorageSessionResponse, error) {
	return callStorage(ctx, s.protocolFor(extensionID), func(p PluginProtocol) (StorageSessionResponse, error) {
		return p.StoragePutBegin(request)
	}, StorageSessionResponse{Reason: "extension.hook_timeout", Message: "Storage PutBegin exceeded the host timeout."})
}

func (s *ProtocolStarter) StoragePutChunk(ctx context.Context, extensionID string, request StoragePutChunkRequest) (StorageResult, error) {
	return callStorage(ctx, s.protocolFor(extensionID), func(p PluginProtocol) (StorageResult, error) {
		return p.StoragePutChunk(request)
	}, StorageResult{Reason: "extension.hook_timeout", Message: "Storage PutChunk exceeded the host timeout."})
}

func (s *ProtocolStarter) StorageOpen(ctx context.Context, extensionID string, request StorageOpenRequest) (StorageSessionResponse, error) {
	return callStorage(ctx, s.protocolFor(extensionID), func(p PluginProtocol) (StorageSessionResponse, error) {
		return p.StorageOpen(request)
	}, StorageSessionResponse{Reason: "extension.hook_timeout", Message: "Storage Open exceeded the host timeout."})
}

func (s *ProtocolStarter) StorageGetChunk(ctx context.Context, extensionID string, request StorageGetChunkRequest) (StorageGetChunkResponse, error) {
	return callStorage(ctx, s.protocolFor(extensionID), func(p PluginProtocol) (StorageGetChunkResponse, error) {
		return p.StorageGetChunk(request)
	}, StorageGetChunkResponse{Reason: "extension.hook_timeout", Message: "Storage GetChunk exceeded the host timeout."})
}

func (s *ProtocolStarter) StorageClose(ctx context.Context, extensionID string, request StorageCloseRequest) (StorageResult, error) {
	return callStorage(ctx, s.protocolFor(extensionID), func(p PluginProtocol) (StorageResult, error) {
		return p.StorageClose(request)
	}, StorageResult{Reason: "extension.hook_timeout", Message: "Storage Close exceeded the host timeout."})
}

func (s *ProtocolStarter) StorageDelete(ctx context.Context, extensionID string, request StorageObjectRequest) (StorageResult, error) {
	return callStorage(ctx, s.protocolFor(extensionID), func(p PluginProtocol) (StorageResult, error) {
		return p.StorageDelete(request)
	}, StorageResult{Reason: "extension.hook_timeout", Message: "Storage Delete exceeded the host timeout."})
}

func (s *ProtocolStarter) StorageStat(ctx context.Context, extensionID string, request StorageStatRequest) (StorageStatResponse, error) {
	return callStorage(ctx, s.protocolFor(extensionID), func(p PluginProtocol) (StorageStatResponse, error) {
		return p.StorageStat(request)
	}, StorageStatResponse{Reason: "extension.hook_timeout", Message: "Storage Stat exceeded the host timeout."})
}

func (s *ProtocolStarter) StorageExists(ctx context.Context, extensionID string, request StorageExistsRequest) (StorageExistsResponse, error) {
	return callStorage(ctx, s.protocolFor(extensionID), func(p PluginProtocol) (StorageExistsResponse, error) {
		return p.StorageExists(request)
	}, StorageExistsResponse{Reason: "extension.hook_timeout", Message: "Storage Exists exceeded the host timeout."})
}

func (s *ProtocolStarter) StoragePublicURL(ctx context.Context, extensionID string, request StoragePublicURLRequest) (StorageURLResponse, error) {
	return callStorage(ctx, s.protocolFor(extensionID), func(p PluginProtocol) (StorageURLResponse, error) {
		return p.StoragePublicURL(request)
	}, StorageURLResponse{Reason: "extension.hook_timeout", Message: "Storage PublicURL exceeded the host timeout."})
}

func (s *ProtocolStarter) StorageSignedURL(ctx context.Context, extensionID string, request StorageSignedURLRequest) (StorageURLResponse, error) {
	return callStorage(ctx, s.protocolFor(extensionID), func(p PluginProtocol) (StorageURLResponse, error) {
		return p.StorageSignedURL(request)
	}, StorageURLResponse{Reason: "extension.hook_timeout", Message: "Storage SignedURL exceeded the host timeout."})
}

func (s *ProtocolStarter) StorageProbe(ctx context.Context, extensionID string, request StorageProbeRequest) (StorageProbeResponse, error) {
	return callStorage(ctx, s.protocolFor(extensionID), func(p PluginProtocol) (StorageProbeResponse, error) {
		return p.StorageProbe(request)
	}, StorageProbeResponse{Reason: "extension.hook_timeout", Message: "Storage Probe exceeded the host timeout."})
}

func ServeProtocolPlugin(impl PluginProtocol) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins: map[string]plugin.Plugin{
			pluginProtocolName: &netRPCPlugin{Impl: impl},
		},
	})
}

type netRPCPlugin struct {
	Impl PluginProtocol
}

func (p *netRPCPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	if p.Impl == nil {
		return nil, fmt.Errorf("plugin protocol implementation is required")
	}
	return &netRPCServer{Impl: p.Impl}, nil
}

func (*netRPCPlugin) Client(_ *plugin.MuxBroker, client *rpc.Client) (interface{}, error) {
	return &netRPCClient{client: client}, nil
}

type netRPCClient struct {
	client *rpc.Client
}

func (c *netRPCClient) Health() (PluginHealth, error) {
	var response PluginHealth
	err := c.client.Call("Plugin.Health", PluginEmptyRequest{}, &response)
	return response, err
}

func (c *netRPCClient) RouteTarget() (PluginRouteTarget, error) {
	var response PluginRouteTarget
	err := c.client.Call("Plugin.RouteTarget", PluginEmptyRequest{}, &response)
	return response, err
}

func (c *netRPCClient) InvokeHook(input PluginHookRequest) (PluginHookResponse, error) {
	var response PluginHookResponse
	err := c.client.Call("Plugin.InvokeHook", input, &response)
	return response, err
}

func (c *netRPCClient) ProviderProbe(input ProviderProbeRequest) (ProviderProbeResponse, error) {
	var response ProviderProbeResponse
	err := c.client.Call("Plugin.ProviderProbe", input, &response)
	return response, err
}

func (c *netRPCClient) SendMail(input MailProviderRequest) (MailProviderResponse, error) {
	var response MailProviderResponse
	err := c.client.Call("Plugin.SendMail", input, &response)
	return response, err
}

func (c *netRPCClient) StoragePutBegin(input StoragePutBeginRequest) (StorageSessionResponse, error) {
	var response StorageSessionResponse
	err := c.client.Call("Plugin.StoragePutBegin", input, &response)
	return response, err
}

func (c *netRPCClient) StoragePutChunk(input StoragePutChunkRequest) (StorageResult, error) {
	var response StorageResult
	err := c.client.Call("Plugin.StoragePutChunk", input, &response)
	return response, err
}

func (c *netRPCClient) StorageOpen(input StorageOpenRequest) (StorageSessionResponse, error) {
	var response StorageSessionResponse
	err := c.client.Call("Plugin.StorageOpen", input, &response)
	return response, err
}

func (c *netRPCClient) StorageGetChunk(input StorageGetChunkRequest) (StorageGetChunkResponse, error) {
	var response StorageGetChunkResponse
	err := c.client.Call("Plugin.StorageGetChunk", input, &response)
	return response, err
}

func (c *netRPCClient) StorageClose(input StorageCloseRequest) (StorageResult, error) {
	var response StorageResult
	err := c.client.Call("Plugin.StorageClose", input, &response)
	return response, err
}

func (c *netRPCClient) StorageDelete(input StorageObjectRequest) (StorageResult, error) {
	var response StorageResult
	err := c.client.Call("Plugin.StorageDelete", input, &response)
	return response, err
}

func (c *netRPCClient) StorageStat(input StorageStatRequest) (StorageStatResponse, error) {
	var response StorageStatResponse
	err := c.client.Call("Plugin.StorageStat", input, &response)
	return response, err
}

func (c *netRPCClient) StorageExists(input StorageExistsRequest) (StorageExistsResponse, error) {
	var response StorageExistsResponse
	err := c.client.Call("Plugin.StorageExists", input, &response)
	return response, err
}

func (c *netRPCClient) StoragePublicURL(input StoragePublicURLRequest) (StorageURLResponse, error) {
	var response StorageURLResponse
	err := c.client.Call("Plugin.StoragePublicURL", input, &response)
	return response, err
}

func (c *netRPCClient) StorageSignedURL(input StorageSignedURLRequest) (StorageURLResponse, error) {
	var response StorageURLResponse
	err := c.client.Call("Plugin.StorageSignedURL", input, &response)
	return response, err
}

func (c *netRPCClient) StorageProbe(input StorageProbeRequest) (StorageProbeResponse, error) {
	var response StorageProbeResponse
	err := c.client.Call("Plugin.StorageProbe", input, &response)
	return response, err
}

type netRPCServer struct {
	Impl PluginProtocol
}

func (s *netRPCServer) Health(_ PluginEmptyRequest, response *PluginHealth) error {
	health, err := s.Impl.Health()
	*response = health
	return err
}

func (s *netRPCServer) RouteTarget(_ PluginEmptyRequest, response *PluginRouteTarget) error {
	target, err := s.Impl.RouteTarget()
	*response = target
	return err
}

func (s *netRPCServer) InvokeHook(input PluginHookRequest, response *PluginHookResponse) error {
	result, err := s.Impl.InvokeHook(input)
	*response = result
	return err
}

func (s *netRPCServer) ProviderProbe(input ProviderProbeRequest, response *ProviderProbeResponse) error {
	result, err := s.Impl.ProviderProbe(input)
	*response = result
	return err
}

func (s *netRPCServer) SendMail(input MailProviderRequest, response *MailProviderResponse) error {
	result, err := s.Impl.SendMail(input)
	*response = result
	return err
}

func (s *netRPCServer) StoragePutBegin(input StoragePutBeginRequest, response *StorageSessionResponse) error {
	result, err := s.Impl.StoragePutBegin(input)
	*response = result
	return err
}

func (s *netRPCServer) StoragePutChunk(input StoragePutChunkRequest, response *StorageResult) error {
	result, err := s.Impl.StoragePutChunk(input)
	*response = result
	return err
}

func (s *netRPCServer) StorageOpen(input StorageOpenRequest, response *StorageSessionResponse) error {
	result, err := s.Impl.StorageOpen(input)
	*response = result
	return err
}

func (s *netRPCServer) StorageGetChunk(input StorageGetChunkRequest, response *StorageGetChunkResponse) error {
	result, err := s.Impl.StorageGetChunk(input)
	*response = result
	return err
}

func (s *netRPCServer) StorageClose(input StorageCloseRequest, response *StorageResult) error {
	result, err := s.Impl.StorageClose(input)
	*response = result
	return err
}

func (s *netRPCServer) StorageDelete(input StorageObjectRequest, response *StorageResult) error {
	result, err := s.Impl.StorageDelete(input)
	*response = result
	return err
}

func (s *netRPCServer) StorageStat(input StorageStatRequest, response *StorageStatResponse) error {
	result, err := s.Impl.StorageStat(input)
	*response = result
	return err
}

func (s *netRPCServer) StorageExists(input StorageExistsRequest, response *StorageExistsResponse) error {
	result, err := s.Impl.StorageExists(input)
	*response = result
	return err
}

func (s *netRPCServer) StoragePublicURL(input StoragePublicURLRequest, response *StorageURLResponse) error {
	result, err := s.Impl.StoragePublicURL(input)
	*response = result
	return err
}

func (s *netRPCServer) StorageSignedURL(input StorageSignedURLRequest, response *StorageURLResponse) error {
	result, err := s.Impl.StorageSignedURL(input)
	*response = result
	return err
}

func (s *netRPCServer) StorageProbe(input StorageProbeRequest, response *StorageProbeResponse) error {
	result, err := s.Impl.StorageProbe(input)
	*response = result
	return err
}
