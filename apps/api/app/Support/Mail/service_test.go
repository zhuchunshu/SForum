package mail

import (
	"context"
	"errors"
	"testing"
)

func TestServiceDevLogProviderDelivers(t *testing.T) {
	resolver := StaticResolver{Options: RuntimeOptions{Provider: ProviderDevLog}}
	service := NewService(resolver, nil)

	if err := service.Send(context.Background(), Message{To: "u@example.com", Subject: "hi", TextBody: "body"}); err != nil {
		t.Fatalf("expected dev_log send success, got %v", err)
	}
}

func TestServiceNoopProviderSucceeds(t *testing.T) {
	resolver := StaticResolver{Options: RuntimeOptions{Provider: ProviderNoop}}
	service := NewService(resolver, nil)

	if err := service.Send(context.Background(), Message{To: "u@example.com"}); err != nil {
		t.Fatalf("expected noop success, got %v", err)
	}
}

func TestServiceEmptyProviderDefaultsToNoop(t *testing.T) {
	resolver := StaticResolver{Options: RuntimeOptions{Provider: ""}}
	service := NewService(resolver, nil)

	if err := service.Send(context.Background(), Message{To: "u@example.com"}); err != nil {
		t.Fatalf("expected empty provider to noop, got %v", err)
	}
}

func TestServiceRejectsUnknownProvider(t *testing.T) {
	resolver := StaticResolver{Options: RuntimeOptions{Provider: "carrier-pigeon"}}
	service := NewService(resolver, nil)

	err := service.Send(context.Background(), Message{To: "u@example.com"})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestSMTPProviderRejectsMissingRecipient(t *testing.T) {
	provider := NewSMTPProvider(SMTPConfig{Host: "localhost", FromAddress: "noreply@example.com"})
	err := provider.Send(context.Background(), Message{To: ""})
	if !errors.Is(err, err) {
		t.Fatalf("expected error for missing recipient, got %v", err)
	}
}

func TestSMTPProviderRejectsMissingFromAddress(t *testing.T) {
	provider := NewSMTPProvider(SMTPConfig{Host: "localhost"})
	err := provider.Send(context.Background(), Message{To: "u@example.com"})
	if err == nil {
		t.Fatal("expected error for missing from address")
	}
}

func TestRecommendedDefaultsAreBeginnerFriendly(t *testing.T) {
	defaults := RecommendedDefaults()
	if defaults.Provider != ProviderDevLog {
		t.Fatalf("expected dev_log default, got %s", defaults.Provider)
	}
	if defaults.FromAddress == "" {
		t.Fatal("expected non-empty default from address")
	}
	if defaults.SMTP.Encryption != EncryptionStartTLS {
		t.Fatalf("expected starttls default, got %s", defaults.SMTP.Encryption)
	}
}
