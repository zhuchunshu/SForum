package apitokens

import (
	"context"
	"testing"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

func TestCreateAuthenticateRevoke(t *testing.T) {
	store := &memStore{byPublic: map[string]Record{}, byID: map[int64]Record{}}
	svc := NewService(store, nil)
	actor := identity.Actor{
		ID: 7, Status: identity.UserStatusActive,
		Permissions: map[string]bool{
			identity.PermissionTopicCreate: true,
			identity.PermissionPostCreate:  true,
		},
	}
	created, err := svc.Create(context.Background(), actor, CreateInput{
		Name: "ci", Scopes: []string{identity.PermissionTopicCreate},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Plaintext == "" || created.ID == 0 {
		t.Fatalf("created=%#v", created)
	}

	auth, err := svc.AuthenticatePlaintext(context.Background(), created.Plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if auth.UserID != 7 || len(auth.Scopes) != 1 {
		t.Fatalf("auth=%#v", auth)
	}

	if err := svc.Revoke(context.Background(), actor, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AuthenticatePlaintext(context.Background(), created.Plaintext); err != ErrTokenRevoked {
		t.Fatalf("want revoked, got %v", err)
	}
}

func TestRestrictActorStripsSuperAdmin(t *testing.T) {
	actor := identity.Actor{
		ID: 1, Status: identity.UserStatusActive,
		RoleKeys: []string{identity.RoleSuperAdmin, identity.RoleMember},
		Permissions: map[string]bool{
			identity.PermissionTopicCreate: true,
		},
	}
	if !actor.IsSuperAdmin() {
		t.Fatal("expected super admin")
	}
	restricted := RestrictActor(actor, []string{identity.PermissionTopicCreate})
	if restricted.IsSuperAdmin() {
		t.Fatal("PAT must not retain super_admin bypass")
	}
	if !restricted.Can(identity.PermissionTopicCreate) {
		t.Fatal("expected topic.create")
	}
	if restricted.Can(identity.PermissionSettingsManage) {
		t.Fatal("settings.manage must not pass without scope")
	}
}

func TestRejectScopeUserLacks(t *testing.T) {
	svc := NewService(&memStore{byPublic: map[string]Record{}, byID: map[int64]Record{}}, nil)
	actor := identity.Actor{ID: 1, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionPostCreate: true}}
	_, err := svc.Create(context.Background(), actor, CreateInput{Name: "x", Scopes: []string{identity.PermissionSettingsManage}})
	if err != ErrScopeNotAllowed {
		t.Fatalf("got %v", err)
	}
}

type memStore struct {
	next     int64
	byPublic map[string]Record
	byID     map[int64]Record
}

func (m *memStore) Create(_ context.Context, userID int64, publicID, tokenHash, name string, scopes []string, expiresAt *time.Time) (Record, error) {
	m.next++
	rec := Record{ID: m.next, UserID: userID, PublicID: publicID, TokenHash: tokenHash, Name: name, Scopes: scopes, ExpiresAt: expiresAt, CreatedAt: time.Now().UTC()}
	m.byPublic[publicID] = rec
	m.byID[rec.ID] = rec
	return rec, nil
}
func (m *memStore) ListByUser(context.Context, int64, bool) ([]Record, error) { return nil, nil }
func (m *memStore) GetByPublicID(_ context.Context, publicID string) (Record, error) {
	rec, ok := m.byPublic[publicID]
	if !ok {
		return Record{}, ErrTokenNotFound
	}
	return rec, nil
}
func (m *memStore) GetByIDForUser(_ context.Context, userID, id int64) (Record, error) {
	rec, ok := m.byID[id]
	if !ok || rec.UserID != userID {
		return Record{}, ErrTokenNotFound
	}
	return rec, nil
}
func (m *memStore) Revoke(_ context.Context, userID, id int64) error {
	rec, ok := m.byID[id]
	if !ok || rec.UserID != userID {
		return ErrTokenNotFound
	}
	now := time.Now().UTC()
	rec.RevokedAt = &now
	m.byID[id] = rec
	m.byPublic[rec.PublicID] = rec
	return nil
}
func (m *memStore) TouchLastUsed(_ context.Context, id int64) error {
	rec, ok := m.byID[id]
	if !ok {
		return nil
	}
	now := time.Now().UTC()
	rec.LastUsedAt = &now
	m.byID[id] = rec
	m.byPublic[rec.PublicID] = rec
	return nil
}
