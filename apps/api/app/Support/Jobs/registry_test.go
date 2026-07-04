package jobs

import (
	"errors"
	"testing"

	"github.com/riverqueue/river"
)

func TestRegistryBuildsWorkers(t *testing.T) {
	registry := NewRegistry()
	called := false
	registry.Add(func(workers *river.Workers) error {
		called = true
		return nil
	})

	workers, err := registry.Build()
	if err != nil {
		t.Fatalf("build workers: %v", err)
	}
	if workers == nil {
		t.Fatal("expected workers bundle")
	}
	if !called {
		t.Fatal("expected registrar to be called")
	}
}

func TestRegistryReturnsRegistrarError(t *testing.T) {
	expected := errors.New("bad registration")
	registry := NewRegistry()
	registry.Add(func(workers *river.Workers) error {
		return expected
	})

	if _, err := registry.Build(); !errors.Is(err, expected) {
		t.Fatalf("expected registrar error, got %v", err)
	}
}
