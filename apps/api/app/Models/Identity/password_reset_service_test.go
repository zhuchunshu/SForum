package identity

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// newResetFakeStore 构造一个预置用户的密码重置测试 store。
func newResetFakeStore() *fakeStore {
	store := &fakeStore{
		nextUserID:         100,
		users:              map[int64]CurrentUser{},
		userEmails:         map[int64]string{},
		credentials:        map[int64]string{},
		loginIndex:         map[string]int64{},
		roles:              map[string]Role{},
		roleByID:           map[int64]Role{},
		userRoleIDs:        map[int64][]int64{},
		rolePerms:          map[int64][]string{},
		userOverrides:      map[int64]PermissionOverrides{},
		consumeResetUserID: 42,
	}
	store.users[42] = CurrentUser{ID: 42, Username: "alice", DisplayName: "Alice", Status: UserStatusActive}
	store.userEmails[42] = "alice@example.com"
	store.loginIndex["alice@example.com"] = 42
	store.credentials[42] = "$argon2id$v=19$m=65536,t=1,p=4$c2FsdA$kZXcwB2Q"
	return store
}

func TestPasswordResetRequestSilentlySucceedsForUnknownEmail(t *testing.T) {
	store := newResetFakeStore()
	service := NewPasswordResetService(store, &fakeResetQueue{store: store}, PasswordResetConfig{})

	err := service.RequestPasswordReset(context.Background(), RequestPasswordResetInput{Email: "nobody@example.com"})
	if err != nil {
		t.Fatalf("expected silent success for unknown email, got %v", err)
	}
	if store.createdResetToken.UserID != 0 {
		t.Fatalf("expected no token created for unknown email, got %#v", store.createdResetToken)
	}
}

func TestPasswordResetRequestCreatesTokenForKnownUser(t *testing.T) {
	store := newResetFakeStore()
	service := NewPasswordResetService(store, &fakeResetQueue{store: store}, PasswordResetConfig{})

	err := service.RequestPasswordReset(context.Background(), RequestPasswordResetInput{Email: "alice@example.com", IP: "127.0.0.1"})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if store.createdResetToken.UserID != 42 {
		t.Fatalf("expected token for user 42, got %d", store.createdResetToken.UserID)
	}
	if store.createdResetToken.TokenHash == "" {
		t.Fatal("expected non-empty token hash")
	}
	if store.createdResetToken.ExpiresAt.Before(time.Now()) {
		t.Fatal("expected future expiry")
	}
	if store.createdResetToken.RequestIPHash == "" {
		t.Fatal("expected ip hash")
	}
}

type fakeResetQueue struct {
	store *fakeStore
	mail  PasswordResetMail
}

func (q *fakeResetQueue) QueuePasswordReset(_ context.Context, token CreatePasswordResetTokenInput, mail PasswordResetMail) error {
	q.store.createdResetToken = token
	q.mail = mail
	return nil
}

func TestPasswordResetRequestIgnoresEmptyEmail(t *testing.T) {
	store := newResetFakeStore()
	service := NewPasswordResetService(store, nil, PasswordResetConfig{})

	if err := service.RequestPasswordReset(context.Background(), RequestPasswordResetInput{Email: "  "}); err != nil {
		t.Fatalf("expected nil for empty email, got %v", err)
	}
}

func TestPasswordResetConfirmUpdatesPassword(t *testing.T) {
	store := newResetFakeStore()
	service := NewPasswordResetService(store, nil, PasswordResetConfig{})

	err := service.ConfirmPasswordReset(context.Background(), ConfirmPasswordResetInput{
		Token:       "some-token",
		NewPassword: "a-very-strong-password",
	})
	if err != nil {
		t.Fatalf("expected confirm success, got %v", err)
	}
	if store.updatedPasswordUserID != 42 {
		t.Fatalf("expected password update for user 42, got %d", store.updatedPasswordUserID)
	}
	if store.updatedPasswordHash == "" {
		t.Fatal("expected non-empty password hash")
	}
}

// TestPasswordResetConfirmRevokesSessionDirectory 验证：密码重置成功后，除了递增令牌版本号
// 让旧会话鉴权失效，还要同步撤销 user_sessions 目录，避免设备列表显示失真与
// EnforceMaxSessions 把幽灵会话计入活跃数。
func TestPasswordResetConfirmRevokesSessionDirectory(t *testing.T) {
	store := newResetFakeStore()
	service := NewPasswordResetService(store, nil, PasswordResetConfig{})

	err := service.ConfirmPasswordReset(context.Background(), ConfirmPasswordResetInput{
		Token:       "some-token",
		NewPassword: "a-very-strong-password",
	})
	if err != nil {
		t.Fatalf("expected confirm success, got %v", err)
	}

	// 应有一条针对该用户的目录撤销调用，reason 为 password_reset。
	var found bool
	for _, call := range store.revokeCalls {
		if call.userID == 42 && call.reason == RevokeReasonPasswordReset && !call.others {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected RevokeUserSessions(42, password_reset) to be called, revokeCalls=%#v", store.revokeCalls)
	}
}

func TestPasswordResetConfirmRejectsMissingToken(t *testing.T) {
	store := newResetFakeStore()
	service := NewPasswordResetService(store, nil, PasswordResetConfig{})

	err := service.ConfirmPasswordReset(context.Background(), ConfirmPasswordResetInput{Token: "  ", NewPassword: "a-very-strong-password"})
	if !errors.Is(err, ErrPasswordResetTokenNotFound) {
		t.Fatalf("expected ErrPasswordResetTokenNotFound, got %v", err)
	}
}

func TestPasswordResetConfirmRejectsWeakPassword(t *testing.T) {
	store := newResetFakeStore()
	service := NewPasswordResetService(store, nil, PasswordResetConfig{})

	err := service.ConfirmPasswordReset(context.Background(), ConfirmPasswordResetInput{Token: "some-token", NewPassword: "short"})
	fields := registerInvalidFields(t, err)
	if !fieldMessagesContain(fields, FieldPassword, MessagePasswordMin) {
		t.Fatalf("expected password min policy error, got %#v", fields)
	}
}

func TestPasswordResetConfirmUsesConfiguredPasswordPolicy(t *testing.T) {
	store := newResetFakeStore()
	service := NewPasswordResetServiceWithPasswordPolicy(store, nil, PasswordResetConfig{}, staticPasswordPolicyResolver{policy: PasswordPolicy{
		MinLength:     8,
		MaxLength:     64,
		RequireSymbol: true,
	}})

	err := service.ConfirmPasswordReset(context.Background(), ConfirmPasswordResetInput{Token: "some-token", NewPassword: "longenough"})
	fields := registerInvalidFields(t, err)
	if !fieldMessagesContain(fields, FieldPassword, MessagePasswordSymbol) {
		t.Fatalf("expected symbol policy error, got %#v", fields)
	}

	err = service.ConfirmPasswordReset(context.Background(), ConfirmPasswordResetInput{Token: "some-token", NewPassword: "long-enough!"})
	if err != nil {
		t.Fatalf("expected password reset to accept configured policy: %v", err)
	}
}

func TestPasswordResetConfirmPropagatesConsumedTokenError(t *testing.T) {
	store := newResetFakeStore()
	store.consumeResetTokenErr = ErrPasswordResetTokenNotFound
	service := NewPasswordResetService(store, nil, PasswordResetConfig{})

	err := service.ConfirmPasswordReset(context.Background(), ConfirmPasswordResetInput{Token: "expired-token", NewPassword: "a-very-strong-password"})
	if !errors.Is(err, ErrPasswordResetTokenNotFound) {
		t.Fatalf("expected token not found, got %v", err)
	}
}

func TestPasswordResetEmailContainsResetURL(t *testing.T) {
	service := NewPasswordResetService(nil, nil, PasswordResetConfig{SiteName: "TestSite", SiteURL: "https://forum.test"})
	body := service.resetEmailBody("alice", "https://forum.test/reset-password?token=abc", time.Now().Add(30*time.Minute))
	if !strings.Contains(body, "https://forum.test/reset-password?token=abc") {
		t.Fatalf("expected reset URL in body, got %q", body)
	}
	if !strings.Contains(body, "alice") {
		t.Fatalf("expected username in body, got %q", body)
	}
}
