package entitymeta

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

const MaximumTransactionalValues = 50

type preparedExtensionValue struct {
	definition fieldRow
	item       MetaValue
	valueText  *string
}

// ValidateExtensionValuesTx locks and validates the exact entity/field plan
// without writing values. It is intended for an authoritative Host Command
// Prepare phase in the same transaction as the later mutation.
func (s *PostgresStore) ValidateExtensionValuesTx(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
	actorUserID int64,
	entityType string,
	entityID int64,
	inputs []UpsertValueInput,
) error {
	_, err := prepareExtensionValuesTx(ctx, tx, extensionID, actorUserID, entityType, entityID, inputs)
	return err
}

// UpsertExtensionValuesTx atomically writes fields owned by one exact
// extension. The caller owns commit/rollback and must authorize the actor for
// entity_meta.manage before entering this method.
func (s *PostgresStore) UpsertExtensionValuesTx(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
	actorUserID int64,
	entityType string,
	entityID int64,
	inputs []UpsertValueInput,
) ([]MetaValue, error) {
	prepared, err := prepareExtensionValuesTx(ctx, tx, extensionID, actorUserID, entityType, entityID, inputs)
	if err != nil {
		return nil, err
	}
	result := make([]MetaValue, 0, len(prepared))
	for _, value := range prepared {
		item := value.item
		if value.valueText == nil {
			if _, err := tx.Exec(ctx, `
				DELETE FROM entity_meta_values
				WHERE entity_type = $1 AND entity_id = $2 AND field_key = $3
			`, entityType, entityID, item.FieldKey); err != nil {
				return nil, fmt.Errorf("delete extension entity meta value: %w", err)
			}
			result = append(result, item)
			continue
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO entity_meta_values (
			  entity_type, entity_id, field_key, value_text, updated_at, updated_by_user_id
			)
			VALUES ($1, $2, $3, $4, transaction_timestamp(), $5)
			ON CONFLICT (entity_type, entity_id, field_key) DO UPDATE SET
			  value_text = EXCLUDED.value_text,
			  updated_at = EXCLUDED.updated_at,
			  updated_by_user_id = EXCLUDED.updated_by_user_id
			RETURNING updated_at
		`, entityType, entityID, item.FieldKey, *value.valueText, actorUserID).Scan(&item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("upsert extension entity meta value: %w", err)
		}
		item.Value, err = parseStoredValue(value.definition.ValueType, *value.valueText)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func prepareExtensionValuesTx(
	ctx context.Context,
	tx pgx.Tx,
	extensionID string,
	actorUserID int64,
	entityType string,
	entityID int64,
	inputs []UpsertValueInput,
) ([]preparedExtensionValue, error) {
	extensionID = strings.TrimSpace(extensionID)
	if tx == nil || extensionID == "" || actorUserID <= 0 || !validEntityType(entityType) || entityID <= 0 ||
		len(inputs) == 0 || len(inputs) > MaximumTransactionalValues {
		return nil, ErrInvalid
	}
	if err := lockTransactionalEntity(ctx, tx, entityType, entityID); err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(inputs))
	result := make([]preparedExtensionValue, 0, len(inputs))
	for _, input := range inputs {
		fieldKey := strings.TrimSpace(input.FieldKey)
		if fieldKey == "" || seen[fieldKey] {
			return nil, ErrInvalid
		}
		seen[fieldKey] = true
		definition, err := loadTransactionalField(ctx, tx, fieldKey)
		if err != nil {
			return nil, err
		}
		if definition.EntityType != entityType {
			return nil, ErrInvalid
		}
		if !definition.Enabled {
			return nil, ErrFieldDisabled
		}
		if definition.OwnerExtensionID != extensionID {
			return nil, ErrPermission
		}

		prepared := preparedExtensionValue{definition: definition, item: MetaValue{
			FieldKey: fieldKey, EntityType: entityType, EntityID: entityID,
			ValueType: definition.ValueType, Visibility: definition.Visibility,
			Label: labelMap(definition.LabelZHCN, definition.LabelENUS),
		}}
		if input.Value == nil {
			if definition.Required {
				return nil, ErrInvalid
			}
			result = append(result, prepared)
			continue
		}

		valueText, err := normalizeValue(definition.ValueType, input.Value, definition.Constraints)
		if err != nil {
			return nil, err
		}
		prepared.valueText = &valueText
		result = append(result, prepared)
	}
	return result, nil
}

func lockTransactionalEntity(ctx context.Context, tx pgx.Tx, entityType string, entityID int64) error {
	query := ""
	switch entityType {
	case EntityUser:
		query = `SELECT id FROM users WHERE id = $1 FOR SHARE`
	case EntityTopic:
		query = `SELECT id FROM topics WHERE id = $1 FOR SHARE`
	default:
		return ErrInvalid
	}
	var lockedID int64
	if err := tx.QueryRow(ctx, query, entityID).Scan(&lockedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEntityNotFound
		}
		return fmt.Errorf("lock entity meta target: %w", err)
	}
	return nil
}

func loadTransactionalField(ctx context.Context, tx pgx.Tx, fieldKey string) (fieldRow, error) {
	row, err := scanFieldRow(tx.QueryRow(ctx, `
		SELECT id, field_key, entity_type, value_type, visibility,
		       label_zh_cn, label_en_us, description_zh_cn, description_en_us,
		       owner_extension_id, required, enabled, sort_order, constraints, created_at, updated_at
		FROM entity_field_definitions
		WHERE field_key = $1
		FOR SHARE
	`, fieldKey))
	if errors.Is(err, pgx.ErrNoRows) {
		return fieldRow{}, ErrNotFound
	}
	if err != nil {
		return fieldRow{}, fmt.Errorf("load transactional entity meta field: %w", err)
	}
	return row, nil
}
