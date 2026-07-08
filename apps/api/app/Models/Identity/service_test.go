package identity

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
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

func TestRegisterEmitsUserRegisteredEvent(t *testing.T) {
	_, store := newTestService(t)
	publisher := &fakeIdentityEventPublisher{}
	service := NewServiceWithEvents(store, publisher)

	current, err := service.Register(testContext(t), RegisterInput{
		Username: "member",
		Email:    "member@example.com",
		Password: "correct horse battery staple",
		Locale:   "zh-CN",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if len(publisher.names) != 1 || publisher.names[0] != appevents.UserRegistered {
		t.Fatalf("expected user.registered event, got %#v", publisher.names)
	}
	if publisher.payloads[0]["userId"] != current.ID {
		t.Fatalf("expected payload user id %d, got %#v", current.ID, publisher.payloads[0])
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

func TestRegisterRejectsInvalidFields(t *testing.T) {
	service, _ := newTestService(t)

	_, err := service.Register(testContext(t), RegisterInput{
		Username: " ",
		Email:    "not-an-email",
		Password: "short",
	})
	fields := registerInvalidFields(t, err)
	expected := FieldMessages{
		FieldUsername: {MessageUsernameRequired},
		FieldEmail:    {MessageEmailInvalid},
		FieldPassword: {MessagePasswordMin},
	}
	if !reflect.DeepEqual(fields, expected) {
		t.Fatalf("expected fields %#v, got %#v", expected, fields)
	}
}

func TestRegisterUsesConfiguredPasswordPolicy(t *testing.T) {
	_, store := newTestService(t)
	service := NewServiceWithPasswordPolicy(store, staticPasswordPolicyResolver{policy: PasswordPolicy{
		MinLength:        8,
		MaxLength:        64,
		RequireUppercase: true,
		RequireNumber:    true,
	}})

	err := service.ValidateRegister(context.Background(), RegisterInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "lowercaseonly",
	})
	fields := registerInvalidFields(t, err)
	if !fieldMessagesContain(fields, FieldPassword, MessagePasswordUppercase) {
		t.Fatalf("expected uppercase policy error, got %#v", fields)
	}
	if !fieldMessagesContain(fields, FieldPassword, MessagePasswordNumber) {
		t.Fatalf("expected number policy error, got %#v", fields)
	}

	_, err = service.Register(context.Background(), RegisterInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "Alice123",
	})
	if err != nil {
		t.Fatalf("expected password matching configured policy to register: %v", err)
	}
}

func TestRegisterRejectsMissingEmail(t *testing.T) {
	service, _ := newTestService(t)

	_, err := service.Register(testContext(t), RegisterInput{
		Username: "admin",
		Email:    " ",
		Password: "correct horse battery staple",
	})
	fields := registerInvalidFields(t, err)
	expected := FieldMessages{FieldEmail: {MessageEmailRequired}}
	if !reflect.DeepEqual(fields, expected) {
		t.Fatalf("expected fields %#v, got %#v", expected, fields)
	}
}

func TestRegisterRejectsDuplicateUsernameAndEmail(t *testing.T) {
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

	_, err = service.Register(ctx, RegisterInput{
		Username: " admin ",
		Email:    "ADMIN@example.com",
		Password: "correct horse battery staple",
	})
	fields := registerInvalidFields(t, err)
	expected := FieldMessages{
		FieldUsername: {MessageUsernameTaken},
		FieldEmail:    {MessageEmailTaken},
	}
	if !reflect.DeepEqual(fields, expected) {
		t.Fatalf("expected fields %#v, got %#v", expected, fields)
	}
}

func TestRegisterRejectsDuplicateUsername(t *testing.T) {
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

	_, err = service.Register(ctx, RegisterInput{
		Username: "ADMIN",
		Email:    "new@example.com",
		Password: "correct horse battery staple",
	})
	fields := registerInvalidFields(t, err)
	expected := FieldMessages{FieldUsername: {MessageUsernameTaken}}
	if !reflect.DeepEqual(fields, expected) {
		t.Fatalf("expected fields %#v, got %#v", expected, fields)
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
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

	_, err = service.Register(ctx, RegisterInput{
		Username: "member",
		Email:    "ADMIN@example.com",
		Password: "correct horse battery staple",
	})
	fields := registerInvalidFields(t, err)
	expected := FieldMessages{FieldEmail: {MessageEmailTaken}}
	if !reflect.DeepEqual(fields, expected) {
		t.Fatalf("expected fields %#v, got %#v", expected, fields)
	}
}

func TestRegisterRollsBackWhenCurrentUserAccessCannotLoad(t *testing.T) {
	service, store := newTestService(t)
	store.loadAccessErr = errors.New("load current user access failed")

	_, err := service.Register(testContext(t), RegisterInput{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "correct horse battery staple",
	})
	if err == nil {
		t.Fatal("expected Register to return an error")
	}
	if len(store.users) != 0 {
		t.Fatalf("expected user insert to roll back, got %#v", store.users)
	}
	if len(store.credentials) != 0 {
		t.Fatalf("expected credential insert to roll back, got %#v", store.credentials)
	}
	if len(store.loginIndex) != 0 {
		t.Fatalf("expected login index to roll back, got %#v", store.loginIndex)
	}
	if store.nextUserID != 1 {
		t.Fatalf("expected next user id to roll back to 1, got %d", store.nextUserID)
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

func TestLoginPropagatesCredentialStoreErrors(t *testing.T) {
	service, store := newTestService(t)
	ctx := testContext(t)

	_, err := service.Register(ctx, RegisterInput{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	store.credentialErr = errors.New("load current user access failed")
	_, err = service.Login(ctx, LoginInput{
		Login:    "admin",
		Password: "correct horse battery staple",
	})
	if err == nil || errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected internal credential loading error, got %v", err)
	}
}

func TestLoginRejectsDisabledUser(t *testing.T) {
	service, store := newTestService(t)
	ctx := testContext(t)

	registered, err := service.Register(ctx, RegisterInput{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	disabled := store.users[registered.ID]
	disabled.Status = UserStatusDisabled
	store.users[registered.ID] = disabled

	_, err = service.Login(ctx, LoginInput{
		Login:    "admin",
		Password: "correct horse battery staple",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials for disabled user, got %v", err)
	}
}

func TestRecordLoginAuditDelegatesToStore(t *testing.T) {
	service, store := newTestService(t)

	err := service.RecordLoginAudit(testContext(t), LoginAudit{
		UserID:      42,
		Action:      AuditActionLogin,
		IPAddress:   "203.0.113.10",
		UserAgent:   "test-agent",
		SessionHash: "hash",
	})
	if err != nil {
		t.Fatalf("RecordLoginAudit returned error: %v", err)
	}
	if len(store.loginAudits) != 1 {
		t.Fatalf("expected one login audit, got %#v", store.loginAudits)
	}
	if store.loginAudits[0].IPAddress != "203.0.113.10" || store.loginAudits[0].SessionHash != "hash" {
		t.Fatalf("unexpected login audit: %#v", store.loginAudits[0])
	}
}

func registerInvalidFields(t *testing.T, err error) FieldMessages {
	t.Helper()
	var registerErr *RegisterInvalidError
	if !errors.As(err, &registerErr) {
		t.Fatalf("expected register invalid error, got %v", err)
	}
	return registerErr.Fields
}

type staticPasswordPolicyResolver struct {
	policy PasswordPolicy
	err    error
}

func (r staticPasswordPolicyResolver) PasswordPolicy(context.Context) (PasswordPolicy, error) {
	return r.policy, r.err
}

func newTestService(t *testing.T) (*Service, *fakeStore) {
	t.Helper()

	store := &fakeStore{
		nextUserID:    1,
		users:         map[int64]CurrentUser{},
		userEmails:    map[int64]string{},
		credentials:   map[int64]string{},
		loginIndex:    map[string]int64{},
		roles:         map[string]Role{},
		roleByID:      map[int64]Role{},
		userRoleIDs:   map[int64][]int64{},
		rolePerms:     map[int64][]string{},
		userOverrides: map[int64]PermissionOverrides{},
		nextCustomID:  100,
	}
	store.seedRole(Role{ID: 1, Key: RoleSuperAdmin, Alias: "超级管理员", IsSystem: true, IsDeletable: false, IsEnabled: true})
	store.seedRole(Role{ID: 2, Key: RoleMember, Alias: "普通会员", IsSystem: true, IsDefault: true, IsDeletable: false, IsEnabled: true})
	store.rolePerms[1] = []string{PermissionAdminAccess, PermissionRoleManage}
	store.rolePerms[2] = []string{PermissionTopicCreate, PermissionPostCreate}

	return NewService(store), store
}

type fakeIdentityEventPublisher struct {
	names    []string
	payloads []map[string]any
}

func (p *fakeIdentityEventPublisher) Emit(_ context.Context, envelope appevents.Envelope) appevents.Result {
	p.names = append(p.names, envelope.Name)
	p.payloads = append(p.payloads, envelope.Payload)
	return appevents.Result{OK: true}
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	return context.Background()
}

type fakeStore struct {
	mu            sync.Mutex
	nextUserID    int64
	nextCustomID  int64
	users         map[int64]CurrentUser
	userEmails    map[int64]string
	credentials   map[int64]string
	loginIndex    map[string]int64
	roles         map[string]Role
	roleByID      map[int64]Role
	userRoleIDs   map[int64][]int64
	rolePerms     map[int64][]string
	userOverrides map[int64]PermissionOverrides
	loginAudits   []LoginAudit
	credentialErr error
	loadAccessErr error
	// 密码重置测试钩子。
	createdResetToken      CreatePasswordResetTokenInput
	consumedResetTokenHash string
	consumeResetTokenErr   error
	consumeResetUserID     int64
	passwordUpdated        bool
	updatedPasswordHash    string
	updatedPasswordUserID  int64
}

func (s *fakeStore) seedRole(role Role) {
	s.roles[role.Key] = role
	s.roleByID[role.ID] = role
}

func (s *fakeStore) WithBootstrapTx(ctx context.Context, fn func(context.Context, TxStore) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := s.snapshot()
	if err := fn(ctx, s); err != nil {
		s.restore(snapshot)
		return err
	}
	return nil
}

func (s *fakeStore) AnyUserExists(context.Context) (bool, error) {
	return len(s.users) > 0, nil
}

func (s *fakeStore) FindRegistrationConflicts(_ context.Context, username string, email string) (RegistrationConflicts, error) {
	return RegistrationConflicts{
		UsernameTaken: s.loginIndex[strings.ToLower(username)] != 0,
		EmailTaken:    s.loginIndex[strings.ToLower(email)] != 0,
	}, nil
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
	s.userEmails[user.ID] = input.Email
	s.loginIndex[strings.ToLower(input.Username)] = user.ID
	s.loginIndex[strings.ToLower(input.Email)] = user.ID
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

func (s *fakeStore) LoadCurrentUserAccess(_ context.Context, current *CurrentUser) error {
	if s.loadAccessErr != nil {
		return s.loadAccessErr
	}
	*current = s.withAccess(*current)
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
	if s.credentialErr != nil {
		return CredentialUser{}, s.credentialErr
	}
	userID, ok := s.loginIndex[strings.ToLower(login)]
	if !ok {
		return CredentialUser{}, ErrCredentialNotFound
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

func (s *fakeStore) ListPermissions(context.Context) ([]Permission, error) {
	permissions := make([]Permission, 0, len(SeedPermissions))
	for _, permission := range SeedPermissions {
		permissions = append(permissions, Permission{
			Key:         permission.Key,
			Module:      permission.Module,
			Description: permission.Description,
		})
	}
	return permissions, nil
}

func (s *fakeStore) ListPermissionMatrix(context.Context) (PermissionMatrix, error) {
	permissions, _ := s.ListPermissions(context.Background())
	roles := make([]RolePermissionSet, 0, len(s.roles))
	for _, role := range s.roles {
		roles = append(roles, RolePermissionSet{
			RoleKey:        role.Key,
			PermissionKeys: slices.Clone(s.rolePerms[role.ID]),
		})
	}
	return PermissionMatrix{Permissions: permissions, Roles: roles}, nil
}

func (s *fakeStore) ListUsers(_ context.Context, input UserListInput) (AdminUserList, error) {
	items := make([]AdminUserSummary, 0, len(s.users))
	for _, user := range s.users {
		items = append(items, s.adminSummary(user))
	}
	return AdminUserList{Items: items, Total: int64(len(items)), Page: input.Page, PerPage: input.PerPage}, nil
}

func (s *fakeStore) GetAdminUser(_ context.Context, userID int64) (AdminUserDetail, error) {
	user, ok := s.users[userID]
	if !ok {
		return AdminUserDetail{}, errors.New("user not found")
	}
	user = s.withAccess(user)
	return AdminUserDetail{
		AdminUserSummary:    s.adminSummary(user),
		Permissions:         slices.Clone(user.Permissions),
		PermissionOverrides: s.cloneOverrides(userID),
	}, nil
}

func (s *fakeStore) ListRoles(context.Context) ([]Role, error) {
	roles := make([]Role, 0, len(s.roles))
	for _, role := range s.roles {
		role.PermissionKeys = slices.Clone(s.rolePerms[role.ID])
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

func (s *fakeStore) ReplaceUserRoles(_ context.Context, _ int64, targetUserID int64, roleKeys []string) (AdminUserDetail, error) {
	roleIDs := make([]int64, 0, len(roleKeys))
	for _, roleKey := range roleKeys {
		role, ok := s.roles[roleKey]
		if !ok {
			return AdminUserDetail{}, errors.New("role not found")
		}
		roleIDs = append(roleIDs, role.ID)
	}
	s.userRoleIDs[targetUserID] = roleIDs
	return s.GetAdminUser(context.Background(), targetUserID)
}

func (s *fakeStore) ReplaceUserPermissionOverrides(_ context.Context, _ int64, targetUserID int64, overrides PermissionOverrides) (AdminUserDetail, error) {
	s.userOverrides[targetUserID] = PermissionOverrides{
		Allow: slices.Clone(overrides.Allow),
		Deny:  slices.Clone(overrides.Deny),
	}
	return s.GetAdminUser(context.Background(), targetUserID)
}

func (s *fakeStore) RecordLoginAudit(_ context.Context, input LoginAudit) error {
	s.loginAudits = append(s.loginAudits, input)
	return nil
}

func (s *fakeStore) CreatePasswordResetToken(_ context.Context, input CreatePasswordResetTokenInput) (PasswordResetToken, error) {
	s.createdResetToken = input
	return PasswordResetToken{
		ID:        1,
		UserID:    input.UserID,
		TokenHash: input.TokenHash,
		ExpiresAt: input.ExpiresAt,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (s *fakeStore) ConsumePasswordResetToken(_ context.Context, tokenHash string) (int64, error) {
	if s.consumeResetTokenErr != nil {
		return 0, s.consumeResetTokenErr
	}
	s.consumedResetTokenHash = tokenHash
	s.passwordUpdated = true
	return s.consumeResetUserID, nil
}

func (s *fakeStore) UpdateUserPassword(_ context.Context, userID int64, passwordHash string) error {
	s.updatedPasswordHash = passwordHash
	s.updatedPasswordUserID = userID
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
	overrides := s.userOverrides[user.ID]
	for _, permission := range overrides.Allow {
		permissionSet[permission] = true
	}
	for _, permission := range overrides.Deny {
		delete(permissionSet, permission)
	}
	permissions := make([]string, 0, len(permissionSet))
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}
	user.RoleKeys = roleKeys
	user.Permissions = permissions
	return user
}

func (s *fakeStore) adminSummary(user CurrentUser) AdminUserSummary {
	user = s.withAccess(user)
	return AdminUserSummary{
		ID:                  user.ID,
		Username:            user.Username,
		Email:               s.userEmails[user.ID],
		DisplayName:         user.DisplayName,
		Locale:              user.Locale,
		Status:              user.Status,
		IsInitialSuperAdmin: user.IsInitialSuperAdmin,
		RoleKeys:            slices.Clone(user.RoleKeys),
	}
}

func (s *fakeStore) cloneOverrides(userID int64) PermissionOverrides {
	overrides := s.userOverrides[userID]
	return PermissionOverrides{
		Allow: slices.Clone(overrides.Allow),
		Deny:  slices.Clone(overrides.Deny),
	}
}

type fakeStoreSnapshot struct {
	nextUserID    int64
	nextCustomID  int64
	users         map[int64]CurrentUser
	userEmails    map[int64]string
	credentials   map[int64]string
	loginIndex    map[string]int64
	roles         map[string]Role
	roleByID      map[int64]Role
	userRoleIDs   map[int64][]int64
	rolePerms     map[int64][]string
	userOverrides map[int64]PermissionOverrides
}

func (s *fakeStore) snapshot() fakeStoreSnapshot {
	return fakeStoreSnapshot{
		nextUserID:    s.nextUserID,
		nextCustomID:  s.nextCustomID,
		users:         cloneIntCurrentUserMap(s.users),
		userEmails:    cloneIntStringMap(s.userEmails),
		credentials:   cloneIntStringMap(s.credentials),
		loginIndex:    cloneStringIntMap(s.loginIndex),
		roles:         cloneStringRoleMap(s.roles),
		roleByID:      cloneIntRoleMap(s.roleByID),
		userRoleIDs:   cloneIntIntSliceMap(s.userRoleIDs),
		rolePerms:     cloneIntStringSliceMap(s.rolePerms),
		userOverrides: cloneIntOverridesMap(s.userOverrides),
	}
}

func (s *fakeStore) restore(snapshot fakeStoreSnapshot) {
	s.nextUserID = snapshot.nextUserID
	s.nextCustomID = snapshot.nextCustomID
	s.users = snapshot.users
	s.userEmails = snapshot.userEmails
	s.credentials = snapshot.credentials
	s.loginIndex = snapshot.loginIndex
	s.roles = snapshot.roles
	s.roleByID = snapshot.roleByID
	s.userRoleIDs = snapshot.userRoleIDs
	s.rolePerms = snapshot.rolePerms
	s.userOverrides = snapshot.userOverrides
}

func cloneIntCurrentUserMap(input map[int64]CurrentUser) map[int64]CurrentUser {
	output := make(map[int64]CurrentUser, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func cloneIntStringMap(input map[int64]string) map[int64]string {
	output := make(map[int64]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func cloneStringIntMap(input map[string]int64) map[string]int64 {
	output := make(map[string]int64, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func cloneStringRoleMap(input map[string]Role) map[string]Role {
	output := make(map[string]Role, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func cloneIntRoleMap(input map[int64]Role) map[int64]Role {
	output := make(map[int64]Role, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func cloneIntIntSliceMap(input map[int64][]int64) map[int64][]int64 {
	output := make(map[int64][]int64, len(input))
	for key, value := range input {
		output[key] = slices.Clone(value)
	}
	return output
}

func cloneIntStringSliceMap(input map[int64][]string) map[int64][]string {
	output := make(map[int64][]string, len(input))
	for key, value := range input {
		output[key] = slices.Clone(value)
	}
	return output
}

func cloneIntOverridesMap(input map[int64]PermissionOverrides) map[int64]PermissionOverrides {
	output := make(map[int64]PermissionOverrides, len(input))
	for key, value := range input {
		output[key] = PermissionOverrides{
			Allow: slices.Clone(value.Allow),
			Deny:  slices.Clone(value.Deny),
		}
	}
	return output
}
