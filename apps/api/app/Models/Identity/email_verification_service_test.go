package identity

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type emailVerificationTestStore struct {
	target   EmailVerificationTarget
	created  CreateEmailVerificationTokenInput
	verified bool
}

func (s *emailVerificationTestStore) GetEmailVerificationTarget(context.Context, int64) (EmailVerificationTarget, error) {
	return s.target, nil
}
func (s *emailVerificationTestStore) CreateEmailVerificationToken(_ context.Context, input CreateEmailVerificationTokenInput) (EmailVerificationToken, error) {
	s.created = input
	return EmailVerificationToken{UserID: input.UserID, Email: input.Email, TokenHash: input.TokenHash, ExpiresAt: input.ExpiresAt}, nil
}
func (s *emailVerificationTestStore) ConfirmEmailVerification(context.Context, string) (int64, error) {
	s.verified = true
	return s.target.UserID, nil
}
func (s *emailVerificationTestStore) IsEmailVerified(context.Context, int64) (bool, error) {
	return s.verified, nil
}

type emailVerificationTestPolicy struct{ policy EmailVerificationPolicy }

func (p emailVerificationTestPolicy) EmailVerificationPolicy(context.Context) (EmailVerificationPolicy, error) {
	return p.policy, nil
}

type emailVerificationTestQueue struct {
	mail  EmailVerificationMail
	token CreateEmailVerificationTokenInput
}

func (q *emailVerificationTestQueue) QueueEmailVerification(_ context.Context, token CreateEmailVerificationTokenInput, mail EmailVerificationMail) error {
	q.token = token
	q.mail = mail
	return nil
}

func TestEmailVerificationRequestQueuesOnlyWhenRequired(t *testing.T) {
	store := &emailVerificationTestStore{target: EmailVerificationTarget{
		UserID: 42, Email: "alice@example.com", Username: "alice", Locale: "en-US", Status: UserStatusActive,
	}}
	queue := &emailVerificationTestQueue{}
	service := NewEmailVerificationService(store, queue, emailVerificationTestPolicy{policy: EmailVerificationPolicy{Required: true, BlockContent: true}}, EmailVerificationConfig{
		TokenTTL: 2 * time.Hour,
	})

	sent, err := service.Request(context.Background(), 42, "127.0.0.1", "")
	if err != nil || !sent {
		t.Fatalf("request sent=%v err=%v", sent, err)
	}
	if queue.token.TokenHash == "" || strings.Contains(queue.mail.VerifyURL, queue.token.TokenHash) {
		t.Fatalf("token must be opaque in the URL and stored as a hash: input=%#v mail=%#v", queue.token, queue.mail)
	}
	if !strings.Contains(queue.mail.VerifyURL, "token=") || queue.mail.Locale != "en-US" {
		t.Fatalf("unexpected verification mail: %#v", queue.mail)
	}
}

func TestEmailVerificationGateAllowsDisabledOrVerifiedAndDeniesUnverified(t *testing.T) {
	store := &emailVerificationTestStore{target: EmailVerificationTarget{UserID: 7, Status: UserStatusActive}}
	service := NewEmailVerificationService(store, nil, emailVerificationTestPolicy{policy: EmailVerificationPolicy{Required: true, BlockContent: true}}, EmailVerificationConfig{})
	if err := service.RequireVerifiedForContent(context.Background(), 7); !errors.Is(err, ErrEmailVerificationRequired) {
		t.Fatalf("unverified gate error=%v", err)
	}
	store.verified = true
	if err := service.RequireVerifiedForContent(context.Background(), 7); err != nil {
		t.Fatalf("verified gate error=%v", err)
	}
	service = NewEmailVerificationService(store, nil, emailVerificationTestPolicy{policy: EmailVerificationPolicy{}}, EmailVerificationConfig{})
	store.verified = false
	if err := service.RequireVerifiedForContent(context.Background(), 7); err != nil {
		t.Fatalf("disabled policy gate error=%v", err)
	}
}

func TestEmailVerificationConfirmDelegatesOpaqueToken(t *testing.T) {
	store := &emailVerificationTestStore{target: EmailVerificationTarget{UserID: 9}}
	service := NewEmailVerificationService(store, nil, nil, EmailVerificationConfig{})
	userID, err := service.Confirm(context.Background(), "raw-token")
	if err != nil || userID != 9 || !store.verified {
		t.Fatalf("confirm userID=%d verified=%v err=%v", userID, store.verified, err)
	}
}
