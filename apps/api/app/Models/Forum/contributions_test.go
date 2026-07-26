package forum

import (
	"errors"
	"testing"
	"time"
)

func TestPublicContributionEventStripsStaffOnlyFields(t *testing.T) {
	t.Parallel()
	from := int64(3)
	summary := ForumRevisionSummary{
		ID:                     10,
		RevisionNo:             2,
		Current:                true,
		Actor:                  &UserSummary{ID: 7, Username: "mod", DisplayName: "版主"},
		Operation:              "edit",
		Origin:                 "staff",
		Reason:                 "cross-author cleanup",
		ChangedFields:          []string{"title", "content"},
		CommittedAt:            time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC),
		RestoredFromRevisionNo: &from,
		SnapshotComplete:       true,
		RestorableFields:       []string{"title", "content"},
		Redacted:               false,
	}
	event := publicContributionEvent(summary)
	if event.RevisionNo != 2 || !event.Current || event.Operation != "edit" || event.Origin != "staff" {
		t.Fatalf("unexpected public event core fields: %+v", event)
	}
	if event.Actor == nil || event.Actor.ID != 7 {
		t.Fatalf("expected staff actor fully exposed, got %+v", event.Actor)
	}
	if event.RestoredFromRevisionNo == nil || *event.RestoredFromRevisionNo != 3 {
		t.Fatalf("expected restore pointer, got %+v", event.RestoredFromRevisionNo)
	}
	// 公开事件不得携带 reason；结构体也没有 restorableFields 字段。
	if len(event.ChangedFields) != 2 {
		t.Fatalf("changed fields = %v", event.ChangedFields)
	}
}

func TestEnsureAuthorInContributorsPrependsMissingAuthor(t *testing.T) {
	t.Parallel()
	author := UserSummary{ID: 1, Username: "alice", DisplayName: "Alice"}
	editor := UserSummary{ID: 2, Username: "bob", DisplayName: "Bob"}
	topic := &TopicDetail{
		TopicSummary: TopicSummary{AuthorUserID: 1, Author: &author},
	}
	items, count := ensureAuthorInContributors(topic, []UserSummary{editor}, 1)
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	if len(items) != 2 || items[0].ID != 1 || items[1].ID != 2 {
		t.Fatalf("items = %+v", items)
	}
}

func TestEnsureAuthorInContributorsKeepsAuthorFirstWhenPresent(t *testing.T) {
	t.Parallel()
	author := UserSummary{ID: 1, Username: "alice", DisplayName: "Alice"}
	editor := UserSummary{ID: 2, Username: "bob", DisplayName: "Bob"}
	topic := &TopicDetail{
		TopicSummary: TopicSummary{AuthorUserID: 1, Author: &author},
	}
	items, count := ensureAuthorInContributors(topic, []UserSummary{author, editor}, 2)
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	if len(items) != 2 || items[0].ID != 1 {
		t.Fatalf("items = %+v", items)
	}
}

func TestListTopicContributionTimelineRejectsInvalidTopicID(t *testing.T) {
	t.Parallel()
	service := &Service{store: &serviceFakeStore{}}
	if _, err := service.ListTopicContributionTimeline(t.Context(), 0, RevisionListInput{}); !errors.Is(err, ErrTopicNotFound) {
		t.Fatalf("err = %v, want ErrTopicNotFound", err)
	}
}
