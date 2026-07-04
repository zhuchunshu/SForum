package bootstrap

import (
	"testing"

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
