package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type frontendTrustDB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type PostgresFrontendTrustStore struct {
	db frontendTrustDB
}

func NewPostgresFrontendTrustStore(pool *pgxpool.Pool) *PostgresFrontendTrustStore {
	return newPostgresFrontendTrustStore(pool)
}

func newPostgresFrontendTrustStore(db frontendTrustDB) *PostgresFrontendTrustStore {
	return &PostgresFrontendTrustStore{db: db}
}

func (s *PostgresFrontendTrustStore) FrontendGrant(ctx context.Context, extensionID string, version string, digest string) (FrontendTrustGrant, error) {
	return s.liveFrontendGrant(ctx, extensionID, version, digest)
}

func (s *PostgresFrontendTrustStore) LiveFrontendGrants(ctx context.Context, extensionID string) ([]FrontendTrustGrant, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+frontendTrustGrantColumns()+`
		FROM extension_frontend_trust_grants
		WHERE revoked_at IS NULL
		  AND ($1 = '' OR extension_id = $1)
		ORDER BY extension_id, extension_version, package_digest, id
	`, strings.TrimSpace(extensionID))
	if err != nil {
		return nil, fmt.Errorf("list live frontend trust grants: %w", err)
	}
	defer rows.Close()
	items := make([]FrontendTrustGrant, 0)
	for rows.Next() {
		grant, err := scanFrontendTrustGrant(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, grant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate live frontend trust grants: %w", err)
	}
	return items, nil
}

func (s *PostgresFrontendTrustStore) CreateFrontendGrant(ctx context.Context, input FrontendTrustGrantInput) (FrontendTrustGrant, error) {
	points := canonicalFrontendTrustSet(input.ContributionPoints)
	components := canonicalFrontendTrustSet(input.ComponentIDs)
	pointsJSON, err := json.Marshal(points)
	if err != nil {
		return FrontendTrustGrant{}, fmt.Errorf("marshal frontend trust contribution points: %w", err)
	}
	componentsJSON, err := json.Marshal(components)
	if err != nil {
		return FrontendTrustGrant{}, fmt.Errorf("marshal frontend trust component ids: %w", err)
	}

	grant, err := scanFrontendTrustGrant(s.db.QueryRow(ctx, `
		INSERT INTO extension_frontend_trust_grants (
			extension_id, extension_version, package_digest, api_version,
			contribution_points, component_ids, granted_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7)
		ON CONFLICT (extension_id, extension_version, package_digest)
		  WHERE revoked_at IS NULL
		DO NOTHING
		RETURNING `+frontendTrustGrantColumns(),
		input.ExtensionID,
		input.ExtensionVersion,
		input.PackageDigest,
		input.APIVersion,
		pointsJSON,
		componentsJSON,
		nullableActorID(input.GrantedByUserID),
	))
	if err == nil {
		return grant, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return FrontendTrustGrant{}, fmt.Errorf("create frontend trust grant: %w", err)
	}

	existing, err := s.liveFrontendGrant(ctx, input.ExtensionID, input.ExtensionVersion, input.PackageDigest)
	if err != nil {
		if errors.Is(err, ErrFrontendGrantNotFound) {
			return FrontendTrustGrant{}, ErrFrontendGrantStateConflict
		}
		return FrontendTrustGrant{}, err
	}
	if existing.RevocationRequestedAt != nil {
		return FrontendTrustGrant{}, ErrFrontendGrantStateConflict
	}
	if existing.APIVersion != input.APIVersion ||
		!slices.Equal(existing.ContributionPoints, points) ||
		!slices.Equal(existing.ComponentIDs, components) {
		return FrontendTrustGrant{}, ErrFrontendGrantConflict
	}
	return existing, nil
}

func (s *PostgresFrontendTrustStore) RequestFrontendRevocation(ctx context.Context, input FrontendRevocationInput) (FrontendTrustGrant, error) {
	grant, err := scanFrontendTrustGrant(s.db.QueryRow(ctx, `
		UPDATE extension_frontend_trust_grants
		SET revocation_requested_at = now(),
		    revocation_requested_by_user_id = $4
		WHERE extension_id = $1
		  AND extension_version = $2
		  AND package_digest = $3
		  AND revocation_requested_at IS NULL
		  AND revoked_at IS NULL
		RETURNING `+frontendTrustGrantColumns(),
		input.ExtensionID,
		input.ExtensionVersion,
		input.PackageDigest,
		nullableActorID(input.RequestedByUserID),
	))
	if err == nil {
		return grant, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return FrontendTrustGrant{}, fmt.Errorf("request frontend trust revocation: %w", err)
	}
	if _, loadErr := s.liveFrontendGrant(ctx, input.ExtensionID, input.ExtensionVersion, input.PackageDigest); loadErr != nil {
		return FrontendTrustGrant{}, loadErr
	}
	return FrontendTrustGrant{}, ErrFrontendGrantStateConflict
}

func (s *PostgresFrontendTrustStore) RequestAllFrontendRevocations(ctx context.Context, requestedByUserID int64) ([]FrontendTrustGrant, error) {
	rows, err := s.db.Query(ctx, `
		UPDATE extension_frontend_trust_grants
		SET revocation_requested_by_user_id = CASE
		      WHEN revocation_requested_at IS NULL THEN $1
		      ELSE revocation_requested_by_user_id
		    END,
		    revocation_requested_at = COALESCE(revocation_requested_at, now())
		WHERE revoked_at IS NULL
		RETURNING `+frontendTrustGrantColumns(),
		nullableActorID(requestedByUserID),
	)
	if err != nil {
		return nil, fmt.Errorf("request all frontend trust revocations: %w", err)
	}
	defer rows.Close()
	items := make([]FrontendTrustGrant, 0)
	for rows.Next() {
		grant, err := scanFrontendTrustGrant(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, grant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate requested frontend trust revocations: %w", err)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ExtensionID != items[j].ExtensionID {
			return items[i].ExtensionID < items[j].ExtensionID
		}
		if items[i].ExtensionVersion != items[j].ExtensionVersion {
			return items[i].ExtensionVersion < items[j].ExtensionVersion
		}
		if items[i].PackageDigest != items[j].PackageDigest {
			return items[i].PackageDigest < items[j].PackageDigest
		}
		return items[i].ID < items[j].ID
	})
	return items, nil
}

func (s *PostgresFrontendTrustStore) FinalizeFrontendRevocation(ctx context.Context, input FrontendFinalizeInput) (FrontendTrustGrant, error) {
	grant, err := scanFrontendTrustGrant(s.db.QueryRow(ctx, `
		UPDATE extension_frontend_trust_grants
		SET revoked_at = now(),
		    revoked_by_user_id = revocation_requested_by_user_id
		WHERE extension_id = $1
		  AND extension_version = $2
		  AND package_digest = $3
		  AND revocation_requested_at IS NOT NULL
		  AND revoked_at IS NULL
		RETURNING `+frontendTrustGrantColumns(),
		input.ExtensionID,
		input.ExtensionVersion,
		input.PackageDigest,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		existing, loadErr := s.exactFrontendGrant(ctx, input.ExtensionID, input.ExtensionVersion, input.PackageDigest, false)
		if loadErr == nil && existing.RevokedAt != nil {
			return existing, nil
		}
		if loadErr != nil && !errors.Is(loadErr, ErrFrontendGrantNotFound) {
			return FrontendTrustGrant{}, loadErr
		}
		return FrontendTrustGrant{}, ErrFrontendGrantStateConflict
	}
	if err != nil {
		return FrontendTrustGrant{}, fmt.Errorf("finalize frontend trust revocation: %w", err)
	}
	return grant, nil
}

func (s *PostgresFrontendTrustStore) FinalizeFrontendRevocations(ctx context.Context, releaseID int64) error {
	_, err := s.db.Exec(ctx, `
		WITH target_release AS (
			SELECT id
			FROM web_releases
			WHERE id = $1
			  AND status = 'active'
		)
		UPDATE extension_frontend_trust_grants AS grants
		SET revoked_at = now(),
		    revoked_by_user_id = grants.revocation_requested_by_user_id
		FROM target_release
		WHERE grants.revocation_requested_at IS NOT NULL
		  AND grants.revoked_at IS NULL
		  AND NOT EXISTS (
			SELECT 1
			FROM web_release_extensions AS release_extensions
			WHERE release_extensions.web_release_id = target_release.id
			  AND release_extensions.extension_id = grants.extension_id
			  AND release_extensions.extension_version = grants.extension_version
			  AND release_extensions.package_digest = grants.package_digest
		  )
	`, releaseID)
	if err != nil {
		return fmt.Errorf("finalize frontend trust revocations: %w", err)
	}
	return nil
}

func (s *PostgresFrontendTrustStore) liveFrontendGrant(ctx context.Context, extensionID string, version string, digest string) (FrontendTrustGrant, error) {
	return s.exactFrontendGrant(ctx, extensionID, version, digest, true)
}

func (s *PostgresFrontendTrustStore) exactFrontendGrant(
	ctx context.Context,
	extensionID string,
	version string,
	digest string,
	liveOnly bool,
) (FrontendTrustGrant, error) {
	grant, err := scanFrontendTrustGrant(s.db.QueryRow(ctx, `
		SELECT `+frontendTrustGrantColumns()+`
		FROM extension_frontend_trust_grants
		WHERE extension_id = $1
		  AND extension_version = $2
		  AND package_digest = $3
		  AND ($4 = false OR revoked_at IS NULL)
		ORDER BY granted_at DESC, id DESC
		LIMIT 1
	`, extensionID, version, digest, liveOnly))
	if errors.Is(err, pgx.ErrNoRows) {
		return FrontendTrustGrant{}, ErrFrontendGrantNotFound
	}
	if err != nil {
		return FrontendTrustGrant{}, fmt.Errorf("load frontend trust grant: %w", err)
	}
	return grant, nil
}

func frontendTrustGrantColumns() string {
	return `id, extension_id, extension_version, package_digest, api_version,
	  contribution_points, component_ids, COALESCE(granted_by_user_id, 0), granted_at,
	  revocation_requested_at, COALESCE(revocation_requested_by_user_id, 0),
	  revoked_at, COALESCE(revoked_by_user_id, 0)`
}

type frontendTrustGrantScanner interface {
	Scan(...any) error
}

func scanFrontendTrustGrant(row frontendTrustGrantScanner) (FrontendTrustGrant, error) {
	var grant FrontendTrustGrant
	var pointsJSON []byte
	var componentsJSON []byte
	if err := row.Scan(
		&grant.ID,
		&grant.ExtensionID,
		&grant.ExtensionVersion,
		&grant.PackageDigest,
		&grant.APIVersion,
		&pointsJSON,
		&componentsJSON,
		&grant.GrantedByUserID,
		&grant.GrantedAt,
		&grant.RevocationRequestedAt,
		&grant.RevocationRequestedByUserID,
		&grant.RevokedAt,
		&grant.RevokedByUserID,
	); err != nil {
		return FrontendTrustGrant{}, err
	}
	if err := json.Unmarshal(pointsJSON, &grant.ContributionPoints); err != nil {
		return FrontendTrustGrant{}, fmt.Errorf("decode frontend trust contribution points: %w", err)
	}
	if err := json.Unmarshal(componentsJSON, &grant.ComponentIDs); err != nil {
		return FrontendTrustGrant{}, fmt.Errorf("decode frontend trust component ids: %w", err)
	}
	grant.ContributionPoints = canonicalFrontendTrustSet(grant.ContributionPoints)
	grant.ComponentIDs = canonicalFrontendTrustSet(grant.ComponentIDs)
	return grant, nil
}

func canonicalFrontendTrustSet(values []string) []string {
	result := append([]string{}, values...)
	sort.Strings(result)
	return slices.Compact(result)
}
