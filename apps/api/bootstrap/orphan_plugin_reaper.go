package bootstrap

import (
	"log/slog"

	devhygiene "github.com/zhuchunshu/sforum/apps/api/app/Support/DevHygiene"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

// startDevelopmentOrphanPluginReaper 在 development 下延迟清理 air 热重载遗留的
// 扩展 backend plugin 孤儿。air 顺序是 pre_cmd → build → 启动新进程 → 再停旧进程，
// 因此 pre_cmd 清理时旧 API 仍拥有插件；必须在 kill_delay 之后再扫。
func startDevelopmentOrphanPluginReaper(cfg config.Config, logger *slog.Logger) (stop func()) {
	return devhygiene.StartOrphanPluginReaper(devhygiene.ReaperConfig{
		Enabled: devhygiene.ShouldEnableDevelopmentOrphanReaper(cfg.AppEnv),
		Delays:  devhygiene.DefaultReaperDelays(),
		Logger:  logger,
	})
}
