package options

import (
	"context"
	"errors"
	"testing"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

func TestServiceReturnsDefaultSiteName(t *testing.T) {
	service := NewServiceWithCacheTTL(&fakeStore{}, time.Minute)

	value, err := service.WebOption(context.Background(), NameSiteName)
	if err != nil {
		t.Fatalf("WebOption returned error: %v", err)
	}
	if value != "SForum" {
		t.Fatalf("expected default site name, got %q", value)
	}
}

func TestServiceUsesCacheUntilInvalidated(t *testing.T) {
	store := &fakeStore{items: map[string]string{NameSiteName: "First"}}
	service := NewServiceWithCacheTTL(store, time.Minute)
	ctx := context.Background()

	first, err := service.WebOption(ctx, NameSiteName)
	if err != nil {
		t.Fatalf("first WebOption returned error: %v", err)
	}
	store.items[NameSiteName] = "Second"
	second, err := service.WebOption(ctx, NameSiteName)
	if err != nil {
		t.Fatalf("second WebOption returned error: %v", err)
	}
	service.Invalidate()
	third, err := service.WebOption(ctx, NameSiteName)
	if err != nil {
		t.Fatalf("third WebOption returned error: %v", err)
	}

	if first != "First" || second != "First" || third != "Second" {
		t.Fatalf("expected cache then invalidation, got %q %q %q", first, second, third)
	}
	if store.listCalls != 2 {
		t.Fatalf("expected 2 list calls, got %d", store.listCalls)
	}
}

func TestServiceUpdateRequiresSettingsPermission(t *testing.T) {
	service := NewServiceWithCacheTTL(&fakeStore{}, time.Minute)
	actor := identity.Actor{ID: 2, Status: identity.UserStatusActive, Permissions: map[string]bool{}}

	_, err := service.Update(context.Background(), actor, UpdateInput{Name: NameSiteName, Value: "Example"})
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestServiceUpdateInvalidatesCache(t *testing.T) {
	store := &fakeStore{items: map[string]string{NameSiteName: "First"}}
	service := NewServiceWithCacheTTL(store, time.Minute)
	ctx := context.Background()
	actor := identity.Actor{
		ID:          1,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionSettingsManage: true},
	}

	if _, err := service.WebOption(ctx, NameSiteName); err != nil {
		t.Fatalf("prime cache: %v", err)
	}
	updated, err := service.Update(ctx, actor, UpdateInput{Name: NameSiteName, Value: "  Example Forum  "})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	value, err := service.WebOption(ctx, NameSiteName)
	if err != nil {
		t.Fatalf("WebOption after update returned error: %v", err)
	}

	if updated.Value != "Example Forum" || value != "Example Forum" {
		t.Fatalf("expected updated value, got option=%q value=%q", updated.Value, value)
	}
}

func TestServiceRejectsUnknownOrEmptyOption(t *testing.T) {
	service := NewServiceWithCacheTTL(&fakeStore{}, time.Minute)
	actor := identity.Actor{
		ID:          1,
		Status:      identity.UserStatusActive,
		Permissions: map[string]bool{identity.PermissionSettingsManage: true},
	}

	if _, err := service.Get(context.Background(), "missing"); !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("expected invalid option for unknown get, got %v", err)
	}
	if _, err := service.Update(context.Background(), actor, UpdateInput{Name: NameSiteName, Value: ""}); !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("expected invalid option for empty update, got %v", err)
	}
}

type fakeStore struct {
	items     map[string]string
	listCalls int
}

func (s *fakeStore) List(context.Context) ([]Option, error) {
	s.listCalls++
	options := make([]Option, 0, len(s.items))
	for name, value := range s.items {
		options = append(options, Option{Name: name, Value: value})
	}
	return options, nil
}

func (s *fakeStore) Upsert(_ context.Context, input UpdateInput) (Option, error) {
	if s.items == nil {
		s.items = map[string]string{}
	}
	s.items[input.Name] = input.Value
	return Option{Name: input.Name, Value: input.Value}, nil
}
