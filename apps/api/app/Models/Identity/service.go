package identity

import (
	"context"
	"net/mail"
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

func (s *Service) ListRoles(ctx context.Context, actor Actor) ([]Role, error) {
	if !actor.Can(PermissionRoleManage) {
		return nil, ErrPermissionDenied
	}
	return s.store.ListRoles(ctx)
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
	return s.store.ReplaceRolePermissions(ctx, actor.ID, roleKey, permissions)
}
