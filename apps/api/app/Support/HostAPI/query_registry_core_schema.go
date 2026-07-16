package hostapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// queryRegistryCoreResultSchemaDocument builds a Draft 2020-12 schema matching
// Host Query row shapes after normalizeProtocolV2QueryValue:
// BIGINT → string, INTEGER → number, BOOLEAN → bool, timestamps/text → string,
// nullable columns allow null. Field selection is a subset, so required is empty.
func queryRegistryCoreResultSchemaDocument(definition protocolV2QueryDefinition) ([]byte, string, error) {
	properties, err := queryRegistryCoreResultSchemaProperties(definition)
	if err != nil {
		return nil, "", err
	}
	document := map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"$id":                  "https://sforum.invalid/query-result/" + definition.ResultSchemaID + "@" + definition.ResultSchemaVersion,
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
	}
	body, err := json.Marshal(document)
	if err != nil {
		return nil, "", fmt.Errorf("%w: marshal result schema for %s", ErrQueryRegistryCoreInvalid, definition.ID)
	}
	digest := sha256.Sum256(body)
	return body, hex.EncodeToString(digest[:]), nil
}

func queryRegistryCoreResultSchemaProperties(definition protocolV2QueryDefinition) (map[string]any, error) {
	var properties map[string]any
	switch strings.TrimSpace(definition.ResultSchemaID) {
	case QuerySafeUserResultSchemaID:
		properties = map[string]any{
			"id":           schemaString(),
			"username":     schemaString(),
			"display_name": schemaString(),
			"created_at":   schemaString(),
			"updated_at":   schemaString(),
		}
	case QueryPublicTopicResultSchemaID:
		properties = map[string]any{
			"id":               schemaString(),
			"category_id":      schemaString(),
			"category_slug":    schemaString(),
			"author_user_id":   schemaStringOrNull(),
			"title":            schemaString(),
			"slug":             schemaString(),
			"status":           schemaString(),
			"is_pinned":        schemaBoolean(),
			"comment_count":    schemaString(),
			"view_count":       schemaString(),
			"last_activity_at": schemaString(),
			"created_at":       schemaString(),
			"updated_at":       schemaString(),
			"html_content":     schemaString(),
			"plain_text":       schemaString(),
			"source_format":    schemaString(),
			"render_version":   schemaString(),
			"content_hash":     schemaString(),
		}
	case QueryPublicAttachmentSchemaID:
		properties = map[string]any{
			"id":              schemaString(),
			"public_id":       schemaString(),
			"owner_user_id":   schemaStringOrNull(),
			"original_name":   schemaString(),
			"content_type":    schemaString(),
			"extension":       schemaString(),
			"size_bytes":      schemaString(),
			"sha256":          schemaString(),
			"image_width":     schemaNumberOrNull(),
			"image_height":    schemaNumberOrNull(),
			"reference_count": schemaNumber(),
			"created_at":      schemaString(),
			"updated_at":      schemaString(),
		}
	default:
		return nil, fmt.Errorf(
			"%w: no Host-aligned JSON result schema for %q",
			ErrQueryRegistryCoreUnsupported, definition.ResultSchemaID,
		)
	}
	if len(definition.Fields) != len(properties) {
		return nil, fmt.Errorf(
			"%w: Host fields do not exactly match schema %q",
			ErrQueryRegistryCoreUnsupported, definition.ResultSchemaID,
		)
	}
	for _, field := range definition.Fields {
		if _, exists := properties[strings.TrimSpace(field.Name)]; !exists {
			return nil, fmt.Errorf(
				"%w: Host field %q is absent from schema %q",
				ErrQueryRegistryCoreUnsupported, field.Name, definition.ResultSchemaID,
			)
		}
	}
	return properties, nil
}

func schemaString() map[string]any {
	return map[string]any{"type": "string"}
}

func schemaStringOrNull() map[string]any {
	return map[string]any{"type": []any{"string", "null"}}
}

func schemaBoolean() map[string]any {
	return map[string]any{"type": "boolean"}
}

func schemaNumber() map[string]any {
	return map[string]any{"type": "number"}
}

func schemaNumberOrNull() map[string]any {
	return map[string]any{"type": []any{"number", "null"}}
}
