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

func (s *PostgresFrontendTrustStore) FrontendGrant(ctx context.Context, extensionID string, version string, adminFrontendDigest string) (FrontendTrustGrant, error) {
	return s.exactFrontendGrant(ctx, extensionID, version, adminFrontendDigest, true)
}

func (s *PostgresFrontendTrustStore) LiveFrontendGrants(ctx context.Context, extensionID string) ([]FrontendTrustGrant, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+frontendTrustGrantColumns()+`
		FROM extension_frontend_trust_grants
		WHERE revoked_at IS NULL
		  AND ($1 = '' OR extension_id = $1)
		ORDER BY extension_id, extension_version, admin_frontend_digest, id
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
	components := canonicalFrontendTrustSet(input.ComponentIDs)
	componentsJSON, err := json.Marshal(components)
	if err != nil {
		return FrontendTrustGrant{}, fmt.Errorf("marshal frontend trust component ids: %w", err)
	}

	grant, err := scanFrontendTrustGrant(s.db.QueryRow(ctx, `
		INSERT INTO extension_frontend_trust_grants (
			extension_id, extension_version, package_digest, admin_frontend_digest, api_version,
			component_ids, granted_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
		ON CONFLICT (extension_id, extension_version, api_version, admin_frontend_digest)
		  WHERE revoked_at IS NULL
		DO NOTHING
		RETURNING `+frontendTrustGrantColumns(),
		input.ExtensionID,
		input.ExtensionVersion,
		input.PackageDigest,
		input.AdminFrontendDigest,
		input.APIVersion,
		componentsJSON,
		nullableActorID(input.GrantedByUserID),
	))
	if err == nil {
		return grant, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return FrontendTrustGrant{}, fmt.Errorf("create frontend trust grant: %w", err)
	}

	existing, err := s.exactFrontendGrant(ctx, input.ExtensionID, input.ExtensionVersion, input.AdminFrontendDigest, true)
	if err != nil {
		if errors.Is(err, ErrFrontendGrantNotFound) {
			return FrontendTrustGrant{}, ErrFrontendGrantStateConflict
		}
		return FrontendTrustGrant{}, err
	}
	if existing.PackageDigest != input.PackageDigest ||
		existing.APIVersion != input.APIVersion ||
		!slices.Equal(existing.ComponentIDs, components) {
		return FrontendTrustGrant{}, ErrFrontendGrantConflict
	}
	return existing, nil
}

func (s *PostgresFrontendTrustStore) RevokeFrontendGrant(ctx context.Context, input FrontendRevocationInput) (FrontendTrustGrant, error) {
	grant, err := scanFrontendTrustGrant(s.db.QueryRow(ctx, `
		UPDATE extension_frontend_trust_grants
		SET revoked_at = now(),
		    revoked_by_user_id = $4
		WHERE extension_id = $1
		  AND extension_version = $2
		  AND admin_frontend_digest = $3
		  AND revoked_at IS NULL
		RETURNING `+frontendTrustGrantColumns(),
		input.ExtensionID,
		input.ExtensionVersion,
		input.AdminFrontendDigest,
		nullableActorID(input.RequestedByUserID),
	))
	if err == nil {
		return grant, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return FrontendTrustGrant{}, fmt.Errorf("revoke frontend trust grant: %w", err)
	}

	existing, loadErr := s.exactFrontendGrant(ctx, input.ExtensionID, input.ExtensionVersion, input.AdminFrontendDigest, false)
	if loadErr == nil && existing.RevokedAt != nil {
		return existing, nil
	}
	if loadErr != nil && !errors.Is(loadErr, ErrFrontendGrantNotFound) {
		return FrontendTrustGrant{}, loadErr
	}
	return FrontendTrustGrant{}, ErrFrontendGrantNotFound
}

func (s *PostgresFrontendTrustStore) RevokeAllFrontendGrants(ctx context.Context, extensionID string, actorUserID int64) error {
	_, err := s.db.Exec(ctx, `
		UPDATE extension_frontend_trust_grants
		SET revoked_at = now(),
		    revoked_by_user_id = $2
		WHERE revoked_at IS NULL
		  AND ($1 = '' OR extension_id = $1)
	`, strings.TrimSpace(extensionID), nullableActorID(actorUserID))
	if err != nil {
		return fmt.Errorf("revoke frontend trust grants: %w", err)
	}
	return nil
}

func (s *PostgresFrontendTrustStore) exactFrontendGrant(
	ctx context.Context,
	extensionID string,
	version string,
	adminFrontendDigest string,
	liveOnly bool,
) (FrontendTrustGrant, error) {
	grant, err := scanFrontendTrustGrant(s.db.QueryRow(ctx, `
		SELECT `+frontendTrustGrantColumns()+`
		FROM extension_frontend_trust_grants
		WHERE extension_id = $1
		  AND extension_version = $2
		  AND admin_frontend_digest = $3
		  AND ($4 = false OR revoked_at IS NULL)
		ORDER BY granted_at DESC, id DESC
		LIMIT 1
	`, extensionID, version, adminFrontendDigest, liveOnly))
	if errors.Is(err, pgx.ErrNoRows) {
		return FrontendTrustGrant{}, ErrFrontendGrantNotFound
	}
	if err != nil {
		return FrontendTrustGrant{}, fmt.Errorf("load frontend trust grant: %w", err)
	}
	return grant, nil
}

func frontendTrustGrantColumns() string {
	return `id, extension_id, extension_version, package_digest, admin_frontend_digest, api_version,
	  component_ids, COALESCE(granted_by_user_id, 0), granted_at,
	  revoked_at, COALESCE(revoked_by_user_id, 0)`
}

type frontendTrustGrantScanner interface {
	Scan(...any) error
}

func scanFrontendTrustGrant(row frontendTrustGrantScanner) (FrontendTrustGrant, error) {
	var grant FrontendTrustGrant
	var componentsJSON []byte
	if err := row.Scan(
		&grant.ID,
		&grant.ExtensionID,
		&grant.ExtensionVersion,
		&grant.PackageDigest,
		&grant.AdminFrontendDigest,
		&grant.APIVersion,
		&componentsJSON,
		&grant.GrantedByUserID,
		&grant.GrantedAt,
		&grant.RevokedAt,
		&grant.RevokedByUserID,
	); err != nil {
		return FrontendTrustGrant{}, err
	}
	if err := json.Unmarshal(componentsJSON, &grant.ComponentIDs); err != nil {
		return FrontendTrustGrant{}, fmt.Errorf("decode frontend trust component ids: %w", err)
	}
	grant.ComponentIDs = canonicalFrontendTrustSet(grant.ComponentIDs)
	return grant, nil
}

func canonicalFrontendTrustSet(values []string) []string {
	result := append([]string{}, values...)
	sort.Strings(result)
	return slices.Compact(result)
}
