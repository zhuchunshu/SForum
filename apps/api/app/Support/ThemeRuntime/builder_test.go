package themeruntime

import (
	"context"
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
	builder := NewBuilder(Config{ReleaseRoot: root})
	if err := builder.WriteCurrent(context.Background(), CurrentRelease{
		ReleaseID:   1,
		ExtensionID: "starter.theme",
		Server:      server,
	}); err != nil {
		t.Fatalf("write current: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "current.json"))
	if err != nil {
		t.Fatalf("read current: %v", err)
	}
	if !strings.Contains(string(raw), "starter.theme") || !strings.Contains(string(raw), server) {
		t.Fatalf("current.json missing release data: %s", raw)
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
