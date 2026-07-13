package extensions

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
)

var (
	ErrStagedVersionInvalid     = errors.New("extensions: invalid staged version identity")
	ErrStagedVersionNotFound    = errors.New("extensions: staged version not found")
	ErrStagedVersionConflict    = errors.New("extensions: staged version identity conflict")
	ErrExtensionVersionInvalid  = errors.New("extensions: invalid extension version identity")
	ErrExtensionVersionNotFound = errors.New("extensions: extension version not found")
	ErrExtensionVersionConflict = errors.New("extensions: extension version identity conflict")
)

// StagedVersionCASInput 把升级写入同时绑定到活动制品和不可变候选。
// 数据库 id 仅用于宿主内部 CAS；version + digest 是可持久化、可恢复的制品身份。
type StagedVersionCASInput struct {
	ExtensionID                 string
	ExpectedActiveVersionID     int64 `json:"-"`
	ExpectedActiveVersion       string
	ExpectedActivePackageDigest string
	ExpectedStagedVersionID     int64 `json:"-"`
	ExpectedStagedVersion       string
	ExpectedPackageDigest       string
}

func validateStagedVersionCASInput(input StagedVersionCASInput) error {
	if input.ExtensionID == "" || input.ExtensionID != normalizeID(input.ExtensionID) ||
		!isExactExtensionVersionIdentity(input.ExpectedActiveVersionID, input.ExpectedActiveVersion, input.ExpectedActivePackageDigest) ||
		!isExactExtensionVersionIdentity(input.ExpectedStagedVersionID, input.ExpectedStagedVersion, input.ExpectedPackageDigest) {
		return ErrStagedVersionInvalid
	}
	return nil
}

// RollbackExtensionVersionInput 只允许从一个精确活动制品切回一个精确历史制品。
type RollbackExtensionVersionInput struct {
	ExtensionID                 string
	ExpectedActiveVersionID     int64 `json:"-"`
	ExpectedActiveVersion       string
	ExpectedActivePackageDigest string
	TargetVersionID             int64 `json:"-"`
	TargetVersion               string
	TargetPackageDigest         string
}

func validateRollbackExtensionVersionInput(input RollbackExtensionVersionInput) error {
	if input.ExtensionID == "" || input.ExtensionID != normalizeID(input.ExtensionID) ||
		!isExactExtensionVersionIdentity(input.ExpectedActiveVersionID, input.ExpectedActiveVersion, input.ExpectedActivePackageDigest) ||
		!isExactExtensionVersionIdentity(input.TargetVersionID, input.TargetVersion, input.TargetPackageDigest) ||
		input.ExpectedActiveVersionID == input.TargetVersionID {
		return ErrExtensionVersionInvalid
	}
	return nil
}

// ExactExtensionVersionInput 通过稳定的 version + digest 读取不可变快照，不暴露数据库 id。
type ExactExtensionVersionInput struct {
	ExtensionID   string
	Version       string
	PackageDigest string
}

func validateExactExtensionVersionInput(input ExactExtensionVersionInput) error {
	if input.ExtensionID == "" || input.ExtensionID != normalizeID(input.ExtensionID) ||
		!isExactExtensionVersion(input.Version, input.PackageDigest) {
		return ErrExtensionVersionInvalid
	}
	return nil
}

// ExactExtensionVersionRepository 是 lifecycle 恢复所需的最小版本事务面。
// Service 可选择性依赖该接口，旧 Store 实现无需伪造历史版本能力。
type ExactExtensionVersionRepository interface {
	PromoteStagedVersion(context.Context, StagedVersionCASInput) (Extension, error)
	DiscardStagedVersion(context.Context, StagedVersionCASInput) (Extension, error)
	RollbackExtensionVersion(context.Context, RollbackExtensionVersionInput) (Extension, error)
	GetExtensionVersion(context.Context, ExactExtensionVersionInput) (ExtensionVersion, error)
	ListExtensionVersions(context.Context, string) ([]ExtensionVersion, error)
}

func isExactExtensionVersionIdentity(versionID int64, version, digest string) bool {
	return versionID > 0 && isExactExtensionVersion(version, digest)
}

func isExactExtensionVersion(version, digest string) bool {
	return version != "" && version == strings.TrimSpace(version) && isCanonicalStagedVersionDigest(digest)
}

func isCanonicalStagedVersionDigest(digest string) bool {
	if len(digest) != 64 || digest != strings.ToLower(digest) || digest != strings.TrimSpace(digest) {
		return false
	}
	decoded, err := hex.DecodeString(digest)
	return err == nil && len(decoded) == 32
}
