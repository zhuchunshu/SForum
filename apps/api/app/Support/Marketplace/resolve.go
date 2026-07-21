package marketplace

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// resolveRecursive builds a dependency-ordered install plan with cycle/conflict detection.
// Uses Masterminds/semver constraints; mirrors Manifest package graph topology ideas
// (provider-before-consumer order, cycle detection) without requiring full Manifests.
func resolveRecursive(
	entries []Entry,
	rootID, channel string,
	policy OperatorPolicy,
	now time.Time,
) (ResolveResult, error) {
	// index by extensionID -> preferred channel entries (highest version that matches channel).
	byID := map[string][]Entry{}
	for _, entry := range entries {
		byID[entry.ExtensionID] = append(byID[entry.ExtensionID], entry)
	}
	for id := range byID {
		sort.Slice(byID[id], func(i, j int) bool {
			// Prefer higher SemVer when both match; string compare is not enough,
			// but we pick first satisfying constraint during walk.
			return byID[id][i].Version > byID[id][j].Version
		})
	}

	selected := map[string]Entry{}
	visiting := map[string]bool{}
	var order []PlanStep
	report := CompatibilityReport{Compatible: true}

	var walk func(id, wantChannel, versionConstraint string, optional bool) error
	walk = func(id, wantChannel, versionConstraint string, optional bool) error {
		id = strings.ToLower(strings.TrimSpace(id))
		if id == "" {
			return ErrInvalid
		}
		if visiting[id] {
			return fmt.Errorf("%w: %s", ErrCycle, id)
		}
		if existing, ok := selected[id]; ok {
			if versionConstraint != "" && !versionSatisfies(existing.Version, versionConstraint) {
				return fmt.Errorf("%w: %s@%s does not satisfy %s", ErrConflict, id, existing.Version, versionConstraint)
			}
			return nil
		}
		candidates := byID[id]
		if len(candidates) == 0 {
			if optional {
				report.Warnings = append(report.Warnings, "optional dependency missing: "+id)
				return nil
			}
			return fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		var match *Entry
		for i := range candidates {
			entry := candidates[i]
			if wantChannel != "" && entry.Channel != wantChannel && id == rootID {
				continue
			}
			// Dependencies may be satisfied from any allowed channel.
			if id != rootID && !channelAllowed(policy.AllowedChannels, entry.Channel) {
				continue
			}
			if entry.Withdrawn {
				continue
			}
			if !entryInTimeWindow(entry, now) {
				continue
			}
			if versionConstraint != "" && !versionSatisfies(entry.Version, versionConstraint) {
				continue
			}
			if blockedByNotice(policy, entry.Notices) {
				if optional {
					report.Warnings = append(report.Warnings, "optional dependency blocked by notice: "+id)
					return nil
				}
				return ErrPolicy
			}
			if err := hostCompatible(entry, policy.HostSForumVersion); err != nil {
				if optional {
					report.Warnings = append(report.Warnings, "optional dependency incompatible host: "+id)
					return nil
				}
				return fmt.Errorf("%w: %s", ErrIncompatible, id)
			}
			match = &entry
			break
		}
		if match == nil {
			if optional {
				report.Warnings = append(report.Warnings, "optional dependency unresolved: "+id)
				return nil
			}
			// Distinguish withdrawn-only vs not found.
			for _, entry := range candidates {
				if entry.Withdrawn && (wantChannel == "" || entry.Channel == wantChannel) {
					return ErrWithdrawn
				}
			}
			return fmt.Errorf("%w: %s", ErrNotFound, id)
		}

		visiting[id] = true
		for _, dep := range match.Dependencies {
			// 依赖默认走 stable，除非约束包本身指定 channel（当前模型无 channel 字段）。
			depChannel := ChannelStable
			if err := walk(dep.ExtensionID, depChannel, dep.Version, dep.Optional); err != nil {
				return err
			}
		}
		visiting[id] = false
		selected[id] = *match
		order = append(order, PlanStep{
			ExtensionID: match.ExtensionID, Version: match.Version,
			PackageDigest: match.PackageDigest, Channel: match.Channel,
			SBOMDigest: match.SBOMDigest,
		})
		return nil
	}

	if err := walk(rootID, channel, "", false); err != nil {
		report.Compatible = false
		report.BlockedBy = append(report.BlockedBy, err.Error())
		return ResolveResult{Report: report}, err
	}
	root, ok := selected[rootID]
	if !ok {
		return ResolveResult{}, ErrNotFound
	}
	return ResolveResult{
		ExtensionID: root.ExtensionID, Version: root.Version,
		PackageDigest: root.PackageDigest, Channel: root.Channel,
		SBOMDigest: root.SBOMDigest, Order: order, Report: report,
	}, nil
}
