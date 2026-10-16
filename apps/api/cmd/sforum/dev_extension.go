package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	devhygiene "github.com/zhuchunshu/sforum/apps/api/app/Support/DevHygiene"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

// M8：`sforum extension dev <package-root>` —— 把「编辑 → 浏览器」压成一条命令。
//
// 依据 M0 ADR（knowledge/decisions/2026-10-16-extension-authoring-contracts-freeze.md 6.2）：
//  1. 门禁与 DevHygiene reaper 完全一致（APP_ENV=development），生产/测试在入口拒绝；
//  2. **不做信任绕过**：摘要仍然变化，操作员仍要确认；本命令只自动化既有 CLI 步骤；
//  3. fsnotify 是显式模块依赖；
//  4. 不引入第二条构建管线：直接调用 runExtensionBuild（它内部已含 digest/validate/test）；
//  5. 只监听包内源输入，忽略构建产物/归档/VCS/临时文件，避免自触发循环；
//  6. 通过既有暂存路径发布，绝不原地覆盖活动摘要。

const (
	devDefaultDebounce = 400 * time.Millisecond
	// 事件风暴上限：一次事务最多合并这么多次触发，超出只记一次丢弃，避免无限重建。
	devMaximumCoalescedTriggers = 64
)

var errExtensionDevDisabled = errors.New("extensions: extension dev is development-only")

// devPackageRootAllowedRoots 返回允许在开发循环中监视的根目录。
// builtin 源码树被显式排除：受保护内置必须走 build-builtin-plugins.sh + 管理员激活，
// 在那里原地重建会绕开受保护路径。
func devPackageRootAllowedRoots(repoRoot string) []string {
	return []string{
		filepath.Join(repoRoot, "extensions", "dev"),
		filepath.Join(repoRoot, "extensions", "optional"),
		filepath.Join(repoRoot, "extensions", "fixtures"),
	}
}

// devForbiddenRoots 是明确拒绝的根：受保护内置源码树。
func devForbiddenRoots(repoRoot string) []string {
	return []string{filepath.Join(repoRoot, "extensions", "builtin")}
}

// resolveDevPackageRoot 校验包根：必须是 Manifest V3 包，且位于允许的开发根内。
func resolveDevPackageRoot(root string) (string, error) {
	abs, err := resolveExtensionPackageRoot(root)
	if err != nil {
		return "", err
	}
	repoRoot, err := findRepoRoot()
	if err != nil {
		return "", err
	}
	for _, forbidden := range devForbiddenRoots(repoRoot) {
		if devPathWithin(abs, forbidden) {
			return "", fmt.Errorf(
				"%w: %s is a protected built-in tree; rebuild with scripts/build-builtin-plugins.sh and activate through the admin flow",
				errExtensionDevDisabled, abs,
			)
		}
	}
	for _, allowed := range devPackageRootAllowedRoots(repoRoot) {
		if devPathWithin(abs, allowed) {
			return abs, nil
		}
	}
	return "", fmt.Errorf(
		"%w: %s is outside the development roots (%s)",
		errExtensionDevDisabled, abs, strings.Join(devPackageRootAllowedRoots(repoRoot), ", "),
	)
}

// devPathWithin 报告 path 是否等于 root 或位于 root 之下（两者都已绝对化）。
func devPathWithin(path, root string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if relative == "." {
		return true
	}
	return !strings.HasPrefix(relative, "..") && !filepath.IsAbs(relative)
}

// devWatchExcludedDirectories 是永不进入监视集合的目录名：构建产物、依赖、VCS、归档。
func devWatchExcludedDirectories() map[string]struct{} {
	return map[string]struct{}{
		"node_modules": {}, "dist": {}, "vendor": {}, ".git": {}, ".nuxt": {},
		"tmp": {}, "temp": {}, ".cache": {}, "coverage": {}, "__pycache__": {},
	}
}

// devWatchExcludedSuffixes 是永不触发重建的文件后缀：归档、编辑器交换文件、锁文件、临时文件。
func devWatchExcludedSuffixes() []string {
	return []string{
		".sforum.zip", ".zip", ".tmp", ".swp", ".swx", ".swo", "~", ".log", ".map",
	}
}

// devWatchRelevant 判断包内相对路径是否是「源输入」。
// 只有源输入会触发重建；构建产物与临时文件必须被忽略，否则循环会自己触发自己。
func devWatchRelevant(relative string) bool {
	relative = filepath.ToSlash(strings.TrimSpace(relative))
	if relative == "" || relative == "." {
		return false
	}
	segments := strings.Split(relative, "/")
	for index, segment := range segments {
		if index == len(segments)-1 {
			break
		}
		if _, excluded := devWatchExcludedDirectories()[strings.ToLower(segment)]; excluded {
			return false
		}
	}
	lower := strings.ToLower(filepath.Base(relative))
	if strings.HasPrefix(lower, ".") && lower != ".htaccess" {
		return false
	}
	for _, suffix := range devWatchExcludedSuffixes() {
		if strings.HasSuffix(lower, suffix) {
			return false
		}
	}
	return true
}

// devBuildResult 是一次构建事务的结果，携带精确制品身份。
type devBuildResult struct {
	PackageDigest string
	Phases        []string
}

// devBuildRunner 抽象一次「构建 + 摘要 + 校验 + 契约测试」事务。
// 生产实现直接调用既有 CLI 实现，不 shell 解析输出。
type devBuildRunner interface {
	Run(ctx context.Context, root string) (devBuildResult, error)
}

// devCommandBuildRunner 复用既有实现：runExtensionBuild 内部已依次执行
// 作者侧前端构建、摘要刷新、包校验与契约测试。
type devCommandBuildRunner struct {
	out         io.Writer
	parent      *cobra.Command
	skipInstall bool
}

func (r devCommandBuildRunner) Run(ctx context.Context, root string) (devBuildResult, error) {
	command := &cobra.Command{}
	command.SetOut(r.out)
	command.SetErr(r.out)
	command.SetContext(ctx)
	if r.parent != nil {
		command.SetArgs(nil)
	}
	if err := runExtensionBuild(command, root, extensionBuildOptions{SkipInstall: r.skipInstall}); err != nil {
		return devBuildResult{}, err
	}
	digest, err := extensionpackage.DigestTree(root)
	if err != nil {
		return devBuildResult{}, err
	}
	return devBuildResult{PackageDigest: digest, Phases: []string{"build", "digest", "validate", "test"}}, nil
}

// devStatusSink 输出一条阶段/状态流；文本模式给人看，JSON 模式给编辑器集成。
type devStatusSink interface {
	Phase(name string, fields map[string]any)
	Failed(stage string, err error, keptDigest string)
}

type devTextStatusSink struct{ out io.Writer }

func (s devTextStatusSink) Phase(name string, fields map[string]any) {
	digest, _ := fields["packageDigest"].(string)
	line := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), name)
	if digest != "" {
		line += " digest=" + digest
	}
	if trigger, ok := fields["trigger"].(string); ok && trigger != "" {
		line += " trigger=" + trigger
	}
	if count, ok := fields["changed"].(int); ok && count > 0 {
		line += fmt.Sprintf(" changed=%d", count)
	}
	fmt.Fprintln(s.out, line)
}

func (s devTextStatusSink) Failed(stage string, err error, keptDigest string) {
	fmt.Fprintf(s.out, "[%s] FAILED %s: %v\n", time.Now().Format("15:04:05"), stage, err)
	if keptDigest != "" {
		fmt.Fprintf(s.out, "[%s] last healthy artifact stays active: %s\n", time.Now().Format("15:04:05"), keptDigest)
	}
}

type devJSONStatusSink struct {
	out  io.Writer
	mu   sync.Mutex
	seen bool
}

func (s *devJSONStatusSink) write(payload map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintln(s.out, string(body))
	s.seen = true
}

func (s *devJSONStatusSink) Phase(name string, fields map[string]any) {
	payload := map[string]any{"event": "phase", "phase": name}
	for key, value := range fields {
		payload[key] = value
	}
	s.write(payload)
}

func (s *devJSONStatusSink) Failed(stage string, err error, keptDigest string) {
	payload := map[string]any{"event": "failed", "stage": stage, "error": err.Error()}
	if keptDigest != "" {
		payload["lastHealthyDigest"] = keptDigest
	}
	s.write(payload)
}

// devWatcher 抽象文件监视器，便于测试注入确定性事件。
type devWatcher interface {
	Add(path string) error
	Events() <-chan string
	Errors() <-chan error
	Close() error
}

type devFSWatcher struct{ watcher *fsnotify.Watcher }

func newDevFSWatcher() (devWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &devFSWatcher{watcher: watcher}, nil
}

func (w *devFSWatcher) Add(path string) error { return w.watcher.Add(path) }

func (w *devFSWatcher) Events() <-chan string {
	out := make(chan string, 256)
	go func() {
		defer close(out)
		for event := range w.watcher.Events {
			out <- event.Name
		}
	}()
	return out
}

func (w *devFSWatcher) Errors() <-chan error { return w.watcher.Errors }

func (w *devFSWatcher) Close() error { return w.watcher.Close() }

// devFileStamp 是监视集合内一个源输入的戳。构建事务自身会重写清单（摘要刷新），
// 那是**我们自己**的写入：必须与作者的编辑区分开，否则循环会自己触发自己。
type devFileStamp struct {
	modTime int64
	size    int64
}

func devStampFile(path string) (devFileStamp, bool) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return devFileStamp{}, false
	}
	return devFileStamp{modTime: info.ModTime().UnixNano(), size: info.Size()}, true
}

// devWatchLoop 是去抖 + 合并 + 单事务的编辑循环。
type devWatchLoop struct {
	root       string
	runner     devBuildRunner
	watcher    devWatcher
	status     devStatusSink
	debounce   time.Duration
	lastDigest string
	stamps     map[string]devFileStamp
}

// run 阻塞直到 ctx 取消。返回值是最后一个健康制品摘要（可能为空）。
func (l *devWatchLoop) run(ctx context.Context) (string, error) {
	if l.debounce <= 0 {
		l.debounce = devDefaultDebounce
	}
	if l.watcher == nil || l.runner == nil || l.status == nil {
		return l.lastDigest, errors.New("extension dev loop is not configured")
	}
	if err := l.watchTree(); err != nil {
		return l.lastDigest, err
	}
	l.snapshotWatchedInputs()
	l.status.Phase("watching", map[string]any{"root": l.root})
	// 启动即对账一次：作者在启动前刚编辑过时，包可能处于摘要过期状态。
	l.buildOnce(ctx, map[string]any{"trigger": "startup"})

	events := l.watcher.Events()
	watcherErrors := l.watcher.Errors()
	pending := map[string]struct{}{}
	var timer *time.Timer
	var timerC <-chan time.Time
	dropped := 0

	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return l.lastDigest, nil
		case err, ok := <-watcherErrors:
			if !ok {
				watcherErrors = nil
				continue
			}
			l.status.Failed("watch", err, l.lastDigest)
		case path, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			if !l.eventRelevant(path) {
				continue
			}
			if len(pending) >= devMaximumCoalescedTriggers {
				dropped++
			} else {
				pending[path] = struct{}{}
			}
			if timer == nil {
				timer = time.NewTimer(l.debounce)
			} else {
				timer.Reset(l.debounce)
			}
			timerC = timer.C
		case <-timerC:
			timerC = nil
			changed := make([]string, 0, len(pending))
			for path := range pending {
				changed = append(changed, path)
			}
			pending = map[string]struct{}{}
			fields := map[string]any{"changed": len(changed), "trigger": "change"}
			if dropped > 0 {
				fields["coalescedOverflow"] = dropped
				dropped = 0
			}
			// 单事务：本次构建期间到达的事件留在 pending，下一轮去抖后再执行。
			l.buildOnce(ctx, fields)
		}
	}
}

func (l *devWatchLoop) buildOnce(ctx context.Context, fields map[string]any) {
	l.status.Phase("building", fields)
	result, err := l.runner.Run(ctx, l.root)
	// 事务结束后立刻重新快照：本次事务自己写入的清单/产物不算作者编辑。
	l.snapshotWatchedInputs()
	if err != nil {
		l.status.Failed("build", err, l.lastDigest)
		return
	}
	l.lastDigest = result.PackageDigest
	l.status.Phase("built", map[string]any{"packageDigest": result.PackageDigest})
	// 不绕过信任：摘要变化后仍需操作员在管理端确认。这里只报告下一步。
	l.status.Phase("reload_required", map[string]any{
		"packageDigest": result.PackageDigest,
		"next":          "confirm the exact artifact in the admin extensions flow to stage and activate it",
	})
}

// eventRelevant 只接受包内、可监视、属于源输入、且**内容状态确实变化**的事件。
// 构建事务自己重写的文件（摘要刷新后的清单）会与上次快照一致，因此被忽略：
// 这是防止重建循环的关键，不能只靠后缀/目录过滤。
func (l *devWatchLoop) eventRelevant(path string) bool {
	relative, err := filepath.Rel(l.root, path)
	if err != nil {
		return false
	}
	if strings.HasPrefix(relative, "..") || relative == "." {
		return false
	}
	if !devWatchRelevant(relative) {
		return false
	}
	stamp, exists := devStampFile(path)
	if !exists {
		// 删除或改名：内容状态消失本身就是作者动作，必须重建。
		return true
	}
	if previous, known := l.stamps[path]; known && previous == stamp {
		return false
	}
	return true
}

// snapshotWatchedInputs 记录监视集合内全部源输入的当前戳。
func (l *devWatchLoop) snapshotWatchedInputs() {
	stamps := make(map[string]devFileStamp, len(l.stamps)+16)
	err := filepath.WalkDir(l.root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			if path != l.root {
				if _, excluded := devWatchExcludedDirectories()[strings.ToLower(entry.Name())]; excluded {
					return filepath.SkipDir
				}
			}
			return nil
		}
		relative, relErr := filepath.Rel(l.root, path)
		if relErr != nil || !devWatchRelevant(relative) {
			return nil
		}
		if stamp, ok := devStampFile(path); ok {
			stamps[path] = stamp
		}
		return nil
	})
	if err != nil {
		return
	}
	l.stamps = stamps
}

// watchTree 递归登记包内目录（跳过被排除目录），不监视单个文件。
func (l *devWatchLoop) watchTree() error {
	return filepath.WalkDir(l.root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if path != l.root {
			if _, excluded := devWatchExcludedDirectories()[strings.ToLower(entry.Name())]; excluded {
				return filepath.SkipDir
			}
		}
		return l.watcher.Add(path)
	})
}

func newExtensionDevCommand() *cobra.Command {
	var (
		asJSON      bool
		debounce    time.Duration
		once        bool
		skipInstall bool
	)
	cmd := &cobra.Command{
		Use:   "dev [path]",
		Short: "Watch a development extension package and rebuild it on source changes",
		Long: `Development-only edit loop for one extension package.

It watches package-contained source inputs, coalesces changes, runs one build
transaction at a time through the existing build/digest/validate/test
implementations, and reports the exact package digest of every cycle.

It never bypasses trust: the digest changes on every rebuild, so the operator
still confirms the exact artifact in the admin flow before it becomes active.
The last healthy artifact stays active when a build or preflight fails.

Requires APP_ENV=development and a package under extensions/dev, extensions/optional,
or extensions/fixtures. Protected built-in trees are refused.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			return runExtensionDev(cmd, root, extensionDevOptions{
				AsJSON: asJSON, Debounce: debounce, Once: once, SkipInstall: skipInstall,
			})
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Emit one JSON object per phase for editor integrations")
	cmd.Flags().DurationVar(&debounce, "debounce", devDefaultDebounce, "Coalesce file events for this long before building")
	cmd.Flags().BoolVar(&once, "once", false, "Run exactly one build transaction and exit (no watcher)")
	cmd.Flags().BoolVar(&skipInstall, "skip-install", false, "Skip bun install and use the existing frontend dependencies")
	return cmd
}

type extensionDevOptions struct {
	AsJSON      bool
	Debounce    time.Duration
	Once        bool
	SkipInstall bool
}

func runExtensionDev(cmd *cobra.Command, root string, opts extensionDevOptions) error {
	// 门禁与 DevHygiene reaper 完全一致：唯一被仓库信任的开发模式判据。
	if !devhygiene.ShouldEnableDevelopmentOrphanReaper(os.Getenv("APP_ENV")) {
		return fmt.Errorf(
			"%w: APP_ENV must be development (current: %q)",
			errExtensionDevDisabled, strings.TrimSpace(os.Getenv("APP_ENV")),
		)
	}
	abs, err := resolveDevPackageRoot(root)
	if err != nil {
		return err
	}
	// 只要求「这里有一个 Manifest V3 包」。作者刚编辑过文件时摘要必然过期——那正是
	// 本命令要修的状态；真正的校验由构建事务内的 validate/test 阶段给出。
	manifestPath := filepath.Join(abs, extensionmanifest.ManifestFileName)
	if info, err := os.Stat(manifestPath); err != nil || info.IsDir() {
		return fmt.Errorf("extension dev requires a package manifest at %s: %w", manifestPath, err)
	}

	out := cmd.OutOrStdout()
	var status devStatusSink
	if opts.AsJSON {
		status = &devJSONStatusSink{out: out}
	} else {
		status = devTextStatusSink{out: out}
	}
	runner := devCommandBuildRunner{out: out, parent: cmd, skipInstall: opts.SkipInstall}

	// 裸 cobra.Command（测试与编辑器集成）没有 context；SIGINT/SIGTERM 仍必须被接管，
	// 否则退出会留下 watcher 与构建子进程。
	baseContext := cmd.Context()
	if baseContext == nil {
		baseContext = context.Background()
	}
	ctx, stop := signal.NotifyContext(baseContext, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if opts.Once {
		loop := &devWatchLoop{root: abs, runner: runner, status: status}
		loop.buildOnce(ctx, map[string]any{"trigger": "once"})
		return nil
	}

	watcher, err := newDevFSWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	loop := &devWatchLoop{
		root: abs, runner: runner, watcher: watcher, status: status, debounce: opts.Debounce,
	}
	digest, err := loop.run(ctx)
	if err != nil {
		return err
	}
	status.Phase("stopped", map[string]any{"packageDigest": digest})
	return nil
}
