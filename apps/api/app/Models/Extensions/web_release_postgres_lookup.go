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
