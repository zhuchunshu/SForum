package extensions

import (
	"encoding/hex"
	"errors"
	"strings"
)

var (
	ErrStagedVersionInvalid  = errors.New("extensions: invalid staged version identity")
	ErrStagedVersionNotFound = errors.New("extensions: staged version not found")
	ErrStagedVersionConflict = errors.New("extensions: staged version identity conflict")
)

// StagedVersionCASInput binds a lifecycle write to one immutable candidate.
// Both the database id and digest are required so stale operations cannot
// promote or discard a newer upload that reused the same extension id.
type StagedVersionCASInput struct {
	ExtensionID             string
	ExpectedStagedVersionID int64
	ExpectedPackageDigest   string
}

func validateStagedVersionCASInput(input StagedVersionCASInput) error {
	if input.ExtensionID == "" || input.ExtensionID != normalizeID(input.ExtensionID) ||
		input.ExpectedStagedVersionID <= 0 || !isCanonicalStagedVersionDigest(input.ExpectedPackageDigest) {
		return ErrStagedVersionInvalid
	}
	return nil
}

func isCanonicalStagedVersionDigest(digest string) bool {
	if len(digest) != 64 || digest != strings.ToLower(digest) || digest != strings.TrimSpace(digest) {
		return false
	}
	decoded, err := hex.DecodeString(digest)
	return err == nil && len(decoded) == 32
}
