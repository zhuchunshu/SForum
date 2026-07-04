package redis

import "testing"

func TestParseAddr(t *testing.T) {
	host, port, err := ParseAddr("redis:6379")
	if err != nil {
		t.Fatalf("ParseAddr returned error: %v", err)
	}
	if host != "redis" {
		t.Fatalf("expected host redis, got %q", host)
	}
	if port != 6379 {
		t.Fatalf("expected port 6379, got %d", port)
	}
}

func TestParseAddrRejectsMissingPort(t *testing.T) {
	_, _, err := ParseAddr("redis")
	if err == nil {
		t.Fatal("expected error for missing port")
	}
}
