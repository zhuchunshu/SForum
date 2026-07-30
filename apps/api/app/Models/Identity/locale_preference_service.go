package identity

import (
	"context"
	"strings"
)

// LocalePreferenceService owns validation and persistence for the current
// user's private language preference.
type LocalePreferenceService struct {
	store CurrentUserLocaleStore
}

func NewLocalePreferenceService(store CurrentUserLocaleStore) *LocalePreferenceService {
	return &LocalePreferenceService{store: store}
}

func (s *LocalePreferenceService) Update(ctx context.Context, userID int64, locale string) (CurrentUser, error) {
	locale = strings.TrimSpace(locale)
	if userID <= 0 || locale == "" {
		return CurrentUser{}, ErrInvalidUserUpdate
	}
	if s == nil || s.store == nil {
		return CurrentUser{}, ErrUserLocaleUpdateUnavailable
	}
	return s.store.UpdateCurrentUserLocale(ctx, userID, locale)
}
