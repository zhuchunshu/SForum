package webreleaseruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

type Builder struct {
	config Config
	runner CommandRunner
}

type execCommandRunner struct{}

func (execCommandRunner) Run(ctx context.Context, command Command) (string, error) {
	cmd := exec.CommandContext(ctx, command.Path, command.Args...)
	cmd.Dir = command.Dir
	cmd.Env = command.Env
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func NewBuilder(config Config) *Builder {
	if config.ReleaseRoot == "" {
		config.ReleaseRoot = "../../storage/theme-releases"
	}
	if config.WebRoot == "" {
		config.WebRoot = "../web"
	}
	if config.ExtensionRoot == "" {
		config.ExtensionRoot = "../../storage/extensions"
	}
	if config.BunPath == "" {
		config.BunPath = "bun"
	}
	if config.NodePath == "" {
		config.NodePath = "node"
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
	if config.SourceEnvironment == nil {
		config.SourceEnvironment = os.Environ()
	}
	config.ReleaseRoot = absolutePath(config.ReleaseRoot)
	config.WebRoot = resolveWebRoot(config.WebRoot)
	config.ExtensionRoot = absolutePath(config.ExtensionRoot)
	if strings.TrimSpace(config.DefaultThemeLayer) != "" {
		config.DefaultThemeLayer = absolutePath(config.DefaultThemeLayer)
	}
	runner := config.Runner
	if runner == nil {
		runner = execCommandRunner{}
	}
	return &Builder{config: config, runner: runner}
}

func (b *Builder) Prepare(_ context.Context, detail extensions.WebReleaseDetail) (PreparedRelease, error) {
	if detail.ID <= 0 {
		return PreparedRelease{}, fmt.Errorf("web release id must be positive")
	}
	var composition extensions.WebComposition
	if err := json.Unmarshal(detail.CompositionSnapshot, &composition); err != nil {
		return PreparedRelease{}, fmt.Errorf("decode web release composition: %w", err)
	}
	if composition.SDKVersion != AdminSDKVersion || composition.Contract != BuildContractVersion || composition.BunVersion != BunVersion {
		return PreparedRelease{}, fmt.Errorf("web release host build identity is stale")
	}
	compositionBody, err := canonicalJSON(detail.CompositionSnapshot)
	if err != nil {
		return PreparedRelease{}, err
	}
	compositionDigest := sha256.Sum256(compositionBody)
	if hex.EncodeToString(compositionDigest[:]) != detail.CompositionHash {
		return PreparedRelease{}, fmt.Errorf("web release composition hash changed")
	}
	if composition.Theme.ExtensionID != detail.ActiveThemeID || composition.Theme.Version != detail.ThemeVersion || composition.Theme.PackageDigest != detail.ThemePackageDigest {
		return PreparedRelease{}, fmt.Errorf("web release theme snapshot does not match composition")
	}
	if err := validateExtensionSnapshots(composition.Extensions, detail.Extensions); err != nil {
		return PreparedRelease{}, err
	}
	releaseDir := filepath.Join(b.config.ReleaseRoot, "releases", strconv.FormatInt(detail.ID, 10))
	workspace := filepath.Join(releaseDir, "workspace")
	devInput := filepath.Join(releaseDir, "dev-input")
	if err := os.RemoveAll(workspace); err != nil {
		return PreparedRelease{}, err
	}
	if err := os.RemoveAll(devInput); err != nil {
		return PreparedRelease{}, err
	}
	if err := copyTree(b.config.WebRoot, workspace, hostSourceExcluded); err != nil {
		return PreparedRelease{}, fmt.Errorf("copy web source: %w", err)
	}
	if err := os.MkdirAll(devInput, 0o755); err != nil {
		return PreparedRelease{}, err
	}
	if err := linkHostDependencies(b.config.WebRoot, workspace); err != nil {
		return PreparedRelease{}, err
	}

	// 公开主题 Layer 已退役：仅当 composition 仍携带 ThemeLayerPath 时复制（兼容旧 release）。
	// 默认不再注入 SFORUM_THEME_LAYER；管理端插件前端 registry 仍照常生成。
	themeLayer := ""
	defaultThemeLayer := ""
	if strings.TrimSpace(detail.ThemeLayerPath) != "" && strings.TrimSpace(detail.ThemePackageDigest) != "" {
		themeSource := b.snapshotPath(detail.ActiveThemeID, detail.ThemeVersion, detail.ThemePackageDigest)
		if err := verifySnapshot(themeSource, detail.ThemePackageDigest); err != nil {
			return PreparedRelease{}, fmt.Errorf("verify theme snapshot: %w", err)
		}
		themeTarget := filepath.Join(devInput, "theme")
		if err := copyTree(themeSource, themeTarget, nil); err != nil {
			return PreparedRelease{}, fmt.Errorf("copy theme snapshot: %w", err)
		}
		if err := verifySnapshot(themeTarget, detail.ThemePackageDigest); err != nil {
			return PreparedRelease{}, fmt.Errorf("verify copied theme snapshot: %w", err)
		}
		layerRelative, err := filepath.Rel(themeSource, detail.ThemeLayerPath)
		if err != nil || layerRelative == ".." || strings.HasPrefix(layerRelative, ".."+string(filepath.Separator)) {
			return PreparedRelease{}, fmt.Errorf("theme layer escapes immutable snapshot")
		}
		resolved, err := secureChildDirectory(themeTarget, filepath.ToSlash(layerRelative))
		if err != nil {
			return PreparedRelease{}, fmt.Errorf("resolve copied theme layer: %w", err)
		}
		themeLayer = resolved
		defaultThemeLayer = themeLayer
	}
	if b.config.DefaultThemeLayer != "" {
		// 可选：仅兼容旧构建脚本仍读取 SFORUM_DEFAULT_THEME_LAYER 时复制；host 已不 extends layer。
		defaultThemeLayer = filepath.Join(devInput, "default-theme")
		if err := copyTree(b.config.DefaultThemeLayer, defaultThemeLayer, nil); err != nil {
			return PreparedRelease{}, fmt.Errorf("copy default theme fallback: %w", err)
		}
	}

	registryExtensions := make([]RegistryExtension, 0, len(detail.Extensions))
	pluginRoots := make(map[string]string, len(detail.Extensions))
	pluginFrontends := make(map[string]string, len(detail.Extensions))
	for _, snapshot := range detail.Extensions {
		source := b.snapshotPath(snapshot.ExtensionID, snapshot.ExtensionVersion, snapshot.PackageDigest)
		if err := verifySnapshot(source, snapshot.PackageDigest); err != nil {
			return PreparedRelease{}, fmt.Errorf("verify extension %s snapshot: %w", snapshot.ExtensionID, err)
		}
		target := filepath.Join(devInput, "extensions", snapshot.ExtensionID)
		if !pathWithin(filepath.Join(devInput, "extensions"), target) {
			return PreparedRelease{}, fmt.Errorf("unsafe extension id %q", snapshot.ExtensionID)
		}
		if err := copyTree(source, target, nil); err != nil {
			return PreparedRelease{}, fmt.Errorf("copy extension %s: %w", snapshot.ExtensionID, err)
		}
		if err := verifySnapshot(target, snapshot.PackageDigest); err != nil {
			return PreparedRelease{}, fmt.Errorf("verify copied extension %s: %w", snapshot.ExtensionID, err)
		}
		frontend, err := secureChildDirectory(target, snapshot.FrontendRoot)
		if err != nil {
			return PreparedRelease{}, fmt.Errorf("resolve extension %s frontend: %w", snapshot.ExtensionID, err)
		}
		pluginFrontends[snapshot.ExtensionID] = frontend
		pluginRoots[snapshot.ExtensionID] = target
		registryExtensions = append(registryExtensions, RegistryExtension{SourceRoot: target, Snapshot: snapshot})
	}
	registryRoot := filepath.Join(devInput, "registry")
	if _, err := GenerateRegistry(RegistryInput{Root: registryRoot, ReleaseID: detail.ID, ReloadMode: detail.ReloadMode, Extensions: registryExtensions}); err != nil {
		return PreparedRelease{}, err
	}
	if err := b.writeGuardPolicy(filepath.Join(devInput, "guard-policy.json"), composition, pluginFrontends); err != nil {
		return PreparedRelease{}, err
	}
	return PreparedRelease{
		Detail: detail, Composition: composition, ReleaseDir: releaseDir, Workspace: workspace,
		DevInput: devInput, RegistryRoot: registryRoot, ThemeLayer: themeLayer, DefaultThemeLayer: defaultThemeLayer, PluginFrontends: pluginFrontends,
		PluginRoots: pluginRoots,
	}, nil
}

func (b *Builder) Install(ctx context.Context, prepared PreparedRelease) ([]DependencySnapshot, string, error) {
	ids := make([]string, 0, len(prepared.PluginFrontends))
	for id := range prepared.PluginFrontends {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	logs := make([]string, 0, len(ids))
	snapshots := make([]DependencySnapshot, 0, len(ids))
	for _, id := range ids {
		frontend := prepared.PluginFrontends[id]
		installCtx, cancel := context.WithTimeout(ctx, b.config.BuildTimeout)
		log, err := b.runner.Run(installCtx, Command{
			Path: b.config.BunPath, Args: []string{"install", "--frozen-lockfile", "--ignore-scripts"},
			Dir: frontend, Env: InstallEnvironment(b.config.SourceEnvironment),
		})
		cancel()
		logs = append(logs, log)
		if err != nil {
			return nil, boundedLog(logs...), fmt.Errorf("install admin frontend %s: %w", id, err)
		}
		snapshot, ok := findReleaseExtension(prepared.Detail.Extensions, id)
		if !ok {
			return nil, boundedLog(logs...), fmt.Errorf("missing release snapshot for %s", id)
		}
		summary, err := extensionpackage.InspectAdminFrontend(extensionpackage.FrontendInspectInput{
			PackageRoot: prepared.PluginRoots[id],
			Root:        snapshot.FrontendRoot, Components: snapshot.ComponentMap, Locales: snapshot.LocaleMap, HostPeers: b.config.HostPeers,
		})
		if err != nil {
			return nil, boundedLog(logs...), fmt.Errorf("inspect installed admin frontend %s: %w", id, err)
		}
		if summary.LockDigest != snapshot.LockfileDigest {
			return nil, boundedLog(logs...), fmt.Errorf("extension %s lockfile changed", id)
		}
		if err := linkPluginHostPeers(frontend, prepared.Workspace); err != nil {
			return nil, boundedLog(logs...), fmt.Errorf("provide admin frontend host peers for %s: %w", id, err)
		}
		digest := dependencySnapshotDigest(id, summary, prepared.Composition.BunVersion, b.config.HostPeers)
		dependencies := make([]extensions.Dependency, len(summary.Resolved))
		copy(dependencies, summary.Resolved)
		snapshots = append(snapshots, DependencySnapshot{ExtensionID: id, Dependencies: dependencies, Digest: digest})
	}
	return snapshots, boundedLog(logs...), nil
}

func (b *Builder) Build(ctx context.Context, prepared PreparedRelease, previousLog string) (BuildResult, error) {
	artifact := filepath.Join(prepared.ReleaseDir, "artifact")
	// 隔离 workspace 没有 dev watcher；使用标准 .nuxt 同时满足根 tsconfig、Vite 和 vue-tsc。
	buildDir := filepath.Join(prepared.Workspace, ".nuxt")
	_ = os.RemoveAll(artifact)
	_ = os.RemoveAll(buildDir)
	environment := BuildEnvironment(b.config.SourceEnvironment, map[string]string{
		"SFORUM_THEME_LAYER":         prepared.ThemeLayer,
		"SFORUM_DEFAULT_THEME_LAYER": prepared.DefaultThemeLayer,
		"SFORUM_ADMIN_REGISTRY_ROOT": prepared.RegistryRoot,
		"SFORUM_WEB_RELEASE_ID":      strconv.FormatInt(prepared.Detail.ID, 10),
		"SFORUM_NITRO_OUTPUT_DIR":    artifact,
		"NUXT_BUILD_DIR":             buildDir,
	})
	buildCtx, cancel := context.WithTimeout(ctx, b.config.BuildTimeout)
	defer cancel()
	// typecheck：始终执行并写入 build log；是否阻断由后台选项 / 回退配置决定。
	typecheckHardFail := b.resolveTypecheckFail(ctx)
	typecheckLog, typecheckErr := b.runner.Run(buildCtx, Command{Path: b.config.BunPath, Args: []string{"run", "typecheck"}, Dir: prepared.Workspace, Env: environment})
	if typecheckErr != nil {
		if typecheckHardFail {
			return BuildResult{BuildLog: boundedLog(previousLog, typecheckLog)}, fmt.Errorf("web release typecheck failed: %w", typecheckErr)
		}
		// 非阻断：标注后继续 build；运维可在「扩展 → Web 发布」打开硬失败开关。
		typecheckLog = boundedLog(
			"=== web release typecheck FAILED (non-blocking; enable web_release.typecheck_fail in admin to hard-fail) ===",
			typecheckLog,
			fmt.Sprintf("=== typecheck error: %v ===", typecheckErr),
		)
	} else {
		typecheckLog = boundedLog("=== web release typecheck OK ===", typecheckLog)
	}
	buildLog, err := b.runner.Run(buildCtx, Command{Path: b.config.BunPath, Args: []string{"run", "build"}, Dir: prepared.Workspace, Env: environment})
	result := BuildResult{ArtifactPath: artifact, BuildLog: boundedLog(previousLog, typecheckLog, buildLog)}
	if err != nil {
		return result, fmt.Errorf("web release build failed: %w", err)
	}
	result.ServerEntry = filepath.Join(artifact, "server", "index.mjs")
	if info, err := os.Stat(result.ServerEntry); err != nil || !info.Mode().IsRegular() {
		return result, fmt.Errorf("web release server entry is missing")
	}
	return result, nil
}

func (b *Builder) resolveTypecheckFail(ctx context.Context) bool {
	if b != nil && b.config.TypecheckPolicy != nil {
		return b.config.TypecheckPolicy.TypecheckFail(ctx)
	}
	if b == nil {
		return false
	}
	return b.config.TypecheckFail
}

func (b *Builder) Verify(ctx context.Context, prepared PreparedRelease, result BuildResult) (BuildResult, error) {
	previewLog, err := b.healthCheck(ctx, result.ServerEntry)
	result.BuildLog = boundedLog(result.BuildLog, previewLog)
	if err != nil {
		return result, err
	}
	digest, err := ArtifactDigestTree(result.ArtifactPath)
	if err != nil {
		return result, fmt.Errorf("digest web release artifact: %w", err)
	}
	result.ArtifactDigest = digest
	manifest := ReleaseManifest{
		SchemaVersion: ReleaseManifestSchemaVersion, ReleaseID: prepared.Detail.ID,
		CompositionHash: prepared.Detail.CompositionHash, ArtifactPath: result.ArtifactPath,
		ArtifactDigest: digest, ServerEntry: result.ServerEntry, ThemeID: prepared.Detail.ActiveThemeID,
		ThemeVersion: prepared.Detail.ThemeVersion, ThemeLayer: prepared.ThemeLayer,
		DevInput: prepared.DevInput, RegistryRoot: prepared.RegistryRoot,
		ReloadMode: prepared.Detail.ReloadMode, BuiltAt: time.Now().UTC(),
	}
	result.ManifestPath = filepath.Join(prepared.ReleaseDir, "release.json")
	if err := writeJSONAtomic(result.ManifestPath, manifest); err != nil {
		return result, err
	}
	return result, nil
}

func (b *Builder) snapshotPath(id string, version string, digest string) string {
	return filepath.Join(b.config.ExtensionRoot, id, version, digest)
}

func (b *Builder) healthCheck(ctx context.Context, server string) (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	previewCtx, cancel := context.WithTimeout(ctx, b.config.PreviewTimeout)
	defer cancel()
	cmd := exec.CommandContext(previewCtx, b.config.NodePath, server)
	cmd.Env = BuildEnvironment(b.config.SourceEnvironment, map[string]string{"HOST": "127.0.0.1", "PORT": strconv.Itoa(port)})
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Start(); err != nil {
		return output.String(), err
	}
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	url := "http://127.0.0.1:" + strconv.Itoa(port) + b.config.PreviewPath
	for {
		select {
		case err := <-wait:
			return output.String(), fmt.Errorf("web release preview exited: %w", err)
		case <-previewCtx.Done():
			_ = cmd.Process.Kill()
			<-wait
			return output.String(), fmt.Errorf("web release preview timed out: %w", previewCtx.Err())
		case <-ticker.C:
			request, _ := http.NewRequestWithContext(previewCtx, http.MethodGet, url, nil)
			response, requestErr := http.DefaultClient.Do(request)
			if requestErr == nil && response.StatusCode < http.StatusInternalServerError {
				_ = response.Body.Close()
				_ = cmd.Process.Kill()
				<-wait
				return output.String(), nil
			}
			if response != nil {
				_ = response.Body.Close()
			}
		}
	}
}

func (b *Builder) writeGuardPolicy(target string, composition extensions.WebComposition, roots map[string]string) error {
	type rootPolicy struct {
		Root         string   `json:"root"`
		Dependencies []string `json:"dependencies"`
	}
	policy := struct {
		Roots     []rootPolicy `json:"roots"`
		HostPeers []string     `json:"hostPeers"`
	}{Roots: make([]rootPolicy, 0, len(composition.Extensions)), HostPeers: sortedMapKeys(b.config.HostPeers)}
	for _, item := range composition.Extensions {
		root := roots[item.ExtensionID]
		if root == "" {
			return fmt.Errorf("missing copied frontend root for %s", item.ExtensionID)
		}
		dependencies := make([]string, 0, len(item.Dependencies.Direct))
		for _, dependency := range item.Dependencies.Direct {
			if _, host := b.config.HostPeers[dependency.Name]; !host {
				dependencies = append(dependencies, dependency.Name)
			}
		}
		sort.Strings(dependencies)
		policy.Roots = append(policy.Roots, rootPolicy{Root: root, Dependencies: dependencies})
	}
	return writeJSONAtomic(target, policy)
}

func validateExtensionSnapshots(composition []extensions.WebExtensionSnapshot, stored []extensions.WebReleaseExtension) error {
	if len(composition) != len(stored) {
		return fmt.Errorf("web release extension snapshots do not match composition")
	}
	storedByID := make(map[string]extensions.WebReleaseExtension, len(stored))
	for _, item := range stored {
		storedByID[item.ExtensionID] = item
	}
	for _, item := range composition {
		persisted, ok := storedByID[item.ExtensionID]
		if !ok || persisted.ExtensionVersion != item.Version || persisted.PackageDigest != item.PackageDigest || persisted.FrontendRoot != item.FrontendRoot || persisted.LockfileDigest != item.Dependencies.LockDigest {
			return fmt.Errorf("web release extension %s snapshot does not match composition", item.ExtensionID)
		}
	}
	return nil
}

func findReleaseExtension(items []extensions.WebReleaseExtension, id string) (extensions.WebReleaseExtension, bool) {
	for _, item := range items {
		if item.ExtensionID == id {
			return item, true
		}
	}
	return extensions.WebReleaseExtension{}, false
}

func sortedMapKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
