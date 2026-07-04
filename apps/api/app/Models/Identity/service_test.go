package identity

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
)

func TestRegisterFirstUserAssignsSuperAdminAndMember(t *testing.T) {
	service, _ := newTestService(t)

	first, err := service.Register(testContext(t), RegisterInput{
		Username:    "admin",
		Email:       "admin@example.com",
		Password:    "correct horse battery staple",
		DisplayName: "Admin",
		Locale:      "zh-CN",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if !slices.Contains(first.RoleKeys, RoleSuperAdmin) {
		t.Fatalf("expected first user to have super_admin, got %v", first.RoleKeys)
	}
	if !slices.Contains(first.RoleKeys, RoleMember) {
		t.Fatalf("expected first user to have member, got %v", first.RoleKeys)
	}
	if !slices.Contains(first.Permissions, PermissionAdminAccess) {
		t.Fatalf("expected first user to have admin access permission, got %v", first.Permissions)
	}
	if !first.IsInitialSuperAdmin {
		t.Fatal("expected first user to be initial super admin")
	}
}

func TestRegisterSecondUserAssignsDefaultMember(t *testing.T) {
	service, _ := newTestService(t)
	ctx := testContext(t)

	_, err := service.Register(ctx, RegisterInput{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("first Register returned error: %v", err)
	}

	second, err := service.Register(ctx, RegisterInput{
		Username:    "member1",
		Email:       "member1@example.com",
		Password:    "correct horse battery staple",
		DisplayName: "Member One",
		Locale:      "zh-CN",
	})
	if err != nil {
		t.Fatalf("second Register returned error: %v", err)
	}
	if slices.Contains(second.RoleKeys, RoleSuperAdmin) {
		t.Fatalf("expected second user not to have super_admin, got %v", second.RoleKeys)
	}
	if !slices.Contains(second.RoleKeys, RoleMember) {
		t.Fatalf("expected second user to have member, got %v", second.RoleKeys)
	}
	if !slices.Contains(second.Permissions, PermissionTopicCreate) {
		t.Fatalf("expected second user to have topic create permission, got %v", second.Permissions)
	}
	if second.IsInitialSuperAdmin {
		t.Fatal("expected second user not to be initial super admin")
	}
}

func TestRegistrationStatusTracksBootstrapUser(t *testing.T) {
	service, _ := newTestService(t)
	ctx := testContext(t)

	status, err := service.RegistrationStatus(ctx)
	if err != nil {
		t.Fatalf("RegistrationStatus returned error: %v", err)
	}
	if !status.NextUserIsInitialSuperAdmin {
		t.Fatal("expected next user to be initial super admin before any user exists")
	}

	_, err = service.Register(ctx, RegisterInput{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	status, err = service.RegistrationStatus(ctx)
	if err != nil {
		t.Fatalf("RegistrationStatus after register returned error: %v", err)
	}
	if status.NextUserIsInitialSuperAdmin {
		t.Fatal("expected next user not to be initial super admin after a user exists")
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	service, _ := newTestService(t)
	ctx := testContext(t)

	_, err := service.Register(ctx, RegisterInput{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	_, err = service.Login(ctx, LoginInput{
		Login:    "admin",
		Password: "wrong password",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func newTestService(t *testing.T) (*Service, *fakeStore) {
	t.Helper()

	store := &fakeStore{
		nextUserID:   1,
		users:        map[int64]CurrentUser{},
		credentials:  map[int64]string{},
		loginIndex:   map[string]int64{},
		roles:        map[string]Role{},
		roleByID:     map[int64]Role{},
		userRoleIDs:  map[int64][]int64{},
		rolePerms:    map[int64][]string{},
		nextCustomID: 100,
	}
	store.seedRole(Role{ID: 1, Key: RoleSuperAdmin, Alias: "超级管理员", IsSystem: true, IsDeletable: false, IsEnabled: true})
	store.seedRole(Role{ID: 2, Key: RoleMember, Alias: "普通会员", IsSystem: true, IsDefault: true, IsDeletable: false, IsEnabled: true})
	store.rolePerms[1] = []string{PermissionAdminAccess, PermissionRoleManage}
	store.rolePerms[2] = []string{PermissionTopicCreate, PermissionPostCreate}

	return NewService(store), store
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	return context.Background()
}

type fakeStore struct {
	mu           sync.Mutex
	nextUserID   int64
	nextCustomID int64
	users        map[int64]CurrentUser
	credentials  map[int64]string
	loginIndex   map[string]int64
	roles        map[string]Role
	roleByID     map[int64]Role
	userRoleIDs  map[int64][]int64
	rolePerms    map[int64][]string
}

func (s *fakeStore) seedRole(role Role) {
	s.roles[role.Key] = role
	s.roleByID[role.ID] = role
}

func (s *fakeStore) WithBootstrapTx(ctx context.Context, fn func(context.Context, TxStore) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn(ctx, s)
}

func (s *fakeStore) AnyUserExists(context.Context) (bool, error) {
	return len(s.users) > 0, nil
}

func (s *fakeStore) CreateUser(_ context.Context, input CreateUserInput) (CurrentUser, error) {
	user := CurrentUser{
		ID:                  s.nextUserID,
		Username:            input.Username,
		DisplayName:         input.DisplayName,
		Locale:              input.Locale,
		Status:              UserStatusActive,
		IsInitialSuperAdmin: input.IsInitialSuperAdmin,
	}
	s.nextUserID++
	s.users[user.ID] = user
	s.loginIndex[input.Username] = user.ID
	s.loginIndex[input.Email] = user.ID
	return user, nil
}

func (s *fakeStore) CreateCredential(_ context.Context, userID int64, passwordHash string) error {
	s.credentials[userID] = passwordHash
	return nil
}

func (s *fakeStore) GetDefaultRole(context.Context) (Role, error) {
	return s.roles[RoleMember], nil
}

func (s *fakeStore) GetRole(_ context.Context, roleKey string) (Role, error) {
	role, ok := s.roles[roleKey]
	if !ok {
		return Role{}, errors.New("role not found")
	}
	return role, nil
}

func (s *fakeStore) AssignRole(_ context.Context, userID int64, roleID int64) error {
	s.userRoleIDs[userID] = append(s.userRoleIDs[userID], roleID)
	return nil
}

func (s *fakeStore) GetCurrentUser(_ context.Context, userID int64) (CurrentUser, error) {
	user, ok := s.users[userID]
	if !ok {
		return CurrentUser{}, errors.New("user not found")
	}
	return s.withAccess(user), nil
}

func (s *fakeStore) GetCredentialByLogin(_ context.Context, login string) (CredentialUser, error) {
	userID, ok := s.loginIndex[login]
	if !ok {
		return CredentialUser{}, errors.New("credential not found")
	}
	user, err := s.GetCurrentUser(context.Background(), userID)
	if err != nil {
		return CredentialUser{}, err
	}
	return CredentialUser{CurrentUser: user, PasswordHash: s.credentials[userID]}, nil
}

func (s *fakeStore) LoadActor(ctx context.Context, userID int64) (Actor, error) {
	user, err := s.GetCurrentUser(ctx, userID)
	if err != nil {
		return Actor{}, err
	}
	permissions := map[string]bool{}
	for _, permission := range user.Permissions {
		permissions[permission] = true
	}
	return Actor{ID: user.ID, Status: user.Status, RoleKeys: user.RoleKeys, Permissions: permissions}, nil
}

func (s *fakeStore) ListRoles(context.Context) ([]Role, error) {
	roles := make([]Role, 0, len(s.roles))
	for _, role := range s.roles {
		roles = append(roles, role)
	}
	return roles, nil
}

func (s *fakeStore) CreateRole(_ context.Context, input RoleInput) (Role, error) {
	role := Role{ID: s.nextCustomID, Key: input.Key, Alias: input.Alias, Description: input.Description, IsDeletable: true, IsEnabled: true}
	s.nextCustomID++
	s.seedRole(role)
	return role, nil
}

func (s *fakeStore) UpdateRole(_ context.Context, roleKey string, input RoleInput) (Role, error) {
	role := s.roles[roleKey]
	role.Alias = input.Alias
	role.Description = input.Description
	s.seedRole(role)
	return role, nil
}

func (s *fakeStore) DeleteRole(_ context.Context, roleKey string) error {
	delete(s.roles, roleKey)
	return nil
}

func (s *fakeStore) ReplaceRolePermissions(_ context.Context, _ int64, roleKey string, permissions []string) error {
	role := s.roles[roleKey]
	s.rolePerms[role.ID] = permissions
	return nil
}

func (s *fakeStore) withAccess(user CurrentUser) CurrentUser {
	roleIDs := s.userRoleIDs[user.ID]
	roleKeys := make([]string, 0, len(roleIDs))
	permissionSet := map[string]bool{}
	for _, roleID := range roleIDs {
		role := s.roleByID[roleID]
		roleKeys = append(roleKeys, role.Key)
		for _, permission := range s.rolePerms[roleID] {
			permissionSet[permission] = true
		}
	}
	permissions := make([]string, 0, len(permissionSet))
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}
	user.RoleKeys = roleKeys
	user.Permissions = permissions
	return user
}
