package pages

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore 持久化 page provider 绑定。
type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) ListBindings(ctx context.Context) ([]ProviderBinding, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT page_id, extension_id, contribution_id, version, package_digest,
		       COALESCE(approved_by, 0), COALESCE(template_path, ''), COALESCE(contract_version, '')
		FROM page_provider_bindings
		ORDER BY page_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProviderBinding
	for rows.Next() {
		var b ProviderBinding
		if err := rows.Scan(
			&b.PageID, &b.ExtensionID, &b.ContributionID, &b.Version, &b.PackageDigest,
			&b.ApprovedBy, &b.TemplatePath, &b.ContractVersion,
		); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *PostgresStore) GetBinding(ctx context.Context, pageID string) (ProviderBinding, bool, error) {
	var b ProviderBinding
	err := s.pool.QueryRow(ctx, `
		SELECT page_id, extension_id, contribution_id, version, package_digest,
		       COALESCE(approved_by, 0), COALESCE(template_path, ''), COALESCE(contract_version, '')
		FROM page_provider_bindings WHERE page_id = $1`, pageID).
		Scan(
			&b.PageID, &b.ExtensionID, &b.ContributionID, &b.Version, &b.PackageDigest,
			&b.ApprovedBy, &b.TemplatePath, &b.ContractVersion,
		)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProviderBinding{}, false, nil
	}
	if err != nil {
		return ProviderBinding{}, false, err
	}
	return b, true, nil
}

func (s *PostgresStore) UpsertBinding(ctx context.Context, binding ProviderBinding) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO page_provider_bindings (
			page_id, extension_id, contribution_id, version, package_digest,
			approved_by, template_path, contract_version, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, 0), $7, $8, NOW())
		ON CONFLICT (page_id) DO UPDATE SET
			extension_id = EXCLUDED.extension_id,
			contribution_id = EXCLUDED.contribution_id,
			version = EXCLUDED.version,
			package_digest = EXCLUDED.package_digest,
			approved_by = EXCLUDED.approved_by,
			template_path = EXCLUDED.template_path,
			contract_version = EXCLUDED.contract_version,
			updated_at = NOW()`,
		binding.PageID, binding.ExtensionID, binding.ContributionID, binding.Version,
		binding.PackageDigest, binding.ApprovedBy, binding.TemplatePath, binding.ContractVersion)
	return err
}

func (s *PostgresStore) DeleteBinding(ctx context.Context, pageID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM page_provider_bindings WHERE page_id = $1`, pageID)
	return err
}

func (s *PostgresStore) ReplaceExtensionBindings(ctx context.Context, extensionIDs []string, bindings []ProviderBinding) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if len(extensionIDs) != 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM page_provider_bindings WHERE extension_id = ANY($1)`, extensionIDs); err != nil {
			return err
		}
	}
	for _, binding := range bindings {
		if _, err := tx.Exec(ctx, `
			INSERT INTO page_provider_bindings (
				page_id, extension_id, contribution_id, version, package_digest,
				approved_by, template_path, contract_version, updated_at
			) VALUES ($1, $2, $3, $4, $5, NULLIF($6, 0), $7, $8, NOW())
			ON CONFLICT (page_id) DO UPDATE SET
				extension_id = EXCLUDED.extension_id,
				contribution_id = EXCLUDED.contribution_id,
				version = EXCLUDED.version,
				package_digest = EXCLUDED.package_digest,
				approved_by = EXCLUDED.approved_by,
				template_path = EXCLUDED.template_path,
				contract_version = EXCLUDED.contract_version,
				updated_at = NOW()`,
			binding.PageID, binding.ExtensionID, binding.ContributionID, binding.Version,
			binding.PackageDigest, binding.ApprovedBy, binding.TemplatePath, binding.ContractVersion); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *PostgresStore) ReconcileExtensionBindings(ctx context.Context, activeExtensionID string, staleExtensionIDs []string, allowed []ProviderBinding) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if len(staleExtensionIDs) != 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM page_provider_bindings WHERE extension_id = ANY($1) AND extension_id <> $2`, staleExtensionIDs, activeExtensionID); err != nil {
			return err
		}
	}
	allowedByPage := make(map[string]ProviderBinding, len(allowed))
	for _, binding := range allowed {
		allowedByPage[binding.PageID] = binding
	}
	rows, err := tx.Query(ctx, `
		SELECT page_id, extension_id, contribution_id, version, package_digest,
		       COALESCE(approved_by, 0), COALESCE(template_path, ''), COALESCE(contract_version, '')
		FROM page_provider_bindings WHERE extension_id = $1`, activeExtensionID)
	if err != nil {
		return err
	}
	var remove []string
	for rows.Next() {
		var binding ProviderBinding
		if err := rows.Scan(&binding.PageID, &binding.ExtensionID, &binding.ContributionID, &binding.Version,
			&binding.PackageDigest, &binding.ApprovedBy, &binding.TemplatePath, &binding.ContractVersion); err != nil {
			rows.Close()
			return err
		}
		exact, ok := allowedByPage[binding.PageID]
		if !ok || !sameProviderArtifact(binding, exact) {
			remove = append(remove, binding.PageID)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if len(remove) != 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM page_provider_bindings WHERE page_id = ANY($1)`, remove); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
