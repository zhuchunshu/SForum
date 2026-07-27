package extensioncomposition

import "time"

// Inspector exposes a redacted, read-only view of live component composition.
// Runtime artifacts, schemas, rendered values and raw errors stay behind the
// runtime package boundary.
type Inspector interface {
	Inspect(limit int) Snapshot
}

type Snapshot struct {
	Revision          uint64     `json:"revision"`
	SafeMode          bool       `json:"safeMode"`
	TargetCount       int        `json:"targetCount"`
	ContributionCount int        `json:"contributionCount"`
	Conflicts         []Conflict `json:"conflicts"`
	Traces            []Trace    `json:"traces"`
}

type Conflict struct {
	TargetID              string `json:"targetId"`
	TargetContractVersion string `json:"targetContractVersion"`
	CandidateCount        int    `json:"candidateCount"`
	WinnerContributionID  string `json:"winnerContributionId,omitempty"`
	ExplicitSelection     bool   `json:"explicitSelection"`
}

type Trace struct {
	ID                    string    `json:"id"`
	Revision              uint64    `json:"revision"`
	TargetID              string    `json:"targetId"`
	TargetContractVersion string    `json:"targetContractVersion"`
	StartedAt             time.Time `json:"startedAt"`
	DurationMicros        int64     `json:"durationMicros"`
	Status                string    `json:"status"`
}
