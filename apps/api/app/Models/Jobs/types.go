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
