// Package audit 提供宿主统一审计写入入口（F1.4 最小集）。
//
// 权限/角色变更已由 identity.PostgresStore 在事务内写入 audit_events；
// 本包供 options、extensions 等模块在业务成功后追加同表记录，避免各处手写 SQL。
// 保留期清理由 schedule registry 的 audit.cleanup_events 周期任务执行。
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// 稳定 action 常量。新增敏感操作时优先复用或扩展此列表。
const (
	ActionSettingsUpdate             = "settings.update"
	ActionExtensionSettingsAction    = "extension.settings.action"
	ActionExtensionEnable            = "extension.enable"
	ActionExtensionDisable           = "extension.disable"
	ActionExtensionActivate          = "extension.theme_activate"
	ActionExtensionInstalled         = "extension.install"
	ActionExtensionUpgraded          = "extension.upgrade"
	ActionExtensionUninstalled       = "extension.uninstall"
	ActionExtensionFrontendGrant     = "extension.frontend_trust_grant"
	ActionExtensionFrontendRevoke    = "extension.frontend_trust_revoke"
	ActionExtensionTrustChallenge    = "extension.trust_challenge"
	ActionExtensionTrustGrant        = "extension.trust_grant"
	ActionExtensionTrustRevoke       = "extension.trust_revoke"
	ActionExtensionTrustDenied       = "extension.trust_denied"
	ActionExtensionSafeModeBoot      = "extension.safe_mode_boot"
	ActionExtensionCLIRecovery       = "extension.cli_recovery"
	ActionExtensionActivationFailed  = "extension.activation_failed"
	ActionExtensionActivationSkipped = "extension.activation_skipped"
	// ActionExtensionBackendDenied 非 super_admin 试图引入/执行非内置后端插件。
	ActionExtensionBackendDenied = "extension.backend_execution_denied"

	// Page Registry：核心页 replace 批准 / 恢复 / 冲突选择。
	ActionPageReplaceApprove = "pages.replace_approve"
	ActionPageRestoreCore    = "pages.restore_core"
	ActionPageConflictSelect = "pages.conflict_select"

	// RecommendedRetentionDays 审计日志推荐保留天数（清理 job 默认）。
	RecommendedRetentionDays = 90
)

// Event 是一条 append-only 审计记录。
type Event struct {
	ActorUserID  int64
	TargetUserID int64 // 0 表示无目标用户
	Action       string
	Metadata     map[string]any
}

// Writer 抽象，便于测试注入。
type Writer interface {
	Append(ctx context.Context, event Event) error
}

// Cleaner 清理过期审计行。
type Cleaner interface {
	DeleteOlderThan(ctx context.Context, keepDays int) (int64, error)
}

// PostgresWriter 直接写入 identity 的 audit_events 表。
type PostgresWriter struct {
	Pool *pgxpool.Pool
}

func NewPostgresWriter(pool *pgxpool.Pool) *PostgresWriter {
	return &PostgresWriter{Pool: pool}
}

func (w *PostgresWriter) Append(ctx context.Context, event Event) error {
	if w == nil || w.Pool == nil {
		return fmt.Errorf("audit writer is not configured")
	}
	if event.Action == "" {
		return fmt.Errorf("audit action is required")
	}
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		metadata = []byte(`{}`)
	}
	if len(metadata) == 0 {
		metadata = []byte(`{}`)
	}

	var actor any
	if event.ActorUserID > 0 {
		actor = event.ActorUserID
	}
	var target any
	if event.TargetUserID > 0 {
		target = event.TargetUserID
	}

	_, err = w.Pool.Exec(ctx, `
		INSERT INTO audit_events (actor_user_id, target_user_id, action, metadata)
		VALUES ($1, $2, $3, $4::jsonb)
	`, actor, target, event.Action, string(metadata))
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

// DeleteOlderThan 删除 created_at 早于 keepDays 的审计行。keepDays<=0 时使用推荐值。
func (w *PostgresWriter) DeleteOlderThan(ctx context.Context, keepDays int) (int64, error) {
	if w == nil || w.Pool == nil {
		return 0, fmt.Errorf("audit writer is not configured")
	}
	if keepDays <= 0 {
		keepDays = RecommendedRetentionDays
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -keepDays)
	tag, err := w.Pool.Exec(ctx, `DELETE FROM audit_events WHERE created_at < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old audit events: %w", err)
	}
	return tag.RowsAffected(), nil
}

// NoopWriter 测试与未装配场景。
type NoopWriter struct{}

func (NoopWriter) Append(context.Context, Event) error { return nil }

// Ensure 返回可用 Writer。
func Ensure(w Writer) Writer {
	if w == nil {
		return NoopWriter{}
	}
	return w
}
