package health

import (
	"context"
	"time"
)

// Component 状态：ok / degraded / error / skipped。
// ready 端点对 postgres=error 返回 503；redis error 记为 degraded 仍 200（F1 默认策略）。
// Meilisearch 已拆为可选 search.provider 插件，不再作为 core readiness 组件。
const (
	StatusOK       = "ok"
	StatusDegraded = "degraded"
	StatusError    = "error"
	StatusSkipped  = "skipped"
)

// Checker 探测单一依赖。error 表示检查失败（组件 status=error）。
type Checker interface {
	Name() string
	// Required 为 true 时，失败会使整体 ready=false。
	Required() bool
	Check(ctx context.Context) error
}

// ComponentResult 是单个依赖的探测结果。
type ComponentResult struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Required bool   `json:"required"`
	Error    string `json:"error,omitempty"`
	Latency  string `json:"latency,omitempty"`
}

// ReadyReport 是 /ready 响应体。
type ReadyReport struct {
	Status     string               `json:"status"` // ready | not_ready
	Ready      bool                 `json:"ready"`
	CheckedAt  time.Time            `json:"checkedAt"`
	Components []ComponentResult    `json:"components"`
	Recovery   *RecoveryRequirement `json:"recovery,omitempty"`
}

// Evaluate 运行全部 checker 并汇总。
// 策略：任一 required 失败 → not_ready；仅非 required 失败 → ready 但 status 可标 degraded（仍 ready=true）。
func Evaluate(ctx context.Context, checkers []Checker) ReadyReport {
	now := time.Now().UTC()
	report := ReadyReport{
		Status:     "ready",
		Ready:      true,
		CheckedAt:  now,
		Components: make([]ComponentResult, 0, len(checkers)),
	}
	if len(checkers) == 0 {
		return report
	}

	for _, checker := range checkers {
		if checker == nil {
			continue
		}
		result := ComponentResult{
			Name:     checker.Name(),
			Required: checker.Required(),
			Status:   StatusOK,
		}
		start := time.Now()
		err := checker.Check(ctx)
		result.Latency = time.Since(start).Round(time.Millisecond).String()
		if err != nil {
			result.Status = StatusError
			result.Error = err.Error()
			if checker.Required() {
				report.Ready = false
				report.Status = "not_ready"
			}
		}
		report.Components = append(report.Components, result)
	}

	// 非 required 失败时仍 ready=true，但对外 status 用 degraded 便于运维扫一眼。
	if report.Ready {
		for _, c := range report.Components {
			if c.Status == StatusError {
				report.Status = StatusDegraded
				break
			}
		}
	}
	return report
}
