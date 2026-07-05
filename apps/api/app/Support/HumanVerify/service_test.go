package humanverify

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDisabledServiceSkipsVerification(t *testing.T) {
	service := NewService(ServiceConfig{Enabled: false, ChallengeTTL: time.Minute}, fakeProvider{}, NewMemoryStore())

	if err := service.Verify(context.Background(), VerifyRequest{Purpose: PurposeRegister}); err != nil {
		t.Fatalf("disabled service should skip verification, got %v", err)
	}
}

func TestServiceSkipsDisabledPurpose(t *testing.T) {
	service := NewService(
		ServiceConfig{
			Enabled:         true,
			ChallengeTTL:    time.Minute,
			EnabledPurposes: map[Purpose]bool{PurposeRegister: false, PurposeLoginRisk: true},
		},
		fakeProvider{},
		NewMemoryStore(),
	)

	challenge, err := service.Challenge(context.Background(), PurposeRegister, Subject{})
	if err != nil {
		t.Fatalf("disabled purpose challenge returned error: %v", err)
	}
	if challenge.Provider != ProviderDisabled {
		t.Fatalf("expected disabled challenge for register purpose, got %#v", challenge)
	}
	if err := service.Verify(context.Background(), VerifyRequest{Purpose: PurposeRegister}); err != nil {
		t.Fatalf("disabled purpose verify should skip token requirement, got %v", err)
	}
	if err := service.Verify(context.Background(), VerifyRequest{Purpose: PurposeLoginRisk}); !errors.Is(err, ErrRequired) {
		t.Fatalf("enabled purpose should still require a token, got %v", err)
	}
}

func TestServiceRequiresTokenWhenEnabled(t *testing.T) {
	service := NewService(ServiceConfig{Enabled: true, ChallengeTTL: time.Minute}, fakeProvider{}, NewMemoryStore())

	err := service.Verify(context.Background(), VerifyRequest{Purpose: PurposeRegister})
	if !errors.Is(err, ErrRequired) {
		t.Fatalf("expected ErrRequired, got %v", err)
	}
}

func TestServiceRejectsInvalidProviderResult(t *testing.T) {
	service := NewService(
		ServiceConfig{Enabled: true, ChallengeTTL: time.Minute},
		fakeProvider{result: VerifyResult{Verified: false, Code: CodeInvalid}},
		NewMemoryStore(),
	)

	err := service.Verify(context.Background(), VerifyRequest{
		Purpose:  PurposeRegister,
		Provider: ProviderAltcha,
		Token:    "bad-token",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestServiceRejectsUnsupportedProvider(t *testing.T) {
	service := NewService(
		ServiceConfig{Enabled: true, ChallengeTTL: time.Minute},
		fakeProvider{result: VerifyResult{Verified: true, Code: CodeOK}},
		NewMemoryStore(),
	)

	err := service.Verify(context.Background(), VerifyRequest{
		Purpose:  PurposeRegister,
		Provider: "other",
		Token:    "valid-token",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestServiceRejectsReplayedToken(t *testing.T) {
	service := NewService(
		ServiceConfig{Enabled: true, ChallengeTTL: time.Minute},
		fakeProvider{result: VerifyResult{Verified: true, Code: CodeOK}},
		NewMemoryStore(),
	)
	req := VerifyRequest{
		Purpose:  PurposeRegister,
		Provider: ProviderAltcha,
		Token:    "token-1",
	}

	if err := service.Verify(context.Background(), req); err != nil {
		t.Fatalf("first verify returned error: %v", err)
	}
	if err := service.Verify(context.Background(), req); !errors.Is(err, ErrReplayed) {
		t.Fatalf("expected replay error, got %v", err)
	}
}

type fakeProvider struct {
	result VerifyResult
}

func (p fakeProvider) Challenge(context.Context, Purpose, Subject) (Challenge, error) {
	return Challenge{
		Provider: ProviderAltcha,
		Purpose:  PurposeRegister,
		Payload:  map[string]any{"challenge": "fake"},
	}, nil
}

func (p fakeProvider) Verify(context.Context, VerifyRequest) (VerifyResult, error) {
	if p.result.Code == "" {
		return VerifyResult{Verified: true, Code: CodeOK}, nil
	}
	return p.result, nil
}
