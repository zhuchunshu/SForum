package main

import "testing"

func TestHasArgument(t *testing.T) {
	if !hasArgument([]string{"--check-no-pending"}, "--check-no-pending") {
		t.Fatal("expected zero-downtime schema guard argument")
	}
	if hasArgument([]string{"--version"}, "--check-no-pending") {
		t.Fatal("unexpected schema guard argument")
	}
	if !hasArgument([]string{"--check-online-safe"}, "--check-online-safe") {
		t.Fatal("expected online migration guard argument")
	}
}

func TestValidateArguments(t *testing.T) {
	for _, arguments := range [][]string{
		nil,
		{"--check-no-pending"},
		{"--check-online-safe"},
	} {
		if err := validateArguments(arguments); err != nil {
			t.Fatalf("expected arguments %v to be accepted: %v", arguments, err)
		}
	}
	for _, arguments := range [][]string{
		{"--unknown"},
		{"--check-no-pending", "--check-online-safe"},
	} {
		if err := validateArguments(arguments); err == nil {
			t.Fatalf("expected arguments %v to be rejected", arguments)
		}
	}
}
