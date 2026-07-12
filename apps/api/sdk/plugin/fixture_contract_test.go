package plugin_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Capabilities"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	pluginsdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// apps/api/sdk/plugin -> repo root
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
}

func fixtureRoot(t *testing.T, rel string) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "extensions/fixtures", rel)
}

func TestFixtureEventsContract(t *testing.T) {
	root := fixtureRoot(t, "plugins/sforum-contract-events")
	report, err := pluginsdk.LoadAndTest(root, pluginsdk.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK {
		t.Fatalf("expected pass, got errors=%d checks=%#v", report.Errors, report.Checks)
	}
	if report.Manifest.ID != "sforum.contract.events" {
		t.Fatalf("id=%s", report.Manifest.ID)
	}
	// 事件与贡献点应出现在 ok checks 中。
	codes := checkCodes(report)
	for _, want := range []string{"manifest.ok", "event.known", "contribution.point_ok"} {
		if !codes[want] {
			t.Fatalf("missing check %s in %#v", want, report.Checks)
		}
	}
}

func TestFixtureSchedulesContract(t *testing.T) {
	root := fixtureRoot(t, "plugins/sforum-contract-schedules")
	report, err := pluginsdk.LoadAndTest(root, pluginsdk.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK {
		t.Fatalf("expected pass: %#v", report.Checks)
	}
	// 宿主 schedule 目录必须非空；fixture 本身不声明 cron。
	schedules := pluginsdk.CoreSchedules()
	if len(schedules) == 0 {
		t.Fatal("core schedules empty")
	}
	foundIdentity := false
	for _, s := range schedules {
		if s.ID == "identity.cleanup_sessions" {
			foundIdentity = true
			break
		}
	}
	if !foundIdentity {
		t.Fatalf("missing identity.cleanup_sessions in %#v", schedules)
	}
}

func TestFixtureHostAPIManifestContract(t *testing.T) {
	root := fixtureRoot(t, "plugins/sforum-contract-hostapi")
	// 未构建二进制时允许 skip；完整 runtime 见 TestFixtureHostAPIRuntimeHandshake。
	report, err := pluginsdk.LoadAndTest(root, pluginsdk.Options{SkipBackendBinary: true})
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK {
		t.Fatalf("expected pass with skip binary: %#v", report.Checks)
	}
	keys, implied := extensionmanifest.ResolvedCapabilities(report.Manifest)
	set := capabilities.NewSet(keys)
	if !set.Has(capabilities.HostAPI) || !set.Has(capabilities.JobsEnqueue) {
		t.Fatalf("caps=%v implied=%v", keys, implied)
	}
	// 显式声明不应标为 implied。
	if implied[capabilities.HostAPI] {
		t.Fatal("host.api was explicit, should not be implied")
	}
}

func TestFixtureHostAPIRuntimeHandshake(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping plugin binary build in short mode")
	}
	fixturePkg := fixtureRoot(t, "plugins/sforum-contract-hostapi")
	backendDir := filepath.Join(fixturePkg, "backend")

	// 在 temp 安装树中构建 plugin 二进制，避免污染 fixtures 目录。
	packageRoot := filepath.Join(t.TempDir(), "sforum.contract.hostapi", "1.0.0")
	filesBackend := filepath.Join(packageRoot, "files", "backend")
	if err := os.MkdirAll(filesBackend, 0o755); err != nil {
		t.Fatal(err)
	}
	binaryPath := filepath.Join(filesBackend, "plugin")
	build := exec.Command("go", "build", "-o", binaryPath, ".")
	build.Dir = backendDir
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fixture plugin: %v\n%s", err, out)
	}

	// Host API 网关：授予 host.api。
	svc := hostapi.New(hostapi.Config{
		Capabilities: staticCaps{set: capabilities.NewSet([]string{capabilities.HostAPI, capabilities.SettingsOwn, capabilities.JobsEnqueue})},
	})
	gw := hostapi.NewGateway(svc)
	t.Cleanup(func() { _ = gw.Close() })

	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		HostAPI: gw,
	})
	// PackageDigest 为空 + 非 builtin → 从 package.zip 旁 files/ 解析 entry。
	extension := extensions.Extension{
		ID:          "sforum.contract.hostapi",
		Source:      extensions.SourceUploaded,
		PackagePath: filepath.Join(packageRoot, "package.zip"),
		Manifest: extensionmanifest.Manifest{
			ID:      "sforum.contract.hostapi",
			Type:    extensionmanifest.TypePlugin,
			Version: "1.0.0",
			Backend: extensionmanifest.ManifestBackend{
				Entry:           "backend/plugin",
				RPC:             "hashicorp-go-plugin",
				ProtocolVersion: 1,
			},
			Events: []extensionmanifest.ManifestEvent{
				{Name: "topic.created", Kind: "observe"},
				{Name: "topic.before_create", Kind: "filter"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	target, err := starter.Start(ctx, extension)
	if err != nil {
		t.Fatalf("Start fixture plugin: %v", err)
	}
	t.Cleanup(func() { _ = starter.Stop(context.Background(), extension) })
	if target.BaseURL != "" {
		t.Fatalf("fixture should not expose HTTP routes, got %q", target.BaseURL)
	}

	// Health 已在 Start 内调用（含 Host API Ping）。再测 observe + filter hooks。
	observe := starter.InvokeHook(ctx, extension, extensionsruntime.HookInput{
		Name: "topic.created", Kind: "observe", Timeout: time.Second,
		Payload: map[string]any{"topicId": int64(1)},
	})
	if !observe.OK {
		t.Fatalf("observe hook: %#v", observe)
	}
	filter := starter.InvokeHook(ctx, extension, extensionsruntime.HookInput{
		Name: "topic.before_create", Kind: "filter", Timeout: time.Second,
		Payload: map[string]any{"title": "x"}, PatchFields: []string{"title"},
	})
	if !filter.OK {
		t.Fatalf("filter hook: %#v", filter)
	}
}

type staticCaps struct {
	set capabilities.Set
}

func (s staticCaps) CapabilitiesFor(context.Context, string) (capabilities.Set, error) {
	return s.set, nil
}

func (s staticCaps) DeclaredJobKinds(context.Context, string) ([]string, error) {
	return []string{"sforum.contract.hostapi.demo"}, nil
}

func checkCodes(report pluginsdk.Report) map[string]bool {
	out := map[string]bool{}
	for _, c := range report.Checks {
		out[c.Code] = true
	}
	return out
}
