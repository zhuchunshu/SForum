package identity

import (
	"context"
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

func (s *Service) Register(ctx context.Context, input RegisterInput) (CurrentUser, error) {
	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return CurrentUser{}, err
	}

	username := strings.TrimSpace(input.Username)
	email := strings.TrimSpace(input.Email)
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = username
	}
	locale := strings.TrimSpace(input.Locale)
	if locale == "" {
		locale = "zh-CN"
	}

	var current CurrentUser
	err = s.store.WithBootstrapTx(ctx, func(ctx context.Context, tx TxStore) error {
		hasAnyUser, err := tx.AnyUserExists(ctx)
		if err != nil {
			return err
		}

		current, err = tx.CreateUser(ctx, CreateUserInput{
			Username:            username,
			Email:               email,
			DisplayName:         displayName,
			Locale:              locale,
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

		return nil
	})
	if err != nil {
		return CurrentUser{}, err
	}

	return current, nil
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

	return credential.CurrentUser, nil
}

func (s *Service) CurrentUser(ctx context.Context, userID int64) (CurrentUser, error) {
	return s.store.GetCurrentUser(ctx, userID)
}
