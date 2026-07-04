package bootstrap

import (
	"testing"
	"time"

	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

func TestAPIAddressUsesConfiguredHostAndPort(t *testing.T) {
	cfg := config.Config{HTTPHost: "127.0.0.1", HTTPPort: "8081"}

	if got := apiAddress(cfg); got != "127.0.0.1:8081" {
		t.Fatalf("expected configured address, got %q", got)
	}
}

func TestAPICloseRunsCleanupOnce(t *testing.T) {
	calls := 0
	api := &API{
		close: func() {
			calls++
		},
	}

	api.Close()
	api.Close()

	if calls != 1 {
		t.Fatalf("expected cleanup once, got %d", calls)
	}
}

func TestNewHumanVerifyServiceRespectsDisabledProvider(t *testing.T) {
	service, err := newHumanVerifyService(config.Config{
		HumanVerificationProvider: "disabled",
		AltchaChallengeTTL:        time.Minute,
		AltchaCost:                1,
	}, humanverify.NewMemoryStore())
	if err != nil {
		t.Fatalf("newHumanVerifyService returned error: %v", err)
	}
	if service.Enabled() {
		t.Fatal("expected disabled human verifier")
	}
}

func TestNewHumanVerifyServiceEnablesAltchaProvider(t *testing.T) {
	service, err := newHumanVerifyService(config.Config{
		HumanVerificationProvider: "altcha",
		AltchaSecret:              "test-secret",
		AltchaChallengeTTL:        time.Minute,
		AltchaCost:                1,
	}, humanverify.NewMemoryStore())
	if err != nil {
		t.Fatalf("newHumanVerifyService returned error: %v", err)
	}
	if !service.Enabled() {
		t.Fatal("expected enabled human verifier")
	}
}
