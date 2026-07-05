package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"net/rpc"
	"os"
	"os/exec"
	"sync"

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

	handshakeConfig = plugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "SFORUM_PLUGIN",
		MagicCookieValue: "sforum-plugin-v1",
	}
)

type ProtocolStarterConfig struct{}

type ProtocolStarter struct {
	mu        sync.Mutex
	clients   map[string]*plugin.Client
	protocols map[string]PluginProtocol
}

type PluginProtocol interface {
	Health() (PluginHealth, error)
	RouteTarget() (PluginRouteTarget, error)
	InvokeHook(PluginHookRequest) (PluginHookResponse, error)
}

type PluginHealth struct {
	OK bool
}

type PluginRouteTarget struct {
	BaseURL string
}

type PluginHookRequest struct {
	Name    string
	Payload map[string]any
}

type PluginHookResponse struct {
	OK      bool
	Reason  string
	Message string
}

type PluginEmptyRequest struct{}

func NewProtocolStarter(ProtocolStarterConfig) *ProtocolStarter {
	return &ProtocolStarter{clients: map[string]*plugin.Client{}, protocols: map[string]PluginProtocol{}}
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

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: handshakeConfig,
		Logger:          hclog.NewNullLogger(),
		Plugins: map[string]plugin.Plugin{
			pluginProtocolName: &netRPCPlugin{},
		},
		Cmd:              exec.CommandContext(ctx, path),
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
	if err != nil || target.BaseURL == "" {
		client.Kill()
		if err != nil {
			return RouteTarget{}, err
		}
		return RouteTarget{}, fmt.Errorf("plugin route target is empty")
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
	return RouteTarget{BaseURL: target.BaseURL}, nil
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
	return nil
}

func (s *ProtocolStarter) InvokeHook(_ context.Context, extension extensions.Extension, input HookInput) HookResult {
	s.mu.Lock()
	protocol := s.protocols[extension.ID]
	s.mu.Unlock()
	if protocol == nil {
		return HookResult{OK: false, Reason: "extension.runtime_unavailable", Message: "Plugin runtime is not available."}
	}
	response, err := protocol.InvokeHook(PluginHookRequest{Name: input.Name, Payload: input.Payload})
	if err != nil {
		return HookResult{OK: false, Reason: "extension.hook_failed", Message: err.Error()}
	}
	return HookResult{OK: response.OK, Reason: response.Reason, Message: response.Message}
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
