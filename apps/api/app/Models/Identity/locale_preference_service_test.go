package identity

import (
	"context"
	"errors"
	"testing"
)

type localePreferenceStore struct {
	userID int64
	locale string
	result CurrentUser
	err    error
}

func (s *localePreferenceStore) UpdateCurrentUserLocale(_ context.Context, userID int64, locale string) (CurrentUser, error) {
	s.userID = userID
	s.locale = locale
	return s.result, s.err
}

func TestLocalePreferenceServiceUpdate(t *testing.T) {
	store := &localePreferenceStore{result: CurrentUser{ID: 42, Locale: "zh-CN"}}
	result, err := NewLocalePreferenceService(store).Update(t.Context(), 42, " zh-CN ")
	if err != nil {
		t.Fatal(err)
	}
	if result.Locale != "zh-CN" || store.userID != 42 || store.locale != "zh-CN" {
		t.Fatalf("result=%#v store user=%d locale=%q", result, store.userID, store.locale)
	}
}

func TestLocalePreferenceServiceRejectsInvalidOrUnavailableUpdates(t *testing.T) {
	service := NewLocalePreferenceService(nil)
	for name, test := range map[string]struct {
		userID int64
		locale string
		want   error
	}{
		"missing user":   {userID: 0, locale: "zh-CN", want: ErrInvalidUserUpdate},
		"missing locale": {userID: 42, locale: " ", want: ErrInvalidUserUpdate},
		"missing store":  {userID: 42, locale: "zh-CN", want: ErrUserLocaleUpdateUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := service.Update(t.Context(), test.userID, test.locale)
			if !errors.Is(err, test.want) {
				t.Fatalf("error=%v want=%v", err, test.want)
			}
		})
	}
}
