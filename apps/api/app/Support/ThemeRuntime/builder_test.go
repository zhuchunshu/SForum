package themeruntime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuilderWritesCurrentReleaseAtomically(t *testing.T) {
	root := t.TempDir()
	server := filepath.Join(root, "releases", "1", ".output", "server", "index.mjs")
	if err := os.MkdirAll(filepath.Dir(server), 0o755); err != nil {
		t.Fatalf("mkdir server: %v", err)
	}
	if err := os.WriteFile(server, []byte("console.log('ok')\n"), 0o644); err != nil {
		t.Fatalf("write server: %v", err)
	}
	layer := filepath.Join(root, "storage", "extensions", "starter.theme", "1.0.0", "files", "layer")
	if err := os.MkdirAll(layer, 0o755); err != nil {
		t.Fatalf("mkdir layer: %v", err)
	}
	builder := NewBuilder(Config{ReleaseRoot: root})
	if err := builder.WriteCurrent(context.Background(), CurrentRelease{
		ReleaseID:   1,
		ExtensionID: "starter.theme",
		Mode:        CurrentModeUploaded,
		Server:      server,
		LayerPath:   layer,
	}); err != nil {
		t.Fatalf("write current: %v", err)
	}
	var current CurrentRelease
	raw, err := os.ReadFile(filepath.Join(root, "current.json"))
	if err != nil {
		t.Fatalf("read current: %v", err)
	}
	if err := json.Unmarshal(raw, &current); err != nil {
		t.Fatalf("decode current: %v", err)
	}
	if current.ExtensionID != "starter.theme" {
		t.Fatalf("unexpected extensionId: %q", current.ExtensionID)
	}
	if current.Mode != CurrentModeUploaded {
		t.Fatalf("expected uploaded mode, got %q", current.Mode)
	}
	if current.Server != server {
		t.Fatalf("expected absolute server path %q, got %q", server, current.Server)
	}
	if !filepath.IsAbs(current.Server) {
		t.Fatalf("server must be absolute, got %q", current.Server)
	}
	if current.LayerPath != layer {
		t.Fatalf("expected layerPath %q, got %q", layer, current.LayerPath)
	}
	if !filepath.IsAbs(current.LayerPath) {
		t.Fatalf("layerPath must be absolute, got %q", current.LayerPath)
	}
	if current.ActivatedAt == "" {
		t.Fatal("expected non-empty activatedAt")
	}
}

func TestBuilderWritesDefaultCurrentRelease(t *testing.T) {
	root := t.TempDir()
	builder := NewBuilder(Config{ReleaseRoot: root})
	if err := builder.WriteCurrent(context.Background(), CurrentRelease{
		ExtensionID: DefaultThemeExtensionID,
		Mode:        CurrentModeDefault,
	}); err != nil {
		t.Fatalf("write default current: %v", err)
	}
	var current CurrentRelease
	raw, err := os.ReadFile(filepath.Join(root, "current.json"))
	if err != nil {
		t.Fatalf("read current: %v", err)
	}
	if err := json.Unmarshal(raw, &current); err != nil {
		t.Fatalf("decode default current: %v", err)
	}
	if current.ExtensionID != DefaultThemeExtensionID {
		t.Fatalf("expected default theme extension id, got %q", current.ExtensionID)
	}
	if current.Mode != CurrentModeDefault {
		t.Fatalf("expected default mode, got %q", current.Mode)
	}
	if current.Server != "" || current.LayerPath != "" {
		t.Fatalf("default current must omit server/layerPath, got %#v", current)
	}
	if current.ActivatedAt == "" {
		t.Fatal("expected non-empty activatedAt for default current")
	}
}

func TestBuilderNormalizesRelativeCurrentPaths(t *testing.T) {
	root, err := os.MkdirTemp("", "sforum-theme-current-*")
	if err != nil {
		t.Fatalf("mkdir temp root: %v", err)
	}
	defer os.RemoveAll(root)
	builder := NewBuilder(Config{ReleaseRoot: root})
	// 传入相对路径，WriteCurrent 应转成绝对路径写入。
	if err := builder.WriteCurrent(context.Background(), CurrentRelease{
		ExtensionID: "starter.theme",
		Server:      filepath.Join("releases", "2", ".output", "server", "index.mjs"),
		LayerPath:   filepath.Join("storage", "extensions", "layer"),
	}); err != nil {
		t.Fatalf("write current with relative paths: %v", err)
	}
	var current CurrentRelease
	raw, err := os.ReadFile(filepath.Join(root, "current.json"))
	if err != nil {
		t.Fatalf("read current: %v", err)
	}
	if err := json.Unmarshal(raw, &current); err != nil {
		t.Fatalf("decode current: %v", err)
	}
	if !filepath.IsAbs(current.Server) {
		t.Fatalf("relative server must be absolutized, got %q", current.Server)
	}
	if !filepath.IsAbs(current.LayerPath) {
		t.Fatalf("relative layerPath must be absolutized, got %q", current.LayerPath)
	}
	if current.Mode != CurrentModeUploaded {
		t.Fatalf("server+layerPath present should infer uploaded mode, got %q", current.Mode)
	}
}

func TestBuilderRejectsMissingLayer(t *testing.T) {
	builder := NewBuilder(Config{ReleaseRoot: t.TempDir(), WebRoot: t.TempDir(), BunPath: "bun", BuildTimeout: time.Second})
	_, err := builder.Build(context.Background(), BuildInput{
		ReleaseID:   1,
		ExtensionID: "starter.theme",
		LayerPath:   filepath.Join(t.TempDir(), "missing"),
	})
	if err == nil {
		t.Fatal("expected missing layer error")
	}
}

func TestBuilderIncludesPreviewOutputInBuildLogOnHealthCheckFailure(t *testing.T) {
	root := t.TempDir()
	webRoot := t.TempDir()
	layerRoot := t.TempDir()
	fakeBun := filepath.Join(root, "fake-bun")
	script := `#!/bin/sh
if [ "$1" = "run" ] && [ "$2" = "build" ]; then
  mkdir -p "$SFORUM_NITRO_OUTPUT_DIR/server"
  printf "console.log('preview')\n" > "$SFORUM_NITRO_OUTPUT_DIR/server/index.mjs"
  echo "build ok"
  exit 0
fi
echo "preview boot failed"
exit 1
`
	if err := os.WriteFile(fakeBun, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bun: %v", err)
	}
	builder := NewBuilder(Config{
		ReleaseRoot:    root,
		WebRoot:        webRoot,
		BunPath:        fakeBun,
		BuildTimeout:   time.Second,
		PreviewTimeout: 50 * time.Millisecond,
	})

	result, err := builder.Build(context.Background(), BuildInput{
		ReleaseID:   1,
		ExtensionID: "starter.theme",
		LayerPath:   layerRoot,
	})

	if err == nil {
		t.Fatal("expected preview health check failure")
	}
	if !strings.Contains(result.BuildLog, "build ok") {
		t.Fatalf("expected build output in build log, got %q", result.BuildLog)
	}
	if !strings.Contains(result.BuildLog, "preview boot failed") {
		t.Fatalf("expected preview output in build log, got %q", result.BuildLog)
	}
}
