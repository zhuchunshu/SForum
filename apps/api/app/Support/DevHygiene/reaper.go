package devhygiene

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// ReaperConfig 控制开发态孤儿插件延迟清理（应对 air：先起新进程再杀旧进程）。
type ReaperConfig struct {
	// Enabled 为 false 时 Start 立即返回 no-op stop。
	Enabled bool
	// Delays 是相对启动时刻的清理触发延迟；默认覆盖 air kill_delay=2s 之后。
	Delays []time.Duration
	// Cleanup 可注入；默认 CleanupOrphanExtensionPlugins。
	Cleanup func() (CleanupResult, error)
	Logger  *slog.Logger
	// Clock 可注入；默认 time.Now / NewTimer。
	After func(d time.Duration) <-chan time.Time
}

// DefaultReaperDelays 在 air kill_delay(2s) 之后再扫两轮，覆盖热重载竞态。
func DefaultReaperDelays() []time.Duration {
	return []time.Duration{3 * time.Second, 6 * time.Second}
}

// ShouldEnableDevelopmentOrphanReaper 仅 development 默认启用。
func ShouldEnableDevelopmentOrphanReaper(appEnv string) bool {
	return strings.EqualFold(strings.TrimSpace(appEnv), "development")
}

// StartOrphanPluginReaper 在后台按 Delays 触发清理；返回 stop 取消后续轮次。
func StartOrphanPluginReaper(cfg ReaperConfig) (stop func()) {
	if !cfg.Enabled {
		return func() {}
	}
	delays := cfg.Delays
	if len(delays) == 0 {
		delays = DefaultReaperDelays()
	}
	cleanup := cfg.Cleanup
	if cleanup == nil {
		cleanup = func() (CleanupResult, error) {
			return CleanupOrphanExtensionPlugins(CleanupOptions{})
		}
	}
	after := cfg.After
	if after == nil {
		after = time.After
	}
	logger := cfg.Logger

	ctx, cancel := context.WithCancel(context.Background())
	var once sync.Once
	stop = func() {
		once.Do(cancel)
	}

	go func() {
		// 累计延迟：每项是距启动的绝对等待（与 DefaultReaperDelays 语义一致）。
		start := time.Now()
		for i, absolute := range delays {
			wait := absolute - time.Since(start)
			if wait < 0 {
				wait = 0
			}
			select {
			case <-ctx.Done():
				return
			case <-after(wait):
			}
			if ctx.Err() != nil {
				return
			}
			result, err := cleanup()
			if logger != nil {
				if err != nil {
					logger.Warn("orphan extension plugin reaper failed", "pass", i+1, "error", err)
					continue
				}
				if len(result.Selected) > 0 {
					logger.Info("orphan extension plugin reaper cleaned processes",
						"pass", i+1,
						"selected", len(result.Selected),
						"signaled", len(result.Signaled),
						"pids", result.Selected,
					)
				}
			}
		}
	}()
	return stop
}
