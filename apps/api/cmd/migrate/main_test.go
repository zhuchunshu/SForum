package main

import "testing"

func TestHasArgument(t *testing.T) {
	if !hasArgument([]string{"--check-no-pending"}, "--check-no-pending") {
		t.Fatal("expected zero-downtime schema guard argument")
	}
	if hasArgument([]string{"--version"}, "--check-no-pending") {
		t.Fatal("unexpected schema guard argument")
	}
}
