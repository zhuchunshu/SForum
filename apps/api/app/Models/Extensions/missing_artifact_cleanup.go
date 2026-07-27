package extensions

import (
	"context"
	"errors"
	"fmt"
	"sort"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	audit "github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
)

const (
	CodeMissingArtifactCleanupInvalid     = "extension.missing_artifact_cleanup_invalid"
	CodeMissingArtifactCleanupUnavailable = "extension.missing_artifact_cleanup_unavailable"

	MissingArtifactDataPreserve        = "preserve"
	MissingArtifactDataDiscardSettings = "discard_settings"
)

var (
	ErrMissingArtifactCleanupInvalid     = errors.New("extensions: missing artifact cleanup invalid")
	ErrMissingArtifactCleanupUnavailable = errors.New("extensions: missing artifact cleanup unavailable")
)

type MissingArtifactCleanupInput struct {
	ExtensionIDs []string `json:"extensionIds"`
	DataMode     string   `json:"dataMode,omitempty"`
}

type MissingArtifactCleanupItem struct {
	ExtensionID      string `json:"extensionId"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	Version          string `json:"version"`
	PackageDigest    string `json:"packageDigest"`
	PackagePath      string `json:"-"`
	DataMode         string `json:"dataMode"`
	RetainSettings   bool   `json:"retainSettings"`
	BusinessDataKept bool   `json:"businessDataKept"`
}

type MissingArtifactCleanupResult struct {
	Removed []MissingArtifactCleanupItem `json:"removed"`
}

type missingArtifactCleanupStore interface {
	CleanupMissingArtifacts(context.Context, int64, []MissingArtifactCleanupItem) error
}

func (s *LifecycleService) CleanupMissingArtifacts(
	ctx context.Context,
	actor identity.Actor,
	input MissingArtifactCleanupInput,
) (MissingArtifactCleanupResult, error) {
	if !actor.IsSuperAdmin() {
		return MissingArtifactCleanupResult{}, identity.ErrPermissionDenied
	}
	dataMode, err := normalizeMissingArtifactDataMode(input.DataMode)
	if err != nil {
		return MissingArtifactCleanupResult{}, err
	}
	ids := normalizedMissingArtifactIDs(input.ExtensionIDs)
	if len(ids) == 0 || len(ids) > 100 {
		return MissingArtifactCleanupResult{}, ErrMissingArtifactCleanupInvalid
	}
	repository, ok := s.store.(missingArtifactCleanupStore)
	if !ok {
		return MissingArtifactCleanupResult{}, ErrMissingArtifactCleanupUnavailable
	}

	items := make([]MissingArtifactCleanupItem, 0, len(ids))
	for _, id := range ids {
		extension, loadErr := s.store.Get(ctx, id)
		if loadErr != nil {
			return MissingArtifactCleanupResult{}, loadErr
		}
		if extension.Source == SourceBuiltin || extension.IsSystem || !extension.IsDeletable ||
			extension.ID == DefaultThemeID {
			return MissingArtifactCleanupResult{}, ErrNotDeletable
		}
		if extension.Status == StatusEnabled {
			return MissingArtifactCleanupResult{}, ErrMustDisableFirst
		}
		if artifactState(extension.PackagePath) != ArtifactMissing {
			return MissingArtifactCleanupResult{}, ErrMissingArtifactCleanupInvalid
		}
		items = append(items, MissingArtifactCleanupItem{
			ExtensionID:      extension.ID,
			Name:             extension.Name,
			Type:             extension.Type,
			Version:          extension.Version,
			PackageDigest:    extension.PackageDigest,
			PackagePath:      extension.PackagePath,
			DataMode:         dataMode,
			RetainSettings:   dataMode == MissingArtifactDataPreserve,
			BusinessDataKept: true,
		})
	}
	if err := repository.CleanupMissingArtifacts(ctx, actor.ID, items); err != nil {
		return MissingArtifactCleanupResult{}, err
	}
	for _, item := range items {
		if s.pageRegistry != nil {
			s.pageRegistry.ClearExtension(item.ExtensionID)
		}
		s.appendAudit(ctx, actor, audit.ActionExtensionUninstalled, map[string]any{
			"extensionId":      item.ExtensionID,
			"type":             item.Type,
			"version":          item.Version,
			"reason":           "artifact_missing",
			"dataMode":         item.DataMode,
			"retainSettings":   item.RetainSettings,
			"businessDataKept": true,
		})
	}
	return MissingArtifactCleanupResult{Removed: items}, nil
}

func normalizeMissingArtifactDataMode(value string) (string, error) {
	switch value {
	case "", MissingArtifactDataPreserve:
		return MissingArtifactDataPreserve, nil
	case MissingArtifactDataDiscardSettings:
		return MissingArtifactDataDiscardSettings, nil
	default:
		return "", fmt.Errorf("%w: invalid data mode", ErrMissingArtifactCleanupInvalid)
	}
}

func normalizedMissingArtifactIDs(values []string) []string {
	unique := make(map[string]struct{}, len(values))
	for _, value := range values {
		if id := normalizeID(value); id != "" {
			unique[id] = struct{}{}
		}
	}
	ids := make([]string, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
