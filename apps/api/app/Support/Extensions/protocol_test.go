package extensionsruntime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

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

	starter := NewProtocolStarter(ProtocolStarterConfig{})
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
}

func TestProtocolStarterHelperProcess(t *testing.T) {
	if os.Getenv("SFORUM_PLUGIN_HELPER") != "1" {
		return
	}
	ServeProtocolPlugin(helperProtocol{})
	os.Exit(0)
}

func helperPluginLauncher(t *testing.T) string {
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

type helperProtocol struct{}

func (helperProtocol) Health() (PluginHealth, error) {
	return PluginHealth{OK: true}, nil
}

func (helperProtocol) RouteTarget() (PluginRouteTarget, error) {
	return PluginRouteTarget{BaseURL: "http://127.0.0.1:43123"}, nil
}

func (helperProtocol) InvokeHook(PluginHookRequest) (PluginHookResponse, error) {
	return PluginHookResponse{OK: true}, nil
}

var _ Starter = (*ProtocolStarter)(nil)
var _ = extensions.TypePlugin
