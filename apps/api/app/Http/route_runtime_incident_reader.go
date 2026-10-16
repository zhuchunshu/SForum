package http

import (
	"context"
	"fmt"
)

// RouteRuntimeIncidentReadMaximum 是管理员诊断读取的上限：事故表按路由与时间索引
// 增长，读取必须显式有界，避免一次请求把整张表拉进内存。
const RouteRuntimeIncidentReadMaximum = 200

// RouteRuntimeIncidentReader 是事故表的只读边界（M5：崩溃循环隔离必须在重启后仍可
// 被管理员看见）。它与写入接口分开，避免 recorder 的测试替身被迫实现读取。
type RouteRuntimeIncidentReader interface {
	ListRecentRouteRuntimeIncidents(ctx context.Context, limit int) ([]RouteRuntimeIncidentRecord, error)
}

const routeRuntimeIncidentSelectRecent = `
	SELECT ` + routeRuntimeIncidentColumns + `
	FROM extension_route_runtime_incidents
	ORDER BY created_at DESC, id DESC
	LIMIT $1
`

// ListRecentRouteRuntimeIncidents 返回最近的事故记录（新的在前）。
// 记录只包含已披露的失败语义，不含请求头、正文或任何凭据。
func (s *PostgresRouteRuntimeIncidentStore) ListRecentRouteRuntimeIncidents(
	ctx context.Context,
	limit int,
) ([]RouteRuntimeIncidentRecord, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return nil, ErrRouteRuntimeIncidentInvalid
	}
	if limit <= 0 || limit > RouteRuntimeIncidentReadMaximum {
		limit = RouteRuntimeIncidentReadMaximum
	}
	rows, err := s.pool.Query(ctx, routeRuntimeIncidentSelectRecent, limit)
	if err != nil {
		return nil, fmt.Errorf("list route runtime incidents: %w", err)
	}
	defer rows.Close()
	records := make([]RouteRuntimeIncidentRecord, 0, limit)
	for rows.Next() {
		record, scanErr := scanRouteRuntimeIncident(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list route runtime incidents: %w", err)
	}
	return records, nil
}

// RouteRuntimeIncidentView 是给管理端的脱敏投影：事件 id、失败码与隔离结果足以让
// 操作员定位事故，不需要暴露完整证据结构。
type RouteRuntimeIncidentView struct {
	ID                int64  `json:"id"`
	IncidentKey       string `json:"incidentKey"`
	RouteID           string `json:"routeId"`
	ContractVersion   string `json:"contractVersion,omitempty"`
	Method            string `json:"method"`
	PathSignature     string `json:"pathSignature"`
	Phase             string `json:"phase"`
	Action            string `json:"action"`
	FailureCode       string `json:"failureCode"`
	CauseClass        string `json:"causeClass"`
	CommitState       string `json:"commitState"`
	ExtensionID       string `json:"extensionId"`
	ExtensionVersion  string `json:"extensionVersion"`
	PackageDigest     string `json:"packageDigest"`
	RuntimeInstanceID string `json:"runtimeInstanceId"`
	LocalResult       string `json:"localResult"`
	Quarantined       bool   `json:"quarantined"`
	Resolved          bool   `json:"resolved"`
	AuditEventID      int64  `json:"auditEventId"`
	CreatedAt         string `json:"createdAt"`
	ResolvedAt        string `json:"resolvedAt,omitempty"`
}

// RouteRuntimeIncidentViews 把持久记录投影成管理端视图，保持输入顺序。
func RouteRuntimeIncidentViews(records []RouteRuntimeIncidentRecord) []RouteRuntimeIncidentView {
	views := make([]RouteRuntimeIncidentView, 0, len(records))
	for _, record := range records {
		view := RouteRuntimeIncidentView{
			ID: record.ID, IncidentKey: record.Evidence.IncidentKey,
			RouteID: record.Evidence.RouteID, ContractVersion: record.Evidence.ContractVersion,
			Method: record.Evidence.Method, PathSignature: record.Evidence.PathSignature,
			Phase: string(record.Evidence.Phase), Action: record.Evidence.Action,
			FailureCode: string(record.Evidence.FailureCode), CauseClass: record.Evidence.CauseClass,
			CommitState:       string(record.Evidence.CommitState),
			ExtensionID:       record.Evidence.Artifact.ExtensionID,
			ExtensionVersion:  record.Evidence.Artifact.ExtensionVersion,
			PackageDigest:     record.Evidence.Artifact.PackageDigest,
			RuntimeInstanceID: record.Evidence.Artifact.RuntimeInstanceID,
			LocalResult:       string(record.LocalResult),
			Quarantined:       record.LocalResult == RouteRuntimeIncidentQuarantined,
			Resolved:          record.ResolvedAt != nil,
			AuditEventID:      record.AuditEventID,
			CreatedAt:         record.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		}
		if record.ResolvedAt != nil {
			view.ResolvedAt = record.ResolvedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		}
		views = append(views, view)
	}
	return views
}

// RouteRuntimeIncidentSummary 是页面顶部的聚合：待处理隔离、已隔离、已恢复。
type RouteRuntimeIncidentSummary struct {
	Pending     int `json:"pending"`
	Quarantined int `json:"quarantined"`
	Recovered   int `json:"recovered"`
	Failed      int `json:"failed"`
}

// SummarizeRouteRuntimeIncidents 暴露给控制器层复用。
func SummarizeRouteRuntimeIncidents(records []RouteRuntimeIncidentRecord) RouteRuntimeIncidentSummary {
	var summary RouteRuntimeIncidentSummary
	for _, record := range records {
		// local_quarantine_result 是**隔离决定**，resolved_at 只是这一行账目的收尾：
		// 一条 quarantined 行表示该精确运行时表面已被关闭，无论账目是否收尾。
		switch record.LocalResult {
		case RouteRuntimeIncidentQuarantined:
			summary.Quarantined++
		case RouteRuntimeIncidentPending:
			summary.Pending++
		case RouteRuntimeIncidentFailed:
			summary.Failed++
		default:
			// stale_missing / stale_artifact：制品已不在，无需隔离即可恢复。
			summary.Recovered++
		}
	}
	return summary
}
