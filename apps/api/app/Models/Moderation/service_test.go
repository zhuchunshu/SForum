package moderation

import (
	"context"
	"errors"
	"testing"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

func TestCreateReportAllowsActiveUser(t *testing.T) {
	store := &fakeStore{}
	validator := &fakeValidator{topic: true}
	service := NewService(store, validator)
	actor := identity.Actor{ID: 5, Status: identity.UserStatusActive}

	report, err := service.CreateReport(context.Background(), actor, CreateReportInput{
		TargetType: TargetTypeTopic,
		TargetID:   10,
		ReasonCode: ReasonSpam,
		Body:       "spam content",
	})
	if err != nil {
		t.Fatalf("expected create success, got %v", err)
	}
	if report.TargetType != TargetTypeTopic || report.TargetID != 10 {
		t.Fatalf("unexpected report: %#v", report)
	}
	if store.createdInput.ReporterUserID != 5 {
		t.Fatalf("expected reporter 5, got %d", store.createdInput.ReporterUserID)
	}
}

func TestCreateReportRejectsUnauthenticated(t *testing.T) {
	service := NewService(&fakeStore{}, &fakeValidator{topic: true})
	_, err := service.CreateReport(context.Background(), identity.Actor{ID: 0}, CreateReportInput{
		TargetType: TargetTypeTopic, TargetID: 1, ReasonCode: ReasonSpam,
	})
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestCreateReportRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name  string
		input CreateReportInput
	}{
		{name: "bad target type", input: CreateReportInput{TargetType: "user", TargetID: 1, ReasonCode: ReasonSpam}},
		{name: "bad reason", input: CreateReportInput{TargetType: TargetTypeTopic, TargetID: 1, ReasonCode: "bogus"}},
		{name: "zero target id", input: CreateReportInput{TargetType: TargetTypeTopic, TargetID: 0, ReasonCode: ReasonSpam}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewService(&fakeStore{}, &fakeValidator{topic: true})
			_, err := service.CreateReport(context.Background(), identity.Actor{ID: 5, Status: identity.UserStatusActive}, tc.input)
			if !errors.Is(err, ErrReportInvalid) {
				t.Fatalf("expected ErrReportInvalid, got %v", err)
			}
		})
	}
}

func TestCreateReportRejectsUnreportableTarget(t *testing.T) {
	service := NewService(&fakeStore{}, &fakeValidator{topic: false})
	_, err := service.CreateReport(context.Background(), identity.Actor{ID: 5, Status: identity.UserStatusActive}, CreateReportInput{
		TargetType: TargetTypeTopic, TargetID: 99, ReasonCode: ReasonSpam,
	})
	if !errors.Is(err, ErrReportTargetInvalid) {
		t.Fatalf("expected ErrReportTargetInvalid, got %v", err)
	}
}

func TestCreateReportPropagatesDuplicateError(t *testing.T) {
	store := &fakeStore{createErr: ErrReportDuplicate}
	service := NewService(store, &fakeValidator{topic: true})
	_, err := service.CreateReport(context.Background(), identity.Actor{ID: 5, Status: identity.UserStatusActive}, CreateReportInput{
		TargetType: TargetTypeTopic, TargetID: 10, ReasonCode: ReasonSpam,
	})
	if !errors.Is(err, ErrReportDuplicate) {
		t.Fatalf("expected ErrReportDuplicate, got %v", err)
	}
}

func TestListReportsRequiresReviewPermission(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, nil)
	_, err := service.ListReports(context.Background(), identity.Actor{ID: 5, Status: identity.UserStatusActive, Permissions: map[string]bool{}}, ReportListInput{})
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}

	reviewer := identity.Actor{ID: 9, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionModerationReportReview: true}}
	list, err := service.ListReports(context.Background(), reviewer, ReportListInput{Page: 0, PerPage: 0})
	if err != nil {
		t.Fatalf("expected list success, got %v", err)
	}
	if list.Page != 1 || list.PerPage != 20 {
		t.Fatalf("expected normalized pagination, got page=%d perPage=%d", list.Page, list.PerPage)
	}
}

func TestUpdateReportRequiresReviewPermission(t *testing.T) {
	service := NewService(&fakeStore{}, nil)
	_, err := service.UpdateReport(context.Background(), identity.Actor{ID: 5, Status: identity.UserStatusActive, Permissions: map[string]bool{}}, UpdateReportInput{ReportID: 1, Status: StatusResolved})
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestUpdateReportRejectsInvalidStatus(t *testing.T) {
	service := NewService(&fakeStore{}, nil)
	reviewer := identity.Actor{ID: 9, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionModerationReportReview: true}}
	_, err := service.UpdateReport(context.Background(), reviewer, UpdateReportInput{ReportID: 1, Status: "bogus"})
	if !errors.Is(err, ErrReportInvalid) {
		t.Fatalf("expected ErrReportInvalid, got %v", err)
	}
}

type fakeStore struct {
	createdInput CreateReportInput
	createErr    error
	listResult   ReportList
}

func (s *fakeStore) CreateReport(_ context.Context, input CreateReportInput) (Report, error) {
	if s.createErr != nil {
		return Report{}, s.createErr
	}
	s.createdInput = input
	return Report{ID: 1, TargetType: input.TargetType, TargetID: input.TargetID, ReasonCode: input.ReasonCode, Status: StatusOpen}, nil
}

func (s *fakeStore) ListReports(_ context.Context, input ReportListInput) (ReportList, error) {
	return ReportList{Items: []Report{}, Total: 0, Page: input.Page, PerPage: input.PerPage}, nil
}

func (s *fakeStore) GetReport(context.Context, int64) (Report, error) {
	return Report{}, nil
}

func (s *fakeStore) UpdateReport(_ context.Context, input UpdateReportInput) (Report, error) {
	return Report{ID: input.ReportID, Status: input.Status, ReviewNote: input.ReviewNote}, nil
}

type fakeValidator struct {
	topic   bool
	comment bool
}

func (v *fakeValidator) IsReportableTopic(context.Context, int64) (bool, error) {
	return v.topic, nil
}

func (v *fakeValidator) IsReportableComment(context.Context, int64) (bool, error) {
	return v.comment, nil
}
