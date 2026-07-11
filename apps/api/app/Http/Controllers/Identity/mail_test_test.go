package identitycontroller

import "testing"

func TestNormalizeTestRecipient(t *testing.T) {
	for _, value := range []string{"", "not-an-email", "Name <user@example.com>", "user@example.com,other@example.com"} {
		if _, ok := normalizeTestRecipient(value); ok {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
	if got, ok := normalizeTestRecipient(" USER@example.com "); !ok || got != "USER@example.com" {
		t.Fatalf("got %q, %v", got, ok)
	}
}
