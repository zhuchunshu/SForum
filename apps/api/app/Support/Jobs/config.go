package jobs

import (
	"github.com/riverqueue/river"

	"github.com/zhuchunshu/sforum/apps/api/config"
)

type Config struct {
	CriticalWorkers      int
	DefaultWorkers       int
	SearchWorkers        int
	MailWorkers          int
	NotificationsWorkers int
	MaintenanceWorkers   int
	ThemeWorkers         int
}

func FromAppConfig(cfg config.Config) Config {
	return Config{
		CriticalWorkers:      positiveOrDefault(cfg.JobQueueCriticalWorkers, 4),
		DefaultWorkers:       positiveOrDefault(cfg.JobQueueDefaultWorkers, 8),
		SearchWorkers:        positiveOrDefault(cfg.JobQueueSearchWorkers, 6),
		MailWorkers:          positiveOrDefault(cfg.JobQueueMailWorkers, 4),
		NotificationsWorkers: positiveOrDefault(cfg.JobQueueNotificationsWorkers, 6),
		MaintenanceWorkers:   positiveOrDefault(cfg.JobQueueMaintenanceWorkers, 2),
		ThemeWorkers:         positiveOrDefault(cfg.JobQueueThemeWorkers, 1),
	}
}

func (cfg Config) RiverQueues() map[string]river.QueueConfig {
	return map[string]river.QueueConfig{
		QueueCritical:      {MaxWorkers: cfg.CriticalWorkers},
		QueueDefault:       {MaxWorkers: cfg.DefaultWorkers},
		QueueSearch:        {MaxWorkers: cfg.SearchWorkers},
		QueueMail:          {MaxWorkers: cfg.MailWorkers},
		QueueNotifications: {MaxWorkers: cfg.NotificationsWorkers},
		QueueMaintenance:   {MaxWorkers: cfg.MaintenanceWorkers},
		QueueTheme:         {MaxWorkers: cfg.ThemeWorkers},
	}
}

func positiveOrDefault(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}
