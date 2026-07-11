package adminoverview

import "time"

const (
	WindowDays = 7

	ActionModerationQueue       = "moderation_queue"
	ActionPendingTags           = "pending_tags"
	ActionOrphanAttachments     = "orphan_attachments"
	ActionFailedExtensionEvents = "failed_extension_events"
	ActionThemeReleaseProgress  = "theme_release_progress"
)

type AdminOverview struct {
	GeneratedAt   time.Time          `json:"generatedAt"`
	WindowDays    int                `json:"windowDays"`
	Runtime       RuntimeStats       `json:"runtime"`
	Community     CommunityStats     `json:"community"`
	Attachments   AttachmentStats    `json:"attachments"`
	Moderation    ModerationStats    `json:"moderation"`
	Extensions    ExtensionStats     `json:"extensions"`
	Trends        TrendStats         `json:"trends"`
	TopCategories []CategoryActivity `json:"topCategories"`
	Actions       []OverviewAction   `json:"actions"`
}

type StoreSnapshot struct {
	Community     CommunityStats
	Attachments   AttachmentStats
	Moderation    ModerationStats
	Extensions    ExtensionStats
	Trends        []TrendDay
	TopCategories []CategoryActivity
}

type RuntimeStats struct {
	StartedAt      time.Time            `json:"startedAt"`
	UptimeSeconds  int64                `json:"uptimeSeconds"`
	MemoryBytes    uint64               `json:"memoryBytes"`
	HeapAllocBytes uint64               `json:"heapAllocBytes"`
	HeapSysBytes   uint64               `json:"heapSysBytes"`
	GoroutineCount int                  `json:"goroutineCount"`
	GCCount        uint32               `json:"gcCount"`
	LastGCPauseNs  uint64               `json:"lastGcPauseNs"`
	Database       DatabaseRuntimeStats `json:"database"`
	// Worker 心跳与队列积压（F1.2）；探测失败时字段可为空。
	Worker   *WorkerRuntimeStats `json:"worker,omitempty"`
	QueueLag *QueueLagStats      `json:"queueLag,omitempty"`
}

// WorkerRuntimeStats 来自 Redis heartbeat（独立或嵌入 worker 共用 key）。
type WorkerRuntimeStats struct {
	LastSeenAt *time.Time `json:"lastSeenAt,omitempty"`
	AgeSeconds *int64     `json:"ageSeconds,omitempty"`
	Stale      bool       `json:"stale"`
	// Status: ok | stale | unknown
	Status string `json:"status"`
}

// QueueLagStats 廉价聚合：River 中等待执行的 job 数量（available+scheduled+retryable）。
type QueueLagStats struct {
	Waiting int64 `json:"waiting"`
	Running int64 `json:"running"`
	Failed  int64 `json:"failed"`
}

type DatabaseRuntimeStats struct {
	MaxConnections      int32 `json:"maxConnections"`
	TotalConnections    int32 `json:"totalConnections"`
	AcquiredConnections int32 `json:"acquiredConnections"`
	IdleConnections     int32 `json:"idleConnections"`
}

type CommunityStats struct {
	UserCount         int64 `json:"userCount"`
	ActiveUserCount   int64 `json:"activeUserCount"`
	DisabledUserCount int64 `json:"disabledUserCount"`
	BannedUserCount   int64 `json:"bannedUserCount"`
	TopicCount        int64 `json:"topicCount"`
	ActiveTopicCount  int64 `json:"activeTopicCount"`
	LockedTopicCount  int64 `json:"lockedTopicCount"`
	HiddenTopicCount  int64 `json:"hiddenTopicCount"`
	DeletedTopicCount int64 `json:"deletedTopicCount"`
	CommentCount      int64 `json:"commentCount"`
	PostCount         int64 `json:"postCount"`
	CategoryCount     int64 `json:"categoryCount"`
	TagCount          int64 `json:"tagCount"`
	PendingTagCount   int64 `json:"pendingTagCount"`
	TotalViews        int64 `json:"totalViews"`
}

type AttachmentStats struct {
	TotalCount    int64 `json:"totalCount"`
	ActiveCount   int64 `json:"activeCount"`
	DisabledCount int64 `json:"disabledCount"`
	DeletedCount  int64 `json:"deletedCount"`
	OrphanCount   int64 `json:"orphanCount"`
	TotalBytes    int64 `json:"totalBytes"`
}

type ModerationStats struct {
	OpenCount      int64 `json:"openCount"`
	ReviewingCount int64 `json:"reviewingCount"`
	ResolvedCount  int64 `json:"resolvedCount"`
	RejectedCount  int64 `json:"rejectedCount"`
}

type ExtensionStats struct {
	TotalCount                  int64 `json:"totalCount"`
	EnabledCount                int64 `json:"enabledCount"`
	PluginCount                 int64 `json:"pluginCount"`
	ThemeCount                  int64 `json:"themeCount"`
	InstalledPluginRuntimeCount int64 `json:"installedPluginRuntimeCount"`
	FailedEventCount            int64 `json:"failedEventCount"`
	PendingThemeReleaseCount    int64 `json:"pendingThemeReleaseCount"`
	FailedThemeReleaseCount     int64 `json:"failedThemeReleaseCount"`
	ActiveThemeReleaseCount     int64 `json:"activeThemeReleaseCount"`
}

type TrendStats struct {
	Days []TrendDay `json:"days"`
}

type TrendDay struct {
	Date         string `json:"date"`
	TopicCount   int64  `json:"topicCount"`
	CommentCount int64  `json:"commentCount"`
	UserCount    int64  `json:"userCount"`
}

type CategoryActivity struct {
	ID           int64  `json:"id"`
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	TopicCount   int64  `json:"topicCount"`
	CommentCount int64  `json:"commentCount"`
}

type OverviewAction struct {
	Key      string `json:"key"`
	Count    int64  `json:"count"`
	Severity string `json:"severity"`
	Route    string `json:"route"`
}
