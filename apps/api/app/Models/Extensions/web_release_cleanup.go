package extensions

import (
	"context"
	"fmt"
	"sort"
	"time"
)

const (
	WebReleaseSuccessfulArtifactRetention = 5
	WebReleaseFailedArtifactRetention     = 7 * 24 * time.Hour
	WebReleaseBuildLogRetention           = 30 * 24 * time.Hour
)

type WebReleaseCleanupRecord struct {
	ID                int64
	Status            WebReleaseStatus
	PreviousReleaseID *int64
	CompletedAt       *time.Time
	HasArtifact       bool
	HasBuildLog       bool
}

type WebReleaseCleanupResult struct {
	ArtifactReleaseIDs []int64
	BuildLogReleaseIDs []int64
}

// SelectWebReleaseCleanup 只选择可重建产物和过期日志，release/event 元数据永久保留。
func SelectWebReleaseCleanup(records []WebReleaseCleanupRecord, now time.Time) WebReleaseCleanupResult {
	protected := make(map[int64]struct{})
	successful := make([]WebReleaseCleanupRecord, 0, len(records))
	for _, record := range records {
		if record.Status == WebReleaseActive {
			protected[record.ID] = struct{}{}
			if record.PreviousReleaseID != nil {
				protected[*record.PreviousReleaseID] = struct{}{}
			}
		}
		if record.Status == WebReleaseInactive || record.Status == WebReleaseRolledBack {
			successful = append(successful, record)
		}
	}
	sort.Slice(successful, func(i, j int) bool {
		left, right := successful[i], successful[j]
		if left.CompletedAt != nil && right.CompletedAt != nil && !left.CompletedAt.Equal(*right.CompletedAt) {
			return left.CompletedAt.After(*right.CompletedAt)
		}
		if left.CompletedAt != nil && right.CompletedAt == nil {
			return true
		}
		if left.CompletedAt == nil && right.CompletedAt != nil {
			return false
		}
		return left.ID > right.ID
	})
	for index, record := range successful {
		if index >= WebReleaseSuccessfulArtifactRetention {
			break
		}
		protected[record.ID] = struct{}{}
	}

	result := WebReleaseCleanupResult{}
	for _, record := range records {
		if record.HasBuildLog && olderThan(record.CompletedAt, now, WebReleaseBuildLogRetention) {
			result.BuildLogReleaseIDs = append(result.BuildLogReleaseIDs, record.ID)
		}
		if !record.HasArtifact {
			continue
		}
		if _, keep := protected[record.ID]; keep {
			continue
		}
		switch record.Status {
		case WebReleaseInactive, WebReleaseRolledBack:
			result.ArtifactReleaseIDs = append(result.ArtifactReleaseIDs, record.ID)
		case WebReleaseFailed, WebReleaseSuperseded:
			if olderThan(record.CompletedAt, now, WebReleaseFailedArtifactRetention) {
				result.ArtifactReleaseIDs = append(result.ArtifactReleaseIDs, record.ID)
			}
		}
	}
	sort.Slice(result.ArtifactReleaseIDs, func(i, j int) bool { return result.ArtifactReleaseIDs[i] < result.ArtifactReleaseIDs[j] })
	sort.Slice(result.BuildLogReleaseIDs, func(i, j int) bool { return result.BuildLogReleaseIDs[i] < result.BuildLogReleaseIDs[j] })
	return result
}

func olderThan(completedAt *time.Time, now time.Time, retention time.Duration) bool {
	return completedAt != nil && !completedAt.After(now.Add(-retention))
}

func (s *PostgresWebReleaseStore) CleanupWebReleases(ctx context.Context, now time.Time) (WebReleaseCleanupResult, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, status, previous_release_id, completed_at,
		       artifact_path <> '', build_log <> ''
		FROM web_releases
		WHERE status IN ('active', 'inactive', 'rolled_back', 'failed', 'superseded')
	`)
	if err != nil {
		return WebReleaseCleanupResult{}, fmt.Errorf("list web releases for cleanup: %w", err)
	}
	defer rows.Close()
	records := make([]WebReleaseCleanupRecord, 0)
	for rows.Next() {
		var record WebReleaseCleanupRecord
		if err := rows.Scan(&record.ID, &record.Status, &record.PreviousReleaseID, &record.CompletedAt, &record.HasArtifact, &record.HasBuildLog); err != nil {
			return WebReleaseCleanupResult{}, fmt.Errorf("scan web release cleanup record: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return WebReleaseCleanupResult{}, fmt.Errorf("iterate web release cleanup records: %w", err)
	}

	result := SelectWebReleaseCleanup(records, now)
	if len(result.BuildLogReleaseIDs) > 0 {
		if _, err := s.db.Exec(ctx, `UPDATE web_releases SET build_log = '' WHERE id = ANY($1)`, result.BuildLogReleaseIDs); err != nil {
			return WebReleaseCleanupResult{}, fmt.Errorf("clean expired web release logs: %w", err)
		}
	}
	return result, nil
}
