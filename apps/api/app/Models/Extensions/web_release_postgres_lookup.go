package extensions

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (s *PostgresWebReleaseStore) ActiveWebRelease(ctx context.Context) (WebRelease, error) {
	return loadActiveWebRelease(ctx, s.db)
}

func (s *PostgresWebReleaseStore) HasLiveWebRelease(ctx context.Context) (bool, error) {
	var exists bool
	if err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM web_releases
			WHERE status IN (
				'queued', 'resolving', 'installing', 'building',
				'verifying', 'ready', 'activating', 'active'
			)
		)
	`).Scan(&exists); err != nil {
		return false, fmt.Errorf("check live web releases: %w", err)
	}
	return exists, nil
}

// LatestProgressWebReleaseForExtension 返回与该扩展相关的进行中或最近失败发布，
// 供插件列表展示启停/信任变更的进度条（不含已成功 active 的历史记录）。
func (s *PostgresWebReleaseStore) LatestProgressWebReleaseForExtension(ctx context.Context, extensionID string) (WebRelease, error) {
	extensionID = normalizeID(extensionID)
	if extensionID == "" {
		return WebRelease{}, ErrWebReleaseNotFound
	}
	release, err := scanWebRelease(s.db.QueryRow(ctx, `
		SELECT `+webReleaseColumns+`
		FROM web_releases wr
		WHERE wr.status IN (
			'queued', 'resolving', 'installing', 'building',
			'verifying', 'ready', 'activating', 'failed'
		)
		  AND (
		    wr.trigger_extension_id = $1
		    OR EXISTS (
		      SELECT 1
		      FROM web_release_extension_effects e
		      WHERE e.web_release_id = wr.id
		        AND e.extension_id = $1
		    )
		  )
		ORDER BY wr.desired_generation DESC, wr.id DESC
		LIMIT 1
	`, extensionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return WebRelease{}, ErrWebReleaseNotFound
	}
	if err != nil {
		return WebRelease{}, fmt.Errorf("load progress web release for extension: %w", err)
	}
	return release, nil
}

func (s *PostgresWebReleaseStore) ActiveWebReleaseTx(ctx context.Context, tx pgx.Tx) (WebRelease, error) {
	return loadActiveWebRelease(ctx, pgxWebReleaseSQL{queryer: tx})
}

func loadActiveWebRelease(ctx context.Context, db webReleaseSQL) (WebRelease, error) {
	release, err := scanWebRelease(db.QueryRow(ctx, `
		SELECT `+webReleaseColumns+`
		FROM web_releases
		WHERE status = 'active'
		ORDER BY desired_generation DESC, id DESC
		LIMIT 1
	`))
	if errors.Is(err, pgx.ErrNoRows) {
		return WebRelease{}, ErrWebReleaseNotFound
	}
	if err != nil {
		return WebRelease{}, fmt.Errorf("load active web release: %w", err)
	}
	return release, nil
}

func (s *PostgresWebReleaseStore) LiveWebReleasesByCompositionTx(
	ctx context.Context,
	tx pgx.Tx,
	compositionHash string,
) ([]WebReleaseDetail, error) {
	db := pgxWebReleaseSQL{queryer: tx}
	rows, err := db.Query(ctx, `
		SELECT `+webReleaseColumns+`
		FROM web_releases
		WHERE composition_hash = $1
		  AND status IN (
		    'queued', 'resolving', 'installing', 'building',
		    'verifying', 'ready', 'activating', 'active'
		  )
		ORDER BY desired_generation DESC, id DESC
	`, compositionHash)
	if err != nil {
		return nil, fmt.Errorf("list live web releases by composition: %w", err)
	}
	releases := make([]WebRelease, 0)
	for rows.Next() {
		release, err := scanWebRelease(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		releases = append(releases, release)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate live web releases by composition: %w", err)
	}
	rows.Close()

	result := make([]WebReleaseDetail, len(releases))
	for index, release := range releases {
		effects, err := s.listWebReleaseEffects(ctx, db, release.ID)
		if err != nil {
			return nil, err
		}
		result[index] = WebReleaseDetail{WebRelease: release, Effects: effects}
	}
	return result, nil
}
