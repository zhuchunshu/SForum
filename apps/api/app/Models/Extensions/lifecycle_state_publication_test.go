package extensions

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestLifecycleStatePublicationInputAcceptsEveryCanonicalBoundary(t *testing.T) {
	source := LifecycleStatePublicationArtifact{
		ExtensionID: "demo.plugin", Version: "1.0.0", PackageDigest: strings.Repeat("a", 64), VersionID: 11,
	}
	target := LifecycleStatePublicationArtifact{
		ExtensionID: "demo.plugin", Version: "2.0.0", PackageDigest: strings.Repeat("b", 64), VersionID: 12,
	}
	tests := []struct {
		operation LifecycleMachineOperation
		source    *LifecycleStatePublicationArtifact
		target    LifecycleStatePublicationArtifact
	}{
		{LifecycleMachineInstall, nil, target},
		{LifecycleMachineEnable, nil, target},
		{LifecycleMachineDisable, &target, target},
		{LifecycleMachineUpgrade, &source, target},
		{LifecycleMachineRollback, &source, target},
		{LifecycleMachineUninstall, &target, target},
	}
	for _, test := range tests {
		t.Run(string(test.operation), func(t *testing.T) {
			input := lifecycleStatePublicationTestInput(t, test.operation, test.source, test.target)
			if err := validatePrepareLifecycleStatePublicationInput(input); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLifecycleStatePublicationInputRejectsStaleOrAmbiguousIdentity(t *testing.T) {
	source := LifecycleStatePublicationArtifact{
		ExtensionID: "demo.plugin", Version: "1.0.0", PackageDigest: strings.Repeat("a", 64), VersionID: 11,
	}
	target := LifecycleStatePublicationArtifact{
		ExtensionID: "demo.plugin", Version: "2.0.0", PackageDigest: strings.Repeat("b", 64), VersionID: 12,
	}
	base := lifecycleStatePublicationTestInput(t, LifecycleMachineUpgrade, &source, target)
	tests := []struct {
		name   string
		mutate func(*PrepareLifecycleStatePublicationInput)
	}{
		{"operation", func(input *PrepareLifecycleStatePublicationInput) { input.OperationID = 0 }},
		{"position", func(input *PrepareLifecycleStatePublicationInput) { input.Position-- }},
		{"step", func(input *PrepareLifecycleStatePublicationInput) { input.StepID += ".stale" }},
		{"attempt", func(input *PrepareLifecycleStatePublicationInput) { input.Attempt = 0 }},
		{"mode", func(input *PrepareLifecycleStatePublicationInput) { input.Mode = LifecycleStatePublicationDeactivate }},
		{"source missing", func(input *PrepareLifecycleStatePublicationInput) { input.Source = nil }},
		{"same source", func(input *PrepareLifecycleStatePublicationInput) { input.Source = &input.Target }},
		{"foreign source", func(input *PrepareLifecycleStatePublicationInput) { input.Source.ExtensionID = "other.plugin" }},
		{"target id", func(input *PrepareLifecycleStatePublicationInput) { input.Target.VersionID = 0 }},
		{"target digest", func(input *PrepareLifecycleStatePublicationInput) {
			input.Target.PackageDigest = strings.Repeat("B", 64)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := base
			copySource := *base.Source
			input.Source = &copySource
			test.mutate(&input)
			if err := validatePrepareLifecycleStatePublicationInput(input); !errors.Is(err, ErrLifecycleStatePublicationInvalid) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestLifecycleStatePublicationRecordRechecksPersistedSourceAndTarget(t *testing.T) {
	sourceArtifact := LifecycleStatePublicationArtifact{
		ExtensionID: "demo.plugin", Version: "1.0.0", PackageDigest: strings.Repeat("a", 64), VersionID: 11,
	}
	targetArtifact := LifecycleStatePublicationArtifact{
		ExtensionID: "demo.plugin", Version: "2.0.0", PackageDigest: strings.Repeat("b", 64), VersionID: 12,
	}
	input := lifecycleStatePublicationTestInput(t, LifecycleMachineUpgrade, &sourceArtifact, targetArtifact)
	source := lifecycleStateVector{
		Status: StatusEnabled, Active: sourceArtifact, Staged: &targetArtifact,
	}
	target, err := lifecycleTargetState(input, source)
	if err != nil {
		t.Fatal(err)
	}
	record := lifecycleStatePublicationRecord{
		OperationID: input.OperationID, Operation: input.Operation, Position: input.Position,
		StepID: input.StepID, Mode: input.Mode, ExtensionID: input.Target.ExtensionID,
		Source: source, Target: target,
	}
	if !record.matchesInput(input) {
		t.Fatal("exact persisted state did not match input")
	}
	record.Source.Active.PackageDigest = strings.Repeat("c", 64)
	if record.matchesInput(input) {
		t.Fatal("persisted source drift matched immutable marker input")
	}
	record.Source.Active = sourceArtifact
	record.Target.Status = StatusDisabled
	if record.matchesInput(input) {
		t.Fatal("persisted target semantics drift matched input")
	}
}

func lifecycleStatePublicationTestInput(
	t *testing.T,
	operation LifecycleMachineOperation,
	source *LifecycleStatePublicationArtifact,
	target LifecycleStatePublicationArtifact,
) PrepareLifecycleStatePublicationInput {
	t.Helper()
	position, mode, err := lifecycleStatePublicationPoint(operation)
	if err != nil {
		t.Fatal(err)
	}
	path, err := RecommendedLifecyclePath(operation)
	if err != nil {
		t.Fatal(err)
	}
	return PrepareLifecycleStatePublicationInput{
		OperationID: 41, Operation: operation, Position: position,
		StepID:  fmt.Sprintf("lifecycle.%s.%02d.host.%s", operation, position, path[position].State),
		Attempt: 2, Mode: mode, Source: source, Target: target,
	}
}
