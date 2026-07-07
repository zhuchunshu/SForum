package profile

import (
	"context"
	"errors"
	"testing"

	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

func TestServiceGetPublicProfileAggregatesData(t *testing.T) {
	store := &fakeStore{
		user:   UserProfileSummary{UserID: 7, Username: "alice", DisplayName: "Alice"},
		profile: Profile{UserID: 7, Bio: "hello"},
		stats:   ProfileStats{TopicCount: 3, CommentCount: 12},
		recent:  []forum.TopicSummary{{ID: 1, Title: "t1"}},
	}
	service := NewService(store)

	result, err := service.GetPublicProfile(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetPublicProfile error: %v", err)
	}
	if result.UserID != 7 || result.Username != "alice" || result.DisplayName != "Alice" {
		t.Fatalf("unexpected user summary: %#v", result)
	}
	if result.Profile.Bio != "hello" {
		t.Fatalf("unexpected bio: %q", result.Profile.Bio)
	}
	if result.TopicCount != 3 || result.CommentCount != 12 {
		t.Fatalf("unexpected stats: %#v", result)
	}
	if len(result.RecentTopics) != 1 || result.RecentTopics[0].Title != "t1" {
		t.Fatalf("unexpected recent topics: %#v", result.RecentTopics)
	}
}

func TestServiceGetPublicProfileRejectsEmptyUsername(t *testing.T) {
	service := NewService(&fakeStore{})
	_, err := service.GetPublicProfile(context.Background(), "   ")
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("expected ErrProfileNotFound, got %v", err)
	}
}

func TestServiceUpdateMyProfileRequiresActor(t *testing.T) {
	service := NewService(&fakeStore{
		user:    UserProfileSummary{UserID: 7, Username: "alice", DisplayName: "Alice"},
		profile: Profile{UserID: 7},
	})
	_, err := service.UpdateMyProfile(context.Background(), identity.Actor{ID: 0}, UpdateProfileInput{})
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied for no actor, got %v", err)
	}
}

func TestServiceUpdateMyProfileTrimsAndPersists(t *testing.T) {
	store := &fakeStore{
		user:    UserProfileSummary{UserID: 7, Username: "alice", DisplayName: "Alice"},
		profile: Profile{UserID: 7, Bio: "old"},
	}
	service := NewService(store)
	bio := "  new bio  "
	website := "https://example.com"
	updated, err := service.UpdateMyProfile(context.Background(), identity.Actor{ID: 7}, UpdateProfileInput{
		Bio:        &bio,
		WebsiteURL: &website,
	})
	if err != nil {
		t.Fatalf("UpdateMyProfile error: %v", err)
	}
	if updated.Bio != "new bio" {
		t.Fatalf("expected trimmed bio, got %q", updated.Bio)
	}
	if updated.WebsiteURL != "https://example.com" {
		t.Fatalf("expected website url, got %q", updated.WebsiteURL)
	}
}

func TestServiceUpdateMyProfileRejectsInvalidWebsite(t *testing.T) {
	service := NewService(&fakeStore{
		user:    UserProfileSummary{UserID: 7, Username: "alice"},
		profile: Profile{UserID: 7},
	})
	bad := "not-a-url"
	_, err := service.UpdateMyProfile(context.Background(), identity.Actor{ID: 7}, UpdateProfileInput{WebsiteURL: &bad})
	if !errors.Is(err, ErrProfileInvalid) {
		t.Fatalf("expected ErrProfileInvalid, got %v", err)
	}
}

func TestServiceUpdateMyProfileCannotTargetOtherUser(t *testing.T) {
	store := &fakeStore{
		user:    UserProfileSummary{UserID: 7, Username: "alice"},
		profile: Profile{UserID: 7},
	}
	service := NewService(store)
	// actor.ID 决定写入目标；传 actor 9 但 store 只有用户 7 的资料。
	bio := "hijack"
	_, err := service.UpdateMyProfile(context.Background(), identity.Actor{ID: 9}, UpdateProfileInput{Bio: &bio})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// upsert 入参的 UserID 应为 actor.ID（9），证明只能写自己的资料。
	if store.upserted.UserID != 9 {
		t.Fatalf("expected upsert target actor id 9, got %d", store.upserted.UserID)
	}
}

type fakeStore struct {
	user     UserProfileSummary
	profile  Profile
	stats    ProfileStats
	recent   []forum.TopicSummary
	upserted Profile
	err      error
}

func (s *fakeStore) GetProfile(context.Context, int64) (Profile, error) {
	return s.profile, nil
}

func (s *fakeStore) UpsertProfile(_ context.Context, input Profile) (Profile, error) {
	s.upserted = input
	return input, nil
}

func (s *fakeStore) GetUserSummaryByUsername(context.Context, string) (UserProfileSummary, error) {
	if s.err != nil {
		return UserProfileSummary{}, s.err
	}
	return s.user, nil
}

func (s *fakeStore) GetUserSummaryByID(context.Context, int64) (UserProfileSummary, error) {
	return s.user, nil
}

func (s *fakeStore) GetProfileStats(context.Context, int64) (ProfileStats, error) {
	return s.stats, nil
}

func (s *fakeStore) ListRecentTopics(context.Context, int64, int) ([]forum.TopicSummary, error) {
	return s.recent, nil
}
