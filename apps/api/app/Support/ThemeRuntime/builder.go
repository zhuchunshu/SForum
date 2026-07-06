package themeruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	ReleaseRoot    string
	WebRoot        string
	BunPath        string
	BuildTimeout   time.Duration
	PreviewTimeout time.Duration
	PreviewPath    string
}

type Builder struct {
	config Config
}

type BuildInput struct {
	ReleaseID   int64
	ExtensionID string
	LayerPath   string
}

type BuildResult struct {
	ArtifactPath string
	ServerEntry  string
	BuildLog     string
}

type CurrentRelease struct {
	ReleaseID   int64  `json:"releaseId"`
	ExtensionID string `json:"extensionId"`
	Server      string `json:"server"`
	ActivatedAt string `json:"activatedAt"`
}

func NewBuilder(config Config) *Builder {
	if config.ReleaseRoot == "" {
		config.ReleaseRoot = "/var/lib/sforum/theme-releases"
	}
	if config.WebRoot == "" {
		config.WebRoot = "/app/apps/web"
	}
	if config.BunPath == "" {
		config.BunPath = "bun"
	}
	if config.BuildTimeout <= 0 {
		config.BuildTimeout = 5 * time.Minute
	}
	if config.PreviewTimeout <= 0 {
		config.PreviewTimeout = 30 * time.Second
	}
	if config.PreviewPath == "" {
		config.PreviewPath = "/"
	}
	return &Builder{config: config}
}

func (b *Builder) Build(ctx context.Context, input BuildInput) (BuildResult, error) {
	layerInfo, err := os.Stat(input.LayerPath)
	if err != nil || !layerInfo.IsDir() {
		return BuildResult{}, fmt.Errorf("theme layer is not available: %s", input.LayerPath)
	}
	releaseDir := filepath.Join(b.config.ReleaseRoot, "releases", strconv.FormatInt(input.ReleaseID, 10))
	artifactPath := filepath.Join(releaseDir, ".output")
	buildDir := filepath.Join(releaseDir, ".nuxt-build")
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		return BuildResult{}, fmt.Errorf("create release dir: %w", err)
	}
	buildCtx, cancel := context.WithTimeout(ctx, b.config.BuildTimeout)
	defer cancel()
	cmd := exec.CommandContext(buildCtx, b.config.BunPath, "run", "build")
	cmd.Dir = b.config.WebRoot
	cmd.Env = append(os.Environ(),
		"SFORUM_THEME_LAYER="+input.LayerPath,
		"SFORUM_NITRO_OUTPUT_DIR="+artifactPath,
		"NUXT_BUILD_DIR="+buildDir,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return BuildResult{BuildLog: string(output)}, fmt.Errorf("theme build failed: %w", err)
	}
	server := filepath.Join(artifactPath, "server", "index.mjs")
	if _, err := os.Stat(server); err != nil {
		return BuildResult{ArtifactPath: artifactPath, BuildLog: string(output)}, fmt.Errorf("theme server entry missing: %w", err)
	}
	if err := b.HealthCheck(ctx, server); err != nil {
		return BuildResult{ArtifactPath: artifactPath, ServerEntry: server, BuildLog: string(output)}, err
	}
	return BuildResult{ArtifactPath: artifactPath, ServerEntry: server, BuildLog: string(output)}, nil
}

func (b *Builder) HealthCheck(ctx context.Context, server string) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("reserve preview port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	previewCtx, cancel := context.WithTimeout(ctx, b.config.PreviewTimeout)
	defer cancel()
	cmd := exec.CommandContext(previewCtx, b.config.BunPath, server)
	cmd.Env = append(os.Environ(), "HOST=127.0.0.1", "PORT="+strconv.Itoa(port))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start theme preview: %w", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	url := "http://127.0.0.1:" + strconv.Itoa(port) + b.config.PreviewPath
	deadline := time.Now().Add(b.config.PreviewTimeout)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(previewCtx, http.MethodGet, url, nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode < http.StatusInternalServerError {
			_ = resp.Body.Close()
			return nil
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("theme preview health check failed")
}

func (b *Builder) WriteCurrent(_ context.Context, current CurrentRelease) error {
	current.ActivatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := os.MkdirAll(b.config.ReleaseRoot, 0o755); err != nil {
		return fmt.Errorf("create release root: %w", err)
	}
	raw, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal current release: %w", err)
	}
	tmp := filepath.Join(b.config.ReleaseRoot, "current.json.tmp")
	final := filepath.Join(b.config.ReleaseRoot, "current.json")
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("write current release temp file: %w", err)
	}
	return os.Rename(tmp, final)
}
