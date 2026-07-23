package moderation

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	SourcePrePublish = "pre_publish"
	SourceReport     = "report"

	ActionApprove        = "approve"
	ActionReject         = "reject"
	ActionKeepAndClose   = "keep_and_close"
	ActionHideAndClose   = "hide_and_close"
	ActionDeleteAndClose = "delete_and_close"
)

var (
	ErrDecisionInvalid = errors.New("moderation: invalid decision")
	ErrTaskNotFound    = errors.New("moderation: task not found")
	ErrTaskConflict    = errors.New("moderation: task conflict")
)

type QueueCounts struct {
	PendingContent int64 `json:"pendingContent"`
	OpenReports    int64 `json:"openReports"`
	// ProcessedToday 是今日 KPI，仅右栏概览使用；历史 tab 徽章应使用 HistoryTotal。
	ProcessedToday int64 `json:"processedToday"`
	// HistoryTotal 是处理记录全量条数，与 workbench history 列表 total 对齐。
	HistoryTotal int64 `json:"historyTotal"`
}

type WorkbenchListInput struct {
	TargetType string
	Page       int
	PerPage    int
}

type PendingItem struct {
	TargetType string    `json:"targetType"`
	TargetID   int64     `json:"targetId"`
	TopicID    int64     `json:"topicId,omitempty"`
	Title      string    `json:"title"`
	Excerpt    string    `json:"excerpt"`
	AuthorID   int64     `json:"authorId"`
	AuthorName string    `json:"authorName"`
	Category   string    `json:"category"`
	Triggers   []string  `json:"triggers"`
	CreatedAt  time.Time `json:"createdAt"`
	// IPAddress / LastEditIP 仅当调用方持有 moderation.view_ip 时由 service 填充。
	IPAddress  string `json:"ipAddress,omitempty"`
	LastEditIP string `json:"lastEditIp,omitempty"`
}

type PendingList struct {
	Items   []PendingItem `json:"items"`
	Total   int64         `json:"total"`
	Page    int           `json:"page"`
	PerPage int           `json:"perPage"`
}

type ReportItem struct {
	Report
	Title            string `json:"title"`
	Excerpt          string `json:"excerpt"`
	TargetAuthorID   int64  `json:"targetAuthorId"`
	TargetAuthorName string `json:"targetAuthorName"`
	Category         string `json:"category"`
	TargetStatus     string `json:"targetStatus"`
	TargetTopicID    int64  `json:"targetTopicId,omitempty"`
	// IPAddress / LastEditIP 仅当调用方持有 moderation.view_ip 时由 service 填充。
	IPAddress  string `json:"ipAddress,omitempty"`
	LastEditIP string `json:"lastEditIp,omitempty"`
}

type ReportItemList struct {
	Items   []ReportItem `json:"items"`
	Total   int64        `json:"total"`
	Page    int          `json:"page"`
	PerPage int          `json:"perPage"`
}

type ReviewContextInput struct {
	Source     string
	TargetType string
	TargetID   int64
	ReportID   int64
}

type ContextComment struct {
	ID         int64  `json:"id"`
	AuthorName string `json:"authorName"`
	HTML       string `json:"html"`
}

type ReviewContext struct {
	Source      string           `json:"source"`
	TargetType  string           `json:"targetType"`
	TargetID    int64            `json:"targetId"`
	TopicID     int64            `json:"topicId"`
	ReportID    int64            `json:"reportId,omitempty"`
	Title       string           `json:"title"`
	HTML        string           `json:"html"`
	AuthorID    int64            `json:"authorId"`
	AuthorName  string           `json:"authorName"`
	Category    string           `json:"category"`
	Status      string           `json:"status"`
	Triggers    []string         `json:"triggers"`
	ParentTopic string           `json:"parentTopic,omitempty"`
	Nearby      []ContextComment `json:"nearby,omitempty"`
	CreatedAt   time.Time        `json:"createdAt"`
	// IPAddress 创建时真实客户端 IP；LastEditIP 最近一次编辑 IP。
	// 仅当调用方持有 moderation.view_ip 时由 service 保留，否则清空。
	IPAddress  string `json:"ipAddress,omitempty"`
	LastEditIP string `json:"lastEditIp,omitempty"`
}

type DecisionInput struct {
	Source         string `json:"source"`
	TargetType     string `json:"targetType"`
	TargetID       int64  `json:"targetId"`
	ReportID       int64  `json:"reportId,omitempty"`
	Action         string `json:"action"`
	ReviewNote     string `json:"reviewNote"`
	ReviewerUserID int64  `json:"-"`
}

type Decision struct {
	ID             int64     `json:"id"`
	Source         string    `json:"source"`
	TargetType     string    `json:"targetType"`
	TargetID       int64     `json:"targetId"`
	ReportID       *int64    `json:"reportId,omitempty"`
	Action         string    `json:"action"`
	ReviewerUserID int64     `json:"reviewerUserId"`
	ReviewerName   string    `json:"reviewerName"`
	ReviewNote     string    `json:"reviewNote"`
	Triggers       []string  `json:"triggers"`
	CreatedAt      time.Time `json:"createdAt"`
}

type DecisionListInput struct {
	Action     string
	TargetType string
	ReviewerID int64
	Page       int
	PerPage    int
}

type DecisionList struct {
	Items   []Decision `json:"items"`
	Total   int64      `json:"total"`
	Page    int        `json:"page"`
	PerPage int        `json:"perPage"`
}

type WorkbenchStore interface {
	QueueCounts(ctx context.Context) (QueueCounts, error)
	ListPending(ctx context.Context, input WorkbenchListInput) (PendingList, error)
	ListReportItems(ctx context.Context, input WorkbenchListInput) (ReportItemList, error)
	ListDecisions(ctx context.Context, input DecisionListInput) (DecisionList, error)
	GetReviewContext(ctx context.Context, input ReviewContextInput) (ReviewContext, error)
	SubmitDecision(ctx context.Context, input DecisionInput) (Decision, error)
}

// ValidateDecision validates a moderation action without reading or writing
// state. Host Commands call it while planning, then SubmitDecisionTx performs
// the same validation again inside the authoritative transaction.
func ValidateDecision(input DecisionInput) error {
	return validateDecision(&input)
}

func validateDecision(input *DecisionInput) error {
	input.ReviewNote = strings.TrimSpace(input.ReviewNote)
	if input.TargetID <= 0 || !isValidTargetType(input.TargetType) || len(input.ReviewNote) > 2000 {
		return ErrDecisionInvalid
	}
	if input.Source == SourcePrePublish {
		if input.Action != ActionApprove && input.Action != ActionReject {
			return ErrDecisionInvalid
		}
	} else if input.Source == SourceReport {
		if input.ReportID <= 0 || (input.Action != ActionKeepAndClose && input.Action != ActionHideAndClose && input.Action != ActionDeleteAndClose) {
			return ErrDecisionInvalid
		}
	} else {
		return ErrDecisionInvalid
	}
	if (input.Action == ActionReject || input.Action == ActionHideAndClose || input.Action == ActionDeleteAndClose) && input.ReviewNote == "" {
		return ErrDecisionInvalid
	}
	return nil
}
