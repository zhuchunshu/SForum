package bootstrap

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestDeriveQueryRegistryCursorSecretIsStableAndInstallationScoped(t *testing.T) {
	installationA := strings.Repeat("a", 64)
	installationB := strings.Repeat("b", 64)
	first, err := deriveQueryRegistryCursorSecret("session-hash-secret-a", installationA)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := deriveQueryRegistryCursorSecret("session-hash-secret-a", installationA)
	if err != nil {
		t.Fatal(err)
	}
	otherInstallation, err := deriveQueryRegistryCursorSecret("session-hash-secret-a", installationB)
	if err != nil {
		t.Fatal(err)
	}
	otherRoot, err := deriveQueryRegistryCursorSecret("session-hash-secret-b", installationA)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 32 || !bytes.Equal(first, replay) || bytes.Equal(first, otherInstallation) || bytes.Equal(first, otherRoot) {
		t.Fatal("cursor secret derivation is not stable and domain isolated")
	}
}

func TestDeriveQueryRegistryCursorSecretRejectsInvalidInputs(t *testing.T) {
	for _, input := range []struct {
		secret       string
		installation string
	}{
		{installation: strings.Repeat("a", 64)},
		{secret: " secret ", installation: strings.Repeat("a", 64)},
		{secret: "secret", installation: "not-an-installation-id"},
		{secret: "secret", installation: strings.Repeat("A", 64)},
	} {
		if _, err := deriveQueryRegistryCursorSecret(input.secret, input.installation); !errors.Is(err, errProductionQueryRegistrySecret) {
			t.Fatalf("derive error = %v, want invalid secret", err)
		}
	}
}
