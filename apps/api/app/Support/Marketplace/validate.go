package marketplace

import (
	"regexp"
	"strings"
	"time"

	semver "github.com/Masterminds/semver/v3"
)

// Extension id matches Manifest V3 (exported subset; avoid importing full validator).
var extensionIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,80}$`)

// sha256Hex is 64 lowercase hex chars.
var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

func validExtensionID(id string) bool {
	return extensionIDPattern.MatchString(id)
}

func validSHA256Digest(value string) bool {
	return sha256Hex.MatchString(strings.ToLower(strings.TrimSpace(value)))
}

func validSemverVersion(value string) bool {
	_, err := semver.StrictNewVersion(strings.TrimSpace(value))
	return err == nil
}

func validSemverConstraint(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	_, err := semver.NewConstraint(value)
	return err == nil
}

func normalizeEntry(entry *Entry) error {
	if entry == nil {
		return ErrInvalid
	}
	entry.ExtensionID = strings.ToLower(strings.TrimSpace(entry.ExtensionID))
	entry.Version = strings.TrimSpace(entry.Version)
	entry.PackageDigest = strings.ToLower(strings.TrimSpace(entry.PackageDigest))
	entry.SBOMDigest = strings.ToLower(strings.TrimSpace(entry.SBOMDigest))
	entry.Channel = strings.ToLower(strings.TrimSpace(entry.Channel))
	if entry.Channel == "" {
		entry.Channel = ChannelStable
	}
	if !validExtensionID(entry.ExtensionID) {
		return ErrInvalid
	}
	if !validSemverVersion(entry.Version) {
		return ErrInvalid
	}
	if !validSHA256Digest(entry.PackageDigest) {
		return ErrDigest
	}
	if entry.SBOMDigest != "" && !validSHA256Digest(entry.SBOMDigest) {
		return ErrDigest
	}
	if !validSemverConstraint(entry.MinSForumVersion) || !validSemverConstraint(entry.MaxSForumVersion) {
		return ErrInvalid
	}
	for i := range entry.Dependencies {
		dep := &entry.Dependencies[i]
		dep.ExtensionID = strings.ToLower(strings.TrimSpace(dep.ExtensionID))
		dep.Version = strings.TrimSpace(dep.Version)
		if !validExtensionID(dep.ExtensionID) || !validSemverConstraint(dep.Version) {
			return ErrInvalid
		}
	}
	for i := range entry.Notices {
		entry.Notices[i].Kind = strings.ToLower(strings.TrimSpace(entry.Notices[i].Kind))
		entry.Notices[i].Severity = strings.ToLower(strings.TrimSpace(entry.Notices[i].Severity))
	}
	return nil
}

func entryInTimeWindow(entry Entry, now time.Time) bool {
	if !entry.AvailableFrom.IsZero() && now.Before(entry.AvailableFrom) {
		return false
	}
	if !entry.AvailableUntil.IsZero() && now.After(entry.AvailableUntil) {
		return false
	}
	return true
}

func hostCompatible(entry Entry, hostVersion string) error {
	hostVersion = strings.TrimSpace(hostVersion)
	if hostVersion == "" {
		// 未配置 Host 版本时仅在有约束时放行并留给 report 警告。
		return nil
	}
	host, err := semver.StrictNewVersion(hostVersion)
	if err != nil {
		return ErrIncompatible
	}
	if strings.TrimSpace(entry.MinSForumVersion) != "" {
		c, err := semver.NewConstraint(entry.MinSForumVersion)
		if err != nil || !c.Check(host) {
			return ErrIncompatible
		}
	}
	if strings.TrimSpace(entry.MaxSForumVersion) != "" {
		c, err := semver.NewConstraint(entry.MaxSForumVersion)
		if err != nil || !c.Check(host) {
			return ErrIncompatible
		}
	}
	return nil
}

func versionSatisfies(version, constraint string) bool {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" {
		return true
	}
	v, err := semver.StrictNewVersion(version)
	if err != nil {
		return false
	}
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return false
	}
	return c.Check(v)
}
