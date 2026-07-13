package moderation

import (
	"errors"
	"time"
)

const (
	TargetTypeTopic   = "topic"
	TargetTypeComment = "comment"

	ReasonSpam     = "spam"
	ReasonAbuse    = "abuse"
	ReasonIllegal  = "illegal"
	ReasonOffTopic = "off_topic"
	ReasonOther    = "other"

	StatusOpen      = "open"
	StatusReviewing = "reviewing"
	StatusResolved  = "resolved"
	StatusRejected  = "rejected"

	CodeReportNotFound      = "moderation.report_not_found"
	CodeReportInvalid       = "moderation.report_invalid"
	CodeReportDuplicate     = "moderation.report_duplicate"
	CodeReportTargetInvalid = "moderation.report_target_invalid"
)

var (
	ErrReportNotFound      = errors.New("moderation: report not found")
	ErrReportInvalid       = errors.New("moderation: invalid report input")
	ErrReportDuplicate     = errors.New("moderation: duplicate open report")
	ErrReportTargetInvalid = errors.New("moderation: reported target not found or not reportable")
)

// Report 是 moderation_reports 行的领域结构。
type Report struct {
	ID             int64      `json:"id"`
	ReporterUserID int64      `json:"reporterUserId"`
	ReporterName   string     `json:"reporterName,omitempty"`
	TargetType     string     `json:"targetType"`
	TargetID       int64      `json:"targetId"`
	ReasonCode     string     `json:"reasonCode"`
	Body           string     `json:"body"`
	Status         string     `json:"status"`
	ReviewerUserID *int64     `json:"reviewerUserId,omitempty"`
	ReviewerName   string     `json:"reviewerName,omitempty"`
	ReviewNote     string     `json:"reviewNote"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	ResolvedAt     *time.Time `json:"resolvedAt,omitempty"`
}

// CreateReportInput 是举报创建入参。
type CreateReportInput struct {
	ReporterUserID int64
	TargetType     string
	TargetID       int64
	ReasonCode     string
	Body           string
}

// ReportListInput 是审核队列列表查询入参。
type ReportListInput struct {
	Status     string
	TargetType string
	ReporterID int64
	Page       int
	PerPage    int
}

// ReportList 是分页举报列表。
type ReportList struct {
	Items   []Report `json:"items"`
	Total   int64    `json:"total"`
	Page    int      `json:"page"`
	PerPage int      `json:"perPage"`
}

// UpdateReportInput 是审核更新入参。
type UpdateReportInput struct {
	ReportID       int64
	ReviewerUserID int64
	Status         string
	ReviewNote     string
}
