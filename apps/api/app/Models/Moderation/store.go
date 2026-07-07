package moderation

import "context"

type Store interface {
	CreateReport(ctx context.Context, input CreateReportInput) (Report, error)
	ListReports(ctx context.Context, input ReportListInput) (ReportList, error)
	GetReport(ctx context.Context, reportID int64) (Report, error)
	UpdateReport(ctx context.Context, input UpdateReportInput) (Report, error)
}

// TargetValidator 校验被举报的 topic/comment 是否存在且可被公开举报。
type TargetValidator interface {
	IsReportableTopic(ctx context.Context, topicID int64) (bool, error)
	IsReportableComment(ctx context.Context, commentID int64) (bool, error)
}
