package webreleaseruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

var ErrPointerNotFound = errors.New("web release pointer not found")

type ReleaseReader interface {
	WebRelease(context.Context, int64) (extensions.WebReleaseDetail, error)
}

type PointerStore struct {
	root     string
	releases ReleaseReader
}

func NewPointerStore(root string, releases ReleaseReader) *PointerStore {
	return &PointerStore{root: absolutePath(root), releases: releases}
}

func (s *PointerStore) WriteCurrent(_ context.Context, current CurrentRelease) error {
	if current.SchemaVersion == 0 {
		current.SchemaVersion = ReleaseManifestSchemaVersion
	}
	if current.RequestedAt.IsZero() {
		current.RequestedAt = time.Now().UTC()
	}
	return writeJSONAtomic(filepath.Join(s.root, "current.json"), current)
}

func (s *PointerStore) ReadActive(_ context.Context) (ActiveRelease, error) {
	var active ActiveRelease
	if err := readJSON(filepath.Join(s.root, "active.json"), &active); err != nil {
		return ActiveRelease{}, err
	}
	return active, nil
}

func (s *PointerStore) ReadFailure(_ context.Context, releaseID int64) (Failure, error) {
	var failure Failure
	if err := readJSON(filepath.Join(s.root, "failures", fmt.Sprintf("%d.json", releaseID)), &failure); err != nil {
		return Failure{}, err
	}
	if failure.ReleaseID != releaseID {
		return Failure{}, fmt.Errorf("failure acknowledgement release mismatch")
	}
	return failure, nil
}

func (s *PointerStore) RestorePrevious(ctx context.Context, failed extensions.WebReleaseDetail) error {
	if failed.PreviousReleaseID == nil {
		err := os.Remove(filepath.Join(s.root, "current.json"))
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if s.releases == nil {
		return fmt.Errorf("previous release reader is unavailable")
	}
	previous, err := s.releases.WebRelease(ctx, *failed.PreviousReleaseID)
	if err != nil {
		return err
	}
	return s.WriteCurrent(ctx, CurrentRelease{
		SchemaVersion: ReleaseManifestSchemaVersion, ReleaseID: previous.ID,
		CompositionHash: previous.CompositionHash, ArtifactPath: previous.ArtifactPath,
		ArtifactDigest: previous.ArtifactDigest, ServerEntry: previous.ServerEntry,
		ThemeID: previous.ActiveThemeID, ThemeVersion: previous.ThemeVersion,
		ReloadMode: previous.ReloadMode,
	})
}

func readJSON(target string, value any) error {
	body, err := os.ReadFile(target)
	if errors.Is(err, os.ErrNotExist) {
		return ErrPointerNotFound
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, value); err != nil {
		return fmt.Errorf("decode %s: %w", target, err)
	}
	return nil
}
