package attachments

import (
	"context"
	"errors"
	"testing"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	storage "github.com/zhuchunshu/sforum/apps/api/app/Support/Storage"
)

func TestRequiresForumReadGate(t *testing.T) {
	if !requiresForumReadGate(nil) {
		t.Fatal("no refs must gate")
	}
	if !requiresForumReadGate([]AttachmentReference{{ResourceType: "post", ResourceID: 1}}) {
		t.Fatal("post ref must gate")
	}
	if requiresForumReadGate([]AttachmentReference{
		{ResourceType: ResourceTypeUser, Context: ContextAvatar},
		{ResourceType: ResourceTypeSEO, Context: "og_image"},
	}) {
		t.Fatal("site-only refs must not gate")
	}
	if !requiresForumReadGate([]AttachmentReference{
		{ResourceType: ResourceTypeUser, Context: ContextAvatar},
		{ResourceType: "topic", ResourceID: 9},
	}) {
		t.Fatal("mixed refs must gate")
	}
}

func TestAuthorizeAttachmentViewLoginRequired(t *testing.T) {
	store := &accessFakeStore{
		item: Attachment{
			ID: 1, PublicID: "abc", Status: StatusActive, Visibility: VisibilityPublic,
		},
		refs: []AttachmentReference{{ResourceType: "post", ResourceID: 3, Context: "inline"}},
	}
	optStore := &fakeOptionStore{items: map[string]string{
		options.NameForumGuestRead: "login_required",
	}}
	service := NewService(store, options.NewService(optStore))

	// 匿名 + 帖子引用 → 401 语义错误
	err := service.authorizeAttachmentView(context.Background(), identity.Actor{}, store.item)
	if !errors.Is(err, ErrGuestLoginRequired) {
		t.Fatalf("anon forum media: %v", err)
	}

	// 已登录 → 允许
	active := identity.Actor{ID: 2, Status: identity.UserStatusActive}
	if err := service.authorizeAttachmentView(context.Background(), active, store.item); err != nil {
		t.Fatalf("auth user: %v", err)
	}

	// 头像引用 → 匿名允许
	store.refs = []AttachmentReference{{ResourceType: ResourceTypeUser, Context: ContextAvatar}}
	if err := service.authorizeAttachmentView(context.Background(), identity.Actor{}, store.item); err != nil {
		t.Fatalf("avatar anon: %v", err)
	}

	// 无引用 fail closed
	store.refs = nil
	if err := service.authorizeAttachmentView(context.Background(), identity.Actor{}, store.item); !errors.Is(err, ErrGuestLoginRequired) {
		t.Fatalf("unreferenced: %v", err)
	}
}

func TestAuthorizeAttachmentViewPublicMode(t *testing.T) {
	store := &accessFakeStore{
		item: Attachment{
			ID: 1, PublicID: "abc", Status: StatusActive, Visibility: VisibilityPublic,
		},
	}
	optStore := &fakeOptionStore{items: map[string]string{
		options.NameForumGuestRead: "public",
	}}
	service := NewService(store, options.NewService(optStore))
	if err := service.authorizeAttachmentView(context.Background(), identity.Actor{}, store.item); err != nil {
		t.Fatalf("public mode: %v", err)
	}
}

func TestDecorateURLProxiesUnderLoginRequired(t *testing.T) {
	store := &accessFakeStore{
		item: Attachment{
			ID: 1, PublicID: "pub1", Status: StatusActive, Visibility: VisibilityPublic,
			ObjectKey: "k", Provider: "local",
		},
		refs: []AttachmentReference{{ResourceType: "post", ResourceID: 1}},
	}
	optStore := &fakeOptionStore{items: map[string]string{
		options.NameForumGuestRead: "login_required",
	}}
	adapter := &fakeStorageAdapter{publicBaseURL: "https://cdn.example.com"}
	service := NewServiceWithAdapterFactory(store, options.NewService(optStore), func(storage.Config) (storage.Adapter, error) {
		return adapter, nil
	})
	// 需要 import storage — 用 decorate 直接测
	decorated := service.decorateURL(context.Background(), store.item)
	if decorated.URL != contentURLPath("pub1") {
		t.Fatalf("expected proxy URL, got %q", decorated.URL)
	}

	// 头像仍可用 CDN
	store.refs = []AttachmentReference{{ResourceType: ResourceTypeUser, Context: ContextAvatar}}
	decorated = service.decorateURL(context.Background(), store.item)
	if decorated.URL != "https://cdn.example.com/k" {
		t.Fatalf("avatar should keep CDN URL, got %q", decorated.URL)
	}
}

type accessFakeStore struct {
	item Attachment
	refs []AttachmentReference
}

func (s *accessFakeStore) Create(context.Context, CreateAttachmentInput) (Attachment, error) {
	return Attachment{}, ErrInvalidAttachment
}
func (s *accessFakeStore) GetByPublicID(_ context.Context, publicID string) (Attachment, error) {
	if s.item.PublicID == publicID {
		return s.item, nil
	}
	return Attachment{}, ErrAttachmentNotFound
}
func (s *accessFakeStore) GetByID(_ context.Context, id int64) (Attachment, error) {
	if s.item.ID == id {
		return s.item, nil
	}
	return Attachment{}, ErrAttachmentNotFound
}
func (s *accessFakeStore) List(context.Context, AttachmentListInput) (AttachmentList, error) {
	return AttachmentList{}, nil
}
func (s *accessFakeStore) ListReferences(context.Context, int64) ([]AttachmentReference, error) {
	return s.refs, nil
}
func (s *accessFakeStore) UpdateStatus(context.Context, int64, string, bool) (Attachment, error) {
	return Attachment{}, nil
}
func (s *accessFakeStore) ListCleanupCandidates(context.Context, time.Time, int) ([]Attachment, error) {
	return nil, nil
}
func (s *accessFakeStore) DeleteMetadata(context.Context, int64) error { return nil }
