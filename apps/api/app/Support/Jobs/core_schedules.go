package jobs

import (
	"time"

	"github.com/riverqueue/river"
)

// 核心 schedule 稳定 ID。新增维护任务时先加常量与 CoreScheduleDefinitions，再在 bootstrap 注入 Constructor。
const (
	ScheduleIdentityCleanupSessions    = "identity.cleanup_sessions"
	ScheduleExtensionWebReleaseCleanup = "extension.web_release_cleanup"
	ScheduleAttachmentsCleanupOrphans  = "attachments.cleanup_orphans"
	// ScheduleAuditCleanupEvents 清理过期 audit_events（F1.4 保留期 job）。
	ScheduleAuditCleanupEvents = "audit.cleanup_events"
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
			ID:          ScheduleExtensionWebReleaseCleanup,
			JobKind:     ScheduleExtensionWebReleaseCleanup,
			Queue:       QueueMaintenance,
			Interval:    24 * time.Hour,
			Owner:       "extensions",
			Enabled:     true,
			Description: "清理过期 Web Release 构建产物与日志",
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
			ID:          ScheduleAuditCleanupEvents,
			JobKind:     ScheduleAuditCleanupEvents,
			Queue:       QueueMaintenance,
			Interval:    24 * time.Hour,
			Owner:       "audit",
			Enabled:     true,
			Description: "清理超过保留期的审计日志（默认 90 天）",
			RunOnStart:  false,
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
