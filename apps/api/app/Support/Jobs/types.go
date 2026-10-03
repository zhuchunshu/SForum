package jobs

import (
	"time"

	"github.com/riverqueue/river"
)

const (
	QueueCritical      = "critical"
	QueueDefault       = river.QueueDefault
	QueueSearch        = "search"
	QueueMail          = "mail"
	QueueNotifications = "notifications"
	QueueMaintenance   = "maintenance"
	// QueueAI 承载 AI 生成任务。它单独成队是因为这类任务慢（秒级）且失败模式
	// 与其它队列不同：放共享队列会占用工作槽，拖住邮件与通知投递。
	QueueAI = "ai"
)

type EnqueueOptions struct {
	Queue       string
	MaxAttempts int
	ScheduledAt time.Time
	Unique      river.UniqueOpts
}

func (opts EnqueueOptions) RiverInsertOpts() *river.InsertOpts {
	insertOpts := &river.InsertOpts{}
	if opts.Queue != "" {
		insertOpts.Queue = opts.Queue
	}
	if opts.MaxAttempts > 0 {
		insertOpts.MaxAttempts = opts.MaxAttempts
	}
	if !opts.ScheduledAt.IsZero() {
		insertOpts.ScheduledAt = opts.ScheduledAt
	}
	insertOpts.UniqueOpts = opts.Unique
	return insertOpts
}
