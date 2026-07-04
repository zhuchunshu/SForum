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
