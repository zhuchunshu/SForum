package extensions

import (
	"errors"
	"strings"
	"testing"
)

func TestExactVersionRepositoryInputValidation(t *testing.T) {
	validStaged := StagedVersionCASInput{
		ExtensionID:             "demo.plugin",
		ExpectedActiveVersionID: 1, ExpectedActiveVersion: "1.0.0",
		ExpectedActivePackageDigest: strings.Repeat("a", 64),
		ExpectedStagedVersionID:     2, ExpectedStagedVersion: "2.0.0",
		ExpectedPackageDigest: strings.Repeat("b", 64),
	}
	if err := validateStagedVersionCASInput(validStaged); err != nil {
		t.Fatal(err)
	}
	missingActive := validStaged
	missingActive.ExpectedActiveVersion = ""
	if err := validateStagedVersionCASInput(missingActive); !errors.Is(err, ErrStagedVersionInvalid) {
		t.Fatalf("missing active identity error = %v", err)
	}
	uppercaseTarget := validStaged
	uppercaseTarget.ExpectedPackageDigest = strings.Repeat("B", 64)
	if err := validateStagedVersionCASInput(uppercaseTarget); !errors.Is(err, ErrStagedVersionInvalid) {
		t.Fatalf("uppercase target digest error = %v", err)
	}

	validRollback := RollbackExtensionVersionInput{
		ExtensionID:             "demo.plugin",
		ExpectedActiveVersionID: 2, ExpectedActiveVersion: "2.0.0",
		ExpectedActivePackageDigest: strings.Repeat("b", 64),
		TargetVersionID:             1, TargetVersion: "1.0.0",
		TargetPackageDigest: strings.Repeat("a", 64),
	}
	if err := validateRollbackExtensionVersionInput(validRollback); err != nil {
		t.Fatal(err)
	}
	sameVersionID := validRollback
	sameVersionID.TargetVersionID = sameVersionID.ExpectedActiveVersionID
	if err := validateRollbackExtensionVersionInput(sameVersionID); !errors.Is(err, ErrExtensionVersionInvalid) {
		t.Fatalf("same target identity error = %v", err)
	}
	if err := validateExactExtensionVersionInput(ExactExtensionVersionInput{
		ExtensionID: "demo.plugin", Version: "1.0.0", PackageDigest: strings.Repeat("a", 64),
	}); err != nil {
		t.Fatal(err)
	}
	if err := validateExactExtensionVersionInput(ExactExtensionVersionInput{
		ExtensionID: "demo.plugin", Version: " 1.0.0", PackageDigest: strings.Repeat("a", 64),
	}); !errors.Is(err, ErrExtensionVersionInvalid) {
		t.Fatalf("non-canonical version error = %v", err)
	}
}
