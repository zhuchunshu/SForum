package jobs

import (
	"encoding/json"
	"time"
)

type Overview struct {
	Counts map[string]int64 `json:"counts"`
	Queues []Queue          `json:"queues"`
}

type Queue struct {
	Name      string     `json:"name"`
	PausedAt  *time.Time `json:"pausedAt,omitempty"`
	UpdatedAt time.Time  `json:"updatedAt"`
	Available int64      `json:"available"`
	Running   int64      `json:"running"`
	Failed    int64      `json:"failed"`
}

type Job struct {
	ID          int64           `json:"id"`
	Kind        string          `json:"kind"`
	Queue       string          `json:"queue"`
	State       string          `json:"state"`
	Attempt     int             `json:"attempt"`
	MaxAttempts int             `json:"maxAttempts"`
	Priority    int             `json:"priority"`
	Args        json.RawMessage `json:"args"`
	Metadata    json.RawMessage `json:"metadata"`
	Tags        []string        `json:"tags"`
	Errors      any             `json:"errors"`
	AttemptedBy []string        `json:"attemptedBy"`
	CreatedAt   time.Time       `json:"createdAt"`
	ScheduledAt time.Time       `json:"scheduledAt"`
	AttemptedAt *time.Time      `json:"attemptedAt,omitempty"`
	FinalizedAt *time.Time      `json:"finalizedAt,omitempty"`
}

type ListInput struct {
	Queue string
	Kind  string
	State string
	Limit int
}

// Schedule 是 admin 周期任务目录投影。
// Enabled 来自 web_options 覆盖（缺失=目录默认 true）。
// LastRunAt / NextRunAt 由 River 历史 + interval 估算。
type Schedule struct {
	ID              string     `json:"id"`
	JobKind         string     `json:"jobKind"`
	Queue           string     `json:"queue"`
	IntervalSeconds int64      `json:"intervalSeconds,omitempty"`
	Cron            string     `json:"cron,omitempty"`
	Owner           string     `json:"owner"`
	Enabled         bool       `json:"enabled"`
	Description     string     `json:"description"`
	RunOnStart      bool       `json:"runOnStart"`
	LastRunAt       *time.Time `json:"lastRunAt,omitempty"`
	NextRunAt       *time.Time `json:"nextRunAt,omitempty"`
}

// TriggerResult 是手动触发一次 schedule 的结果。
type TriggerResult struct {
	ScheduleID string `json:"scheduleId"`
	JobID      int64  `json:"jobId"`
	Kind       string `json:"kind"`
	Queue      string `json:"queue"`
	// UniqueSkipped 为 true 表示 River 因唯一约束跳过插入，返回的是已有 job。
	UniqueSkipped bool `json:"uniqueSkipped,omitempty"`
}
