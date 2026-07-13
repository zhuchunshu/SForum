package main

import (
	"context"
	"strings"
	"testing"
)

func TestRootCommandIncludesOutOfBandRecoveryCommands(t *testing.T) {
	root := newRootCommand()
	for _, path := range []string{"extension list", "extension disable", "extension disable-all"} {
		command, _, err := root.Find(strings.Fields(path))
		if err != nil || command == nil {
			t.Fatalf("find %q: command=%v err=%v", path, command, err)
		}
	}
}

func TestRecoveryRequiresOnlyExplicitDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	called := false
	err := withRecoveryRepository(context.Background(), "", func(recoveryRepository) error {
		called = true
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "database url is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("recovery callback ran without a database connection")
	}
}
