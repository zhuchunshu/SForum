package extensionsruntime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func TestValidatePluginRouteTargetAllowsLoopback(t *testing.T) {
	for _, raw := range []string{
		"http://127.0.0.1:43123",
		"http://localhost:8080/hooks",
		"https://[::1]:9443",
	} {
		if err := validatePluginRouteTarget(raw); err != nil {
			t.Fatalf("expected %q allowed, got %v", raw, err)
		}
	}
}

func TestIsPluginRouteTargetNone(t *testing.T) {
	for _, raw := range []string{"", "  ", "disabled", "DISABLED", "none", "None"} {
		if !isPluginRouteTargetNone(raw) {
			t.Fatalf("expected %q treated as no-route target", raw)
		}
	}
	for _, raw := range []string{"http://127.0.0.1:1", "disabled-http", "not-none"} {
		if isPluginRouteTargetNone(raw) {
			t.Fatalf("expected %q not treated as no-route target", raw)
		}
	}
}

func TestValidatePluginRouteTargetRejectsSSRF(t *testing.T) {
	for _, raw := range []string{
		"file:///etc/passwd",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.5:80/",
		"http://192.168.1.1/",
		"http://user:pass@127.0.0.1:1/",
		"ftp://127.0.0.1/",
		"disabled", // 无路由哨兵不走 validate；若误传入必须拒绝
	} {
		if err := validatePluginRouteTarget(raw); err == nil {
			t.Fatalf("expected %q rejected", raw)
		}
	}
}

func TestBuildPluginProcessEnvOmitsHostSecrets(t *testing.T) {
	env := buildPluginProcessEnv([]string{
		"PATH=/usr/bin",
		"HOME=/home/sforum",
		"DATABASE_URL=postgres://secret",
		"SESSION_HASH_SECRET=super-secret",
		"SFORUM_SETTING_HOST=smtp.example.com",
		"LANG=C.UTF-8",
		"RANDOM_JUNK=1",
	})
	joined := strings.Join(env, "\n")
	for _, want := range []string{"PATH=/usr/bin", "HOME=/home/sforum", "SFORUM_SETTING_HOST=smtp.example.com", "LANG=C.UTF-8"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected env to contain %q, got %v", want, env)
		}
	}
	for _, deny := range []string{"DATABASE_URL=", "SESSION_HASH_SECRET=", "RANDOM_JUNK="} {
		if strings.Contains(joined, deny) {
			t.Fatalf("env must not contain host secret/junk %q, got %v", deny, env)
		}
	}
}

func TestProtocolStarterRejectsUnsupportedRPC(t *testing.T) {
	starter := NewProtocolStarter(ProtocolStarterConfig{})
	extension := runtimeExtension("bad.protocol")
	extension.Manifest.Backend.RPC = "custom"
	_, err := starter.Start(context.Background(), extension)
	if err == nil {
		t.Fatal("expected unsupported protocol error")
	}
}

func TestProtocolStarterRequiresBackendEntry(t *testing.T) {
	starter := NewProtocolStarter(ProtocolStarterConfig{})
	extension := runtimeExtension("missing.entry")
	extension.Manifest.Backend.Entry = ""
	_, err := starter.Start(context.Background(), extension)
	if err == nil {
		t.Fatal("expected missing entry error")
	}
}

func TestProtocolStarterPerformsHashicorpHandshake(t *testing.T) {
	packageRoot := filepath.Join(t.TempDir(), "runtime.plugin", "1.0.0")
	filesRoot := filepath.Join(packageRoot, "files", "backend")
	if err := os.MkdirAll(filesRoot, 0o755); err != nil {
		t.Fatalf("create installed file tree: %v", err)
	}
	targetBinary := filepath.Join(filesRoot, "plugin")
	if err := os.WriteFile(targetBinary, []byte(helperPluginLauncher(t)), 0o755); err != nil {
		t.Fatalf("install helper plugin launcher: %v", err)
	}

	starter := NewProtocolStarter(ProtocolStarterConfig{Settings: staticPluginSettings{"runtime.plugin": {"host": "smtp.example.com"}}})
	extension := runtimeExtension("runtime.plugin")
	extension.PackagePath = filepath.Join(packageRoot, "package.zip")

	target, err := starter.Start(context.Background(), extension)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer starter.Stop(context.Background(), extension)
	if target.BaseURL != "http://127.0.0.1:43123" {
		t.Fatalf("unexpected route target: %#v", target)
	}
	response, err := starter.SendMail(context.Background(), extension.ID, MailProviderRequest{
		DeliveryID: "41", To: []string{"member@example.com"}, Subject: "Mention",
	})
	if err != nil {
		t.Fatalf("send mail rpc: %v", err)
	}
	if !response.OK || response.Reason != "smtp.accepted" {
		t.Fatalf("unexpected mail response: %#v", response)
	}
	providerProbe, err := starter.ProviderProbe(context.Background(), extension.ID, ProviderProbeRequest{Slot: "mail.provider"})
	if err != nil {
		t.Fatalf("provider probe rpc: %v", err)
	}
	if providerProbe.OK || providerProbe.Reason != "plugin.provider_probe_not_implemented" {
		t.Fatalf("expected provider probe not implemented, got %#v", providerProbe)
	}
	// E6.2：未实现存储的插件经 ProtocolNoop 返回明确 reason。
	probe, err := starter.StorageProbe(context.Background(), extension.ID, StorageProbeRequest{})
	if err != nil {
		t.Fatalf("storage probe rpc: %v", err)
	}
	if probe.OK || probe.Reason != "plugin.storage_not_implemented" {
		t.Fatalf("expected storage not implemented, got %#v", probe)
	}
	telemetry := starter.ProtocolTelemetry(extension.ID)
	if telemetry.ProtocolVersion != 1 || telemetry.Transport != "net/rpc" || !telemetry.Deprecated ||
		telemetry.StartCount != 1 || telemetry.CallCount != 3 || telemetry.LastCallAt == nil {
		t.Fatalf("unexpected v1 telemetry: %#v", telemetry)
	}
}

type staticPluginSettings map[string]map[string]string

func (s staticPluginSettings) ListSettings(_ context.Context, extensionID string) (map[string]string, error) {
	return s[extensionID], nil
}

func TestProtocolStarterHelperProcess(t *testing.T) {
	if os.Getenv("SFORUM_PLUGIN_HELPER") != "1" {
		return
	}
	ServeProtocolPlugin(helperProtocol{})
	os.Exit(0)
}

func helperPluginLauncher(t testing.TB) string {
	t.Helper()
	testBinary, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatalf("locate test binary: %v", err)
	}
	return "#!/bin/sh\nSFORUM_PLUGIN_HELPER=1 exec " + shellQuote(testBinary) + " -test.run=TestProtocolStarterHelperProcess -- \"$@\"\n"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

type helperProtocol struct {
	ProtocolNoop
}

func (helperProtocol) Health() (PluginHealth, error) {
	return PluginHealth{OK: true}, nil
}

func (helperProtocol) RouteTarget() (PluginRouteTarget, error) {
	return PluginRouteTarget{BaseURL: "http://127.0.0.1:43123"}, nil
}

func (helperProtocol) InvokeHook(PluginHookRequest) (PluginHookResponse, error) {
	return PluginHookResponse{OK: true}, nil
}

func (helperProtocol) SendMail(request MailProviderRequest) (MailProviderResponse, error) {
	if request.DeliveryID != "41" || len(request.To) != 1 {
		return MailProviderResponse{Classification: "permanent", Reason: "smtp.invalid_request"}, nil
	}
	if os.Getenv("SFORUM_SETTING_HOST") != "smtp.example.com" {
		return MailProviderResponse{Classification: "permanent", Reason: "smtp.settings_missing"}, nil
	}
	return MailProviderResponse{OK: true, Reason: "smtp.accepted"}, nil
}

var _ Starter = (*ProtocolStarter)(nil)
var _ = extensions.TypePlugin
