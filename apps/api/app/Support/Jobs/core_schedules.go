package jobs

import (
	"time"

	"github.com/riverqueue/river"
)

// 核心 schedule 稳定 ID。新增维护任务时先加常量与 CoreScheduleDefinitions，再在 bootstrap 注入 Constructor。
const (
	ScheduleIdentityCleanupSessions         = "identity.cleanup_sessions"
	ScheduleAttachmentsCleanupOrphans       = "attachments.cleanup_orphans"
	ScheduleAttachmentsReconcileCompression = "attachments.reconcile_compression"
	// ScheduleAuditCleanupEvents 清理过期 audit_events（F1.4 保留期 job）。
	ScheduleAuditCleanupEvents = "audit.cleanup_events"
	// ScheduleForumAutoLockIdle 按站点 autoLockIdleDays 锁定闲置主题。
	ScheduleForumAutoLockIdle = "forum.auto_lock_idle"
	// ScheduleForumFlushViewCounts 将 Redis 浏览增量刷入 topics.view_count / hot_score。
	ScheduleForumFlushViewCounts = "forum.flush_view_counts"
	// ScheduleSearchReconcile 增量修复当前搜索 provider 的漏建、过期和幽灵文档。
	ScheduleSearchReconcile = "search.reconcile"
)

// CoreScheduleDefinitions 返回宿主内置 schedule 目录模板（无 Constructor）。
// bootstrap 负责填入 Constructor 后 Register；admin 列表可直接用这些元数据。
func CoreScheduleDefinitions() []ScheduleDefinition {
	return []ScheduleDefinition{
		{
			ID:          ScheduleIdentityCleanupSessions,
			JobKind:     ScheduleIdentityCleanupSessions,
			Queue:       QueueDefault,
			Interval:    24 * time.Hour,
			Owner:       "identity",
			Enabled:     true,
			Description: "清理已下线超过保留期的历史会话行",
			RunOnStart:  false,
		},
		{
			ID:          ScheduleAttachmentsCleanupOrphans,
			JobKind:     ScheduleAttachmentsCleanupOrphans,
			Queue:       QueueMaintenance,
			Interval:    24 * time.Hour,
			Owner:       "attachments",
			Enabled:     true,
			Description: "清理超过保留期且无引用的孤儿附件",
			RunOnStart:  false,
		},
		{
			ID:          ScheduleAttachmentsReconcileCompression,
			JobKind:     ScheduleAttachmentsReconcileCompression,
			Queue:       QueueMaintenance,
			Interval:    time.Minute,
			Owner:       "attachments",
			Enabled:     true,
			Description: "补偿附件图片压缩任务入队失败并继续待处理任务",
			RunOnStart:  true,
		},
		{
			ID:          ScheduleAuditCleanupEvents,
			JobKind:     ScheduleAuditCleanupEvents,
			Queue:       QueueMaintenance,
			Interval:    24 * time.Hour,
			Owner:       "audit",
			Enabled:     true,
			Description: "清理超过保留期的审计日志（默认 90 天）",
			RunOnStart:  false,
		},
		{
			ID:          ScheduleForumAutoLockIdle,
			JobKind:     ScheduleForumAutoLockIdle,
			Queue:       QueueMaintenance,
			Interval:    24 * time.Hour,
			Owner:       "forum",
			Enabled:     true,
			Description: "按站点 autoLockIdleDays 锁定闲置主题（0 关闭时 job 空跑）",
			RunOnStart:  false,
		},
		{
			ID:          ScheduleForumFlushViewCounts,
			JobKind:     ScheduleForumFlushViewCounts,
			Queue:       QueueMaintenance,
			Interval:    45 * time.Second,
			Owner:       "forum",
			Enabled:     true,
			Description: "将 Redis 主题浏览增量刷入 PG view_count/hot_score（D3）",
			RunOnStart:  false,
		},
		{
			ID:          ScheduleSearchReconcile,
			JobKind:     ScheduleSearchReconcile,
			Queue:       QueueMaintenance,
			Interval:    15 * time.Minute,
			Owner:       "search",
			Enabled:     true,
			Description: "核对并修复当前搜索 provider 的缺失、过期和幽灵文档",
			RunOnStart:  true,
		},
	}
}

// NewCoreScheduleRegistry 用内置模板构建 registry。
// constructors 按 schedule ID 注入；缺失时条目仍进入 catalog，但不生成 River PeriodicJob。
func NewCoreScheduleRegistry(constructors map[string]river.PeriodicJobConstructor) (*ScheduleRegistry, error) {
	reg := NewScheduleRegistry()
	for _, def := range CoreScheduleDefinitions() {
		if constructors != nil {
			def.Constructor = constructors[def.ID]
		}
		if err := reg.Register(def); err != nil {
			return nil, err
		}
	}
	return reg, nil
}
