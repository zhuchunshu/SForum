package postgres

import (
	"context"
	"strings"
	"testing"
)

func TestAdvisoryLockerRejectsMissingPool(t *testing.T) {
	err := NewAdvisoryLocker(nil).WithLock(context.Background(), "test", func(context.Context) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("expected missing pool error, got %v", err)
	}
}
