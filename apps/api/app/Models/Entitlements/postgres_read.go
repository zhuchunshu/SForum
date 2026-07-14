package entitlements

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *PostgresRepository) Get(ctx context.Context, id int64) (Entitlement, error) {
	if id <= 0 {
		return Entitlement{}, fmt.Errorf("%w: entitlement id is required", ErrInvalidInput)
	}
	return loadEntitlement(ctx, r.pool, id, false)
}

func (r *PostgresRepository) Effective(ctx context.Context, input EffectiveInput) (Entitlement, bool, error) {
	prepared, err := prepareEffective(input)
	if err != nil {
		return Entitlement{}, false, err
	}
	base := `
		SELECT id, subject_type, subject_id, scope_kind, resource_type, resource_id, capability,
		       status, source_type, source_id, valid_from, valid_until, revoked_at, expired_at,
		       revision, created_at, updated_at
		FROM entitlements
		WHERE subject_type = $1 AND subject_id = $2 AND status = 'active'
		  AND valid_from <= $3 AND (valid_until IS NULL OR valid_until > $3)`
	var row pgx.Row
	if prepared.Scope.Kind == ScopeResource {
		row = r.pool.QueryRow(ctx, base+`
		  AND scope_kind = 'resource' AND resource_type = $4 AND resource_id = $5
		ORDER BY valid_until DESC NULLS FIRST, id DESC LIMIT 1
	`, prepared.Subject.Type, prepared.Subject.ID, prepared.At,
			prepared.Scope.ResourceType, prepared.Scope.ResourceID)
	} else {
		row = r.pool.QueryRow(ctx, base+`
		  AND scope_kind = 'capability' AND capability = $4
		ORDER BY valid_until DESC NULLS FIRST, id DESC LIMIT 1
	`, prepared.Subject.Type, prepared.Subject.ID, prepared.At, prepared.Scope.Capability)
	}
	entitlement, err := scanEntitlement(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Entitlement{}, false, nil
	}
	if err != nil {
		return Entitlement{}, false, fmt.Errorf("resolve effective entitlement: %w", err)
	}
	return entitlement, true, nil
}

func loadEntitlement(ctx context.Context, db interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id int64, forUpdate bool) (Entitlement, error) {
	query := `
		SELECT id, subject_type, subject_id, scope_kind, resource_type, resource_id, capability,
		       status, source_type, source_id, valid_from, valid_until, revoked_at, expired_at,
		       revision, created_at, updated_at
		FROM entitlements WHERE id = $1`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	entitlement, err := scanEntitlement(db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Entitlement{}, ErrNotFound
	}
	if err != nil {
		return Entitlement{}, fmt.Errorf("load entitlement: %w", err)
	}
	return entitlement, nil
}

func scanEntitlement(row rowScanner) (Entitlement, error) {
	var item Entitlement
	var resourceType, resourceID, capability *string
	err := row.Scan(
		&item.ID, &item.Subject.Type, &item.Subject.ID, &item.Scope.Kind,
		&resourceType, &resourceID, &capability, &item.Status,
		&item.Source.Type, &item.Source.ID, &item.ValidFrom, &item.ValidUntil,
		&item.RevokedAt, &item.ExpiredAt, &item.Revision, &item.CreatedAt, &item.UpdatedAt,
	)
	if resourceType != nil {
		item.Scope.ResourceType = *resourceType
	}
	if resourceID != nil {
		item.Scope.ResourceID = *resourceID
	}
	if capability != nil {
		item.Scope.Capability = *capability
	}
	return item, err
}

func nullableInt64(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}
