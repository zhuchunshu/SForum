package extensions

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const themeRuntimeGenesisRecoveryTimeout = 5 * time.Second

// EnsureInitialThemeRuntimePublication imports a legacy active theme into the
// immutable desired-state ledger at most once. Theme activation and genesis
// share one transaction advisory lock, so whichever commits first becomes the
// authority observed by every waiter.
func (s *PostgresStore) EnsureInitialThemeRuntimePublication(
	ctx context.Context,
) (ThemeRuntimePublication, error) {
	if s == nil || s.pool == nil || ctx == nil {
		return ThemeRuntimePublication{}, ErrThemePublicationConflict
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ThemeRuntimePublication{}, fmt.Errorf("begin initial theme runtime publication: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(
		ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, themeRuntimeActivationLockKey,
	); err != nil {
		return ThemeRuntimePublication{}, fmt.Errorf("lock initial theme runtime publication: %w", err)
	}

	latest, err := loadLatestThemeRuntimePublication(ctx, tx, false)
	switch {
	case err == nil:
		if !validPersistedThemeRuntimePublication(latest) {
			return ThemeRuntimePublication{}, fmt.Errorf(
				"%w: existing theme runtime publication is invalid",
				ErrThemePublicationConflict,
			)
		}
		if err := tx.Rollback(ctx); err != nil {
			return ThemeRuntimePublication{}, fmt.Errorf("release initial theme runtime publication lock: %w", err)
		}
		return latest, nil
	case !errors.Is(err, ErrThemePublicationNotFound):
		return ThemeRuntimePublication{}, fmt.Errorf("load initial theme runtime publication: %w", err)
	}

	active, err := lockInitialThemeRuntimeArtifact(ctx, tx)
	if err != nil {
		return ThemeRuntimePublication{}, err
	}
	approved, actorUserID, err := lockThemeApproval(ctx, tx, active.ID)
	if err != nil {
		return ThemeRuntimePublication{}, fmt.Errorf("lock initial theme replacement approval: %w", err)
	}
	publication, err := insertThemeRuntimePublication(ctx, tx, ThemeRuntimePublication{
		DesiredState:             ThemeRuntimePublicationActive,
		ThemeID:                  active.ID,
		ThemeVersion:             active.Version,
		PackageDigest:            active.Digest,
		CoreReplacementsApproved: approved,
		ActorUserID:              actorUserID,
		Reason:                   ThemeRuntimePublicationStartupRepair,
	})
	if err != nil {
		return ThemeRuntimePublication{}, err
	}
	if err := tx.Commit(ctx); err == nil {
		return publication, nil
	} else {
		commitErr := fmt.Errorf("commit initial theme runtime publication: %w", err)
		recovered, recoveryErr := s.readExactInitialThemeRuntimePublication(ctx, publication)
		if recoveryErr == nil {
			return recovered, nil
		}
		return ThemeRuntimePublication{}, errors.Join(commitErr, recoveryErr)
	}
}

func lockInitialThemeRuntimeArtifact(
	ctx context.Context,
	tx pgx.Tx,
) (themePublicationTuple, error) {
	var extensionID, extensionType, status string
	var activeVersionID, versionID sql.NullInt64
	var versionExtensionID, version, digest sql.NullString
	var manifestBody []byte
	err := tx.QueryRow(ctx, `
		SELECT e.id, e.type, e.status, e.active_version_id,
		       v.id, v.extension_id, v.version, v.package_digest, v.manifest
		FROM extensions AS e
		JOIN extension_versions AS v
		  ON v.id = e.active_version_id AND v.extension_id = e.id
		WHERE e.type = 'theme' AND e.status = 'enabled'
		FOR UPDATE OF e, v
	`).Scan(
		&extensionID, &extensionType, &status, &activeVersionID,
		&versionID, &versionExtensionID, &version, &digest, &manifestBody,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return themePublicationTuple{}, fmt.Errorf(
			"%w: no active theme is available for initial publication",
			ErrThemePublicationConflict,
		)
	}
	if err != nil {
		return themePublicationTuple{}, fmt.Errorf("lock initial active theme: %w", err)
	}
	if extensionType != TypeTheme || status != StatusEnabled ||
		extensionID == "" || extensionID != strings.TrimSpace(extensionID) ||
		!activeVersionID.Valid || activeVersionID.Int64 <= 0 ||
		!versionID.Valid || versionID.Int64 != activeVersionID.Int64 ||
		!versionExtensionID.Valid || versionExtensionID.String != extensionID ||
		!version.Valid || version.String == "" || version.String != strings.TrimSpace(version.String) ||
		!digest.Valid || !validPackageDigest(digest.String) || len(manifestBody) == 0 {
		return themePublicationTuple{}, fmt.Errorf(
			"%w: active theme has no exact active artifact",
			ErrThemePublicationConflict,
		)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestBody, &manifest); err != nil ||
		manifest.ID != extensionID || manifest.Type != TypeTheme || manifest.Version != version.String {
		return themePublicationTuple{}, fmt.Errorf(
			"%w: active theme manifest does not match its active artifact",
			ErrThemePublicationConflict,
		)
	}
	return themePublicationTuple{
		ID: extensionID, VersionID: versionID.Int64, Version: version.String, Digest: digest.String,
	}, nil
}

func validPersistedThemeRuntimePublication(publication ThemeRuntimePublication) bool {
	return publication.Revision > 0 && !publication.CreatedAt.IsZero() && validThemeRuntimePublication(publication)
}

func (s *PostgresStore) readExactInitialThemeRuntimePublication(
	ctx context.Context,
	expected ThemeRuntimePublication,
) (ThemeRuntimePublication, error) {
	if expected.Revision <= 0 {
		return ThemeRuntimePublication{}, ErrThemePublicationConflict
	}
	verifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), themeRuntimeGenesisRecoveryTimeout)
	defer cancel()
	stored, err := s.ThemeRuntimePublicationByRevision(verifyCtx, expected.Revision)
	if err != nil {
		return ThemeRuntimePublication{}, fmt.Errorf("verify initial theme runtime publication commit: %w", err)
	}
	if !validPersistedThemeRuntimePublication(stored) || !sameThemeRuntimePublication(stored, expected) {
		return ThemeRuntimePublication{}, fmt.Errorf(
			"%w: committed initial theme runtime publication changed",
			ErrThemePublicationConflict,
		)
	}
	return stored, nil
}

var _ InitialThemeRuntimePublicationEnsurer = (*PostgresStore)(nil)
