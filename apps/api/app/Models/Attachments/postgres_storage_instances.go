package attachments

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

func nullableUserID(id int64) any {
	if id <= 0 {
		return nil
	}
	return id
}

func (s *PostgresStore) CreateStorageInstance(ctx context.Context, input StorageInstanceCreate) (StorageInstance, error) {
	payload, err := json.Marshal(input.Settings)
	if err != nil {
		return StorageInstance{}, err
	}
	return scanStorageInstance(s.pool.QueryRow(ctx, `
		INSERT INTO attachment_storage_instances (id, extension_id, name, settings, created_by_user_id)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id::text, extension_id, name, settings, config_revision, status,
		  last_probe_status, last_probe_message, last_probe_at, created_at, updated_at,
		  0::bigint
	`, input.ID, input.ExtensionID, input.Name, payload, nullableUserID(input.CreatedByUserID)))
}

func (s *PostgresStore) UpdateStorageInstance(ctx context.Context, id string, expectedRevision int64, name string, settings map[string]string) (StorageInstance, error) {
	payload, err := json.Marshal(settings)
	if err != nil {
		return StorageInstance{}, err
	}
	item, err := scanStorageInstance(s.pool.QueryRow(ctx, `
		UPDATE attachment_storage_instances
		SET name=$3, settings=$4, config_revision=config_revision+1, status='unverified',
		    last_probe_status='', last_probe_message='', last_probe_at=NULL, updated_at=now()
		WHERE id=$1::uuid AND config_revision=$2
		RETURNING id::text, extension_id, name, settings, config_revision, status,
		  last_probe_status, last_probe_message, last_probe_at, created_at, updated_at,
		  (SELECT count(*) FROM attachments WHERE provider = 'instance:' || $1)
	`, id, expectedRevision, name, payload))
	if errors.Is(err, pgx.ErrNoRows) {
		return StorageInstance{}, ErrStorageInstanceInvalid
	}
	return item, err
}

func (s *PostgresStore) GetStorageInstance(ctx context.Context, id string) (StorageInstance, error) {
	item, err := scanStorageInstance(s.pool.QueryRow(ctx, `
		SELECT i.id::text, i.extension_id, i.name, i.settings, i.config_revision, i.status,
		 i.last_probe_status, i.last_probe_message, i.last_probe_at, i.created_at, i.updated_at,
		 (SELECT count(*) FROM attachments a WHERE a.provider = 'instance:' || i.id::text)
		FROM attachment_storage_instances i WHERE i.id=$1::uuid
	`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return StorageInstance{}, ErrStorageInstanceInvalid
	}
	return item, err
}

func (s *PostgresStore) ListStorageInstances(ctx context.Context) ([]StorageInstance, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT i.id::text, i.extension_id, i.name, i.settings, i.config_revision, i.status,
		 i.last_probe_status, i.last_probe_message, i.last_probe_at, i.created_at, i.updated_at,
		 (SELECT count(*) FROM attachments a WHERE a.provider = 'instance:' || i.id::text)
		FROM attachment_storage_instances i ORDER BY i.created_at, i.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StorageInstance{}
	for rows.Next() {
		item, scanErr := scanStorageInstance(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *PostgresStore) UpdateStorageInstanceProbe(ctx context.Context, id, status, message string) error {
	state := "error"
	if status == "ok" {
		state = "ready"
	}
	_, err := s.pool.Exec(ctx, `UPDATE attachment_storage_instances
		SET status=$2, last_probe_status=$3, last_probe_message=$4, last_probe_at=now(), updated_at=now()
		WHERE id=$1::uuid`, id, state, status, message)
	return err
}

func (s *PostgresStore) DeleteStorageInstance(ctx context.Context, id string) error {
	command, err := s.pool.Exec(ctx, `DELETE FROM attachment_storage_instances i
		WHERE i.id=$1::uuid
		  AND NOT EXISTS (SELECT 1 FROM attachments a WHERE a.provider='instance:' || i.id::text)`, id)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		var exists bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM attachment_storage_instances WHERE id=$1::uuid)`, id).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return ErrStorageInstanceReferenced
		}
		return ErrStorageInstanceInvalid
	}
	return nil
}

type storageInstanceScanner interface{ Scan(...any) error }

func scanStorageInstance(row storageInstanceScanner) (StorageInstance, error) {
	var item StorageInstance
	var raw []byte
	err := row.Scan(&item.ID, &item.ExtensionID, &item.Name, &raw, &item.ConfigRevision, &item.Status,
		&item.LastProbeStatus, &item.LastProbeMessage, &item.LastProbeAt, &item.CreatedAt, &item.UpdatedAt, &item.AttachmentCount)
	if err != nil {
		return StorageInstance{}, err
	}
	item.Values = map[string]string{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &item.Values); err != nil {
			return StorageInstance{}, err
		}
	}
	item.ID = strings.TrimSpace(item.ID)
	return item, nil
}
