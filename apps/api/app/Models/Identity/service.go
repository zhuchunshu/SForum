package identity

import (
	"context"
	"errors"
	"net/mail"
	"sort"
	"strings"
)

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

type RegisterInput struct {
	Username    string
	Email       string
	Password    string
	DisplayName string
	Locale      string
}

type LoginInput struct {
	Login    string
	Password string
}

func (s *Service) RegistrationStatus(ctx context.Context) (RegistrationStatus, error) {
	hasAnyUser, err := s.store.AnyUserExists(ctx)
	if err != nil {
		return RegistrationStatus{}, err
	}

	return RegistrationStatus{NextUserIsInitialSuperAdmin: !hasAnyUser}, nil
}

func (s *Service) ValidateRegister(ctx context.Context, input RegisterInput) error {
	normalized := normalizeRegisterInput(input)
	fields := validateRegisterInput(normalized.Username, normalized.Email, input.Password)
	if len(fields) > 0 {
		return NewRegisterInvalid(fields)
	}

	conflicts, err := s.store.FindRegistrationConflicts(ctx, normalized.Username, normalized.Email)
	if err != nil {
		return err
	}
	if conflictFields := registrationConflictFields(conflicts); len(conflictFields) > 0 {
		return NewRegisterInvalid(conflictFields)
	}
	return nil
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (CurrentUser, error) {
	normalized := normalizeRegisterInput(input)

	fields := validateRegisterInput(normalized.Username, normalized.Email, input.Password)
	if len(fields) > 0 {
		return CurrentUser{}, NewRegisterInvalid(fields)
	}

	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return CurrentUser{}, err
	}

	var current CurrentUser
	err = s.store.WithBootstrapTx(ctx, func(ctx context.Context, tx TxStore) error {
		hasAnyUser, err := tx.AnyUserExists(ctx)
		if err != nil {
			return err
		}
		conflicts, err := tx.FindRegistrationConflicts(ctx, normalized.Username, normalized.Email)
		if err != nil {
			return err
		}
		conflictFields := registrationConflictFields(conflicts)
		if len(conflictFields) > 0 {
			return NewRegisterInvalid(conflictFields)
		}

		current, err = tx.CreateUser(ctx, CreateUserInput{
			Username:            normalized.Username,
			Email:               normalized.Email,
			DisplayName:         normalized.DisplayName,
			Locale:              normalized.Locale,
			IsInitialSuperAdmin: !hasAnyUser,
		})
		if err != nil {
			return err
		}
		if err := tx.CreateCredential(ctx, current.ID, passwordHash); err != nil {
			return err
		}

		member, err := tx.GetDefaultRole(ctx)
		if err != nil {
			return err
		}
		if err := tx.AssignRole(ctx, current.ID, member.ID); err != nil {
			return err
		}
		current.RoleKeys = append(current.RoleKeys, member.Key)

		if !hasAnyUser {
			superAdmin, err := tx.GetRole(ctx, RoleSuperAdmin)
			if err != nil {
				return err
			}
			if err := tx.AssignRole(ctx, current.ID, superAdmin.ID); err != nil {
				return err
			}
			current.RoleKeys = append(current.RoleKeys, superAdmin.Key)
		}

		return tx.LoadCurrentUserAccess(ctx, &current)
	})
	if err != nil {
		return CurrentUser{}, err
	}

	return current, nil
}

type normalizedRegisterInput struct {
	Username    string
	Email       string
	DisplayName string
	Locale      string
}

func normalizeRegisterInput(input RegisterInput) normalizedRegisterInput {
	username := strings.TrimSpace(input.Username)
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = username
	}
	locale := strings.TrimSpace(input.Locale)
	if locale == "" {
		locale = "zh-CN"
	}

	return normalizedRegisterInput{
		Username:    username,
		Email:       strings.TrimSpace(input.Email),
		DisplayName: displayName,
		Locale:      locale,
	}
}

func validateRegisterInput(username string, email string, password string) FieldMessages {
	fields := FieldMessages{}
	if username == "" {
		addFieldMessage(fields, FieldUsername, MessageUsernameRequired)
	}
	if email == "" {
		addFieldMessage(fields, FieldEmail, MessageEmailRequired)
	} else if !isValidEmail(email) {
		addFieldMessage(fields, FieldEmail, MessageEmailInvalid)
	}
	if len([]rune(password)) < 12 {
		addFieldMessage(fields, FieldPassword, MessagePasswordMin)
	}
	return fields
}

func isValidEmail(email string) bool {
	parsed, err := mail.ParseAddress(email)
	return err == nil && parsed.Address == email
}

func registrationConflictFields(conflicts RegistrationConflicts) FieldMessages {
	fields := FieldMessages{}
	if conflicts.UsernameTaken {
		addFieldMessage(fields, FieldUsername, MessageUsernameTaken)
	}
	if conflicts.EmailTaken {
		addFieldMessage(fields, FieldEmail, MessageEmailTaken)
	}
	return fields
}

func addFieldMessage(fields FieldMessages, field string, message string) {
	fields[field] = append(fields[field], message)
}

func (s *Service) Login(ctx context.Context, input LoginInput) (CurrentUser, error) {
	credential, err := s.store.GetCredentialByLogin(ctx, strings.TrimSpace(input.Login))
	if err != nil {
		if !errors.Is(err, ErrCredentialNotFound) {
			return CurrentUser{}, err
		}
		return CurrentUser{}, ErrInvalidCredentials
	}

	ok, err := VerifyPassword(input.Password, credential.PasswordHash)
	if err != nil {
		return CurrentUser{}, err
	}
	if !ok {
		return CurrentUser{}, ErrInvalidCredentials
	}
	if credential.Status != UserStatusActive {
		return CurrentUser{}, ErrInvalidCredentials
	}

	return credential.CurrentUser, nil
}

func (s *Service) RecordLoginAudit(ctx context.Context, input LoginAudit) error {
	return s.store.RecordLoginAudit(ctx, input)
}

func (s *Service) CurrentUser(ctx context.Context, userID int64) (CurrentUser, error) {
	return s.store.GetCurrentUser(ctx, userID)
}

func (s *Service) Actor(ctx context.Context, userID int64) (Actor, error) {
	return s.store.LoadActor(ctx, userID)
}

func (s *Service) ListPermissions(ctx context.Context, actor Actor) ([]Permission, error) {
	if !canManagePermissions(actor) {
		return nil, ErrPermissionDenied
	}
	return s.store.ListPermissions(ctx)
}

func (s *Service) ListPermissionMatrix(ctx context.Context, actor Actor) (PermissionMatrix, error) {
	if !canManagePermissions(actor) {
		return PermissionMatrix{}, ErrPermissionDenied
	}
	return s.store.ListPermissionMatrix(ctx)
}

func (s *Service) ListRoles(ctx context.Context, actor Actor) ([]Role, error) {
	if !actor.Can(PermissionRoleManage) {
		return nil, ErrPermissionDenied
	}
	return s.store.ListRoles(ctx)
}

func (s *Service) ListUsers(ctx context.Context, actor Actor, input UserListInput) (AdminUserList, error) {
	if !actor.Can(PermissionUserManage) {
		return AdminUserList{}, ErrPermissionDenied
	}
	input.Page, input.PerPage = normalizePage(input.Page, input.PerPage)
	input.Query = strings.TrimSpace(input.Query)
	input.Status = strings.TrimSpace(input.Status)
	input.RoleKey = strings.TrimSpace(input.RoleKey)
	return s.store.ListUsers(ctx, input)
}

func (s *Service) GetAdminUser(ctx context.Context, actor Actor, userID int64) (AdminUserDetail, error) {
	if !actor.Can(PermissionUserManage) {
		return AdminUserDetail{}, ErrPermissionDenied
	}
	return s.store.GetAdminUser(ctx, userID)
}

func (s *Service) CreateRole(ctx context.Context, actor Actor, input RoleInput) (Role, error) {
	if !actor.Can(PermissionRoleManage) {
		return Role{}, ErrPermissionDenied
	}
	return s.store.CreateRole(ctx, input)
}

func (s *Service) UpdateRole(ctx context.Context, actor Actor, roleKey string, input RoleInput) (Role, error) {
	if !actor.Can(PermissionRoleManage) {
		return Role{}, ErrPermissionDenied
	}
	input.Key = roleKey
	return s.store.UpdateRole(ctx, roleKey, input)
}

func (s *Service) DeleteRole(ctx context.Context, actor Actor, roleKey string) error {
	if !actor.Can(PermissionRoleManage) {
		return ErrPermissionDenied
	}
	if roleKey == RoleMember {
		return ErrDefaultRoleLocked
	}
	if roleKey == RoleSuperAdmin {
		return ErrSystemRoleLocked
	}
	return s.store.DeleteRole(ctx, roleKey)
}

func (s *Service) ReplaceRolePermissions(ctx context.Context, actor Actor, roleKey string, permissions []string) error {
	if !actor.Can(PermissionRoleManage) {
		return ErrPermissionDenied
	}
	if roleKey == RoleSuperAdmin {
		return ErrSystemRoleLocked
	}
	normalized := normalizeKeyList(permissions)
	if err := s.validatePermissions(ctx, normalized); err != nil {
		return err
	}
	return s.store.ReplaceRolePermissions(ctx, actor.ID, roleKey, normalized)
}

func (s *Service) ReplaceUserRoles(ctx context.Context, actor Actor, targetUserID int64, roleKeys []string) (AdminUserDetail, error) {
	if !actor.Can(PermissionUserManage) {
		return AdminUserDetail{}, ErrPermissionDenied
	}

	target, err := s.store.GetAdminUser(ctx, targetUserID)
	if err != nil {
		return AdminUserDetail{}, err
	}

	normalized := normalizeKeyList(roleKeys)
	if target.IsInitialSuperAdmin && !containsString(normalized, RoleSuperAdmin) {
		return AdminUserDetail{}, ErrInitialSuperAdminLocked
	}
	if err := s.validateRoles(ctx, normalized); err != nil {
		return AdminUserDetail{}, err
	}

	return s.store.ReplaceUserRoles(ctx, actor.ID, targetUserID, normalized)
}

func (s *Service) ReplaceUserPermissionOverrides(ctx context.Context, actor Actor, targetUserID int64, overrides PermissionOverrides) (AdminUserDetail, error) {
	if !actor.Can(PermissionUserManage) {
		return AdminUserDetail{}, ErrPermissionDenied
	}

	target, err := s.store.GetAdminUser(ctx, targetUserID)
	if err != nil {
		return AdminUserDetail{}, err
	}
	if containsString(target.RoleKeys, RoleSuperAdmin) {
		return AdminUserDetail{}, ErrSuperAdminOverridesLocked
	}

	normalized := PermissionOverrides{
		Allow: normalizeKeyList(overrides.Allow),
		Deny:  normalizeKeyList(overrides.Deny),
	}
	if hasOverrideConflict(normalized) {
		return AdminUserDetail{}, ErrPermissionOverrideConflict
	}
	if err := s.validatePermissions(ctx, append(append([]string{}, normalized.Allow...), normalized.Deny...)); err != nil {
		return AdminUserDetail{}, err
	}

	return s.store.ReplaceUserPermissionOverrides(ctx, actor.ID, targetUserID, normalized)
}

func canManagePermissions(actor Actor) bool {
	return actor.Can(PermissionRoleManage) || actor.Can(PermissionUserManage)
}

func normalizePage(page int, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

func normalizeKeyList(values []string) []string {
	seen := map[string]bool{}
	keys := make([]string, 0, len(values))
	for _, value := range values {
		key := strings.TrimSpace(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *Service) validatePermissions(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	permissions, err := s.store.ListPermissions(ctx)
	if err != nil {
		return err
	}
	known := make(map[string]bool, len(permissions))
	for _, permission := range permissions {
		known[permission.Key] = true
	}
	for _, key := range keys {
		if !known[key] {
			return ErrInvalidPermission
		}
	}
	return nil
}

func (s *Service) validateRoles(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	roles, err := s.store.ListRoles(ctx)
	if err != nil {
		return err
	}
	known := make(map[string]bool, len(roles))
	for _, role := range roles {
		known[role.Key] = true
	}
	for _, key := range keys {
		if !known[key] {
			return ErrInvalidRole
		}
	}
	return nil
}

func hasOverrideConflict(overrides PermissionOverrides) bool {
	allow := make(map[string]bool, len(overrides.Allow))
	for _, key := range overrides.Allow {
		allow[key] = true
	}
	for _, key := range overrides.Deny {
		if allow[key] {
			return true
		}
	}
	return false
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
