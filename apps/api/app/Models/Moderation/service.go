package moderation

import (
	"context"
	"strings"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

type Service struct {
	store          Store
	targetValidator TargetValidator
}

func NewService(store Store, validator TargetValidator) *Service {
	return &Service{store: store, targetValidator: validator}
}

// CreateReport 由登录活跃用户发起举报。
func (s *Service) CreateReport(ctx context.Context, actor identity.Actor, input CreateReportInput) (Report, error) {
	if actor.ID <= 0 || actor.Status != identity.UserStatusActive {
		return Report{}, identity.ErrPermissionDenied
	}
	input.ReporterUserID = actor.ID
	if err := validateCreateReportInput(input); err != nil {
		return Report{}, err
	}
	// 校验被举报目标存在且可公开举报。
	if s.targetValidator != nil {
		reportable, err := s.isTargetReportable(ctx, input.TargetType, input.TargetID)
		if err != nil {
			return Report{}, err
		}
		if !reportable {
			return Report{}, ErrReportTargetInvalid
		}
	}
	report, err := s.store.CreateReport(ctx, input)
	if err != nil {
		return Report{}, err
	}
	return report, nil
}

// ListReports 供审核员查看举报队列。
func (s *Service) ListReports(ctx context.Context, actor identity.Actor, input ReportListInput) (ReportList, error) {
	if !actor.Can(identity.PermissionModerationReportReview) {
		return ReportList{}, identity.ErrPermissionDenied
	}
	input.Page, input.PerPage = normalizePage(input.Page, input.PerPage)
	return s.store.ListReports(ctx, input)
}

// UpdateReport 更新举报状态与审核备注。
func (s *Service) UpdateReport(ctx context.Context, actor identity.Actor, input UpdateReportInput) (Report, error) {
	if !actor.Can(identity.PermissionModerationReportReview) {
		return Report{}, identity.ErrPermissionDenied
	}
	if err := validateUpdateReportInput(input); err != nil {
		return Report{}, err
	}
	input.ReviewerUserID = actor.ID
	return s.store.UpdateReport(ctx, input)
}

func (s *Service) isTargetReportable(ctx context.Context, targetType string, targetID int64) (bool, error) {
	switch targetType {
	case TargetTypeTopic:
		return s.targetValidator.IsReportableTopic(ctx, targetID)
	case TargetTypeComment:
		return s.targetValidator.IsReportableComment(ctx, targetID)
	default:
		return false, nil
	}
}

func validateCreateReportInput(input CreateReportInput) error {
	if input.TargetID <= 0 {
		return ErrReportInvalid
	}
	if !isValidTargetType(input.TargetType) {
		return ErrReportInvalid
	}
	if !isValidReasonCode(input.ReasonCode) {
		return ErrReportInvalid
	}
	input.Body = strings.TrimSpace(input.Body)
	if len(input.Body) > 2000 {
		return ErrReportInvalid
	}
	return nil
}

func validateUpdateReportInput(input UpdateReportInput) error {
	if input.ReportID <= 0 {
		return ErrReportInvalid
	}
	if !isValidStatus(input.Status) {
		return ErrReportInvalid
	}
	input.ReviewNote = strings.TrimSpace(input.ReviewNote)
	if len(input.ReviewNote) > 2000 {
		return ErrReportInvalid
	}
	return nil
}

func isValidTargetType(value string) bool {
	return value == TargetTypeTopic || value == TargetTypeComment
}

func isValidReasonCode(value string) bool {
	switch value {
	case ReasonSpam, ReasonAbuse, ReasonIllegal, ReasonOffTopic, ReasonOther:
		return true
	default:
		return false
	}
}

func isValidStatus(value string) bool {
	switch value {
	case StatusOpen, StatusReviewing, StatusResolved, StatusRejected:
		return true
	default:
		return false
	}
}

func normalizePage(page, perPage int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}
