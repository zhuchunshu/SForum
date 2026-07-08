package profile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	attachments "github.com/zhuchunshu/sforum/apps/api/app/Models/Attachments"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
)

func TestServiceGetPublicProfileAggregatesData(t *testing.T) {
	store := &fakeStore{
		user:    UserProfileSummary{UserID: 7, Username: "alice", DisplayName: "Alice"},
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

func TestServiceDecoratesDefaultInitialsAvatar(t *testing.T) {
	store := &fakeStore{
		user:    UserProfileSummary{UserID: 7, Username: "alice", DisplayName: "Alice", Email: "alice@example.com"},
		profile: Profile{UserID: 7},
	}
	service := NewServiceWithAvatar(store, nil, newProfileAvatarOptions(nil))

	result, err := service.GetPublicProfile(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetPublicProfile error: %v", err)
	}
	if result.Profile.Avatar.Kind != AvatarKindInitials || result.Profile.Avatar.URL != "" {
		t.Fatalf("expected initials avatar fallback, got %#v", result.Profile.Avatar)
	}
	if result.Profile.Avatar.Alt != "Alice" {
		t.Fatalf("expected display name alt, got %q", result.Profile.Avatar.Alt)
	}
}

func TestServiceDecoratesGravatarAvatarWithConfiguredHash(t *testing.T) {
	store := &fakeStore{
		user:    UserProfileSummary{UserID: 7, Username: "alice", DisplayName: "Alice", Email: " Alice@Example.COM "},
		profile: Profile{UserID: 7},
	}
	service := NewServiceWithAvatar(store, nil, newProfileAvatarOptions(map[string]string{
		options.NameAvatarDefaultProvider:       options.AvatarProviderGravatar,
		options.NameAvatarGravatarBaseURL:       "https://avatar.example.com/avatar/",
		options.NameAvatarGravatarHashAlgorithm: options.AvatarHashSHA256,
	}))

	result, err := service.GetPublicProfile(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetPublicProfile error: %v", err)
	}
	sum := sha256.Sum256([]byte("alice@example.com"))
	expectedURL := "https://avatar.example.com/avatar/" + hex.EncodeToString(sum[:])
	if result.Profile.Avatar.Kind != AvatarKindGravatar || result.Profile.Avatar.URL != expectedURL {
		t.Fatalf("expected gravatar URL %q, got %#v", expectedURL, result.Profile.Avatar)
	}
}

func TestServiceUploadAvatarRequiresPermissionAndAvatarSwitch(t *testing.T) {
	store := &fakeStore{
		user:    UserProfileSummary{UserID: 7, Username: "alice", DisplayName: "Alice"},
		profile: Profile{UserID: 7},
	}
	uploader := &fakeAvatarUploader{attachment: attachments.Attachment{ID: 88, PublicID: "avatar-public"}}
	service := NewServiceWithAvatar(store, uploader, newProfileAvatarOptions(nil))

	_, err := service.UploadAvatar(context.Background(), identity.Actor{ID: 7, Status: identity.UserStatusActive}, attachments.UploadInput{
		OriginalName: "avatar.jpg",
		ContentType:  "image/jpeg",
		SizeBytes:    4,
		File:         newProfileReadSeekCloser("data"),
	})
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied without attachment.upload, got %v", err)
	}
	if uploader.called {
		t.Fatalf("uploader should not run when actor lacks permission")
	}

	disabled := NewServiceWithAvatar(store, uploader, newProfileAvatarOptions(map[string]string{
		options.NameAvatarAllowUpload: "disabled",
	}))
	_, err = disabled.UploadAvatar(context.Background(), uploadProfileActor(), attachments.UploadInput{
		OriginalName: "avatar.jpg",
		ContentType:  "image/jpeg",
		SizeBytes:    4,
		File:         newProfileReadSeekCloser("data"),
	})
	if !errors.Is(err, ErrAvatarUploadDisabled) {
		t.Fatalf("expected avatar upload disabled, got %v", err)
	}
}

func TestServiceUploadAvatarSetsProfileReference(t *testing.T) {
	avatarID := int64(88)
	store := &fakeStore{
		user:    UserProfileSummary{UserID: 7, Username: "alice", DisplayName: "Alice"},
		profile: Profile{UserID: 7},
		avatarAttachment: AvatarAttachment{
			ID:          avatarID,
			PublicID:    "avatar-public",
			OwnerUserID: 7,
			ContentType: "image/jpeg",
			Status:      "active",
			URL:         "/api/v1/attachments/avatar-public/content",
		},
	}
	uploader := &fakeAvatarUploader{attachment: attachments.Attachment{ID: avatarID, PublicID: "avatar-public"}}
	service := NewServiceWithAvatar(store, uploader, newProfileAvatarOptions(nil))

	updated, err := service.UploadAvatar(context.Background(), uploadProfileActor(), attachments.UploadInput{
		OriginalName: "avatar.jpg",
		ContentType:  "image/jpeg",
		SizeBytes:    4,
		File:         newProfileReadSeekCloser("data"),
	})
	if err != nil {
		t.Fatalf("UploadAvatar returned error: %v", err)
	}
	if store.setAvatarAttachmentID == nil || *store.setAvatarAttachmentID != avatarID || store.setAvatarActorID != 7 {
		t.Fatalf("expected avatar reference for attachment 88 by actor 7, got id=%v actor=%d", store.setAvatarAttachmentID, store.setAvatarActorID)
	}
	if updated.Avatar.Kind != AvatarKindUploaded || updated.Avatar.AttachmentID == nil || *updated.Avatar.AttachmentID != avatarID {
		t.Fatalf("expected uploaded avatar view, got %#v", updated.Avatar)
	}
}

func uploadProfileActor() identity.Actor {
	return identity.Actor{
		ID:          7,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionAttachmentUpload: true},
	}
}

type fakeStore struct {
	user                  UserProfileSummary
	profile               Profile
	stats                 ProfileStats
	recent                []forum.TopicSummary
	upserted              Profile
	avatarAttachment      AvatarAttachment
	setAvatarAttachmentID *int64
	setAvatarActorID      int64
	err                   error
}

func (s *fakeStore) GetProfile(context.Context, int64) (Profile, error) {
	return s.profile, nil
}

func (s *fakeStore) UpsertProfile(_ context.Context, input Profile) (Profile, error) {
	s.upserted = input
	return input, nil
}

func (s *fakeStore) SetAvatarAttachment(_ context.Context, userID int64, attachmentID *int64, actorUserID int64) (Profile, error) {
	s.setAvatarAttachmentID = attachmentID
	s.setAvatarActorID = actorUserID
	s.profile.UserID = userID
	s.profile.AvatarAttachmentID = attachmentID
	return s.profile, nil
}

func (s *fakeStore) GetAvatarAttachment(_ context.Context, attachmentID int64) (AvatarAttachment, error) {
	if s.avatarAttachment.ID == attachmentID {
		return s.avatarAttachment, nil
	}
	return AvatarAttachment{}, ErrProfileInvalid
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

type fakeAvatarUploader struct {
	called     bool
	attachment attachments.Attachment
	err        error
}

func (u *fakeAvatarUploader) UploadAvatar(_ context.Context, _ identity.Actor, _ attachments.UploadInput) (attachments.Attachment, error) {
	u.called = true
	if u.err != nil {
		return attachments.Attachment{}, u.err
	}
	return u.attachment, nil
}

type profileReadSeekCloser struct {
	*strings.Reader
}

func newProfileReadSeekCloser(value string) *profileReadSeekCloser {
	return &profileReadSeekCloser{Reader: strings.NewReader(value)}
}

func (r *profileReadSeekCloser) Close() error { return nil }

type fakeProfileOptionStore struct {
	items map[string]string
}

func newProfileAvatarOptions(values map[string]string) *options.Service {
	return options.NewServiceWithCacheTTL(&fakeProfileOptionStore{items: values}, time.Minute)
}

func (s *fakeProfileOptionStore) List(context.Context) ([]options.Option, error) {
	items := make([]options.Option, 0, len(s.items))
	for name, value := range s.items {
		items = append(items, options.Option{Name: name, Value: value})
	}
	return items, nil
}

func (s *fakeProfileOptionStore) InsertMissing(_ context.Context, input options.UpdateInput) error {
	if s.items == nil {
		s.items = map[string]string{}
	}
	if _, ok := s.items[input.Name]; !ok {
		s.items[input.Name] = input.Value
	}
	return nil
}

func (s *fakeProfileOptionStore) Upsert(_ context.Context, input options.UpdateInput) (options.Option, error) {
	if s.items == nil {
		s.items = map[string]string{}
	}
	s.items[input.Name] = input.Value
	return options.Option{Name: input.Name, Value: input.Value}, nil
}
