package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// M8 门禁：与 DevHygiene reaper 完全同一条判据，生产/测试在入口拒绝。
func TestExtensionDevRefusesNonDevelopmentEnvironments(t *testing.T) {
	for _, env := range []string{"production", "staging", "test", "", "PROD"} {
		t.Run("env="+env, func(t *testing.T) {
			t.Setenv("APP_ENV", env)
			cmd := &cobra.Command{}
			cmd.SetOut(&bytes.Buffer{})
			err := runExtensionDev(cmd, ".", extensionDevOptions{Once: true})
			if !errors.Is(err, errExtensionDevDisabled) {
				t.Fatalf("err = %v; want errExtensionDevDisabled", err)
			}
			if !strings.Contains(err.Error(), "APP_ENV must be development") {
				t.Fatalf("unhelpful refusal: %v", err)
			}
		})
	}
}

// 受保护内置源码树必须被拒绝：原地重建会绕开 build-builtin-plugins.sh + 管理员激活。
func TestExtensionDevRefusesProtectedAndForeignRoots(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]string{
		"builtin tree": filepath.Join(repoRoot, "extensions", "builtin", "plugins", "sforum-shortcodes"),
		"outside repo": t.TempDir(),
		"repo root":    repoRoot,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := resolveDevPackageRoot(target); !errors.Is(err, errExtensionDevDisabled) {
				t.Fatalf("err = %v; want errExtensionDevDisabled", err)
			}
		})
	}
	if _, err := resolveDevPackageRoot(filepath.Join(repoRoot, "extensions", "fixtures", "plugins", "sforum-commerce-workflow")); err != nil {
		t.Fatalf("fixture root must be allowed: %v", err)
	}
}

// 监视集合：只有源输入触发重建；构建产物、依赖、VCS、归档、临时文件必须被忽略，
// 否则循环会自己触发自己。
func TestExtensionDevWatchSetIgnoresBuildOutputs(t *testing.T) {
	for _, relevant := range []string{
		"sforum.extension.json", "theme.json", "manifest/settings.json",
		"backend/main.go", "backend/plugin", "templates/page-docs.html",
		"schemas/order-card-props.json", "migrations/001_init.sql",
		"frontend/admin/src/AdminDashboard.vue", "assets/theme.css",
	} {
		if !devWatchRelevant(relevant) {
			t.Fatalf("%s must trigger a rebuild", relevant)
		}
	}
	for _, ignored := range []string{
		"frontend/admin/dist/dashboard.mjs", "frontend/admin/dist/dashboard.css",
		"node_modules/left-pad/index.js", ".git/HEAD", ".nuxt/build.json",
		"acme.demo.sforum.zip", "backend/plugin.tmp", ".DS_Store",
		"frontend/admin/src/AdminDashboard.vue.swp", "logs/air.log",
		"frontend/admin/dist/dashboard.mjs.map", "tmp/scratch.go",
	} {
		if devWatchRelevant(ignored) {
			t.Fatalf("%s must not trigger a rebuild", ignored)
		}
	}
}

type devTestWatcher struct {
	events chan string
	errs   chan error
	added  []string
	closed atomic.Bool
}

func newDevTestWatcher() *devTestWatcher {
	return &devTestWatcher{events: make(chan string, 64), errs: make(chan error, 4)}
}

func (w *devTestWatcher) Add(path string) error {
	w.added = append(w.added, path)
	return nil
}
func (w *devTestWatcher) Events() <-chan string { return w.events }
func (w *devTestWatcher) Errors() <-chan error  { return w.errs }
func (w *devTestWatcher) Close() error {
	w.closed.Store(true)
	return nil
}

type devTestRunner struct {
	mu        sync.Mutex
	calls     int
	digests   []string
	failures  map[int]error
	active    int32
	maxActive int32
	started   chan struct{}
	release   chan struct{}
}

func (r *devTestRunner) Run(_ context.Context, _ string) (devBuildResult, error) {
	active := atomic.AddInt32(&r.active, 1)
	for {
		previous := atomic.LoadInt32(&r.maxActive)
		if active <= previous || atomic.CompareAndSwapInt32(&r.maxActive, previous, active) {
			break
		}
	}
	if r.started != nil {
		select {
		case r.started <- struct{}{}:
		default:
		}
	}
	if r.release != nil {
		<-r.release
	}
	atomic.AddInt32(&r.active, -1)

	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if err := r.failures[r.calls]; err != nil {
		return devBuildResult{}, err
	}
	digest := r.digests[(r.calls-1)%len(r.digests)]
	return devBuildResult{PackageDigest: digest, Phases: []string{"build"}}, nil
}

type devRecordingSink struct {
	mu     sync.Mutex
	phases []string
	fails  []string
}

func (s *devRecordingSink) Phase(name string, _ map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.phases = append(s.phases, name)
}

func (s *devRecordingSink) Failed(stage string, err error, kept string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fails = append(s.fails, stage+":"+err.Error()+"|kept="+kept)
}

func (s *devRecordingSink) count(name string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	total := 0
	for _, phase := range s.phases {
		if phase == name {
			total++
		}
	}
	return total
}

// 去抖合并 + 单事务：事件风暴只产生一次构建，构建期间到达的事件合并成下一轮。
func TestExtensionDevLoopCoalescesAndRunsOneTransactionAtATime(t *testing.T) {
	root := t.TempDir()
	watcher := newDevTestWatcher()
	sink := &devRecordingSink{}
	runner := &devTestRunner{digests: []string{"digest-a", "digest-b", "digest-c"}}
	loop := &devWatchLoop{
		root: root, runner: runner, watcher: watcher, status: sink, debounce: 40 * time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var lastDigest string
	var runErr error
	go func() {
		lastDigest, runErr = loop.run(ctx)
		close(done)
	}()

	// 一次风暴：5 个事件必须只换来一次构建。
	for index := 0; index < 5; index++ {
		watcher.events <- filepath.Join(root, "backend", "main.go")
	}
	// 非源输入不得触发构建。
	watcher.events <- filepath.Join(root, "frontend", "admin", "dist", "dashboard.mjs")
	waitFor(t, func() bool { return sink.count("built") == 1 }, 3*time.Second)

	cancel()
	<-done
	if runErr != nil {
		t.Fatalf("loop error: %v", runErr)
	}
	if runner.calls != 1 {
		t.Fatalf("build transactions = %d; want 1", runner.calls)
	}
	if lastDigest != "digest-a" {
		t.Fatalf("last digest = %q", lastDigest)
	}
	if atomic.LoadInt32(&runner.maxActive) != 1 {
		t.Fatalf("concurrent build transactions = %d; want 1", runner.maxActive)
	}
	if sink.count("watching") != 1 || sink.count("reload_required") != 1 {
		t.Fatalf("phases = %#v", sink.phases)
	}
}

// 失败必须保留最后一个健康制品，并在状态流里报告原因与保留的摘要。
func TestExtensionDevLoopKeepsLastHealthyDigestOnFailure(t *testing.T) {
	root := t.TempDir()
	watcher := newDevTestWatcher()
	sink := &devRecordingSink{}
	runner := &devTestRunner{
		digests:  []string{"digest-healthy"},
		failures: map[int]error{2: errors.New("preflight rejected the package")},
	}
	loop := &devWatchLoop{
		root: root, runner: runner, watcher: watcher, status: sink, debounce: 30 * time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var lastDigest string
	go func() {
		lastDigest, _ = loop.run(ctx)
		close(done)
	}()

	watcher.events <- filepath.Join(root, "backend", "main.go")
	waitFor(t, func() bool { return sink.count("built") == 1 }, 3*time.Second)
	watcher.events <- filepath.Join(root, "backend", "main.go")
	waitFor(t, func() bool { return len(sink.fails) == 1 }, 3*time.Second)

	cancel()
	<-done
	if lastDigest != "digest-healthy" {
		t.Fatalf("last healthy digest = %q; want digest-healthy", lastDigest)
	}
	if !strings.Contains(sink.fails[0], "kept=digest-healthy") {
		t.Fatalf("failure report = %q", sink.fails[0])
	}
}

func waitFor(t *testing.T, condition func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}

// --once 走真实实现：在 extensions/dev 下生成包，一个命令完成 build+digest+validate+test
// 并打印精确制品身份（不绕过信任：仍需管理端确认后才会生效）。
func TestExtensionDevOnceRunsOneRealBuildTransaction(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(repoRoot, "extensions", "dev", "acme.devloop-probe")
	if _, statErr := os.Stat(target); statErr == nil {
		t.Skipf("dev probe already exists at %s", target)
	}
	if _, err := GenerateExtensionScaffold(makeOptions{
		Kind: "plugin", ID: "acme.devloop-probe", Name: "Dev Loop Probe",
		Description: "M8 dev loop probe.", URL: "https://example.com/devloop",
		AuthorName: "Acme", NoInteraction: true, Backend: true, Out: target,
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(target) })

	t.Setenv("APP_ENV", "development")
	cmd := &cobra.Command{}
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	if err := runExtensionDev(cmd, target, extensionDevOptions{Once: true, AsJSON: true}); err != nil {
		t.Fatalf("extension dev --once: %v\n%s", err, output.String())
	}
	events := map[string]map[string]any{}
	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			t.Fatalf("machine-readable line %q: %v", line, err)
		}
		if event, _ := payload["event"].(string); event != "" {
			phase, _ := payload["phase"].(string)
			events[event+":"+phase] = payload
		}
	}
	built, ok := events["phase:built"]
	if !ok {
		t.Fatalf("no built phase in %s", output.String())
	}
	digest, _ := built["packageDigest"].(string)
	if len(digest) != 64 {
		t.Fatalf("built phase digest = %q", digest)
	}
	reload, ok := events["phase:reload_required"]
	if !ok || !strings.Contains(reload["next"].(string), "admin") {
		t.Fatalf("reload guidance missing: %#v", events)
	}
	if !strings.Contains(events["phase:building"]["trigger"].(string), "once") {
		t.Fatalf("trigger = %#v", events["phase:building"])
	}
}

// 回归：构建事务自身会重写清单（摘要刷新）。那是我们自己的写入，绝不能触发下一轮，
// 否则循环会无限自触发（真机 e2e 里出现过 13k 行输出）。
func TestExtensionDevLoopIgnoresItsOwnBuildWrites(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "backend", "main.go")
	manifest := filepath.Join(root, "sforum.extension.json")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	watcher := newDevTestWatcher()
	sink := &devRecordingSink{}
	// runner 模拟真实构建：重写清单（摘要刷新），并让 watcher 报告该写入。
	runner := &devRunnerWritingManifest{root: root, manifest: manifest, watcher: watcher, digests: []string{"digest-1", "digest-2"}}
	loop := &devWatchLoop{
		root: root, runner: runner, watcher: watcher, status: sink, debounce: 30 * time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_, _ = loop.run(ctx)
		close(done)
	}()

	// 启动对账一次（作者可能在启动前刚编辑过）。
	waitFor(t, func() bool { return sink.count("built") == 1 }, 3*time.Second)

	// 作者编辑一次源文件。
	if err := os.WriteFile(source, []byte("package main // edited\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	watcher.events <- source
	waitFor(t, func() bool { return sink.count("built") == 2 }, 3*time.Second)
	// 给自触发足够的时间窗口：若清单重写被当成作者编辑，这里会出现额外构建。
	time.Sleep(300 * time.Millisecond)
	if got := sink.count("built"); got != 2 {
		t.Fatalf("build transactions = %d after one author edit; the loop is self-triggering", got)
	}

	// 作者再编辑一次：必须仍然能被识别为真实变化。
	if err := os.WriteFile(source, []byte("package main // edited twice\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	watcher.events <- source
	waitFor(t, func() bool { return sink.count("built") == 3 }, 3*time.Second)
	time.Sleep(200 * time.Millisecond)
	if got := sink.count("built"); got != 3 {
		t.Fatalf("build transactions = %d after the second author edit", got)
	}

	cancel()
	<-done
	if runner.calls != 3 {
		t.Fatalf("runner calls = %d; want 3 (startup + two author edits)", runner.calls)
	}
}

// 删除文件同样必须触发重建（内容状态消失是作者动作）。
func TestExtensionDevLoopRebuildsOnDeletedSourceInput(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "backend", "main.go")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	watcher := newDevTestWatcher()
	sink := &devRecordingSink{}
	runner := &devTestRunner{digests: []string{"digest-only"}}
	loop := &devWatchLoop{
		root: root, runner: runner, watcher: watcher, status: sink, debounce: 30 * time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_, _ = loop.run(ctx)
		close(done)
	}()
	waitFor(t, func() bool { return sink.count("watching") == 1 }, 3*time.Second)
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}
	watcher.events <- source
	waitFor(t, func() bool { return sink.count("built") == 1 }, 3*time.Second)
	cancel()
	<-done
}

type devRunnerWritingManifest struct {
	root     string
	manifest string
	watcher  *devTestWatcher
	digests  []string
	calls    int
}

func (r *devRunnerWritingManifest) Run(_ context.Context, _ string) (devBuildResult, error) {
	r.calls++
	body := []byte("{\"manifestVersion\":3,\"call\":" + string(rune('0'+r.calls)) + "}\n")
	if err := os.WriteFile(r.manifest, body, 0o600); err != nil {
		return devBuildResult{}, err
	}
	// 真实 watcher 会把这次写入作为事件投递；测试替身显式复现这一点。
	r.watcher.events <- r.manifest
	return devBuildResult{PackageDigest: r.digests[(r.calls-1)%len(r.digests)]}, nil
}

// CLI 与服务端共用同一外部开发根集合：EXTERNAL_EXTENSION_ROOTS 下的插件仓库
// （含子目录）必须可被 dev 循环监视与 dev-trust create 登记，
// 不得比服务端 DevTrustService.devSourceRootAllowed 更严。
func TestExtensionDevAcceptsConfiguredExternalRoots(t *testing.T) {
	external := t.TempDir()
	packageRoot := filepath.Join(external, "plugins", "sforum.oauth2-provider")
	if err := os.MkdirAll(packageRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []string{
		external,
		filepath.Join(external, "plugins"),
		external + ", ,relative/path," + external,
	}
	for _, raw := range cases {
		t.Run("roots="+raw, func(t *testing.T) {
			t.Setenv("EXTERNAL_EXTENSION_ROOTS", raw)
			if _, err := resolveDevPackageRoot(packageRoot); err != nil {
				t.Fatalf("external root package must be accepted: %v", err)
			}
		})
	}
	t.Run("without configured roots refusal names the env var", func(t *testing.T) {
		t.Setenv("EXTERNAL_EXTENSION_ROOTS", "")
		_, err := resolveDevPackageRoot(packageRoot)
		if !errors.Is(err, errExtensionDevDisabled) {
			t.Fatalf("err = %v; want errExtensionDevDisabled", err)
		}
		if !strings.Contains(err.Error(), "EXTERNAL_EXTENSION_ROOTS") {
			t.Fatalf("refusal must point at the env var: %v", err)
		}
	})
}

type devTestActivator struct {
	mu           sync.Mutex
	calls        []string
	refreshError error
	upgradeError error
}

func (a *devTestActivator) Refresh(_ context.Context, digest string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls = append(a.calls, "refresh:"+digest)
	return a.refreshError
}

func (a *devTestActivator) Upgrade(_ context.Context, digest string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls = append(a.calls, "upgrade:"+digest)
	return a.upgradeError
}

func (a *devTestActivator) recorded() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.calls...)
}

// --auto-upgrade：信任续期成功后必须继续走既有 upgrade 路由，顺序 refresh→upgrade，
// 成功后输出 activated 阶段——这就是「保存即生效」链的最后一环。
func TestExtensionDevLoopAutoUpgradesAfterTrust(t *testing.T) {
	root := t.TempDir()
	watcher := newDevTestWatcher()
	sink := &devRecordingSink{}
	runner := &devTestRunner{digests: []string{"digest-live"}}
	activator := &devTestActivator{}
	loop := &devWatchLoop{
		root: root, runner: runner, watcher: watcher, status: sink, debounce: 30 * time.Millisecond,
		activator: activator, autoUpgrade: true,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_, _ = loop.run(ctx)
		close(done)
	}()
	watcher.events <- filepath.Join(root, "backend", "main.go")
	waitFor(t, func() bool { return sink.count("activated") == 1 }, 3*time.Second)
	cancel()
	<-done

	assertDeepEqual(t, activator.recorded(), []string{"refresh:digest-live", "upgrade:digest-live"})
	if len(sink.fails) != 0 {
		t.Fatalf("unexpected failures: %#v", sink.fails)
	}
}

// 自动激活失败不得中断循环：报告 auto_upgrade_failed 并保留健康制品。
func TestExtensionDevLoopReportsAutoUpgradeFailure(t *testing.T) {
	root := t.TempDir()
	watcher := newDevTestWatcher()
	sink := &devRecordingSink{}
	runner := &devTestRunner{digests: []string{"digest-live"}}
	activator := &devTestActivator{upgradeError: errors.New("lifecycle coordinator invalid: extension is disabled")}
	loop := &devWatchLoop{
		root: root, runner: runner, watcher: watcher, status: sink, debounce: 30 * time.Millisecond,
		activator: activator, autoUpgrade: true,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_, _ = loop.run(ctx)
		close(done)
	}()
	watcher.events <- filepath.Join(root, "backend", "main.go")
	waitFor(t, func() bool { return sink.count("auto_upgrade_failed") == 1 }, 3*time.Second)
	assertDeepEqual(t, activator.recorded(), []string{"refresh:digest-live", "upgrade:digest-live"})
	// 下一次编辑仍会重新走完整链：循环保持存活。
	watcher.events <- filepath.Join(root, "backend", "main.go")
	waitFor(t, func() bool { return sink.count("auto_upgrade_failed") == 2 }, 3*time.Second)
	cancel()
	<-done
}

// 仅 --auto-trust（不带 --auto-upgrade）保持既有行为：止步于 dev_trusted。
func TestExtensionDevLoopWithoutAutoUpgradeStopsAtTrust(t *testing.T) {
	root := t.TempDir()
	watcher := newDevTestWatcher()
	sink := &devRecordingSink{}
	runner := &devTestRunner{digests: []string{"digest-live"}}
	activator := &devTestActivator{}
	loop := &devWatchLoop{
		root: root, runner: runner, watcher: watcher, status: sink, debounce: 30 * time.Millisecond,
		activator: activator, autoUpgrade: false,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_, _ = loop.run(ctx)
		close(done)
	}()
	watcher.events <- filepath.Join(root, "backend", "main.go")
	waitFor(t, func() bool { return sink.count("dev_trusted") == 1 }, 3*time.Second)
	cancel()
	<-done
	if calls := activator.recorded(); len(calls) != 1 || calls[0] != "refresh:digest-live" {
		t.Fatalf("calls = %#v; want only refresh", calls)
	}
}

func assertDeepEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %#v; want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("got %#v; want %#v", got, want)
		}
	}
}
