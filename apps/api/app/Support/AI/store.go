package ai

import (
	"context"
	"errors"
	"time"
)

// 执行状态。denied 表示被闸门拒绝，failed 表示调用失败，succeeded 表示成功。
const (
	ExecutionStatusSucceeded = "succeeded"
	ExecutionStatusFailed    = "failed"
	ExecutionStatusDenied    = "denied"
)

// 用量作用域与窗口，与 ai_usage_counters 的取值保持一致。
const (
	UsageScopeSite      = "site"
	UsageScopeExtension = "extension"
	UsageScopeUser      = "user"

	UsageWindowMinute = "minute"
	UsageWindowDay    = "day"
	UsageWindowMonth  = "month"
)

var (
	// ErrSettingsNotFound 表示配置行缺失，调用方应回落到 RecommendedSettings。
	ErrSettingsNotFound = errors.New("ai: settings row is not found")
)

// SettingsStore 持久化站点 AI 配置文档。
type SettingsStore interface {
	GetSettings(ctx context.Context) (Settings, error)
	SaveSettings(ctx context.Context, settings Settings, actorUserID int64) (Settings, error)
	ResetSettings(ctx context.Context, settings Settings, actorUserID int64) (Settings, error)
}

// UsageByScope 是闸门判定所需的三个作用域用量。
type UsageByScope struct {
	Site      UsageSnapshot
	Extension UsageSnapshot
	User      UsageSnapshot
}

// UsageEntry 是一次成功或失败的调用所消耗的用量。失败调用同样计入次数，
// 否则一个持续失败的插件可以无限重试而不消耗配额。
type UsageEntry struct {
	CallerExtensionID string
	CallerUserID      int64
	InputTokens       int
	OutputTokens      int
	SpendMicros       int64
}

// UsageStore 读写闸门计数。Snapshot 必须只读，Record 必须幂等可重放。
type UsageStore interface {
	Snapshot(ctx context.Context, now time.Time, callerExtensionID string, callerUserID int64) (UsageByScope, error)
	Record(ctx context.Context, now time.Time, entry UsageEntry) error
}

// ExecutionRecord 是一次 AI 执行的完整归因，用于审计、成本看板与回放。
// 它刻意不含提示词与响应正文：那两者可能包含用户内容。
type ExecutionRecord struct {
	ID                int64     `json:"id"`
	CreatedAt         time.Time `json:"createdAt"`
	Purpose           string    `json:"purpose"`
	CostClass         string    `json:"costClass,omitempty"`
	CallerExtensionID string    `json:"callerExtensionId,omitempty"`
	ProfileID         string    `json:"profileId,omitempty"`
	Protocol          string    `json:"protocol,omitempty"`
	Model             string    `json:"model,omitempty"`
	Status            string    `json:"status"`
	GateReason        string    `json:"gateReason,omitempty"`
	GateScope         string    `json:"gateScope,omitempty"`
	PromptVersion     string    `json:"promptVersion,omitempty"`
	ConfigRevision    int64     `json:"configRevision,omitempty"`
	LatencyMS         int       `json:"latencyMs,omitempty"`
	InputTokens       int       `json:"inputTokens,omitempty"`
	OutputTokens      int       `json:"outputTokens,omitempty"`
	CachedTokens      int       `json:"cachedTokens,omitempty"`
	SpendMicros       int64     `json:"spendMicros,omitempty"`
	CacheHit          bool      `json:"cacheHit,omitempty"`
	ErrorSummary      string    `json:"errorSummary,omitempty"`
}

// ExecutionStore 写入与读取执行 trace。
type ExecutionStore interface {
	RecordExecution(ctx context.Context, entry ExecutionRecord) error
	ListExecutions(ctx context.Context, limit int) ([]ExecutionRecord, error)
}

// TruncateDay 与 TruncateMonth 按 UTC 对齐窗口起点。闸门窗口使用 UTC 是刻意
// 选择：站点时区调整不应让当日配额在中途重置。
func TruncateDay(now time.Time) time.Time {
	utc := now.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func TruncateMonth(now time.Time) time.Time {
	utc := now.UTC()
	return time.Date(utc.Year(), utc.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func TruncateMinute(now time.Time) time.Time {
	return now.UTC().Truncate(time.Minute)
}
